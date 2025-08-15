package handlers

import (
	"fmt"
	"io/ioutil"
	"material-api/utils"
	"net/http"
	"os"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/stripe/stripe-go/v72"
	"github.com/stripe/stripe-go/v72/charge"
	"github.com/stripe/stripe-go/v72/customer"
	"github.com/stripe/stripe-go/v72/paymentmethod"

	"github.com/stripe/stripe-go/v72/webhook"
)

func handleWebhook(w http.ResponseWriter, req *http.Request) {
	const MaxBodyBytes = int64(65536)
	req.Body = http.MaxBytesReader(w, req.Body, MaxBodyBytes)
	payload, err := ioutil.ReadAll(req.Body)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading request body: %v\n", err)
		w.WriteHeader(http.StatusServiceUnavailable)
		return
	}

	event, err := webhook.ConstructEvent(payload, req.Header.Get("Stripe-Signature"), "your_webhook_secret")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error verifying webhook signature: %v\n", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// Handle the event
	switch event.Type {
	case "payment_intent.succeeded":
		// Handle successful payment
	case "payment_intent.payment_failed":
		// Handle failed payment
	default:
		fmt.Fprintf(os.Stderr, "Unhandled event type: %s\n", event.Type)
	}

	w.WriteHeader(http.StatusOK)
}

func payemntHandler() {
	stripe.Key = "sk_test_51RuXlvHb6l5GodkUSzPb7ezXiNAGjCgjOo7kD0hh51bocrLRUwmTPkP4RdB2Z8VfTDStl0AupJohwA8hKNbAMGmM00qPUMw5rg"

	params1 := &stripe.CustomerParams{
		Email: stripe.String("customer@example.com"),
		Name:  stripe.String("Jenny Rosen"),
	}
	newCustomer, err := customer.New(params1)
	if err != nil {
		// Uh-oh, something went wrong!
		fmt.Printf("Error creating customer: %v\n", err)
		return
	}
	fmt.Printf("Success! Created customer: %s\n", newCustomer.ID)

	params := &stripe.PaymentMethodParams{
		Type: stripe.String("card"),
		Card: &stripe.PaymentMethodCardParams{
			Number:   stripe.String("4242424242424242"),
			ExpMonth: stripe.String("12"),
			ExpYear:  stripe.String("2023"),
			CVC:      stripe.String("314"),
		},
	}
	pm, err := paymentmethod.New(params)
	if err != nil {
		fmt.Printf("Error creating payment method: %v\n", err)
		return
	}

	// Attach the payment method to the customer
	attachParams := &stripe.PaymentMethodAttachParams{
		Customer: stripe.String(newCustomer.ID),
	}
	pm, err = paymentmethod.Attach(pm.ID, attachParams)
	if err != nil {
		fmt.Printf("Error attaching payment method: %v\n", err)
		return
	}

	chargeParams := &stripe.ChargeParams{
		Amount:      stripe.Int64(2000), // $20.00
		Currency:    stripe.String(string(stripe.CurrencyUSD)),
		Customer:    stripe.String(newCustomer.ID),
		Description: stripe.String("My First Test Charge (created for API docs)"),
	}
	ch, err := charge.New(chargeParams)
	if err != nil {
		fmt.Printf("Error creating charge: %v\n", err)
		return
	}
	fmt.Printf("Success! Charged: %v\n", ch.ID)
}

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
