package controllers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"gin_framework/user_authentication/database"
	helper "gin_framework/user_authentication/helpers"
	"gin_framework/user_authentication/models"
	"io"
	"log"
	"strings"

	"strconv"

	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"golang.org/x/crypto/bcrypt"
)

var userCollection *mongo.Collection = database.OpenCollection(database.Client, "user")
var validate = validator.New()

// HashPassword is used to encrypt the password before it is stored in the DB
func HashPassword(password string) string {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), 14)
	if err != nil {
		log.Panic(err)
	}
	return string(bytes)
}

// VerifyPassword checks the input password while verifying it with the password in the DB.
func VerifyPassword(userPassword string, providedPassword string) (bool, string) {
	err := bcrypt.CompareHashAndPassword([]byte(providedPassword), []byte(userPassword))
	if err != nil {
		return false, "login or password is incorrect"
	}
	return true, ""
}

// SignUp is the API used to create a new user
func SignUp() gin.HandlerFunc {
	return func(c *gin.Context) {
		var ctx, cancel = context.WithTimeout(context.Background(), 100*time.Second)
		defer cancel()
		var user models.User

		if err := c.BindJSON(&user); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		validationErr := validate.Struct(user)
		if validationErr != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": validationErr.Error()})
			return
		}

		// Check if email already exists
		count, err := userCollection.CountDocuments(ctx, bson.M{"email": user.Email})
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "error occurred while checking for the email"})
			return
		}

		if count > 0 {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "this email already exists"})
			return
		}

		// Check if phone number already exists
		count, err = userCollection.CountDocuments(ctx, bson.M{"phone": user.Phone})
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "error occurred while checking for the phone number"})
			return
		}

		if count > 0 {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "this phone number already exists"})
			return
		}

		password := HashPassword(*user.Password)
		user.Password = &password
		user.Created_at = time.Now()
		user.Updated_at = time.Now()
		user.ID = primitive.NewObjectID()
		// user.User_id = user.ID.Hex()
		token, refreshToken, _ := helper.GenerateAllTokens(*user.Email, *user.First_name, *user.Last_name, *user.User_type, user.ID.Hex())
		user.Token = &token
		user.Refresh_token = &refreshToken

		_, insertErr := userCollection.InsertOne(ctx, user)
		if insertErr != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "User item was not created"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "SignUp Successfull"})
	}
}

// Login is the API used to authenticate a user
func Login() gin.HandlerFunc {
	return func(c *gin.Context) {
		var ctx, cancel = context.WithTimeout(context.Background(), 100*time.Second)
		defer cancel()

		var user models.User
		var foundUser models.User

		if err := c.BindJSON(&user); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		err := userCollection.FindOne(ctx, bson.M{"email": user.Email}).Decode(&foundUser)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "This email is not registerd"})
			return
		}

		passwordIsValid, msg := VerifyPassword(*user.Password, *foundUser.Password)
		if !passwordIsValid {
			c.JSON(http.StatusInternalServerError, gin.H{"error": msg})
			return
		}

		token, refreshToken, _ := helper.GenerateAllTokens(*foundUser.Email, *foundUser.First_name, *foundUser.Last_name, *foundUser.User_type, foundUser.ID.Hex())

		helper.UpdateAllTokens(token, refreshToken, foundUser.ID)
		err = userCollection.FindOne(ctx, bson.M{"_id": foundUser.ID}).Decode(&foundUser)

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, foundUser)
	}
}

// GetUsers fetches users without pagination

// func GetUsers() gin.HandlerFunc {
// 	return func(c *gin.Context) {
// 		if err := helper.CheckUserType(c, "ADMIN"); err != nil {
// 			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
// 			return
// 		}

// 		var ctx, cancel = context.WithTimeout(context.Background(), 100*time.Second)
// 		defer cancel()

// 		// Find all users without pagination
// 		cursor, err := userCollection.Find(ctx, bson.M{})
// 		if err != nil {
// 			c.JSON(http.StatusInternalServerError, gin.H{"error": "error occurred while listing user items"})
// 			return
// 		}
// 		defer cursor.Close(ctx)

// 		var allUsers []bson.M
// 		if err := cursor.All(ctx, &allUsers); err != nil {
// 			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
// 			return
// 		}

