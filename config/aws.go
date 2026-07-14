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

	rdsSecretARN := params["rds-secret-arn"]

	var dbSecret rdsSecret

	if err := loadSecret(ctx, secretClient, rdsSecretARN, &dbSecret); err != nil {
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
			Name:     params["db-name"],
			SSLMode:  getOrDefault(params, "db-sslmode", "require"),
			User:     dbSecret.Username,
			Password: dbSecret.Password,
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
			MetricsServerPort: getOrDefault(params, "metrics-server-port", ":9100"),
			ServiceName:       getOrDefault(params, "service-name", "core-service"),
			OTelCollectorAddr: getOrDefault(params, "otel-collector-addr", "localhost:4317"),
		},
		GRPC: GRPCConfig{
			Port: getOrDefault(params, "grpc-port", "9090"),
		},
		SQS: SQS{
			QueueURL: params["sqs-category-events-queue-url"],
		},
		Env: getOrDefault(params, "env", "production"),
	}, nil
}

func loadParameters(ctx context.Context, client *ssm.Client) (map[string]string, error) {
	params := make(map[string]string)

	var nextToken *string

	for {
		out, err := client.GetParametersByPath(ctx, &ssm.GetParametersByPathInput{
			Path:           aws.String(parameterPath),
			Recursive:      aws.Bool(true),
			WithDecryption: aws.Bool(true),
			NextToken:      nextToken,
		})

		if err != nil {
			return nil, fmt.Errorf("read parameter store: %w", err)
		}

		for _, p := range out.Parameters {
			name := strings.TrimPrefix(aws.ToString(p.Name), parameterPath+"/")
			params[name] = aws.ToString(p.Value)
		}

		if out.NextToken == nil {
			break
		}

		nextToken = out.NextToken
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
