package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

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

	// Connect to MongoDB
	mongoURI := os.Getenv("MONGO_URI")
	client, err := mongo.NewClient(options.Client().ApplyURI(mongoURI))
	if err != nil {
		log.Fatal(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	err = client.Connect(ctx)
	if err != nil {
		log.Fatal(err)
	}

	// Initialize email service
	if err := email.InitEmailService(); err != nil {
		log.Printf("Warning: Email service initialization failed: %v", err)
		log.Println("Email functionality will be disabled")
	} else {
		log.Println("Email service initialized successfully")
	}

	// Select database & collection
	dbName := os.Getenv("DB_NAME")
	collectionName := os.Getenv("COLLECTION_NAME1")
	typeCollectionName := os.Getenv("COLLECTION_NAME2")
	db.Collection = client.Database(dbName).Collection(collectionName)
	db.TypeCollection = client.Database(dbName).Collection(typeCollectionName)
	db.ExchangeRateCollection = client.Database(dbName).Collection("rates")
	db.SupplierCollection = client.Database(os.Getenv("DB_NAME")).Collection("suppliers")
	db.ItemCollection = client.Database(os.Getenv("DB_NAME")).Collection("items")
	db.PaymentRecordCollection = client.Database(os.Getenv("DB_NAME")).Collection("payments")
	db.ProfessionalCollection = client.Database(dbName).Collection("professionals")
	db.ProfessionalPaymentRecordCollection = client.Database(dbName).Collection("professional_payment_records")
	db.ProjectCollection = client.Database(dbName).Collection("projects")

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
