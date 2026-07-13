package main

import (
	"context"
	"log"
	"net/http"

	"github.com/DavidOrtegaFarrerons/infergrid/internal/application"
	"github.com/DavidOrtegaFarrerons/infergrid/internal/infrastructure/ollama"
	"github.com/DavidOrtegaFarrerons/infergrid/internal/infrastructure/postgres"
	"github.com/DavidOrtegaFarrerons/infergrid/internal/infrastructure/rabbitmq"
	_ "github.com/jackc/pgx/v5/stdlib"
	amqp "github.com/rabbitmq/amqp091-go"
)

func main() {
	databaseDSN := "postgres://infergrid:infergrid@localhost:5432/infergrid?sslmode=disable"
	db, err := postgres.Open(context.Background(), databaseDSN)
	if err != nil {
		log.Fatalf("Error when connecting to database: %v", err)
	}

	defer db.Close()

	mqconn, err := amqp.Dial("amqp://infergrid:infergrid@localhost:5672/")
	if err != nil {
		log.Fatal(err)
	}

	jobRepository := postgres.NewJobRepository(db)

	inferenceRunner := ollama.NewInferenceRunner(
		ollama.NewClient(
			http.DefaultClient,
			"http://localhost:11434",
			"llama3.2:1b",
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
