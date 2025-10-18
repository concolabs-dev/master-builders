package auth

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
)

// checks if any of the user's roles match the allowed ones.
func RequireRoles(allowedRoles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {

		log.Println("inside the role checker")

		roleValues, exists := c.Get("roles")
		if !exists {
			log.Println("Roles could not found in context. User may not be authenticated.")
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "authentication required"})
			return
		}

		userRoles, ok := roleValues.([]string)
		if !ok {
			log.Printf("Roles in context are not of type []string. Actual type: %T, value: %v", roleValues, roleValues)
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "invalid roles format"})
			return
		}

		// Check if any allowed role is in user's roles
		for _, allowed := range allowedRoles {
			for _, userRole := range userRoles {
				log.Printf("Checking if user role '%s' matches allowed role '%s'", userRole, allowed)
				if allowed == userRole {
					log.Printf("User has required role: %s", allowed)
					c.Next()
					return
				}
			}
		}

		log.Printf("User roles %v do not include any of the allowed roles %v", userRoles, allowedRoles)
		log.Println("Required role is missing")

		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "insufficient permissions"})
	}
}
