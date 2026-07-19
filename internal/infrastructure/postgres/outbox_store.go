package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
)

type OutboxStore struct {
	db *sql.DB
}

func NewOutboxStore(db *sql.DB) OutboxStore {
	return OutboxStore{db: db}
}

type OutboxRow struct {
	ID        int64
	EventType string
	Payload   json.RawMessage
}

type outboxEvent struct {
	aggregateType string
	aggregateID   string
	eventType     string
	payload       json.RawMessage
}

func (s OutboxStore) insertOutboxEvent(ctx context.Context, tx *sql.Tx, event outboxEvent) error {
	query := `INSERT INTO outbox 
    (aggregate_type, aggregate_id, event_type, payload)
	VALUES ($1, $2, $3, $4)
    `

	_, err := tx.ExecContext(ctx, query, event.aggregateType, event.aggregateID, event.eventType, event.payload)
	return err
}

func (s OutboxStore) FetchUnpublished(ctx context.Context, limit int) ([]OutboxRow, error) {
	query := `SELECT id, event_type, payload FROM outbox WHERE published_at IS NULL ORDER BY id LIMIT $1`

	rows, err := s.db.QueryContext(ctx, query, limit)
	if err != nil {
		return nil, err
	}

	defer rows.Close()

	events := make([]OutboxRow, 0)
	for rows.Next() {
		var event OutboxRow
		err := rows.Scan(
			&event.ID,
			&event.EventType,
			&event.Payload,
		)

		if err != nil {
			return nil, err
		}

		events = append(events, event)
	}

	if rows.Err() != nil {
		return nil, rows.Err()
	}

	return events, nil
}

func (s OutboxStore) MarkPublished(ctx context.Context, id int64) error {
	query := `UPDATE outbox SET published_at = NOW() WHERE id = $1`

	_, err := s.db.ExecContext(ctx, query, id)
	return err
}
