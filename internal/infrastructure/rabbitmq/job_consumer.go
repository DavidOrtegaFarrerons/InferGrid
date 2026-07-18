package rabbitmq

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/DavidOrtegaFarrerons/infergrid/internal/application"
	"github.com/DavidOrtegaFarrerons/infergrid/internal/domain/job"
	amqp "github.com/rabbitmq/amqp091-go"
)

type JobConsumer struct {
	channel           *amqp.Channel
	processJobService *application.ProcessJobService
}

func NewJobConsumer(connection *amqp.Connection, processJobService *application.ProcessJobService) (*JobConsumer, error) {
	channel, err := openJobChannel(connection)
	if err != nil {
		return nil, err
	}

	if err = channel.Qos(1, 0, false); err != nil {
		channel.Close()
		return nil, fmt.Errorf("configure consumer QoS: %w", err)
	}

	return &JobConsumer{channel: channel, processJobService: processJobService}, nil
}

func (c *JobConsumer) Run(ctx context.Context) error {
	deliveries, err := c.channel.ConsumeWithContext(
		ctx,
		jobQueue,
		"",
		false,
		false,
		false,
		false,
		nil,
	)

	if err != nil {
		return fmt.Errorf("consume job messages: %w", err)
	}

	for delivery := range deliveries {
		var message JobMessage
		if err := json.Unmarshal(delivery.Body, &message); err != nil {
			if nackErr := delivery.Nack(false, false); nackErr != nil {
				return fmt.Errorf("reject invalid message: %w", nackErr)
			}

			continue
		}

		message.JobID = strings.TrimSpace(message.JobID)

		if message.JobID == "" {
			if nackErr := delivery.Nack(false, false); nackErr != nil {
				return fmt.Errorf("reject empty job ID: %w", nackErr)
			}

			continue
		}

		req := application.ProcessJobRequest{JobID: job.ID(message.JobID)}
		processErr := c.processJobService.Execute(ctx, req)
		if processErr != nil {
			if errors.Is(processErr, application.ErrJobNotFound) {
				if nackErr := delivery.Nack(false, false); nackErr != nil {
					return fmt.Errorf("reject missing job: %w", nackErr)
				}

				continue
			}
			if nackErr := delivery.Nack(false, true); nackErr != nil {
				return fmt.Errorf("requeue job message: %w", nackErr)
			}

			log.Printf("job %s requeued: %v", message.JobID, processErr)
			time.Sleep(time.Second) //Will be a dead letter queue in the future, but works to test the Circuit Breaker
			continue
		}

		if ackErr := delivery.Ack(false); ackErr != nil {
			return fmt.Errorf("acknowledge job message: %w", ackErr)
		}
	}

	if ctx.Err() != nil {
		return ctx.Err()
	}

	return errors.New("RabbitMQ delivery channel closed")
}
