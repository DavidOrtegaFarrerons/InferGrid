package grpctransport

import (
	"context"

	"github.com/DavidOrtegaFarrerons/infergrid/internal/application"
	"github.com/DavidOrtegaFarrerons/infergrid/internal/domain/job"
	"google.golang.org/protobuf/types/known/timestamppb"
)
import inferencev1 "github.com/DavidOrtegaFarrerons/infergrid/proto/inference/v1"

func toProtoJob(inferenceJob *job.Job) *inferencev1.Job {
	result, hasResult := inferenceJob.Result()
	failureReason, hasFailureReason := inferenceJob.FailureReason()

	var protoResult *string
	if hasResult {
		protoResult = &result
	}

	var protoFailureReason *string
	if hasFailureReason {
		protoFailureReason = &failureReason
	}

	return &inferencev1.Job{
		Id:            string(inferenceJob.ID()),
		Prompt:        inferenceJob.Prompt(),
		Status:        toProtoJobStatus(inferenceJob.Status()),
		Result:        protoResult,
		FailureReason: protoFailureReason,
		CreatedAt:     timestamppb.New(inferenceJob.CreatedAt()),
		UpdatedAt:     timestamppb.New(inferenceJob.UpdatedAt()),
	}
}

func toProtoJobStatus(status job.Status) inferencev1.JobStatus {
	switch status {
	case job.StatusPending:
		return inferencev1.JobStatus_JOB_STATUS_PENDING
	case job.StatusRunning:
		return inferencev1.JobStatus_JOB_STATUS_RUNNING
	case job.StatusSucceeded:
		return inferencev1.JobStatus_JOB_STATUS_SUCCEEDED
	case job.StatusFailed:
		return inferencev1.JobStatus_JOB_STATUS_FAILED
	default:
		return inferencev1.JobStatus_JOB_STATUS_UNSPECIFIED
	}
}

type GRPCInferenceServiceServer struct {
	inferencev1.UnimplementedInferenceServiceServer
	submitJobService *application.SubmitJobService
	getJobService    *application.GetJobService
}

func NewGRPCInferenceServiceServer(
	submitJobService *application.SubmitJobService,
	getJobService *application.GetJobService,
) *GRPCInferenceServiceServer {
	return &GRPCInferenceServiceServer{
		submitJobService: submitJobService,
		getJobService:    getJobService,
	}
}

func (s *GRPCInferenceServiceServer) SubmitJob(
	ctx context.Context,
	req *inferencev1.SubmitJobRequest,
) (*inferencev1.SubmitJobResponse, error) {
	resp, err := s.submitJobService.Execute(
		ctx,
		application.SubmitJobRequest{Prompt: req.Prompt},
	)
	if err != nil {
		return nil, mapSubmitJobError(err)
	}

	return &inferencev1.SubmitJobResponse{
		JobId: string(resp.ID),
	}, nil
}

func (s *GRPCInferenceServiceServer) GetJob(
	ctx context.Context,
	req *inferencev1.GetJobRequest,
) (*inferencev1.GetJobResponse, error) {
	resp, err := s.getJobService.Execute(
		ctx,
		application.GetJobRequest{JobID: req.JobId},
	)
	if err != nil {
		return nil, mapGetJobError(err)
	}

	return &inferencev1.GetJobResponse{
		Job: toProtoJob(resp.Job),
	}, nil
}
