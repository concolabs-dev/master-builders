package main

import (
	"context"
	"encoding/json"
	"log"
	"os"
	"strings"
	"time"

	"net/http"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"material-api/auth"
	"material-api/db"
	"material-api/handlers"
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

	jwksURL := os.Getenv("AUT0_JWKS_URL")
	// Initialize JWKS once
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
	// testMaterials()
	go startExchangeRateUpdater()
	// Set up the Gin router

	router := gin.Default()
	router.Use(AuthMiddleware())

	// if !debugMode {
	// 	router.Use(apiSecretMiddleware())
	// }

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

	router.POST("/materials", auth.RequireRoles("admin"), createMaterial)
	router.PUT("/materials/:number", auth.RequireRoles("admin"), updateMaterial)
	router.DELETE("/materials/:id", auth.RequireRoles("admin"), deleteMaterial)
	// Routes for handling types
	router.GET("/types", GetTypes)
	router.GET("/types/:id", GetTypeByID)
	router.POST("/types", auth.RequireRoles("admin"), CreateType)
	router.PUT("/types/:id", auth.RequireRoles("admin"), UpdateType)
	router.DELETE("/types/:id", auth.RequireRoles("admin"), DeleteType)
	router.GET("/forex", getMajorCurrencies)
	// Suppliers endpoints.
	router.POST("/suppliers", auth.RequireOwnership("suppplier"), createSupplier)
	router.GET("/suppliers", getSuppliers)
	router.GET("/suppliers/:id", getSupplierByID)
	router.GET("/suppliers/pid/:pid", getSupplierByPID)
	router.GET("/suppliers/pid/napproved/:pid", getSupplierByPPID)
	router.GET("/suppliers/email/:email", getSupplierByEmail)
	router.PUT("/suppliers/:id", auth.RequireOwnership("suppplier"), updateSupplier)
	router.DELETE("/suppliers/:id", auth.RequireOwnership("suppplier"), deleteSupplier)

	//items routes
	router.GET("/items", getItems)
	router.GET("/items/supplier/:supplierPid", getItemsBySupplier)
	router.GET("/items/material/:materialId", getItemsByMaterial)
	router.POST("/items", auth.RequireOwnership("item"), createItem)
	router.PUT("/items/:id", auth.RequireOwnership("item"), updateItem)
	router.DELETE("/items/:id", auth.RequireOwnership("item"), deleteItem)

	router.POST("/paymentRecords", auth.RequireOwnership("paymentRecord"), createPaymentRecord)
	router.GET("/paymentRecords", getPaymentRecords)
	router.GET("/paymentRecords/:id", getPaymentRecordByID)
	router.PUT("/paymentRecords/:id", auth.RequireOwnership("paymentRecord"), updatePaymentRecord)
	router.DELETE("/paymentRecords/:id", auth.RequireOwnership("paymentRecord"), deletePaymentRecord)

	// router.GET("/items/material/:materialId", getItemsByMaterialID)
	handlers.InitProfessionalCollections(client.Database(dbName))

	router.POST("/professionals", handlers.CreateProfessional)
	router.GET("/professionals", handlers.GetProfessionals)
	router.GET("/professionals/:id", handlers.GetProfessionalByID)
	router.GET("/professionals/pid/:pid", handlers.GetProfessionalByPID)
	router.GET("/professionals/email/:email", handlers.GetProfessionalByEmail)
	router.PUT("/professionals/:id", handlers.UpdateProfessional)
	router.DELETE("/professionals/:id", handlers.DeleteProfessional)
	router.GET("/admin/professionals/all", handlers.GetAllProfessionals)

	// Start the server
	port := os.Getenv("PORT")
	println("Server running on port " + port)
	router.Run(":" + port)
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
	_, err = db.ExchangeRateCollection.InsertOne(ctx, map[string]interface{}{
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

func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader != "" && strings.HasPrefix(authHeader, "Bearer ") {
			token := strings.TrimPrefix(authHeader, "Bearer ")

			roles, userID, err := auth.ParseJWT(token)
			if err != nil {
				// Invalid token
				c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "validation failed"})
				return
			}

			// Valid token: set values
			c.Set("userID", userID)
			c.Set("roles", roles)
		}

		// No token — proceed (public access)
		c.Next()
	}
}
