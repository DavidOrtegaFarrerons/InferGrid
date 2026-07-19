package rabbitmq

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

var errPublishNotConfirmed = errors.New(
	"job message was not confirmed by RabbitMQ",
)

type JobPublisher struct {
	channel *amqp.Channel
}

func NewJobPublisher(
	connection *amqp.Connection,
) (*JobPublisher, error) {
	channel, err := openJobChannel(connection)
	if err != nil {
		return nil, err
	}

	if err = channel.Confirm(false); err != nil {
		channel.Close()
		return nil, fmt.Errorf("enable publisher confirms: %w", err)
	}

	return &JobPublisher{channel: channel}, nil
}

func (q *JobPublisher) Publish(
	ctx context.Context,
	messageID string,
	payload json.RawMessage,
) error {
	confirmation, err := q.channel.PublishWithDeferredConfirmWithContext(
		ctx,
		jobExchange,
		jobRoutingKey,
		false,
		false,
		amqp.Publishing{
			ContentType:  "application/json",
			DeliveryMode: amqp.Persistent,
			MessageId:    messageID,
			Type:         "job.execute",
			Timestamp:    time.Now().UTC(),
			Body:         payload,
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

func (q *JobPublisher) Close() error {
	return q.channel.Close()
}
