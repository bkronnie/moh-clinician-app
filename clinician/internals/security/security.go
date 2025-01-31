package security

import (
	"fmt"
	"log"
	"net/http"
	"strconv"

	"github.com/alexedwards/scs/v2"
	"github.com/gin-gonic/gin"
)

func GetCurrentUser(c *gin.Context, sessionManager *scs.SessionManager) int {

	userID := sessionManager.GetString(c.Request.Context(), "userID")

	log.Printf("Retrieved userID from session: %s", userID)

	userInt, er := strconv.Atoi(userID)
	if er != nil {
		fmt.Println("User not found or not an int")
		return 0
	}
	return userInt

}

func IsAuthenticated(c *gin.Context, sessionManager *scs.SessionManager) bool {

	userID := sessionManager.GetString(c.Request.Context(), "userID")
	if userID == "" {
		return false
	}
	return true
}

func IfAuthenticated(c *gin.Context, sessionManager *scs.SessionManager) {

	userID := sessionManager.GetString(c.Request.Context(), "userID")
	if userID == "" {
		c.Redirect(http.StatusFound, "/login")
		c.Abort()
		return
	}
}
