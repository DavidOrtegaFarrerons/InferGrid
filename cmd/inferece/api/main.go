package main

import (
	"context"
	"database/sql"
	"log"
	"net"
	"os"
	"time"

	"github.com/DavidOrtegaFarrerons/infergrid/internal/application"
	"github.com/DavidOrtegaFarrerons/infergrid/internal/infrastructure/id"
	"github.com/DavidOrtegaFarrerons/infergrid/internal/infrastructure/postgres"
	"github.com/DavidOrtegaFarrerons/infergrid/internal/infrastructure/postgres/migrations"
	grpctransport "github.com/DavidOrtegaFarrerons/infergrid/internal/transport/grpc"
	inferencev1 "github.com/DavidOrtegaFarrerons/infergrid/proto/inference/v1"

	_ "github.com/jackc/pgx/v5/stdlib"
	"google.golang.org/grpc"
)

func main() {

	databaseDSN := "postgres://infergrid:infergrid@localhost:5432/infergrid?sslmode=disable"

	migrationsPath := os.Getenv("BILLING_MIGRATIONS_PATH")
	if migrationsPath == "" {
		migrationsPath = "file://internal/infrastructure/postgres/migrations"
	}

	err := migrations.Run(databaseDSN, migrationsPath)
	if err != nil {
		log.Fatal(err)
	}

	db, err := sql.Open("pgx", databaseDSN)
	if err != nil {
		log.Fatal(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	err = db.PingContext(ctx)
	if err != nil {
		log.Fatal(err)
	}

	defer db.Close()
	log.Println("Connection to DB established")

	jobIdGenerator := id.NewUuidJobIdGenerator()
	jobRepository := postgres.NewJobRepository(db)
	submitJobService := application.NewSubmitJobService(jobIdGenerator, jobRepository)
	getJobService := application.NewGetJobService(jobRepository)
	inferenceGRPCServer := grpctransport.NewGRPCInferenceServiceServer(submitJobService, getJobService)
	grpcServer := grpc.NewServer()
	inferencev1.RegisterInferenceServiceServer(grpcServer, inferenceGRPCServer)

	grpcListener, err := net.Listen("tcp", ":9091")
	if err != nil {
		log.Fatalf("failed to listen: %s", err)
	}

	log.Println("Inference gRPC server running on :9091")

	if err := grpcServer.Serve(grpcListener); err != nil {
		log.Fatalf("gRPC server failed: %s", err)
	}
}
