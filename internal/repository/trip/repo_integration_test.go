//go:build integration

package trip_test

import (
 "context"
 "errors"
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

func insertTrip(t *testing.T, pool *pgxpool.Pool, tr trip.Trip) {
 t.Helper()
 _, err := pool.Exec(context.Background(), `
  INSERT INTO trips (id, user_id, driver_id, start_latitude, start_longitude, end_latitude, end_longitude, price, status, started_at)
  VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
 `, tr.ID, tr.UserID, tr.DriverID, tr.StartPoint.Latitude, tr.StartPoint.Longitude, tr.EndPoint.Latitude, tr.EndPoint.Longitude, tr.Price, tr.Status, tr.StartedAt)
 require.NoError(t, err)
}

func TestRepo_SaveAndGet(t *testing.T) {
 pool := testPool(t)
 repo := newRepo(pool)
 ctx := context.Background()

 tr := sampleTrip()
 insertTrip(t, pool, tr)
 t.Cleanup(func() { _, _ = pool.Exec(ctx, `DELETE FROM trips WHERE id=$1`, tr.ID) })

 got, err := repo.GetByID(ctx, tr.ID)
 require.NoError(t, err)
 require.Equal(t, tr.ID, got.ID)
 require.Equal(t, trip.StatusActive, got.Status)
}

func TestRepo_Save_DriverBusy(t *testing.T) {
 pool := testPool(t)
 ctx := context.Background()

 first := sampleTrip()
 insertTrip(t, pool, first)
 t.Cleanup(func() { _, _ = pool.Exec(ctx, `DELETE FROM trips WHERE driver_id=$1`, first.DriverID) })

 second := sampleTrip()
 second.DriverID = first.DriverID

 _, err := pool.Exec(ctx, `
  INSERT INTO trips (id, user_id, driver_id, start_latitude, start_longitude, end_latitude, end_longitude, price, status, started_at)
  VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
 `, second.ID, second.UserID, second.DriverID, second.StartPoint.Latitude, second.StartPoint.Longitude, second.EndPoint.Latitude, second.EndPoint.Longitude, second.Price, second.Status, second.StartedAt)
 
 require.Error(t, err, "expected unique violation for driver with active trip")
}

func TestRepo_SaveStatusHistory(t *testing.T) {
 pool := testPool(t)
 repo := newRepo(pool)
 ctx := context.Background()

 tr := sampleTrip()
 insertTrip(t, pool, tr)
 t.Cleanup(func() { _, _ = pool.Exec(ctx, `DELETE FROM trips WHERE id=$1`, tr.ID) })

 require.NoError(t, repo.SaveStatusHistory(ctx, tr.ID, "", trip.StatusActive))
 require.NoError(t, repo.SaveStatusHistory(ctx, tr.ID, trip.StatusActive, trip.StatusCompleted))

 var n int
 require.NoError(t, pool.QueryRow(ctx, `SELECT count(*) FROM trip_status_history WHERE trip_id=$1`, tr.ID).Scan(&n))
 require.Equal(t, 2, n)
}

func TestTxManager_Rollback(t *testing.T) {
 pool := testPool(t)
 repo := newRepo(pool)
 tx := infra.NewTxManager(pool, 5*time.Second)
 ctx := context.Background()

 tr := sampleTrip()

 err := tx.Do(ctx, func(ctx context.Context) error {
  if err := repo.Create(ctx, tr); err != nil {
   return err
  }
  
  if err := repo.SaveStatusHistory(ctx, tr.ID, "", trip.StatusActive); err != nil {
   return err
  }

  return errors.New("forced rollback")
 })
 require.Error(t, err)

 var tripCount int
 require.NoError(t, pool.QueryRow(ctx, `SELECT count(*) FROM trips WHERE id=$1`, tr.ID).Scan(&tripCount))
 require.Zero(t, tripCount, "trips should be rolled back")

 var historyCount int
 require.NoError(t, pool.QueryRow(ctx, `SELECT count(*) FROM trip_status_history WHERE trip_id=$1`, tr.ID).Scan(&historyCount))
 require.Zero(t, historyCount, "trip_status_history should be rolled back")
}