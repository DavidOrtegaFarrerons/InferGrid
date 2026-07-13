package rabbitmq

import (
	"fmt"

	amqp "github.com/rabbitmq/amqp091-go"
)

const (
	jobExchange   = "infergrid.jobs"
	jobQueue      = "infergrid.jobs.execute"
	jobRoutingKey = "jobs.execute"
)

func openJobChannel(
	connection *amqp.Connection,
) (*amqp.Channel, error) {
	channel, err := connection.Channel()
	if err != nil {
		return nil, fmt.Errorf("open RabbitMQ channel: %w", err)
	}

	if err = declareJobTopology(channel); err != nil {
		channel.Close()
		return nil, err
	}

	return channel, nil
}

func declareJobTopology(channel *amqp.Channel) error {
	if err := channel.ExchangeDeclare(
		jobExchange,
		"direct",
		true,
		false,
		false,
		false,
		nil,
	); err != nil {
		return fmt.Errorf("declare job exchange: %w", err)
	}

	if _, err := channel.QueueDeclare(
		jobQueue,
		true,
		false,
		false,
		false,
		nil,
	); err != nil {
		return fmt.Errorf("declare job queue: %w", err)
	}

	if err := channel.QueueBind(
		jobQueue,
		jobRoutingKey,
		jobExchange,
		false,
		nil,
	); err != nil {
		return fmt.Errorf("bind job queue: %w", err)
	}

	return nil
}
