package middleware

import (
	"log"
	"net/http"
	"strings"

	"quotation/models"
	"quotation/services"

	"github.com/gin-gonic/gin"
)

func Authenticate(service services.AuthService) gin.HandlerFunc {
	return func(c *gin.Context) {
		scheme, token, ok := strings.Cut(c.GetHeader("Authorization"), " ")
		if !ok || !strings.EqualFold(scheme, "Bearer") || token == "" || strings.Contains(token, " ") {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "authentication required"})
			return
		}
		user, err := service.Authenticate(c.Request.Context(), token)
		if err != nil {
			if err == services.ErrUnauthorized {
				c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "authentication required"})
				return
			}
			log.Printf("authenticate request: %v", err)
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
			return
		}
		c.Set("user", user)
		c.Set("auth_token", token)
		c.Next()
	}
}

func RequireAdmin() gin.HandlerFunc {
	return func(c *gin.Context) {
		value, ok := c.Get("user")
		user, valid := value.(*models.User)
		if !ok || !valid || user.Role != models.RoleAdmin {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "administrator access required"})
			return
		}
		c.Next()
	}
}
