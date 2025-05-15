package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	// "math"
	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"

	"material-api/model"
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

	var materials []model.Material
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cursor, err := collection.Find(ctx, filter)
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
func getMaterials(c *gin.Context) {
	var materials []model.Material
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
func getMaterialByID(c *gin.Context) {
	id, err := primitive.ObjectIDFromHex(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID"})
		return
	}

	var material model.Material
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

	cursor, err := collection.Find(ctx, filter)
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
func createMaterial(c *gin.Context) {
	var material model.Material
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
	var types []model.Type
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cursor, err := typeCollection.Find(ctx, bson.M{})
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

	err = typeCollection.FindOne(ctx, bson.M{"_id": objID}).Decode(&t)
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

	var updateData model.Type
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
func getMajorCurrencies(c *gin.Context) {
	// List of major currencies you want to fetch from the database
	// majorCurrencies := []string{"USD", "EUR", "GBP", "JPY", "CNY", "INR", "AUD", "CAD", "CHF", "SAR", "ZAR", "KRW", "SGD", "AED", "BRL"}

	// Find the latest exchange rates document from MongoDB
	var result model.CurrencyDocument

	// Fetch the most recent exchange rate document

	opts := options.FindOne().SetSort(map[string]int{"timestamp": -1}) // Sort by most recent timestamp
	fmt.Println(opts)
	err := exchangeRateCollection.FindOne(context.Background(), bson.D{}, opts).Decode(&result)
	fmt.Println(result)
	if err != nil {
		log.Println("Error fetching exchange rates from MongoDB:", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch exchange rates"})
		return
	}

	// Create a map for major currencies
	majorRates := map[string]float64{
		"USD": result.ConversionRates.USD,
		"EUR": result.ConversionRates.EUR,
		"GBP": result.ConversionRates.GBP,
		"JPY": result.ConversionRates.JPY,
		"CNY": result.ConversionRates.CNY,
		"INR": result.ConversionRates.INR,
		"AUD": result.ConversionRates.AUD,
		"CAD": result.ConversionRates.CAD,
		"CHF": result.ConversionRates.CHF,
		"SAR": result.ConversionRates.SAR,
		"ZAR": result.ConversionRates.ZAR,
		"KRW": result.ConversionRates.KRW,
		"SGD": result.ConversionRates.SGD,
		"AED": result.ConversionRates.AED,
		"BRL": result.ConversionRates.BRL,
	}

	// Filter and return only the major currencies
	// for _, currency := range majorCurrencies {
	// 	if rate, exists := result.ConversionRates[currency]; exists {
	// 		majorRates[currency] = rate
	// 	}
	// }

	c.JSON(http.StatusOK, gin.H{"major_currencies": majorRates})
}

//supplier
// func createSupplier(c *gin.Context) {
// 	var supplier Supplier

// 	fmt.Println(c)

// 	// Parse form values.
// 	supplier.Email = c.PostForm("email")
// 	supplier.PID = c.PostForm("pid")
// 	supplier.BusinessName = c.PostForm("business_name")
// 	supplier.BusinessDesc = c.PostForm("business_description")
// 	supplier.Telephone = c.PostForm("telephone")
// 	supplier.EmailGiven = c.PostForm("email_given")
// 	supplier.Address = c.PostForm("address")

// 	// Parse location fields.
// 	latStr := c.PostForm("latitude")
// 	lonStr := c.PostForm("longitude")
// 	if latStr != "" && lonStr != "" {
// 		var lat, lon float64
// 		fmt.Sscanf(latStr, "%f", &lat)
// 		fmt.Sscanf(lonStr, "%f", &lon)
// 		supplier.Location = Location{Latitude: lat, Longitude: lon}
// 	}

// 	// Get profile picture URL directly from the form.
// 	supplier.ProfilePicURL = c.PostForm("profile_pic_url")

// 	// Get cover picture URL directly from the form.
// 	supplier.CoverPicURL = c.PostForm("cover_pic_url")
// 	fmt.Println(supplier)
// 	supplier.ID = primitive.NewObjectID()
// 	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
// 	defer cancel()
// 	_, err := supplierCollection.InsertOne(ctx, supplier)
// 	if err != nil {
// 		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create supplier"})
// 		return
// 	}
// 	fmt.Println(supplier)
// 	c.JSON(http.StatusCreated, supplier)
// }
// func createSupplier(c *gin.Context) {
//     var supplier Supplier

//     // Parse JSON body.
//     if err := c.ShouldBindJSON(&supplier); err != nil {
//         c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid JSON data"})
//         return
//     }

//     // Parse location fields.
//     if supplier.Location.Latitude == 0 && supplier.Location.Longitude == 0 {
//         c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid location data"})
//         return
//     }

//     // Set the supplier ID.
//     supplier.ID = primitive.NewObjectID()

//     // Insert the supplier into the database.
//     ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
//     defer cancel()
//     _, err := supplierCollection.InsertOne(ctx, supplier)
//     if err != nil {
//         c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create supplier"})
//         return
//     }

//     // Respond with the created supplier.
//     c.JSON(http.StatusCreated, supplier)
// }

// // getSuppliers returns all suppliers.
// func getSuppliers(c *gin.Context) {
// 	var suppliers []Supplier
// 	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
// 	defer cancel()
// 	cursor, err := supplierCollection.Find(ctx, bson.M{})
// 	if err != nil {
// 		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
// 		return
// 	}
// 	defer cursor.Close(ctx)
// 	for cursor.Next(ctx) {
// 		var supplier Supplier
// 		if err := cursor.Decode(&supplier); err != nil {
// 			continue
// 		}
// 		suppliers = append(suppliers, supplier)
// 	}
// 	c.JSON(http.StatusOK, suppliers)
// }

// getSupplierByID returns a supplier by its ID.
func getSupplierByID(c *gin.Context) {
	idParam := c.Param("id")
	objID, err := primitive.ObjectIDFromHex(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid supplier ID"})
		return
	}
	var supplier model.Supplier
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	err = supplierCollection.FindOne(ctx, bson.M{"_id": objID}).Decode(&supplier)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Supplier not found"})
		return
	}
	c.JSON(http.StatusOK, supplier)
}
func getSupplierByPPID(c *gin.Context) {
	pid := c.Param("pid")
	var supplier model.Supplier
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	err := supplierCollection.FindOne(ctx, bson.M{"pid": pid}).Decode(&supplier)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": fmt.Sprintf("Supplier with PID %s not found", pid)})
		return
	}
	c.JSON(http.StatusOK, supplier)
}

// createSupplier creates a new supplier and also adds a PaymentRecord with Approved=false and Deleted=false.
func createSupplier(c *gin.Context) {
	var supplier model.Supplier

	// Parse JSON body.
	if err := c.ShouldBindJSON(&supplier); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid JSON data"})
		return
	}

	// Validate location.
	if supplier.Location.Latitude == 0 && supplier.Location.Longitude == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid location data"})
		return
	}

	// Set the supplier ID.
	supplier.ID = primitive.NewObjectID()

	// Create a context.
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Insert the supplier into the database.
	_, err := supplierCollection.InsertOne(ctx, supplier)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create supplier"})
		return
	}

	// Create a PaymentRecord for the new supplier with Approved=false and Deleted=false.
	paymentRecord := model.PaymentRecord{
		ID:          primitive.NewObjectID(),
		SupplierPID: supplier.PID,
		Approved:    false,
		Payments:    []model.Payment{}, // Empty payments list.
		Deleted:     false,
	}
	_, err = paymentRecordCollection.InsertOne(ctx, paymentRecord)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Supplier created but failed to create payment record"})
		return
	}

	// Respond with the created supplier.
	c.JSON(http.StatusCreated, supplier)
}

