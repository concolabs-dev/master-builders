package auth

import (
	"log"
	"strings"
	"net/http"
	"github.com/gin-gonic/gin"
)

func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader != "" && strings.HasPrefix(authHeader, "Bearer ") {
			token := strings.TrimPrefix(authHeader, "Bearer ")

			//log.Println("Token found: ", token)

			roles, userID, err := ParseJWT(token)
			if err != nil {
				// Invalid token
				log.Println("Invalid Token can not set roles")
				c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "validation failed"})
				return
			}

			// Valid token: set values
			c.Set("userID", userID)
			c.Set("roles", roles)
		} else {
			// No token — proceed (public access)
			log.Println("No token found. Proceed with public access")
		}
		c.Next()
	}
}

// // New middleware to check for the API secret header.
// func apiSecretMiddleware() gin.HandlerFunc {
// 	return func(c *gin.Context) {
// 		expectedSecret := os.Getenv("BACKEND_API_SECRET")
// 		providedSecret := c.GetHeader("api-secert")
// 		if providedSecret == "" || providedSecret != expectedSecret {
// 			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
// 			return
// 		}
// 		c.Next()
// 	}
// }
