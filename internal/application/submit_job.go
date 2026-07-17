package application

import (
	"context"
	"fmt"

	"github.com/DavidOrtegaFarrerons/infergrid/internal/domain/job"
)

type SubmitJobService struct {
	jobIDGenerator JobIDGenerator
	jobRepository  JobRepository
	jobQueue       JobQueue
}

func NewSubmitJobService(
	jobIDGenerator JobIDGenerator,
	jobRepository JobRepository,
	jobQueue JobQueue,
) *SubmitJobService {
	return &SubmitJobService{
		jobIDGenerator: jobIDGenerator,
		jobRepository:  jobRepository,
		jobQueue:       jobQueue,
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

	if err = s.jobQueue.Enqueue(ctx, inferenceJob.ID()); err != nil {
		return SubmitJobResponse{}, fmt.Errorf("failed to enqueue job: %w", err)
	}

	return SubmitJobResponse{
		ID: inferenceJob.ID(),
	}, nil
}
