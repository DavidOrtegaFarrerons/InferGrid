package grpctransport

import (
	"testing"
	"time"

	"github.com/DavidOrtegaFarrerons/infergrid/internal/domain/job"
	inferencev1 "github.com/DavidOrtegaFarrerons/infergrid/proto/inference/v1"
)

func TestToProtoJobStatus(t *testing.T) {
	tests := []struct {
		in   job.Status
		want inferencev1.JobStatus
	}{
		{job.StatusPending, inferencev1.JobStatus_JOB_STATUS_PENDING},
		{job.StatusRunning, inferencev1.JobStatus_JOB_STATUS_RUNNING},
		{job.StatusSucceeded, inferencev1.JobStatus_JOB_STATUS_SUCCEEDED},
		{job.StatusFailed, inferencev1.JobStatus_JOB_STATUS_FAILED},
		{job.Status("WEIRD"), inferencev1.JobStatus_JOB_STATUS_UNSPECIFIED},
	}

	for _, tt := range tests {
		t.Run(string(tt.in), func(t *testing.T) {
			if got := toProtoJobStatus(tt.in); got != tt.want {
				t.Errorf("toProtoJobStatus(%s) = %s, want %s", tt.in, got, tt.want)
			}
		})
	}
}

func TestToProtoJob(t *testing.T) {
	created := time.Date(2026, 7, 16, 10, 0, 0, 0, time.UTC)
	updated := created.Add(time.Minute)

	t.Run("succeeded job carries result and no failure reason", func(t *testing.T) {
		result := "the output"
		j := job.Restore("id-1", "a prompt", job.StatusSucceeded, &result, nil, created, updated)

		got := toProtoJob(j)
		if got.Id != "id-1" || got.Prompt != "a prompt" {
			t.Errorf("unexpected id/prompt: %+v", got)
		}
		if got.Status != inferencev1.JobStatus_JOB_STATUS_SUCCEEDED {
			t.Errorf("status = %s, want SUCCEEDED", got.Status)
		}
		if got.Result == nil || *got.Result != "the output" {
			t.Errorf("Result = %v, want %q", got.Result, "the output")
		}
		if got.FailureReason != nil {
			t.Errorf("FailureReason = %v, want nil", *got.FailureReason)
		}
		if !got.CreatedAt.AsTime().Equal(created) {
			t.Errorf("CreatedAt = %v, want %v", got.CreatedAt.AsTime(), created)
		}
	})

	t.Run("failed job carries failure reason and no result", func(t *testing.T) {
		reason := "model exploded"
		j := job.Restore("id-1", "a prompt", job.StatusFailed, nil, &reason, created, updated)

		got := toProtoJob(j)
		if got.FailureReason == nil || *got.FailureReason != "model exploded" {
			t.Errorf("FailureReason = %v, want %q", got.FailureReason, "model exploded")
		}
		if got.Result != nil {
			t.Errorf("Result = %v, want nil", *got.Result)
		}
	})

	t.Run("pending job has neither result nor failure reason", func(t *testing.T) {
		j := job.Restore("id-1", "a prompt", job.StatusPending, nil, nil, created, updated)

		got := toProtoJob(j)
		if got.Result != nil || got.FailureReason != nil {
			t.Errorf("expected nil result and failure reason, got %v / %v", got.Result, got.FailureReason)
		}
	})
}
