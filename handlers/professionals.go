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

	// Import your models package or adjust as needed
	"material-api/model"
)

var professionalCollection *mongo.Collection
var professionalPaymentRecordCollection *mongo.Collection

// InitProfessionalCollections initializes the collections for professionals
func InitProfessionalCollections(database *mongo.Database) {
	professionalCollection = database.Collection("professionals")
	professionalPaymentRecordCollection = database.Collection("professional_payment_records")
}

// CreateProfessional creates a new professional and adds a payment record with Approved=false and Deleted=false
func CreateProfessional(c *gin.Context) {
	var professional model.Professional

	// Parse JSON body
	if err := c.ShouldBindJSON(&professional); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid JSON data"})
		return
	}

	// Validate location
	if professional.Location.Latitude == 0 && professional.Location.Longitude == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid location data"})
		return
	}

	// Set the professional ID and PID if not provided
	professional.ID = primitive.NewObjectID()
	if professional.PID == "" {
		professional.PID = primitive.NewObjectID().Hex()
	}

	// Create a context
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Insert the professional into the database
	_, err := professionalCollection.InsertOne(ctx, professional)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create professional"})
		return
	}

	// Create a PaymentRecord for the new professional with Approved=false and Deleted=false
	paymentRecord := model.ProfessionalPaymentRecord{
		ID:              primitive.NewObjectID(),
		ProfessionalPID: professional.PID,
		Approved:        false,
		Payments:        []model.Payment{}, // Empty payments list
		Deleted:         false,
	}

	_, err = professionalPaymentRecordCollection.InsertOne(ctx, paymentRecord)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Professional created but failed to create payment record"})
		return
	}

	// Respond with the created professional
	c.JSON(http.StatusCreated, professional)
}

// GetProfessionals returns all professionals whose PaymentRecord is either approved or marked as deleted
func GetProfessionals(c *gin.Context) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Find all PaymentRecords that satisfy the condition
	cursor, err := professionalPaymentRecordCollection.Find(ctx, bson.M{
		"$or": []bson.M{
			{"approved": true},
			{"deleted": true},
		},
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error while fetching payment records"})
		return
	}
	defer cursor.Close(ctx)

	// Collect the ProfessionalPIDs from these PaymentRecords
	var professionalPIDs []string
	for cursor.Next(ctx) {
		var rec model.ProfessionalPaymentRecord
		if err := cursor.Decode(&rec); err != nil {
			continue
		}
		professionalPIDs = append(professionalPIDs, rec.ProfessionalPID)
	}

	// Find professionals whose 'pid' is in the professionalPIDs list
	professionalsCursor, err := professionalCollection.Find(ctx, bson.M{"pid": bson.M{"$in": professionalPIDs}})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error while fetching professionals"})
		return
	}
	defer professionalsCursor.Close(ctx)

	var professionals []model.Professional
	for professionalsCursor.Next(ctx) {
		var professional model.Professional
		if err := professionalsCursor.Decode(&professional); err != nil {
			continue
		}
		professionals = append(professionals, professional)
	}

	c.JSON(http.StatusOK, professionals)
}

// GetProfessionalByID returns a professional by its ID
func GetProfessionalByID(c *gin.Context) {
	idParam := c.Param("id")
	objID, err := primitive.ObjectIDFromHex(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid professional ID"})
		return
	}

	var professional model.Professional
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err = professionalCollection.FindOne(ctx, bson.M{"_id": objID}).Decode(&professional)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Professional not found"})
		return
	}

	c.JSON(http.StatusOK, professional)
}

// GetProfessionalByPID returns a professional only if its associated PaymentRecord is approved or deleted
func GetProfessionalByPID(c *gin.Context) {
	pid := c.Param("pid")
	if pid == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Professional PID is required"})
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Check for a PaymentRecord for this professional PID where either Approved is true or Deleted is true
	// var paymentRec model.ProfessionalPaymentRecord
	// err := professionalPaymentRecordCollection.FindOne(ctx, bson.M{
	// 	"professional_pid": pid,
	// 	"$or": []bson.M{
	// 		{"approved": true},
	// 		{"deleted": true},
	// 	},
	// }).Decode(&paymentRec)

	// if err != nil {
	// 	c.JSON(http.StatusNotFound, gin.H{"error": fmt.Sprintf("No approved or deleted payment record for professional PID %s", pid)})
	// 	return
	// }

	// If a valid PaymentRecord exists, fetch the professional
	var professional model.Professional
	err := professionalCollection.FindOne(ctx, bson.M{"pid": pid}).Decode(&professional)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": fmt.Sprintf("Professional with PID %s not found", pid)})
		return
	}

	c.JSON(http.StatusOK, professional)
}

// GetProfessionalByEmail returns a professional by its email
func GetProfessionalByEmail(c *gin.Context) {
	email := c.Param("email")

	var professional model.Professional
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := professionalCollection.FindOne(ctx, bson.M{"email": email}).Decode(&professional)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": fmt.Sprintf("Professional with email %s not found", email)})
		return
	}

	c.JSON(http.StatusOK, professional)
}

// UpdateProfessional updates an existing professional
func UpdateProfessional(c *gin.Context) {
	idParam := c.Param("id")
	objID, err := primitive.ObjectIDFromHex(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid professional ID"})
		return
	}

	// Use JSON binding for partial updates
	var updateData map[string]interface{}
	if err := c.ShouldBindJSON(&updateData); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid JSON data"})
		return
	}

	// Remove the "id" field if present
	delete(updateData, "id")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err = professionalCollection.UpdateOne(ctx, bson.M{"_id": objID}, bson.M{"$set": updateData})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update professional"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Professional updated successfully"})
}

// DeleteProfessional removes a professional by its ID
func DeleteProfessional(c *gin.Context) {
	idParam := c.Param("id")
	objID, err := primitive.ObjectIDFromHex(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid professional ID"})
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err = professionalCollection.DeleteOne(ctx, bson.M{"_id": objID})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete professional"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Professional deleted successfully"})
}

// GetAllProfessionals returns all professionals in the database without payment record filtering
// This is useful for admin purposes to see all professionals regardless of approval status
func GetAllProfessionals(c *gin.Context) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cursor, err := professionalCollection.Find(ctx, bson.M{})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error while fetching professionals"})
		return
	}
	defer cursor.Close(ctx)

	var professionals []model.Professional
	if err = cursor.All(ctx, &professionals); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error decoding professionals data"})
		return
	}

	// Return empty array instead of nil when no professionals are found
	if professionals == nil {
		professionals = []model.Professional{}
	}

	c.JSON(http.StatusOK, professionals)
}

func UpdateProfessionalPaymentRecordApprovedStatus(id string, approved bool) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	update := bson.M{"$set": bson.M{"approved": approved}}
	result, err := professionalPaymentRecordCollection.UpdateOne(ctx, bson.M{"professional_pid": id}, update)
	if err != nil {
		return fmt.Errorf("database error while adding a payment status")
	}
	if result.MatchedCount == 0 {
		return fmt.Errorf("no professional payment record found with ID %s", id)
	}

	return nil
}

func AppendPaymentToProfessionalPaymentRecord(id string, payment model.Payment) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	update := bson.M{"$push": bson.M{"payments": payment}}
	result, err := professionalPaymentRecordCollection.UpdateOne(ctx, bson.M{"professional_pid": id}, update)
	if err != nil {
		return fmt.Errorf("database error while adding a payment recrod")
	}
	if result.MatchedCount == 0 {
		return fmt.Errorf("no professional payment record found with ID %s", id)
	}

	return nil
}
