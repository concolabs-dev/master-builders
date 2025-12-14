package handlers

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"material-api/db"
	"material-api/model"
)

// createItem creates a new item.
func CreateItem(c *gin.Context) {
	var item model.Item
	if err := c.ShouldBindJSON(&item); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid JSON data"})
		return
	}

	// Set a new ObjectID for the item.
	item.ID = primitive.NewObjectID()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, err := db.ItemCollection.InsertOne(ctx, item)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create item"})
		return
	}

	c.JSON(http.StatusCreated, item)
}

// createItem creates a new item and then adds its ID to the Items array
// in the corresponding Material document.
// func createItem(c *gin.Context) {
// 	var item Item
// 	if err := c.ShouldBindJSON(&item); err != nil {
// 		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid JSON data"})
// 		return
// 	}

// 	// Generate a new ObjectID for the item.
// 	item.ID = primitive.NewObjectID()

// 	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
// 	defer cancel()

// 	// Insert the new item into the items collection.
// 	_, err := itemCollection.InsertOne(ctx, item)
// 	if err != nil {
// 		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create item"})
// 		return
// 	}

// 	// Convert the item's MaterialId (a string) to a MongoDB ObjectID.
// 	materialObjID, err := primitive.ObjectIDFromHex(item.MaterialId)
// 	if err != nil {
// 		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid material id"})
// 		return
// 	}

// 	// Create a new MaterialItem.
// 	// Here, we store the new item's ObjectID in the 'Name' field of MaterialItem
// 	// per your mapping requirements.
// 	newMaterialItem := MaterialItem{
// 		ID:   item.ID,
// 		Name: item.ID.Hex(), // Store the item ID as a string in the Name field.
// 	}

// 	// Update the corresponding Material document by pushing the newMaterialItem
// 	// into its Items array.
// 	update := bson.M{
// 		"$push": bson.M{
// 			"Items": newMaterialItem,
// 		},
// 	}

// 	// 'collection' is assumed to be the global variable for the Materials collection.
// 	_, err = collection.UpdateOne(ctx, bson.M{"_id": materialObjID}, update)
// 	if err != nil {
// 		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update material with item reference"})
// 		return
// 	}

// 	c.JSON(http.StatusCreated, item)
// }

func GetItemsByMaterialID(c *gin.Context) {
	materialId := c.Param("materialId")
	if materialId == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Material ID is required"})
		return
	}

	var items []model.Item
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Query the items collection using the materialId field.
	filter := bson.M{"materialId": materialId}
	cursor, err := db.ItemCollection.Find(ctx, filter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}
	defer cursor.Close(ctx)

	for cursor.Next(ctx) {
		var item model.Item
		if err := cursor.Decode(&item); err != nil {
			// Skip problematic entries.
			continue
		}
		items = append(items, item)
	}

	c.JSON(http.StatusOK, items)
}

// updateItem updates an existing item by its ID.
func UpdateItem(c *gin.Context) {
	id := c.Param("id")
	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid item ID"})
		return
	}

	var updateData map[string]interface{}
	if err := c.ShouldBindJSON(&updateData); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid JSON data"})
		return
	}
	// Remove the "id" field if it exists.
	delete(updateData, "id")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// 1. Prepare the update document
	update := bson.M{"$set": updateData}

	// 2. Define options for FindOneAndUpdate
	opts := options.FindOneAndUpdate()
	// Set ReturnDocument to After so it returns the document *after* the update.
	opts.SetReturnDocument(options.After)

	// 3. Declare a variable to hold the returned item
	var updatedItem model.Item // Use a specific Item struct here if possible, otherwise interface{}

	// 4. Use FindOneAndUpdate instead of UpdateOne
	err = db.ItemCollection.FindOneAndUpdate(
		ctx,
		bson.M{"_id": objID},
		update,
		opts,
	).Decode(&updatedItem)

	if err != nil {
		// Handle case where item is not found or other errors
		if err == mongo.ErrNoDocuments {
			c.JSON(http.StatusNotFound, gin.H{"error": "Item not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update item: " + err.Error()})
		}
		return
	}

	// 5. Return the decoded updated item
	c.JSON(http.StatusOK, updatedItem)
}

// deleteItem deletes an item by its ID.
func DeleteItem(c *gin.Context) {
	id := c.Param("id")
	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid item ID"})
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, err = db.ItemCollection.DeleteOne(ctx, bson.M{"_id": objID})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete item"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Item deleted successfully"})
}

// getItems retrieves all items.
func GetItems(c *gin.Context) {
	var items []model.Item
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cursor, err := db.ItemCollection.Find(ctx, bson.M{})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}
	defer cursor.Close(ctx)

	for cursor.Next(ctx) {
		var item model.Item
		if err := cursor.Decode(&item); err != nil {
			continue // Skip problematic entries
		}
		items = append(items, item)
	}

	c.JSON(http.StatusOK, items)
}

// getItemsBySupplier retrieves all items for a given supplier PID.
func GetItemsBySupplier(c *gin.Context) {
	supplierPid := c.Param("supplierPid")
	if supplierPid == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Supplier PID is required"})
		return
	}

	var items []model.Item
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	filter := bson.M{"supplierPid": supplierPid}
	cursor, err := db.ItemCollection.Find(ctx, filter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}
	defer cursor.Close(ctx)

	for cursor.Next(ctx) {
		var item model.Item
		if err := cursor.Decode(&item); err != nil {
			continue
		}
		items = append(items, item)
	}

	c.JSON(http.StatusOK, items)
}

// getItemsByMaterial retrieves all items for a given material ID.
func GetItemsByMaterial(c *gin.Context) {
	materialId := c.Param("materialId")
	fmt.Println(materialId)
	if materialId == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Material ID is required"})
		return
	}

	var items []model.Item
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// First: fetch all items for the material
	filter := bson.M{"materialId": materialId}
	cursor, err := db.ItemCollection.Find(ctx, filter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}
	defer cursor.Close(ctx)

	// collect items and unique supplier PIDs
	supplierSet := make(map[string]struct{})
	for cursor.Next(ctx) {
		var item model.Item
		if err := cursor.Decode(&item); err != nil {
			continue
		}
		items = append(items, item)
		if item.SupplierPid != "" {
			supplierSet[item.SupplierPid] = struct{}{}
		}
	}

	// If no suppliers found, return empty list
	if len(supplierSet) == 0 {
		c.JSON(http.StatusOK, []model.Item{})
		return
	}

	// build list of unique PIDs
	var pids []string
	for pid := range supplierSet {
		pids = append(pids, pid)
	}

	// Fetch suppliers with status == "approved" whose pid is in pids
	supFilter := bson.M{"pid": bson.M{"$in": pids}, "status": "approved"}
	supCursor, err := db.SupplierCollection.Find(ctx, supFilter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}
	defer supCursor.Close(ctx)

	approvedSet := make(map[string]struct{})
	for supCursor.Next(ctx) {
		var s model.Supplier
		if err := supCursor.Decode(&s); err != nil {
			continue
		}
		if s.PID != "" {
			approvedSet[s.PID] = struct{}{}
		}
	}

	// Filter items to only those whose SupplierPid is in approvedSet
	var filtered []model.Item
	for _, it := range items {
		if _, ok := approvedSet[it.SupplierPid]; ok {
			filtered = append(filtered, it)
		}
	}

	c.JSON(http.StatusOK, filtered)
}
