package trip

import (
 "context"
 "errors"
 "time"

 sq "github.com/Masterminds/squirrel"
 "github.com/google/uuid"
 "github.com/jackc/pgx/v5"

 "github.com/riiin11/labs/internal/domain/trip"
)

var tripColumns = []string{
 "id", "user_id", "driver_id",
 "start_latitude", "start_longitude", "end_latitude", "end_longitude",
 "price", "status", "started_at", "finished_at",
}

func (r *Repo) Save(ctx context.Context, t trip.Trip) error {
 sql, args, err := sq.Insert("trips").
  Columns("id", "user_id", "driver_id",
   "start_latitude", "start_longitude", "end_latitude", "end_longitude",
   "price", "status", "started_at").
  Values(
   uuid.MustParse(t.ID), uuid.MustParse(t.UserID), uuid.MustParse(t.DriverID),
   t.StartPoint.Latitude, t.StartPoint.Longitude,
   t.EndPoint.Latitude, t.EndPoint.Longitude,
   t.Price, string(t.Status), t.StartedAt).
  PlaceholderFormat(sq.Dollar).
  ToSql()
 if err != nil {
  r.log.ErrorContext(ctx, "build Save sql failed", "err", err)
  return err
 }
 _, err = r.executor(ctx).Exec(ctx, sql, args...)
 if err != nil {
  r.log.ErrorContext(ctx, "exec Save failed", "err", err, "sql", sql, "trip_id", t.ID)
  return mapInsertError(err)
 }
 return nil
}

func (r *Repo) SaveStatusHistory(ctx context.Context, tripID string, from, to trip.Status) error {
 sql, args, err := sq.Insert("trip_status_history").
  Columns("trip_id", "from_status", "to_status", "changed_at").
  Values(uuid.MustParse(tripID), statusPtr(from), string(to), time.Now().UTC()).
  PlaceholderFormat(sq.Dollar).
  ToSql()
 if err != nil {
  r.log.ErrorContext(ctx, "build SaveStatusHistory sql failed", "err", err)
  return err
 }
 _, err = r.executor(ctx).Exec(ctx, sql, args...)
 if err != nil {
  r.log.ErrorContext(ctx, "exec SaveStatusHistory failed", "err", err, "sql", sql, "trip_id", tripID)
 }
 return err
}

func statusPtr(s trip.Status) any {
 if s == "" {
  return nil
 }
 return string(s)
}

func (r *Repo) GetByID(ctx context.Context, id string) (trip.Trip, error) {
 sql, args, err := sq.Select(tripColumns...).
  From("trips").
  Where(sq.Eq{"id": uuid.MustParse(id)}).
  PlaceholderFormat(sq.Dollar).
  ToSql()
 if err != nil {
  return trip.Trip{}, err
 }
 row := r.executor(ctx).QueryRow(ctx, sql, args...)
 t, err := scanTrip(row)
 if errors.Is(err, pgx.ErrNoRows) {
  return trip.Trip{}, trip.ErrTripNotFound
 }
 return t, err
}

func (r *Repo) Finish(ctx context.Context, id string, at time.Time) (trip.Trip, error) {
 sql, args, err := sq.Update("trips").
  Set("status", string(trip.StatusCompleted)).
  Set("finished_at", at).
  Set("updated_at", at).
  Where(sq.Eq{"id": uuid.MustParse(id), "status": string(trip.StatusActive)}).
  Suffix("RETURNING " + returningColumns()).
  PlaceholderFormat(sq.Dollar).
  ToSql()
 if err != nil {
  return trip.Trip{}, err
 }

 done, err := scanTrip(r.executor(ctx).QueryRow(ctx, sql, args...))
 if err == nil {
  return done, nil
 }
 if !errors.Is(err, pgx.ErrNoRows) {
  return trip.Trip{}, err
 }

 checkSQL, checkArgs, _ := sq.Select("status").
  From("trips").
  Where(sq.Eq{"id": uuid.MustParse(id)}).
  PlaceholderFormat(sq.Dollar).
  ToSql()

 var status string
 if err := r.executor(ctx).QueryRow(ctx, checkSQL, checkArgs...).Scan(&status); err != nil {
  if errors.Is(err, pgx.ErrNoRows) {
   return trip.Trip{}, trip.ErrTripNotFound
  }
  return trip.Trip{}, err
 }
 return trip.Trip{}, trip.ErrTripCompleted
}

func returningColumns() string {
 out := ""
 for i, c := range tripColumns {
  if i > 0 {
   out += ", "
  }
  out += c
 }
 return out
}