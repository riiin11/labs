//go:build integration

package trip_test

import (
 "context"
 "log/slog"
 "os"
 "testing"
 "time"

 "github.com/google/uuid"
 "github.com/jackc/pgx/v5/pgxpool"
 "github.com/stretchr/testify/require"

 "github.com/riiin11/labs/internal/domain/trip"
 "github.com/riiin11/labs/internal/infra"
 triprepo "github.com/riiin11/labs/internal/repository/trip"
)

func testPool(t *testing.T) *pgxpool.Pool {
 t.Helper()
 dsn := os.Getenv("DATABASE_URL")
 if dsn == "" {
  t.Skip("DATABASE_URL not set")
 }
 pool, err := pgxpool.New(context.Background(), dsn)
 require.NoError(t, err)
 t.Cleanup(pool.Close)
 return pool
}

func newRepo(pool *pgxpool.Pool) *triprepo.Repo {
 return triprepo.New(pool, slog.Default())
}

func sampleTrip() trip.Trip {
 return trip.Trip{
  ID:         uuid.NewString(),
  UserID:     uuid.NewString(),
  DriverID:   uuid.NewString(),
  StartPoint: trip.Point{Latitude: 59.9, Longitude: 30.3},
  EndPoint:   trip.Point{Latitude: 59.8, Longitude: 30.4},
  Price:      1450,
  Status:     trip.StatusActive,
  StartedAt:  time.Now().UTC(),
 }
}

func TestRepo_SaveAndGet(t *testing.T) {
 pool := testPool(t)
 repo := newRepo(pool)
 ctx := context.Background()

 tr := sampleTrip()
 require.NoError(t, repo.Save(ctx, tr))
 t.Cleanup(func() { _, _ = pool.Exec(ctx, `DELETE FROM trips WHERE id=$1`, tr.ID) })

 got, err := repo.GetByID(ctx, tr.ID)
 require.NoError(t, err)
 require.Equal(t, tr.ID, got.ID)
 require.Equal(t, trip.StatusActive, got.Status)
}

func TestRepo_Save_DriverBusy(t *testing.T) {
 pool := testPool(t)
 repo := newRepo(pool)
 ctx := context.Background()

 first := sampleTrip()
 require.NoError(t, repo.Save(ctx, first))
 t.Cleanup(func() { _, _ = pool.Exec(ctx, `DELETE FROM trips WHERE driver_id=$1`, first.DriverID) })

 second := sampleTrip()
 second.DriverID = first.DriverID

 err := repo.Save(ctx, second)
 require.ErrorIs(t, err, trip.ErrDriverBusy)
}

func TestRepo_SaveStatusHistory(t *testing.T) {
 pool := testPool(t)
 repo := newRepo(pool)
 ctx := context.Background()

 tr := sampleTrip()
 require.NoError(t, repo.Save(ctx, tr))
 t.Cleanup(func() { _, _ = pool.Exec(ctx, `DELETE FROM trips WHERE id=$1`, tr.ID) })

 require.NoError(t, repo.SaveStatusHistory(ctx, tr.ID, "", trip.StatusActive))
 require.NoError(t, repo.SaveStatusHistory(ctx, tr.ID, trip.StatusActive, trip.StatusCompleted))

 var n int
 require.NoError(t, pool.QueryRow(ctx,
  `SELECT count(*) FROM trip_status_history WHERE trip_id=$1`, tr.ID).Scan(&n))
 require.Equal(t, 2, n)
}

func TestRepo_Finish(t *testing.T) {
 pool := testPool(t)
 repo := newRepo(pool)
 ctx := context.Background()

 tr := sampleTrip()
 require.NoError(t, repo.Save(ctx, tr))
 t.Cleanup(func() { _, _ = pool.Exec(ctx, `DELETE FROM trips WHERE id=$1`, tr.ID) })

 done, err := repo.Finish(ctx, tr.ID, time.Now().UTC())
 require.NoError(t, err)
 require.Equal(t, trip.StatusCompleted, done.Status)
 require.NotNil(t, done.FinishedAt)

 _, err = repo.Finish(ctx, tr.ID, time.Now().UTC())
 require.ErrorIs(t, err, trip.ErrTripCompleted)

 _, err = repo.Finish(ctx, uuid.NewString(), time.Now().UTC())
 require.ErrorIs(t, err, trip.ErrTripNotFound)
}

func TestTxManager_Rollback(t *testing.T) {
 pool := testPool(t)
 repo := newRepo(pool)
 tx := infra.NewTxManager(pool)
 ctx := context.Background()

 tr := sampleTrip()
 err := tx.Do(ctx, func(ctx context.Context) error {
  require.NoError(t, repo.Save(ctx, tr))
  return context.Canceled
 })
 require.Error(t, err)

 var n int
 require.NoError(t, pool.QueryRow(ctx,
 ` SELECT count(*) FROM trips WHERE id=$1`, tr.ID).Scan(&n))
 require.Zero(t, n)
}