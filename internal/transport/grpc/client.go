package grpctransport

import (
	"context"

	"github.com/DavidOrtegaFarrerons/infergrid/internal/application"
	inferencev1 "github.com/DavidOrtegaFarrerons/infergrid/proto/inference/v1"
	"google.golang.org/grpc"
)

type GRPCInferenceServiceClient struct {
	cc             inferencev1.InferenceServiceClient
	jobIdGenerator application.JobIDGenerator
}

func NewGRPCInferenceServiceClient(conn grpc.ClientConnInterface, jobIdGenerator application.JobIDGenerator) *GRPCInferenceServiceClient {
	return &GRPCInferenceServiceClient{
		cc:             inferencev1.NewInferenceServiceClient(conn),
		jobIdGenerator: jobIdGenerator,
	}
}

func (c *GRPCInferenceServiceClient) SubmitJob(
	ctx context.Context,
	in *inferencev1.SubmitJobRequest,
	opts ...grpc.CallOption,
) (*inferencev1.SubmitJobResponse, error) {
	return c.cc.SubmitJob(ctx, in, opts...)
}

func (c *GRPCInferenceServiceClient) GetJob(
	ctx context.Context,
	in *inferencev1.GetJobRequest,
	opts ...grpc.CallOption,
) (*inferencev1.GetJobResponse, error) {
	return c.cc.GetJob(ctx, in, opts...)
}
