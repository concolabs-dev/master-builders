package auth

import (
	"errors"
	"net/http"
	"sync"

	"github.com/gin-gonic/gin"
)

type OwnerResolver func(ctx *gin.Context, resource string, id string) (ownerUserID string, err error)

var (
	ownerResolvers = map[string]OwnerResolver{}
	ownMu          sync.RWMutex
)

// Register a resolver that knows how to find the owner of a resource by its ID.
func RegisterOwnerResolver(resource string, r OwnerResolver) {
	ownMu.Lock()
	defer ownMu.Unlock()
	ownerResolvers[resource] = r
}

func resolveOwnerID(ctx *gin.Context, resource, id string) (string, error) {
	ownMu.RLock()
	r, ok := ownerResolvers[resource]
	ownMu.RUnlock()
	if !ok {
		return "", errors.New("no owner resolver registered for resource: " + resource)
	}
	return r(ctx, resource, id)
}

func hasAnyRole(c *gin.Context, allowed map[string]struct{}) bool {
	if v, ok := c.Get("roles"); ok {
		switch roles := v.(type) {
		case []string:
			for _, r := range roles {
				if _, ok := allowed[r]; ok {
					return true
				}
			}
		case []interface{}:
			for _, rv := range roles {
				if rs, ok := rv.(string); ok {
					if _, ok := allowed[rs]; ok {
						return true
					}
				}
			}
		case string:
			if _, ok := allowed[roles]; ok {
				return true
			}
		}
	}
	return false
}

// RequireOwnershipOrRoles allows if the user owns the resource OR has any of the given roles (e.g., "admin").
func RequireOwnershipOrRoles(resource string, roles ...string) gin.HandlerFunc {
	roleSet := make(map[string]struct{}, len(roles))
	for _, r := range roles {
		roleSet[r] = struct{}{}
	}

	return func(c *gin.Context) {
		userID := c.GetString("userID") // ensure your auth sets this in context
		if userID == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}

		// Short-circuit: role-based access (e.g., admin)
		if hasAnyRole(c, roleSet) {
			c.Next()
			return
		}

		// Ownership check
		resID := c.Param("id")
		ownerID, err := resolveOwnerID(c, resource, resID)
		if err != nil {
			// Decide how to map errors; example: not found vs internal error
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "forbidden"})
			return
		}
		if ownerID == userID {
			c.Next()
			return
		}

		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "forbidden"})
	}
}
