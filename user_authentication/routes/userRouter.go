package routes

import (
	controller "gin_framework/user_authentication/controllers"
	"gin_framework/user_authentication/middleware"

	"github.com/gin-gonic/gin"
)

func UserRoutes(incomingRoutes *gin.Engine) {
	incomingRoutes.Use(middleware.Authentication())
	incomingRoutes.GET("/users", controller.GetUsers())
	incomingRoutes.GET("/users/:user_id", controller.GetUser())
	incomingRoutes.PUT("/users/:user_id", controller.UpdateUser())
	incomingRoutes.DELETE("/users/:user_id", controller.DeleteUser())

	incomingRoutes.POST("/user/createOrder", controller.UserCreateOrder())
	incomingRoutes.GET("/user/getOrder/:id", controller.UserGetOrder())
	incomingRoutes.PUT("/user/updateOrder/:id", controller.UserUpdateOrder())
	incomingRoutes.DELETE("/user/deleteOrder/:id", controller.UserDeleteOrder())
	incomingRoutes.GET("/user/getOrders", controller.UserGetOrders())
}
