package utils

import (
	"fmt"
	"material-api/model"
	"strconv"
	"time"
)

func CreatePaymentRecord(amount string) (model.Payment, error) {
	parsedAmount, err := strconv.ParseFloat(amount, 64)
	if err != nil {
		return model.Payment{}, fmt.Errorf("invalid amount format")
	}

	now := time.Now()
	paymentMonth := time.Date(now.Year(), now.Month()+1, 1, 0, 0, 0, 0, now.Location())

	paymentRecord := model.Payment{
		Month:       paymentMonth,
		Amount:      parsedAmount,
		PaymentDate: now,
	}
	return paymentRecord, nil
}