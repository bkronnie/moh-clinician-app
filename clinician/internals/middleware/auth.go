package middleware

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/alexedwards/scs/v2"
	"github.com/gin-gonic/gin"
	"github.com/moh/clinician/internals/models"
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

func RequireRoles(db models.DB, sessionManager *scs.SessionManager, allowedRoles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := sessionManager.GetString(c.Request.Context(), "userID")
		if userID == "" {
			c.Redirect(http.StatusFound, "/login")
			c.Abort()
			return
		}

		id, err := strconv.Atoi(userID)
		if err != nil {
			c.Redirect(http.StatusFound, "/login")
			c.Abort()
			return
		}

		user, err := models.UserByID(c.Request.Context(), db, id)
		if err != nil {
			c.Redirect(http.StatusFound, "/login")
			c.Abort()
			return
		}

		for _, role := range allowedRoles {
			if strings.EqualFold(user.Rights, role) {
				c.Next()
				return
			}
		}

		c.Redirect(http.StatusFound, "/")
		c.Abort()
	}
}
