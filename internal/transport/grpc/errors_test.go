package grpctransport

import (
	"errors"
	"testing"

	"github.com/DavidOrtegaFarrerons/infergrid/internal/application"
	"github.com/DavidOrtegaFarrerons/infergrid/internal/domain/job"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestMapSubmitJobError(t *testing.T) {
	tests := []struct {
		name string
		in   error
		want codes.Code
	}{
		{"empty prompt maps to InvalidArgument", job.ErrEmptyPrompt, codes.InvalidArgument},
		{"unknown error maps to Internal", errors.New("boom"), codes.Internal},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			st, ok := status.FromError(mapSubmitJobError(tt.in))
			if !ok {
				t.Fatalf("expected a gRPC status error")
			}
			if st.Code() != tt.want {
				t.Errorf("code = %s, want %s", st.Code(), tt.want)
			}
		})
	}
}

func TestMapGetJobError(t *testing.T) {
	tests := []struct {
		name string
		in   error
		want codes.Code
	}{
		{"empty id maps to InvalidArgument", job.ErrEmptyID, codes.InvalidArgument},
		{"not found maps to NotFound", application.ErrJobNotFound, codes.NotFound},
		{"unknown error maps to Internal", errors.New("boom"), codes.Internal},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			st, ok := status.FromError(mapGetJobError(tt.in))
			if !ok {
				t.Fatalf("expected a gRPC status error")
			}
			if st.Code() != tt.want {
				t.Errorf("code = %s, want %s", st.Code(), tt.want)
			}
		})
	}
}
