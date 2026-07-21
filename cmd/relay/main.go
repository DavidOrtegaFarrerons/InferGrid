package main

import (
	"context"
	"log"
	"os/signal"
	"syscall"

	"github.com/DavidOrtegaFarrerons/infergrid/internal/config"
	"github.com/DavidOrtegaFarrerons/infergrid/internal/infrastructure/postgres"
	"github.com/DavidOrtegaFarrerons/infergrid/internal/infrastructure/rabbitmq"
	"github.com/DavidOrtegaFarrerons/infergrid/internal/infrastructure/relay"
	"github.com/DavidOrtegaFarrerons/infergrid/internal/observability"
	amqp "github.com/rabbitmq/amqp091-go"

	_ "github.com/jackc/pgx/v5/stdlib"
)

func main() {
	cfg, err := config.LoadAPI()
	if err != nil {
		log.Fatal(err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	db, err := postgres.Open(context.Background(), cfg.Database.DSN)
	if err != nil {
		log.Fatalf("error when connecting to db: %v", err)
	}
	defer db.Close()

	log.Println("Connection to DB established")

	mqconn, err := amqp.Dial(cfg.RabbitMQ.AMQPURL)
	if err != nil {
		log.Fatal(err)
	}

	defer mqconn.Close()

	jobPublisher, err := rabbitmq.NewJobPublisher(mqconn)
	if err != nil {
		log.Fatalf("Job queue setup failed, %v", err)
	}

	outboxStore := postgres.NewOutboxStore(db)
	logger := observability.NewLogger("relay", cfg.Logger.LogLevel)
	newRelay := relay.NewRelay(outboxStore, jobPublisher, logger)

	log.Println("Relay started")
	if err := newRelay.Run(ctx); err != nil {
		log.Fatal(err)
	}
	log.Println("Relay stopped")
}
