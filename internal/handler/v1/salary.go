package v1

import (
	"net/http"

	"hr-system/internal/handler/response"
	"hr-system/internal/models"

	"github.com/gin-gonic/gin"
)

func (h *Handler) AddSalary(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	var req salaryRequest
	if !decodeJSON(c, &req) {
		return
	}
	date, ok := parseDate(c, "effective_date", req.EffectiveDate)
	if !ok {
		return
	}
	s, err := h.services.Salary.Add(c.Request.Context(), id, models.SalaryInput{Amount: req.Amount, EffectiveDate: date})
	if err != nil {
		response.ServiceError(c, err)
		return
	}
	response.JSON(c, http.StatusCreated, newSalaryResponse(s))
}

func (h *Handler) ListEmployeeSalaries(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	items, err := h.services.Salary.ListByEmployee(c.Request.Context(), id)
	if err != nil {
		response.ServiceError(c, err)
		return
	}
	respondSalaries(c, items)
}

func (h *Handler) MySalaries(c *gin.Context) {
	cu, ok := currentUser(c)
	if !ok {
		return
	}
	items, err := h.services.Salary.ListMine(c.Request.Context(), cu)
	if err != nil {
		response.ServiceError(c, err)
		return
	}
	respondSalaries(c, items)
}

func respondSalaries(c *gin.Context, items []models.Salary) {
	result := make([]salaryResponse, 0, len(items))
	for i := range items {
		result = append(result, newSalaryResponse(&items[i]))
	}
	response.JSON(c, http.StatusOK, result)
}
