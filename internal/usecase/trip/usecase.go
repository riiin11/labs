package trip

import (
	"context"
	"log/slog"
	"time"

	"github.com/google/uuid"

	"github.com/riiin11/labs/internal/domain/trip"
)

type Create struct {
	repo tripRepository
	tx   txManager
	m    metrics
	log  *slog.Logger
}

func NewCreate(repo tripRepository, tx txManager, m metrics, log *slog.Logger) *Create {
	return &Create{repo: repo, tx: tx, m: m, log: log}
}

func (u *Create) Create(ctx context.Context, in trip.CreateInput) (trip.Trip, error) {
	if err := in.Validate(); err != nil {
		u.log.WarnContext(ctx, "create validation failed", "err", err)
		return trip.Trip{}, err
	}

	t := trip.Trip{
		ID:         uuid.NewString(),
		UserID:     in.UserID,
		DriverID:   in.DriverID,
		StartPoint: in.StartPoint,
		EndPoint:   in.EndPoint,
		Price:      in.Price,
		Status:     trip.StatusActive,
		StartedAt:  time.Now().UTC(),
	}

	err := u.tx.Do(ctx, func(ctx context.Context) error {
		if err := u.repo.Save(ctx, t); err != nil {
			return err
		}
		if err := u.repo.SaveStatusHistory(ctx, t.ID, "", trip.StatusActive); err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		u.log.ErrorContext(ctx, "create trip failed", "err", err, "trip_id", t.ID, "driver_id", t.DriverID)
		return trip.Trip{}, err
	}

	u.m.IncCreated(ctx)
	u.log.InfoContext(ctx, "trip created", "trip_id", t.ID, "driver_id", t.DriverID)
	return t, nil
}

type Get struct{ repo tripRepository }

func NewGet(repo tripRepository) *Get { return &Get{repo: repo} }

func (u *Get) Get(ctx context.Context, id string) (trip.Trip, error) {
	if _, err := uuid.Parse(id); err != nil {
		return trip.Trip{}, trip.ValidationError{Field: "trip_id", Message: "must be a valid UUID"}
	}
	return u.repo.GetByID(ctx, id)
}

type Finish struct {
	repo tripRepository
	tx   txManager
	m    metrics
	log  *slog.Logger
}

func NewFinish(repo tripRepository, tx txManager, m metrics, log *slog.Logger) *Finish {
	return &Finish{repo: repo, tx: tx, m: m, log: log}
}

func (u *Finish) Finish(ctx context.Context, id string) (trip.Trip, error) {
	if _, err := uuid.Parse(id); err != nil {
		return trip.Trip{}, trip.ValidationError{Field: "trip_id", Message: "must be a valid UUID"}
	}

	var result trip.Trip
	err := u.tx.Do(ctx, func(ctx context.Context) error {
		t, err := u.repo.GetByID(ctx, id)
		if err != nil {
			return err
		}
		if err := t.Finish(time.Now().UTC()); err != nil {
			return err
		}
		if err := u.repo.Save(ctx, t); err != nil {
			return err
		}
		if err := u.repo.SaveStatusHistory(ctx, t.ID, trip.StatusActive, trip.StatusCompleted); err != nil {
			return err
		}
		result = t
		return nil
	})
	if err != nil {
		u.log.ErrorContext(ctx, "finish trip failed", "err", err, "trip_id", id)
		return trip.Trip{}, err
	}

	u.m.IncCompleted(ctx)
	u.log.InfoContext(ctx, "trip completed", "trip_id", result.ID)
	return result, nil
}

type PositionCanceller interface {
	Cancel(tripID string)
}

type SavePosition struct {
	repo      tripRepository
	tx        txManager
	m         metrics
	canceller PositionCanceller
	log       *slog.Logger
}

func NewSavePosition(repo tripRepository, tx txManager, m metrics, c PositionCanceller, log *slog.Logger) *SavePosition {
	return &SavePosition{repo: repo, tx: tx, m: m, canceller: c, log: log}
}

func (u *SavePosition) Save(ctx context.Context, p trip.Position) (trip.Position, error) {
	if err := p.Validate(); err != nil {
		return trip.Position{}, err
	}

	var result trip.Position
	err := u.tx.Do(ctx, func(ctx context.Context) error {
		var err error
		result, err = u.repo.SavePosition(ctx, p)
		return err
	})
	if err != nil {
		u.log.ErrorContext(ctx, "save position failed", "err", err, "trip_id", p.TripID)
		return trip.Position{}, err
	}

	// Отменяем задачу в очереди (если она там была)
	if u.canceller != nil {
		u.canceller.Cancel(p.TripID)
	}

	u.m.IncPositionsSaved(ctx)
	return result, nil
}

type GetPositions struct{ repo tripRepository }

func NewGetPositions(repo tripRepository) *GetPositions {
	return &GetPositions{repo: repo}
}

func (u *GetPositions) Get(ctx context.Context, tripID string) ([]trip.Position, error) {
	if _, err := uuid.Parse(tripID); err != nil {
		return nil, trip.ValidationError{Field: "trip_id", Message: "must be a valid UUID"}
	}
	return u.repo.GetPositions(ctx, tripID)
}