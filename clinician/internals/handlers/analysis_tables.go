package handlers

import (
	"database/sql"
	"log"
	"net/http"

	"github.com/alexedwards/scs/v2"
	"github.com/gin-gonic/gin"
	"github.com/moh/clinician/internals/models"
	"github.com/moh/clinician/internals/utilities"
)

// AnalysisTablesView is the view model for the Analysis Tables page.
type AnalysisTablesView struct {
	SelectedYear         int
	SelectedMonth        int
	SelectedWeek         int
	SelectedFacility     int
	SelectedDepartment   int
	SelectedWeekLabel    string
	AvailableYears       []int
	AvailableMonths      []models.DashboardFilterOption
	AvailableWeeks       []models.ClinicianWeekOption
	AvailableFacilities  []models.DashboardFilterOption
	AvailableDepartments []models.DashboardFilterOption
	Tables               []models.FacilityPerformanceTable
	ClearURL             string
	CurrentURL           string
}

// HandlerAnalysisTables renders the facility-grouped staff performance tables page.
func HandlerAnalysisTables(c *gin.Context, db *sql.DB, sessionManager *scs.SessionManager) {
	sessionData, ok := Get_Session_Data(c, db, sessionManager, nil).(utilities.TemplateData)
	if !ok {
		c.Redirect(http.StatusFound, "/login")
		return
	}

	requestedYear, hasYear := parseOptionalIntQuery(c, "year")
	requestedMonth, hasMonth := parseOptionalIntQuery(c, "month")
	requestedWeek, hasWeek := parseOptionalIntQuery(c, "week")
	selectedFacility, _ := parseOptionalIntQuery(c, "facility")
	selectedDepartment, _ := parseOptionalIntQuery(c, "department")

	selectedYear, selectedMonth, selectedWeek,
		availableYears, availableMonths, availableWeeks,
		selectedWeekLabel, periodStart, periodEnd,
		err := resolveNationalDashboardPeriod(
		c, db,
		requestedYear, hasYear,
		requestedMonth, hasMonth,
		requestedWeek, hasWeek,
	)
	if err != nil {
		log.Println("HandlerAnalysisTables: resolveNationalDashboardPeriod:", err)
		c.Status(http.StatusInternalServerError)
		return
	}

	availableFacilities, err := models.GetDashboardFacilityOptions(c.Request.Context(), db)
	if err != nil {
		log.Println("HandlerAnalysisTables: GetDashboardFacilityOptions:", err)
		c.Status(http.StatusInternalServerError)
		return
	}

	availableDepartments, err := models.GetDashboardDepartmentOptions(c.Request.Context(), db, selectedFacility)
	if err != nil {
		log.Println("HandlerAnalysisTables: GetDashboardDepartmentOptions:", err)
		c.Status(http.StatusInternalServerError)
		return
	}

	tables, err := models.GetFacilityStaffPerformanceTables(c.Request.Context(), db, periodStart, periodEnd, selectedFacility, selectedDepartment)
	if err != nil {
		log.Println("HandlerAnalysisTables: GetFacilityStaffPerformanceTables:", err)
		c.Status(http.StatusInternalServerError)
		return
	}

	view := AnalysisTablesView{
		SelectedYear:         selectedYear,
		SelectedMonth:        selectedMonth,
		SelectedWeek:         selectedWeek,
		SelectedFacility:     selectedFacility,
		SelectedDepartment:   selectedDepartment,
		SelectedWeekLabel:    selectedWeekLabel,
		AvailableYears:       availableYears,
		AvailableMonths:      availableMonths,
		AvailableWeeks:       availableWeeks,
		AvailableFacilities:  availableFacilities,
		AvailableDepartments: availableDepartments,
		Tables:               tables,
		ClearURL:             "/analysis/tables",
		CurrentURL:           c.Request.URL.RequestURI(),
	}

	sessionData.Form = view

	utilities.GenerateHTML(c, sessionData, "base", "analysis-tables")
}
