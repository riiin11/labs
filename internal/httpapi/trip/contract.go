//go:generate mockgen -source ${GOFILE} -destination mocks_test.go -package ${GOPACKAGE}_test

package trip

import (
	"context"

	"github.com/you/trip-service/internal/domain/trip"
)

type (
	createUsecase interface {
		Create(ctx context.Context, in trip.CreateInput) (trip.Trip, error)
	}
	getUsecase interface {
		Get(ctx context.Context, id string) (trip.Trip, error)
	}
	finishUsecase interface {
		Finish(ctx context.Context, id string) (trip.Trip, error)
	}
)