// 		c.JSON(http.StatusOK, allUsers)
// 	}
// }

// GetUsers fetches users with pagination
func GetUsers() gin.HandlerFunc {
	return func(c *gin.Context) {
		if err := helper.CheckUserType(c, "ADMIN"); err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
			return
		}
		var ctx, cancel = context.WithTimeout(context.Background(), 100*time.Second)
		defer cancel()

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

		startIndex := (page - 1) * pageSize

		totalCount, err := userCollection.CountDocuments(ctx, bson.M{})
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to count total users"})
			return
		}
		findOptions := options.Find()
		findOptions.SetSkip(int64(startIndex))
		findOptions.SetLimit(int64(pageSize))
		findOptions.SetProjection(bson.M{
			"_id":        1,
			"first_name": 1,
			"last_name":  1,
			"email":      1,
			"phone":      1,
			"created_at": 1,
			"updated_at": 1,
		})
		result, err := userCollection.Find(ctx, bson.M{}, findOptions)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "error occurred while listing user items"})
			return
		}

		var allUsers []bson.M
		if err = result.All(ctx, &allUsers); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		previousPage := page - 1
		nextPage := page + 1

		if startIndex+len(allUsers) >= int(totalCount) {
			nextPage = 0
		}
		if page == 1 {
			previousPage = 0
		}
		response := gin.H{
			"total_records": totalCount,
			"previous_page": previousPage,
			"current_page":  page,
			"page_size":     pageSize,
			"next_page":     nextPage,
			"users":         allUsers,
		}
		c.JSON(http.StatusOK, response)
	}
}

// GetUser fetches a single user by their ID
func GetUser() gin.HandlerFunc {
	return func(c *gin.Context) {
		userId := c.Param("user_id")

		oid, err := primitive.ObjectIDFromHex(userId)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
			return
		}

		if err := helper.MatchUserTypeToUid(c, userId); err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
			return
		}
		var ctx, cancel = context.WithTimeout(context.Background(), 100*time.Second)
		defer cancel()

		var user models.UserResponse
		err = userCollection.FindOne(ctx, bson.M{"_id": oid}).Decode(&user)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, user)
	}
}

// UpdateUser allows a user to update their information
func UpdateUser() gin.HandlerFunc {
	return func(c *gin.Context) {
		userId := c.Param("user_id")

		oid, err := primitive.ObjectIDFromHex(userId)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
			return
		}

		if err := helper.MatchUserTypeToUid(c, userId); err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
			return
		}

		var ctx, cancel = context.WithTimeout(context.Background(), 100*time.Second)
		defer cancel()

		var user models.User
		if err := c.BindJSON(&user); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		updateObj := bson.M{
			"first_name": user.First_name,
			"last_name":  user.Last_name,
			"email":      user.Email,
			"phone":      user.Phone,
			"updated_at": time.Now(),
		}
		_, err = userCollection.UpdateOne(
			ctx,
			bson.M{"_id": oid},
			bson.M{"$set": updateObj},
		)

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "User updation failed"})
			return
		}

		// Fetch the updated user data
		var updatedUser models.UserResponse
		err = userCollection.FindOne(ctx, bson.M{"_id": oid}).Decode(&updatedUser)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch updated user data"})
			return
		}

		c.JSON(http.StatusOK, updatedUser)
	}
}

// DeleteUser allows an admin or user to delete their own account
func DeleteUser() gin.HandlerFunc {
	return func(c *gin.Context) {
		userId := c.Param("user_id")

		// Convert userId from string to ObjectID
		oid, err := primitive.ObjectIDFromHex(userId)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
			return
		}
		if err := helper.MatchUserTypeToUid(c, userId); err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
			return
		}
		var ctx, cancel = context.WithTimeout(context.Background(), 100*time.Second)
		defer cancel()

		_, err = userCollection.DeleteOne(ctx, bson.M{"_id": oid})
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "user Deletion Failed"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "User deleted successfully"})
	}
}

// common order handler function return Json order data or nil
func CommonOrderHandler(c *gin.Context) (string, []byte) {
	// Extract the token from the request headers
	token := c.GetHeader("Authorization")
	// extracting actual token from token1

	if token == "" {
		c.JSON(http.StatusInternalServerError, gin.H{"error of token : ": "No Authorization header provided"})
		c.Abort()
	}
	token = strings.Split(token, " ")[1]

	// Extract order data from the request body
	var orderData map[string]interface{}
	if err := c.BindJSON(&orderData); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input"})
	}

	// Marshal the order data to JSON
	orderJSON, err := json.Marshal(orderData)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to marshal order data"})
	}
	return token, orderJSON
}

