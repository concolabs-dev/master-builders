package handlers

import (
	"bytes"
	"encoding/json"
	"errors"
	"log"

	"fmt"
	"io"
	"material-api/utils"
	"net/http"

	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
)

const MaxBodyBytes int64 = 64 << 10 // 65536

type TransactionWebhook struct {
	Type          string    `json:"type"           validate:"required"`
	TransactionID string    `json:"transaction_id" validate:"required"`
	Status        string    `json:"status"         validate:"required"`
	UserID        string    `json:"puid"           validate:"required"`
	PackageName   string    `json:"package_name"   validate:"required"`
	Timestamp     time.Time `json:"timestamp"      validate:"required"`
	Amount        int64     `json:"amount"         validate:"required"`
}

type errorItem struct {
	Field   string      `json:"field,omitempty"`
	Code    string      `json:"code"`
	Message string      `json:"message"`
	Detail  interface{} `json:"detail,omitempty"`
}

func respondError(c *gin.Context, status int, code, msg string, detail interface{}) {
	c.JSON(status, gin.H{
		"error": errorItem{
			Code:    code,
			Message: msg,
			Detail:  detail,
		},
		"time": time.Now().UTC(),
	})
}

func HandleWebhook(c *gin.Context) {
	start := time.Now()
	reqID := c.GetHeader("X-Request-ID")
	log.Printf("HandleWebhook start requestId=%s contentType=%s", reqID, c.GetHeader("Content-Type"))
	defer func() {
		log.Printf("HandleWebhook end requestId=%s status=%d duration=%s", reqID, c.Writer.Status(), time.Since(start))
	}()
	// 1) Basic guards
	ct := c.GetHeader("Content-Type")
	if !strings.HasPrefix(strings.ToLower(ct), "application/json") {
		log.Printf("requestId=%s error=unsupported_media_type got=%s", reqID, ct)
		respondError(c, http.StatusUnsupportedMediaType, "unsupported_media_type",
			"Content-Type must be application/json", gin.H{"got": ct})
		return
	}

	secret := os.Getenv("SIGNING_SECRET")
	if secret == "" {
		log.Printf("requestId=%s error=server_misconfigured msg=signing_secret_missing", reqID)
		respondError(c, http.StatusInternalServerError, "server_misconfigured",
			"Signing secret is not configured", nil)
		return
	}

	sig := c.GetHeader("X-Webhook-Signature")
	if sig == "" {
		log.Printf("requestId=%s error=missing_signature msg=signature_header_required", reqID)
		respondError(c, http.StatusBadRequest, "missing_signature",
			"X-Webhook-Signature header is required", nil)
		return
	}

	// 2) Read raw body (max size)
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, MaxBodyBytes)
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		log.Printf("requestId=%s error=read_body_failed detail=%v", reqID, err)
		respondError(c, http.StatusBadRequest, "read_body_failed",
			"Could not read request body", err.Error())
		return
	}
	// If anything else needs to read the body later:
	c.Request.Body = io.NopCloser(bytes.NewBuffer(body))

	// 3) Verify signature
	if !utils.VerifySignature(body, secret, sig) {
		log.Printf("requestId=%s error=invalid_signature", reqID)
		respondError(c, http.StatusUnauthorized, "invalid_signature",
			"The signature header is missing or invalid", nil)
		return
	}

	// 4) Decode JSON with helpful errors
	var req TransactionWebhook
	if err := json.Unmarshal(body, &req); err != nil {
		var ute *json.UnmarshalTypeError
		var se *json.SyntaxError
		switch {
		case errors.As(err, &ute):
			log.Printf("requestId=%s error=invalid_field_type field=%s expected=%s offset=%d", reqID, ute.Field, ute.Type.String(), ute.Offset)
			respondError(c, http.StatusBadRequest, "invalid_field_type",
				fmt.Sprintf("Field %q has wrong type", ute.Field),
				gin.H{"expected": ute.Type.String(), "offset": ute.Offset})
			return
		case errors.As(err, &se):
			log.Printf("requestId=%s error=malformed_json offset=%d", reqID, se.Offset)
			respondError(c, http.StatusBadRequest, "malformed_json",
				"Malformed JSON payload", gin.H{"offset": se.Offset})
			return
		default:
			log.Printf("requestId=%s error=json_unmarshal_failed detail=%v", reqID, err)
			respondError(c, http.StatusBadRequest, "json_unmarshal_failed",
				"Unable to parse JSON payload", err.Error())
			return
		}
	}

	// 5) Validate required fields (in addition to JSON parsing)
	validate := validator.New(validator.WithRequiredStructEnabled())
	if err := validate.Struct(&req); err != nil {
		var verrs validator.ValidationErrors
		if errors.As(err, &verrs) {
			log.Printf("requestId=%s error=validation_failed count=%d", reqID, len(verrs))
			out := make([]errorItem, 0, len(verrs))
			for _, fe := range verrs {
				out = append(out, errorItem{
					Field:   fe.Field(),
					Code:    fe.Tag(),
					Message: fmt.Sprintf("%s is %s", fe.Field(), fe.Tag()),
				})
			}
			c.JSON(http.StatusBadRequest, gin.H{"errors": out, "time": time.Now().UTC()})
			return
		}
		log.Printf("requestId=%s error=validation_failed detail=%v", reqID, err)
		respondError(c, http.StatusBadRequest, "validation_failed", "Invalid payload", err.Error())
		return
	}

	// 6) Route by event type
	switch req.Type {
	case "invoice.paid":
		log.Printf("requestId=%s event=invoice.paid transactionId=%s userId=%s package=%s amount=%d", reqID, req.TransactionID, req.UserID, req.PackageName, req.Amount)

		paymentRecord, err := utils.CreatePaymentRecord(req.Amount)
		if err != nil {
			log.Printf("requestId=%s error=create_payment_record_failed detail=%v", reqID, err)
			respondError(c, http.StatusInternalServerError, "create_payment_record_failed",
				"Could not create payment record", err.Error())
			return
		}
		log.Printf("requestId=%s info=payment_record_created amount=%d", reqID, req.Amount)

		if err := SetProfessionalPaymentRecordPackageName(req.UserID, req.PackageName); err != nil {
			log.Printf("requestId=%s error=set_package_failed userId=%s package=%s detail=%v", reqID, req.UserID, req.PackageName, err)
			respondError(c, http.StatusInternalServerError, "set_package_failed",
				"Could not set package name on payment record", err.Error())
			return
		}
		log.Printf("requestId=%s info=package_set userId=%s package=%s", reqID, req.UserID, req.PackageName)

		if err := UpdateProfessionalPaymentRecordApprovedStatus(req.UserID, true); err != nil {
			log.Printf("requestId=%s error=approve_status_failed userId=%s approved=true detail=%v", reqID, req.UserID, err)
			respondError(c, http.StatusInternalServerError, "approve_status_failed",
				"Could not update approved status", err.Error())
			return
		}
		log.Printf("requestId=%s info=approved_status_updated userId=%s approved=true", reqID, req.UserID)

		if err := AppendPaymentToProfessionalPaymentRecord(req.UserID, paymentRecord); err != nil {
			log.Printf("requestId=%s error=append_payment_failed userId=%s detail=%v", reqID, req.UserID, err)
			respondError(c, http.StatusInternalServerError, "append_payment_failed",
				"Could not append payment to record", err.Error())
			return
		}
		log.Printf("requestId=%s info=payment_appended userId=%s", reqID, req.UserID)

	case "invoice.payment_failed":
		if err := UpdateProfessionalPaymentRecordApprovedStatus(req.UserID, false); err != nil {
			log.Printf("requestId=%s error=approve_status_failed userId=%s approved=false detail=%v", reqID, req.UserID, err)
			respondError(c, http.StatusInternalServerError, "approve_status_failed",
				"Could not update approved status", err.Error())
			return
		}
		log.Printf("requestId=%s event=invoice.payment_failed transactionId=%s userId=%s", reqID, req.TransactionID, req.UserID)

	case "transaction.status_updated":
		if err := UpdateProfessionalPaymentRecordApprovedStatus(req.UserID, false); err != nil {
			log.Printf("requestId=%s error=approve_status_failed userId=%s approved=false detail=%v", reqID, req.UserID, err)
			respondError(c, http.StatusInternalServerError, "approve_status_failed",
				"Could not update approved status", err.Error())
			return
		}
		log.Printf("requestId=%s event=transaction.status_updated transactionId=%s userId=%s", reqID, req.TransactionID, req.UserID)

	default:
		log.Printf("requestId=%s error=unhandled_event_type type=%s", reqID, req.Type)
		respondError(c, http.StatusBadRequest, "unhandled_event_type",
			"Event type is not supported", gin.H{"type": req.Type})
		return
	}

	// 7) Success (you can return a body if desired)
	c.JSON(http.StatusOK, gin.H{
		"status":    "ok",
		"processed": req.Type,
		"time":      time.Now().UTC(),
	})
	log.Printf("requestId=%s success processed=%s", reqID, req.Type)
}
