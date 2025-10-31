package handlers

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"

	"material-api/auth"
	"material-api/db"
	"material-api/email"
	"material-api/model"
)

// getSupplierByID returns a supplier by its ID.
func GetSupplierByID(c *gin.Context) {
	idParam := c.Param("id")
	objID, err := primitive.ObjectIDFromHex(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid supplier ID"})
		return
	}
	var supplier model.Supplier
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	err = db.SupplierCollection.FindOne(ctx, bson.M{"_id": objID}).Decode(&supplier)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Supplier not found"})
		return
	}
	c.JSON(http.StatusOK, supplier)
}
func GetSupplierByPPID(c *gin.Context) {
	pid := c.Param("pid")
	if pid == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Supplier PID is required"})
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var supplier model.Supplier
	err := db.SupplierCollection.FindOne(ctx, bson.M{"pid": pid}).Decode(&supplier)

	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Supplier not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}
		return
	}

	c.JSON(http.StatusOK, supplier)
}

// createSupplier creates a new supplier and also adds a PaymentRecord with Approved=false and Deleted=false.
func CreateSupplier(c *gin.Context) {
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
	_, err := db.SupplierCollection.InsertOne(ctx, supplier)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create supplier"})
		return
	}

	// Create a PaymentRecord for the new supplier with Approved=false and Deleted=false.
	paymentRecord := model.PaymentRecord{
		ID:          primitive.NewObjectID(),
		SupplierPID: supplier.PID,
		Approved:    true,
		Payments:    []model.Payment{}, // Empty payments list.
		Deleted:     false,
	}
	_, err = db.PaymentRecordCollection.InsertOne(ctx, paymentRecord)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Supplier created but failed to create payment record"})
		return
	}
	go func() {
		err := email.SendGeneralMessage(
			[]string{supplier.Email},
			"Welcome to BuildMarket - Your Registration is Complete",
			"Thank you for registering with BuildMarket! Your supplier account has been created successfully. Our team will review your details shortly. You'll receive another notification once your account is approved.",
			supplier.BusinessName,
		)
		if err != nil {
			log.Printf("Failed to send welcome email to %s: %v", supplier.Email, err)
		}
	}()
	token, err := auth.GetManagementToken()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get management token"})
		return
	}

	err = auth.AssignRole(supplier.PID, "rol_H2Nc3mES4d4afJEk", token)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Supplier created but failed to assign Auth0 role"})
		return
	}

	// Respond with the created supplier.
	c.JSON(http.StatusCreated, supplier)
}

// getSupplierByPID returns a supplier only if its associated PaymentRecord is approved or deleted.
func GetSupplierByPID(c *gin.Context) {
	pid := c.Param("pid")
	if pid == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Supplier PID is required"})
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Check for a PaymentRecord for this supplier PID where either Approved is true or Deleted is true.
	var paymentRec model.PaymentRecord
	err := db.PaymentRecordCollection.FindOne(ctx, bson.M{
		"Supplierpid": pid,
		"$or": []bson.M{
			{"Approved": true},
			{"Deleted": true},
		},
	}).Decode(&paymentRec)
	log.Println("===============paymentRec====================", paymentRec)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": fmt.Sprintf("No approved or deleted payment record for supplier PID %s", pid)})
		return
	}

	// If a valid PaymentRecord exists, fetch the supplier.
	var supplier model.Supplier
	err = db.SupplierCollection.FindOne(ctx, bson.M{"pid": pid}).Decode(&supplier)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": fmt.Sprintf("Supplier with PID %s not found", pid)})
		return
	}

	c.JSON(http.StatusOK, supplier)
}

// getSuppliers returns all suppliers whose PaymentRecord is either approved or marked as deleted.
func GetSuppliers(c *gin.Context) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Find all PaymentRecords that satisfy the condition.
	cursor, err := db.PaymentRecordCollection.Find(ctx, bson.M{
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
	suppliersCursor, err := db.SupplierCollection.Find(ctx, bson.M{"pid": bson.M{"$in": supplierPIDs}})
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
func GetSupplierByEmail(c *gin.Context) {
	email := c.Param("email")
	var supplier model.Supplier
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	err := db.SupplierCollection.FindOne(ctx, bson.M{"email": email}).Decode(&supplier)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": fmt.Sprintf("Supplier with email %s not found", email)})
		return
	}
	c.JSON(http.StatusOK, supplier)
}

// updateSupplier updates an existing supplier. Accepts form data for updates including picture URLs.
func UpdateSupplier(c *gin.Context) {
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
	_, err = db.SupplierCollection.UpdateOne(ctx, bson.M{"_id": objID}, bson.M{"$set": updateData})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update supplier"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Supplier updated successfully"})
}

// deleteSupplier removes a supplier by its ID.
func DeleteSupplier(c *gin.Context) {
	idParam := c.Param("id")
	objID, err := primitive.ObjectIDFromHex(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid supplier ID"})
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Load supplier to get PID
	var supplier model.Supplier
	if err := db.SupplierCollection.FindOne(ctx, bson.M{"_id": objID}).Decode(&supplier); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Supplier not found"})
		return
	}

	// Delete items belonging to this supplier
	itemsRes, err := db.ItemCollection.DeleteMany(ctx, bson.M{"supplierPid": supplier.PID})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete supplier items"})
		return
	}

	// Delete supplier
	suppRes, err := db.SupplierCollection.DeleteOne(ctx, bson.M{"_id": objID})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete supplier"})
		return
	}
	if suppRes.DeletedCount == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Supplier not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":      "Supplier and items deleted successfully",
		"deletedItems": itemsRes.DeletedCount,
	})
}

func UpdateSupplierPaymentRecordApprovedStatus(id string, approved bool) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	update := bson.M{"$set": bson.M{"Approved": approved}}
	_, err := db.PaymentRecordCollection.UpdateOne(ctx, bson.M{"Supplierpid": id}, update)

	if err != nil {
		return fmt.Errorf("database error while adding a payment status")
	}

	return nil
}

func SetSupplierPaymentRecordPackageName(id string, packageName string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	update := bson.M{"$set": bson.M{"package_name": packageName}}
	_, err := db.PaymentRecordCollection.UpdateOne(ctx, bson.M{"Supplierpid": id}, update)

	if err != nil {
		return fmt.Errorf("database error while adding a payment status")
	}

	return nil
}

func AppendPaymentToSupplierPaymentRecord(id string, payment model.Payment) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	update := bson.M{"$push": bson.M{"Payments": payment}}
	_, err := db.PaymentRecordCollection.UpdateOne(ctx, bson.M{"Supplierpid": id}, update)
	if err != nil {
		return fmt.Errorf("database error while adding a payment recrod")
	}

	return nil
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