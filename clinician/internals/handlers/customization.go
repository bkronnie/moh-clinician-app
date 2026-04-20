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

	historyPage := 1
	if parsed, err := strconv.Atoi(strings.TrimSpace(c.Query("history_page"))); err == nil && parsed > 0 {
		historyPage = parsed
	}
	historySize := 60
	historyOffset := (historyPage - 1) * historySize

	view, err := models.GetCustomizationView(c.Request.Context(), db, historySize, historyOffset)
	if err != nil {
		c.String(http.StatusInternalServerError, "failed to load customization data")
		return
	}

	historyPages := 1
	if view.HistoryTotal > 0 {
		historyPages = (view.HistoryTotal + historySize - 1) / historySize
	}
	if historyPage > historyPages {
		historyPage = historyPages
	}

	view.HistoryPage = historyPage
	view.HistorySize = historySize
	view.HistoryPages = historyPages
	if historyPage > 1 {
		view.HistoryPrev = buildCustomizationHistoryPageURL(historyPage - 1)
	}
	if historyPage < historyPages {
		view.HistoryNext = buildCustomizationHistoryPageURL(historyPage + 1)
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
	actionMode := strings.ToLower(strings.TrimSpace(c.PostForm("action_mode")))
	guard := strings.TrimSpace(c.PostForm("guard"))

	var opErr error
	var okMessage string
	switch actionMode {
	case "activate":
		opErr = models.SetReportDataElementActive(c.Request.Context(), db, actor.UserID, actor.EmpID, id, true)
		okMessage = "Data element activated"
	case "purge":
		opErr = models.PurgeReportDataElement(c.Request.Context(), db, actor.UserID, actor.EmpID, id, guard)
		okMessage = "Data element permanently deleted"
	default:
		opErr = models.SetReportDataElementActive(c.Request.Context(), db, actor.UserID, actor.EmpID, id, false)
		okMessage = "Data element deactivated"
	}

	if opErr != nil {
		redirectCustomizationError(c, tab, opErr.Error())
		return
	}

	redirectCustomizationOK(c, tab, okMessage)
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

func buildCustomizationHistoryPageURL(page int) string {
	if page <= 1 {
		return "/customization?tab=history"
	}
	q := url.Values{}
	q.Set("tab", "history")
	q.Set("history_page", strconv.Itoa(page))
	return "/customization?" + q.Encode()
}
