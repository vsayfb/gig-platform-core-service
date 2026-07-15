package config

import "time"

func loadEnv() (*Config, error) {
	return &Config{
		DB: DBConfig{
			Host:     mustGetEnv(EnvDBHost),
			Port:     mustGetEnv(EnvDBPort),
			User:     mustGetEnv(EnvDBUser),
			Password: mustGetEnv(EnvDBPassword),
			Name:     mustGetEnv(EnvDBName),
			SSLMode:  getEnv(EnvDBSSLMode, DefaultSSLMode),
		},
		AWS: AWSConfig{
			Region:              mustGetEnv(EnvAWSRegion),
			AccessKeyID:         mustGetEnv(EnvAWSAccessKeyID),
			SecretAccessKey:     mustGetEnv(EnvAWSSecretAccessKey),
			SQSEndpoint:         mustGetEnv(EnvSQSEndpoint),
			SQSCategoryQueueURL: mustGetEnv(EnvSQSCategoryQueueURL),
		},
		JWT: JWTConfig{
			Secret:     mustGetEnv(EnvJWTSecret),
			Expiration: 24 * time.Hour,
		},
		Google: GoogleConfig{
			ClientID: mustGetEnv(EnvGoogleClientID),
		},
		REST: ServerConfig{
			Port:              getEnv(EnvRESTPort, DefaultRestPORT),
			MetricsServerPort: getEnv(EnvMetricsServerPort, DefaultMetricsServerPort),
			ServiceName:       getEnv(EnvServiceName, DefaultServiceName),
			OTelCollectorAddr: getEnv(EnvOTelCollectorAddr, DefaultOtelCollectorAddr),
		},
		GRPC: GRPCConfig{
			Port: getEnv(EnvGRPCPort, DefaultGRPCPORT),
		},
		Env: getEnv(AppEnv, EnvironmentDevelopment),
	}, nil
}
