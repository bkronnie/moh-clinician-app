package middleware

import (
	"net/http"

	"github.com/alexedwards/scs/v2"
	"github.com/gin-gonic/gin"
)

func AuthMiddleware(sessionManager *scs.SessionManager) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Retrieve userID from session
		userID := sessionManager.GetString(c.Request.Context(), "userID")

		if userID == "" {
			// Redirect to the login page
			c.Redirect(http.StatusFound, "/login")
			c.Abort()
			return
		}

		// Continue if authenticated
		c.Next()
	}
}
