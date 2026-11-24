//go:generate go tool go.uber.org/mock/mockgen -source ${GOFILE} -destination mocks_test.go -package ${GOPACKAGE}_test

package trip

import (
	"context"
	"time"

	"github.com/riiin11/labs/internal/domain/trip"
)

type (
	tripRepository interface {
		Save(ctx context.Context, t trip.Trip) error
		SaveStatusHistory(ctx context.Context, tripID string, from, to trip.Status) error
		GetByID(ctx context.Context, id string) (trip.Trip, error)

		SavePosition(ctx context.Context, p trip.Position) (trip.Position, error)
		GetPositions(ctx context.Context, tripID string) ([]trip.Position, error)
		GetStaleTrips(ctx context.Context, staleAfter time.Duration) ([]trip.Trip, error)
	}

	txManager interface {
		Do(ctx context.Context, fn func(ctx context.Context) error) error
	}

	metrics interface {
		IncCreated(ctx context.Context)
		IncCompleted(ctx context.Context)
		
		IncPositionsSaved(ctx context.Context)
	}
)