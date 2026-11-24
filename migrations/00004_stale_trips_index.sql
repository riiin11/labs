-- +goose Up
CREATE INDEX idx_trips_stale_check
    ON trips (status, COALESCE(last_position_at, started_at))
    WHERE status = 'active';

-- +goose Down
DROP INDEX IF EXISTS idx_trips_stale_check;