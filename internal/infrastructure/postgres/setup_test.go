//go:build integration

package postgres

import (
	"context"
	"database/sql"
	"log"
	"os"
	"testing"
	"time"

	"github.com/DavidOrtegaFarrerons/infergrid/internal/infrastructure/postgres/migrations"
	"github.com/testcontainers/testcontainers-go"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"

	_ "github.com/jackc/pgx/v5/stdlib"
)

var testDB *sql.DB

func TestMain(m *testing.M) {
	pgContainer, err := tcpostgres.Run(
		context.Background(),
		"postgres:17-alpine",
		tcpostgres.WithDatabase("test-db"),
		tcpostgres.WithUsername("postgres"),
		tcpostgres.WithPassword("postgres"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).WithStartupTimeout(5*time.Second)),
	)

	if err != nil {
		log.Fatalf("error when creating postgres test container: %v", err)
	}

	connStr, err := pgContainer.ConnectionString(context.Background(), "sslmode=disable")
	if err != nil {
		log.Fatalf("error when getting postgres test container connection string: %v", err)
	}

	testDB, err = Open(context.Background(), connStr)
	if err != nil {
		log.Fatalf("error when connecting to db: %v", err)
	}

	err = migrations.Run(connStr, "file://migrations")
	if err != nil {
		log.Fatal(err)
	}

	code := m.Run()
	testDB.Close()
	pgContainer.Terminate(context.Background())
	os.Exit(code)
}
