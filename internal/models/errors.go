package models

import (
	"errors"
	"strings"
)

var (
	ErrNotFound           = errors.New("not found")
	ErrInvalidInput       = errors.New("invalid input")
	ErrConflict           = errors.New("conflict")
	ErrForbidden          = errors.New("forbidden")
	ErrInvalidToken       = errors.New("invalid token")
	ErrInvalidCredentials = errors.New("invalid email or password")
)

func Message(err, sentinel error) string {
	text := err.Error()
	marker := sentinel.Error() + ": "
	if i := strings.Index(text, marker); i >= 0 {
		return text[i+len(marker):]
	}
	return sentinel.Error()
}
