package handlers

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"

	// Import your models package or adjust as needed
	"material-api/auth"
	"material-api/db"
	"material-api/model"
)

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

// GetProfessionals returns all professionals whose PaymentRecord is either approved or marked as deleted
func GetProfessionals(c *gin.Context) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Find all PaymentRecords that satisfy the condition
	cursor, err := db.ProfessionalPaymentRecordCollection.Find(ctx, bson.M{
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
	professionalsCursor, err := db.ProfessionalCollection.Find(ctx, bson.M{"pid": bson.M{"$in": professionalPIDs}})
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
	err := db.ProfessionalCollection.FindOne(ctx, bson.M{"pid": pid}).Decode(&professional)
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

	err := db.ProfessionalCollection.FindOne(ctx, bson.M{"email": email}).Decode(&professional)
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

// GetAllProfessionals returns all professionals in the database without payment record filtering
// This is useful for admin purposes to see all professionals regardless of approval status
func GetAllProfessionals(c *gin.Context) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cursor, err := db.ProfessionalCollection.Find(ctx, bson.M{})
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

// projects handlers
func CreateProject(c *gin.Context) {
	var project model.Project

	// Parse JSON body
	if err := c.ShouldBindJSON(&project); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid JSON data"})
		return
	}

	// Validate PID
	if project.PID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "PID is required"})
		return
	}

	// Set a new ObjectID for the project
	project.ID = primitive.NewObjectID()

	// Insert the project into the database
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, err := db.ProjectCollection.InsertOne(ctx, project)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create project"})
		return
	}

	c.JSON(http.StatusCreated, project)
}
func GetProjects(c *gin.Context) {
	var projects []model.Project
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cursor, err := db.ProjectCollection.Find(ctx, bson.M{})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}
	defer cursor.Close(ctx)

	for cursor.Next(ctx) {
		var project model.Project
		if err := cursor.Decode(&project); err != nil {
			continue // Skip problematic entries
		}
		projects = append(projects, project)
	}

	c.JSON(http.StatusOK, projects)
}

func GetProjectsByPID(c *gin.Context) {
	pid := c.Param("pid")

	var projects []model.Project
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cursor, err := db.ProjectCollection.Find(ctx, bson.M{"pid": pid})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}
	defer cursor.Close(ctx)

	for cursor.Next(ctx) {
		var project model.Project
		if err := cursor.Decode(&project); err != nil {
			continue // Skip problematic entries
		}
		projects = append(projects, project)
	}

	c.JSON(http.StatusOK, projects)
}

func GetProjectByID(c *gin.Context) {
	idParam := c.Param("id")
	objID, err := primitive.ObjectIDFromHex(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid project ID"})
		return
	}

	var project model.Project
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err = db.ProjectCollection.FindOne(ctx, bson.M{"_id": objID}).Decode(&project)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Project not found"})
		return
	}

	c.JSON(http.StatusOK, project)
}

func UpdateProject(c *gin.Context) {
	idParam := c.Param("id")
	objID, err := primitive.ObjectIDFromHex(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid project ID"})
		return
	}

	var updateData map[string]interface{}
	if err := c.ShouldBindJSON(&updateData); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid JSON data"})
		return
	}

	// Remove the "id" field if present
	delete(updateData, "id")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	update := bson.M{"$set": updateData}
	_, err = db.ProjectCollection.UpdateOne(ctx, bson.M{"_id": objID}, update)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update project"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Project updated successfully"})
}

func DeleteProject(c *gin.Context) {
	idParam := c.Param("id")
	objID, err := primitive.ObjectIDFromHex(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid project ID"})
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err = db.ProjectCollection.DeleteOne(ctx, bson.M{"_id": objID})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete project"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Project deleted successfully"})
}

func GetProjectsWithFilters(c *gin.Context) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Build filter based on query parameters
	filter := bson.M{}

	// Filter by professional PID
	if pid := c.Query("pid"); pid != "" {
		filter["pid"] = pid
	}

	// Filter by year
	if year := c.Query("year"); year != "" {
		filter["year"] = year
	}

	// Filter by professional company type (requires lookup)
	companyType := c.Query("company_type")

	var projects []model.Project

	if companyType != "" {
		// Use aggregation pipeline to join with professionals collection
		pipeline := []bson.M{
			// Match projects with current filters
			{"$match": filter},
			// Lookup professional data
			{
				"$lookup": bson.M{
					"from":         "professionals",
					"localField":   "pid",
					"foreignField": "pid",
					"as":           "professional",
				},
			},
			// Unwind the professional array
			{"$unwind": "$professional"},
			// Match by company type
			{"$match": bson.M{"professional.company_type": companyType}},
			// Remove the professional field from output
			{"$project": bson.M{
				"professional": 0,
			}},
		}

		cursor, err := db.ProjectCollection.Aggregate(ctx, pipeline)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error during aggregation"})
			return
		}
		defer cursor.Close(ctx)

		if err = cursor.All(ctx, &projects); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Error decoding projects"})
			return
		}
	} else {
		// Simple find without company type filter
		cursor, err := db.ProjectCollection.Find(ctx, filter)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
			return
		}
		defer cursor.Close(ctx)

		if err = cursor.All(ctx, &projects); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Error decoding projects"})
			return
		}
	}

	// Return empty array instead of nil when no projects are found
	if projects == nil {
		projects = []model.Project{}
	}

	c.JSON(http.StatusOK, projects)
}

