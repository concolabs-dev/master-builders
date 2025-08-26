package handlers

import (
	"fmt"
	"io"
	"material-api/utils"
	"net/http"

	"os"

	"github.com/gin-gonic/gin"
	"github.com/goccy/go-json"
	"github.com/stripe/stripe-go/v82"
	"github.com/stripe/stripe-go/v82/checkout/session"
	"github.com/stripe/stripe-go/v82/webhook"
)

func PaymentHandler(c *gin.Context) {

	var body struct {
		PriceID string `json:"priceId"`
		PID     string `json:"pid"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	stripe.Key = os.Getenv("STRIPE_SECRET_KEY")

	params := &stripe.CheckoutSessionParams{
		PaymentMethodTypes: stripe.StringSlice([]string{"card"}),
		LineItems: []*stripe.CheckoutSessionLineItemParams{
			{
				Price:    stripe.String(body.PriceID),
				Quantity: stripe.Int64(1),
			},
		},
		Mode:       stripe.String("subscription"),
		SuccessURL: stripe.String(os.Getenv("NEXT_FRONTEND_URL") + "register/success"),
		CancelURL:  stripe.String(os.Getenv("NEXT_FRONTEND_URL")),
		Metadata: map[string]string{
			"pid":     body.PID,
			"priceId": body.PriceID,
		},
	}

	s, err := session.New(params)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"result": s, "ok": true})
}

func HandleWebhook(c *gin.Context) {

	const MaxBodyBytes = int64(65536)
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, MaxBodyBytes)

	// Read request body
	payload, err := io.ReadAll(c.Request.Body)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading request body: %v\n", err)
		c.Status(http.StatusServiceUnavailable)
		return
	}

	endpointSecret := os.Getenv("STRIPE_WEBHOOK_SECRET_KEY")
	event, err := webhook.ConstructEvent(payload, c.Request.Header.Get("Stripe-Signature"),
		endpointSecret)

	if err != nil {
		fmt.Fprintf(os.Stderr, "Error verifying webhook signature: %v\n", err)
		c.Status(http.StatusBadRequest)
		return
	}

	if err := json.Unmarshal(payload, &event); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to parse webhook body json: %v\n", err)
		c.Status(http.StatusBadRequest)
		return
	}

	// Handle events by type
	switch event.Type {
	case "charge.succeeded":
		var paymentIntent stripe.PaymentIntent
		if err := json.Unmarshal(event.Data.Raw, &paymentIntent); err != nil {
			fmt.Fprintf(os.Stderr, "Error parsing webhook JSON: %v\n", err)
			c.Status(http.StatusBadRequest)
			return
		}

		paymentRecord, err := utils.CreatePaymentRecord(paymentIntent.Amount)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		if err := SetProfessionalPaymentRecordPackageName(paymentIntent.Metadata["pid"], paymentIntent.Metadata["packageName"]); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		if err := UpdateProfessionalPaymentRecordApprovedStatus(paymentIntent.Metadata["pid"], true); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		if err := AppendPaymentToProfessionalPaymentRecord(paymentIntent.Metadata["pid"], paymentRecord); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

	case "payment_method.attached":
		var paymentMethod stripe.PaymentMethod
		if err := json.Unmarshal(event.Data.Raw, &paymentMethod); err != nil {
			fmt.Fprintf(os.Stderr, "Error parsing webhook JSON: %v\n", err)
			c.Status(http.StatusBadRequest)
			return
		}
		fmt.Println("PaymentMethod attached:", paymentMethod.ID)

	default:
		fmt.Fprintf(os.Stderr, "Unhandled event type: %s\n", event.Type)
	}

	c.Status(http.StatusOK)
}
