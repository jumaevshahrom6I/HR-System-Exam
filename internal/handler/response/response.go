package response

import (
	"errors"
	"log/slog"
	"net/http"

	"hr-system/internal/models"

	"github.com/gin-gonic/gin"
)

func JSON(c *gin.Context, status int, v any) {
	c.JSON(status, v)
}

func NoContent(c *gin.Context) {
	c.Status(http.StatusNoContent)
}

func Error(c *gin.Context, status int, msg string) {
	fail(c, status, msg, nil)
}

func ServiceError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, models.ErrNotFound):
		fail(c, http.StatusNotFound, models.Message(err, models.ErrNotFound), err)
	case errors.Is(err, models.ErrInvalidInput):
		fail(c, http.StatusBadRequest, models.Message(err, models.ErrInvalidInput), err)
	case errors.Is(err, models.ErrConflict):
		fail(c, http.StatusConflict, models.Message(err, models.ErrConflict), err)
	case errors.Is(err, models.ErrForbidden):
		fail(c, http.StatusForbidden, models.Message(err, models.ErrForbidden), err)
	case errors.Is(err, models.ErrInvalidToken):
		fail(c, http.StatusUnauthorized, models.ErrInvalidToken.Error(), err)
	case errors.Is(err, models.ErrInvalidCredentials):
		fail(c, http.StatusUnauthorized, models.ErrInvalidCredentials.Error(), err)
	default:
		fail(c, http.StatusInternalServerError, "internal server error", err)
	}
}

func fail(c *gin.Context, status int, msg string, cause error) {
	attrs := []any{
		"method", c.Request.Method,
		"path", c.Request.URL.Path,
		"status", status,
		"error", msg,
	}
	if cause != nil {
		attrs[len(attrs)-1] = cause.Error()
	}

	ctx := c.Request.Context()
	if status >= http.StatusInternalServerError {
		slog.ErrorContext(ctx, "request failed", attrs...)
	} else {
		slog.WarnContext(ctx, "request failed", attrs...)
	}

	c.AbortWithStatusJSON(status, gin.H{"error": msg})
}
