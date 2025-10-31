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
	var types []model.Type
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cursor, err := db.TypeCollection.Find(ctx, bson.M{})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}
	defer cursor.Close(ctx)
	for cursor.Next(ctx) {
		var t model.Type
		err := cursor.Decode(&t)
		if err != nil {
			// Log the error along with the raw data causing the issue
			rawData := cursor.Current
			log.Printf("Error decoding type: %v, raw data: %v\n", err, rawData)
			continue
		}
		types = append(types, t)
	}

	c.JSON(http.StatusOK, types)
}

// Get a single type by ID
func GetTypeByID(c *gin.Context) {
	id := c.Param("id")
	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID"})
		return
	}

	var t model.Type
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err = db.TypeCollection.FindOne(ctx, bson.M{"_id": objID}).Decode(&t)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Type not found"})
		return
	}

	c.JSON(http.StatusOK, t)
}

// Create a new type
func CreateType(c *gin.Context) {
	var t model.Type
	if err := c.BindJSON(&t); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	t.ID = primitive.NewObjectID()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := db.TypeCollection.InsertOne(ctx, t)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create type"})
		return
	}

	c.JSON(http.StatusCreated, t)
}

// Update a type by ID
func UpdateType(c *gin.Context) {
	id := c.Param("id")
	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID"})
		return
	}

	var updateData model.Type
	if err := c.BindJSON(&updateData); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	update := bson.M{"$set": updateData}
	_, err = db.TypeCollection.UpdateOne(ctx, bson.M{"_id": objID}, update)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update type"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Type updated successfully"})
}

// Delete a type by ID
func DeleteType(c *gin.Context) {
	id := c.Param("id")
	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID"})
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err = db.TypeCollection.DeleteOne(ctx, bson.M{"_id": objID})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete type"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Type deleted successfully"})
}