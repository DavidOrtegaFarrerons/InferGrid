package relay

import (
	"context"
	"encoding/json"
	"log/slog"
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
	logger          *slog.Logger
	publishInterval time.Duration
	pruneInterval   time.Duration
	retention       time.Duration
	batchSize       int
}

func NewRelay(outboxSource OutboxSource, publisher Publisher, logger *slog.Logger) Relay {
	return Relay{
		outboxSource:    outboxSource,
		publisher:       publisher,
		logger:          logger,
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
		r.logger.Error("fetch unpublished events", "err", err)
		return
	}

	for _, row := range rows {
		id := strconv.FormatInt(row.ID, 10)
		if err := r.publisher.Publish(ctx, id, row.Payload); err != nil {
			r.logger.Error("publish event", "event_id", row.ID, "err", err)
			return
		}

		if err := r.outboxSource.MarkPublished(ctx, row.ID); err != nil {
			r.logger.Warn("mark published event", "event_id", row.ID, "err", err)
		}

		r.logger.Debug("event published", "event_id", row.ID)
	}
}

func (r Relay) prune(ctx context.Context) {
	cutoff := time.Now().Add(-r.retention)
	deleted, err := r.outboxSource.DeletePublishedBefore(ctx, cutoff)
	if err != nil {
		r.logger.Error("prune published events", "err", err)
		return
	}
	if deleted > 0 {
		r.logger.Info("pruned published events", "deleted", deleted)
	}
}
