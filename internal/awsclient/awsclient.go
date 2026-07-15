package awsclient

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	awscfg "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/vsayfb/gig-platform-core-service/config"
)

func New(ctx context.Context, cfg *config.Config) (aws.Config, error) {
	if cfg.Env == config.EnvironmentProduction {
		return awscfg.LoadDefaultConfig(ctx)
	}

	return awscfg.LoadDefaultConfig(
		ctx,
		awscfg.WithRegion(cfg.AWS.Region),
		awscfg.WithCredentialsProvider(
			credentials.NewStaticCredentialsProvider(
				cfg.AWS.AccessKeyID,
				cfg.AWS.SecretAccessKey,
				"",
			),
		),
	)
}
