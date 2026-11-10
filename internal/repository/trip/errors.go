package trip

import (
	"errors"

	"github.com/jackc/pgx/v5/pgconn"

	"github.com/you/trip-service/internal/domain/trip"
)

func mapInsertError(err error) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		return trip.ErrDriverBusy
	}
	return err
}