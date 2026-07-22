package relay

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"testing"
	"time"

	"github.com/DavidOrtegaFarrerons/infergrid/internal/infrastructure/postgres"
)

// testLogger keeps relay log output out of test results.
var testLogger = slog.New(slog.DiscardHandler)

type publishCall struct {
	messageID string
	payload   json.RawMessage
}

type fakePublisher struct {
	calls   []publishCall
	failAt  int  // 1-based index of the call that should fail; 0 = never
	failed  bool // set once the failing call is hit
	failErr error
}

func (p *fakePublisher) Publish(ctx context.Context, messageID string, payload json.RawMessage) error {
	if p.failAt != 0 && len(p.calls)+1 == p.failAt {
		p.failed = true
		if p.failErr != nil {
			return p.failErr
		}
		return errors.New("publish failed")
	}
	p.calls = append(p.calls, publishCall{messageID: messageID, payload: payload})
	return nil
}

type fakeOutboxSource struct {
	rows         []postgres.OutboxRow
	fetchErr     error
	marked       []int64
	markErrFor   map[int64]error
	pruneCutoff  time.Time
	pruneDeleted int64
	pruneErr     error
}

func (s *fakeOutboxSource) FetchUnpublished(ctx context.Context, limit int) ([]postgres.OutboxRow, error) {
	if s.fetchErr != nil {
		return nil, s.fetchErr
	}
	if limit < len(s.rows) {
		return s.rows[:limit], nil
	}
	return s.rows, nil
}

func (s *fakeOutboxSource) MarkPublished(ctx context.Context, id int64) error {
	if err, ok := s.markErrFor[id]; ok {
		return err
	}
	s.marked = append(s.marked, id)
	return nil
}

func (s *fakeOutboxSource) DeletePublishedBefore(ctx context.Context, cutoff time.Time) (int64, error) {
	s.pruneCutoff = cutoff
	return s.pruneDeleted, s.pruneErr
}

func row(id int64) postgres.OutboxRow {
	return postgres.OutboxRow{
		ID:        id,
		EventType: "job.created",
		Payload:   json.RawMessage(`{"job_id":"j"}`),
	}
}

func TestRelay_publishPending(t *testing.T) {
	t.Run("publishes every row in order and marks each published", func(t *testing.T) {
		source := &fakeOutboxSource{rows: []postgres.OutboxRow{row(1), row(2), row(3)}}
		pub := &fakePublisher{}
		r := NewRelay(source, pub, testLogger)

		r.publishPending(context.Background())

		if len(pub.calls) != 3 {
			t.Fatalf("published %d rows, want 3", len(pub.calls))
		}
		wantIDs := []string{"1", "2", "3"}
		for i, want := range wantIDs {
			if pub.calls[i].messageID != want {
				t.Errorf("publish %d messageID = %q, want %q", i, pub.calls[i].messageID, want)
			}
		}
		if len(source.marked) != 3 {
			t.Fatalf("marked %d rows, want 3", len(source.marked))
		}
	})

	t.Run("message id is the outbox row id, not the job id", func(t *testing.T) {
		source := &fakeOutboxSource{rows: []postgres.OutboxRow{row(42)}}
		pub := &fakePublisher{}
		r := NewRelay(source, pub, testLogger)

		r.publishPending(context.Background())

		if pub.calls[0].messageID != "42" {
			t.Errorf("messageID = %q, want %q", pub.calls[0].messageID, "42")
		}
	})

	t.Run("publish failure aborts the batch and leaves the row unmarked", func(t *testing.T) {
		source := &fakeOutboxSource{rows: []postgres.OutboxRow{row(1), row(2), row(3)}}
		pub := &fakePublisher{failAt: 2} // second publish fails
		r := NewRelay(source, pub, testLogger)

		r.publishPending(context.Background())

		// Only row 1 published successfully.
		if len(pub.calls) != 1 || pub.calls[0].messageID != "1" {
			t.Fatalf("published = %v, want only row 1", pub.calls)
		}
		// Row 2 failed to publish, row 3 must never be attempted.
		if !pub.failed {
			t.Error("expected the failing publish to be reached")
		}
		// Only row 1 is marked; the failed row 2 and the skipped row 3 are not.
		if len(source.marked) != 1 || source.marked[0] != 1 {
			t.Errorf("marked = %v, want [1]", source.marked)
		}
	})

	t.Run("fetch error publishes and marks nothing", func(t *testing.T) {
		source := &fakeOutboxSource{fetchErr: errors.New("db down")}
		pub := &fakePublisher{}
		r := NewRelay(source, pub, testLogger)

		r.publishPending(context.Background())

		if len(pub.calls) != 0 {
			t.Errorf("published %d rows, want 0", len(pub.calls))
		}
		if len(source.marked) != 0 {
			t.Errorf("marked %d rows, want 0", len(source.marked))
		}
	})

	t.Run("mark failure is tolerated and the batch continues", func(t *testing.T) {
		source := &fakeOutboxSource{
			rows:       []postgres.OutboxRow{row(1), row(2)},
			markErrFor: map[int64]error{1: errors.New("mark failed")},
		}
		pub := &fakePublisher{}
		r := NewRelay(source, pub, testLogger)

		r.publishPending(context.Background())

		// A mark failure on row 1 must not stop row 2 from publishing.
		if len(pub.calls) != 2 {
			t.Fatalf("published %d rows, want 2 despite mark failure", len(pub.calls))
		}
		// Row 1's mark failed, so only row 2 lands in marked.
		if len(source.marked) != 1 || source.marked[0] != 2 {
			t.Errorf("marked = %v, want [2]", source.marked)
		}
	})

	t.Run("empty batch is a no-op", func(t *testing.T) {
		source := &fakeOutboxSource{}
		pub := &fakePublisher{}
		r := NewRelay(source, pub, testLogger)

		r.publishPending(context.Background())

		if len(pub.calls) != 0 || len(source.marked) != 0 {
			t.Errorf("expected no work on empty batch")
		}
	})
}

func TestRelay_prune(t *testing.T) {
	t.Run("deletes with a cutoff one retention window before now", func(t *testing.T) {
		source := &fakeOutboxSource{pruneDeleted: 5}
		r := NewRelay(source, &fakePublisher{}, testLogger)

		// prune calls time.Now() internally, so bound the expected cutoff between
		// the moment before and after the call, each shifted back by the retention
		// window. This asserts the retention math without a flaky fixed tolerance.
		before := time.Now()
		r.prune(context.Background())
		after := time.Now()

		if source.pruneCutoff.Before(before.Add(-r.retention)) {
			t.Errorf("cutoff %s is older than before-retention %s", source.pruneCutoff, before.Add(-r.retention))
		}
		if source.pruneCutoff.After(after.Add(-r.retention)) {
			t.Errorf("cutoff %s is newer than after-retention %s", source.pruneCutoff, after.Add(-r.retention))
		}
	})

	t.Run("delete error is tolerated and does not panic", func(t *testing.T) {
		source := &fakeOutboxSource{pruneErr: errors.New("db down")}
		r := NewRelay(source, &fakePublisher{}, testLogger)

		r.prune(context.Background()) // must simply log and return
	})
}

func TestRelay_Run_stopsOnContextCancel(t *testing.T) {
	source := &fakeOutboxSource{}
	pub := &fakePublisher{}
	r := NewRelay(source, pub, testLogger)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // already cancelled: Run must return without blocking

	done := make(chan error, 1)
	go func() { done <- r.Run(ctx) }()

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("Run returned %v, want nil on context cancel", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Run did not return after context cancel")
	}
}
