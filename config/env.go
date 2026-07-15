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
		AWS: AWSConfig{
			Region:              mustGetEnv("AWS_REGION"),
			AccessKeyID:         mustGetEnv("AWS_ACCESS_KEY_ID"),
			SecretAccessKey:     mustGetEnv("AWS_SECRET_ACCESS_KEY"),
			SQSEndpoint:         mustGetEnv("AWS_SQS_ENDPOINT"),
			SQSCategoryQueueURL: mustGetEnv("SQS_CATEGORY_QUEUE_URL"),
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
		Env: getEnv("APP_ENV", "development"),
	}, nil
}
