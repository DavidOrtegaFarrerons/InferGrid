package application

import (
	"context"
	"strings"

	"github.com/DavidOrtegaFarrerons/infergrid/internal/domain/job"
)

type JobReader interface {
	GetByID(ctx context.Context, id job.ID) (*job.Job, error)
}

type GetJobService struct {
	jobReader JobReader
}

func NewGetJobService(jobReader JobReader) GetJobService {
	return GetJobService{jobReader: jobReader}
}

type GetJobRequest struct {
	JobID string
}

type GetJobResponse struct {
	Job *job.Job
}

func (s *GetJobService) Execute(ctx context.Context, req GetJobRequest) (GetJobResponse, error) {
	rawID := strings.TrimSpace(req.JobID)
	if rawID == "" {
		return GetJobResponse{}, job.ErrEmptyID
	}

	inferenceJob, err := s.jobReader.GetByID(ctx, job.ID(rawID))
	if err != nil {
		return GetJobResponse{}, err
	}

	return GetJobResponse{Job: inferenceJob}, nil
}
