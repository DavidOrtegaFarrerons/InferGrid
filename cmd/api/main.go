package main

import (
	"context"
	"log"
	"net"

	"github.com/DavidOrtegaFarrerons/infergrid/internal/application"
	"github.com/DavidOrtegaFarrerons/infergrid/internal/config"
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

	cfg, err := config.LoadAPI()
	if err != nil {
		log.Fatal(err)
	}

	db, err := postgres.Open(context.Background(), cfg.Database.DSN)
	if err != nil {
		log.Fatalf("error when connecting to db: %v", err)
	}
	defer db.Close()

	log.Println("Connection to DB established")

	err = migrations.Run(cfg.Database.DSN, cfg.Database.MigrationsPath)
	if err != nil {
		log.Fatal(err)
	}

	mqconn, err := amqp.Dial(cfg.RabbitMQ.AMQPURL)
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

	grpcListener, err := net.Listen("tcp", cfg.Server.ListenAddress)
	if err != nil {
		log.Fatalf("failed to listen: %s", err)
	}

	log.Printf("Inference gRPC server running on %s \n", cfg.Server.ListenAddress)

	if err = grpcServer.Serve(grpcListener); err != nil {
		log.Fatalf("gRPC server failed: %s", err)
	}
}
