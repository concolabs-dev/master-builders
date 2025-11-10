package handlers

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"

	"material-api/db"
	"material-api/model"
	"material-api/utils"
)

func GetTypes(c *gin.Context) {
	start := time.Now()
	log.Println("[INFO] GET /types - fetching all types")

	var types []model.Type
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cursor, err := db.TypeCollection.Find(ctx, bson.M{})
	if err != nil {
		log.Printf("[ERROR] Database query failed: %v\n", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}
	defer cursor.Close(ctx)

	for cursor.Next(ctx) {
		var t model.Type
		if err := cursor.Decode(&t); err != nil {
			rawData := cursor.Current
			log.Printf("[ERROR] Failed to decode type document: %v, raw data: %v\n", err, rawData)
			continue
		}
		types = append(types, t)
	}

	log.Printf("[INFO] Successfully fetched %d types in %v\n", len(types), time.Since(start))
	c.JSON(http.StatusOK, types)
}

// Get a single type by ID
func GetTypeByID(c *gin.Context) {
	id := c.Param("id")
	log.Printf("[INFO] GET /types/%s - fetching type by ID", id)

	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		log.Printf("[WARN] Invalid ObjectID: %v\n", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID"})
		return
	}

	var t model.Type
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err = db.TypeCollection.FindOne(ctx, bson.M{"_id": objID}).Decode(&t)
	if err != nil {
		log.Printf("[WARN] Type not found: %v\n", err)
		c.JSON(http.StatusNotFound, gin.H{"error": "Type not found"})
		return
	}

	log.Printf("[INFO] Type retrieved successfully: %+v\n", t)
	c.JSON(http.StatusOK, t)
}

// Create a new type
func CreateType(c *gin.Context) {
	log.Println("[INFO] POST /types - creating new type")

	var t model.Type
	if err := c.BindJSON(&t); err != nil {
		log.Printf("[WARN] Invalid request body: %v\n", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	t.ID = primitive.NewObjectID()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	result, err := db.TypeCollection.InsertOne(ctx, t)
	if err != nil {
		log.Printf("[ERROR] Failed to insert type: %v\n", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create type"})
		return
	}

	log.Printf("[INFO] Type created successfully with ID %v\n", result.InsertedID)
	c.JSON(http.StatusCreated, t)
}

// Update a type by ID

func UpdateType(c *gin.Context) {
	id := c.Param("id")
	log.Printf("[INFO] PUT /types/%s - starting transactional update", id)

	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		log.Printf("[WARN] Invalid ObjectID: %v\n", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID"})
		return
	}

	// 1. Bind the incoming "new" data
	var newTypeData model.Type
	if err := c.BindJSON(&newTypeData); err != nil {
		log.Printf("[WARN] Invalid request data: %v\n", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	// 2. Start a new MongoDB Session for the transaction
	session, err := db.Client.StartSession() // Assumes db.Client is your mongo.Client
	if err != nil {
		log.Printf("[ERROR] Failed to start transaction session: %v\n", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to start transaction"})
		return
	}
	defer session.EndSession(context.Background())

	// 3. Define the transaction logic
	callback := func(sessCtx mongo.SessionContext) (interface{}, error) {
		var oldTypeData model.Type

		// --- A: Get the OLD Type document ---
		// We must do this *inside* the transaction to get a consistent read.
		err := db.TypeCollection.FindOne(sessCtx, bson.M{"_id": objID}).Decode(&oldTypeData)
		if err != nil {
			log.Printf("[ERROR-TXN] Failed to find old type data: %v\n", err)
			return nil, fmt.Errorf("could not find original type document: %w", err)
		}

		// --- B: Update the Type document itself ---
		// We use $set to replace the document with the new data
		update := bson.M{"$set": newTypeData}
		_, err = db.TypeCollection.UpdateOne(sessCtx, bson.M{"_id": objID}, update)
		if err != nil {
			log.Printf("[ERROR-TXN] Failed to update type document: %v\n", err)
			return nil, fmt.Errorf("could not update type document: %w", err)
		}

		// --- C: Find changes and update related Materials ---
		// This is the "category change logic"
		// We find all changes by comparing the old and new category lists.
		changes := utils.FindCategoryChanges(oldTypeData.Categories, newTypeData.Categories)
		log.Printf("[DEBUG] changes: %v\n", changes)
		if len(changes) == 0 {
			log.Printf("[INFO-TXN] No category name changes detected.")
			return "Update successful, no materials migrated.", nil
		}

		log.Printf("[INFO-TXN] Found %d category changes to migrate...", len(changes))

		// For each change, update all matching materials in db.Collection
		for _, change := range changes {
			// This filter finds materials matching the OLD category data
			filter := bson.M{
				"Category.Category":        change.Old.Category,
				"Category.Subcategory":     change.Old.Subcategory,
				"Category.Sub subcategory": change.Old.SubSubcategory,
			}

			// This update $sets the NEW category data
			materialUpdate := bson.M{
				"$set": bson.M{
					"Category.Category":        change.New.Category,
					"Category.Subcategory":     change.New.Subcategory,
					"Category.Sub subcategory": change.New.SubSubcategory,
				},
			}

			// Run the UpdateMany for this change
			result, err := db.Collection.UpdateMany(sessCtx, filter, materialUpdate)
			if err != nil {
				log.Printf("[ERROR-TXN] Failed to migrate materials: %v\n", err)
				return nil, fmt.Errorf("failed to migrate materials for category '%s': %w", change.Old.Category, err)
			}
			log.Printf("[INFO-TXN] Migrated %d materials for category change: %s -> %s", result.ModifiedCount, change.Old.Category, change.New.Category)
		}

		return "Type updated and materials migrated successfully", nil
	}

	// 4. Run the transaction
	result, err := session.WithTransaction(context.Background(), callback)
	if err != nil {
		log.Printf("[ERROR] Transaction failed and was rolled back: %v\n", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update: " + err.Error()})
		return
	}

	log.Printf("[INFO] Transaction successful: %v", result)
	c.JSON(http.StatusOK, gin.H{"message": result})
}

// Delete a type by ID
func DeleteType(c *gin.Context) {
	id := c.Param("id")
	log.Printf("[INFO] DELETE /types/%s - deleting type", id)

	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		log.Printf("[WARN] Invalid ObjectID: %v\n", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID"})
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	result, err := db.TypeCollection.DeleteOne(ctx, bson.M{"_id": objID})
	if err != nil {
		log.Printf("[ERROR] Failed to delete type: %v\n", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete type"})
		return
	}

	log.Printf("[INFO] Type deletion result: deletedCount=%d\n", result.DeletedCount)
	c.JSON(http.StatusOK, gin.H{"message": "Type deleted successfully"})
}