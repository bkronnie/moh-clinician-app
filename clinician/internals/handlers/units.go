package handlers

import (
	"database/sql"
	"net/http"

	"github.com/alexedwards/scs/v2"
	"github.com/gin-gonic/gin"
)

func HandlerUnitForm(c *gin.Context, db *sql.DB, sessionManager *scs.SessionManager) {

}

func HandlerUnitSave(c *gin.Context, db *sql.DB, sessionManager *scs.SessionManager) {
	c.Redirect(http.StatusFound, "/foo")
}

func HandlerUnitList(c *gin.Context, db *sql.DB, sessionManager *scs.SessionManager) {

}
