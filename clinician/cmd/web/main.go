package main

import (
	"fmt"
	"net/http"
	"path/filepath"
	"time"

	"github.com/alexedwards/scs/v2"
	"github.com/gin-gonic/gin"

	"github.com/moh/clinician/internals/middleware"
	"github.com/moh/clinician/internals/routes"
	"github.com/moh/clinician/internals/utilities"

	_ "github.com/lib/pq"
)

func main() {
	confi := getConfig()

	connStr := "user=" + confi.Ux + " password='" + confi.Px + "' dbname=" + confi.Dx + " sslmode=disable"
	fmt.Println("system started")

	// Move two directories up and into the ui/static folder
	path := filepath.Join("..", "..", "ui", "static")

	router := gin.Default()
	router.Static("/static", path)

	// Initialize session manager
	sessionManager := scs.New()
	sessionManager.Lifetime = 30 * time.Minute
	sessionManager.Cookie.Persist = true
	sessionManager.Cookie.Secure = false // Set to true in production
	sessionManager.Cookie.HttpOnly = true

	// Use scs middleware to manage sessions automatically
	router.Use(func(c *gin.Context) {
		// Load and Save sessions automatically
		sessionManager.LoadAndSave(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// This automatically loads the session into the request context
			c.Request = r.WithContext(r.Context())
			c.Next()
		})).ServeHTTP(c.Writer, c.Request)
	})

	// Middleware for session management
	/* 	router.Use(func(c *gin.Context) {
		// Extract session token from cookie (or other source)
		token, err := c.Cookie("session_token")
		if err != nil || token == "" {
			// Handle missing token (e.g., redirect to login)
			c.Next() // Proceed without session or handle accordingly
			return
		}

		// Load session into context using token
		ctx, err := sessionManager.Load(c.Request.Context(), token)
		if err != nil {
			// Log the error and abort the request
			c.AbortWithError(http.StatusInternalServerError, err)
			return
		}

		// Replace Gin's request with the new context containing the session
		c.Request = c.Request.WithContext(ctx)
		c.Next()
	}) */

	// Database connection
	db, err := openDB(connStr)
	if err != nil {
		utilities.Danger("Failed to open database")
	}

	// Apply the request logging middleware
	router.Use(middleware.RequestLogger())

	// Set up routes
	routes.SetupRoutes(router, db, sessionManager)

	// Run the server
	router.Run(":8081")
}
