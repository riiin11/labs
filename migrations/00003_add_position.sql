-- +goose Up
ALTER TABLE trips ADD COLUMN last_position_at TIMESTAMPTZ;

CREATE TABLE trip_positions (
    id           BIGSERIAL PRIMARY KEY,
    trip_id      UUID NOT NULL REFERENCES trips(id) ON DELETE CASCADE,
    latitude     DOUBLE PRECISION NOT NULL,
    longitude    DOUBLE PRECISION NOT NULL,
    recorded_at  TIMESTAMPTZ NOT NULL
);

CREATE INDEX idx_trip_positions_trip_recorded
    ON trip_positions (trip_id, recorded_at, id);

-- +goose Down
DROP INDEX IF EXISTS idx_trip_positions_trip_recorded;
DROP TABLE IF EXISTS trip_positions;
ALTER TABLE trips DROP COLUMN IF EXISTS last_position_at;