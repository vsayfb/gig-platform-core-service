package squs

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/aws/aws-sdk-go-v2/service/sqs/types"
	"github.com/vsayfb/gig-platform-core-service/pkg/tracing"
)

type SQSPublisher struct {
	client   *sqs.Client
	queueURL string
}

func NewSQSPublisher(client *sqs.Client, queueURL string) *SQSPublisher {
	return &SQSPublisher{
		client:   client,
		queueURL: queueURL,
	}
}

func (p *SQSPublisher) Publish(ctx context.Context, event any) error {
	body, err := json.Marshal(event)

	if err != nil {
		return err
	}

	var lastErr error

	for i := range 3 {

		inp := &sqs.SendMessageInput{
			QueueUrl:          aws.String(p.queueURL),
			MessageBody:       aws.String(string(body)),
			MessageAttributes: make(map[string]types.MessageAttributeValue),
		}

		tracing.InjectTraceContext(ctx, inp.MessageAttributes)

		inf, err := p.client.SendMessage(ctx, inp)

		if err == nil {
			slog.Info("sqs message sent (gig):", "message", inf)

			return nil
		}

		lastErr = err

		// backoff: 100ms, 200ms, 400ms
		time.Sleep(time.Duration(100*(1<<i)) * time.Millisecond)
	}

	return lastErr
}
