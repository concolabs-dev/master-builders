package handlers

import (
	"context"
	"fmt"
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

	if query == "" {
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
					{"Category.SubSubcategory": bson.M{"$regex": query, "$options": "i"}},
				}},
			},
		}
	} else {
		filter = bson.M{
			"$or": []bson.M{
				{"Name": bson.M{"$regex": query, "$options": "i"}},
				{"Category.Category": bson.M{"$regex": query, "$options": "i"}},
				{"Category.Subcategory": bson.M{"$regex": query, "$options": "i"}},
				{"Category.SubSubcategory": bson.M{"$regex": query, "$options": "i"}},
			},
		}
	}

	var materials []model.Material
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cursor, err := db.Collection.Find(ctx, filter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}
	defer cursor.Close(ctx)

	for cursor.Next(ctx) {
		var material model.Material
		if err := cursor.Decode(&material); err != nil {
			continue // Skip problematic entries
		}
		materials = append(materials, material)
	}

	if len(materials) == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "No materials found"})
		return
	}

	c.JSON(http.StatusOK, materials)
}

// Get all materials
func GetMaterials(c *gin.Context) {
	var materials []model.Material
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cursor, err := db.Collection.Find(ctx, bson.M{})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}
	defer cursor.Close(ctx)
	er := 0
	for cursor.Next(ctx) {
		var material model.Material

		if err := cursor.Decode(&material); err != nil {
			// c.JSON(http.StatusInternalServerError, gin.H{"error": "Error decoding"})
			er += 1
			continue

			return
		}
		// if err := cursor.Decode(&material); err != nil {
		// 	log.Println("Error decoding material:", err) // Log the actual error
		// 	continue // Skip problematic entry instead of stopping everything
		// }

		// // Handle NaN values in Unit
		// if math.IsNaN(material.Unit) {
		// 	material.Unit = "" // Default value for NaN
		// }
		// fmt.Println(material)
		materials = append(materials, material)
	}
	fmt.Println(er)

	c.JSON(http.StatusOK, materials)
}

// Get material by ID
func GetMaterialByID(c *gin.Context) {
	id, err := primitive.ObjectIDFromHex(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID"})
		return
	}

	var material model.Material
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	err = db.Collection.FindOne(ctx, bson.M{"_id": id}).Decode(&material)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Material not found"})
		return
	}

	c.JSON(http.StatusOK, material)
}

// Get materials by category, subcategory, or sub-subcategory
func GetMaterialsByCategory(c *gin.Context) {
	category := c.Query("category")             // Required
	subcategory := c.Query("subcategory")       // Optional
	subSubcategory := c.Query("subsubcategory") // Optional

	if category == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Category is required"})
		return
	}

	// Build the query dynamically
	filter := bson.M{"Category.Category": category}

	if subcategory != "" {
		filter["Category.Subcategory"] = subcategory
	}

	if subSubcategory != "" {
		filter["Category.SubSubcategory"] = subSubcategory
	}

	var materials []model.Material
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cursor, err := db.Collection.Find(ctx, filter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}
	defer cursor.Close(ctx)

	for cursor.Next(ctx) {
		var material model.Material
		if err := cursor.Decode(&material); err != nil {
			continue // Skip problematic entries
		}
		materials = append(materials, material)
	}

	if len(materials) == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "No materials found"})
		return
	}

	c.JSON(http.StatusOK, materials)
}

// Create a new material
func CreateMaterial(c *gin.Context) {
	var material model.Material
	if err := c.ShouldBindJSON(&material); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid JSON"})
		return
	}

	material.ID = primitive.NewObjectID()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, err := db.Collection.InsertOne(ctx, material)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not insert material"})
		return
	}

	c.JSON(http.StatusCreated, material)
}

// Update an existing material
// Update an existing material by its "Number" field
func UpdateMaterial(c *gin.Context) {
	numberParam := c.Param("number")
	if numberParam == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid material Number"})
		return
	}

	var updateData map[string]interface{}
	if err := c.ShouldBindJSON(&updateData); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid JSON"})
		return
	}

	if len(updateData) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No fields to update"})
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	update := bson.M{"$set": updateData}
	_, err := db.Collection.UpdateOne(ctx, bson.M{"Number": numberParam}, update)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not update material"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Material updated successfully"})
}

// Delete a material
func DeleteMaterial(c *gin.Context) {
	id, err := primitive.ObjectIDFromHex(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID"})
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, err = db.Collection.DeleteOne(ctx, bson.M{"_id": id})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not delete material"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Material deleted successfully"})
}