// user creating order
func UserCreateOrder() gin.HandlerFunc {
	return func(c *gin.Context) {
		token, orderJSON := CommonOrderHandler(c)

		// Create an HTTP POST request to the order API with the token and order data
		req, err := http.NewRequest("POST", "http://localhost:9090/orders", bytes.NewBuffer(orderJSON))
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create request", "details": err.Error()})
			return
		}

		// Set the token in the Authorization header with the Bearer format
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("Content-Type", "application/json")

		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"user error": "Failed to create order", "details": err.Error()})
			return
		}
		defer resp.Body.Close()

		// Check if the order creation was successful
		if resp.StatusCode != http.StatusOK {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Order creation failed"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "Order created successfully"})
	}
}

func UserUpdateOrder() gin.HandlerFunc {
	return func(c *gin.Context) {
		token, orderJSON := CommonOrderHandler(c)
		orderID := c.Param("id")
		if orderID == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Order ID is missing"})
			return
		}

		// Create an HTTP PATCH request to the order API with the token and order data
		updateOrderURL := fmt.Sprintf("http://localhost:9090/orders/%s", orderID)
		req, err := http.NewRequest("PUT", updateOrderURL, bytes.NewBuffer(orderJSON))

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create request", "details": err.Error()})
			return
		}
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("Content-Type", "application/json")

		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "error generating response", "details": err.Error()})
			return
		}
		defer resp.Body.Close()

		// Check if the order creation was successful
		if resp.StatusCode != http.StatusOK {
			statusCode := resp.StatusCode
			switch statusCode {
			case http.StatusBadRequest: // 400
				c.JSON(http.StatusBadRequest, gin.H{"error": "Bad request, please check the input parameters."})
				return
			case http.StatusUnauthorized: // 401
				c.JSON(http.StatusUnauthorized, gin.H{"error": "User ID not found"})
				return
			case http.StatusForbidden: // 403
				c.JSON(http.StatusForbidden, gin.H{"error": "You are not authorized to update this order"})
				return
			case http.StatusNotFound: // 404
				c.JSON(http.StatusNotFound, gin.H{"error": "No Record Found"})
				return
			case http.StatusInternalServerError: // 500
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update order"})
				return
			}
		}

		c.JSON(http.StatusOK, gin.H{"message": "Order updated successfully"})
	}
}
func UserDeleteOrder() gin.HandlerFunc {
	return func(c *gin.Context) {
		orderID := c.Param("id")

		token := c.GetHeader("Authorization")

		if orderID == "" || token == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Order ID or Access Token missing"})
			return
		}
		// extracting actual token from token

		token = strings.Split(token, " ")[1]

		getOrderURL := fmt.Sprintf("http://localhost:9090/orders/%s", orderID)

		req, err := http.NewRequest("DELETE", getOrderURL, nil)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "error creating request", "details": err.Error()})
			return
		}
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("Content-Type", "application/json")

		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "error generating response", "details": err.Error()})
			return
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			statusCode := resp.StatusCode
			switch statusCode {
			case http.StatusBadRequest: // 400
				c.JSON(http.StatusBadRequest, gin.H{"error": "Bad request, please check the input parameters."})
				return
			case http.StatusUnauthorized: // 401
				c.JSON(http.StatusUnauthorized, gin.H{"error": "User ID not found"})
				return
			case http.StatusForbidden: // 403
				c.JSON(http.StatusForbidden, gin.H{"error": "You are not authorized to delete this order"})
				return
			case http.StatusNotFound: // 404
				c.JSON(http.StatusNotFound, gin.H{"error": "No Record Found"})
				return
			case http.StatusInternalServerError: // 500
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete order"})
				return
			}
		}

		c.JSON(http.StatusOK, gin.H{"message": "Order deleted successfully"})

	}

}
func UserGetOrder() gin.HandlerFunc {
	return func(c *gin.Context) {
		orderID := c.Param("id")

		token := c.GetHeader("Authorization")
		if orderID == "" || token == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Order ID or Access Token missing"})
			return
		}
		token = strings.Split(token, " ")[1]

		getOrderURL := fmt.Sprintf("http://localhost:9090/orders/%s", orderID)

		req, err := http.NewRequest("GET", getOrderURL, nil)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "error creating request", "details": err.Error()})
			return
		}
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("Content-Type", "application/json")

		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "error generating response", "details": err.Error()})
			return
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			statusCode := resp.StatusCode
			switch statusCode {
			case http.StatusBadRequest: // 400
				c.JSON(http.StatusBadRequest, gin.H{"error": "Bad request, please check the input parameters."})
				return
			case http.StatusUnauthorized: // 401
				c.JSON(http.StatusUnauthorized, gin.H{"error": "User ID not found"})
				return
			case http.StatusForbidden: // 403
				c.JSON(http.StatusForbidden, gin.H{"error": "You are not authorized to view this order"})
				return
			case http.StatusNotFound: // 404
				c.JSON(http.StatusNotFound, gin.H{"error": "No Record Found"})
				return
			default: // 500 or any other
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get order"})
				return
			}
		}
		var orderData map[string]interface{}
		if err := json.NewDecoder(resp.Body).Decode(&orderData); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to decode response body", "details": err.Error()})
			return
		}
		if orderData == nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "No orders found."})
		}
		c.JSON(http.StatusOK, gin.H{"Your_order": orderData})

	}
}

