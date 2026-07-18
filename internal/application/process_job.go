package application

import (
	"context"
	"errors"
	"fmt"

	"github.com/DavidOrtegaFarrerons/infergrid/internal/domain/job"
)

type ProcessJobRepository interface {
	GetByID(ctx context.Context, id job.ID) (*job.Job, error)
	Update(ctx context.Context, inferenceJob *job.Job) error
}

type InferenceRunner interface {
	Generate(ctx context.Context, prompt string) (string, error)
}

type ProcessJobService struct {
	processJobRepository ProcessJobRepository
	inferenceRunner      InferenceRunner
}

func NewProcessJobService(processJobRepository ProcessJobRepository, inferenceRunner InferenceRunner) *ProcessJobService {
	return &ProcessJobService{
		processJobRepository: processJobRepository,
		inferenceRunner:      inferenceRunner,
	}
}

type ProcessJobRequest struct {
	JobID job.ID
}

func (s ProcessJobService) Execute(ctx context.Context, request ProcessJobRequest) error {
	inferenceJob, err := s.processJobRepository.GetByID(ctx, request.JobID)
	if err != nil {
		return fmt.Errorf("get job: %w", err)
	}

	if inferenceJob == nil {
		return ErrJobNotFound
	}

	switch inferenceJob.Status() {
	case job.StatusPending:
		if err = inferenceJob.Start(); err != nil {
			return fmt.Errorf("start job: %w", err)
		}

		if err = s.processJobRepository.Update(ctx, inferenceJob); err != nil {
			return fmt.Errorf("persist running job: %w", err)
		}

	case job.StatusRunning:

	case job.StatusSucceeded, job.StatusFailed:
		return nil

	default:
		return fmt.Errorf(
			"unsupported job status: %s",
			inferenceJob.Status(),
		)
	}

	result, err := s.inferenceRunner.Generate(
		ctx,
		inferenceJob.Prompt(),
	)

	if err != nil {
		if errors.Is(err, ErrInferenceUnavailable) {
			return fmt.Errorf("inference unavailable: %w", err)
		}
		if failErr := inferenceJob.Fail(err.Error()); failErr != nil {
			return fmt.Errorf("fail job: %w", failErr)
		}

		if updateErr := s.processJobRepository.Update(ctx, inferenceJob); updateErr != nil {
			return fmt.Errorf("persist failed job: %w", updateErr)
		}

		return nil
	}

	if err = inferenceJob.Succeed(result); err != nil {
		return fmt.Errorf("succeed job: %w", err)
	}

	if err = s.processJobRepository.Update(ctx, inferenceJob); err != nil {
		return fmt.Errorf("persist succeeded job: %w", err)
	}

	return nil
}
