package main

import (
	"context"
	"log"
	"net"

	"github.com/DavidOrtegaFarrerons/infergrid/internal/application"
	"github.com/DavidOrtegaFarrerons/infergrid/internal/infrastructure/id"
	"github.com/DavidOrtegaFarrerons/infergrid/internal/infrastructure/postgres"
	"github.com/DavidOrtegaFarrerons/infergrid/internal/infrastructure/postgres/migrations"
	"github.com/DavidOrtegaFarrerons/infergrid/internal/infrastructure/rabbitmq"
	grpctransport "github.com/DavidOrtegaFarrerons/infergrid/internal/transport/grpc"
	inferencev1 "github.com/DavidOrtegaFarrerons/infergrid/proto/inference/v1"
	amqp "github.com/rabbitmq/amqp091-go"

	_ "github.com/jackc/pgx/v5/stdlib"
	"google.golang.org/grpc"
)

func main() {

	databaseDSN := "postgres://infergrid:infergrid@localhost:5432/infergrid?sslmode=disable"
	db, err := postgres.Open(context.Background(), databaseDSN)
	if err != nil {
		log.Fatalf("error when connecting to db: %v", err)
	}
	defer db.Close()

	log.Println("Connection to DB established")

	migrationsPath := "file://internal/infrastructure/postgres/migrations"
	err = migrations.Run(databaseDSN, migrationsPath)
	if err != nil {
		log.Fatal(err)
	}

	mqconn, err := amqp.Dial("amqp://infergrid:infergrid@localhost:5672/")
	if err != nil {
		log.Fatal(err)
	}

	jobQueue, err := rabbitmq.NewJobQueue(mqconn)
	if err != nil {
		log.Fatalf("Job queue setup failed, %v", err)
	}

	jobIdGenerator := id.NewUuidJobIdGenerator()
	jobRepository := postgres.NewJobRepository(db)
	submitJobService := application.NewSubmitJobService(
		jobIdGenerator,
		jobRepository,
		jobQueue,
	)
	getJobService := application.NewGetJobService(jobRepository)
	inferenceGRPCServer := grpctransport.NewGRPCInferenceServiceServer(submitJobService, getJobService)
	grpcServer := grpc.NewServer()
	inferencev1.RegisterInferenceServiceServer(grpcServer, inferenceGRPCServer)

	grpcListener, err := net.Listen("tcp", ":9091")
	if err != nil {
		log.Fatalf("failed to listen: %s", err)
	}

	log.Println("Inference gRPC server running on :9091")

	if err = grpcServer.Serve(grpcListener); err != nil {
		log.Fatalf("gRPC server failed: %s", err)
	}
}
