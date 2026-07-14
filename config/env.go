package config

import "time"

func loadEnv() (*Config, error) {
	return &Config{
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
			BaseURL:  getEnv("SQS_BASE_URL", "http://localhost:4566"),
		},
		Env: getEnv("APP_ENV", "development"),
	}, nil
}
