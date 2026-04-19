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
	view.ActiveTab = sanitizeCustomizationTab(c.Query("tab"))
	sessionData.Form = view

	utilities.GenerateHTML(c, sessionData, "base", "customization")
}

func HandlerCustomizationFacilitySave(c *gin.Context, db *sql.DB, sessionManager *scs.SessionManager) {
	tab := sanitizeCustomizationTab(c.PostForm("tab"))

	actor, err := SES_SET(c, db, sessionManager)
	if err != nil {
		redirectCustomizationError(c, tab, "Unable to resolve user session")
		return
	}

	id := parsePositiveInt64(c.PostForm("id"))
	name := strings.TrimSpace(c.PostForm("name"))
	level := strings.TrimSpace(c.PostForm("level"))

	if err := models.SaveCustomizationFacility(c.Request.Context(), db, actor.UserID, actor.EmpID, id, name, level); err != nil {
		redirectCustomizationError(c, tab, err.Error())
		return
	}

	if id > 0 {
		redirectCustomizationOK(c, tab, "Facility updated")
		return
	}
	redirectCustomizationOK(c, tab, "Facility created")
}

func HandlerCustomizationFacilityDelete(c *gin.Context, db *sql.DB, sessionManager *scs.SessionManager) {
	tab := sanitizeCustomizationTab(c.PostForm("tab"))

	actor, err := SES_SET(c, db, sessionManager)
	if err != nil {
		redirectCustomizationError(c, tab, "Unable to resolve user session")
		return
	}

	id := parsePositiveInt64(c.Param("id"))
	if err := models.DeleteCustomizationFacility(c.Request.Context(), db, actor.UserID, actor.EmpID, id); err != nil {
		redirectCustomizationError(c, tab, err.Error())
		return
	}

	redirectCustomizationOK(c, tab, "Facility deleted")
}

func HandlerCustomizationDepartmentSave(c *gin.Context, db *sql.DB, sessionManager *scs.SessionManager) {
	tab := sanitizeCustomizationTab(c.PostForm("tab"))

	actor, err := SES_SET(c, db, sessionManager)
	if err != nil {
		redirectCustomizationError(c, tab, "Unable to resolve user session")
		return
	}

	id := parsePositiveInt64(c.PostForm("id"))
	name := strings.TrimSpace(c.PostForm("name"))

	if err := models.SaveCustomizationDepartment(c.Request.Context(), db, actor.UserID, actor.EmpID, id, name); err != nil {
		redirectCustomizationError(c, tab, err.Error())
		return
	}

	if id > 0 {
		redirectCustomizationOK(c, tab, "Department updated")
		return
	}
	redirectCustomizationOK(c, tab, "Department created")
}

func HandlerCustomizationDepartmentDelete(c *gin.Context, db *sql.DB, sessionManager *scs.SessionManager) {
	tab := sanitizeCustomizationTab(c.PostForm("tab"))

	actor, err := SES_SET(c, db, sessionManager)
	if err != nil {
		redirectCustomizationError(c, tab, "Unable to resolve user session")
		return
	}

	id := parsePositiveInt64(c.Param("id"))
	if err := models.DeleteCustomizationDepartment(c.Request.Context(), db, actor.UserID, actor.EmpID, id); err != nil {
		redirectCustomizationError(c, tab, err.Error())
		return
	}

	redirectCustomizationOK(c, tab, "Department deleted")
}

func HandlerCustomizationRoleSave(c *gin.Context, db *sql.DB, sessionManager *scs.SessionManager) {
	tab := sanitizeCustomizationTab(c.PostForm("tab"))

	actor, err := SES_SET(c, db, sessionManager)
	if err != nil {
		redirectCustomizationError(c, tab, "Unable to resolve user session")
		return
	}

	id := parsePositiveInt64(c.PostForm("id"))
	name := strings.TrimSpace(c.PostForm("name"))

	if err := models.SaveCustomizationRole(c.Request.Context(), db, actor.UserID, actor.EmpID, id, name); err != nil {
		redirectCustomizationError(c, tab, err.Error())
		return
	}

	if id > 0 {
		redirectCustomizationOK(c, tab, "Role updated")
		return
	}
	redirectCustomizationOK(c, tab, "Role created")
}

func HandlerCustomizationRoleDelete(c *gin.Context, db *sql.DB, sessionManager *scs.SessionManager) {
	tab := sanitizeCustomizationTab(c.PostForm("tab"))

	actor, err := SES_SET(c, db, sessionManager)
	if err != nil {
		redirectCustomizationError(c, tab, "Unable to resolve user session")
		return
	}

	id := parsePositiveInt64(c.Param("id"))
	if err := models.DeleteCustomizationRole(c.Request.Context(), db, actor.UserID, actor.EmpID, id); err != nil {
		redirectCustomizationError(c, tab, err.Error())
		return
	}

	redirectCustomizationOK(c, tab, "Role deleted")
}

