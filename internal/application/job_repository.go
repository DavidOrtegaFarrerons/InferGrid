package application

import (
	"context"

	"github.com/DavidOrtegaFarrerons/infergrid/internal/domain/job"
)

type JobRepository interface {
	Create(ctx context.Context, job *job.Job) error
	GetByID(ctx context.Context, id job.ID) (*job.Job, error)
	Update(ctx context.Context, job *job.Job) error
}
