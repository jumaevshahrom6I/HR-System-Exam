package models

import (
	"errors"
	"fmt"
	"testing"
)

func TestMessage(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		sentinel error
		want     string
	}{
		{"sentinel only", ErrNotFound, ErrNotFound, "not found"},
		{"validation text", fmt.Errorf("%w: name must be 1-100 characters", ErrInvalidInput), ErrInvalidInput, "name must be 1-100 characters"},
		{"wrapped by service", fmt.Errorf("create department: %w", fmt.Errorf("%w: already exists", ErrConflict)), ErrConflict, "already exists"},
		{"no marker in chain", fmt.Errorf("create department: %w", ErrConflict), ErrConflict, "conflict"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if !errors.Is(tt.err, tt.sentinel) {
				t.Fatalf("broken test: %v does not wrap %v", tt.err, tt.sentinel)
			}
			if got := Message(tt.err, tt.sentinel); got != tt.want {
				t.Fatalf("Message = %q, want %q", got, tt.want)
			}
		})
	}
}
