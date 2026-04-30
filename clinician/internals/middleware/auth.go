package middleware

import (
	"net/http"
	"strconv"

	"clinician/internals/models"
	"clinician/internals/utilities"

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

		roleName, err := models.UserRoleNameByID(c.Request.Context(), db, user.ID)
		if err != nil {
			c.Redirect(http.StatusFound, "/login")
			c.Abort()
			return
		}

		if utilities.RoleMatches(roleName, allowedRoles...) {
			c.Next()
			return
		}

		c.Redirect(http.StatusFound, "/")
		c.Abort()
	}
}
