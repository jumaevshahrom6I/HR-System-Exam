package v1

import (
	"net/http"

	"hr-system/internal/handler/response"

	"github.com/gin-gonic/gin"
)

func (h *Handler) Me(c *gin.Context) {
	cu, ok := currentUser(c)
	if !ok {
		return
	}
	profile, err := h.services.Employee.Me(c.Request.Context(), cu)
	if err != nil {
		response.ServiceError(c, err)
		return
	}
	response.JSON(c, http.StatusOK, newMeResponse(profile))
}
