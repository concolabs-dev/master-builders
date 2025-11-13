package model

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Payment struct {
	Month       time.Time `bson:"Month" json:"Month"`
	Amount      float64   `bson:"Amount" json:"Amount"`
	PaymentDate time.Time `bson:"paymentDate" json:"paymentDate"`
}

type PaymentRecordReponse struct {
	ID          primitive.ObjectID `json:"id,omitempty"`
	PID         string             `json:"pid"`
	Approved    bool               `json:"approved"`
	Payments    []Payment          `json:"payments"`
	Deleted     bool               `json:"deleted"`
	PackageName string             `json:"package_name"`
}

type TransactionWebhook struct {
	Type          string    `json:"type"           validate:"required"`
	TransactionID string    `json:"transaction_id" validate:"required"`
	Status        string    `json:"status"         validate:"required"`
	UserID        string    `json:"puid"           validate:"required"`
	PackageName   string    `json:"package_name"   validate:"required"`
	Timestamp     time.Time `json:"timestamp"      validate:"required"`
	Amount        int64     `json:"amount"         validate:"required"`
}

// Use PaymentRecordReponse insted of this when use payment response
type PaymentRecord struct {
	ID          primitive.ObjectID `bson:"_id,omitempty" json:"id,omitempty"`
	SupplierPID string             `bson:"Supplierpid" json:"Supplierpid"`
	Approved    bool               `bson:"Approved" json:"Approved"`
	Payments    []Payment          `bson:"Payments" json:"Payments"`
	Deleted     bool               `bson:"Deleted" json:"Deleted"`
	PackageName string             `bson:"package_name" json:"package_name"`
}

type ErrorItem struct {
	Field   string      `json:"field,omitempty"`
	Code    string      `json:"code"`
	Message string      `json:"message"`
	Detail  interface{} `json:"detail,omitempty"`
}