package utils

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"material-api/model"
	"time"
)

func CreatePaymentRecord(amount int64) (model.Payment, error) {
	parsedAmount := float64(amount)
	now := time.Now()

	// Get first day of next month
	nextMonth := now.AddDate(0, 1, -now.Day()+1) // move to first day of next month

	paymentRecord := model.Payment{
		Month:       time.Date(nextMonth.Year(), nextMonth.Month(), 1, 0, 0, 0, 0, now.Location()),
		Amount:      parsedAmount,
		PaymentDate: now,
	}
	return paymentRecord, nil
}

// VerifySignature checks if the received signature matches the computed one
func VerifySignature(payload []byte, secret, receivedSig string) bool {
	// Recompute the HMAC
	h := hmac.New(sha256.New, []byte(secret))
	h.Write(payload)
	expectedSig := h.Sum(nil)

	// Decode the received hex signature
	receivedSigBytes, err := hex.DecodeString(receivedSig)
	if err != nil {
		return false // invalid signature format
	}

	// Constant-time comparison (prevents timing attacks)
	return hmac.Equal(expectedSig, receivedSigBytes)

}