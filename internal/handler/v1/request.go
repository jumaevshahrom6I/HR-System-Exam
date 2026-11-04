package v1

import (
	"net/http"
	"strconv"
	"time"

	"hr-system/internal/handler/middleware"
	"hr-system/internal/handler/response"
	"hr-system/internal/models"

	"github.com/gin-gonic/gin"
)

const (
	maxBodyBytes = 1 << 20
	dateLayout   = "2006-01-02"
)

func decodeJSON(c *gin.Context, dst any) bool {
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxBodyBytes)
	if err := c.ShouldBindJSON(dst); err != nil {
		response.Error(c, http.StatusBadRequest, "invalid request body")
		return false
	}
	return true
}

func parseID(c *gin.Context) (int64, bool) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		response.Error(c, http.StatusBadRequest, "invalid id")
		return 0, false
	}
	return id, true
}

func parseDate(c *gin.Context, field, value string) (time.Time, bool) {
	t, err := time.Parse(dateLayout, value)
	if err != nil {
		response.Error(c, http.StatusBadRequest, field+" must be in YYYY-MM-DD format")
		return time.Time{}, false
	}
	return t, true
}

func formatDate(t time.Time) string { return t.Format(dateLayout) }

func currentUser(c *gin.Context) (models.CurrentUser, bool) {
	claims, ok := middleware.ClaimsFromContext(c)
	if !ok {
		response.Error(c, http.StatusUnauthorized, "missing or invalid token")
		return models.CurrentUser{}, false
	}
	return models.CurrentUser{ID: claims.UserID, Role: claims.Role}, true
}

func optionalInt64(c *gin.Context, name string) (*int64, bool) {
	raw := c.Query(name)
	if raw == "" {
		return nil, true
	}
	v, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || v <= 0 {
		response.Error(c, http.StatusBadRequest, "invalid "+name)
		return nil, false
	}
	return &v, true
}

func intParam(c *gin.Context, name string, def int) (int, bool) {
	v, err := strconv.Atoi(c.DefaultQuery(name, strconv.Itoa(def)))
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid "+name)
		return 0, false
	}
	return v, true
}
