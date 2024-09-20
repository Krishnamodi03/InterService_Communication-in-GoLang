package controllers

import (
	"net/http"
	"time"

	"gin_mongo_crud/database"
	"gin_mongo_crud/model"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func GetUsers(c *gin.Context) {
	var users []model.User
	cursor, err := database.DB.Find(database.Ctx, bson.M{})
	if err != nil {
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}
	defer cursor.Close(database.Ctx)
	if err = cursor.All(database.Ctx, &users); err != nil {
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}
	c.JSON(http.StatusOK, users)
}

func GetUser(c *gin.Context) {
	id := c.Param("id")
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}

	var user model.User
	err = database.DB.FindOne(database.Ctx, bson.M{"_id": objectID}).Decode(&user)
	if err != nil {
		c.AbortWithStatus(http.StatusNotFound)
		return
	}
	c.JSON(http.StatusOK, user)
}

func CreateUser(c *gin.Context) {
	var user model.User
	if err := c.BindJSON(&user); err != nil {
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}
	user.Id = primitive.NewObjectID()
	user.CreatedAt = time.Now().Local()
	user.UpdatedAt = time.Now().Local()
	_, err := database.DB.InsertOne(database.Ctx, user)
	if err != nil {
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}
	c.JSON(http.StatusOK, user)
}

func UpdateUser(c *gin.Context) {
	id := c.Param("id")
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}

	var user model.User
	err = database.DB.FindOne(database.Ctx, bson.M{"_id": objectID}).Decode(&user)
	if err != nil {
		c.AbortWithStatus(http.StatusNotFound)
		return
	}

	if err := c.BindJSON(&user); err != nil {
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}

	_, err = database.DB.UpdateOne(database.Ctx, bson.M{"_id": objectID}, bson.M{"$set": user})
	if err != nil {
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}
	c.JSON(http.StatusOK, user)
}

func DeleteUser(c *gin.Context) {
	id := c.Param("id")
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}

	_, err = database.DB.DeleteOne(database.Ctx, bson.M{"_id": objectID})
	if err != nil {
		c.AbortWithStatus(http.StatusNotFound)
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "User deleted", "deletedID": objectID})
}
