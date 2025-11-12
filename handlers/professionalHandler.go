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

	// Import your models package or adjust as needed
	"material-api/auth"
	"material-api/db"
	"material-api/model"
)

// // InitProfessionalCollections initializes the collections for professionals
// func InitProfessionalCollections(database *mongo.Database) {
// 	professionalCollection = database.Collection("professionals")
// 	professionalPaymentRecordCollection = database.Collection("professional_payment_records")
// 	projectCollection = database.Collection("projects")
// }

// CreateProfessional creates a new professional and adds a payment record with Approved=false and Deleted=false
func CreateProfessional(c *gin.Context) {
	var professional model.Professional
	fmt.Println("Creating a new professional")
	// Parse JSON body
	// var rawBody map[string]interface{}
	// if err := c.ShouldBindJSON(&rawBody); err != nil {
	// 	fmt.Println("Raw JSON body:", rawBody)
	// 	c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid JSON data"})
	// 	return
	// }
	// fmt.Println("Raw JSON body:", rawBody)
	if err := c.ShouldBindJSON(&professional); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid JSON data"})
		return
	}

	// Validate location
	// if professional.Location.Latitude == 0 && professional.Location.Longitude == 0 {
	// 	c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid location data"})
	// 	return
	// }
	fmt.Println("Location data is valid")
	// Set the professional ID and PID if not provided
	professional.ID = primitive.NewObjectID()
	if professional.PID == "" {
		professional.PID = primitive.NewObjectID().Hex()
	}

	// Create a context
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Insert the professional into the database
	_, err := db.ProfessionalCollection.InsertOne(ctx, professional)
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

	_, err = db.ProfessionalPaymentRecordCollection.InsertOne(ctx, paymentRecord)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Professional created but failed to create payment record"})
		return
	}

	token, err := auth.GetManagementToken()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get management token"})
		return
	}

	// TODO: replace correct id her and in supplier
	err = auth.AssignRole(professional.PID, "rol_t5Ino0H318hINBXz", token)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Professional created but failed to assign Auth0 role"})
		return
	}

	// Respond with the created professional
	c.JSON(http.StatusCreated, professional)
}

// // GetProfessionals returns all professionals whose PaymentRecord is either approved or marked as deleted
// func GetProfessionals(c *gin.Context) {
// 	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
// 	defer cancel()

// 	// Find all PaymentRecords that satisfy the condition
// 	cursor, err := db.ProfessionalPaymentRecordCollection.Find(ctx, bson.M{
// 		"$or": []bson.M{
// 			{"approved": true},
// 			{"deleted": true},
// 		},
// 	})
// 	if err != nil {
// 		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error while fetching payment records"})
// 		return
// 	}
// 	defer cursor.Close(ctx)

// 	// Collect the ProfessionalPIDs from these PaymentRecords
// 	var professionalPIDs []string
// 	for cursor.Next(ctx) {
// 		var rec model.ProfessionalPaymentRecord
// 		if err := cursor.Decode(&rec); err != nil {
// 			continue
// 		}
// 		professionalPIDs = append(professionalPIDs, rec.ProfessionalPID)
// 	}

// 	// Find professionals whose 'pid' is in the professionalPIDs list
// 	professionalsCursor, err := db.ProfessionalCollection.Find(ctx, bson.M{"pid": bson.M{"$in": professionalPIDs}})
// 	if err != nil {
// 		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error while fetching professionals"})
// 		return
// 	}
// 	defer professionalsCursor.Close(ctx)

// 	var professionals []model.Professional
// 	for professionalsCursor.Next(ctx) {
// 		var professional model.Professional
// 		if err := professionalsCursor.Decode(&professional); err != nil {
// 			continue
// 		}
// 		professionals = append(professionals, professional)
// 	}

// 	c.JSON(http.StatusOK, professionals)
// }

