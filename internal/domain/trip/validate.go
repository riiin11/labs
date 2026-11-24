package trip

import "github.com/google/uuid"

type CreateInput struct {
	UserID     string
	DriverID   string
	StartPoint Point
	EndPoint   Point
	Price      int64
}

func (in CreateInput) Validate() error {
	if _, err := uuid.Parse(in.UserID); err != nil {
		return ValidationError{Field: "user_id", Message: "must be a valid UUID"}
	}
	if _, err := uuid.Parse(in.DriverID); err != nil {
		return ValidationError{Field: "driver_id", Message: "must be a valid UUID"}
	}
	if in.StartPoint.Latitude < -90 || in.StartPoint.Latitude > 90 {
		return ValidationError{Field: "start_point.latitude", Message: "out of range"}
	}
	if in.StartPoint.Longitude < -180 || in.StartPoint.Longitude > 180 {
		return ValidationError{Field: "start_point.longitude", Message: "out of range"}
	}
	if in.EndPoint.Latitude < -90 || in.EndPoint.Latitude > 90 {
		return ValidationError{Field: "end_point.latitude", Message: "out of range"}
	}
	if in.EndPoint.Longitude < -180 || in.EndPoint.Longitude > 180 {
		return ValidationError{Field: "end_point.longitude", Message: "out of range"}
	}
	if in.Price < 0 {
		return ValidationError{Field: "price", Message: "must be >= 0"}
	}
	return nil
}