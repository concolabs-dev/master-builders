package utils

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"material-api/db"
	"material-api/model"
	"net/http"
)

func StartExchangeRateUpdater() {
	ticker := time.NewTicker(24 * time.Hour)
	defer ticker.Stop()

	for {
		UpdateExchangeRates()
		<-ticker.C
	}
}

func UpdateExchangeRates() {
	// apiKey := os.Getenv("EXCHANGE_RATE_API_KEY")
	url := "https://v6.exchangerate-api.com/v6/9b03f47e0242509680dac5fc/latest/LKR"
	resp, err := http.Get(url)
	if err != nil {
		log.Println("Error fetching exchange rates:", err)
		return
	}
	defer resp.Body.Close()

	var result model.ExchangeRateResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		log.Println("Error decoding exchange rate response:", err)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Insert the data into MongoDB
	_, err = db.ExchangeRateCollection.InsertOne(ctx, map[string]interface{}{
		"timestamp":             time.Now(),
		"time_last_update_unix": result.TimeLastUpdateUnix,
		"base_code":             result.BaseCode,
		"conversion_rates":      result.ConversionRates,
	})
	if err != nil {
		log.Println("Error inserting exchange rates into MongoDB:", err)
	} else {
		log.Println("Exchange rates updated successfully")
	}
}

func FindCategoryChanges(oldTree []model.TypeCategory, newTree []model.TypeCategory) []model.ChangeSet {
	var changes []model.ChangeSet

	for i, newCat := range newTree {
		if i >= len(oldTree) {
			continue // This is a new L1 category, not a rename
		}
		oldCat := oldTree[i]

		for j, newSub := range newCat.Subcategories {
			if j >= len(oldCat.Subcategories) {
				continue // New L2, not a rename
			}
			oldSub := oldCat.Subcategories[j]

			for k, newSubSub := range newSub.SubSubcategories {
				if k >= len(oldSub.SubSubcategories) {
					continue // New L3, not a rename
				}
				oldSubSub := oldSub.SubSubcategories[k]

				// We found a potential L3 rename!
				if oldSubSub.Name != newSubSub.Name {
					changes = append(changes, model.ChangeSet{
						Old: model.MaterialCategory{oldCat.Name, oldSub.Name, oldSubSub.Name},
						New: model.MaterialCategory{newCat.Name, newSub.Name, newSubSub.Name},
					})
				}
			} // end L3

			// Check for L2 rename
			if len(newSub.SubSubcategories) == 0 && oldSub.Name != newSub.Name {
				changes = append(changes, model.ChangeSet{
					Old: model.MaterialCategory{oldCat.Name, oldSub.Name, ""},
					New: model.MaterialCategory{newCat.Name, newSub.Name, ""},
				})
			}
		} // end L2

		// Check for L1 rename
		if len(newCat.Subcategories) == 0 && oldCat.Name != newCat.Name {
			changes = append(changes, model.ChangeSet{
				Old: model.MaterialCategory{oldCat.Name, "", ""},
				New: model.MaterialCategory{newCat.Name, "", ""},
			})
		}
	} // end L1

	return changes
}
