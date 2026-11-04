package middleware

import (
	"net/http"
	"strings"

	"hr-system/internal/handler/response"
	"hr-system/internal/models"

	"github.com/gin-gonic/gin"
)

const claimsKey = "claims"

type Authenticator interface {
	Authenticate(accessToken string) (*models.AccessClaims, error)
}

func Auth(authn Authenticator) gin.HandlerFunc {
	return func(c *gin.Context) {
		scheme, token, ok := strings.Cut(c.GetHeader("Authorization"), " ")
		token = strings.TrimSpace(token)
		if !ok || !strings.EqualFold(scheme, "Bearer") || token == "" {
			response.Error(c, http.StatusUnauthorized, "missing or invalid token")
			return
		}

		claims, err := authn.Authenticate(token)
		if err != nil {
			response.ServiceError(c, err)
			return
		}

		c.Set(claimsKey, claims)
		c.Next()
	}
}

func RequireRole(role string) gin.HandlerFunc {
	return func(c *gin.Context) {
		claims, ok := ClaimsFromContext(c)
		if !ok {
			response.Error(c, http.StatusUnauthorized, "missing or invalid token")
			return
		}
		if claims.Role != role {
			response.Error(c, http.StatusForbidden, "forbidden")
			return
		}
		c.Next()
	}
}

func ClaimsFromContext(c *gin.Context) (*models.AccessClaims, bool) {
	v, ok := c.Get(claimsKey)
	if !ok {
		return nil, false
	}
	claims, ok := v.(*models.AccessClaims)
	return claims, ok && claims != nil
}
