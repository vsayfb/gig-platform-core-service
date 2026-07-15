package bootstrap

import (
	"context"
	"fmt"

	"github.com/vsayfb/gig-platform-core-service/config"
	"github.com/vsayfb/gig-platform-core-service/internal/awsclient"
	"github.com/vsayfb/gig-platform-core-service/pkg/squs"
)

func newSQSPublisher(ctx context.Context, cfg *config.Config) (*squs.SQSPublisher, error) {
	awsCfg, err := awsclient.New(ctx, cfg)

	if err != nil {
		return nil, fmt.Errorf("get aws client: %w", err)
	}

	sqsClient := awsclient.NewSQS(awsCfg, cfg)

	return squs.NewSQSPublisher(sqsClient, cfg.AWS.SQSCategoryQueueURL), nil
}
