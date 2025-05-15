package model

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Category struct
type Category struct {
	Category       string  `bson:"Category" json:"Category"`
	Subcategory    *string `bson:"Subcategory,omitempty" json:"Subcategory,omitempty"`
	SubSubcategory *string `bson:"SubSubcategory,omitempty" json:"SubSubcategory,omitempty"`
}

// Material struct
type Material struct {
	ID       primitive.ObjectID `bson:"_id,omitempty" json:"id,omitempty"`
	Number   string             `bson:"Number" json:"Number"`
	Name     string             `bson:"Name" json:"Name"`
	Type     string             `bson:"Type" json:"Type"`
	Category Category           `bson:"Category" json:"Category"`
	Qty      int                `bson:"Qty,omitempty" json:"Qty,omitempty"`
	Unit     string             `bson:"Unit,omitempty" json:"Unit,omitempty"`
	Prices   [][]interface{}    `bson:"Prices" json:"Prices"` // List of [date, price]
	Source   *string            `bson:"Source,omitempty" json:"Source,omitempty"`
	Items    []MaterialItem     `bson:"Items,omitempty" json:"Items,omitempty"` // New field: list of items
}
type MaterialItem struct {
	ID   primitive.ObjectID `bson:"_id,omitempty" json:"id,omitempty"`
	Name string             `bson:"Name" json:"Name"`
	// Add additional fields as required.
}

// SubSubcategory represents the lowest level in the hierarchy
type SubSubcategory struct {
	Name string `bson:"name,omitempty" json:"name,omitempty"`
}

// Subcategory represents a subcategory that may contain sub-subcategories
type Subcategory struct {
	Name             string           `bson:"name,omitempty" json:"name,omitempty"`
	SubSubcategories []SubSubcategory `bson:"sub_subcategories,omitempty" json:"sub_subcategories,omitempty"`
}

// Category represents a category that may contain subcategories
type typeCategory struct {
	Name          string        `bson:"name,omitempty" json:"name,omitempty"`
	Subcategories []Subcategory `bson:"subcategories,omitempty" json:"subcategories,omitempty"`
}

// Type represents the top-level document that contains categories
type Type struct {
	ID         primitive.ObjectID `bson:"_id,omitempty" json:"id,omitempty"`
	Name       string             `bson:"name,omitempty" json:"name,omitempty"`
	Categories []typeCategory     `bson:"categories,omitempty" json:"categories,omitempty"`
}

type CurrencyDocument struct {
	Timestamp          time.Time       `bson:"timestamp"`
	TimeLastUpdateUnix int64           `bson:"time_last_update_unix"`
	BaseCode           string          `bson:"base_code"`
	ConversionRates    ConversionRates `bson:"conversion_rates"`
}
type ConversionRates struct {
	USD float64 `bson:"USD"`
	EUR float64 `bson:"EUR"`
	GBP float64 `bson:"GBP"`
	JPY float64 `bson:"JPY"`
	CNY float64 `bson:"CNY"`
	INR float64 `bson:"INR"`
	AUD float64 `bson:"AUD"`
	CAD float64 `bson:"CAD"`
	CHF float64 `bson:"CHF"`
	SAR float64 `bson:"SAR"`
	ZAR float64 `bson:"ZAR"`
	KRW float64 `bson:"KRW"`
	SGD float64 `bson:"SGD"`
	AED float64 `bson:"AED"`
	BRL float64 `bson:"BRL"`
}

// supplier
type Supplier struct {
	ID            primitive.ObjectID `bson:"_id,omitempty" json:"id,omitempty"`
	Email         string             `bson:"email" json:"email"`
	PID           string             `bson:"pid" json:"pid"`
	BusinessName  string             `bson:"business_name" json:"business_name"`
	BusinessDesc  string             `bson:"business_description" json:"business_description"`
	Telephone     string             `bson:"telephone" json:"telephone"`
	EmailGiven    string             `bson:"email_given" json:"email_given"`
	Address       string             `bson:"address" json:"address"`
	Location      Location           `bson:"location" json:"location"`
	ProfilePicURL string             `bson:"profile_pic_url" json:"profile_pic_url"`
	CoverPicURL   string             `bson:"cover_pic_url" json:"cover_pic_url"`
}

type Location struct {
	Latitude  float64 `bson:"latitude" json:"latitude"`
	Longitude float64 `bson:"longitude" json:"longitude"`
}

// items model
type Item struct {
	ID          primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	Name        string             `bson:"name" json:"name"`
	Description string             `bson:"description" json:"description"`
	SupplierPid string             `bson:"supplierPid" json:"supplierPid"`
	MaterialId  string             `bson:"materialId" json:"materialId"`
	Type        string             `bson:"type" json:"type"`
	Category    string             `bson:"category" json:"category"`
	Subcategory string             `bson:"subcategory" json:"subcategory"`
	Unit        string             `bson:"unit" json:"unit"`
	Price       float64            `bson:"price" json:"price"`
	ImgUrl      string             `bson:"imgUrl" json:"imgUrl"`
}
type Payment struct {
	Month       time.Time `bson:"Month" json:"Month"`
	Amount      float64   `bson:"Amount" json:"Amount"`
	PaymentDate time.Time `bson:"paymentDate" json:"paymentDate"`
}

