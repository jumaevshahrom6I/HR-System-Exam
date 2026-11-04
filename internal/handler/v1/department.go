package v1

import (
	"net/http"

	"hr-system/internal/handler/response"
	"hr-system/internal/models"

	"github.com/gin-gonic/gin"
)

func (h *Handler) CreateDepartment(c *gin.Context) {
	var req departmentRequest
	if !decodeJSON(c, &req) {
		return
	}
	d, err := h.services.Department.Create(c.Request.Context(), models.DepartmentInput{Name: req.Name})
	if err != nil {
		response.ServiceError(c, err)
		return
	}
	response.JSON(c, http.StatusCreated, newDepartmentResponse(d))
}

func (h *Handler) ListDepartments(c *gin.Context) {
	items, err := h.services.Department.List(c.Request.Context())
	if err != nil {
		response.ServiceError(c, err)
		return
	}
	result := make([]departmentResponse, 0, len(items))
	for i := range items {
		result = append(result, newDepartmentResponse(&items[i]))
	}
	response.JSON(c, http.StatusOK, result)
}

func (h *Handler) GetDepartment(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	d, err := h.services.Department.Get(c.Request.Context(), id)
	if err != nil {
		response.ServiceError(c, err)
		return
	}
	response.JSON(c, http.StatusOK, newDepartmentResponse(d))
}

func (h *Handler) UpdateDepartment(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	var req departmentRequest
	if !decodeJSON(c, &req) {
		return
	}
	d, err := h.services.Department.Update(c.Request.Context(), id, models.DepartmentInput{Name: req.Name})
	if err != nil {
		response.ServiceError(c, err)
		return
	}
	response.JSON(c, http.StatusOK, newDepartmentResponse(d))
}

func (h *Handler) DeleteDepartment(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	if err := h.services.Department.Delete(c.Request.Context(), id); err != nil {
		response.ServiceError(c, err)
		return
	}
	response.NoContent(c)
}
