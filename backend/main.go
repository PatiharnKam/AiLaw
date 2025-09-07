// cmd/main.go
package main

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
)

func main() {

	r := gin.Default()

	r.Use(func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Origin, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	})

	{
		r.GET("/health", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{
				"status":  "OK",
				"message": "Server is running",
			})
		})

		r.GET("/hello", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{
				"message": "Hello World!",
				"server":  "Gin-Gonic",
				"port":    "8080",
			})
		})

		r.GET("/hello/:name", func(c *gin.Context) {
			name := c.Param("name")
			c.JSON(http.StatusOK, gin.H{
				"message": "Hello " + name + "!",
				"name":    name,
			})
		})

		r.GET("/greet", func(c *gin.Context) {
			name := c.DefaultQuery("name", "World")
			c.JSON(http.StatusOK, gin.H{
				"message": "Hello " + name + "!",
				"query":   name,
			})
		})
	}

	log.Println("🚀 Server starting on http://localhost:8080")
	log.Println("📍 Endpoints:")
	log.Println("   GET /health")
	log.Println("   GET /hello")
	log.Println("   GET /hello/:name")
	log.Println("   GET /greet?name=YourName")

	if err := r.Run(":8080"); err != nil {
		log.Fatal("Failed to start server:", err)
	}
}
