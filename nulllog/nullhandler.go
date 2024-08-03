package nulllog

import (
	"context"
	"log/slog"
)

type Handler struct{}

// Enabled implements slog.Handler.
func (h *Handler) Enabled(context.Context, slog.Level) bool {
	return false
}

// Handle implements slog.Handler.
func (h *Handler) Handle(context.Context, slog.Record) error {
	return nil
}

// WithAttrs implements slog.Handler.
func (h *Handler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return h
}

// WithGroup implements slog.Handler.
func (h *Handler) WithGroup(name string) slog.Handler {
	return h
}

var _ slog.Handler = &Handler{}

func Logger() *slog.Logger {
	return slog.New(&Handler{})
}
