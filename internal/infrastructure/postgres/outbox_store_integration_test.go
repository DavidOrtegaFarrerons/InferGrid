//go:build integration

package postgres

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// insertOutboxRow inserts one row directly (bypassing the store) so the test can
// control published_at precisely — the store's own insert always leaves it NULL.
func insertOutboxRow(t *testing.T, aggregateID string, publishedAt any) {
	t.Helper()
	_, err := testDB.Exec(
		`INSERT INTO outbox (aggregate_type, aggregate_id, event_type, payload, published_at)
		 VALUES ('job', $1, 'job.created', $2, $3)`,
		aggregateID, json.RawMessage(`{}`), publishedAt,
	)
	assert.NoError(t, err)
}

func TestOutboxStore_DeletePublishedBefore(t *testing.T) {
	resetDB(t)

	now := time.Now()
	cutoff := now.Add(-24 * time.Hour)

	// Three rows spanning the retention boundary:
	insertOutboxRow(t, "old", now.Add(-48*time.Hour))  // published before cutoff -> deleted
	insertOutboxRow(t, "fresh", now.Add(-1*time.Hour)) // published after cutoff  -> kept
	insertOutboxRow(t, "unpublished", nil)             // never published (NULL)   -> kept

	store := NewOutboxStore(testDB)
	deleted, err := store.DeletePublishedBefore(context.Background(), cutoff)
	assert.NoError(t, err)

	// Only the single old, published row falls outside the retention window.
	assert.Equal(t, int64(1), deleted)

	var total int
	err = testDB.QueryRow(`SELECT count(*) FROM outbox`).Scan(&total)
	assert.NoError(t, err)
	assert.Equal(t, 2, total, "fresh and unpublished rows must survive")

	// The old row is gone
	var oldCount int
	err = testDB.QueryRow(`SELECT count(*) FROM outbox WHERE aggregate_id = 'old'`).Scan(&oldCount)
	assert.NoError(t, err)
	assert.Equal(t, 0, oldCount)

	// ...and the unpublished row is untouched, proving the published_at IS NOT NULL
	// guard: a NULL published_at is never "older than" the cutoff and must not be reaped.
	var unpublishedCount int
	err = testDB.QueryRow(`SELECT count(*) FROM outbox WHERE aggregate_id = 'unpublished'`).Scan(&unpublishedCount)
	assert.NoError(t, err)
	assert.Equal(t, 1, unpublishedCount)
}
