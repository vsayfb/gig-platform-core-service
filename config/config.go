package config

import (
	"fmt"
	"log"
	"os"
	"time"
)

type Config struct {
	DB     DBConfig
	JWT    JWTConfig
	Google GoogleConfig
	REST   ServerConfig
	GRPC   GRPCConfig
	Env    string
	SQS    SQS
}

type DBConfig struct {
	Host     string
	Port     string
	User     string
	Password string
	Name     string
	SSLMode  string
}

type JWTConfig struct {
	Secret     string
	Expiration time.Duration
}

type GoogleConfig struct {
	ClientID string
}

type ServerConfig struct {
	Port              string
	ServiceName       string
	MetricsServerPort string
	OTelCollectorAddr string
}

type GRPCConfig struct {
	Port string
}

type SQS struct {
	QueueURL string
	BaseURL  string
}

func Load() (*Config, error) {
	cfg := &Config{
		DB: DBConfig{
			Host:     mustGetEnv("DB_HOST"),
			Port:     mustGetEnv("DB_PORT"),
			User:     mustGetEnv("DB_USER"),
			Password: mustGetEnv("DB_PASSWORD"),
			Name:     mustGetEnv("DB_NAME"),
			SSLMode:  mustGetEnv("DB_SSLMODE"),
		},
		JWT: JWTConfig{
			Secret:     mustGetEnv("JWT_SECRET"),
			Expiration: 24 * time.Hour,
		},
		Google: GoogleConfig{
			ClientID: mustGetEnv("GOOGLE_CLIENT_ID"),
		},
		REST: ServerConfig{
			Port:              mustGetEnv("REST_PORT"),
			MetricsServerPort: mustGetEnv("METRICS_SERVER_PORT"),
			ServiceName:       mustGetEnv("SERVICE_NAME"),
			OTelCollectorAddr: mustGetEnv("OTEL_COLLECTOR_ADDR"),
		},
		GRPC: GRPCConfig{
			Port: mustGetEnv("GRPC_PORT"),
		},
		SQS: SQS{
			QueueURL: mustGetEnv("SQS_QUEUE_URL"),
			BaseURL:  mustGetEnv("SQS_QUEUE_URL"),
		},
		Env: mustGetEnv("APP_ENV"),
	}

	return cfg, nil
}

func mustGetEnv(key string) string {
	v := os.Getenv(key)

	if v == "" {
		log.Fatalf("missing required env var: %s", key)
	}
	return v
}

func (c *DBConfig) DSN() string {
	return fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		c.Host, c.Port, c.User, c.Password, c.Name, c.SSLMode,
	)
}

func (c *DBConfig) URL() string {
	return fmt.Sprintf(
		"postgres://%s:%s@%s:%s/%s?sslmode=%s",
		c.User, c.Password, c.Host, c.Port, c.Name, c.SSLMode,
	)
}
