package handlers

import (
	"database/sql"
	"fmt"
	"net/http"

	"github.com/alexedwards/scs/v2"
	"github.com/gin-gonic/gin"
	"github.com/moh/clinician/internals/models"
	"github.com/moh/clinician/internals/utilities"
)

func HandlerDepartmentForm(c *gin.Context, db *sql.DB, sessionManager *scs.SessionManager) {

}

func HandlerDepartmentSave(c *gin.Context, db *sql.DB, sessionManager *scs.SessionManager) {
	c.Redirect(http.StatusFound, "/foo")
}

func HandlerDepartmentList(c *gin.Context, db *sql.DB, sessionManager *scs.SessionManager) {

	filter := ""
	departments, er := models.Departments(c, db, filter, 0, 0)
	if er != nil {
		fmt.Println(er)
	}

	data := GetPageData("your flash message")
	data.Form = departments

	ses, er := SES_SET(c, db, sessionManager)
	if er != nil {
		fmt.Printf("x1: " + er.Error())
	} else {
		data.Ses = ses
	}

	utilities.GenerateHTML(c, data, "base", "departments")

}
