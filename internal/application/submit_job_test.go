package application

import (
	"context"
	"errors"
	"testing"

	"github.com/DavidOrtegaFarrerons/infergrid/internal/domain/job"
)

type fakeIDGenerator struct {
	id job.ID
}

func (f fakeIDGenerator) Generate() job.ID { return f.id }

type fakeJobRepository struct {
	created   *job.Job
	createErr error
}

func (f *fakeJobRepository) Create(ctx context.Context, j *job.Job) error {
	f.created = j
	return f.createErr
}
func (f *fakeJobRepository) GetByID(ctx context.Context, id job.ID) (*job.Job, error) {
	return nil, nil
}
func (f *fakeJobRepository) Update(ctx context.Context, j *job.Job) error { return nil }

type fakeJobQueue struct {
	enqueuedID job.ID
	enqueueErr error
	called     bool
}

func (f *fakeJobQueue) Enqueue(ctx context.Context, id job.ID) error {
	f.called = true
	f.enqueuedID = id
	return f.enqueueErr
}

func TestSubmitJobService_Execute(t *testing.T) {
	t.Run("invalid prompt returns error before persisting or enqueuing", func(t *testing.T) {
		gen := fakeIDGenerator{id: "generated-id"}
		repo := &fakeJobRepository{}
		queue := &fakeJobQueue{}
		svc := NewSubmitJobService(gen, repo, queue)

		_, err := svc.Execute(context.Background(), SubmitJobRequest{Prompt: "   "})
		if !errors.Is(err, job.ErrEmptyPrompt) {
			t.Fatalf("error = %v, want %v", err, job.ErrEmptyPrompt)
		}
		if repo.created != nil {
			t.Errorf("repository.Create should not be called on invalid input")
		}
		if queue.called {
			t.Errorf("queue.Enqueue should not be called on invalid input")
		}
	})

	t.Run("happy path creates, enqueues, and returns the generated id", func(t *testing.T) {
		gen := fakeIDGenerator{id: "generated-id"}
		repo := &fakeJobRepository{}
		queue := &fakeJobQueue{}
		svc := NewSubmitJobService(gen, repo, queue)

		resp, err := svc.Execute(context.Background(), SubmitJobRequest{Prompt: "a real prompt"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if resp.ID != job.ID("generated-id") {
			t.Errorf("response id = %q, want %q", resp.ID, "generated-id")
		}
		if repo.created == nil || repo.created.ID() != job.ID("generated-id") {
			t.Errorf("repository received wrong job: %v", repo.created)
		}
		if repo.created != nil && repo.created.Prompt() != "a real prompt" {
			t.Errorf("persisted prompt = %q, want %q", repo.created.Prompt(), "a real prompt")
		}
		if queue.enqueuedID != job.ID("generated-id") {
			t.Errorf("enqueued id = %q, want %q", queue.enqueuedID, "generated-id")
		}
	})

	t.Run("repository error is returned and the job is not enqueued", func(t *testing.T) {
		gen := fakeIDGenerator{id: "generated-id"}
		repo := &fakeJobRepository{createErr: errors.New("db down")}
		queue := &fakeJobQueue{}
		svc := NewSubmitJobService(gen, repo, queue)

		_, err := svc.Execute(context.Background(), SubmitJobRequest{Prompt: "a real prompt"})
		if err == nil {
			t.Fatal("expected an error when the repository fails")
		}
		if queue.called {
			t.Errorf("queue.Enqueue should not be called after a repository failure")
		}
	})

	t.Run("queue error is returned", func(t *testing.T) {
		gen := fakeIDGenerator{id: "generated-id"}
		repo := &fakeJobRepository{}
		queue := &fakeJobQueue{enqueueErr: errors.New("broker down")}
		svc := NewSubmitJobService(gen, repo, queue)

		_, err := svc.Execute(context.Background(), SubmitJobRequest{Prompt: "a real prompt"})
		if err == nil {
			t.Fatal("expected an error when the queue fails")
		}
	})
}
