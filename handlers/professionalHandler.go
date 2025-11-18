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
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"

	"github.com/microcosm-cc/bluemonday"
	"material-api/auth"
	"material-api/db"
	"material-api/model"
)

// CreateProfessional creates a new professional and adds a payment record
func CreateProfessional(c *gin.Context) {
	var professional model.Professional
	log.Println("[INFO] CreateProfessional: Request received")

	if err := c.ShouldBindJSON(&professional); err != nil {
		log.Printf("[ERROR] CreateProfessional: Invalid JSON binding: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid JSON data"})
		return
	}

    // Initialize policies
    // 'strict' removes ALL HTML tags (for names, titles, IDs)
    // 'ugc' allows safe HTML like <b>, <i>, <ul> (for descriptions)
    strict := bluemonday.StrictPolicy()
    ugc := bluemonday.UGCPolicy()

    // 1. Standardize Critical Fields
    professional.Email = strict.Sanitize(professional.Email)
    professional.PID = strings.TrimSpace(professional.PID)
    professional.Website = strict.Sanitize(professional.Website)

    // 2. Sanitize Single String Fields
    professional.CompanyName = strict.Sanitize(professional.CompanyName)
    professional.CompanyType = strict.Sanitize(professional.CompanyType)
    professional.Address = strict.Sanitize(professional.Address)
    professional.TelephoneNumber = strict.Sanitize(professional.TelephoneNumber)
    
    // Sanitize URLs to ensure no <script> injection, though they should be validated as URLs too
    professional.CompanyLogoUrl = strict.Sanitize(professional.CompanyLogoUrl)
    professional.CoverImageURL = strict.Sanitize(professional.CoverImageURL)

    // 3. Sanitize Rich Text Fields (Description)
    // Allows formatted text but removes malicious scripts
    professional.CompanyDescription = ugc.Sanitize(professional.CompanyDescription)

    // 4. Sanitize String Arrays (Slices)
    // We must loop through them to sanitize each item
    for i := range professional.Specializations {
        professional.Specializations[i] = strict.Sanitize(professional.Specializations[i])
    }

    for i := range professional.ServicesOffered {
        professional.ServicesOffered[i] = strict.Sanitize(professional.ServicesOffered[i])
    }

    for i := range professional.CertificationsAccreditations {
        professional.CertificationsAccreditations[i] = strict.Sanitize(professional.CertificationsAccreditations[i])
    }

	// 2. Create Context
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Duplicate Check Logic
	filter := bson.M{"email": professional.Email}
	log.Printf("[DEBUG] CreateProfessional: Checking duplicates for Email: %s", professional.Email)

	if professional.PID != "" {
		log.Printf("[DEBUG] CreateProfessional: PID provided (%s), checking Email OR PID", professional.PID)
		filter = bson.M{
			"$or": []bson.M{
				{"email": professional.Email},
				{"pid": professional.PID},
			},
		}
	}

	count, err := db.ProfessionalCollection.CountDocuments(ctx, filter)
	if err != nil {
		log.Printf("[ERROR] CreateProfessional: Database error checking duplicates: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error checking for existing user"})
		return
	}

	if count > 0 {
		log.Printf("[WARN] CreateProfessional: Duplicate user attempted (Email: %s, PID: %s)", professional.Email, professional.PID)
		c.JSON(http.StatusConflict, gin.H{"error": "Professional with this Email or PID already exists"})
		return
	}

	// ID Generation
	professional.ID = primitive.NewObjectID()
	professional.Status = "pending"
	if professional.PID == "" {
		professional.PID = primitive.NewObjectID().Hex()
	}
	log.Printf("[INFO] CreateProfessional: Generated new PID: %s", professional.PID)

	// Insert Professional
	_, err = db.ProfessionalCollection.InsertOne(ctx, professional)
	if err != nil {
		log.Printf("[ERROR] CreateProfessional: Failed to insert professional: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create professional"})
		return
	}

	// Create Payment Record
	paymentRecord := model.ProfessionalPaymentRecord{
		ID:              primitive.NewObjectID(),
		ProfessionalPID: professional.PID,
		Approved:        false,
		Payments:        []model.Payment{},
		Deleted:         false,
	}

	_, err = db.ProfessionalPaymentRecordCollection.InsertOne(ctx, paymentRecord)
	if err != nil {
		log.Printf("[ERROR] CreateProfessional: Failed to insert payment record for PID %s: %v", professional.PID, err)
		// Note: In production, consider rolling back the professional creation here
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Professional created but failed to create payment record"})
		return
	}

	// Auth0 Role Assignment
	token, err := auth.GetManagementToken()
	if err != nil {
		log.Printf("[ERROR] CreateProfessional: Failed to get Auth0 management token: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get management token"})
		return
	}

	err = auth.AssignRole(professional.PID, "rol_t5Ino0H318hINBXz", token)
	if err != nil {
		log.Printf("[ERROR] CreateProfessional: Failed to assign Auth0 role for PID %s: %v", professional.PID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Professional created but failed to assign Auth0 role"})
		return
	}

	log.Printf("[INFO] CreateProfessional: Successfully created professional %s", professional.PID)
	c.JSON(http.StatusCreated, professional)
}

