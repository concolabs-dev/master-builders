package auth

import (
	"bytes"
	"context"
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"

	"material-api/db"
	"material-api/model"
)

// Helper to read and store body for reuse (use this because shouldbindjson will drain the body and then we can not use it inside handlers)
// TODO -  better solution

func readAndStoreBody(c *gin.Context) ([]byte, error) {
	bodyBytes, err := ioutil.ReadAll(c.Request.Body)
	if err != nil {
		return nil, err
	}
	c.Request.Body = ioutil.NopCloser(bytes.NewBuffer(bodyBytes))
	c.Set("rawBody", bodyBytes)
	return bodyBytes, nil
}

func RequireOwnership(resourceType string) gin.HandlerFunc {

	return func(c *gin.Context) {

		log.Println("inside the ownership checker")

		userID := c.GetString("userID")
		roles := c.GetStringSlice("roles")
		method := c.Request.Method
		isOwner := false

		switch resourceType {

		case "professional":
			log.Println("inside the professional case")

			if method == http.MethodPost {
				// Use helper to read and store body
				bodyBytes, err := readAndStoreBody(c)
				if err != nil {
					log.Println("Failed to read request body:", err)
					c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
					return
				}
				var professional model.Professional

				if err := json.Unmarshal(bodyBytes, &professional); err != nil {
					log.Println("Invalid professional data in POST /professional:", err)
					c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "invalid professional data"})
					return
				}

				if professional.PID != userID {
					log.Printf("professional PID (%s) does not match userID (%s) in POST /professional", professional.PID, userID)
					c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "not the owner of professional"})
					return
				}

				isOwner = true

			} else {
				resourceID := c.Param("id") // This is the item's ObjectID
				itemObjID, err := primitive.ObjectIDFromHex(resourceID)

				if err != nil {
					log.Printf("Invalid item ID in professional case: %s, error: %v", resourceID, err)
					c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid item ID"})
					return
				}

				// Fetch the item from the database
				var professional model.Professional
				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()

				err = db.ProfessionalCollection.FindOne(ctx, bson.M{"_id": itemObjID}).Decode(&professional)
				if err != nil {
					log.Printf("Item not found in professional case: %s, error: %v", resourceID, err)
					c.JSON(http.StatusNotFound, gin.H{"error": "Item not found"})
					return
				}

				isOwner = professional.PID == userID
				if !isOwner {
					log.Printf("User %s is not the owner of item %s (owner: %s) in professional case", userID, resourceID, professional.PID)
				}
			}

		case "supplier":

			log.Println("inside the supplier case")

			if method == http.MethodPost {
				// Use helper to read and store body
				bodyBytes, err := readAndStoreBody(c)
				if err != nil {
					log.Println("Failed to read request body:", err)
					c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
					return
				}
				var supplier model.Supplier
				if err := json.Unmarshal(bodyBytes, &supplier); err != nil {
					log.Println("Invalid supplier data in POST /supplier:", err)
					c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "invalid supplier data"})
					return
				}

				if supplier.PID != userID {
					log.Printf("Supplier PID (%s) does not match userID (%s) in POST /supplier", supplier.PID, userID)
					c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "not the owner of supplier"})
					return
				}

				isOwner = true

			} else {
				resourceID := c.Param("id") // This is the item's ObjectID
				itemObjID, err := primitive.ObjectIDFromHex(resourceID)

				if err != nil {
					log.Printf("Invalid item ID in supplier case: %s, error: %v", resourceID, err)
					c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid item ID"})
					return
				}

				// Fetch the item from the database
				var supplier model.Supplier
				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()

				err = db.SupplierCollection.FindOne(ctx, bson.M{"_id": itemObjID}).Decode(&supplier)
				if err != nil {
					log.Printf("Item not found in supplier case: %s, error: %v", resourceID, err)
					c.JSON(http.StatusNotFound, gin.H{"error": "Item not found"})
					return
				}

				isOwner = supplier.PID == userID
				if !isOwner {
					log.Printf("User %s is not the owner of item %s (owner: %s) in supplier case", userID, resourceID, supplier.PID)
				}
			}

		case "item":

			switch method {

			case http.MethodPost:
				bodyBytes, err := readAndStoreBody(c)
				if err != nil {
					log.Println("Failed to read request body:", err)
					c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
					return
				}
				var item model.Item
				if err := json.Unmarshal(bodyBytes, &item); err != nil {
					log.Println("Invalid item data in POST /item:", err)
					c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "invalid item data"})
					return
				}

				isOwner = item.SupplierPid == userID
				if !isOwner {
					log.Printf("User %s is not the owner of item (POST) (owner: %s)", userID, item.SupplierPid)
				}

			case http.MethodPut, http.MethodDelete:

				idParam := c.Param("id")
				objID, err := primitive.ObjectIDFromHex(idParam)
				if err != nil {
					log.Printf("Invalid item ID in PUT/DELETE /item: %s, error: %v", idParam, err)
					c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "invalid item ID"})
					return
				}

				var dbItem model.Item
				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()

				err = db.ItemCollection.FindOne(ctx, bson.M{"_id": objID}).Decode(&dbItem)
				if err != nil {
					log.Printf("Item not found in PUT/DELETE /item: %s, error: %v", idParam, err)
					c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"error": "item not found"})
					return
				}

				if method == http.MethodPut {
					bodyBytes, err := readAndStoreBody(c)
					if err != nil {
						log.Println("Failed to read request body:", err)
						c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
						return
					}
					var reqItem model.Item
					if err := json.Unmarshal(bodyBytes, &reqItem); err != nil {
						log.Println("Invalid update data in PUT /item:", err)
						c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "invalid update data"})
						return
					}
					// comparing both DB and body
					isOwner = dbItem.SupplierPid == userID && reqItem.SupplierPid == userID
					if !isOwner {
						log.Printf("User %s is not the owner of item (PUT) (DB owner: %s, Body owner: %s)", userID, dbItem.SupplierPid, reqItem.SupplierPid)
					}
				} else {
					// DELETE: only check DB
					isOwner = dbItem.SupplierPid == userID
					if !isOwner {
						log.Printf("User %s is not the owner of item (DELETE) (DB owner: %s)", userID, dbItem.SupplierPid)
					}
				}
			}
		case "paymentRecord":
			for _, role := range roles {
				if role == "admin" {
					isOwner = true
					break
				}
			}
			if isOwner {
				log.Printf("User %s is admin, skipping paymentRecord ownership check", userID)
				break // skip DB call
			}

			raw := c.Param("pid") // e.g. "\"google-oauth2|101...\""
			unescaped, _ := url.PathUnescape(raw)
			pid := strings.Trim(unescaped, "\"") // remove any surrounding quotes
			typeParam := c.Param("type")
			log.Printf("pid: ", pid, "user : ", userID)
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			switch typeParam {
			case "supplier":
				var record model.PaymentRecord
				err := db.PaymentRecordCollection.FindOne(ctx, bson.M{"Supplierpid": pid, "Deleted": false}).Decode(&record)
				if err != nil {
					log.Printf("Payment record not found: %s, error: %v", pid, err)
					c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"error": "payment record not found"})
					return
				}
				isOwner = record.SupplierPID == userID
				if !isOwner {
					log.Printf("User %s is not the owner of payment record %s (owner: %s)", userID, pid, record.SupplierPID)
				}
			case "professional":
				var record model.ProfessionalPaymentRecord

				err := db.ProfessionalPaymentRecordCollection.FindOne(
					ctx,
					bson.M{"professional_pid": pid, "deleted": false},
				).Decode(&record)

				if err != nil {
					log.Printf("Payment record not found: %s, error: %v", pid, err)
					c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"error": "payment record not found for pid : " + pid})
					return
				}

				isOwner = record.ProfessionalPID == userID
				if !isOwner {
					log.Printf("User %s is not the owner of payment record %s (owner: %s)", userID, pid, record.ProfessionalPID)
				}

				log.Printf("Found record: %+v", record)
			}
		}

		if !isOwner {
			log.Printf("Ownership check failed for user %s on resource type %s", userID, resourceType)
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "not owner"})
			return
		}
		c.Next()
	}
}
