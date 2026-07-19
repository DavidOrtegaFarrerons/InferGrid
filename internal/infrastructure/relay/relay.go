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

type Relay struct {
	outboxSource OutboxSource
	publisher    Publisher
}

func NewRelay(outboxSource OutboxSource, publisher Publisher) Relay {
	return Relay{
		outboxSource: outboxSource,
		publisher:    publisher,
	}
}

func (r Relay) Run(ctx context.Context) error {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			rows, err := r.outboxSource.FetchUnpublished(ctx, 10)
			if err != nil {
				log.Printf("relay: fetch unpublished events: %v", err)
				continue
			}
			for _, row := range rows {
				id := strconv.FormatInt(row.ID, 10)
				if err := r.publisher.Publish(ctx, id, row.Payload); err != nil {
					log.Printf("relay: publish event %d: %v", row.ID, err)
					break
				}

				if err := r.outboxSource.MarkPublished(ctx, row.ID); err != nil {
					log.Printf("relay: mark event %d published: %v", row.ID, err)
				}
			}
		case <-ctx.Done():
			return nil
		}
	}

}
