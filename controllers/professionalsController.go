package controllers

import (
	"github.com/gin-gonic/gin"
	paymentHandler "material-api/handlers"
)


func RegisterProfessionalsRoutes(router *gin.Engine) {
	router.POST("/payment/notify", paymentHandler.HandleSubscriptionNotification)
}