// getSupplierByPID returns a supplier only if its associated PaymentRecord is approved or deleted.
func getSupplierByPID(c *gin.Context) {
	pid := c.Param("pid")
	if pid == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Supplier PID is required"})
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Check for a PaymentRecord for this supplier PID where either Approved is true or Deleted is true.
	var paymentRec model.PaymentRecord
	err := paymentRecordCollection.FindOne(ctx, bson.M{
		"Supplierpid": pid,
		"$or": []bson.M{
			{"Approved": true},
			{"Deleted": true},
		},
	}).Decode(&paymentRec)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": fmt.Sprintf("No approved or deleted payment record for supplier PID %s", pid)})
		return
	}

	// If a valid PaymentRecord exists, fetch the supplier.
	var supplier model.Supplier
	err = supplierCollection.FindOne(ctx, bson.M{"pid": pid}).Decode(&supplier)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": fmt.Sprintf("Supplier with PID %s not found", pid)})
		return
	}

	c.JSON(http.StatusOK, supplier)
}

// getSuppliers returns all suppliers whose PaymentRecord is either approved or marked as deleted.
func getSuppliers(c *gin.Context) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Find all PaymentRecords that satisfy the condition.
	cursor, err := paymentRecordCollection.Find(ctx, bson.M{
		"$or": []bson.M{
			{"Approved": true},
			{"Deleted": true},
		},
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error while fetching payment records"})
		return
	}
	defer cursor.Close(ctx)

	// Collect the SupplierPIDs from these PaymentRecords.
	var supplierPIDs []string
	for cursor.Next(ctx) {
		var rec model.PaymentRecord
		if err := cursor.Decode(&rec); err != nil {
			continue
		}
		supplierPIDs = append(supplierPIDs, rec.SupplierPID)
	}

	// Now find suppliers whose 'pid' is in the supplierPIDs list.
	suppliersCursor, err := supplierCollection.Find(ctx, bson.M{"pid": bson.M{"$in": supplierPIDs}})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error while fetching suppliers"})
		return
	}
	defer suppliersCursor.Close(ctx)

	var suppliers []model.Supplier
	for suppliersCursor.Next(ctx) {
		var supplier model.Supplier
		if err := suppliersCursor.Decode(&supplier); err != nil {
			continue
		}
		suppliers = append(suppliers, supplier)
	}

	c.JSON(http.StatusOK, suppliers)
}