// GetAllProfessionals returns all Professionals, joined with their payment record.
func GetAllProfessionals(c *gin.Context) {
	log.Println("[INFO] GetAllProfessionals called")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// This aggregation pipeline is the "JOIN"
	pipeline := mongo.Pipeline{
		// Stage 1: $lookup
		{
			{Key: "$lookup", Value: bson.M{
				"from":         "professional_payment_records",
				"localField":   "pid",
				"foreignField": "professional_pid",
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
				"professional": "$$ROOT",
				"record":       "$recordArray",
			}},
		},
		// Stage 4: $project (cleanup)
		{
			{Key: "$project", Value: bson.M{
				"professional.recordArray": 0,
			}},
		},
	}

	// Run the aggregation on the SupplierCollection
	cursor, err := db.ProfessionalCollection.Aggregate(ctx, pipeline)
	if err != nil {
		log.Printf("[ERROR] Database aggregation error: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}
	defer cursor.Close(ctx)

	// Decode all results into our slice
	var professionals []model.ProfessionalWithRecord
	if err = cursor.All(ctx, &professionals); err != nil {
		log.Printf("[ERROR] Failed to decode aggregation results: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to decode data"})
		return
	}

	// Handle case where no professionals are found at all
	if professionals == nil {
		professionals = []model.ProfessionalWithRecord{}
	}

	professionalWithRecordResponse := make([]model.ProfessionalWithRecordResponse, len(professionals))

	// Loop over the database results and map them to the frontend struct
	for i, professional := range professionals {
		// Handle the 'null' records from the $unwind
		var payments []model.Payment
		if professional.Record.Payments != nil {
			payments = professional.Record.Payments
		} else {
			payments = []model.Payment{}
		}

		professionalWithRecordResponse[i] = model.ProfessionalWithRecordResponse{
			Professional: professional.Professional,
			Record: model.PaymentRecordReponse{
				ID:          professional.Record.ID,
				PID:         professional.Record.ProfessionalPID,
				Approved:    professional.Record.Approved,
				Payments:    payments,
				Deleted:     professional.Record.Deleted,
				PackageName: professional.Record.PackageName,
			},
		}
	}

	log.Printf("[INFO] GetAllProfessionals returning %d combined supplier records", len(professionalWithRecordResponse))
	c.JSON(http.StatusOK, professionalWithRecordResponse)
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

	err = db.ProfessionalCollection.FindOne(ctx, bson.M{"_id": objID}).Decode(&professional)
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

	filter := bson.M{
		"pid": pid,
	}

	var professional model.Professional
	err := db.ProfessionalCollection.FindOne(ctx, filter).Decode(&professional)

	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Professional not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error: " + err.Error()})
		}
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

	filter := bson.M{
		"email":  email,
	}

	err := db.ProfessionalCollection.FindOne(ctx, filter).Decode(&professional)
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

	_, err = db.ProfessionalCollection.UpdateOne(ctx, bson.M{"_id": objID}, bson.M{"$set": updateData})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update professional"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Professional updated successfully"})
}

func DeleteProfessional(c *gin.Context) {
	idParam := c.Param("id")
	objID, err := primitive.ObjectIDFromHex(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid professional ID"})
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// First, get the professional to retrieve their PID
	var professional model.Professional
	err = db.ProfessionalCollection.FindOne(ctx, bson.M{"_id": objID}).Decode(&professional)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Professional not found"})
		return
	}

	// Delete all projects associated with this professional's PID
	_, err = db.ProjectCollection.DeleteMany(ctx, bson.M{"pid": professional.PID})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete associated projects"})
		return
	}

	// Delete the professional
	_, err = db.ProfessionalCollection.DeleteOne(ctx, bson.M{"_id": objID})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete professional"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Professional and associated projects deleted successfully"})
}

// GetApprovedProfessionals returns all professionals in the database without payment record filtering
// This is useful for admin purposes to see all professionals regardless of approval status
func GetApprovedProfessionals(c *gin.Context) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	filter := bson.M{"status": "approved"}
	log.Printf("[DEBUG] GetApprovedProfessionals: Finding professional with filter: %v", filter)

	cursor, err := db.ProfessionalCollection.Find(ctx, filter)
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

// SearchProfessionals searches professionals by company name, description, specializations, and services offered
func SearchProfessionals(c *gin.Context) {
	searchQuery := c.Query("q")
	if searchQuery == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Search query parameter 'q' is required"})
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Create a text search filter for multiple fields
	filter := bson.M{
		"$or": []bson.M{
			{"company_name": bson.M{"$regex": searchQuery, "$options": "i"}},        // Case-insensitive search in company name
			{"company_description": bson.M{"$regex": searchQuery, "$options": "i"}}, // Case-insensitive search in description
			{"specializations": bson.M{"$regex": searchQuery, "$options": "i"}},     // Case-insensitive search in specializations array
			{"services_offered": bson.M{"$regex": searchQuery, "$options": "i"}},    // Case-insensitive search in services offered array
		},
		"status": "approved",
	}

	// Optional: Add additional filters
	if companyType := c.Query("company_type"); companyType != "" {
		filter["company_type"] = bson.M{"$regex": companyType, "$options": "i"}
	}

	if location := c.Query("location"); location != "" {
		filter["address"] = bson.M{"$regex": location, "$options": "i"}
	}

	cursor, err := db.ProfessionalCollection.Find(ctx, filter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error during search"})
		return
	}
	defer cursor.Close(ctx)

	var professionals []model.Professional
	if err = cursor.All(ctx, &professionals); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error decoding professionals"})
		return
	}

	// Return empty array instead of nil when no professionals are found
	if professionals == nil {
		professionals = []model.Professional{}
	}

	c.JSON(http.StatusOK, gin.H{
		"query":   searchQuery,
		"count":   len(professionals),
		"results": professionals,
	})
}

