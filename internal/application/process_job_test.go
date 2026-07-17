package application

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/DavidOrtegaFarrerons/infergrid/internal/domain/job"
)

type fakeProcessRepo struct {
	getJob    *job.Job
	getErr    error
	updates   []*job.Job
	updateErr error
}

func (f *fakeProcessRepo) GetByID(ctx context.Context, id job.ID) (*job.Job, error) {
	return f.getJob, f.getErr
}

func (f *fakeProcessRepo) Update(ctx context.Context, j *job.Job) error {
	f.updates = append(f.updates, j)
	return f.updateErr
}

type fakeRunner struct {
	result    string
	err       error
	gotPrompt string
	called    bool
}

func (f *fakeRunner) Generate(ctx context.Context, prompt string) (string, error) {
	f.called = true
	f.gotPrompt = prompt
	return f.result, f.err
}

func restoreWithStatus(status job.Status) *job.Job {
	past := time.Now().UTC().Add(-time.Hour)
	return job.Restore("id-1", "my prompt", status, nil, nil, past, past)
}

func TestProcessJobService_Execute(t *testing.T) {
	t.Run("repository lookup error is wrapped", func(t *testing.T) {
		repo := &fakeProcessRepo{getErr: errors.New("db down")}
		runner := &fakeRunner{}
		svc := NewProcessJobService(repo, runner)

		err := svc.Execute(context.Background(), ProcessJobRequest{JobID: "id-1"})
		if err == nil || !strings.Contains(err.Error(), "get job") {
			t.Fatalf("error = %v, want it to wrap \"get job\"", err)
		}
	})

	t.Run("missing job returns ErrJobNotFound", func(t *testing.T) {
		repo := &fakeProcessRepo{getJob: nil}
		runner := &fakeRunner{}
		svc := NewProcessJobService(repo, runner)

		err := svc.Execute(context.Background(), ProcessJobRequest{JobID: "id-1"})
		if !errors.Is(err, ErrJobNotFound) {
			t.Fatalf("error = %v, want %v", err, ErrJobNotFound)
		}
	})

	t.Run("pending job is started, run, and marked succeeded", func(t *testing.T) {
		pending, err := job.New("id-1", "my prompt")
		if err != nil {
			t.Fatalf("building job: %v", err)
		}
		repo := &fakeProcessRepo{getJob: pending}
		runner := &fakeRunner{result: "the output"}
		svc := NewProcessJobService(repo, runner)

		if err := svc.Execute(context.Background(), ProcessJobRequest{JobID: "id-1"}); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if runner.gotPrompt != "my prompt" {
			t.Errorf("runner prompt = %q, want %q", runner.gotPrompt, "my prompt")
		}
		if pending.Status() != job.StatusSucceeded {
			t.Errorf("status = %s, want %s", pending.Status(), job.StatusSucceeded)
		}
		if got, ok := pending.Result(); !ok || got != "the output" {
			t.Errorf("result = %q, %v; want %q, true", got, ok, "the output")
		}
		// One persist for RUNNING, one for SUCCEEDED.
		if len(repo.updates) != 2 {
			t.Errorf("update count = %d, want 2", len(repo.updates))
		}
	})

	t.Run("inference failure marks the job failed and returns nil", func(t *testing.T) {
		pending, err := job.New("id-1", "my prompt")
		if err != nil {
			t.Fatalf("building job: %v", err)
		}
		repo := &fakeProcessRepo{getJob: pending}
		runner := &fakeRunner{err: errors.New("model exploded")}
		svc := NewProcessJobService(repo, runner)

		if err := svc.Execute(context.Background(), ProcessJobRequest{JobID: "id-1"}); err != nil {
			t.Fatalf("expected nil error on inference failure, got %v", err)
		}
		if pending.Status() != job.StatusFailed {
			t.Errorf("status = %s, want %s", pending.Status(), job.StatusFailed)
		}
		if reason, ok := pending.FailureReason(); !ok || reason != "model exploded" {
			t.Errorf("failure reason = %q, %v; want %q, true", reason, ok, "model exploded")
		}
	})

	t.Run("already running job runs without an extra start persist", func(t *testing.T) {
		running := restoreWithStatus(job.StatusRunning)
		repo := &fakeProcessRepo{getJob: running}
		runner := &fakeRunner{result: "the output"}
		svc := NewProcessJobService(repo, runner)

		if err := svc.Execute(context.Background(), ProcessJobRequest{JobID: "id-1"}); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if running.Status() != job.StatusSucceeded {
			t.Errorf("status = %s, want %s", running.Status(), job.StatusSucceeded)
		}
		if len(repo.updates) != 1 {
			t.Errorf("update count = %d, want 1 (only the succeeded persist)", len(repo.updates))
		}
	})

	for _, terminal := range []job.Status{job.StatusSucceeded, job.StatusFailed} {
		t.Run("terminal job "+string(terminal)+" is a no-op", func(t *testing.T) {
			done := restoreWithStatus(terminal)
			repo := &fakeProcessRepo{getJob: done}
			runner := &fakeRunner{}
			svc := NewProcessJobService(repo, runner)

			if err := svc.Execute(context.Background(), ProcessJobRequest{JobID: "id-1"}); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if runner.called {
				t.Errorf("runner should not be called for a %s job", terminal)
			}
			if len(repo.updates) != 0 {
				t.Errorf("update count = %d, want 0", len(repo.updates))
			}
		})
	}

	t.Run("unsupported status returns an error", func(t *testing.T) {
		weird := restoreWithStatus(job.Status("WEIRD"))
		repo := &fakeProcessRepo{getJob: weird}
		runner := &fakeRunner{}
		svc := NewProcessJobService(repo, runner)

		err := svc.Execute(context.Background(), ProcessJobRequest{JobID: "id-1"})
		if err == nil || !strings.Contains(err.Error(), "unsupported job status") {
			t.Fatalf("error = %v, want it to mention \"unsupported job status\"", err)
		}
		if runner.called {
			t.Errorf("runner should not be called for an unsupported status")
		}
	})
}
