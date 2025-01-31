package handlers

import (
	"database/sql"
	"net/http"

	"github.com/alexedwards/scs/v2"
	"github.com/gin-gonic/gin"
)

func AuthUserHandler(c *gin.Context, db *sql.DB, sessionManager *scs.SessionManager) {
	c.Redirect(http.StatusFound, "/foo")
}

func HandlerUserForm(c *gin.Context, db *sql.DB, sessionManager *scs.SessionManager) {

}

func HandlerUserSave(c *gin.Context, db *sql.DB, sessionManager *scs.SessionManager) {
	c.Redirect(http.StatusFound, "/foo")
}

func HandlerUserList(c *gin.Context, db *sql.DB, sessionManager *scs.SessionManager) {

}
