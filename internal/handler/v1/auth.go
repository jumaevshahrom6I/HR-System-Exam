package v1

import (
	"net/http"

	"hr-system/internal/handler/response"
	"hr-system/internal/models"

	"github.com/gin-gonic/gin"
)

func (h *Handler) Login(c *gin.Context) {
	var req loginRequest
	if !decodeJSON(c, &req) {
		return
	}
	pair, err := h.services.Auth.Login(c.Request.Context(), models.LoginInput{Email: req.Email, Password: req.Password})
	if err != nil {
		response.ServiceError(c, err)
		return
	}
	response.JSON(c, http.StatusOK, newTokenResponse(pair))
}

func (h *Handler) Refresh(c *gin.Context) {
	var req refreshRequest
	if !decodeJSON(c, &req) {
		return
	}
	pair, err := h.services.Auth.Refresh(c.Request.Context(), req.RefreshToken)
	if err != nil {
		response.ServiceError(c, err)
		return
	}
	response.JSON(c, http.StatusOK, newTokenResponse(pair))
}

func (h *Handler) Logout(c *gin.Context) {
	var req refreshRequest
	if !decodeJSON(c, &req) {
		return
	}
	if err := h.services.Auth.Logout(c.Request.Context(), req.RefreshToken); err != nil {
		response.ServiceError(c, err)
		return
	}
	response.NoContent(c)
}