// SearchProjects searches projects by name or description
func SearchProjects(c *gin.Context) {
	searchQuery := c.Query("q")
	if searchQuery == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Search query parameter 'q' is required"})
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Create a text search filter for name and description
	filter := bson.M{
		"$or": []bson.M{
			{"name": bson.M{"$regex": searchQuery, "$options": "i"}},        // Case-insensitive search in name
			{"description": bson.M{"$regex": searchQuery, "$options": "i"}}, // Case-insensitive search in description
		},
	}

	// Optional: Add additional filters
	if pid := c.Query("pid"); pid != "" {
		filter["pid"] = pid
	}

	if year := c.Query("year"); year != "" {
		filter["year"] = year
	}

	if projectType := c.Query("type"); projectType != "" {
		filter["type"] = bson.M{"$regex": projectType, "$options": "i"}
	}

	cursor, err := db.ProjectCollection.Find(ctx, filter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error during search"})
		return
	}
	defer cursor.Close(ctx)

	var projects []model.Project
	if err = cursor.All(ctx, &projects); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error decoding projects"})
		return
	}

	// Return empty array instead of nil when no projects are found
	if projects == nil {
		projects = []model.Project{}
	}

	c.JSON(http.StatusOK, gin.H{
		"query":   searchQuery,
		"count":   len(projects),
		"results": projects,
	})
}

// GetProjectsWithProfessionalInfo returns projects with their associated professional information
func GetProjectsWithProfessionalInfo(c *gin.Context) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Build aggregation pipeline to include professional data
	pipeline := []bson.M{
		// Lookup professional data
		{
			"$lookup": bson.M{
				"from":         "professionals",
				"localField":   "pid",
				"foreignField": "pid",
				"as":           "professional",
			},
		},
		// Unwind the professional array (convert from array to object)
		{
			"$unwind": bson.M{
				"path":                       "$professional",
				"preserveNullAndEmptyArrays": true, // Keep projects even if no professional found
			},
		},
	}

	// Add filters if provided
	matchStage := bson.M{}

	if pid := c.Query("pid"); pid != "" {
		matchStage["pid"] = pid
	}

	if year := c.Query("year"); year != "" {
		matchStage["year"] = year
	}

	if companyType := c.Query("company_type"); companyType != "" {
		matchStage["professional.company_type"] = companyType
	}

	if len(matchStage) > 0 {
		pipeline = append(pipeline, bson.M{"$match": matchStage})
	}

	// Add projection stage to transform _id to id
	pipeline = append(pipeline, bson.M{
		"$project": bson.M{
			"id":          "$_id",
			"_id":         0, // Remove _id field
			"name":        1,
			"type":        1,
			"location":    1,
			"year":        1,
			"description": 1,
			"images":      1,
			"pid":         1,
			"professional": bson.M{
				"id":                            "$professional._id",
				"company_name":                  "$professional.company_name",
				"company_type":                  "$professional.company_type",
				"company_description":           "$professional.company_description",
				"year_founded":                  "$professional.year_founded",
				"number_of_employees":           "$professional.number_of_employees",
				"email":                         "$professional.email",
				"telephone_number":              "$professional.telephone_number",
				"website":                       "$professional.website",
				"address":                       "$professional.address",
				"location":                      "$professional.location",
				"specializations":               "$professional.specializations",
				"services_offered":              "$professional.services_offered",
				"certifications_accreditations": "$professional.certifications_accreditations",
				"company_logo_url":              "$professional.company_logo_url",
				"cover_image_url":               "$professional.cover_image_url",
				"pid":                           "$professional.pid",
			},
		},
	})

	cursor, err := db.ProjectCollection.Aggregate(ctx, pipeline)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error during aggregation"})
		return
	}
	defer cursor.Close(ctx)

	var results []bson.M
	if err = cursor.All(ctx, &results); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error decoding results"})
		return
	}

	// Return empty array instead of nil when no results are found
	if results == nil {
		results = []bson.M{}
	}

	c.JSON(http.StatusOK, results)
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
	}

	// Optional: Add additional filters
	if companyType := c.Query("company_type"); companyType != "" {
		filter["company_type"] = bson.M{"$regex": companyType, "$options": "i"}
	}

	if location := c.Query("location"); location != "" {
		filter["address"] = bson.M{"$regex": location, "$options": "i"}
	}

	cursor, err := professionalCollection.Find(ctx, filter)
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
	filter := bson.M{}

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

	cursor, err := professionalCollection.Find(ctx, filter)
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
	companyTypes, err := professionalCollection.Distinct(ctx, "company_type", bson.M{})
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
