package grpctransport

import (
	"errors"
	"log"

	"github.com/DavidOrtegaFarrerons/infergrid/internal/application"
	"github.com/DavidOrtegaFarrerons/infergrid/internal/domain/job"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func mapSubmitJobError(err error) error {
	switch {
	case errors.Is(err, job.ErrEmptyPrompt):
		return status.Error(codes.InvalidArgument, err.Error())
	default:
		return internalError(err)
	}
}

func mapGetJobError(err error) error {
	switch {
	case errors.Is(err, job.ErrEmptyID):
		return status.Error(codes.InvalidArgument, err.Error())
	case errors.Is(err, application.ErrJobNotFound):
		return status.Error(codes.NotFound, "job not found")
	default:
		return internalError(err)
	}
}

func internalError(err error) error {
	log.Printf("gRPC internal error: %v", err)

	return status.Error(codes.Internal, "internal server error")
}
