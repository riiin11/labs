package trip

import (
	"errors"

	"github.com/jackc/pgx/v5/pgconn"

	"github.com/riiin11/labs/internal/domain/trip"
)

func mapInsertError(err error) error {
	if err == nil {
		return nil
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		return trip.ErrDriverBusy
	}
	return err
}