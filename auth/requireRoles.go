package auth

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// checks if any of the user's roles match the allowed ones.
func RequireRoles(allowedRoles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		roleValues, exists := c.Get("roles")
		if !exists {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "authentication required"})
			return
		}

		userRoles, ok := roleValues.([]string)
		if !ok {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "invalid roles format"})
			return
		}

		// Check if any allowed role is in user's roles
		for _, allowed := range allowedRoles {
			for _, userRole := range userRoles {
				if allowed == userRole {
					c.Next()
					return
				}
			}
		}

		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "insufficient permissions"})
	}
}
