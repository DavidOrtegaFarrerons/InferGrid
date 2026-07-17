package application

import (
	"context"
	"errors"
	"testing"

	"github.com/DavidOrtegaFarrerons/infergrid/internal/domain/job"
)

type fakeJobReader struct {
	job    *job.Job
	err    error
	gotID  job.ID
	called bool
}

func (f *fakeJobReader) GetByID(ctx context.Context, id job.ID) (*job.Job, error) {
	f.called = true
	f.gotID = id
	return f.job, f.err
}

func TestGetJobService_Execute(t *testing.T) {
	existing, err := job.New("real-id", "a prompt")
	if err != nil {
		t.Fatalf("building test job: %v", err)
	}

	t.Run("empty id returns ErrEmptyID without hitting the reader", func(t *testing.T) {
		reader := &fakeJobReader{}
		svc := NewGetJobService(reader)

		_, err := svc.Execute(context.Background(), GetJobRequest{JobID: "   "})
		if !errors.Is(err, job.ErrEmptyID) {
			t.Fatalf("error = %v, want %v", err, job.ErrEmptyID)
		}
		if reader.called {
			t.Errorf("reader should not be called on invalid input")
		}
	})

	t.Run("reader error is propagated", func(t *testing.T) {
		wantErr := errors.New("db down")
		reader := &fakeJobReader{err: wantErr}
		svc := NewGetJobService(reader)

		_, err := svc.Execute(context.Background(), GetJobRequest{JobID: "real-id"})
		if !errors.Is(err, wantErr) {
			t.Fatalf("error = %v, want %v", err, wantErr)
		}
	})

	t.Run("happy path trims id and returns the job", func(t *testing.T) {
		reader := &fakeJobReader{job: existing}
		svc := NewGetJobService(reader)

		resp, err := svc.Execute(context.Background(), GetJobRequest{JobID: "  real-id  "})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if reader.gotID != job.ID("real-id") {
			t.Errorf("reader got id %q, want trimmed %q", reader.gotID, "real-id")
		}
		if resp.Job != existing {
			t.Errorf("returned job = %v, want %v", resp.Job, existing)
		}
	})
}
