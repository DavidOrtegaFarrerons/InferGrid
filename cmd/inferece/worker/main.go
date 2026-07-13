package main

import (
	"context"
	"log"

	"github.com/DavidOrtegaFarrerons/infergrid/internal/infrastructure/postgres"
)

func main() {
	databaseDSN := "postgres://infergrid:infergrid@localhost:5432/infergrid?sslmode=disable"
	db, err := postgres.Open(context.Background(), databaseDSN)
	if err != nil {
		log.Fatalf("Error when connecting to database: %v", err)
	}

	defer db.Close()

	jobRepository := postgres.NewJobRepository(db)
	
}
