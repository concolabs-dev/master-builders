package handlers

import (
	"material-api/utils"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

func HandleSubscriptionNotification(c *gin.Context) {
	var req struct {
		ProfessionalPid string `json:"professional_pid"`
		StatusCode      string `json:"status_code"`
		StatusMessage   string `json:"status_message"`
		MessageType     string `json:"message_type"`
		PayHereAmount   string `json:"payhere_amount"`
	}

	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input"})
		return
	}

	// Convert status_code to int
	statusCode, err := strconv.Atoi(req.StatusCode)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid status code"})
		return
	}

	switch statusCode {
	case 2: // Success
		paymentRecord, err := utils.CreatePaymentRecord(req.PayHereAmount)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		if err := UpdateProfessionalPaymentRecordApprovedStatus(req.ProfessionalPid, true); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		if err := AppendPaymentToProfessionalPaymentRecord(req.ProfessionalPid, paymentRecord); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusCreated, gin.H{
			"message":       "Payment processed successfully",
			"statusMessage": req.StatusMessage,
			"messageType":   req.MessageType,
		})
		return

	case 0:
		c.JSON(http.StatusAccepted, gin.H{
			"message":       "Payment is pending",
			"statusMessage": req.StatusMessage,
			"messageType":   req.MessageType,
		})
		// TODO: handle the logic
		return

	case -1:
		c.JSON(http.StatusOK, gin.H{
			"message":       "Payment was cancelled by the user",
			"statusMessage": req.StatusMessage,
			"messageType":   req.MessageType,
		})
		// TODO: handle the logic
		return

	case -2:
		c.JSON(http.StatusOK, gin.H{
			"message":       "Payment failed",
			"statusMessage": req.StatusMessage,
			"messageType":   req.MessageType,
		})
		// TODO: handle the logic
		return

	case -3:
		c.JSON(http.StatusOK, gin.H{
			"message":       "Payment was chargedback",
			"statusMessage": req.StatusMessage,
			"messageType":   req.MessageType,
		})
		// TODO: handle the logic
		return

	default:
		c.JSON(http.StatusBadRequest, gin.H{
			"message":       "Unknown status code",
			"statusMessage": req.StatusMessage,
			"messageType":   req.MessageType,
		})
	}
}
