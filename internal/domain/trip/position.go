package trip

import "time"

type Position struct {
	ID          int64     `json:"id"`
	TripID      string    `json:"trip_id"`
	Latitude    float64   `json:"latitude"`
	Longitude   float64   `json:"longitude"`
	RecordedAt  time.Time `json:"recorded_at"`
}

// Validate проверяет координаты и время
func (p Position) Validate() error {
	if p.Latitude < -90 || p.Latitude > 90 {
		return ValidationError{Field: "latitude", Message: "must be between -90 and 90"}
	}
	if p.Longitude < -180 || p.Longitude > 180 {
		return ValidationError{Field: "longitude", Message: "must be between -180 and 180"}
	}
	if p.RecordedAt.After(time.Now().Add(time.Minute)) {
		return ValidationError{Field: "recorded_at", Message: "cannot be in the future"}
	}
	return nil
}