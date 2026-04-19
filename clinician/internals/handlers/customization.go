package handlers

import (
	"database/sql"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/alexedwards/scs/v2"
	"github.com/gin-gonic/gin"
	"github.com/moh/clinician/internals/models"
	"github.com/moh/clinician/internals/utilities"
)

func HandlerCustomization(c *gin.Context, db *sql.DB, sessionManager *scs.SessionManager) {
	sessionData, ok := Get_Session_Data(c, db, sessionManager, nil).(utilities.TemplateData)
	if !ok {
		c.String(http.StatusInternalServerError, "unable to load session")
		return
	}

	view, err := models.GetCustomizationView(c.Request.Context(), db, 120)
	if err != nil {
		c.String(http.StatusInternalServerError, "failed to load customization data")
		return
	}

	view.Success = strings.TrimSpace(c.Query("ok"))
	view.Error = strings.TrimSpace(c.Query("err"))
	sessionData.Form = view

	utilities.GenerateHTML(c, sessionData, "base", "customization")
}

func HandlerCustomizationFacilitySave(c *gin.Context, db *sql.DB, sessionManager *scs.SessionManager) {
	actor, err := SES_SET(c, db, sessionManager)
	if err != nil {
		redirectCustomizationError(c, "Unable to resolve user session")
		return
	}

	id := parsePositiveInt64(c.PostForm("id"))
	name := strings.TrimSpace(c.PostForm("name"))
	level := strings.TrimSpace(c.PostForm("level"))

	if err := models.SaveCustomizationFacility(c.Request.Context(), db, actor.UserID, actor.EmpID, id, name, level); err != nil {
		redirectCustomizationError(c, err.Error())
		return
	}

	if id > 0 {
		redirectCustomizationOK(c, "Facility updated")
		return
	}
	redirectCustomizationOK(c, "Facility created")
}

func HandlerCustomizationFacilityDelete(c *gin.Context, db *sql.DB, sessionManager *scs.SessionManager) {
	actor, err := SES_SET(c, db, sessionManager)
	if err != nil {
		redirectCustomizationError(c, "Unable to resolve user session")
		return
	}

	id := parsePositiveInt64(c.Param("id"))
	if err := models.DeleteCustomizationFacility(c.Request.Context(), db, actor.UserID, actor.EmpID, id); err != nil {
		redirectCustomizationError(c, err.Error())
		return
	}

	redirectCustomizationOK(c, "Facility deleted")
}

func HandlerCustomizationDepartmentSave(c *gin.Context, db *sql.DB, sessionManager *scs.SessionManager) {
	actor, err := SES_SET(c, db, sessionManager)
	if err != nil {
		redirectCustomizationError(c, "Unable to resolve user session")
		return
	}

	id := parsePositiveInt64(c.PostForm("id"))
	name := strings.TrimSpace(c.PostForm("name"))

	if err := models.SaveCustomizationDepartment(c.Request.Context(), db, actor.UserID, actor.EmpID, id, name); err != nil {
		redirectCustomizationError(c, err.Error())
		return
	}

	if id > 0 {
		redirectCustomizationOK(c, "Department updated")
		return
	}
	redirectCustomizationOK(c, "Department created")
}

func HandlerCustomizationDepartmentDelete(c *gin.Context, db *sql.DB, sessionManager *scs.SessionManager) {
	actor, err := SES_SET(c, db, sessionManager)
	if err != nil {
		redirectCustomizationError(c, "Unable to resolve user session")
		return
	}

	id := parsePositiveInt64(c.Param("id"))
	if err := models.DeleteCustomizationDepartment(c.Request.Context(), db, actor.UserID, actor.EmpID, id); err != nil {
		redirectCustomizationError(c, err.Error())
		return
	}

	redirectCustomizationOK(c, "Department deleted")
}

func HandlerCustomizationRoleSave(c *gin.Context, db *sql.DB, sessionManager *scs.SessionManager) {
	actor, err := SES_SET(c, db, sessionManager)
	if err != nil {
		redirectCustomizationError(c, "Unable to resolve user session")
		return
	}

	id := parsePositiveInt64(c.PostForm("id"))
	name := strings.TrimSpace(c.PostForm("name"))

	if err := models.SaveCustomizationRole(c.Request.Context(), db, actor.UserID, actor.EmpID, id, name); err != nil {
		redirectCustomizationError(c, err.Error())
		return
	}

	if id > 0 {
		redirectCustomizationOK(c, "Role updated")
		return
	}
	redirectCustomizationOK(c, "Role created")
}

func HandlerCustomizationRoleDelete(c *gin.Context, db *sql.DB, sessionManager *scs.SessionManager) {
	actor, err := SES_SET(c, db, sessionManager)
	if err != nil {
		redirectCustomizationError(c, "Unable to resolve user session")
		return
	}

	id := parsePositiveInt64(c.Param("id"))
	if err := models.DeleteCustomizationRole(c.Request.Context(), db, actor.UserID, actor.EmpID, id); err != nil {
		redirectCustomizationError(c, err.Error())
		return
	}

	redirectCustomizationOK(c, "Role deleted")
}

func HandlerCustomizationClinicalRoleSave(c *gin.Context, db *sql.DB, sessionManager *scs.SessionManager) {
	actor, err := SES_SET(c, db, sessionManager)
	if err != nil {
		redirectCustomizationError(c, "Unable to resolve user session")
		return
	}

	id := parsePositiveInt64(c.PostForm("id"))
	title := strings.TrimSpace(c.PostForm("title"))

	if err := models.SaveCustomizationClinicalRole(c.Request.Context(), db, actor.UserID, actor.EmpID, id, title); err != nil {
		redirectCustomizationError(c, err.Error())
		return
	}

	if id > 0 {
		redirectCustomizationOK(c, "Clinical role updated")
		return
	}
	redirectCustomizationOK(c, "Clinical role created")
}

func HandlerCustomizationClinicalRoleDelete(c *gin.Context, db *sql.DB, sessionManager *scs.SessionManager) {
	actor, err := SES_SET(c, db, sessionManager)
	if err != nil {
		redirectCustomizationError(c, "Unable to resolve user session")
		return
	}

	id := parsePositiveInt64(c.Param("id"))
	if err := models.DeleteCustomizationClinicalRole(c.Request.Context(), db, actor.UserID, actor.EmpID, id); err != nil {
		redirectCustomizationError(c, err.Error())
		return
	}

	redirectCustomizationOK(c, "Clinical role deleted")
}

func parsePositiveInt64(raw string) int64 {
	v, err := strconv.ParseInt(strings.TrimSpace(raw), 10, 64)
	if err != nil || v < 0 {
		return 0
	}
	return v
}

func redirectCustomizationOK(c *gin.Context, msg string) {
	c.Redirect(http.StatusFound, "/customization?ok="+url.QueryEscape(msg))
}

func redirectCustomizationError(c *gin.Context, msg string) {
	c.Redirect(http.StatusFound, "/customization?err="+url.QueryEscape(msg))
}