// getSupplierByEmail returns a supplier by its email.
func getSupplierByEmail(c *gin.Context) {
	email := c.Param("email")
	var supplier model.Supplier
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	err := supplierCollection.FindOne(ctx, bson.M{"email": email}).Decode(&supplier)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": fmt.Sprintf("Supplier with email %s not found", email)})
		return
	}
	c.JSON(http.StatusOK, supplier)
}

// updateSupplier updates an existing supplier. Accepts form data for updates including picture URLs.
func updateSupplier(c *gin.Context) {
	idParam := c.Param("id")
	objID, err := primitive.ObjectIDFromHex(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid supplier ID"})
		return
	}

	// Use JSON binding for partial updates.
	var updateData map[string]interface{}
	if err := c.ShouldBindJSON(&updateData); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid JSON data"})
		return
	}

	// Remove the "id" field if present.
	delete(updateData, "id")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, err = supplierCollection.UpdateOne(ctx, bson.M{"_id": objID}, bson.M{"$set": updateData})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update supplier"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Supplier updated successfully"})
}

// deleteSupplier removes a supplier by its ID.
func deleteSupplier(c *gin.Context) {
	idParam := c.Param("id")
	objID, err := primitive.ObjectIDFromHex(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid supplier ID"})
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, err = supplierCollection.DeleteOne(ctx, bson.M{"_id": objID})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete supplier"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Supplier deleted successfully"})
}

