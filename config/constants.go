package config

const (
	EnvironmentDevelopment = "development"
	EnvironmentProduction  = "production"
)

const (
	AppEnv         = "APP_ENV"
	EnvServiceName = "SERVICE_NAME"

	EnvDBHost     = "DB_HOST"
	EnvDBPort     = "DB_PORT"
	EnvDBUser     = "DB_USER"
	EnvDBPassword = "DB_PASSWORD"
	EnvDBName     = "DB_NAME"
	EnvDBSSLMode  = "DB_SSLMODE"

	EnvJWTSecret = "JWT_SECRET"

	EnvRESTPort          = "REST_PORT"
	EnvGRPCPort          = "GRPC_PORT"
	EnvMetricsServerPort = "METRICS_SERVER_PORT"
	EnvOTelCollectorAddr = "OTEL_COLLECTOR_ADDR"

	EnvGoogleClientID = "GOOGLE_CLIENT_ID"

	EnvAWSRegion           = "AWS_REGION"
	EnvAWSAccessKeyID      = "AWS_ACCESS_KEY_ID"
	EnvAWSSecretAccessKey  = "AWS_SECRET_ACCESS_KEY"
	EnvSQSEndpoint         = "AWS_SQS_ENDPOINT"
	EnvSQSCategoryQueueURL = "SQS_CATEGORY_QUEUE_URL"
)

const (
	DefaultRestPORT          = "8080"
	DefaultGRPCPORT          = "9090"
	DefaultMetricsServerPort = ":9100"
	DefaultOtelCollectorAddr = "localhost:4317"
	DefaultServiceName       = "core-service"
	DefaultSSLMode           = "disable"
)
