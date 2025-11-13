package db

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// Package-level variables to be accessed by handlers
var (
	Client                              *mongo.Client
	MaterialCollection                  *mongo.Collection
	TypeCollection                      *mongo.Collection
	ExchangeRateCollection              *mongo.Collection
	SupplierCollection                  *mongo.Collection
	ItemCollection                      *mongo.Collection
	PaymentRecordCollection             *mongo.Collection
	ProfessionalCollection              *mongo.Collection
	ProfessionalPaymentRecordCollection *mongo.Collection
	ProjectCollection                   *mongo.Collection
)

// InitDB initializes the MongoDB client and all collections.
// It returns an error if the connection fails.
func InitDB() error {
	// 1. Get Mongo URI from environment
	mongoURI := os.Getenv("MONGO_URI")
	if mongoURI == "" {
		return fmt.Errorf("MONGO_URI environment variable not set")
	}

	// 2. Create a new client
	client, err := mongo.NewClient(options.Client().ApplyURI(mongoURI))
	if err != nil {
		return fmt.Errorf("failed to create mongo client: %w", err)
	}

	// 3. Connect to MongoDB
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	err = client.Connect(ctx)
	if err != nil {
		return fmt.Errorf("failed to connect to mongo: %w", err)
	}

	// 4. Ping the database to verify connection
	err = client.Ping(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to ping mongo: %w", err)
	}

	log.Println("Successfully connected to MongoDB!")

	// 5. Assign the client and collections to package variables
	Client = client

	// Get database name from env, or use a default
	dbName := os.Getenv("DB_NAME")
	if dbName == "" {
		dbName = "materials_db" // Set a default database name
	}

	database := Client.Database(dbName)

	// Initialize all collections from your list
	MaterialCollection = database.Collection("materials")
	TypeCollection = database.Collection("types")
	ExchangeRateCollection = database.Collection("rates")
	SupplierCollection = database.Collection("suppliers")
	ItemCollection = database.Collection("items")
	PaymentRecordCollection = database.Collection("payments")
	ProfessionalCollection = database.Collection("professionals")
	ProfessionalPaymentRecordCollection = database.Collection("professional_payment_records")
	ProjectCollection = database.Collection("projects")

	log.Println("All database collections initialized.")
	return nil
}

// DisconnectDB disconnects the client (for graceful shutdown).
func DisconnectDB(ctx context.Context) {
	if Client != nil {
		log.Println("Disconnecting from MongoDB...")
		if err := Client.Disconnect(ctx); err != nil {
			log.Printf("Error disconnecting from MongoDB: %v", err)
		}
	}
}
