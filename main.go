package main

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"material-api/auth"
	"material-api/db"
	"material-api/email"
	"material-api/utils"
)

func main() {

	// Load environment variables
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	// Read DEBUG environment variable
	debugMode := os.Getenv("DEBUG") == "true"
	if debugMode {
		log.Println("INFO: Running in DEBUG mode. Authentication and Backend API checks will be disabled.")
	} else {
		log.Println("INFO: Running in PRODUCTION mode. Authentication and Backend API checks are ENABLED.")
	}

	// Initialize JWKS once
	jwksURL := os.Getenv("AUTH0_JWKS_URL")
	if err := auth.InitializeJWKS(jwksURL); err != nil {
		log.Fatalf("Failed to initialize JWKS: %v", err)
	}

	if err := db.InitDB(); err != nil {
        log.Fatalf("Failed to initialize database: %v", err)
    }

	// Initialize email service
	if err := email.InitEmailService(); err != nil {
		log.Printf("Warning: Email service initialization failed: %v", err)
		log.Println("Email functionality will be disabled")
	} else {
		log.Println("Email service initialized successfully")
	}

	// Start exchange rate updater in a separate goroutine
	go utils.StartExchangeRateUpdater()
	
	// Set up the Gin router
	router := gin.Default()
	router.Use(auth.AuthMiddleware())

	router.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"*"}, // Change to your frontend URL
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization", "api-secert"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	router.Use(func(c *gin.Context) {
		fmt.Printf("[GIN] %s %s", c.Request.Method, c.Request.URL.Path)
		c.Next()
	})

	// Register routes
	RegisterRoutes(router)

	// Start the server
	port := os.Getenv("PORT")
	println("Server running on port " + port)
	router.Run(":" + port)
}