// PaymentRecord represents the main model.
type PaymentRecord struct {
	ID          primitive.ObjectID `bson:"_id,omitempty" json:"id,omitempty"`
	SupplierPID string             `bson:"Supplierpid" json:"Supplierpid"`
	Approved    bool               `bson:"Approved" json:"Approved"`
	Payments    []Payment          `bson:"Payments" json:"Payments"`
	Deleted     bool               `bson:"Deleted" json:"Deleted"`
}

// package main

// import (
// 	"time"
// 	"go.mongodb.org/mongo-driver/bson/primitive"
// )

// // Category struct
// type Category struct {
// 	Category       string  `bson:"Category" json:"Category"`
// 	Subcategory    *string `bson:"Subcategory,omitempty" json:"Subcategory,omitempty"`
// 	SubSubcategory *string `bson:"SubSubcategory,omitempty" json:"SubSubcategory,omitempty"`
// }

// // Material struct
// type Material struct {
// 	ID       primitive.ObjectID `bson:"_id,omitempty" json:"id,omitempty"`
// 	Number   string             `bson:"Number" json:"Number"`
// 	Name     string             `bson:"Name" json:"Name"`
// 	Type     string             `bson:"Type" json:"Type"`
// 	Category Category           `bson:"Category" json:"Category"`
// 	Qty      int                `bson:"Qty,omitempty" json:"Qty,omitempty"`
// 	Unit     string             `bson:"Unit,omitempty" json:"Unit,omitempty"`
// 	Prices   [][]interface{}    `bson:"Prices" json:"Prices"` // List of [date, price]
// 	Source   *string            `bson:"Source,omitempty" json:"Source,omitempty"`
// }

// // SubSubcategory represents the lowest level in the hierarchy
// type SubSubcategory struct {
// 	Name string `bson:"name,omitempty" json:"name,omitempty"`
// }

// // Subcategory represents a subcategory that may contain sub-subcategories
// type Subcategory struct {
// 	Name            string           `bson:"name,omitempty" json:"name,omitempty"`
// 	SubSubcategories []SubSubcategory `bson:"sub_subcategories,omitempty" json:"sub_subcategories,omitempty"`
// }

// // Category represents a category that may contain subcategories
// type typeCategory struct {
// 	Name         string        `bson:"name,omitempty" json:"name,omitempty"`
// 	Subcategories []Subcategory `bson:"subcategories,omitempty" json:"subcategories,omitempty"`
// }

// // Type represents the top-level document that contains categories
// type Type struct {
// 	ID         primitive.ObjectID `bson:"_id,omitempty" json:"id,omitempty"`
// 	Name       string             `bson:"name,omitempty" json:"name,omitempty"`
// 	Categories []typeCategory         `bson:"categories,omitempty" json:"categories,omitempty"`
// }

// type CurrencyDocument struct {
// 	Timestamp              time.Time           `bson:"timestamp"`
// 	TimeLastUpdateUnix     int64            `bson:"time_last_update_unix"`
// 	BaseCode              string           `bson:"base_code"`
// 	ConversionRates       ConversionRates `bson:"conversion_rates"`
// }
// type ConversionRates struct {
// 	USD float64 `bson:"USD"`
// 	EUR float64 `bson:"EUR"`
// 	GBP float64 `bson:"GBP"`
// 	JPY float64 `bson:"JPY"`
// 	CNY float64 `bson:"CNY"`
// 	INR float64 `bson:"INR"`
// 	AUD float64 `bson:"AUD"`
// 	CAD float64 `bson:"CAD"`
// 	CHF float64 `bson:"CHF"`
// 	SAR float64 `bson:"SAR"`
// 	ZAR float64 `bson:"ZAR"`
// 	KRW float64 `bson:"KRW"`
// 	SGD float64 `bson:"SGD"`
// 	AED float64 `bson:"AED"`
// 	BRL float64 `bson:"BRL"`}

// //supplier
// type Supplier struct {
// 	ID             primitive.ObjectID `bson:"_id,omitempty" json:"id,omitempty"`
// 	Email          string             `bson:"email" json:"email"`
// 	PID            string             `bson:"pid" json:"pid"`
// 	BusinessName   string             `bson:"business_name" json:"business_name"`
// 	BusinessDesc   string             `bson:"business_description" json:"business_description"`
// 	Telephone      string             `bson:"telephone" json:"telephone"`
// 	EmailGiven     string             `bson:"email_given" json:"email_given"`
// 	Address        string             `bson:"address" json:"address"`
// 	Location       Location           `bson:"location" json:"location"`
// 	ProfilePicURL  string             `bson:"profile_pic_url" json:"profile_pic_url"`
// 	CoverPicURL    string             `bson:"cover_pic_url" json:"cover_pic_url"`
// }

// type Location struct {
// 	Latitude  float64 `bson:"latitude" json:"latitude"`
// 	Longitude float64 `bson:"longitude" json:"longitude"`
// }
// //items model
// type Item struct {
// 	ID          primitive.ObjectID `bson:"_id,omitempty" json:"id"`
// 	Name        string             `bson:"name" json:"name"`
// 	Description string             `bson:"description" json:"description"`
// 	SupplierPid string             `bson:"supplierPid" json:"supplierPid"`
// 	MaterialId  string             `bson:"materialId" json:"materialId"`
// 	Type        string             `bson:"type" json:"type"`
// 	Category    string             `bson:"category" json:"category"`
// 	Subcategory string             `bson:"subcategory" json:"subcategory"`
// 	Unit        string             `bson:"unit" json:"unit"`
// 	Price       float64            `bson:"price" json:"price"`
// 	ImgUrl      string             `bson:"imgUrl" json:"imgUrl"`
// }
