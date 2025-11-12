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

	"material-api/db"
	"material-api/model"
)

func SearchMaterials(c *gin.Context) {
	query := c.Query("q")
	subcategory := c.Query("subcategory")
	log.Printf("[INFO] SearchMaterials called: q='%s', subcategory='%s'", query, subcategory)

	if query == "" {
		log.Println("[WARN] SearchMaterials: Search query is required")
		c.JSON(http.StatusBadRequest, gin.H{"error": "Search query is required"})
		return
	}

	var filter bson.M

	if subcategory != "" {
		filter = bson.M{
			"$and": []bson.M{
				{"Category.Subcategory": subcategory},
				{"$or": []bson.M{
					{"Name": bson.M{"$regex": query, "$options": "i"}},
					{"Category.Sub subcategory": bson.M{"$regex": query, "$options": "i"}},
				}},
			},
		}
	} else {
		filter = bson.M{
			"$or": []bson.M{
				{"Name": bson.M{"$regex": query, "$options": "i"}},
				{"Category.Category": bson.M{"$regex": query, "$options": "i"}},
				{"Category.Subcategory": bson.M{"$regex": query, "$options": "i"}},
				{"Category.Sub subcategory": bson.M{"$regex": query, "$options": "i"}},
			},
		}
	}
	log.Printf("[DEBUG] SearchMaterials: Using filter: %v", filter)

	var materials []model.Material
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cursor, err := db.MaterialCollection.Find(ctx, filter)
	if err != nil {
		log.Printf("[ERROR] SearchMaterials: Database error: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}
	defer cursor.Close(ctx)

	for cursor.Next(ctx) {
		var material model.Material
		if err := cursor.Decode(&material); err != nil {
			log.Printf("[WARN] SearchMaterials: Failed to decode material: %v", err)
			continue // Skip problematic entries
		}
		materials = append(materials, material)
	}

	if len(materials) == 0 {
		log.Println("[INFO] SearchMaterials: No materials found")
		c.JSON(http.StatusNotFound, gin.H{"error": "No materials found"})
		return
	}

	log.Printf("[INFO] SearchMaterials: Returning %d materials", len(materials))
	c.JSON(http.StatusOK, materials)
}

// Get all materials
func GetMaterials(c *gin.Context) {
	log.Println("[INFO] GetMaterials called")
	var materials []model.Material
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cursor, err := db.MaterialCollection.Find(ctx, bson.M{})
	if err != nil {
		log.Printf("[ERROR] GetMaterials: Database error: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}
	defer cursor.Close(ctx)

	decodeErrors := 0
	for cursor.Next(ctx) {
		var material model.Material

		if err := cursor.Decode(&material); err != nil {
			log.Printf("[WARN] GetMaterials: Failed to decode material: %v", err)
			decodeErrors++
			continue
			// The 'return' here was unreachable code, so I removed it.
		}
		materials = append(materials, material)
	}

	if decodeErrors > 0 {
		log.Printf("[WARN] GetMaterials: Skipped %d materials due to decoding errors", decodeErrors)
	}
	log.Printf("[INFO] GetMaterials: Returning %d materials", len(materials))
	c.JSON(http.StatusOK, materials)
}

// Get material by ID
func GetMaterialByID(c *gin.Context) {
	idParam := c.Param("id")
	log.Printf("[INFO] GetMaterialByID called for id: %s", idParam)

	id, err := primitive.ObjectIDFromHex(idParam)
	if err != nil {
		log.Printf("[WARN] GetMaterialByID: Invalid ID format: %s", idParam)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID"})
		return
	}

	var material model.Material
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	log.Printf("[DEBUG] GetMaterialByID: Finding document with _id: %s", id)
	err = db.MaterialCollection.FindOne(ctx, bson.M{"_id": id}).Decode(&material)
	if err != nil {
		log.Printf("[WARN] GetMaterialByID: Material not found for _id: %s, error: %v", id, err)
		c.JSON(http.StatusNotFound, gin.H{"error": "Material not found"})
		return
	}

	log.Printf("[INFO] GetMaterialByID: Successfully found material %s", id)
	c.JSON(http.StatusOK, material)
}

// Get materials by category, subcategory, or sub-subcategory
func GetMaterialsByCategory(c *gin.Context) {
	category := c.Query("category")
	subcategory := c.Query("subcategory")
	subSubcategory := c.Query("subSubcategory")
	log.Printf("[INFO] GetMaterialsByCategory called: category='%s', subcategory='%s', subSubcategory='%s'", category, subcategory, subSubcategory)

	if category == "" {
		log.Println("[WARN] GetMaterialsByCategory: 'category' is required")
		c.JSON(http.StatusBadRequest, gin.H{"error": "Category is required"})
		return
	}

	// Build the query dynamically
	filter := bson.M{"Category.Category": category}

	if subcategory != "" {
		filter["Category.Subcategory"] = subcategory
	}

	if subSubcategory != "" {
		filter["Category.Sub subcategory"] = subSubcategory
	}
	log.Printf("[DEBUG] GetMaterialsByCategory: Using filter: %v", filter)

	var materials []model.Material
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cursor, err := db.MaterialCollection.Find(ctx, filter)
	if err != nil {
		log.Printf("[ERROR] GetMaterialsByCategory: Database error: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}
	defer cursor.Close(ctx)

	for cursor.Next(ctx) {
		var material model.Material
		if err := cursor.Decode(&material); err != nil {
			log.Printf("[WARN] GetMaterialsByCategory: Failed to decode material: %v", err)
			continue // Skip problematic entries
		}
		materials = append(materials, material)
	}

	if len(materials) == 0 {
		log.Println("[INFO] GetMaterialsByCategory: No materials found for filter")
		c.JSON(http.StatusNotFound, gin.H{"error": "No materials found"})
		return
	}

	log.Printf("[INFO] GetMaterialsByCategory: Returning %d materials", len(materials))
	c.JSON(http.StatusOK, materials)
}

// Create a new material
func CreateMaterial(c *gin.Context) {
	log.Println("[INFO] CreateMaterial called")
	var material model.Material
	if err := c.ShouldBindJSON(&material); err != nil {
		log.Printf("[WARN] CreateMaterial: Invalid JSON payload: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid JSON"})
		return
	}

	material.ID = primitive.NewObjectID()
	log.Printf("[DEBUG] CreateMaterial: Attempting to insert new material with ID: %s", material.ID.Hex())

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, err := db.MaterialCollection.InsertOne(ctx, material)
	if err != nil {
		log.Printf("[ERROR] CreateMaterial: Could not insert material: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not insert material"})
		return
	}

	log.Printf("[INFO] CreateMaterial: Successfully created material: %s", material.ID.Hex())
	c.JSON(http.StatusCreated, material)
}

// Update an existing material by its "Number" field
func UpdateMaterial(c *gin.Context) {
	numberParam := c.Param("number")
	log.Printf("[INFO] UpdateMaterial called for Number: %s", numberParam)

	if numberParam == "" {
		log.Println("[WARN] UpdateMaterial: Invalid material Number (empty)")
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid material Number"})
		return
	}

	var updateData map[string]interface{}
	if err := c.ShouldBindJSON(&updateData); err != nil {
		log.Printf("[WARN] UpdateMaterial: Invalid JSON payload: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid JSON"})
		return
	}

	if len(updateData) == 0 {
		log.Println("[WARN] UpdateMaterial: No fields to update")
		c.JSON(http.StatusBadRequest, gin.H{"error": "No fields to update"})
		return
	}
	log.Printf("[DEBUG] UpdateMaterial: Updating %d fields for Number: %s", len(updateData), numberParam)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	update := bson.M{"$set": updateData}

	result, err := db.MaterialCollection.UpdateOne(ctx, bson.M{"Number": numberParam}, update)
	if err != nil {
		log.Printf("[ERROR] UpdateMaterial: Could not update material: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not update material"})
		return
	}

	if result.MatchedCount == 0 {
		log.Printf("[WARN] UpdateMaterial: No material found with Number: %s", numberParam)
		c.JSON(http.StatusNotFound, gin.H{"error": "Material not found"})
		return
	}

	log.Printf("[INFO] UpdateMaterial: Successfully updated material (Matched: %d, Modified: %d)", result.MatchedCount, result.ModifiedCount)
	c.JSON(http.StatusOK, gin.H{"message": "Material updated successfully"})
}

// Delete a material
func DeleteMaterial(c *gin.Context) {
	idParam := c.Param("id")
	log.Printf("[INFO] DeleteMaterial called for id: %s", idParam)

	id, err := primitive.ObjectIDFromHex(idParam)
	if err != nil {
		log.Printf("[WARN] DeleteMaterial: Invalid ID format: %s", idParam)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID"})
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	log.Printf("[DEBUG] DeleteMaterial: Deleting document with _id: %s", id)
	result, err := db.MaterialCollection.DeleteOne(ctx, bson.M{"_id": id})
	if err != nil {
		log.Printf("[ERROR] DeleteMaterial: Could not delete material: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not delete material"})
		return
	}

	if result.DeletedCount == 0 {
		log.Printf("[WARN] DeleteMaterial: No material found to delete with _id: %s", id)
		c.JSON(http.StatusNotFound, gin.H{"error": "Material not found"})
		return
	}

	log.Printf("[INFO] DeleteMaterial: Successfully deleted material: %s", id)
	c.JSON(http.StatusOK, gin.H{"message": "Material deleted successfully"})
}

// UpdateMaterialCategory is a service function, not a handler.
// It's called by other functions (e.g., UpdateType handler).
func UpdateMaterialCategory(ctx context.Context, oldCat model.Category, newCat model.Category) (int64, error) {
	log.Printf("[INFO] UpdateMaterialCategory called: %v -> %v", oldCat, newCat)

	// 1. Define the filter
	filter := bson.M{
		"Category.Category":       oldCat.Category,
		"Category.Subcategory":    oldCat.Subcategory,
		"Category.Sub subcategory": oldCat.SubSubcategory,
	}
	log.Printf("[DEBUG] UpdateMaterialCategory: Using filter: %v", filter)

	// 2. Define the update
	update := bson.M{
		"$set": bson.M{
			"Category.Category":       newCat.Category,
			"Category.Subcategory":    newCat.Subcategory,
			"Category.Sub subcategory": newCat.SubSubcategory,
		},
	}
	log.Printf("[DEBUG] UpdateMaterialCategory: Using update: %v", update)

	// 3. Set a timeout (using the provided context as a base)
	updateCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	// 4. Execute the UpdateMany operation
	result, err := db.MaterialCollection.UpdateMany(updateCtx, filter, update)
	if err != nil {
		log.Printf("[ERROR] UpdateMaterialCategory: UpdateMany failed: %v", err)
		return 0, fmt.Errorf("could not update material categories: %w", err)
	}

	// 5. Return the number of documents modified
	log.Printf("[INFO] UpdateMaterialCategory: Modified %d documents", result.ModifiedCount)
	return result.ModifiedCount, nil
}
