package trip_test

import (
	"context"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/riiin11/labs/internal/domain/trip"
	tripuc "github.com/riiin11/labs/internal/usecase/trip"
)

func TestFinish_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repo := NewMocktripRepository(ctrl)
	tx := NewMocktxManager(ctrl)
	m := NewMockmetrics(ctrl)
	log := slog.Default()

	tripID := "123e4567-e89b-12d3-a456-426614174000"

	tx.EXPECT().Do(gomock.Any(), gomock.Any()).DoAndReturn(
		func(ctx context.Context, fn func(context.Context) error) error {
			return fn(ctx)
		},
	)

	repo.EXPECT().GetByID(gomock.Any(), tripID).Return(trip.Trip{
		ID:     tripID,
		Status: trip.StatusActive,
	}, nil)

	repo.EXPECT().Save(gomock.Any(), gomock.Any()).DoAndReturn(
		func(ctx context.Context, tr trip.Trip) error {
			require.Equal(t, trip.StatusCompleted, tr.Status)
			require.NotNil(t, tr.FinishedAt)
			return nil
		},
	)

	repo.EXPECT().SaveStatusHistory(gomock.Any(), tripID, trip.StatusActive, trip.StatusCompleted).Return(nil)
	m.EXPECT().IncCompleted(gomock.Any()).Return()

	uc := tripuc.NewFinish(repo, tx, m, log)

	res, err := uc.Finish(context.Background(), tripID)

	require.NoError(t, err)
	require.Equal(t, tripID, res.ID)
	require.Equal(t, trip.StatusCompleted, res.Status)
}

func TestFinish_AlreadyCompleted(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repo := NewMocktripRepository(ctrl)
	tx := NewMocktxManager(ctrl)
	m := NewMockmetrics(ctrl)
	log := slog.Default()

	tripID := "123e4567-e89b-12d3-a456-426614174000"

	tx.EXPECT().Do(gomock.Any(), gomock.Any()).DoAndReturn(
		func(ctx context.Context, fn func(context.Context) error) error {
			return fn(ctx)
		},
	)

	repo.EXPECT().GetByID(gomock.Any(), tripID).Return(trip.Trip{
		ID:     tripID,
		Status: trip.StatusCompleted,
	}, nil)

	uc := tripuc.NewFinish(repo, tx, m, log)
	_, err := uc.Finish(context.Background(), tripID)

	require.ErrorIs(t, err, trip.ErrTripCompleted)
}

func TestFinish_TripNotFound(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repo := NewMocktripRepository(ctrl)
	tx := NewMocktxManager(ctrl)
	m := NewMockmetrics(ctrl)
	log := slog.Default()

	tripID := "123e4567-e89b-12d3-a456-426614174000"

	tx.EXPECT().Do(gomock.Any(), gomock.Any()).DoAndReturn(
		func(ctx context.Context, fn func(context.Context) error) error {
			return fn(ctx)
		},
	)

	repo.EXPECT().GetByID(gomock.Any(), tripID).Return(trip.Trip{}, trip.ErrTripNotFound)

	uc := tripuc.NewFinish(repo, tx, m, log)
	_, err := uc.Finish(context.Background(), tripID)

	require.ErrorIs(t, err, trip.ErrTripNotFound)
}

func TestFinish_InvalidUUID(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repo := NewMocktripRepository(ctrl)
	tx := NewMocktxManager(ctrl)
	m := NewMockmetrics(ctrl)
	log := slog.Default()

	uc := tripuc.NewFinish(repo, tx, m, log)

	_, err := uc.Finish(context.Background(), "not-a-uuid")

	var valErr trip.ValidationError
	require.ErrorAs(t, err, &valErr)
	require.Equal(t, "trip_id", valErr.Field)
}