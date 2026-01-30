package config

import (
	"fmt"

	"github.com/ilyakaznacheev/cleanenv"
)

type Config struct {
	App      App      `yaml:"app"`
	HTTP     HTTP     `yaml:"http"`
	Log      Log      `yaml:"log"`
	Postgres Postgres `yaml:"postgres"`
	Redis    Redis    `yaml:"redis"`
	Kafka    Kafka    `yaml:"kafka"`
}

type App struct {
	Name    string `yaml:"name" env:"APP_NAME" env-default:"project-api"`
	Version string `yaml:"version" env:"APP_VERSION" env-default:"1.0.0"`
}

type HTTP struct {
	Port string `yaml:"port" env:"HTTP_PORT" env-default:"8080"`
}

type Log struct {
	Level string `yaml:"level" env:"LOG_LEVEL" env-default:"info"`
}

type Postgres struct {
	Host     string `yaml:"host" env:"POSTGRES_HOST" env-default:"localhost"`
	Port     string `yaml:"port" env:"POSTGRES_PORT" env-default:"5432"`
	User     string `yaml:"user" env:"POSTGRES_USER" env-default:"user"`
	Password string `yaml:"password" env:"POSTGRES_PASSWORD" env-default:"password"`
	DBName   string `yaml:"dbname" env:"POSTGRES_DB" env-default:"project_db"`
}

type Redis struct {
	Addr string `yaml:"addr" env:"REDIS_ADDR" env-default:"localhost:6379"`
}

type Kafka struct {
	Brokers []string `yaml:"brokers" env:"KAFKA_BROKERS" env-default:"localhost:9092"`
	Topic   string   `yaml:"topic" env:"KAFKA_TOPIC" env-default:"orders-events"`
	GroupID string   `yaml:"group_id" env:"KAFKA_GROUP_ID" env-default:"orders-consumer-group-1"`
}

func New() (*Config, error) {
	cfg := &Config{}

	if err := cleanenv.ReadConfig("config.yaml", cfg); err != nil {
		// fallback to env vars if file not found
		if err := cleanenv.ReadEnv(cfg); err != nil {
			return nil, fmt.Errorf("config error: %w", err)
		}
	} else {
		// Allow env vars to override config file
		cleanenv.ReadEnv(cfg)
	}

	return cfg, nil
}
