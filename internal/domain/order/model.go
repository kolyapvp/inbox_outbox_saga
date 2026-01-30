package order

import (
	"time"
)

type Order struct {
	ID          string    `json:"id"`
	UserID      string    `json:"user_id"`
	Status      string    `json:"status"`
	TotalAmount float64   `json:"total_amount"`
	FromCity    string    `json:"from_city"`
	ToCity      string    `json:"to_city"`
	TravelDate  string    `json:"travel_date"` // YYYY-MM-DD
	TravelTime  string    `json:"travel_time"` // HH:MM
	Airline     string    `json:"airline"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}
