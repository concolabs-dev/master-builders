package controllers

import (
	"github.com/gin-gonic/gin"
	professionalHandler "material-api/handlers"
)

func RegisterPaymentRoutes(router *gin.Engine) {
		router.POST("/professionals", professionalHandler.CreateProfessional)
}
