package application

import (
	"context"
	"log"

	"github.com/DavidOrtegaFarrerons/infergrid/internal/domain/job"
)

type SubmitJobService struct {
	jobIDGenerator JobIDGenerator
	jobRepository  JobRepository
}

func NewSubmitJobService(
	jobIDGenerator JobIDGenerator,
	jobRepository JobRepository,
) SubmitJobService {
	return SubmitJobService{
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
		log.Printf("failed to submit job: %v \n", err)
	}

	return SubmitJobResponse{
		ID: inferenceJob.ID(),
	}, nil
}
