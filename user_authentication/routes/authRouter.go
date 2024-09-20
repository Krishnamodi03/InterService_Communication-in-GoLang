package routes

import (
	controller "gin_framework/user_authentication/controllers"

	"github.com/gin-gonic/gin"
)

func AuthRoutes(incomingRoutes *gin.Engine) {
	incomingRoutes.POST("/auth/signup", controller.SignUp())
	incomingRoutes.POST("/auth/login", controller.Login())
	incomingRoutes.POST("/auth/refresh-token/:user_id", controller.RefreshToken())
}
