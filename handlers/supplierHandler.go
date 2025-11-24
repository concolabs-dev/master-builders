package handlers

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/microcosm-cc/bluemonday"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"material-api/auth"
	"material-api/db"
	"material-api/email"
	"material-api/model"
)

// getSupplierByID returns a supplier by its ID.
func GetSupplierByID(c *gin.Context) {
	idParam := c.Param("id")
	log.Printf("[INFO] GetSupplierByID called with id=%s", idParam)

	objID, err := primitive.ObjectIDFromHex(idParam)
	if err != nil {
		log.Printf("[WARN] Invalid supplier ID '%s': %v", idParam, err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid supplier ID"})
		return
	}
	var supplier model.Supplier
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	log.Printf("[DEBUG] Fetching supplier with _id=%s from database", objID.Hex())
	err = db.SupplierCollection.FindOne(ctx, bson.M{"_id": objID}).Decode(&supplier)
	if err != nil {
		log.Printf("[WARN] Supplier not found for id=%s: %v", idParam, err)
		c.JSON(http.StatusNotFound, gin.H{"error": "Supplier not found"})
		return
	}

	log.Printf("[INFO] Supplier fetched successfully for id=%s", idParam)
	c.JSON(http.StatusOK, supplier)
}

func GetSupplierByPPID(c *gin.Context) {
	pid := c.Param("pid")
	log.Printf("[INFO] GetSupplierByPPID called with pid=%s", pid)

	if pid == "" {
		log.Printf("[WARN] GetSupplierByPPID called without pid")
		c.JSON(http.StatusBadRequest, gin.H{"error": "Supplier PID is required"})
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var supplier model.Supplier
	log.Printf("[DEBUG] Fetching supplier with pid=%s from database", pid)
	filter := bson.M{
		"pid": pid,
	}
	err := db.SupplierCollection.FindOne(ctx, filter).Decode(&supplier)

	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			log.Printf("[WARN] Supplier not found for pid=%s", pid)
			c.JSON(http.StatusNotFound, gin.H{"error": "Supplier not found"})
		} else {
			log.Printf("[ERROR] Database error while fetching supplier by pid=%s: %v", pid, err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}
		return
	}

	log.Printf("[INFO] Supplier fetched successfully for pid=%s", pid)
	c.JSON(http.StatusOK, supplier)
}

// createSupplier creates a new supplier and also adds a PaymentRecord with Approved=false and Deleted=false.
func CreateSupplier(c *gin.Context) {
	log.Println("[INFO] CreateSupplier called")

	var supplier model.Supplier

	// Parse JSON body.
	if err := c.ShouldBindJSON(&supplier); err != nil {
		log.Printf("[WARN] Invalid JSON data in CreateSupplier: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid JSON data"})
		return
	}

	// Initialize policies
	strict := bluemonday.StrictPolicy()
	ugc := bluemonday.UGCPolicy()

	// 1. Standardize Critical Fields (Email & PID)
	// CRITICAL FIX: Sanitize email first to strip tags, then trim, then lowercase
	supplier.Email = strict.Sanitize(supplier.Email)
	supplier.Email = strings.ToLower(strings.TrimSpace(supplier.Email))
	supplier.PID = strings.TrimSpace(supplier.PID)

	// 2. Sanitize Plain Text Fields (Strict Policy - No HTML Expected)
	supplier.BusinessName = strict.Sanitize(supplier.BusinessName)
	supplier.Telephone = strict.Sanitize(supplier.Telephone)
	supplier.EmailGiven = strict.Sanitize(supplier.EmailGiven) // Assuming this is an alternative contact email
	supplier.Address = strict.Sanitize(supplier.Address)
	supplier.ProfilePicURL = strict.Sanitize(supplier.ProfilePicURL)
	supplier.CoverPicURL = strict.Sanitize(supplier.CoverPicURL)

	// 3. Sanitize Rich Text Fields (UGC Policy - Allows safe formatting)
	supplier.BusinessDesc = ugc.Sanitize(supplier.BusinessDesc)

	// Validate location.
	if supplier.Location.Latitude == 0 && supplier.Location.Longitude == 0 {
		log.Printf("[WARN] Invalid location data for supplier PID=%s", supplier.PID)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid location data"})
		return
	}

	// Create a context.
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Duplicate Check Logic
	filter := bson.M{"email": supplier.Email}
	log.Printf("[DEBUG] CreateSupplier: Checking duplicates for Email: %s", supplier.Email)

	if supplier.PID != "" {
		log.Printf("[DEBUG] CreateSupplier: PID provided (%s), checking Email OR PID", supplier.PID)
		filter = bson.M{
			"$or": []bson.M{
				{"email": supplier.Email},
				{"pid": supplier.PID},
			},
		}
	}

	count, err := db.SupplierCollection.CountDocuments(ctx, filter)
	if err != nil {
		log.Printf("[ERROR] CreateSupplier: Database error checking duplicates: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error checking for existing user"})
		return
	}

	if count > 0 {
		log.Printf("[WARN] CreateSupplier: Duplicate user attempted (Email: %s, PID: %s)", supplier.Email, supplier.PID)
		c.JSON(http.StatusConflict, gin.H{"error": "Supplier with this Email or PID already exists"})
		return
	}

	// Set the supplier ID.
	supplier.ID = primitive.NewObjectID()
	supplier.Status = "pending"
	if supplier.PID == "" {
		supplier.PID = primitive.NewObjectID().Hex()
	}
	log.Printf("[DEBUG] Assigned new ObjectID=%s and PID=%s", supplier.ID.Hex(), supplier.PID)

	// Insert the supplier into the database.
	log.Printf("[DEBUG] Inserting supplier PID=%s into SupplierCollection", supplier.PID)
	_, err = db.SupplierCollection.InsertOne(ctx, supplier)
	if err != nil {
		log.Printf("[ERROR] Failed to create supplier PID=%s: %v", supplier.PID, err)
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
	log.Printf("[DEBUG] Creating PaymentRecord for supplier PID=%s", supplier.PID)
	_, err = db.PaymentRecordCollection.InsertOne(ctx, paymentRecord)
	if err != nil {
		log.Printf("[ERROR] Supplier PID=%s created but failed to create payment record: %v", supplier.PID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Supplier created but failed to create payment record"})
		return
	}

	go func() {
		log.Printf("[INFO] Sending welcome email to supplier PID=%s, email=%s", supplier.PID, supplier.Email)
		err := email.SendGeneralMessage(
			[]string{supplier.Email},
			"Welcome to BuildMarket - Your Registration is Complete",
			"Thank you for registering with BuildMarket! Your supplier account has been created successfully. Our team will review your details shortly. You'll receive another notification once your account is approved.",
			supplier.BusinessName,
		)
		if err != nil {
			log.Printf("[ERROR] Failed to send welcome email to %s: %v", supplier.Email, err)
		}
	}()

	log.Printf("[DEBUG] Requesting management token for supplier PID=%s", supplier.PID)
	token, err := auth.GetManagementToken()
	if err != nil {
		log.Printf("[ERROR] Failed to get management token for supplier PID=%s: %v", supplier.PID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get management token"})
		return
	}

	log.Printf("[DEBUG] Assigning Auth0 role to supplier PID=%s", supplier.PID)
	err = auth.AssignRole(supplier.PID, "rol_H2Nc3mES4d4afJEk", token)
	if err != nil {
		log.Printf("[ERROR] Supplier PID=%s created but failed to assign Auth0 role: %v", supplier.PID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Supplier created but failed to assign Auth0 role"})
		return
	}

	log.Printf("[INFO] Supplier created successfully PID=%s", supplier.PID)
	// Respond with the created supplier.
	c.JSON(http.StatusCreated, supplier)
}

// getSupplierByPID returns a supplier only if its associated PaymentRecord is approved or deleted.
func GetSupplierByPID(c *gin.Context) {
	pid := c.Param("pid")
	log.Printf("[INFO] GetSupplierByPID called with pid=%s", pid)

	if pid == "" {
		log.Printf("[WARN] GetSupplierByPID called without pid")
		c.JSON(http.StatusBadRequest, gin.H{"error": "Supplier PID is required"})
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// // Check for a PaymentRecord for this supplier PID where either Approved is true or Deleted is true.
	// var paymentRec model.PaymentRecord
	// log.Printf("[DEBUG] Looking up PaymentRecord for supplier PID=%s", pid)
	// err := db.PaymentRecordCollection.FindOne(ctx, bson.M{
	// 	"Supplierpid": pid,
	// 	"$or": []bson.M{
	// 		{"Approved": true},
	// 		{"Deleted": true},
	// 	},
	// }).Decode(&paymentRec)

	// if err != nil {
	// 	log.Printf("[WARN] No approved or deleted payment record found for supplier PID=%s: %v", pid, err)
	// 	c.JSON(http.StatusNotFound, gin.H{"error": fmt.Sprintf("No approved or deleted payment record for supplier PID %s", pid)})
	// 	return
	// }

	// If a valid PaymentRecord exists, fetch the supplier.
	var supplier model.Supplier
	log.Printf("[DEBUG] Fetching supplier with pid=%s due to valid PaymentRecord", pid)

	filter := bson.M{
		"pid":    pid,
		"status": "approved",
	}
	log.Printf("[DEBUG] GetSuppliersByStatus: Finding suppliers with filter: %v", filter)

	err := db.SupplierCollection.FindOne(ctx, filter).Decode(&supplier)
	if err != nil {
		log.Printf("[WARN] Supplier with PID %s not found after valid payment record: %v", pid, err)
		c.JSON(http.StatusNotFound, gin.H{"error": fmt.Sprintf("Supplier with PID %s not found", pid)})
		return
	}

	log.Printf("[INFO] Supplier fetched successfully for pid=%s", pid)
	c.JSON(http.StatusOK, supplier)
}

// getSuppliers returns all suppliers whose PaymentRecord is either approved or marked as deleted.
func GetSuppliers(c *gin.Context) {
	log.Println("[INFO] GetSuppliers called")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Find all PaymentRecords that satisfy the condition.
	log.Println("[DEBUG] Fetching PaymentRecords with Approved=true or Deleted=true")
	cursor, err := db.PaymentRecordCollection.Find(ctx, bson.M{
		"$or": []bson.M{
			{"Approved": true},
			{"Deleted": true},
		},
	})
	if err != nil {
		log.Printf("[ERROR] Database error while fetching payment records: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error while fetching payment records"})
		return
	}
	defer cursor.Close(ctx)

	// Collect the SupplierPIDs from these PaymentRecords.
	var supplierPIDs []string
	for cursor.Next(ctx) {
		var rec model.PaymentRecord
		if err := cursor.Decode(&rec); err != nil {
			log.Printf("[WARN] Failed to decode PaymentRecord: %v", err)
			continue
		}
		supplierPIDs = append(supplierPIDs, rec.SupplierPID)
	}
	log.Printf("[DEBUG] Collected %d supplier PIDs from PaymentRecords", len(supplierPIDs))

	// Now find suppliers whose 'pid' is in the supplierPIDs list.
	log.Printf("[DEBUG] Fetching suppliers for %d PIDs", len(supplierPIDs))
	suppliersCursor, err := db.SupplierCollection.Find(ctx, bson.M{"pid": bson.M{"$in": supplierPIDs}})
	if err != nil {
		log.Printf("[ERROR] Database error while fetching suppliers: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error while fetching suppliers"})
		return
	}
	defer suppliersCursor.Close(ctx)

	var suppliers []model.Supplier
	for suppliersCursor.Next(ctx) {
		var supplier model.Supplier
		if err := suppliersCursor.Decode(&supplier); err != nil {
			log.Printf("[WARN] Failed to decode Supplier: %v", err)
			continue
		}
		suppliers = append(suppliers, supplier)
	}

	log.Printf("[INFO] GetSuppliers returning %d suppliers", len(suppliers))
	c.JSON(http.StatusOK, suppliers)
}

// GetSuppliersByStatus returns all suppliers that match a given status.
// The status is provided as a query parameter (e.g., /api/suppliers/status?q=active)
func GetApprovedSuppliers(c *gin.Context) {
	log.Printf("[INFO] GetApprovedSuppliers called")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// 2. Create the filter dynamically based on the query param
	filter := bson.M{"status": "approved"}
	log.Printf("[DEBUG] GetSuppliersByStatus: Finding suppliers with filter: %v", filter)

	suppliersCursor, err := db.SupplierCollection.Find(ctx, filter)
	if err != nil {
		log.Printf("[ERROR] GetSuppliersByStatus: Database error: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error while fetching suppliers"})
		return
	}
	defer suppliersCursor.Close(ctx)

	// 3. Decode the suppliers
	var suppliers []model.Supplier
	for suppliersCursor.Next(ctx) {
		var supplier model.Supplier
		if err := suppliersCursor.Decode(&supplier); err != nil {
			log.Printf("[WARN] GetSuppliersByStatus: Failed to decode Supplier: %v", err)
			continue // Skip this supplier
		}
		suppliers = append(suppliers, supplier)
	}

	// 4. Return the results
	// It's usually better to return 200 OK with an empty list,
	// as it's easier for frontends to handle than a 404.
	if len(suppliers) == 0 {
		log.Printf("[INFO] GetSuppliersByStatus: No suppliers found")
	} else {
		log.Printf("[INFO] GetSuppliersByStatus: returning %d suppliers", len(suppliers))
	}

	c.JSON(http.StatusOK, suppliers)
}

// GetAllSuppliers returns all suppliers, joined with their payment record.
func GetAllSuppliers(c *gin.Context) {
	log.Println("[INFO] GetAllSuppliers called")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// This aggregation pipeline is the "JOIN"
	pipeline := mongo.Pipeline{
		// Stage 1: $lookup
		{
			{Key: "$lookup", Value: bson.M{
				"from":         "payments",
				"localField":   "pid",
				"foreignField": "Supplierpid",
				"as":           "recordArray",
			}},
		},
		// Stage 2: $unwind
		{
			{Key: "$unwind", Value: bson.M{
				"path":                       "$recordArray",
				"preserveNullAndEmptyArrays": true,
			}},
		},
		// Stage 3: $project
		{
			{Key: "$project", Value: bson.M{
				"supplier": "$$ROOT",
				"record":   "$recordArray",
			}},
		},
		// Stage 4: $project (cleanup)
		{
			{Key: "$project", Value: bson.M{
				"supplier.recordArray": 0,
			}},
		},
	}

	// Run the aggregation on the SupplierCollection
	cursor, err := db.SupplierCollection.Aggregate(ctx, pipeline)
	if err != nil {
		log.Printf("[ERROR] Database aggregation error: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}
	defer cursor.Close(ctx)

	// Decode all results into our slice
	var suppliers []model.SupplierWithRecord
	if err = cursor.All(ctx, &suppliers); err != nil {
		log.Printf("[ERROR] Failed to decode aggregation results: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to decode data"})
		return
	}

	// Handle case where no suppliers are found at all
	if suppliers == nil {
		suppliers = []model.SupplierWithRecord{}
	}

	supplierWithRecordResponse := make([]model.SupplierWithRecordResponse, len(suppliers))

	// Loop over the database results and map them to the frontend struct
	for i, supplier := range suppliers {
		// Handle the 'null' records from the $unwind
		var payments []model.Payment
		if supplier.Record.Payments != nil {
			payments = supplier.Record.Payments
		} else {
			payments = []model.Payment{}
		}

		supplierWithRecordResponse[i] = model.SupplierWithRecordResponse{
			Supplier: supplier.Supplier,
			Record: model.PaymentRecordReponse{
				ID:          supplier.Record.ID,
				PID:         supplier.Record.SupplierPID,
				Approved:    supplier.Record.Approved,
				Payments:    payments,
				Deleted:     supplier.Record.Deleted,
				PackageName: supplier.Record.PackageName,
			},
		}
	}

	log.Printf("[INFO] GetAllSuppliers returning %d combined supplier records", len(supplierWithRecordResponse))
	c.JSON(http.StatusOK, supplierWithRecordResponse)
}

// getSupplierByEmail returns a supplier by its email.
func GetSupplierByEmail(c *gin.Context) {
	emailParam := c.Param("email")
	log.Printf("[INFO] GetSupplierByEmail called with email=%s", emailParam)

	var supplier model.Supplier
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	log.Printf("[DEBUG] Fetching supplier with email=%s from database", emailParam)
	err := db.SupplierCollection.FindOne(ctx, bson.M{"email": emailParam}).Decode(&supplier)
	if err != nil {
		log.Printf("[WARN] Supplier with email %s not found: %v", emailParam, err)
		c.JSON(http.StatusNotFound, gin.H{"error": fmt.Sprintf("Supplier with email %s not found", emailParam)})
		return
	}

	log.Printf("[INFO] Supplier fetched successfully for email=%s", emailParam)
	c.JSON(http.StatusOK, supplier)
}

// updateSupplier updates an existing supplier. Accepts form data for updates including picture URLs.
func UpdateSupplier(c *gin.Context) {
	idParam := c.Param("id")
	log.Printf("[INFO] UpdateSupplier called with id=%s", idParam)

	objID, err := primitive.ObjectIDFromHex(idParam)
	if err != nil {
		log.Printf("[WARN] Invalid supplier ID '%s': %v", idParam, err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid supplier ID"})
		return
	}

	var updateData map[string]interface{}
	if err := c.ShouldBindJSON(&updateData); err != nil {
		log.Printf("[WARN] Invalid JSON data in UpdateSupplier for id=%s: %v", idParam, err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid JSON data"})
		return
	}

	delete(updateData, "id")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	update := bson.M{"$set": updateData}

	opts := options.FindOneAndUpdate()

	opts.SetReturnDocument(options.After)

	var updatedSupplier model.Supplier

	log.Printf("[DEBUG] Updating supplier _id=%s with data=%v", objID.Hex(), updateData)

	err = db.SupplierCollection.FindOneAndUpdate(
		ctx,
		bson.M{"_id": objID},
		update,
		opts,
	).Decode(&updatedSupplier)

	if err != nil {
		if err == mongo.ErrNoDocuments {
			log.Printf("[WARN] Supplier not found for ID: %s", idParam)
			c.JSON(http.StatusNotFound, gin.H{"error": "Supplier not found"})
		} else {
			log.Printf("[ERROR] Failed to update supplier id=%s: %v", idParam, err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update supplier"})
		}
		return
	}

	log.Printf("[INFO] Supplier updated successfully id=%s", idParam)
	c.JSON(http.StatusOK, updatedSupplier)
}

// deleteSupplier removes a supplier by its ID.
func DeleteSupplier(c *gin.Context) {
	idParam := c.Param("id")
	log.Printf("[INFO] DeleteSupplier called with id=%s", idParam)

	objID, err := primitive.ObjectIDFromHex(idParam)
	if err != nil {
		log.Printf("[WARN] Invalid supplier ID '%s': %v", idParam, err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid supplier ID"})
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Load supplier to get PID
	log.Printf("[DEBUG] Fetching supplier _id=%s before delete", objID.Hex())
	var supplier model.Supplier
	if err := db.SupplierCollection.FindOne(ctx, bson.M{"_id": objID}).Decode(&supplier); err != nil {
		log.Printf("[WARN] Supplier not found for delete id=%s: %v", idParam, err)
		c.JSON(http.StatusNotFound, gin.H{"error": "Supplier not found"})
		return
	}

	// Delete items belonging to this supplier
	log.Printf("[DEBUG] Deleting items for supplier PID=%s", supplier.PID)
	itemsRes, err := db.ItemCollection.DeleteMany(ctx, bson.M{"supplierPid": supplier.PID})
	if err != nil {
		log.Printf("[ERROR] Failed to delete items for supplier PID=%s: %v", supplier.PID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete supplier items"})
		return
	}

	// Delete supplier
	log.Printf("[DEBUG] Deleting supplier _id=%s", objID.Hex())
	suppRes, err := db.SupplierCollection.DeleteOne(ctx, bson.M{"_id": objID})
	if err != nil {
		log.Printf("[ERROR] Failed to delete supplier id=%s: %v", idParam, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete supplier"})
		return
	}
	if suppRes.DeletedCount == 0 {
		log.Printf("[WARN] Supplier delete attempted but no document deleted for id=%s", idParam)
		c.JSON(http.StatusNotFound, gin.H{"error": "Supplier not found"})
		return
	}

	log.Printf("[INFO] Supplier and items deleted successfully id=%s, deletedItems=%d", idParam, itemsRes.DeletedCount)
	c.JSON(http.StatusOK, gin.H{
		"message":      "Supplier and items deleted successfully",
		"deletedItems": itemsRes.DeletedCount,
	})
}

func UpdateSupplierPaymentRecordApprovedStatus(id string, approved bool, status string) error {
	log.Printf("[INFO] UpdateSupplierPaymentRecordApprovedStatus called for Supplierpid=%s approved=%t", id, approved)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	update := bson.M{"$set": bson.M{"Approved": approved}}
	log.Printf("[DEBUG] Updating PaymentRecord Approved field for Supplierpid=%s", id)
	_, err := db.PaymentRecordCollection.UpdateOne(ctx, bson.M{"Supplierpid": id}, update)
	if err != nil {
		log.Printf("[ERROR] Database error while updating Approved status for Supplierpid=%s: %v", id, err)
		return fmt.Errorf("database error while adding a payment status")
	}

	statusUpdate := bson.M{"$set": bson.M{"status": status}}
	_, err = db.SupplierCollection.UpdateOne(ctx, bson.M{"pid": id}, statusUpdate)
	if err != nil {
		log.Printf("[ERROR] Database error while updating Approved status for Supplierpid=%s: %v", id, err)
		return fmt.Errorf("database error while adding a payment status")
	}

	log.Printf("[INFO] Approved status updated successfully for Supplierpid=%s", id)
	return nil
}

func SetSupplierPaymentRecordPackageName(id string, packageName string) error {
	log.Printf("[INFO] SetSupplierPaymentRecordPackageName called for Supplierpid=%s packageName=%s", id, packageName)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	update := bson.M{"$set": bson.M{"package_name": packageName}}
	log.Printf("[DEBUG] Updating PaymentRecord package_name for Supplierpid=%s", id)
	_, err := db.PaymentRecordCollection.UpdateOne(ctx, bson.M{"Supplierpid": id}, update)

	if err != nil {
		log.Printf("[ERROR] Database error while setting package_name for Supplierpid=%s: %v", id, err)
		return fmt.Errorf("database error while adding a payment status")
	}

	log.Printf("[INFO] package_name set successfully for Supplierpid=%s", id)
	return nil
}

func AppendPaymentToSupplierPaymentRecord(id string, payment model.Payment) error {
	log.Printf("[INFO] AppendPaymentToSupplierPaymentRecord called for Supplierpid=%s", id)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	update := bson.M{"$push": bson.M{"Payments": payment}}
	log.Printf("[DEBUG] Appending payment to PaymentRecord for Supplierpid=%s", id)
	_, err := db.PaymentRecordCollection.UpdateOne(ctx, bson.M{"Supplierpid": id}, update)
	if err != nil {
		log.Printf("[ERROR] Database error while appending payment for Supplierpid=%s: %v", id, err)
		return fmt.Errorf("database error while adding a payment recrod")
	}

	log.Printf("[INFO] Payment appended successfully for Supplierpid=%s", id)
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
