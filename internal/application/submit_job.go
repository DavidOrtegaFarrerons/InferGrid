package application

import (
	"context"
	"fmt"

	"github.com/DavidOrtegaFarrerons/infergrid/internal/domain/job"
)

type SubmitJobService struct {
	jobIDGenerator JobIDGenerator
	jobRepository  JobRepository
}

func NewSubmitJobService(
	jobIDGenerator JobIDGenerator,
	jobRepository JobRepository,
) *SubmitJobService {
	return &SubmitJobService{
		jobIDGenerator: jobIDGenerator,
		jobRepository:  jobRepository,
	}
}

type SubmitJobRequest struct {
	Prompt string
}

type SubmitJobResponse struct {
	ID job.ID
}

func (s *SubmitJobService) Execute(
	ctx context.Context,
	request SubmitJobRequest,
) (SubmitJobResponse, error) {
	inferenceJob, err := job.New(
		s.jobIDGenerator.Generate(),
		request.Prompt,
	)
	if err != nil {
		return SubmitJobResponse{}, err
	}

	if err = s.jobRepository.Create(ctx, inferenceJob); err != nil {
		return SubmitJobResponse{}, fmt.Errorf("failed to submit job: %w", err)
	}

	return SubmitJobResponse{
		ID: inferenceJob.ID(),
	}, nil
}
