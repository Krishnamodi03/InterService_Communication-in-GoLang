package main

import (
	"gin_framework/order_api/controllers"
	"gin_framework/order_api/middleware"
	"os"

	"github.com/gin-gonic/gin"
	_ "github.com/heroku/x/hmetrics/onload"
)

func main() {
	port := os.Getenv("PORT")

	if port == "" {
		port = "9090"
	}

	router := gin.New()
	router.Use(gin.Logger())

	router.Use(middleware.ValidateToken())

	router.GET("/orders", controllers.GetOrders)
	router.GET("/orders/:id", controllers.GetOrder)
	router.POST("/orders", controllers.CreateOrder)
	router.PUT("/orders/:id", controllers.UpdateOrder)
	router.DELETE("/orders/:id", controllers.DeleteOrder)

	router.Run(":" + port)
}