// GetAllProfessionals returns all Professionals, joined with their payment record.
func GetAllProfessionals(c *gin.Context) {
	log.Println("[INFO] GetAllProfessionals: Request received")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	pipeline := mongo.Pipeline{
		{{Key: "$lookup", Value: bson.M{
			"from":         "professional_payment_records",
			"localField":   "pid",
			"foreignField": "professional_pid",
			"as":           "recordArray",
		}}},
		{{Key: "$unwind", Value: bson.M{
			"path":                       "$recordArray",
			"preserveNullAndEmptyArrays": true,
		}}},
		{{Key: "$project", Value: bson.M{
			"professional": "$$ROOT",
			"record":       "$recordArray",
		}}},
		{{Key: "$project", Value: bson.M{
			"professional.recordArray": 0,
		}}},
	}

	cursor, err := db.ProfessionalCollection.Aggregate(ctx, pipeline)
	if err != nil {
		log.Printf("[ERROR] GetAllProfessionals: Aggregation failed: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}
	defer cursor.Close(ctx)

	var professionals []model.ProfessionalWithRecord
	if err = cursor.All(ctx, &professionals); err != nil {
		log.Printf("[ERROR] GetAllProfessionals: Failed to decode results: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to decode data"})
		return
	}

	if professionals == nil {
		professionals = []model.ProfessionalWithRecord{}
	}

	professionalWithRecordResponse := make([]model.ProfessionalWithRecordResponse, len(professionals))

	for i, professional := range professionals {
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

	log.Printf("[INFO] GetAllProfessionals: Returning %d records", len(professionalWithRecordResponse))
	c.JSON(http.StatusOK, professionalWithRecordResponse)
}

// GetProfessionalByID returns a professional by its ID
func GetProfessionalByID(c *gin.Context) {
	idParam := c.Param("id")
	log.Printf("[INFO] GetProfessionalByID: Request for ID: %s", idParam)

	objID, err := primitive.ObjectIDFromHex(idParam)
	if err != nil {
		log.Printf("[WARN] GetProfessionalByID: Invalid Hex ID: %s", idParam)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid professional ID"})
		return
	}

	var professional model.Professional
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err = db.ProfessionalCollection.FindOne(ctx, bson.M{"_id": objID}).Decode(&professional)
	if err != nil {
		log.Printf("[WARN] GetProfessionalByID: Not found (ID: %s, Err: %v)", idParam, err)
		c.JSON(http.StatusNotFound, gin.H{"error": "Professional not found"})
		return
	}

	c.JSON(http.StatusOK, professional)
}

// GetProfessionalByPID returns a professional
func GetProfessionalByPID(c *gin.Context) {
	pid := c.Param("pid")
	if pid == "" {
		log.Println("[WARN] GetProfessionalByPID: PID parameter missing")
		c.JSON(http.StatusBadRequest, gin.H{"error": "Professional PID is required"})
		return
	}
	log.Printf("[INFO] GetProfessionalByPID: Request for PID: %s", pid)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var professional model.Professional
	err := db.ProfessionalCollection.FindOne(ctx, bson.M{"pid": pid}).Decode(&professional)

	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			log.Printf("[WARN] GetProfessionalByPID: Not found (PID: %s)", pid)
			c.JSON(http.StatusNotFound, gin.H{"error": "Professional not found"})
		} else {
			log.Printf("[ERROR] GetProfessionalByPID: DB error (PID: %s, Err: %v)", pid, err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error: " + err.Error()})
		}
		return
	}

	c.JSON(http.StatusOK, professional)
}

// GetProfessionalByEmail returns a professional by its email
func GetProfessionalByEmail(c *gin.Context) {
	email := c.Param("email")
	log.Printf("[INFO] GetProfessionalByEmail: Request for Email: %s", email)

	var professional model.Professional
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	filter := bson.M{
		"email": email,
	}

	err := db.ProfessionalCollection.FindOne(ctx, filter).Decode(&professional)
	if err != nil {
		log.Printf("[WARN] GetProfessionalByEmail: Not found (Email: %s)", email)
		c.JSON(http.StatusNotFound, gin.H{"error": fmt.Sprintf("Professional with email %s not found", email)})
		return
	}

	c.JSON(http.StatusOK, professional)
}

// UpdateProfessional updates an existing professional
func UpdateProfessional(c *gin.Context) {
	idParam := c.Param("id")
	log.Printf("[INFO] UpdateProfessional: Request for ID: %s", idParam)

	objID, err := primitive.ObjectIDFromHex(idParam)
	if err != nil {
		log.Printf("[WARN] UpdateProfessional: Invalid Hex ID: %s", idParam)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid professional ID"})
		return
	}

	var updateData map[string]interface{}
	if err := c.ShouldBindJSON(&updateData); err != nil {
		log.Printf("[ERROR] UpdateProfessional: Invalid JSON binding: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid JSON data"})
		return
	}

	delete(updateData, "id")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	result, err := db.ProfessionalCollection.UpdateOne(ctx, bson.M{"_id": objID}, bson.M{"$set": updateData})
	if err != nil {
		log.Printf("[ERROR] UpdateProfessional: DB Update failed: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update professional"})
		return
	}

	log.Printf("[INFO] UpdateProfessional: Success (Matched: %d, Modified: %d)", result.MatchedCount, result.ModifiedCount)
	c.JSON(http.StatusOK, gin.H{"message": "Professional updated successfully"})
}

