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
			SSLMode:  getEnv("DB_SSLMODE", "require"),
		},
		JWT: JWTConfig{
			Secret:     mustGetEnv("JWT_SECRET"),
			Expiration: 24 * time.Hour,
		},
		Google: GoogleConfig{
			ClientID: mustGetEnv("GOOGLE_CLIENT_ID"),
		},
		REST: ServerConfig{
			Port:              getEnv("REST_PORT", "8080"),
			MetricsServerPort: getEnv("METRICS_SERVER_PORT", ":9100"),
			ServiceName:       getEnv("SERVICE_NAME", "core-service"),
			OTelCollectorAddr: getEnv("OTEL_COLLECTOR_ADDR", "localhost:4317"),
		},
		GRPC: GRPCConfig{
			Port: getEnv("GRPC_PORT", "9090"),
		},
		SQS: SQS{
			QueueURL: mustGetEnv("SQS_QUEUE_URL"),
			BaseURL:  mustGetEnv("SQS_QUEUE_URL"),
		},
		Env: getEnv("APP_ENV", "production"),
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

func getEnv(key, defaultValue string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultValue
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
