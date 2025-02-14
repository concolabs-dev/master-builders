package main

import (
	"context"
	"log"
	"os"
	"time"
	"github.com/gin-contrib/cors" 
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var collection *mongo.Collection
var typeCollection *mongo.Collection
func main() {
	// Load environment variables
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
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
	collection = client.Database(dbName).Collection(collectionName)
	typeCollection = client.Database(dbName).Collection(typeCollectionName)
	// testMaterials()
	// Set up the Gin router
	router := gin.Default()
	router.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"*"}, // Change to your frontend URL
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))
	router.GET("/materials", getMaterials)
	router.GET("/materials/:id", getMaterialByID)
	router.GET("/materials/filter", getMaterialsByCategory)
	

	router.POST("/materials", createMaterial)
	router.PUT("/materials/:id", updateMaterial)
	router.DELETE("/materials/:id", deleteMaterial)
	// Routes for handling types
	router.GET("/types", GetTypes)
	router.GET("/types/:id", GetTypeByID)
	router.POST("/types", CreateType)
	router.PUT("/types/:id", UpdateType)
	router.DELETE("/types/:id", DeleteType)
	// Start the server
	port := os.Getenv("PORT")
	println("Server running on port " + port)
	router.Run(":" + port)
}
