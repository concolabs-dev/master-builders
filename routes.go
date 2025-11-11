package main

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"material-api/auth"
	"material-api/email"
	"material-api/handlers"
)

func RegisterRoutes(router *gin.Engine) {

	//materials routes
	router.GET("/search", handlers.SearchMaterials)
	router.GET("/materials", handlers.GetMaterials)
	router.GET("/materials/:id", handlers.GetMaterialByID)
	router.GET("/materials/filter", handlers.GetMaterialsByCategory)
	router.POST("/materials", auth.RequireRoles("admin"), handlers.CreateMaterial)
	router.PUT("/materials/:number", auth.RequireRoles("admin"), handlers.UpdateMaterial)
	router.DELETE("/materials/:id", auth.RequireRoles("admin"), handlers.DeleteMaterial)

	//types routes
	router.GET("/types", handlers.GetTypes)
	router.GET("/types/:id", handlers.GetTypeByID)
	router.POST("/types", auth.RequireRoles("admin"), handlers.CreateType)
	router.PUT("/types/:id", auth.RequireRoles("admin"), handlers.UpdateType)
	router.DELETE("/types/:id", auth.RequireRoles("admin"), handlers.DeleteType)
	router.GET("/forex", handlers.GetMajorCurrencies)

	//suppliers routes
	router.POST("/suppliers", auth.RequireOwnership("supplier"), handlers.CreateSupplier)
	// router.GET("/suppliers", handlers.GetSuppliers)
	router.GET("/suppliers-all", handlers.GetAllSuppliers)
	router.GET("/suppliers/:id", handlers.GetSupplierByID)
	router.GET("/suppliers/pid/:pid", handlers.GetSupplierByPID)
	router.GET("/suppliers/pid/napproved/:pid", handlers.GetSupplierByPPID)
	router.GET("/suppliers/email/:email", handlers.GetSupplierByEmail)
	router.PUT("/suppliers/:id", auth.RequireOwnershipOrRoles("supplier", "admin"), handlers.UpdateSupplier)
	router.DELETE("/suppliers/:id", auth.RequireOwnershipOrRoles("supplier", "admin"), handlers.DeleteSupplier)
	router.GET("/suppliers", handlers.GetApprovedSuppliers)

	//items routes
	router.GET("/items", handlers.GetItems)
	router.GET("/items/supplier/:supplierPid", handlers.GetItemsBySupplier)
	router.GET("/items/material/:materialId", handlers.GetItemsByMaterial)
	router.POST("/items", auth.RequireOwnership("item"), handlers.CreateItem)
	router.PUT("/items/:id", auth.RequireOwnership("item"), handlers.UpdateItem)
	router.DELETE("/items/:id", auth.RequireOwnership("item"), handlers.DeleteItem)

	//paymentRecords routes
	router.POST("/paymentRecords", auth.RequireRoles("admin"), handlers.CreatePaymentRecord)
	router.GET("/paymentRecords", auth.RequireOwnership("paymentRecord"), handlers.GetPaymentRecords)
	router.GET("/paymentRecords/:pid/:type", auth.RequireOwnership("paymentRecord"), handlers.GetPaymentRecordByID)
	router.PUT("/paymentRecords/:id", auth.RequireRoles("admin"), handlers.UpdatePaymentRecord)
	router.DELETE("/paymentRecords/:id", auth.RequireRoles("admin"), handlers.DeletePaymentRecord)

	//professionals routes
	router.POST("/professionals", auth.RequireOwnership("professional"), handlers.CreateProfessional)
	router.GET("/professionals", handlers.GetAllProfessionals)
	router.GET("/professionals/:id", handlers.GetProfessionalByID)
	router.GET("/professionals/pid/:pid", handlers.GetProfessionalByPID)
	router.GET("/professionals/email/:email", handlers.GetProfessionalByEmail)
	router.PUT("/professionals/:id", auth.RequireOwnership("professional"), handlers.UpdateProfessional)
	// router.DELETE("/professionals/:id", auth.RequireOwnership("professional"), handlers.DeleteProfessional)
	router.DELETE(
		"/professionals/:id",
		auth.RequireOwnershipOrRoles("professional", "admin"),
		handlers.DeleteProfessional,
	)
	router.GET("/admin/professionals/all", handlers.GetAllProfessionals)
	router.GET("/professionals/search", handlers.SearchProfessionals)         // Search professionals
	router.GET("/professionals/filter", handlers.GetProfessionalsWithFilters) // Filter professionals
	router.GET("/professionals/types", handlers.GetProfessionalTypes)         // Get all professional types
	// router.POST("/projects", handlers.CreateProject)                     // Create a new project
	// router.GET("/projects", handlers.GetProjects)                        // Get all projects
	// router.GET("/projects/professional/:pid", handlers.GetProjectsByPID) // Get projects by professional PID
	// router.GET("/projects/:id", handlers.GetProjectByID)                 // Get a project by ID
	// router.PUT("/projects/:id", handlers.UpdateProject)                  // Update a project
	// router.DELETE("/projects/:id", handlers.DeleteProject)               // Delete a project

	//materials routes
	router.POST("/projects", handlers.CreateProject)                                    // Create a new project
	router.GET("/projects", handlers.GetProjects)                                       // Get all projects
	router.GET("/projects/filter", handlers.GetProjectsWithFilters)                     // Get projects with filters
	router.GET("/projects/search", handlers.SearchProjects)                             // Search projects
	router.GET("/projects/with-professional", handlers.GetProjectsWithProfessionalInfo) // Get projects with professional info
	router.GET("/projects/professional/:pid", handlers.GetProjectsByPID)                // Get projects by professional PID
	router.GET("/projects/:id", handlers.GetProjectByID)                                // Get a project by ID
	router.PUT("/projects/:id", handlers.UpdateProject)                                 // Update a project
	router.DELETE("/projects/:id", handlers.DeleteProject)                              // Delete a project

	// Stripe webhook routes
	router.POST("/webhook", handlers.HandleWebhook)

	// Email routes
	email.RegisterRoutesEmail(router)

	// Health check route
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status": "ok",
		})
	})
}
