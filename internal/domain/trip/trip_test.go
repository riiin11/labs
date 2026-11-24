package trip_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/riiin11/labs/internal/domain/trip"
)

func TestFinish(t *testing.T) {
	now := time.Date(2026, 8, 24, 12, 0, 0, 0, time.UTC)

	t.Run("active trip becomes completed", func(t *testing.T) {
		tr := trip.Trip{ID: "t1", Status: trip.StatusActive, StartedAt: now}
		err := tr.Finish(now.Add(18 * time.Minute))
		require.NoError(t, err)
		require.Equal(t, trip.StatusCompleted, tr.Status)
		require.NotNil(t, tr.FinishedAt)
		require.Equal(t, now.Add(18*time.Minute), *tr.FinishedAt)
	})

	t.Run("completed trip returns ErrTripCompleted", func(t *testing.T) {
		fin := now
		tr := trip.Trip{ID: "t1", Status: trip.StatusCompleted, FinishedAt: &fin}
		err := tr.Finish(now.Add(time.Hour))
		require.ErrorIs(t, err, trip.ErrTripCompleted)
		require.Equal(t, trip.StatusCompleted, tr.Status)
		require.Equal(t, fin, *tr.FinishedAt)
	})
}