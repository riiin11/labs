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
			return err // 23505 → trip.ErrDriverBusy на уровне репозитория
		}
		return u.repo.SaveStatusHistory(ctx, t.ID, trip.StatusActive)
	})
	if err != nil {
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
		t, err := u.repo.Finish(ctx, id, time.Now().UTC())
		if err != nil {
			return err
		}
		result = t
		return u.repo.SaveStatusHistory(ctx, t.ID, trip.StatusCompleted)
	})
	if err != nil {
		return trip.Trip{}, err
	}

	u.m.IncCompleted(ctx)
	u.log.InfoContext(ctx, "trip completed", "trip_id", result.ID)
	return result, nil
}