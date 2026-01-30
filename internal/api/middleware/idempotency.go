package middleware

import (
	"fmt"
	"net/http"
	"time"

	"github.com/redis/go-redis/v9"
)

func Idempotency(redisClient *redis.Client) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Only apply to state-changing methods
			if r.Method != http.MethodPost && r.Method != http.MethodPut && r.Method != http.MethodPatch {
				next.ServeHTTP(w, r)
				return
			}

			key := r.Header.Get("Idempotency-Key")
			if key == "" {
				next.ServeHTTP(w, r)
				return
			}

			idemKey := fmt.Sprintf("idempotency:%s", key)
			ctx := r.Context()

			// 1. Check if key exists (Success or In-Progress)
			val, err := redisClient.Get(ctx, idemKey).Result()
			if err == nil {
				// Already processed
				w.Header().Set("X-Idempotency-Hit", "true")
				w.WriteHeader(http.StatusConflict) // Or 200 OK depending on requirement
				w.Write([]byte(fmt.Sprintf(`{"error": "request already processed", "original_response": %s}`, val)))
				return
			} else if err != redis.Nil {
				// Redis error
				next.ServeHTTP(w, r)
				return
			}

			// 2. Lock key (In-Progress)
			// Using SETNX with a short TTL to prevent forever-lock if crash
			acquired, err := redisClient.SetNX(ctx, idemKey, "PROCESSING", 10*time.Second).Result()
			if err != nil || !acquired {
				w.WriteHeader(http.StatusConflict)
				w.Write([]byte(`{"error": "concurrent request"}`))
				return
			}

			// 3. Process Request
			// We need to capture the response to store it.
			// Currently simple implementation: we just mark as "COMPLETED" after success.
			// For full idempotency, we would wrap ResponseWriter.

			next.ServeHTTP(w, r)

			// 4. Update Key to "COMPLETED" (or store actual response)
			// Extending TTL to 24h
			redisClient.Set(ctx, idemKey, "\"COMPLETED\"", 24*time.Hour)
		})
	}
}
