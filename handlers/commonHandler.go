package handlers

import (
	"context"
	"fmt"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"

	"material-api/db"
	"material-api/model"
)


func GetMajorCurrencies(c *gin.Context) {
	// List of major currencies you want to fetch from the database
	// majorCurrencies := []string{"USD", "EUR", "GBP", "JPY", "CNY", "INR", "AUD", "CAD", "CHF", "SAR", "ZAR", "KRW", "SGD", "AED", "BRL"}

	// Find the latest exchange rates document from MongoDB
	var result model.CurrencyDocument

	// Fetch the most recent exchange rate document

	opts := options.FindOne().SetSort(map[string]int{"timestamp": -1}) // Sort by most recent timestamp
	fmt.Println(opts)
	err := db.ExchangeRateCollection.FindOne(context.Background(), bson.D{}, opts).Decode(&result)
	fmt.Println(result)
	if err != nil {
		log.Println("Error fetching exchange rates from MongoDB:", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch exchange rates"})
		return
	}

	// Create a map for major currencies
	majorRates := map[string]float64{
		"USD": result.ConversionRates.USD,
		"EUR": result.ConversionRates.EUR,
		"GBP": result.ConversionRates.GBP,
		"JPY": result.ConversionRates.JPY,
		"CNY": result.ConversionRates.CNY,
		"INR": result.ConversionRates.INR,
		"AUD": result.ConversionRates.AUD,
		"CAD": result.ConversionRates.CAD,
		"CHF": result.ConversionRates.CHF,
		"SAR": result.ConversionRates.SAR,
		"ZAR": result.ConversionRates.ZAR,
		"KRW": result.ConversionRates.KRW,
		"SGD": result.ConversionRates.SGD,
		"AED": result.ConversionRates.AED,
		"BRL": result.ConversionRates.BRL,
	}

	// Filter and return only the major currencies
	// for _, currency := range majorCurrencies {
	// 	if rate, exists := result.ConversionRates[currency]; exists {
	// 		majorRates[currency] = rate
	// 	}
	// }

	c.JSON(http.StatusOK, gin.H{"major_currencies": majorRates})
}