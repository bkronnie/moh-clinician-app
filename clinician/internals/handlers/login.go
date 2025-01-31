package handlers

import (
	"database/sql"
	"fmt"
	"net/http"

	"github.com/alexedwards/scs/v2"
	"github.com/gin-gonic/gin"
	"github.com/moh/clinician/internals/models"
	"github.com/moh/clinician/internals/security"
	"github.com/moh/clinician/internals/utilities"
)

func HandlerLoginForm(c *gin.Context, db *sql.DB, sessionManager *scs.SessionManager) {

	if security.IsAuthenticated(c, sessionManager) {
		// If authenticated, redirect to the home page or any other page
		c.Redirect(http.StatusFound, "/")
		return
	}

	count, er := models.UserCount(c, db, "")
	if er != nil {
		//c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal Server Error"})
		utilities.Danger(er.Error())
		utilities.GenerateHTML(c, nil, "login")
		return
	}

	if count > 0 {
		fmt.Println("login form loading")
		utilities.GenerateHTML(c, nil, "login")
	} else {
		fmt.Println("setup form loading")
		utilities.GenerateHTML(c, nil, "setup")
	}

}

func HandlerLoginForgot(c *gin.Context) {
	fmt.Println("accessing help")
	utilities.GenerateHTML(c, nil, "forgot")
}

func HandlerHelp(c *gin.Context) {
	fmt.Println("accessing help")
	utilities.GenerateHTML(c, nil, "help")

}
