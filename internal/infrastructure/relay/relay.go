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
}

type Publisher interface {
	Publish(ctx context.Context, messageID string, payload json.RawMessage) error
}

const (
	defaultInterval  = 1 * time.Second
	defaultBatchSize = 10
)

type Relay struct {
	outboxSource OutboxSource
	publisher    Publisher
	interval     time.Duration
	batchSize    int
}

func NewRelay(outboxSource OutboxSource, publisher Publisher) Relay {
	return Relay{
		outboxSource: outboxSource,
		publisher:    publisher,
		interval:     defaultInterval,
		batchSize:    defaultBatchSize,
	}
}

func (r Relay) Run(ctx context.Context) error {
	ticker := time.NewTicker(r.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			r.publishPending(ctx)
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
