package logger

import (
	"context"
	"fmt"
	"log/slog"
	"os"
)

const (
	EnvLocal = "local"
	Prod     = "prod"
)

type Config struct {
	Env   string
	Level slog.Level
	File  string
}

func New(cfg Config) (*slog.Logger, func() error, error) {
	opts := &slog.HandlerOptions{
		Level: cfg.Level}

	if cfg.Env != EnvLocal {
		return slog.New(contextHandler{slog.NewJSONHandler(os.Stdout, opts)}), func() error {
			return nil
		}, nil
	}

	f, err := os.OpenFile(cfg.File, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)

	if err != nil {
		return nil, nil, fmt.Errorf("open log file: %w", err)
	}

	return slog.New(contextHandler{slog.NewTextHandler(f, opts)}), f.Close, nil
}

type ctxKey struct{}

func WithRequestID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, ctxKey{}, id)
}

func RequestIDFromContext(ctx context.Context) string {
	id, _ := ctx.Value(ctxKey{}).(string)
	return id
}

type contextHandler struct {
	slog.Handler
}

func (h contextHandler) Handle(ctx context.Context, r slog.Record) error {
	if id := RequestIDFromContext(ctx); id != "" {
		r.AddAttrs(slog.String("request_id", id))
	}
	return h.Handler.Handle(ctx, r)
}

func (h contextHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return contextHandler{h.Handler.WithAttrs(attrs)}
}

func (h contextHandler) WithGroup(name string) slog.Handler {
	return contextHandler{h.Handler.WithGroup(name)}
}
