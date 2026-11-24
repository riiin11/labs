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

// TestTxManager_Rollback — сценарий 8 раздела 5: внутри одной транзакции
// пишем и поездку, и её историю, затем возвращаем ошибку. После отката
// в ОБЕИХ таблицах не должно остаться строк.
func TestTxManager_Rollback(t *testing.T) {
	pool := testPool(t)
	repo := newRepo(pool)
	tx := infra.NewTxManager(pool, 5*time.Second)
	ctx := context.Background()

	tr := sampleTrip()
	historyFrom := trip.Status("")
	historyTo := trip.StatusActive

	err := tx.Do(ctx, func(ctx context.Context) error {
		require.NoError(t, repo.Save(ctx, tr))
		require.NoError(t, repo.SaveStatusHistory(ctx, tr.ID, historyFrom, historyTo))

		return errors.New("forced rollback")
	})
	require.Error(t, err)

	var tripCount int
	require.NoError(t, pool.QueryRow(ctx,
		`SELECT count(*) FROM trips WHERE id=$1`, tr.ID).Scan(&tripCount))
	require.Zero(t, tripCount, "trips should be rolled back")

	var historyCount int
	require.NoError(t, pool.QueryRow(ctx,
		`SELECT count(*) FROM trip_status_history WHERE trip_id=$1`, tr.ID).Scan(&historyCount))
	require.Zero(t, historyCount, "trip_status_history should be rolled back")
}