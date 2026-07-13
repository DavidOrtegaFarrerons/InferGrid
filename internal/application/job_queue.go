package application

import (
	"context"

	"github.com/DavidOrtegaFarrerons/infergrid/internal/domain/job"
)

type JobQueue interface {
	Enqueue(ctx context.Context, id job.ID) error
}
