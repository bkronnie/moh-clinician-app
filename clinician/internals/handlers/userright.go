package handlers

import (
	"database/sql"
	"net/http"

	"github.com/alexedwards/scs/v2"
	"github.com/gin-gonic/gin"
)

func HandlerUserRightForm(c *gin.Context, db *sql.DB, sessionManager *scs.SessionManager) {
	c.Redirect(http.StatusFound, "/customization")
}

func HandlerUserRightSave(c *gin.Context, db *sql.DB, sessionManager *scs.SessionManager) {
	c.Redirect(http.StatusFound, "/customization")
}

func HandlerUserRightList(c *gin.Context, db *sql.DB, sessionManager *scs.SessionManager) {
	c.Redirect(http.StatusFound, "/customization")
}
