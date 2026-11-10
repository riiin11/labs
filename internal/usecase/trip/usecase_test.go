package trip_test

import (
 "context"
 "errors"
 "log/slog"
 "testing"

 "github.com/stretchr/testify/require"
 "go.uber.org/mock/gomock"

 "github.com/riiin11/labs/internal/domain/trip"
 tripuc "github.com/riiin11/labs/internal/usecase/trip"
)

func newTxMock(ctrl *gomock.Controller) *MocktxManager {
 tx := NewMocktxManager(ctrl)
 tx.EXPECT().Do(gomock.Any(), gomock.Any()).
  DoAndReturn(func(ctx context.Context, fn func(context.Context) error) error {
   return fn(ctx)
  }).AnyTimes()
 return tx
}

func validInput() trip.CreateInput {
 return trip.CreateInput{
  UserID:     "5cb72c04-7650-45c9-a79b-bcdba0631e0c",
  DriverID:   "8860b315-ec86-42eb-a17c-7c163d721ff5",
  StartPoint: trip.Point{Latitude: 59.9398, Longitude: 30.3146},
  EndPoint:   trip.Point{Latitude: 59.9290, Longitude: 30.3626},
  Price:      1450,
 }
}

func TestCreate(t *testing.T) {
 t.Run("success writes trip and history", func(t *testing.T) {
  ctrl := gomock.NewController(t)
  repo := NewMocktripRepository(ctrl)
  tx := newTxMock(ctrl)
  m := NewMockmetrics(ctrl)

  repo.EXPECT().Save(gomock.Any(), gomock.Any()).Return(nil)
  repo.EXPECT().SaveStatusHistory(gomock.Any(), gomock.Any(), trip.Status(""), trip.StatusActive).Return(nil)
  m.EXPECT().IncCreated(gomock.Any())

  uc := tripuc.NewCreate(repo, tx, m, slog.Default())
  got, err := uc.Create(context.Background(), validInput())

  require.NoError(t, err)
  require.Equal(t, trip.StatusActive, got.Status)
  require.NotEmpty(t, got.ID)
 })

 t.Run("driver busy", func(t *testing.T) {
  ctrl := gomock.NewController(t)
  repo := NewMocktripRepository(ctrl)
  tx := newTxMock(ctrl)
  m := NewMockmetrics(ctrl)

  repo.EXPECT().Save(gomock.Any(), gomock.Any()).Return(trip.ErrDriverBusy)

  uc := tripuc.NewCreate(repo, tx, m, slog.Default())
  _, err := uc.Create(context.Background(), validInput())

  require.ErrorIs(t, err, trip.ErrDriverBusy)
 })

 t.Run("history error propagates", func(t *testing.T) {
  ctrl := gomock.NewController(t)
  repo := NewMocktripRepository(ctrl)
  tx := newTxMock(ctrl)
  m := NewMockmetrics(ctrl)

  boom := errors.New("boom")
  repo.EXPECT().Save(gomock.Any(), gomock.Any()).Return(nil)
  repo.EXPECT().SaveStatusHistory(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(boom)

  uc := tripuc.NewCreate(repo, tx, m, slog.Default())
  _, err := uc.Create(context.Background(), validInput())

  require.ErrorIs(t, err, boom)
 })

 t.Run("validation error does not touch repo", func(t *testing.T) {
  ctrl := gomock.NewController(t)
  repo := NewMocktripRepository(ctrl)
  tx := newTxMock(ctrl)
  m := NewMockmetrics(ctrl)

  in := validInput()
  in.Price = -1

  uc := tripuc.NewCreate(repo, tx, m, slog.Default())
  _, err := uc.Create(context.Background(), in)

  var ve trip.ValidationError
  require.ErrorAs(t, err, &ve)
 })
}

func TestFinish(t *testing.T) {
 t.Run("success", func(t *testing.T) {
  ctrl := gomock.NewController(t)
  repo := NewMocktripRepository(ctrl)
  tx := newTxMock(ctrl)
  m := NewMockmetrics(ctrl)

  id := "1f0a9c62-4a1c-4f2e-9d33-2a4bb0f0b111"
  done := trip.Trip{ID: id, Status: trip.StatusCompleted}

  repo.EXPECT().Finish(gomock.Any(), id, gomock.Any()).Return(done, nil)
  repo.EXPECT().SaveStatusHistory(gomock.Any(), id, trip.StatusActive, trip.StatusCompleted).Return(nil)
  m.EXPECT().IncCompleted(gomock.Any())

  uc := tripuc.NewFinish(repo, tx, m, slog.Default())
  got, err := uc.Finish(context.Background(), id)

  require.NoError(t, err)
  require.Equal(t, trip.StatusCompleted, got.Status)
 })

 t.Run("already completed", func(t *testing.T) {
  ctrl := gomock.NewController(t)
  repo := NewMocktripRepository(ctrl)
  tx := newTxMock(ctrl)
  m := NewMockmetrics(ctrl)

  id := "1f0a9c62-4a1c-4f2e-9d33-2a4bb0f0b111"
  repo.EXPECT().Finish(gomock.Any(), id, gomock.Any()).Return(trip.Trip{}, trip.ErrTripCompleted)

  uc := tripuc.NewFinish(repo, tx, m, slog.Default())
  _, err := uc.Finish(context.Background(), id)

  require.ErrorIs(t, err, trip.ErrTripCompleted)
 })

 t.Run("not found", func(t *testing.T) {
  ctrl := gomock.NewController(t)
  repo:= NewMocktripRepository(ctrl)
  tx := newTxMock(ctrl)
  m := NewMockmetrics(ctrl)

  id := "1f0a9c62-4a1c-4f2e-9d33-2a4bb0f0b111"
  repo.EXPECT().Finish(gomock.Any(), id, gomock.Any()).Return(trip.Trip{}, trip.ErrTripNotFound)

  uc := tripuc.NewFinish(repo, tx, m, slog.Default())
  _, err := uc.Finish(context.Background(), id)

  require.ErrorIs(t, err, trip.ErrTripNotFound)
 })

 t.Run("bad uuid", func(t *testing.T) {
  ctrl := gomock.NewController(t)
  repo := NewMocktripRepository(ctrl)
  tx := newTxMock(ctrl)
  m := NewMockmetrics(ctrl)

  uc := tripuc.NewFinish(repo, tx, m, slog.Default())
  _, err := uc.Finish(context.Background(), "not-uuid")

  var ve trip.ValidationError
  require.ErrorAs(t, err, &ve)
 })

 t.Run("repo error propagates", func(t *testing.T) {
  ctrl := gomock.NewController(t)
  repo := NewMocktripRepository(ctrl)
  tx := newTxMock(ctrl)
  m := NewMockmetrics(ctrl)

  id := "1f0a9c62-4a1c-4f2e-9d33-2a4bb0f0b111"
  boom := errors.New("boom")
  repo.EXPECT().Finish(gomock.Any(), id, gomock.Any()).Return(trip.Trip{}, boom)

  uc := tripuc.NewFinish(repo, tx, m, slog.Default())
  _, err := uc.Finish(context.Background(), id)

  require.ErrorIs(t, err, boom)
 })
}

func TestGet(t *testing.T) {
 t.Run("not found", func(t *testing.T) {
  ctrl := gomock.NewController(t)
  repo := NewMocktripRepository(ctrl)

  id := "1f0a9c62-4a1c-4f2e-9d33-2a4bb0f0b111"
  repo.EXPECT().GetByID(gomock.Any(), id).Return(trip.Trip{}, trip.ErrTripNotFound)

  uc := tripuc.NewGet(repo)
  _, err := uc.Get(context.Background(), id)

  require.ErrorIs(t, err, trip.ErrTripNotFound)
 })

 t.Run("success", func(t *testing.T) {
  ctrl := gomock.NewController(t)
  repo := NewMocktripRepository(ctrl)

  id := "1f0a9c62-4a1c-4f2e-9d33-2a4bb0f0b111"
  want := trip.Trip{ID: id, Status: trip.StatusActive}
  repo.EXPECT().GetByID(gomock.Any(), id).Return(want, nil)

  uc := tripuc.NewGet(repo)
  got, err := uc.Get(context.Background(), id)

  require.NoError(t, err)
  require.Equal(t, want, got)
 })

 t.Run("bad uuid", func(t *testing.T) {
  ctrl := gomock.NewController(t)
  repo := NewMocktripRepository(ctrl)

  uc := tripuc.NewGet(repo)
  _, err := uc.Get(context.Background(), "x")

  var ve trip.ValidationError
  require.ErrorAs(t, err, &ve)
 })
}