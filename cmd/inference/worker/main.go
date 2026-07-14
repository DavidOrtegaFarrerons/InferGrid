package main

import (
	"context"
	"log"
	"net/http"

	"github.com/DavidOrtegaFarrerons/infergrid/internal/application"
	"github.com/DavidOrtegaFarrerons/infergrid/internal/config"
	"github.com/DavidOrtegaFarrerons/infergrid/internal/infrastructure/ollama"
	"github.com/DavidOrtegaFarrerons/infergrid/internal/infrastructure/postgres"
	"github.com/DavidOrtegaFarrerons/infergrid/internal/infrastructure/rabbitmq"
	_ "github.com/jackc/pgx/v5/stdlib"
	amqp "github.com/rabbitmq/amqp091-go"
)

func main() {
	cfg, err := config.LoadWorker()
	if err != nil {
		log.Fatal(err)
	}

	db, err := postgres.Open(context.Background(), cfg.Database.DSN)
	if err != nil {
		log.Fatalf("Error when connecting to database: %v", err)
	}

	defer db.Close()

	mqconn, err := amqp.Dial(cfg.RabbitMQ.AMQPURL)
	if err != nil {
		log.Fatal(err)
	}

	jobRepository := postgres.NewJobRepository(db)

	inferenceRunner := ollama.NewInferenceRunner(
		ollama.NewClient(
			http.DefaultClient,
			cfg.Ollama.BaseURL,
			cfg.Ollama.Model,
		),
	)
	processJobService := application.NewProcessJobService(jobRepository, inferenceRunner)
	jobQueue, err := rabbitmq.NewJobConsumer(mqconn, processJobService)
	if err != nil {
		log.Fatalf("Job queue setup failed, %v", err)
	}

	err = jobQueue.Run(context.Background())
	if err != nil {
		log.Fatalf("job queue execution ended: %v", err)
	}
}
