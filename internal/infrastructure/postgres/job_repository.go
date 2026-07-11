package postgres

import (
	"context"
	"database/sql"
	"errors"
	"log"
	"time"

	"github.com/DavidOrtegaFarrerons/infergrid/internal/application"
	"github.com/DavidOrtegaFarrerons/infergrid/internal/domain/job"
)

type jobRow struct {
	ID            string
	Prompt        string
	Status        string
	Result        sql.NullString
	FailureReason sql.NullString
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

func (r jobRow) toDomain() *job.Job {
	var result *string
	if r.Result.Valid {
		result = &r.Result.String
	}

	var failureReason *string
	if r.FailureReason.Valid {
		failureReason = &r.FailureReason.String
	}

	return job.Restore(
		job.ID(r.ID),
		r.Prompt,
		job.Status(r.Status),
		result,
		failureReason,
		r.CreatedAt,
		r.UpdatedAt,
	)
}

type JobRepository struct {
	db *sql.DB
}

func (r JobRepository) GetByID(ctx context.Context, id job.ID) (*job.Job, error) {
	query := `SELECT id, prompt, status, result, failure_reason, created_at, updated_at
			 FROM jobs WHERE id = $1`

	var inferenceJob jobRow
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&inferenceJob.ID,
		&inferenceJob.Prompt,
		&inferenceJob.Status,
		&inferenceJob.Result,
		&inferenceJob.FailureReason,
		&inferenceJob.CreatedAt,
		&inferenceJob.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, application.ErrJobNotFound
		}

		return nil, err
	}

	return inferenceJob.toDomain(), nil
}

func (r JobRepository) Update(ctx context.Context, job *job.Job) error {
	panic("implement me")
}

func NewJobRepository(db *sql.DB) JobRepository {
	return JobRepository{db: db}
}

func (r JobRepository) Create(
	ctx context.Context,
	inferenceJob *job.Job,
) error {
	log.Printf("Create called for job %s", inferenceJob.ID())

	query := `
		INSERT INTO jobs (
			id,
			prompt,
			status,
			created_at,
			updated_at
		)
		VALUES ($1, $2, $3, $4, $5)
	`

	_, err := r.db.ExecContext(
		ctx,
		query,
		string(inferenceJob.ID()),
		inferenceJob.Prompt(),
		string(inferenceJob.Status()),
		inferenceJob.CreatedAt(),
		inferenceJob.UpdatedAt(),
	)
	if err != nil {
		log.Printf("Create failed for job %s: %v", inferenceJob.ID(), err)
		return err
	}

	return nil
}
