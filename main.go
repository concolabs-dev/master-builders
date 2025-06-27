package main

import (
	"context"
	"encoding/json"
	"log"
	"os"
	"time"

	"net/http"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"material-api/email"
	"material-api/handlers"
)

var (
	collection              *mongo.Collection
	typeCollection          *mongo.Collection
	exchangeRateCollection  *mongo.Collection
	supplierCollection      *mongo.Collection
	itemCollection          *mongo.Collection
	paymentRecordCollection *mongo.Collection
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
	collection = client.Database(dbName).Collection(collectionName)
	typeCollection = client.Database(dbName).Collection(typeCollectionName)
	exchangeRateCollection = client.Database(dbName).Collection("rates")
	supplierCollection = client.Database(os.Getenv("DB_NAME")).Collection("suppliers")
	itemCollection = client.Database(os.Getenv("DB_NAME")).Collection("items")
	paymentRecordCollection = client.Database(os.Getenv("DB_NAME")).Collection("payments")
	// testMaterials()
	go startExchangeRateUpdater()
	// Set up the Gin router

	router := gin.Default()

	if !debugMode {
		router.Use(apiSecretMiddleware())
	}

	router.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"*"}, // Change to your frontend URL
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization", "api-secert"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))
	router.GET("/search", searchMaterials)
	router.GET("/materials", getMaterials)
	router.GET("/materials/:id", getMaterialByID)
	router.GET("/materials/filter", getMaterialsByCategory)

	router.POST("/materials", createMaterial)
	router.PUT("/materials/:number", updateMaterial)
	router.DELETE("/materials/:id", deleteMaterial)
	// Routes for handling types
	router.GET("/types", GetTypes)
	router.GET("/types/:id", GetTypeByID)
	router.POST("/types", CreateType)
	router.PUT("/types/:id", UpdateType)
	router.DELETE("/types/:id", DeleteType)
	router.GET("/forex", getMajorCurrencies)
	// Suppliers endpoints.
	router.POST("/suppliers", createSupplier)
	router.GET("/suppliers", getSuppliers)
	router.GET("/suppliers/:id", getSupplierByID)
	router.GET("/suppliers/pid/:pid", getSupplierByPID)
	router.GET("/suppliers/pid/napproved/:pid", getSupplierByPPID)
	router.GET("/suppliers/email/:email", getSupplierByEmail)
	router.PUT("/suppliers/:id", updateSupplier)
	router.DELETE("/suppliers/:id", deleteSupplier)

	//items routes
	router.GET("/items", getItems)
	router.GET("/items/supplier/:supplierPid", getItemsBySupplier)
	router.GET("/items/material/:materialId", getItemsByMaterial)
	router.POST("/items", createItem)
	router.PUT("/items/:id", updateItem)
	router.DELETE("/items/:id", deleteItem)

	router.POST("/paymentRecords", createPaymentRecord)
	router.GET("/paymentRecords", getPaymentRecords)
	router.GET("/paymentRecords/:id", getPaymentRecordByID)
	router.PUT("/paymentRecords/:id", updatePaymentRecord)
	router.DELETE("/paymentRecords/:id", deletePaymentRecord)

	// router.GET("/items/material/:materialId", getItemsByMaterialID)
	handlers.InitProfessionalCollections(client.Database(dbName))

	router.POST("/professionals", handlers.CreateProfessional)
	router.GET("/professionals", handlers.GetAllProfessionals)
	router.GET("/professionals/:id", handlers.GetProfessionalByID)
	router.GET("/professionals/pid/:pid", handlers.GetProfessionalByPID)
	router.GET("/professionals/email/:email", handlers.GetProfessionalByEmail)
	router.PUT("/professionals/:id", handlers.UpdateProfessional)
	router.DELETE("/professionals/:id", handlers.DeleteProfessional)
	router.GET("/admin/professionals/all", handlers.GetAllProfessionals)

	router.POST("/projects", handlers.CreateProject)                     // Create a new project
	router.GET("/projects", handlers.GetProjects)                        // Get all projects
	router.GET("/projects/professional/:pid", handlers.GetProjectsByPID) // Get projects by professional PID
	router.GET("/projects/:id", handlers.GetProjectByID)                 // Get a project by ID
	router.PUT("/projects/:id", handlers.UpdateProject)                  // Update a project
	router.DELETE("/projects/:id", handlers.DeleteProject)               // Delete a project
	email.RegisterRoutes(router)
	// Start the server
	port := os.Getenv("PORT")
	println("Server running on port " + port)
	router.Run(":" + port)
}

// New middleware to check for the API secret header.
func apiSecretMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		expectedSecret := os.Getenv("BACKEND_API_SECRET")
		providedSecret := c.GetHeader("api-secert")
		if providedSecret == "" || providedSecret != expectedSecret {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
			return
		}
		c.Next()
	}
}

func startExchangeRateUpdater() {
	ticker := time.NewTicker(24 * time.Hour)
	defer ticker.Stop()

	for {
		updateExchangeRates()
		<-ticker.C
	}
}

type ExchangeRateResponse struct {
	Result             string             `json:"result"`
	TimeLastUpdateUnix int64              `json:"time_last_update_unix"`
	BaseCode           string             `json:"base_code"`
	ConversionRates    map[string]float64 `json:"conversion_rates"`
}

func updateExchangeRates() {
	// apiKey := os.Getenv("EXCHANGE_RATE_API_KEY")
	url := "https://v6.exchangerate-api.com/v6/9b03f47e0242509680dac5fc/latest/LKR"
	resp, err := http.Get(url)
	if err != nil {
		log.Println("Error fetching exchange rates:", err)
		return
	}
	defer resp.Body.Close()

	var result ExchangeRateResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		log.Println("Error decoding exchange rate response:", err)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Insert the data into MongoDB
	_, err = exchangeRateCollection.InsertOne(ctx, map[string]interface{}{
		"timestamp":             time.Now(),
		"time_last_update_unix": result.TimeLastUpdateUnix,
		"base_code":             result.BaseCode,
		"conversion_rates":      result.ConversionRates,
	})
	if err != nil {
		log.Println("Error inserting exchange rates into MongoDB:", err)
	} else {
		log.Println("Exchange rates updated successfully")
	}
}
