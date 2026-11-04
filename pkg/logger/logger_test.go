package logger

import (
	"bytes"
	"context"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestContextHandlerAddsRequestID(t *testing.T) {
	var buf bytes.Buffer
	log := slog.New(contextHandler{slog.NewTextHandler(&buf, nil)})

	log.InfoContext(WithRequestID(context.Background(), "abc123"), "request")
	log.InfoContext(context.Background(), "no request")

	lines := bytes.Split(bytes.TrimSpace(buf.Bytes()), []byte("\n"))
	require.Len(t, lines, 2)
	require.Contains(t, string(lines[0]), "request_id=abc123")
	require.NotContains(t, string(lines[1]), "request_id")
}

func TestRequestIDFromContext(t *testing.T) {
	require.Equal(t, "", RequestIDFromContext(context.Background()))
	require.Equal(t, "x", RequestIDFromContext(WithRequestID(context.Background(), "x")))
}
