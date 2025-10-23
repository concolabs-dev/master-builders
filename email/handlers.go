package email

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// RegisterRoutesEmail registers all email-related routes to the provided router
func RegisterRoutesEmail(router *gin.Engine) {
	router.GET("/send-test-email", SendTestEmail)
	router.POST("/send-email", SendCustomEmail)
}

// SendTestEmail sends a test email based on the specified type
func SendTestEmail(c *gin.Context) {
	to := []string{c.Query("email")}
	if len(to) == 0 || to[0] == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Email address is required"})
		return
	}

	emailType := c.Query("type")

	var err error

	switch emailType {
	case "general":
		err = SendGeneralMessage(
			to,
			"Welcome to Build Market LK",
			"Thank you for registering with Build Market LK. We're excited to have you on board!",
			"Valued Customer",
		)
	case "bill":
		// Sample bill data
		billData := map[string]interface{}{
			"InvoiceNumber": "INV-2025-001",
			"InvoiceDate":   time.Now().Format("Jan 2, 2006"),
			"ClientName":    "John Doe",
			"DueDate":       time.Now().AddDate(0, 0, 30).Format("Jan 2, 2006"),
			"Items": []map[string]string{
				{
					"Name":        "Construction Material",
					"Description": "Premium quality cement",
					"Quantity":    "10",
					"UnitPrice":   "LKR 1,200.00",
					"Total":       "LKR 12,000.00",
				},
				{
					"Name":        "Sand",
					"Description": "Fine sand for construction",
					"Quantity":    "5",
					"UnitPrice":   "LKR 2,500.00",
					"Total":       "LKR 12,500.00",
				},
			},
			"Subtotal":       "LKR 24,500.00",
			"TaxRate":        "15",
			"TaxAmount":      "LKR 3,675.00",
			"TotalAmount":    "LKR 28,175.00",
			"PaymentDetails": "Please make payment to Account No: 123456789 at Bank of Ceylon",
			"Note":           "Payment is due within 30 days. Late payments are subject to a 2% fee.",
			"PaymentLink":    "https://buildmarketlk.com/pay/INV-2025-001",
		}

		err = SendBill(to, "Your Invoice from Build Market LK", billData)
	case "payment":
		// Sample payment data
		paymentData := map[string]interface{}{
			"ReceiptNumber": "REC-2025-001",
			"PaymentDate":   time.Now().Format("Jan 2, 2006"),
			"ClientName":    "John Doe",
			"InvoiceNumber": "INV-2025-001",
			"AmountPaid":    "LKR 28,175.00",
			"PaymentMethod": "Bank Transfer",
			"TransactionID": "TRX123456789",
			"PaymentStatus": "Completed",
			"AccountLink":   "https://buildmarketlk.com/account",
		}

		err = SendPaymentSlip(to, "Payment Receipt from Build Market LK", paymentData)
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid email type. Use 'general', 'bill', or 'payment'"})
		return
	}

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to send email: %v", err)})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Email sent successfully"})
}

// CustomEmailRequest is the structure for sending custom emails
type CustomEmailRequest struct {
	To        []string               `json:"to" binding:"required"`
	Subject   string                 `json:"subject" binding:"required"`
	Type      string                 `json:"type" binding:"required"`
	Data      map[string]interface{} `json:"data,omitempty"`
	Message   string                 `json:"message,omitempty"`
	Recipient string                 `json:"recipient,omitempty"`
}

// SendCustomEmail handles requests to send custom emails
func SendCustomEmail(c *gin.Context) {
	var req CustomEmailRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var err error

	switch req.Type {
	case "general":
		message := req.Message
		recipient := req.Recipient
		if message == "" {
			message = "Thank you for using Build Market LK."
		}
		if recipient == "" {
			recipient = "Valued Customer"
		}
		err = SendGeneralMessage(req.To, req.Subject, message, recipient)

	case "bill":
		err = SendBill(req.To, req.Subject, req.Data)

	case "payment":
		err = SendPaymentSlip(req.To, req.Subject, req.Data)

	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid email type. Use 'general', 'bill', or 'payment'"})
		return
	}

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to send email: %v", err)})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Email sent successfully"})
}

// SendSupplierWelcomeEmail sends a welcome email to a new supplier
func SendSupplierWelcomeEmail(c *gin.Context) {
	var req struct {
		Email string `json:"email" binding:"required,email"`
		Name  string `json:"name" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	to := []string{req.Email}
	subject := "Welcome to Build Market LK Supplier Network"
	message := "Thank you for registering as a supplier with Build Market LK. Your account has been created successfully. Our team will review your information and get back to you shortly."

	err := SendGeneralMessage(to, subject, message, req.Name)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to send email: %v", err)})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Welcome email sent successfully"})
}

// SendProfessionalWelcomeEmail sends a welcome email to a new professional
func SendProfessionalWelcomeEmail(c *gin.Context) {
	var req struct {
		Email string `json:"email" binding:"required,email"`
		Name  string `json:"name" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	to := []string{req.Email}
	subject := "Welcome to Build Market LK Professional Network"
	message := "Thank you for registering as a professional with Build Market LK. Your account has been created successfully. You can now start creating projects and connecting with suppliers."

	err := SendGeneralMessage(to, subject, message, req.Name)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to send email: %v", err)})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Welcome email sent successfully"})
}

// SendOrderConfirmationEmail sends an order confirmation email
func SendOrderConfirmationEmail(c *gin.Context) {
	var req struct {
		Email       string                 `json:"email" binding:"required,email"`
		Name        string                 `json:"name" binding:"required"`
		OrderNumber string                 `json:"orderNumber" binding:"required"`
		OrderData   map[string]interface{} `json:"orderData" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	to := []string{req.Email}
	subject := "Order Confirmation - Build Market LK"

	// Add additional data to the order data
	orderData := req.OrderData
	orderData["ClientName"] = req.Name
	orderData["OrderNumber"] = req.OrderNumber
	orderData["OrderDate"] = time.Now().Format("Jan 2, 2006")

	err := SendBill(to, subject, orderData)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to send email: %v", err)})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Order confirmation email sent successfully"})
}
