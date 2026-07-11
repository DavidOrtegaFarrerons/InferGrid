package id

import (
	"github.com/DavidOrtegaFarrerons/infergrid/internal/domain/job"
	"github.com/google/uuid"
)

type UuidJobIdGenerator struct{}

func NewUuidJobIdGenerator() UuidJobIdGenerator {
	return UuidJobIdGenerator{}
}

func (g UuidJobIdGenerator) Generate() job.ID {
	return job.ID(uuid.New().String())
}
