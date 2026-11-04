package v1

import (
	"net/http"

	"hr-system/internal/handler/response"
	"hr-system/internal/models"

	"github.com/gin-gonic/gin"
)

func (h *Handler) CreateEmployee(c *gin.Context) {
	var req createEmployeeRequest
	if !decodeJSON(c, &req) {
		return
	}
	hireDate, ok := parseDate(c, "hire_date", req.HireDate)
	if !ok {
		return
	}
	e, err := h.services.Employee.Create(c.Request.Context(), models.EmployeeInput{
		Email:        req.Email,
		Password:     req.Password,
		FirstName:    req.FirstName,
		LastName:     req.LastName,
		Phone:        req.Phone,
		DepartmentID: req.DepartmentID,
		PositionID:   req.PositionID,
		HireDate:     hireDate,
	})
	if err != nil {
		response.ServiceError(c, err)
		return
	}
	response.JSON(c, http.StatusCreated, newEmployeeResponse(e))
}

func (h *Handler) GetEmployee(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	e, err := h.services.Employee.Get(c.Request.Context(), id)
	if err != nil {
		response.ServiceError(c, err)
		return
	}
	response.JSON(c, http.StatusOK, newEmployeeResponse(e))
}

func (h *Handler) ListEmployees(c *gin.Context) {
	filter, ok := parseEmployeeFilter(c)
	if !ok {
		return
	}
	page, ok := intParam(c, "page", 1)
	if !ok {
		return
	}
	limit, ok := intParam(c, "limit", 20)
	if !ok {
		return
	}
	filter.Page, filter.Limit = page, limit

	result, err := h.services.Employee.List(c.Request.Context(), filter)
	if err != nil {
		response.ServiceError(c, err)
		return
	}
	response.JSON(c, http.StatusOK, newEmployeePageResponse(result))
}

func parseEmployeeFilter(c *gin.Context) (models.EmployeeFilter, bool) {
	departmentID, ok := optionalInt64(c, "department_id")
	if !ok {
		return models.EmployeeFilter{}, false
	}
	positionID, ok := optionalInt64(c, "position_id")
	if !ok {
		return models.EmployeeFilter{}, false
	}
	return models.EmployeeFilter{
		Q:            c.Query("q"),
		DepartmentID: departmentID,
		PositionID:   positionID,
		Status:       c.Query("status"),
	}, true
}

func (h *Handler) UpdateEmployee(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	var req updateEmployeeRequest
	if !decodeJSON(c, &req) {
		return
	}
	e, err := h.services.Employee.Update(c.Request.Context(), id, models.EmployeeUpdate{
		FirstName:    req.FirstName,
		LastName:     req.LastName,
		Phone:        req.Phone,
		DepartmentID: req.DepartmentID,
		PositionID:   req.PositionID,
		Status:       req.Status,
	})
	if err != nil {
		response.ServiceError(c, err)
		return
	}
	response.JSON(c, http.StatusOK, newEmployeeResponse(e))
}

func (h *Handler) DeleteEmployee(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	if err := h.services.Employee.Delete(c.Request.Context(), id); err != nil {
		response.ServiceError(c, err)
		return
	}
	response.NoContent(c)
}
