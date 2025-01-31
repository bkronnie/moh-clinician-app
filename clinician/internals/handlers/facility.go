package handlers

import (
	"database/sql"

	"github.com/alexedwards/scs/v2"
	"github.com/gin-gonic/gin"
	"github.com/moh/clinician/internals/models"
	"github.com/moh/clinician/internals/utilities"
)

func HandlerFacilityForm(c *gin.Context, db *sql.DB, sessionManager *scs.SessionManager) {

}

func HandlerFacilitySave(c *gin.Context, db *sql.DB, sessionManager *scs.SessionManager) {

}

func HandlerFacilityList(c *gin.Context, db *sql.DB, sessionManager *scs.SessionManager) {
	var zdata any

	dts, er := models.Facilitys(c, db, "", 0, 0)
	if er == nil {
		zdata = dts
	} else {
		zdata = nil
	}

	data := Get_Session_Data(c, db, sessionManager, zdata)
	utilities.GenerateHTML(c, data, "base", "facilities")
}
