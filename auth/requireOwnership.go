package auth

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"

	"material-api/db"
	"material-api/model"
)

func RequireOwnership(resourceType string) gin.HandlerFunc {

	return func(c *gin.Context) {

		userID := c.GetString("userID")
		roles := c.GetStringSlice("roles")
		method := c.Request.Method
		isOwner := false

		switch resourceType {

		case "supplier":
			if method == http.MethodPost {
				// get the body to supplier model
				var supplier model.Supplier
				if err := c.ShouldBindJSON(&supplier); err != nil {
					c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "invalid supplier data"})
					return
				}

				if supplier.PID != userID {
					c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "not the owner of supplier"})
					return
				}

				isOwner = true

			} else {
				resourceID := c.Param("id") // This is the item's ObjectID
				itemObjID, err := primitive.ObjectIDFromHex(resourceID)

				if err != nil {
					c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid item ID"})
					return
				}

				// Fetch the item from the database
				var item model.Item
				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()

				err = db.ItemCollection.FindOne(ctx, bson.M{"_id": itemObjID}).Decode(&item)
				if err != nil {
					c.JSON(http.StatusNotFound, gin.H{"error": "Item not found"})
					return
				}

				isOwner = item.SupplierPid == userID
			}

		case "item":

			switch method {

			case http.MethodPost:

				var item model.Item
				if err := c.ShouldBindJSON(&item); err != nil {
					c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "invalid item data"})
					return
				}

				isOwner = item.SupplierPid == userID

			case http.MethodPut, http.MethodDelete:

				idParam := c.Param("id")
				objID, err := primitive.ObjectIDFromHex(idParam)
				if err != nil {
					c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "invalid item ID"})
					return
				}

				var dbItem model.Item
				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()

				err = db.ItemCollection.FindOne(ctx, bson.M{"_id": objID}).Decode(&dbItem)
				if err != nil {
					c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"error": "item not found"})
					return
				}

				if method == http.MethodPut {
					var reqItem model.Item
					if err := c.ShouldBindJSON(&reqItem); err != nil {
						c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "invalid update data"})
						return
					}
					// comparing both DB and body
					isOwner = dbItem.SupplierPid == userID && reqItem.SupplierPid == userID
				} else {
					// DELETE: only check DB
					isOwner = dbItem.SupplierPid == userID
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
				break // skip DB call
			}

			idParam := c.Param("id")
			objID, err := primitive.ObjectIDFromHex(idParam)
			if err != nil {
				c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "invalid payment ID"})
				return
			}

			var record model.PaymentRecord
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			err = db.PaymentRecordCollection.FindOne(ctx, bson.M{"_id": objID}).Decode(&record)
			if err != nil {
				c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"error": "payment record not found"})
				return
			}

			isOwner = record.SupplierPID == userID
		}

		if !isOwner {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "not owner"})
			return
		}
		c.Next()
	}
}
