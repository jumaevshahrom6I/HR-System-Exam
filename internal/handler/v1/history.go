package v1

import (
	"net/http"

	"hr-system/internal/handler/response"

	"github.com/gin-gonic/gin"
)

func (h *Handler) EmployeeHistory(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	items, err := h.services.Employee.History(c.Request.Context(), id)
	if err != nil {
		response.ServiceError(c, err)
		return
	}
	result := make([]historyResponse, 0, len(items))
	for i := range items {
		result = append(result, newHistoryResponse(&items[i]))
	}
	response.JSON(c, http.StatusOK, result)
}
