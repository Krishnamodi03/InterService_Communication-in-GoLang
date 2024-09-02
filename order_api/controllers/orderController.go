package controllers

import (
	"log"
	"net/http"
	"strconv"
	"time"

	"gin_framework/order_api/database"
	"gin_framework/order_api/model"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func GetOrders(c *gin.Context) {
	page, err := strconv.Atoi(c.Query("page"))
	if err != nil || page < 1 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid page parameter"})
		return
	}
	pageSize, err := strconv.Atoi(c.Query("pageSize"))
	if err != nil || pageSize < 1 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid pagesize parameter"})
		return
	}

	// Calculate the number of documents to skip
	skip := (page - 1) * pageSize
	userType, exists := c.Get("user_type")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User type not found"})
		return
	}
	var orders []model.Order

	if userType == "ADMIN" {
		// Admins can access all orders
		cursor, err := database.DB.Find(database.Ctx, bson.M{}, options.Find().SetSkip(int64(skip)).SetLimit(int64(pageSize)))
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			c.Abort()
			return
		}
		defer cursor.Close(database.Ctx)
		if err = cursor.All(database.Ctx, &orders); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			c.Abort()
			return
		}
		c.JSON(http.StatusOK, orders)
	} else {
		// Non-admins can only access their own orders
		userId, exists := c.Get("userId")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "User ID not found"})
			return
		}
		oid, err := primitive.ObjectIDFromHex(userId.(string))
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid user ID"})
			return
		}
		cursor, err := database.DB.Find(database.Ctx, bson.M{"userID": oid}, options.Find().SetSkip(int64(skip)).SetLimit(int64(pageSize)))
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		defer cursor.Close(database.Ctx)
		if err = cursor.All(database.Ctx, &orders); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			c.Abort()
			return
		}
		c.JSON(http.StatusOK, orders)
	}
}

func GetOrder(c *gin.Context) {
	id := c.Param("id")
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		log.Println("Error:", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Bad request, please check the input parameters."})
		c.Abort()
		return
	}

	// Get the userId from the token
	userId, exists := c.Get("userId")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User ID not found"})
		return
	}

	// Convert userId to ObjectID type
	oid, err := primitive.ObjectIDFromHex(userId.(string))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid user ID"})
		return
	}

	// Find the order document
	var order model.Order
	err = database.DB.FindOne(database.Ctx, bson.M{"_id": objectID}).Decode(&order)
	if err != nil {
		c.AbortWithStatus(http.StatusNotFound)
		return
	}
	// Get user_type
	userType, exist := c.Get("user_type")
	if !exist {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User type not found"})
		return
	}
	// check user_type
	if userType.(string) != "ADMIN" {
		// Check if the userId from the token matches the UserID in the order document
		if oid != order.UserID {
			c.JSON(http.StatusForbidden, gin.H{"error": "You are not authorized to view this order"})
			return
		}
	}

	c.JSON(http.StatusOK, order)
}
func CreateOrder(c *gin.Context) {
	var order model.Order

	// Bind JSON to order model
	if err := c.BindJSON(&order); err != nil {
		log.Println("Error:", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Bad request, please check the input parameters.", "details": err.Error()})
		c.Abort()
		return
	}

	// Retrieve user ID from the JWT token
	userId, exists := c.Get("userId")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User ID not found"})
		return
	}

	// Convert userId to ObjectID type
	oid, err := primitive.ObjectIDFromHex(userId.(string))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid user ID"})
		return
	}

	order.UserID = oid
	order.Id = primitive.NewObjectID()
	order.Created_at = time.Now().Local()
	order.Updated_at = time.Now().Local()
	_, err = database.DB.InsertOne(database.Ctx, order)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"order error": "Failed to create order", "details": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Order created successfully", "order": order})
}

func UpdateOrder(c *gin.Context) {
	id := c.Param("id")
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		log.Println("Error:", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Bad request, please check the input parameters.", "details": err.Error()})
		c.Abort()
		return
	}

	// Get the userId from the token
	userId, exists := c.Get("userId")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User ID not found"})
		return
	}

	// Convert userId to ObjectID type
	oid, err := primitive.ObjectIDFromHex(userId.(string))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid user ID", "details": err.Error()})
		return
	}

	// Find the order document
	var order model.Order
	err = database.DB.FindOne(database.Ctx, bson.M{"_id": objectID}).Decode(&order)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "No Record Found", "details": err.Error()})
		c.Abort()
		return
	}
	// Get user_type
	userType, exist := c.Get("user_type")
	if !exist {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User type not found"})
		return
	}
	// check user_type
	if userType.(string) != "ADMIN" {
		// Check if the userId from the token matches the UserID in the order document
		if oid != order.UserID {
			c.JSON(http.StatusForbidden, gin.H{"error": "You are not authorized to update this order"})
			return
		}
	}

	// Bind JSON to order model
	if err := c.BindJSON(&order); err != nil {
		log.Println("Error:", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Bad request, please check the input parameters.", "details": err.Error()})
		c.Abort()
		return
	}

	// Update the order document
	_, err = database.DB.UpdateOne(database.Ctx, bson.M{"_id": objectID}, bson.M{"$set": order})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update order", "details": err.Error()})
		return
	}
	c.JSON(http.StatusOK, order)
}

func DeleteOrder(c *gin.Context) {
	id := c.Param("id")
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		log.Println("Error:", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Bad request, please check the input parameters.", "details": err.Error()})
		c.Abort()
		return
	}

	// Get the userId from the token
	userId, exists := c.Get("userId")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User ID not found"})
		return
	}

	// Convert userId to ObjectID type
	oid, err := primitive.ObjectIDFromHex(userId.(string))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid user ID", "details": err.Error()})
		return
	}

	// Find the order document
	var order model.Order
	err = database.DB.FindOne(database.Ctx, bson.M{"_id": objectID}).Decode(&order)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "No Record Found", "details": err.Error()})
		return
	}
	// Get user_type
	userType, exist := c.Get("user_type")
	if !exist {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User type not found"})
		return
	}
	// check user_type
	if userType.(string) != "ADMIN" {

		// Check if the userId from the token matches the UserID in the order document
		if oid != order.UserID {
			c.JSON(http.StatusForbidden, gin.H{"error": "You are not authorized to delete this order"})
			return
		}
	}

	// Delete the order document
	_, err = database.DB.DeleteOne(database.Ctx, bson.M{"_id": objectID})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete order", "details": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Order deleted", "deletedID": objectID})
}
