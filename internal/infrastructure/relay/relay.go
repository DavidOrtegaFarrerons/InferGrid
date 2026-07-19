package relay

import (
	"context"
	"encoding/json"
	"log"
	"strconv"
	"time"

	"github.com/DavidOrtegaFarrerons/infergrid/internal/infrastructure/postgres"
)

type OutboxSource interface {
	FetchUnpublished(ctx context.Context, limit int) ([]postgres.OutboxRow, error)
	MarkPublished(ctx context.Context, id int64) error
	DeletePublishedBefore(ctx context.Context, cutoff time.Time) (int64, error)
}

type Publisher interface {
	Publish(ctx context.Context, messageID string, payload json.RawMessage) error
}

const (
	defaultPublishInterval = 1 * time.Second
	defaultPruneInterval   = 1 * time.Hour
	defaultRetention       = 24 * time.Hour
	defaultBatchSize       = 10
)

type Relay struct {
	outboxSource    OutboxSource
	publisher       Publisher
	publishInterval time.Duration
	pruneInterval   time.Duration
	retention       time.Duration
	batchSize       int
}

func NewRelay(outboxSource OutboxSource, publisher Publisher) Relay {
	return Relay{
		outboxSource:    outboxSource,
		publisher:       publisher,
		publishInterval: defaultPublishInterval,
		pruneInterval:   defaultPruneInterval,
		retention:       defaultRetention,
		batchSize:       defaultBatchSize,
	}
}

func (r Relay) Run(ctx context.Context) error {
	publishTicker := time.NewTicker(r.publishInterval)
	defer publishTicker.Stop()
	pruneTicker := time.NewTicker(r.pruneInterval)
	defer pruneTicker.Stop()

	for {
		select {
		case <-publishTicker.C:
			r.publishPending(ctx)
		case <-pruneTicker.C:
			r.prune(ctx)
		case <-ctx.Done():
			return nil
		}
	}
}

func (r Relay) publishPending(ctx context.Context) {
	rows, err := r.outboxSource.FetchUnpublished(ctx, r.batchSize)
	if err != nil {
		log.Printf("relay: fetch unpublished events: %v", err)
		return
	}

	for _, row := range rows {
		id := strconv.FormatInt(row.ID, 10)
		if err := r.publisher.Publish(ctx, id, row.Payload); err != nil {
			log.Printf("relay: publish event %d: %v", row.ID, err)
			return
		}

		if err := r.outboxSource.MarkPublished(ctx, row.ID); err != nil {
			log.Printf("relay: mark event %d published: %v", row.ID, err)
		}
	}
}

func (r Relay) prune(ctx context.Context) {
	cutoff := time.Now().Add(-r.retention)
	deleted, err := r.outboxSource.DeletePublishedBefore(ctx, cutoff)
	if err != nil {
		log.Printf("relay: prune published events: %v", err)
		return
	}
	if deleted > 0 {
		log.Printf("relay: pruned %d published events", deleted)
	}
}
