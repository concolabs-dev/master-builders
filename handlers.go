package main

import (
	"context"
	"net/http"
	"time"
	"fmt"
	"log"
	// "math"
	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)
func searchMaterials(c *gin.Context) {
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

    var materials []Material
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()

    cursor, err := collection.Find(ctx, filter)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
        return
    }
    defer cursor.Close(ctx)

    for cursor.Next(ctx) {
        var material Material
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
func getMaterials(c *gin.Context) {
	var materials []Material
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cursor, err := collection.Find(ctx, bson.M{})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}
	defer cursor.Close(ctx)
	er := 0
	for cursor.Next(ctx) {
		var material Material

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
func getMaterialByID(c *gin.Context) {
	id, err := primitive.ObjectIDFromHex(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID"})
		return
	}

	var material Material
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	err = collection.FindOne(ctx, bson.M{"_id": id}).Decode(&material)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Material not found"})
		return
	}

	c.JSON(http.StatusOK, material)
}
// Get materials by category, subcategory, or sub-subcategory
func getMaterialsByCategory(c *gin.Context) {
	category := c.Query("category")         // Required
	subcategory := c.Query("subcategory")   // Optional
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

	var materials []Material
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cursor, err := collection.Find(ctx, filter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}
	defer cursor.Close(ctx)

	for cursor.Next(ctx) {
		var material Material
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
func createMaterial(c *gin.Context) {
	var material Material
	if err := c.ShouldBindJSON(&material); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid JSON"})
		return
	}

	material.ID = primitive.NewObjectID()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, err := collection.InsertOne(ctx, material)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not insert material"})
		return
	}

	c.JSON(http.StatusCreated, material)
}

// Update an existing material
// Update an existing material by its "Number" field
func updateMaterial(c *gin.Context) {
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
    _, err := collection.UpdateOne(ctx, bson.M{"Number": numberParam}, update)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not update material"})
        return
    }

    c.JSON(http.StatusOK, gin.H{"message": "Material updated successfully"})
}


// Delete a material
func deleteMaterial(c *gin.Context) {
	id, err := primitive.ObjectIDFromHex(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID"})
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, err = collection.DeleteOne(ctx, bson.M{"_id": id})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not delete material"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Material deleted successfully"})
}



func GetTypes(c *gin.Context) {
	var types []Type
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cursor, err := typeCollection.Find(ctx, bson.M{})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}
	defer cursor.Close(ctx)
	for cursor.Next(ctx) {
		var t Type
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

	var t Type
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err = typeCollection.FindOne(ctx, bson.M{"_id": objID}).Decode(&t)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Type not found"})
		return
	}

	c.JSON(http.StatusOK, t)
}

// Create a new type
func CreateType(c *gin.Context) {
	var t Type
	if err := c.BindJSON(&t); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	t.ID = primitive.NewObjectID()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := typeCollection.InsertOne(ctx, t)
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

	var updateData Type
	if err := c.BindJSON(&updateData); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	update := bson.M{"$set": updateData}
	_, err = typeCollection.UpdateOne(ctx, bson.M{"_id": objID}, update)
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

	_, err = typeCollection.DeleteOne(ctx, bson.M{"_id": objID})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete type"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Type deleted successfully"})
}