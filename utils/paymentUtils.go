package utils

import (
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