func DeleteProfessional(c *gin.Context) {
	idParam := c.Param("id")
	log.Printf("[INFO] DeleteProfessional: Request for ID: %s", idParam)

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
		log.Printf("[WARN] DeleteProfessional: Professional to delete not found (ID: %s)", idParam)
		c.JSON(http.StatusNotFound, gin.H{"error": "Professional not found"})
		return
	}

	// Delete associated projects
	delProjects, err := db.ProjectCollection.DeleteMany(ctx, bson.M{"pid": professional.PID})
	if err != nil {
		log.Printf("[ERROR] DeleteProfessional: Failed to delete associated projects (PID: %s): %v", professional.PID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete associated projects"})
		return
	}
	log.Printf("[INFO] DeleteProfessional: Deleted %d projects for PID %s", delProjects.DeletedCount, professional.PID)

	// Delete the professional
	delProf, err := db.ProfessionalCollection.DeleteOne(ctx, bson.M{"_id": objID})
	if err != nil {
		log.Printf("[ERROR] DeleteProfessional: Failed to delete professional (ID: %s): %v", idParam, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete professional"})
		return
	}

	log.Printf("[INFO] DeleteProfessional: Successfully deleted professional (DeletedCount: %d)", delProf.DeletedCount)
	c.JSON(http.StatusOK, gin.H{"message": "Professional and associated projects deleted successfully"})
}

// GetApprovedProfessionals returns all professionals in the database with status "approved"
func GetApprovedProfessionals(c *gin.Context) {
	log.Println("[INFO] GetApprovedProfessionals: Request received")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	filter := bson.M{"status": "approved"}

	cursor, err := db.ProfessionalCollection.Find(ctx, filter)
	if err != nil {
		log.Printf("[ERROR] GetApprovedProfessionals: Find failed: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error while fetching professionals"})
		return
	}
	defer cursor.Close(ctx)

	var professionals []model.Professional
	if err = cursor.All(ctx, &professionals); err != nil {
		log.Printf("[ERROR] GetApprovedProfessionals: Decode failed: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error decoding professionals data"})
		return
	}

	if professionals == nil {
		professionals = []model.Professional{}
	}

	log.Printf("[INFO] GetApprovedProfessionals: Returning %d records", len(professionals))
	c.JSON(http.StatusOK, professionals)
}

