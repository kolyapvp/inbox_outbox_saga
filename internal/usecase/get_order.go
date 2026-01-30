package usecase

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"project/internal/infrastructure/postgres"

	"github.com/redis/go-redis/v9"
)

type OrderDTO struct {
	ID          string    `json:"id"`
	UserID      string    `json:"user_id"`
	TotalAmount float64   `json:"total_amount"`
	Status      string    `json:"status"`
	FromCity    string    `json:"from_city"`
	ToCity      string    `json:"to_city"`
	TravelDate  string    `json:"travel_date"`
	TravelTime  string    `json:"travel_time"`
	Airline     string    `json:"airline"`
	CreatedAt   time.Time `json:"created_at"`
}

type GetOrder struct {
	redisClient *redis.Client
	orderRepo   *postgres.OrderRepository
}

func NewGetOrder(redisClient *redis.Client, orderRepo *postgres.OrderRepository) *GetOrder {
	return &GetOrder{
		redisClient: redisClient,
		orderRepo:   orderRepo,
	}
}

func (uc *GetOrder) Execute(ctx context.Context, orderID string) (*OrderDTO, error) {
	cacheKey := fmt.Sprintf("order:%s", orderID)

	if uc.redisClient != nil {
		val, err := uc.redisClient.Get(ctx, cacheKey).Result()
		if err == nil {
			var order OrderDTO
			if err := json.Unmarshal([]byte(val), &order); err == nil {
				return &order, nil
			}
		}
	}

	dbOrder, err := uc.orderRepo.GetByID(ctx, orderID)
	if err != nil {
		return nil, fmt.Errorf("get order: %w", err)
	}

	order := &OrderDTO{
		ID:          dbOrder.ID,
		UserID:      dbOrder.UserID,
		TotalAmount: dbOrder.TotalAmount,
		Status:      dbOrder.Status,
		FromCity:    dbOrder.FromCity,
		ToCity:      dbOrder.ToCity,
		TravelDate:  dbOrder.TravelDate,
		TravelTime:  dbOrder.TravelTime,
		Airline:     dbOrder.Airline,
		CreatedAt:   dbOrder.CreatedAt,
	}

	if uc.redisClient != nil {
		data, _ := json.Marshal(order)
		// TTL reduced to 1 second to allow quick status updates
		uc.redisClient.Set(ctx, cacheKey, data, 1*time.Second)
	}

	return order, nil
}
