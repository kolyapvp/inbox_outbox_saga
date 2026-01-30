package api

import (
	"log"
	"net/http"

	"project/internal/api/middleware"

	"github.com/go-chi/chi/v5"
	ChiMiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/redis/go-redis/v9"
)

func NewRouter(h *Handlers, redisClient *redis.Client) http.Handler {
	r := chi.NewRouter()

	r.Use(ChiMiddleware.Logger)
	r.Use(ChiMiddleware.Recoverer)
	r.Use(ChiMiddleware.RequestID)

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	r.Route("/api/v1", func(r chi.Router) {
		// Add domain routes here
		// r.Mount("/users", userRouter)
	})

	// Idempotent Order Creation
	r.With(middleware.Idempotency(redisClient)).Post("/orders", h.CreateOrder)

	// Cached Order Get
	r.Get("/orders/{id}", h.GetOrder)
	r.Get("/orders/{id}/workflow", h.GetWorkflow)

	// Refund Order (Idempotent by nature of state machine usually, but could add middleware)
	r.Post("/orders/{id}/refund", h.RefundOrder)

	r.Handle("/metrics", promhttp.Handler())

	log.Println("Registered routes: POST /orders (Idempotent), GET /orders/{id} (Cached), GET /metrics")

	return r
}
