package observability

import (
	"log/slog"
	"os"
)

func NewLogger(service string, level slog.Leveler) *slog.Logger {
	return slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: level})).With("service", service)
}