// SearchProfessionals searches professionals by company name, description, specializations, and services offered
func SearchProfessionals(c *gin.Context) {
	searchQuery := c.Query("q")
	log.Printf("[INFO] SearchProfessionals: Query='%s'", searchQuery)

	if searchQuery == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Search query parameter 'q' is required"})
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	filter := bson.M{
		"$or": []bson.M{
			{"company_name": bson.M{"$regex": searchQuery, "$options": "i"}},
			{"company_description": bson.M{"$regex": searchQuery, "$options": "i"}},
			{"specializations": bson.M{"$regex": searchQuery, "$options": "i"}},
			{"services_offered": bson.M{"$regex": searchQuery, "$options": "i"}},
		},
		"status": "approved",
	}

	// Optional filters
	if companyType := c.Query("company_type"); companyType != "" {
		filter["company_type"] = bson.M{"$regex": companyType, "$options": "i"}
	}
	if location := c.Query("location"); location != "" {
		filter["address"] = bson.M{"$regex": location, "$options": "i"}
	}

	cursor, err := db.ProfessionalCollection.Find(ctx, filter)
	if err != nil {
		log.Printf("[ERROR] SearchProfessionals: DB Error: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error during search"})
		return
	}
	defer cursor.Close(ctx)

	var professionals []model.Professional
	if err = cursor.All(ctx, &professionals); err != nil {
		log.Printf("[ERROR] SearchProfessionals: Decode error: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error decoding professionals"})
		return
	}

	if professionals == nil {
		professionals = []model.Professional{}
	}

	log.Printf("[INFO] SearchProfessionals: Found %d results for '%s'", len(professionals), searchQuery)
	c.JSON(http.StatusOK, gin.H{
		"query":   searchQuery,
		"count":   len(professionals),
		"results": professionals,
	})
}

