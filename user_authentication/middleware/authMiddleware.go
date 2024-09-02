package middleware

import (
	helper "gin_framework/user_authentication/helpers"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// Authz validates token and authorizes users
func Authentication() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Retrieve the token from the 'Token' header
		clientToken := c.GetHeader("Authorization")
		// extracting actual token from token
		if clientToken == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "No Authorization header provided"})
			c.Abort()
			return
		}

		clientToken = strings.Split(clientToken, " ")[1]

		claims, err := helper.ValidateToken(clientToken)
		if err != "" {
			c.JSON(http.StatusInternalServerError, gin.H{"tokenerror": err})
			c.Abort()
			return
		}

		c.Set("email", claims.Email)
		c.Set("first_name", claims.First_name)
		c.Set("last_name", claims.Last_name)
		c.Set("uid", claims.Uid)
		c.Set("user_type", claims.User_type)

		c.Next()

	}
}
