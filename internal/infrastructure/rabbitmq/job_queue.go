package rabbitmq

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/DavidOrtegaFarrerons/infergrid/internal/domain/job"
	amqp "github.com/rabbitmq/amqp091-go"
)

var errPublishNotConfirmed = errors.New(
	"job message was not confirmed by RabbitMQ",
)

type JobQueue struct {
	channel *amqp.Channel
}

type JobMessage struct {
	JobID string `json:"job_id"`
}

func NewJobQueue(
	connection *amqp.Connection,
) (*JobQueue, error) {
	channel, err := openJobChannel(connection)
	if err != nil {
		return nil, err
	}

	if err = channel.Confirm(false); err != nil {
		channel.Close()
		return nil, fmt.Errorf("enable publisher confirms: %w", err)
	}

	return &JobQueue{channel: channel}, nil
}

func (q *JobQueue) Enqueue(
	ctx context.Context,
	id job.ID,
) error {
	body, err := json.Marshal(JobMessage{
		JobID: string(id),
	})
	if err != nil {
		return fmt.Errorf("encode job message: %w", err)
	}

	confirmation, err := q.channel.PublishWithDeferredConfirmWithContext(
		ctx,
		jobExchange,
		jobRoutingKey,
		false,
		false,
		amqp.Publishing{
			ContentType:  "application/json",
			DeliveryMode: amqp.Persistent,
			MessageId:    string(id),
			Type:         "job.execute",
			Timestamp:    time.Now().UTC(),
			Body:         body,
		},
	)
	if err != nil {
		return fmt.Errorf("publish job message: %w", err)
	}

	confirmed, err := confirmation.WaitContext(ctx)
	if err != nil {
		return fmt.Errorf(
			"wait for job message confirmation: %w",
			err,
		)
	}

	if !confirmed {
		return errPublishNotConfirmed
	}

	return nil
}

func (q *JobQueue) Close() error {
	return q.channel.Close()
}
