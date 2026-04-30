package handlers

import (
	"database/sql"
	"fmt"
	"net/http"

	"clinician/internals/models"
	"clinician/internals/security"
	"clinician/internals/utilities"

	"github.com/alexedwards/scs/v2"
	"github.com/gin-gonic/gin"
)

func HandlerLoginForm(c *gin.Context, db *sql.DB, sessionManager *scs.SessionManager) {
	viewData := gin.H{
		"Message": c.Query("message"),
		"Error":   c.Query("error"),
	}

	if security.IsAuthenticated(c, sessionManager) {
		// If authenticated, redirect to the home page or any other page
		c.Redirect(http.StatusFound, "/")
		return
	}

	count, er := models.UserCount(c, db, "")
	if er != nil {
		//c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal Server Error"})
		utilities.Danger(er.Error())
		utilities.GenerateHTML(c, viewData, "login")
		return
	}

	if count > 0 {
		fmt.Println("login form loading")
		utilities.GenerateHTML(c, viewData, "login")
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