// GetProfessionalsWithFilters returns professionals filtered by company type and other criteria
func GetProfessionalsWithFilters(c *gin.Context) {
	log.Println("[INFO] GetProfessionalsWithFilters: Request received")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	filter := bson.M{
		"status": "approved",
	}

	// Build filter logs for debug
	debugFilter := make(map[string]string)

	if companyType := c.Query("company_type"); companyType != "" {
		filter["company_type"] = bson.M{"$regex": companyType, "$options": "i"}
		debugFilter["company_type"] = companyType
	}
	if location := c.Query("location"); location != "" {
		filter["address"] = bson.M{"$regex": location, "$options": "i"}
		debugFilter["location"] = location
	}
	// ... (years and employees logic same as before)
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
	if specialization := c.Query("specialization"); specialization != "" {
		filter["specializations"] = bson.M{"$regex": specialization, "$options": "i"}
		debugFilter["specialization"] = specialization
	}
	if service := c.Query("service"); service != "" {
		filter["services_offered"] = bson.M{"$regex": service, "$options": "i"}
		debugFilter["service"] = service
	}

	log.Printf("[DEBUG] GetProfessionalsWithFilters: Applying filters: %v", debugFilter)

	cursor, err := db.ProfessionalCollection.Find(ctx, filter)
	if err != nil {
		log.Printf("[ERROR] GetProfessionalsWithFilters: DB Error: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error during filtering"})
		return
	}
	defer cursor.Close(ctx)

	var professionals []model.Professional
	if err = cursor.All(ctx, &professionals); err != nil {
		log.Printf("[ERROR] GetProfessionalsWithFilters: Decode Error: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error decoding professionals"})
		return
	}

	if professionals == nil {
		professionals = []model.Professional{}
	}

	log.Printf("[INFO] GetProfessionalsWithFilters: Returning %d records", len(professionals))
	c.JSON(http.StatusOK, professionals)
}

// GetProfessionalTypes returns all unique company types
func GetProfessionalTypes(c *gin.Context) {
	log.Println("[INFO] GetProfessionalTypes: Request received")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	companyTypes, err := db.ProfessionalCollection.Distinct(ctx, "company_type", bson.M{})
	if err != nil {
		log.Printf("[ERROR] GetProfessionalTypes: DB Error: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error while fetching professional types"})
		return
	}

	var validTypes []interface{}
	for _, ct := range companyTypes {
		if str, ok := ct.(string); ok && str != "" {
			validTypes = append(validTypes, str)
		}
	}

	log.Printf("[INFO] GetProfessionalTypes: Found %d types", len(validTypes))
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
		log.Printf("[ERROR] UpdateProfessionalPaymentRecordApprovedStatus: Failed to update payment record (PID: %s): %v", id, err)
		return fmt.Errorf("database error while adding a payment status")
	}

	statusUpdate := bson.M{"$set": bson.M{"status": status}}
	_, err = db.ProfessionalCollection.UpdateOne(ctx, bson.M{"pid": id}, statusUpdate)
	if err != nil {
		log.Printf("[ERROR] UpdateProfessionalPaymentRecordApprovedStatus: Failed to update professional status (PID: %s): %v", id, err)
		return fmt.Errorf("database error while adding a payment status")
	}

	log.Printf("[INFO] UpdateProfessionalPaymentRecordApprovedStatus: Success for PID %s (Approved: %v, Status: %s)", id, approved, status)
	return nil
}

func SetProfessionalPaymentRecordPackageName(id string, packageName string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	update := bson.M{"$set": bson.M{"package_name": packageName}}
	_, err := db.ProfessionalPaymentRecordCollection.UpdateOne(ctx, bson.M{"professional_pid": id}, update)

	if err != nil {
		log.Printf("[ERROR] SetProfessionalPaymentRecordPackageName: Failed (PID: %s): %v", id, err)
		return fmt.Errorf("database error while adding a payment status")
	}

	log.Printf("[INFO] SetProfessionalPaymentRecordPackageName: Updated package name to '%s' for PID %s", packageName, id)
	return nil
}

func AppendPaymentToProfessionalPaymentRecord(id string, payment model.Payment) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	update := bson.M{"$push": bson.M{"payments": payment}}
	_, err := db.ProfessionalPaymentRecordCollection.UpdateOne(ctx, bson.M{"professional_pid": id}, update)
	if err != nil {
		log.Printf("[ERROR] AppendPaymentToProfessionalPaymentRecord: Failed (PID: %s): %v", id, err)
		return fmt.Errorf("database error while adding a payment record")
	}

	log.Printf("[INFO] AppendPaymentToProfessionalPaymentRecord: Payment appended for PID %s", id)
	return nil
}
