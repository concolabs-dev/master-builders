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

	jwksURL := os.Getenv("AUTH0_JWKS_URL")
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

	router.GET("/search", handlers.SearchMaterials)
	router.GET("/materials", handlers.GetMaterials)
	router.GET("/materials/:id", handlers.GetMaterialByID)
	router.GET("/materials/filter", handlers.GetMaterialsByCategory)

	router.POST("/materials", auth.RequireRoles("admin"), handlers.CreateMaterial)
	router.PUT("/materials/:number", auth.RequireRoles("admin"), handlers.UpdateMaterial)
	router.DELETE("/materials/:id", auth.RequireRoles("admin"), handlers.DeleteMaterial)
	// Routes for handling types
	router.GET("/types", handlers.GetTypes)
	router.GET("/types/:id", handlers.GetTypeByID)
	router.POST("/types", auth.RequireRoles("admin"), handlers.CreateType)
	router.PUT("/types/:id", auth.RequireRoles("admin"), handlers.UpdateType)
	router.DELETE("/types/:id", auth.RequireRoles("admin"), handlers.DeleteType)
	router.GET("/forex", handlers.GetMajorCurrencies)
	// Suppliers endpoints.
	router.POST("/suppliers", auth.RequireOwnership("supplier"), handlers.CreateSupplier)
	router.GET("/suppliers", handlers.GetSuppliers)
	router.GET("/suppliers/:id", handlers.GetSupplierByID)
	router.GET("/suppliers/pid/:pid", handlers.GetSupplierByPID)
	router.GET("/suppliers/pid/napproved/:pid", handlers.GetSupplierByPPID)
	router.GET("/suppliers/email/:email", handlers.GetSupplierByEmail)
	router.PUT("/suppliers/:id", auth.RequireOwnership("supplier"), handlers.UpdateSupplier)
	router.DELETE("/suppliers/:id", auth.RequireOwnership("supplier"), handlers.DeleteSupplier)

	//items routes
	router.GET("/items", handlers.GetItems)
	router.GET("/items/supplier/:supplierPid", handlers.GetItemsBySupplier)
	router.GET("/items/material/:materialId", handlers.GetItemsByMaterial)
	router.POST("/items", auth.RequireOwnership("item"), handlers.CreateItem)
	router.PUT("/items/:id", auth.RequireOwnership("item"), handlers.UpdateItem)
	router.DELETE("/items/:id", auth.RequireOwnership("item"), handlers.DeleteItem)

	router.POST("/paymentRecords", auth.RequireRoles("admin"), handlers.CreatePaymentRecord)
	router.GET("/paymentRecords", auth.RequireOwnership("paymentRecord"), handlers.GetPaymentRecords)
	router.GET("/paymentRecords/:id", auth.RequireOwnership("paymentRecord"), handlers.GetPaymentRecordByID)
	router.PUT("/paymentRecords/:id", auth.RequireRoles("admin"), handlers.UpdatePaymentRecord)
	router.DELETE("/paymentRecords/:id", auth.RequireRoles("admin"), handlers.DeletePaymentRecord)

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

			//log.Println("Token found: ", token)

			roles, userID, err := auth.ParseJWT(token)
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
