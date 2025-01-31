package middleware

import (
	"log"
	"time"

	"github.com/gin-gonic/gin"
)

func RequestLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Start timer
		startTime := time.Now()

		// Process request
		c.Next()

		// Calculate request duration
		duration := time.Since(startTime)

		// Get the client's IP address
		clientIP := c.ClientIP()

		// Get the user identity from the context (assuming it's set in AuthMiddleware)
		user, exists := c.Get("user")
		if !exists {
			user = "anonymous"
		}

		// Log the request details
		log.Printf("[%s] %s - %s - %s - %s - %d %s\n",
			time.Now().Format(time.RFC3339),
			c.Request.Method,
			c.Request.URL.Path,
			user,
			clientIP,
			c.Writer.Status(),
			duration,
		)
	}
}