func HandlerCustomizationClinicalRoleSave(c *gin.Context, db *sql.DB, sessionManager *scs.SessionManager) {
	tab := sanitizeCustomizationTab(c.PostForm("tab"))

	actor, err := SES_SET(c, db, sessionManager)
	if err != nil {
		redirectCustomizationError(c, tab, "Unable to resolve user session")
		return
	}

	id := parsePositiveInt64(c.PostForm("id"))
	title := strings.TrimSpace(c.PostForm("title"))

	if err := models.SaveCustomizationClinicalRole(c.Request.Context(), db, actor.UserID, actor.EmpID, id, title); err != nil {
		redirectCustomizationError(c, tab, err.Error())
		return
	}

	if id > 0 {
		redirectCustomizationOK(c, tab, "Clinical role updated")
		return
	}
	redirectCustomizationOK(c, tab, "Clinical role created")
}

func HandlerCustomizationClinicalRoleDelete(c *gin.Context, db *sql.DB, sessionManager *scs.SessionManager) {
	tab := sanitizeCustomizationTab(c.PostForm("tab"))

	actor, err := SES_SET(c, db, sessionManager)
	if err != nil {
		redirectCustomizationError(c, tab, "Unable to resolve user session")
		return
	}

	id := parsePositiveInt64(c.Param("id"))
	if err := models.DeleteCustomizationClinicalRole(c.Request.Context(), db, actor.UserID, actor.EmpID, id); err != nil {
		redirectCustomizationError(c, tab, err.Error())
		return
	}

	redirectCustomizationOK(c, tab, "Clinical role deleted")
}

func HandlerCustomizationDataElementSave(c *gin.Context, db *sql.DB, sessionManager *scs.SessionManager) {
	tab := sanitizeCustomizationTab(c.PostForm("tab"))

	actor, err := SES_SET(c, db, sessionManager)
	if err != nil {
		redirectCustomizationError(c, tab, "Unable to resolve user session")
		return
	}

	id := parsePositiveInt64(c.PostForm("id"))
	elementKey := strings.TrimSpace(c.PostForm("element_key"))
	columnName := strings.TrimSpace(c.PostForm("column_name"))
	displayName := strings.TrimSpace(c.PostForm("display_name"))

	if err := models.SaveReportDataElement(c.Request.Context(), db, actor.UserID, actor.EmpID, id, elementKey, columnName, displayName); err != nil {
		redirectCustomizationError(c, tab, err.Error())
		return
	}

	if id > 0 {
		redirectCustomizationOK(c, tab, "Data element updated")
		return
	}
	redirectCustomizationOK(c, tab, "Data element created")
}

func HandlerCustomizationDataElementDelete(c *gin.Context, db *sql.DB, sessionManager *scs.SessionManager) {
	tab := sanitizeCustomizationTab(c.PostForm("tab"))

	actor, err := SES_SET(c, db, sessionManager)
	if err != nil {
		redirectCustomizationError(c, tab, "Unable to resolve user session")
		return
	}

	id := parsePositiveInt64(c.Param("id"))
	if err := models.DeleteReportDataElement(c.Request.Context(), db, actor.UserID, actor.EmpID, id); err != nil {
		redirectCustomizationError(c, tab, err.Error())
		return
	}

	redirectCustomizationOK(c, tab, "Data element deleted")
}

func parsePositiveInt64(raw string) int64 {
	v, err := strconv.ParseInt(strings.TrimSpace(raw), 10, 64)
	if err != nil || v < 0 {
		return 0
	}
	return v
}

func redirectCustomizationOK(c *gin.Context, tab string, msg string) {
	q := url.Values{}
	q.Set("ok", msg)
	q.Set("tab", sanitizeCustomizationTab(tab))
	c.Redirect(http.StatusFound, "/customization?"+q.Encode())
}

func redirectCustomizationError(c *gin.Context, tab string, msg string) {
	q := url.Values{}
	q.Set("err", msg)
	q.Set("tab", sanitizeCustomizationTab(tab))
	c.Redirect(http.StatusFound, "/customization?"+q.Encode())
}

func sanitizeCustomizationTab(raw string) string {
	tab := strings.ToLower(strings.TrimSpace(raw))
	switch tab {
	case "facilities", "departments", "roles", "clinical-roles", "data-elements", "history":
		return tab
	default:
		return "facilities"
	}
}
