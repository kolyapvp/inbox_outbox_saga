package ticket

import "time"

type Ticket struct {
	ID         string    `json:"id"`
	OrderID    string    `json:"order_id"`
	FromCity   string    `json:"from_city"`
	ToCity     string    `json:"to_city"`
	TravelDate string    `json:"travel_date"` // YYYY-MM-DD
	TravelTime string    `json:"travel_time"`
	Airline    string    `json:"airline"`
	Status     string    `json:"status"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}
