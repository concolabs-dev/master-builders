package db

import "go.mongodb.org/mongo-driver/mongo"

var (
	Collection                          *mongo.Collection
	TypeCollection                      *mongo.Collection
	ExchangeRateCollection              *mongo.Collection
	SupplierCollection                  *mongo.Collection
	ItemCollection                      *mongo.Collection
	PaymentRecordCollection             *mongo.Collection
	ProfessionalCollection              *mongo.Collection
	ProfessionalPaymentRecordCollection *mongo.Collection
	ProjectCollection                   *mongo.Collection
)
