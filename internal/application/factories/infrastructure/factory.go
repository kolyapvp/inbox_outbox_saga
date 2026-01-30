package infrastructure

import (
	"context"
	"fmt"
	"time"

	"project/internal/config"
	"project/internal/infrastructure/postgres"
	"project/internal/infrastructure/redis"

	pgxpool "github.com/jackc/pgx/v5/pgxpool"
	go_redis "github.com/redis/go-redis/v9"
)

type Factory struct {
	cfg      *config.Config
	pgPool   *pgxpool.Pool
	redisCli *go_redis.Client
}

func NewFactory(cfg *config.Config) *Factory {
	return &Factory{
		cfg: cfg,
	}
}

func (f *Factory) Postgres(ctx context.Context) (*pgxpool.Pool, error) {
	if f.pgPool != nil {
		return f.pgPool, nil
	}

	var pool *pgxpool.Pool
	var err error

	// Retry connection up to 5 times
	for i := 0; i < 5; i++ {
		pool, err = postgres.NewClient(ctx, postgres.Config{
			Host:     f.cfg.Postgres.Host,
			Port:     f.cfg.Postgres.Port,
			User:     f.cfg.Postgres.User,
			Password: f.cfg.Postgres.Password,
			DBName:   f.cfg.Postgres.DBName,
		})
		if err == nil {
			break
		}
		fmt.Printf("Failed to connect to postgres (attempt %d/5): %v. Retrying in 2s...\n", i+1, err)
		time.Sleep(2 * time.Second)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to init postgres after retries: %w", err)
	}

	f.pgPool = pool
	return pool, nil
}

func (f *Factory) Redis(ctx context.Context) (*go_redis.Client, error) {
	if f.redisCli != nil {
		return f.redisCli, nil
	}

	client, err := redis.NewClient(ctx, redis.Config{
		Addr: f.cfg.Redis.Addr,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to init redis: %w", err)
	}

	f.redisCli = client
	return client, nil
}

func (f *Factory) Close() {
	if f.pgPool != nil {
		f.pgPool.Close()
	}
	if f.redisCli != nil {
		f.redisCli.Close()
	}
}
