package trip_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/you/trip-service/internal/domain/trip"
)

func validInput() trip.CreateInput {
	return trip.CreateInput{
		UserID:     "5cb72c04-7650-45c9-a79b-bcdba0631e0c",
		DriverID:   "8860b315-ec86-42eb-a17c-7c163d721ff5",
		StartPoint: trip.Point{Latitude: 59.9398, Longitude: 30.3146},
		EndPoint:   trip.Point{Latitude: 59.9290, Longitude: 30.3626},
		Price:      1450,
	}
}

func TestValidate(t *testing.T) {
	cases := []struct {
		name   string
		mutate func(*trip.CreateInput)
		field  string
	}{
		{"ok", func(*trip.CreateInput) {}, ""},
		{"empty user_id", func(in *trip.CreateInput) { in.UserID = "" }, "user_id"},
		{"bad user_id", func(in *trip.CreateInput) { in.UserID = "not-uuid" }, "user_id"},
		{"empty driver_id", func(in *trip.CreateInput) { in.DriverID = "" }, "driver_id"},
		{"bad driver_id", func(in *trip.CreateInput) { in.DriverID = "x" }, "driver_id"},
		{"start lat", func(in *trip.CreateInput) { in.StartPoint.Latitude = 91 }, "start_point.latitude"},
		{"start lon", func(in *trip.CreateInput) { in.StartPoint.Longitude = -181 }, "start_point.longitude"},
		{"end lat", func(in *trip.CreateInput) { in.EndPoint.Latitude = -91 }, "end_point.latitude"},
		{"end lon", func(in *trip.CreateInput) { in.EndPoint.Longitude = 181 }, "end_point.longitude"},
		{"negative price", func(in *trip.CreateInput) { in.Price = -1 }, "price"},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			in := validInput()
			c.mutate(&in)
			err := in.Validate()
			if c.field == "" {
				require.NoError(t, err)
				return
			}
			var ve trip.ValidationError
			require.ErrorAs(t, err, &ve)
			require.Equal(t, c.field, ve.Field)
		})
	}
}