//go:generate go tool go.uber.org/mock/mockgen -source ${GOFILE} -destination mocks_test.go -package ${GOPACKAGE}_test

package trip

import (
	"context"

	"github.com/riiin11/labs/internal/domain/trip"
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