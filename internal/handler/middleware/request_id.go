package middleware

import (
	"crypto/rand"
	"encoding/hex"

	"hr-system/pkg/logger"

	"github.com/gin-gonic/gin"
)

const requestIDHeader = "X-Request-ID"

func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		b := make([]byte, 8)
		id := "unknown"
		if _, err := rand.Read(b); err == nil {
			id = hex.EncodeToString(b)
		}
		c.Header(requestIDHeader, id)
		c.Request = c.Request.WithContext(logger.WithRequestID(c.Request.Context(), id))
		c.Next()
	}
}