// GetProfessionalsWithFilters returns professionals filtered by company type and other criteria
func GetProfessionalsWithFilters(c *gin.Context) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Build filter based on query parameters
	filter := bson.M{
		"status": "approved",
	}

	// Filter by company type (professional type)
	if companyType := c.Query("company_type"); companyType != "" {
		filter["company_type"] = bson.M{"$regex": companyType, "$options": "i"} // Case-insensitive match
	}

	// Filter by location/address
	if location := c.Query("location"); location != "" {
		filter["address"] = bson.M{"$regex": location, "$options": "i"}
	}

	// Filter by year founded (range filters)
	if yearFrom := c.Query("year_from"); yearFrom != "" {
		if filter["year_founded"] == nil {
			filter["year_founded"] = bson.M{}
		}
		filter["year_founded"].(bson.M)["$gte"] = yearFrom
	}

	if yearTo := c.Query("year_to"); yearTo != "" {
		if filter["year_founded"] == nil {
			filter["year_founded"] = bson.M{}
		}
		filter["year_founded"].(bson.M)["$lte"] = yearTo
	}

	// Filter by number of employees (range filters)
	if employeesMin := c.Query("employees_min"); employeesMin != "" {
		if filter["number_of_employees"] == nil {
			filter["number_of_employees"] = bson.M{}
		}
		filter["number_of_employees"].(bson.M)["$gte"] = employeesMin
	}

	if employeesMax := c.Query("employees_max"); employeesMax != "" {
		if filter["number_of_employees"] == nil {
			filter["number_of_employees"] = bson.M{}
		}
		filter["number_of_employees"].(bson.M)["$lte"] = employeesMax
	}

	// Filter by specialization
	if specialization := c.Query("specialization"); specialization != "" {
		filter["specializations"] = bson.M{"$regex": specialization, "$options": "i"}
	}

	// Filter by service offered
	if service := c.Query("service"); service != "" {
		filter["services_offered"] = bson.M{"$regex": service, "$options": "i"}
	}

	cursor, err := db.ProfessionalCollection.Find(ctx, filter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error during filtering"})
		return
	}
	defer cursor.Close(ctx)

	var professionals []model.Professional
	if err = cursor.All(ctx, &professionals); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error decoding professionals"})
		return
	}

	// Return empty array instead of nil when no professionals are found
	if professionals == nil {
		professionals = []model.Professional{}
	}

	c.JSON(http.StatusOK, professionals)
}

// GetProfessionalTypes returns all unique company types (professional types) in the database
func GetProfessionalTypes(c *gin.Context) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Use MongoDB's distinct operation to get unique company types
	companyTypes, err := db.ProfessionalCollection.Distinct(ctx, "company_type", bson.M{})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error while fetching professional types"})
		return
	}

	// Filter out empty values
	var validTypes []interface{}
	for _, ct := range companyTypes {
		if str, ok := ct.(string); ok && str != "" {
			validTypes = append(validTypes, str)
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"professional_types": validTypes,
		"count":              len(validTypes),
	})
}

func UpdateProfessionalPaymentRecordApprovedStatus(id string, approved bool, status string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	update := bson.M{"$set": bson.M{"approved": approved}}
	_, err := db.ProfessionalPaymentRecordCollection.UpdateOne(ctx, bson.M{"professional_pid": id}, update)

	if err != nil {
		log.Printf("[ERROR] Database error while updating Approved status for professional_pid=%s: %v", id, err)
		return fmt.Errorf("database error while adding a payment status")
	}
	_, err = db.ProfessionalCollection.UpdateOne(ctx, bson.M{"pid": id}, status)
	if err != nil {
		return fmt.Errorf("database error while adding a payment status")
	}

	return nil
}

func SetProfessionalPaymentRecordPackageName(id string, packageName string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	update := bson.M{"$set": bson.M{"package_name": packageName}}
	_, err := db.ProfessionalPaymentRecordCollection.UpdateOne(ctx, bson.M{"professional_pid": id}, update)

	if err != nil {
		return fmt.Errorf("database error while adding a payment status")
	}

	return nil
}

func AppendPaymentToProfessionalPaymentRecord(id string, payment model.Payment) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	update := bson.M{"$push": bson.M{"payments": payment}}
	_, err := db.ProfessionalPaymentRecordCollection.UpdateOne(ctx, bson.M{"professional_pid": id}, update)
	if err != nil {
		return fmt.Errorf("database error while adding a payment recrod")
	}

	return nil
}
