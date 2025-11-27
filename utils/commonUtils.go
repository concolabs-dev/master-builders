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

func FindTypeChanges(oldType model.Type, newType model.Type) []model.ChangeSet {
    oldTree := oldType.Categories
    newTree := newType.Categories

    var changes []model.ChangeSet
    // Check for Type name change once
    typeRenamed := oldType.Name != newType.Name

    for i, newCat := range newTree {
        // Skip new L1 categories
        if i >= len(oldTree) {
            continue
        }
        oldCat := oldTree[i]

        // --- L3 and L2 Checks ---
        for j, newSub := range newCat.Subcategories {
            // Skip new L2 categories
            if j >= len(oldCat.Subcategories) {
                continue
            }
            oldSub := oldCat.Subcategories[j]

            // Check for L3 changes
            for k, newSubSub := range newSub.SubSubcategories {
                // Skip new L3 categories
                if k >= len(oldSub.SubSubcategories) {
                    continue
                }
                oldSubSub := oldSub.SubSubcategories[k]

                // Change Condition: L3 name changed OR Type name changed
                if oldSubSub.Name != newSubSub.Name || typeRenamed {
                    changes = append(changes, model.ChangeSet{
                        Old: model.MaterialCategory{Type: oldType.Name, Category: oldCat.Name, Subcategory: oldSub.Name, SubSubcategory: oldSubSub.Name},
                        New: model.MaterialCategory{Type: newType.Name, Category: newCat.Name, Subcategory: newSub.Name, SubSubcategory: newSubSub.Name},
                    })
                }
            } // end L3

            // Check for L2 rename (only if L2 has no children and its name changed, OR Type name changed)
            if len(newSub.SubSubcategories) == 0 && (oldSub.Name != newSub.Name || typeRenamed) {
                changes = append(changes, model.ChangeSet{
                    Old: model.MaterialCategory{Type: oldType.Name, Category: oldCat.Name, Subcategory: oldSub.Name, SubSubcategory: ""},
                    New: model.MaterialCategory{Type: newType.Name, Category: newCat.Name, Subcategory: newSub.Name, SubSubcategory: ""},
                })
            }
        } // end L2

        // Check for L1 rename (only if L1 has no children and its name changed, OR Type name changed)
        if len(newCat.Subcategories) == 0 && (oldCat.Name != newCat.Name || typeRenamed) {
            changes = append(changes, model.ChangeSet{
                Old: model.MaterialCategory{Type: oldType.Name, Category: oldCat.Name, Subcategory: "", SubSubcategory: ""},
                New: model.MaterialCategory{Type: newType.Name, Category: newCat.Name, Subcategory: "", SubSubcategory: ""},
            })
        }
    } // end L1

    return changes
}