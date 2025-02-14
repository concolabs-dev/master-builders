package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Sample test data
var sampleMaterials = []Material{
	{
		ID:     primitive.NewObjectID(),
		Number: "M-001",
		Name:   "Steel Rod",
		Type:   "Construction",
		Category: Category{
			Category:    "Metals",
			Subcategory: ptr("Steel"),
		},
		Qty:    100,
		Unit:   "kg",
		Prices: [][]interface{}{{"2024-02-01", 120.50}, {"2024-02-10", 122.00}},
		Source: ptr("Supplier A"),
	},
	{
		ID:     primitive.NewObjectID(),
		Number: "M-002",
		Name:   "Copper Wire",
		Type:   "Electrical",
		Category: Category{
			Category:       "Metals",
			Subcategory:    ptr("Copper"),
			SubSubcategory: ptr("Electrical"),
		},
		Qty:    50,
		Unit:   "m",
		Prices: [][]interface{}{{"2024-02-05", 200.00}, {"2024-02-15", 210.75}},
		Source: ptr("Supplier B"),
	},
}

// Helper function to create string pointers
func ptr(s string) *string {
	return &s
}

func testMaterials() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// **1. Insert Sample Data**
	fmt.Println("Inserting sample materials...")
	for _, material := range sampleMaterials {
		_, err := collection.InsertOne(ctx, material)
		if err != nil {
			log.Fatalf("Error inserting material: %v", err)
		}
	}
	fmt.Println("✅ Sample materials inserted successfully!")

	// **2. Verify Data is Inserted**
	fmt.Println("Verifying inserted materials...")
	var materials []Material
	cursor, err := collection.Find(ctx, bson.M{})
	if err != nil {
		log.Fatalf("Error retrieving materials: %v", err)
	}
	defer cursor.Close(ctx)

	for cursor.Next(ctx) {
		var material Material
		if err := cursor.Decode(&material); err != nil {
			log.Println("Error decoding material:", err)
			continue
		}
		materials = append(materials, material)
	}

	if len(materials) > 0 {
		fmt.Println("✅ Retrieved materials:")
		for _, m := range materials {
			fmt.Printf("  - %s (%s), Type: %s, Qty: %d %s\n", m.Name, m.Number, m.Type, m.Qty, m.Unit)
		}
	} else {
		log.Fatal("❌ No materials found after insertion!")
	}

	// **3. Delete All Materials**
	fmt.Println("Deleting all test materials...")
	_, err = collection.DeleteMany(ctx, bson.M{})
	if err != nil {
		log.Fatalf("Error deleting materials: %v", err)
	}
	fmt.Println("✅ All test materials deleted successfully! 🎉")
}
