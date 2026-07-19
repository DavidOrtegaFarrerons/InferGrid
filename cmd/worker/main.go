package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	"github.com/DavidOrtegaFarrerons/infergrid/internal/application"
	"github.com/DavidOrtegaFarrerons/infergrid/internal/config"
	"github.com/DavidOrtegaFarrerons/infergrid/internal/infrastructure/ollama"
	openaicompatible "github.com/DavidOrtegaFarrerons/infergrid/internal/infrastructure/openai-compatible"
	"github.com/DavidOrtegaFarrerons/infergrid/internal/infrastructure/postgres"
	"github.com/DavidOrtegaFarrerons/infergrid/internal/infrastructure/rabbitmq"
	"github.com/DavidOrtegaFarrerons/infergrid/internal/infrastructure/resilience"
	_ "github.com/jackc/pgx/v5/stdlib"
	amqp "github.com/rabbitmq/amqp091-go"
)

func main() {
	cfg, err := config.LoadWorker()
	if err != nil {
		log.Fatal(err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	db, err := postgres.Open(context.Background(), cfg.Database.DSN)
	if err != nil {
		log.Fatalf("Error when connecting to database: %v", err)
	}

	defer db.Close()

	mqconn, err := amqp.Dial(cfg.RabbitMQ.AMQPURL)
	if err != nil {
		log.Fatal(err)
	}
	defer mqconn.Close()

	outboxStore := postgres.NewOutboxStore(db)
	jobRepository := postgres.NewJobRepository(db, outboxStore)

	var inferenceRunner application.InferenceRunner

	switch cfg.Provider {
	case config.ProviderOllama:
		inferenceRunner = ollama.NewInferenceRunner(
			ollama.NewClient(
				http.DefaultClient,
				cfg.Ollama.BaseURL,
				cfg.Ollama.Model,
			),
		)
	case config.ProviderOpenAICompatible:
		inferenceRunner = openaicompatible.NewInferenceRunner(
			openaicompatible.NewClient(
				http.DefaultClient,
				cfg.OpenAICompatible.BaseURL,
				cfg.OpenAICompatible.Model,
				cfg.OpenAICompatible.APIKey,
			),
		)
	default:
		log.Fatalf("unknown inference provider: %s", cfg.Provider)
	}

	inferenceRunner = resilience.NewCircuitBreakerRunner(
		inferenceRunner,
		resilience.NewCircuitBreaker(
			5,
			30*time.Second,
		),
	)

	processJobService := application.NewProcessJobService(jobRepository, inferenceRunner)
	jobQueue, err := rabbitmq.NewJobConsumer(mqconn, processJobService)
	if err != nil {
		log.Fatalf("Job queue setup failed, %v", err)
	}

	// A cancelled context is the expected shutdown signal, not a failure, so
	// only a different error is fatal.
	err = jobQueue.Run(ctx)
	if err != nil && !errors.Is(err, context.Canceled) {
		log.Fatalf("job queue execution ended: %v", err)
	}
	log.Println("worker stopped")
}
