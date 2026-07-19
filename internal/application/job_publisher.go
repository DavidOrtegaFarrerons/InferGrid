package application

import (
	"context"
	"encoding/json"
)

type JobPublisher interface {
	Publish(ctx context.Context, messageID string, payload json.RawMessage) error
}
