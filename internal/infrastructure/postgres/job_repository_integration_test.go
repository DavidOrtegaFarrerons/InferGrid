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
