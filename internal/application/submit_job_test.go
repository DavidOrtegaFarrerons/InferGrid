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

func TestSubmitJobService_Execute(t *testing.T) {
	t.Run("invalid prompt returns error before persisting", func(t *testing.T) {
		gen := fakeIDGenerator{id: "generated-id"}
		repo := &fakeJobRepository{}
		svc := NewSubmitJobService(gen, repo)

		_, err := svc.Execute(context.Background(), SubmitJobRequest{Prompt: "   "})
		if !errors.Is(err, job.ErrEmptyPrompt) {
			t.Fatalf("error = %v, want %v", err, job.ErrEmptyPrompt)
		}
		if repo.created != nil {
			t.Errorf("repository.Create should not be called on invalid input")
		}
	})

	t.Run("happy path creates and returns the generated id", func(t *testing.T) {
		gen := fakeIDGenerator{id: "generated-id"}
		repo := &fakeJobRepository{}
		svc := NewSubmitJobService(gen, repo)

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
	})

	t.Run("repository error is returned", func(t *testing.T) {
		gen := fakeIDGenerator{id: "generated-id"}
		repo := &fakeJobRepository{createErr: errors.New("db down")}
		svc := NewSubmitJobService(gen, repo)

		_, err := svc.Execute(context.Background(), SubmitJobRequest{Prompt: "a real prompt"})
		if err == nil {
			t.Fatal("expected an error when the repository fails")
		}
	})
}
