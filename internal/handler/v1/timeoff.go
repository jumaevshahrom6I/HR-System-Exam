package v1

import (
	"net/http"

	"hr-system/internal/handler/response"
	"hr-system/internal/models"

	"github.com/gin-gonic/gin"
)

func (h *Handler) CreateTimeOff(c *gin.Context) {
	cu, ok := currentUser(c)
	if !ok {
		return
	}
	var req timeOffRequest
	if !decodeJSON(c, &req) {
		return
	}
	start, ok := parseDate(c, "start_date", req.StartDate)
	if !ok {
		return
	}
	end, ok := parseDate(c, "end_date", req.EndDate)
	if !ok {
		return
	}

	t, err := h.services.TimeOff.Create(c.Request.Context(), cu, models.TimeOffInput{
		LeaveType: req.LeaveType,
		StartDate: start,
		EndDate:   end,
	})
	if err != nil {
		response.ServiceError(c, err)
		return
	}
	response.JSON(c, http.StatusCreated, newTimeOffResponse(t))
}

func (h *Handler) ListTimeOff(c *gin.Context) {
	cu, ok := currentUser(c)
	if !ok {
		return
	}
	employeeID, ok := optionalInt64(c, "employee_id")
	if !ok {
		return
	}

	items, err := h.services.TimeOff.List(c.Request.Context(), cu, models.TimeOffFilter{
		Status:     c.Query("status"),
		EmployeeID: employeeID,
	})
	if err != nil {
		response.ServiceError(c, err)
		return
	}
	result := make([]timeOffResponse, 0, len(items))
	for i := range items {
		result = append(result, newTimeOffResponse(&items[i]))
	}
	response.JSON(c, http.StatusOK, result)
}

func (h *Handler) DecideTimeOff(c *gin.Context) {
	cu, ok := currentUser(c)
	if !ok {
		return
	}
	id, ok := parseID(c)
	if !ok {
		return
	}
	var req timeOffDecisionRequest
	if !decodeJSON(c, &req) {
		return
	}

	t, err := h.services.TimeOff.UpdateStatus(c.Request.Context(), cu, id, req.Status)
	if err != nil {
		response.ServiceError(c, err)
		return
	}
	response.JSON(c, http.StatusOK, newTimeOffResponse(t))
}
