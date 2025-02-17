package main

import (
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
}

// SubSubcategory represents the lowest level in the hierarchy
type SubSubcategory struct {
	Name string `bson:"name,omitempty" json:"name,omitempty"`
}

// Subcategory represents a subcategory that may contain sub-subcategories
type Subcategory struct {
	Name            string           `bson:"name,omitempty" json:"name,omitempty"`
	SubSubcategories []SubSubcategory `bson:"sub_subcategories,omitempty" json:"sub_subcategories,omitempty"`
}

// Category represents a category that may contain subcategories
type typeCategory struct {
	Name         string        `bson:"name,omitempty" json:"name,omitempty"`
	Subcategories []Subcategory `bson:"subcategories,omitempty" json:"subcategories,omitempty"`
}

// Type represents the top-level document that contains categories
type Type struct {
	ID         primitive.ObjectID `bson:"_id,omitempty" json:"id,omitempty"`
	Name       string             `bson:"name,omitempty" json:"name,omitempty"`
	Categories []typeCategory         `bson:"categories,omitempty" json:"categories,omitempty"`
}