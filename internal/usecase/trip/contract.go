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
  Finish(ctx context.Context, id string, at time.Time) (trip.Trip, error)
 }

 txManager interface {
  Do(ctx context.Context, fn func(ctx context.Context) error) error
 }

 metrics interface {
  IncCreated(ctx context.Context)
  IncCompleted(ctx context.Context)
 }
)