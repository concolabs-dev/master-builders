package handlers

import (
	"context"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"

	"material-api/db"
	"material-api/model"
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
	log.Printf("[INFO] PUT /types/%s - updating type", id)

	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		log.Printf("[WARN] Invalid ObjectID: %v\n", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID"})
		return
	}

	var updateData model.Type
	if err := c.BindJSON(&updateData); err != nil {
		log.Printf("[WARN] Invalid request data: %v\n", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	update := bson.M{"$set": updateData}
	result, err := db.TypeCollection.UpdateOne(ctx, bson.M{"_id": objID}, update)
	if err != nil {
		log.Printf("[ERROR] Failed to update type: %v\n", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update type"})
		return
	}

	log.Printf("[INFO] Type update result: matched=%d modified=%d\n", result.MatchedCount, result.ModifiedCount)
	c.JSON(http.StatusOK, gin.H{"message": "Type updated successfully"})
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
