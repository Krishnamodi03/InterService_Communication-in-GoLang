package main

import (
	"log"
	// "net/http"
	"gin_mongo_crud/controllers"

	"github.com/gin-gonic/gin"
)

func main() {
	r := gin.Default()
	r.GET("/users", controllers.GetUsers)
	r.GET("/users/:id", controllers.GetUser)
	r.POST("/users", controllers.CreateUser)
	r.PUT("/users/:id", controllers.UpdateUser)
	r.DELETE("/users/:id", controllers.DeleteUser)

	err := r.Run()
	if err != nil {
		log.Fatal("Unable to start the server")
	}
}
