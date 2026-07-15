package config

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awscfg "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
)

const parameterPath = "/gerek/app"

type rdsSecret struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type jwtSecret struct {
	Secret string `json:"secret"`
}

func loadAWS(ctx context.Context) (*Config, error) {
	awsCfg, err := awscfg.LoadDefaultConfig(ctx)

	if err != nil {
		return nil, fmt.Errorf("load aws config: %w", err)
	}

	ssmClient := ssm.NewFromConfig(awsCfg)
	secretClient := secretsmanager.NewFromConfig(awsCfg)

	params, err := loadParameters(ctx, ssmClient)

	if err != nil {
		return nil, err
	}

	var dbSecret rdsSecret

	if err := loadSecret(ctx, secretClient, params["rds-secret-arn"], &dbSecret); err != nil {
		return nil, err
	}

	var jwt jwtSecret

	if err := loadSecret(ctx, secretClient, params["jwt-secret-arn"], &jwt); err != nil {
		return nil, err
	}

	return &Config{
		DB: DBConfig{
			Host:     params["db-host"],
			Port:     params["db-port"],
			User:     dbSecret.Username,
			Password: dbSecret.Password,
			Name:     params["db-name"],
			SSLMode:  "require",
		},
		JWT: JWTConfig{
			Secret:     jwt.Secret,
			Expiration: 24 * time.Hour,
		},
		Google: GoogleConfig{
			ClientID: params["google-client-id"],
		},
		REST: ServerConfig{
			Port:              getOrDefault(params, "rest-port", "8080"),
			ServiceName:       getOrDefault(params, "service-name", "core-service"),
			MetricsServerPort: getOrDefault(params, "metrics-server-port", "9091"),
			OTelCollectorAddr: getOrDefault(params, "otel-collector-addr", "localhost:4317"),
		},
		GRPC: GRPCConfig{
			Port: getOrDefault(params, "grpc-port", "9090"),
		},
		Env: "production",
	}, nil
}

func loadParameters(ctx context.Context, client *ssm.Client) (map[string]string, error) {
	names := []string{
		parameter("db-host"),
		parameter("db-port"),
		parameter("db-name"),
		parameter("google-client-id"),
		parameter("sqs-category-events-queue-url"),
		parameter("rds-secret-arn"),
		parameter("jwt-secret-arn"),
	}

	out, err := client.GetParameters(ctx, &ssm.GetParametersInput{
		Names:          names,
		WithDecryption: aws.Bool(true),
	})

	if err != nil {
		return nil, fmt.Errorf("read parameter store: %w", err)
	}

	params := make(map[string]string)

	for _, p := range out.Parameters {
		key := strings.TrimPrefix(aws.ToString(p.Name), parameterPath+"/")
		params[key] = aws.ToString(p.Value)
	}

	return params, nil
}

func loadSecret(
	ctx context.Context,
	client *secretsmanager.Client,
	name string,
	dst any,
) error {

	out, err := client.GetSecretValue(ctx, &secretsmanager.GetSecretValueInput{
		SecretId: aws.String(name),
	})

	if err != nil {
		return err
	}

	return json.Unmarshal([]byte(aws.ToString(out.SecretString)), dst)
}

func getOrDefault(values map[string]string, key, def string) string {
	envKey := strings.ToUpper(strings.ReplaceAll(key, "-", "_"))

	if v := os.Getenv(envKey); v != "" {
		return v
	}

	if v, ok := values[key]; ok && v != "" {
		return v
	}

	return def
}

func parameter(name string) string {
	return parameterPath + "/" + name
}
