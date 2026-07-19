//go:build integration

package postgres

import (
	"context"
	"testing"

	"github.com/DavidOrtegaFarrerons/infergrid/internal/domain/job"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestJobRepository_Create_persistsJobAndOutbox(t *testing.T) {
	resetDB(t)

	jobRepository := NewJobRepository(testDB, NewOutboxStore(testDB))

	jobID := job.ID(uuid.New().String())
	expectedInferenceJob, err := job.New(jobID, "hello AI!")
	assert.NoError(t, err)

	err = jobRepository.Create(context.Background(), expectedInferenceJob)
	assert.NoError(t, err)

	actualInferenceJob, err := jobRepository.GetByID(context.Background(), jobID)
	assert.NoError(t, err)
	assert.NotNil(t, actualInferenceJob)
	assert.Equal(t, expectedInferenceJob.ID(), actualInferenceJob.ID())
	assert.Equal(t, expectedInferenceJob.Prompt(), actualInferenceJob.Prompt())
	assert.Equal(t, expectedInferenceJob.Status(), actualInferenceJob.Status())
	assert.True(t, expectedInferenceJob.CreatedAt().Equal(actualInferenceJob.CreatedAt()),
		"CreatedAt mismatch: want %s, got %s", expectedInferenceJob.CreatedAt(), actualInferenceJob.CreatedAt())
	assert.True(t, expectedInferenceJob.UpdatedAt().Equal(actualInferenceJob.UpdatedAt()),
		"UpdatedAt mismatch: want %s, got %s", expectedInferenceJob.UpdatedAt(), actualInferenceJob.UpdatedAt())

	outboxQuery := `SELECT count(*) FROM outbox WHERE aggregate_id = $1`

	var count int
	err = testDB.QueryRow(outboxQuery, string(actualInferenceJob.ID())).Scan(&count)
	assert.NoError(t, err)
	assert.Equal(t, 1, count)
}

// TestJobRepository_Create_rollsBackJobWhenOutboxInsertFails proves the guarantee
// that makes the outbox pattern worth having: the job row and the outbox row are
// written atomically.
func TestJobRepository_Create_rollsBackJobWhenOutboxInsertFails(t *testing.T) {
	resetDB(t)

	// Force *only* the outbox insert to fail.
	_, err := testDB.Exec(`ALTER TABLE outbox ADD CONSTRAINT force_fail CHECK (false) NOT VALID`)
	assert.NoError(t, err)
	t.Cleanup(func() {
		_, _ = testDB.Exec(`ALTER TABLE outbox DROP CONSTRAINT force_fail`)
	})

	jobRepository := NewJobRepository(testDB, NewOutboxStore(testDB))

	jobID := job.ID(uuid.New().String())
	inferenceJob, err := job.New(jobID, "hello AI!")
	assert.NoError(t, err)

	// Act. Inside Create the job insert runs and succeeds; then the outbox insert
	// hits the constraint and errors, so the deferred Rollback undoes everything.
	err = jobRepository.Create(context.Background(), inferenceJob)
	assert.Error(t, err)

	// Assert the rollback: the job row must NOT exist, even though its own INSERT
	// ran cleanly moments earlier.
	var jobCount int
	err = testDB.QueryRow(`SELECT count(*) FROM jobs WHERE id = $1`, string(jobID)).Scan(&jobCount)
	assert.NoError(t, err)
	assert.Equal(t, 0, jobCount, "job row must be rolled back when the outbox insert fails")
}
