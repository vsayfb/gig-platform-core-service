package bootstrap

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	awscfg "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/sqs"

	"github.com/vsayfb/gig-platform-core-service/config"
	"github.com/vsayfb/gig-platform-core-service/pkg/squs"
)

func newSQSPublisher(ctx context.Context, cfg *config.Config) (*squs.SQSPublisher, error) {
	awsConfig, err := awscfg.LoadDefaultConfig(
		ctx,
		awscfg.WithRegion("us-east-1"),
		awscfg.WithCredentialsProvider(
			credentials.NewStaticCredentialsProvider("test", "test", ""),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("load aws config: %w", err)
	}

	sqsClient := sqs.NewFromConfig(awsConfig, func(o *sqs.Options) {
		o.BaseEndpoint = aws.String(cfg.SQS.BaseURL)
	})

	return squs.NewSQSPublisher(sqsClient, cfg.SQS.QueueURL), nil
}
