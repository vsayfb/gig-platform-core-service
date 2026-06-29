package squs

import (
	"context"
	"encoding/json"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
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

		_, err = p.client.SendMessage(ctx, &sqs.SendMessageInput{
			QueueUrl:    aws.String(p.queueURL),
			MessageBody: aws.String(string(body)),
		})

		if err == nil {
			return nil
		}

		lastErr = err

		// backoff: 100ms, 200ms, 400ms
		time.Sleep(time.Duration(100*(1<<i)) * time.Millisecond)
	}

	return lastErr
}
