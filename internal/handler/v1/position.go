package v1

import (
	"net/http"

	"hr-system/internal/handler/response"
	"hr-system/internal/models"

	"github.com/gin-gonic/gin"
)

func (h *Handler) CreatePosition(c *gin.Context) {
	var req positionRequest
	if !decodeJSON(c, &req) {
		return
	}
	p, err := h.services.Position.Create(c.Request.Context(), models.PositionInput{Name: req.Name, BaseSalary: req.BaseSalary})
	if err != nil {
		response.ServiceError(c, err)
		return
	}
	response.JSON(c, http.StatusCreated, newPositionResponse(p))
}

func (h *Handler) ListPositions(c *gin.Context) {
	items, err := h.services.Position.List(c.Request.Context())
	if err != nil {
		response.ServiceError(c, err)
		return
	}
	result := make([]positionResponse, 0, len(items))
	for i := range items {
		result = append(result, newPositionResponse(&items[i]))
	}
	response.JSON(c, http.StatusOK, result)
}

func (h *Handler) GetPosition(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	p, err := h.services.Position.Get(c.Request.Context(), id)
	if err != nil {
		response.ServiceError(c, err)
		return
	}
	response.JSON(c, http.StatusOK, newPositionResponse(p))
}

func (h *Handler) UpdatePosition(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	var req positionRequest
	if !decodeJSON(c, &req) {
		return
	}
	p, err := h.services.Position.Update(c.Request.Context(), id, models.PositionInput{Name: req.Name, BaseSalary: req.BaseSalary})
	if err != nil {
		response.ServiceError(c, err)
		return
	}
	response.JSON(c, http.StatusOK, newPositionResponse(p))
}

func (h *Handler) DeletePosition(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	if err := h.services.Position.Delete(c.Request.Context(), id); err != nil {
		response.ServiceError(c, err)
		return
	}
	response.NoContent(c)
}