// getItems retrieves all items.
func getItems(c *gin.Context) {
	var items []model.Item
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cursor, err := itemCollection.Find(ctx, bson.M{})
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
func getItemsBySupplier(c *gin.Context) {
	supplierPid := c.Param("supplierPid")
	if supplierPid == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Supplier PID is required"})
		return
	}

	var items []model.Item
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	filter := bson.M{"supplierPid": supplierPid}
	cursor, err := itemCollection.Find(ctx, filter)
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
func getItemsByMaterial(c *gin.Context) {
	materialId := c.Param("materialId")
	fmt.Println(materialId)
	if materialId == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Material ID is required"})
		return
	}

	var items []model.Item
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	filter := bson.M{"materialId": materialId}
	cursor, err := itemCollection.Find(ctx, filter)
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

// createItem creates a new item.
func createItem(c *gin.Context) {
	var item model.Item
	if err := c.ShouldBindJSON(&item); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid JSON data"})
		return
	}

	// Set a new ObjectID for the item.
	item.ID = primitive.NewObjectID()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, err := itemCollection.InsertOne(ctx, item)
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

func getItemsByMaterialID(c *gin.Context) {
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
	cursor, err := itemCollection.Find(ctx, filter)
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
func updateItem(c *gin.Context) {
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
	update := bson.M{"$set": updateData}
	_, err = itemCollection.UpdateOne(ctx, bson.M{"_id": objID}, update)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update item"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Item updated successfully"})
}

// deleteItem deletes an item by its ID.
func deleteItem(c *gin.Context) {
	id := c.Param("id")
	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid item ID"})
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, err = itemCollection.DeleteOne(ctx, bson.M{"_id": objID})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete item"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Item deleted successfully"})
}

func createPaymentRecord(c *gin.Context) {
	var record model.PaymentRecord
	if err := c.ShouldBindJSON(&record); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid JSON data"})
		return
	}

	// Set a new ObjectID for the record.
	record.ID = primitive.NewObjectID()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := paymentRecordCollection.InsertOne(ctx, record)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create payment record"})
		return
	}

	c.JSON(http.StatusCreated, record)
}
func getPaymentRecords(c *gin.Context) {
	var records []model.PaymentRecord
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Only return records that are not marked as deleted.
	cursor, err := paymentRecordCollection.Find(ctx, bson.M{"Deleted": false})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}
	defer cursor.Close(ctx)

	for cursor.Next(ctx) {
		var record model.PaymentRecord
		if err := cursor.Decode(&record); err != nil {
			continue // Skip problematic entries.
		}
		records = append(records, record)
	}

	c.JSON(http.StatusOK, records)
}
func getPaymentRecordByID(c *gin.Context) {
	idParam := c.Param("id")
	objID, err := primitive.ObjectIDFromHex(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID"})
		return
	}

	var record model.PaymentRecord
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err = paymentRecordCollection.FindOne(ctx, bson.M{"_id": objID, "Deleted": false}).Decode(&record)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Payment record not found"})
		return
	}

	c.JSON(http.StatusOK, record)
}
func updatePaymentRecord(c *gin.Context) {
	idParam := c.Param("id")
	objID, err := primitive.ObjectIDFromHex(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID"})
		return
	}

	var updateData map[string]interface{}
	if err := c.ShouldBindJSON(&updateData); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid JSON data"})
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	update := bson.M{"$set": updateData}
	_, err = paymentRecordCollection.UpdateOne(ctx, bson.M{"_id": objID}, update)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update payment record"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Payment record updated successfully"})
}
func deletePaymentRecord(c *gin.Context) {
	idParam := c.Param("id")
	objID, err := primitive.ObjectIDFromHex(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID"})
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Soft delete by setting the Deleted field to true.
	update := bson.M{"$set": bson.M{"Deleted": true}}
	_, err = paymentRecordCollection.UpdateOne(ctx, bson.M{"_id": objID}, update)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete payment record"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Payment record deleted successfully"})
}
