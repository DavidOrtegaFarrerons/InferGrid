package application

import (
	"github.com/DavidOrtegaFarrerons/infergrid/internal/domain/job"
)

type JobIDGenerator interface {
	Generate() job.ID
}