func UserGetOrders() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Extract the token from the request headers
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

		token := c.GetHeader("Authorization")

		if token == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "No Authorization header provided"})
			c.Abort()
			return
		}
		token = strings.Split(token, " ")[1]

		// Create an HTTP GET request to the order API to fetch all orders
		// ?page=1&pageSize=1
		getOrdersURL := fmt.Sprintf("http://localhost:9090/orders?page=%v&pageSize=%v", page, pageSize)
		req, err := http.NewRequest("GET", getOrdersURL, nil)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "error creating request", "details": err.Error()})
			return
		}

		// Set the token in the request header
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("Content-Type", "application/json")

		// Send the request to the order API
		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "error generating response", "details": err.Error()})
			return
		}
		defer resp.Body.Close()

		// Check if the order retrieval was successful
		if resp.StatusCode != http.StatusOK {
			statusCode := resp.StatusCode
			switch statusCode {
			case http.StatusBadRequest: // 400
				c.JSON(http.StatusBadRequest, gin.H{"error": "Bad request, please check the input parameters."})
				return
			case http.StatusUnauthorized: // 401
				c.JSON(http.StatusUnauthorized, gin.H{"error": "User ID not found"})
				return
			case http.StatusForbidden: // 403
				c.JSON(http.StatusForbidden, gin.H{"error": "You are not authorized to access these orders"})
				return
			case http.StatusNotFound: // 404
				c.JSON(http.StatusNotFound, gin.H{"error": "No Record Found"})
				return
			default: // 500 or any other
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get order"})
				return
			}
		}

		//  Log the response body
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			log.Println("Error reading response body:", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to read response body", "details": err.Error()})
			return
		}

		// Parse the response body to get all orders
		var allOrders []map[string]interface{}
		if err := json.Unmarshal(body, &allOrders); err != nil {
			log.Println("Error decoding response body:", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to decode response", "details": err.Error()})
			return
		}

		// Return the filtered orders to the client
		if allOrders == nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "No orders found."})
			return
		}
		c.JSON(http.StatusOK, gin.H{"Your_order": allOrders})
	}
}

// RefreshToken generates new access-token when old one expires
func RefreshToken() gin.HandlerFunc {
	return func(c *gin.Context) {
		refreshToken := c.GetHeader("Authorization")
		userId := c.Param("user_id")
		// converting userId to primitive.ObjectID
		oid, err := primitive.ObjectIDFromHex(userId)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
			return
		}
		if refreshToken == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "no refresh token provided"})
			return
		}
		refreshToken = strings.Split(refreshToken, " ")[1]

		newAccessToken, err := helper.GenerateNewAccessToken(refreshToken)

		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
			return
		}
		helper.UpdateAllTokens(newAccessToken, refreshToken, oid)

		c.JSON(http.StatusOK, gin.H{"access_token": newAccessToken})
	}
}
