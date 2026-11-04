package v1

import (
	"context"
	"net/http"
	"time"

	"hr-system/internal/handler/response"

	"github.com/gin-gonic/gin"
)

func (h *Handler) Health(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
	defer cancel()

	for _, check := range h.checks {
		if err := check.Ping(ctx); err != nil {
			response.Error(c, http.StatusServiceUnavailable, check.Name+" is unavailable")
			return
		}
	}
	response.JSON(c, http.StatusOK, gin.H{"status": "ok"})
}
