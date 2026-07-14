package config

import (
	"context"
	"fmt"
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

func Load(ctx context.Context) (*Config, error) {
	env := getEnv("APP_ENV", "development")

	if env == "production" {
		return loadAWS(ctx)
	}

	return loadEnv()
}

func getEnv(key, defaultValue string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultValue
}

func mustGetEnv(key string) string {
	v := os.Getenv(key)

	if v == "" {
		panic(fmt.Sprintf("missing required env var: %s", key))
	}

	return v
}

func (c *DBConfig) DSN() string {
	return fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		c.Host,
		c.Port,
		c.User,
		c.Password,
		c.Name,
		c.SSLMode,
	)
}
