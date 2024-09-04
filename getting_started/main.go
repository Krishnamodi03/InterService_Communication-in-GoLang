package main

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func main() {
	r := gin.Default()
	r.GET("/ping", test)
	r.Run(":9090") // listen and serve on 0.0.0.0:8080 (for windows "localhost:8080")
}
func test(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"message":         "pong",
		"you are at port": 9090,
	})
}
