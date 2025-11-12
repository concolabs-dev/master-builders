package handlers

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"

	"material-api/db"
	"material-api/model"
)

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
