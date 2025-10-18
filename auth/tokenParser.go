package auth

import (
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/MicahParks/keyfunc"
	"github.com/golang-jwt/jwt/v4"
)

// Claims defines your JWT payload structure.
type CustomClaims struct {
	Roles []string `json:"https://build-market.com/api/v2/roles"`
	jwt.RegisteredClaims
}

var jwks *keyfunc.JWKS

// InitializeJWKS sets up the JWKS with automatic key rotation.
func InitializeJWKS(jwksURL string) error {
	var err error
	jwks, err = keyfunc.Get(jwksURL, keyfunc.Options{
		RefreshInterval:   time.Hour,
		RefreshRateLimit:  time.Minute,
		RefreshTimeout:    10 * time.Second,
		RefreshUnknownKID: true,
		RefreshErrorHandler: func(err error) {
			fmt.Printf("JWKS refresh error: %v\n", err)
		},
	})
	return err
}

// ParseJWT validates the token and returns userId and role.
func ParseJWT(tokenString string) ([]string, string, error) {
	if jwks == nil {
		log.Println("JWKS not initialized")
		return nil, "", errors.New("JWKS not initialized")
	}

	claims := &CustomClaims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, jwks.Keyfunc)
	if err != nil || !token.Valid {
		log.Println("Invalid or expired token")
		return nil, "", errors.New("invalid or expired token")
	}

	if claims.ExpiresAt != nil && claims.ExpiresAt.Time.Before(time.Now()) {
		log.Println("Token expired")
		return nil, "", errors.New("token expired")
	}

	log.Println("Roles and UserId found: ", claims.Roles, "------", claims.Subject)

	return claims.Roles, claims.Subject, nil
}
