package handlers

import (
	"database/sql"
	"errors"
	"fmt"
	"log"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/alexedwards/scs/v2"
	"github.com/gin-gonic/gin"
	"github.com/moh/clinician/internals/models"
	"github.com/moh/clinician/internals/utilities"
)

type ReportSubmissionsView struct {
	Role               string
	ViewMode           string
	Mode               string
	ShowWeekSummary    bool
	ScopeTitle         string
	ScopeSubtitle      string
	FilterSummary      string
	SelectedFacility   int
	FacilityOptions    []models.DashboardFilterOption
	SelectedDepartment int
	DepartmentOptions  []models.DashboardFilterOption
	SelectedStatus     string
	SelectedYear       int
	SelectedMonth      int
	SelectedWeek       int
	SelectedWeekLabel  string
	AvailableYears     []int
	AvailableMonths    []models.DashboardFilterOption
	AvailableWeeks     []models.ClinicianWeekOption
	Rows               []*models.ReportSubmissionListRow
	FacilityRows       []*models.FacilitySubmissionSummaryRow
	WeekRows           []*ReportSubmissionWeekSummaryRow
	CurrentURL         string
	AllURL             string
	SubmittedURL       string
	PendingURL         string
	ApprovedURL        string
	DeclinedURL        string
	DraftURL           string
	ExportCSVURL       string
	ExportPDFURL       string
	CanApprove         bool
	CanView            bool
	CanSubmitAll       bool
	SubmitAllURL       string
	PendingCount       int
	ClearFiltersURL    string
	FacilityModeURL    string
	BackToWeeksURL     string
	Page               int
	PageSize           int
	TotalRows          int
	TotalPages         int
	PrevPageURL        string
	NextPageURL        string
}

type ReportSubmissionWeekSummaryRow struct {
	Summary      *models.FacilitySubmissionSummaryRow
	DrilldownURL string
}

func buildReportSubmissionsView(c *gin.Context, db *sql.DB, sesDetails utilities.SessionDetails, paginate bool) (ReportSubmissionsView, error) {
	requestedFacility, hasFacility := parseOptionalIntQuery(c, "facility")
	requestedDepartment, hasDepartment := parseOptionalIntQuery(c, "department")
	requestedYear, hasYear := parseOptionalIntQuery(c, "year")
	requestedMonth, hasMonth := parseOptionalIntQuery(c, "month")
	requestedWeek, hasWeek := parseOptionalIntQuery(c, "week")
	selectedStatus := normalizeReportSubmissionStatus(c.Query("status"))
	viewMode := normalizeReportSubmissionViewMode(roleFromRights(sesDetails.Rights), c.Query("view"))
	requestedMode := normalizeReportSubmissionMode(roleFromRights(sesDetails.Rights), c.Query("mode"))
	allowDraftFilter := roleFromRights(sesDetails.Rights) == utilities.RoleFacilityAdmin && requestedMode != "reports"
	if roleFromRights(sesDetails.Rights) != utilities.RoleStaff && !allowDraftFilter && selectedStatus == "draft" {
		selectedStatus = "all"
	}

	selectedYear, selectedMonth, selectedWeek, availableYears, availableMonths, availableWeeks, selectedWeekLabel, err := resolveReportSubmissionPeriod(c, db, requestedYear, hasYear, requestedMonth, hasMonth, requestedWeek, hasWeek)
	if err != nil {
		return ReportSubmissionsView{}, err
	}

	facilityOptions, err := models.GetDashboardFacilityOptions(c.Request.Context(), db)
	if err != nil {
		return ReportSubmissionsView{}, err
	}

	view := ReportSubmissionsView{
		Role:              roleFromRights(sesDetails.Rights),
		ViewMode:          viewMode,
		Mode:              requestedMode,
		ShowWeekSummary:   roleFromRights(sesDetails.Rights) == utilities.RoleFacilityAdmin && requestedMode != "reports",
		SelectedStatus:    selectedStatus,
		SelectedYear:      selectedYear,
		SelectedMonth:     selectedMonth,
		SelectedWeek:      selectedWeek,
		SelectedWeekLabel: selectedWeekLabel,
		AvailableYears:    availableYears,
		AvailableMonths:   availableMonths,
		AvailableWeeks:    availableWeeks,
		CanApprove:        roleFromRights(sesDetails.Rights) == utilities.RoleFacilityAdmin || roleFromRights(sesDetails.Rights) == utilities.RoleNationalAdmin,
		CanView:           roleFromRights(sesDetails.Rights) != utilities.RoleStaff,
		CanSubmitAll:      roleFromRights(sesDetails.Rights) == utilities.RoleFacilityAdmin,
		Page:              1,
		PageSize:          50,
	}

	if paginate {
		if requestedPage, hasPage := parseOptionalIntQuery(c, "page"); hasPage && requestedPage > 0 {
			view.Page = requestedPage
		}
	}

	scopeFacilityID := int64(0)
	scopeEmployeeID := int64(0)
	if roleFromRights(sesDetails.Rights) == utilities.RoleStaff {
		scopeEmployeeID = sesDetails.EmpID
		view.ScopeTitle = "My Reports"
		view.ScopeSubtitle = "Review all of your saved, submitted, approved, and declined reports in one place using the same filtered reports screen used across the application."
		view.SelectedFacility = int(sesDetails.HFID)
		view.FacilityOptions = []models.DashboardFilterOption{{
			ID:   int(sesDetails.HFID),
			Name: sesDetails.HFName,
		}}
	} else if roleFromRights(sesDetails.Rights) == utilities.RoleFacilityAdmin {
		scopeFacilityID = sesDetails.HFID
		view.SelectedFacility = int(sesDetails.HFID)
		view.FacilityOptions = []models.DashboardFilterOption{{
			ID:   int(sesDetails.HFID),
			Name: sesDetails.HFName,
		}}
		view.ScopeTitle = "Facility Report Submissions"
		view.ScopeSubtitle = "Review all report submissions for your facility, inspect the saved data in read-only mode, approve pending submissions, and submit the full weekly batch at once."
	} else {
		if hasFacility {
			view.SelectedFacility = requestedFacility
		}
		view.FacilityOptions = facilityOptions
		view.ScopeTitle = "National Report Submissions"
		view.ScopeSubtitle = "Review report submissions across facilities. The default view is facility-level weekly submission status with drill-down to staff rows."
	}

	departmentOptions, err := models.GetReportSubmissionDepartmentOptions(c.Request.Context(), db, scopeFacilityID, view.SelectedFacility)
	if err != nil {
		return ReportSubmissionsView{}, err
	}
	view.DepartmentOptions = departmentOptions
	if roleFromRights(sesDetails.Rights) != utilities.RoleStaff && hasDepartment && containsFilterOption(departmentOptions, requestedDepartment) {
		view.SelectedDepartment = requestedDepartment
	}

	if roleFromRights(sesDetails.Rights) == utilities.RoleFacilityAdmin && view.ShowWeekSummary {
		view.PageSize = 0
		view.Page = 1
		weekRows, err := models.GetFacilityWeeklySubmissionSummaries(c.Request.Context(), db, int(sesDetails.HFID), view.SelectedDepartment, selectedStatus, selectedYear, selectedMonth, selectedWeek)
		if err != nil {
			return ReportSubmissionsView{}, err
		}
		view.FacilityRows = weekRows
		view.TotalRows = len(weekRows)
		for _, row := range weekRows {
			if row == nil {
				continue
			}
			if row.PendingCount > 0 {
				view.PendingCount += row.PendingCount
			}
			drilldownURL := buildFacilityWeekDrilldownURL(view.SelectedFacility, view.SelectedDepartment, selectedStatus, row)
			view.WeekRows = append(view.WeekRows, &ReportSubmissionWeekSummaryRow{Summary: row, DrilldownURL: drilldownURL})
		}
	} else if roleFromRights(sesDetails.Rights) == utilities.RoleNationalAdmin && viewMode == "facility" {
		view.PageSize = 0
		view.Page = 1
		facilityRows, err := models.GetFacilitySubmissionSummaries(c.Request.Context(), db, view.SelectedFacility, selectedStatus, selectedYear, selectedMonth, selectedWeek)
		if err != nil {
			return ReportSubmissionsView{}, err
		}
		view.FacilityRows = facilityRows
		view.TotalRows = len(facilityRows)
		for _, row := range facilityRows {
			view.PendingCount += row.PendingCount
		}
	} else {
		limit := 0
		offset := 0
		if paginate {
			limit = view.PageSize
			offset = (view.Page - 1) * view.PageSize
		}

		rows, totalRows, err := models.GetReportSubmissionsPaged(c.Request.Context(), db, scopeFacilityID, scopeEmployeeID, sesDetails.EmpID, view.SelectedFacility, view.SelectedDepartment, selectedStatus, selectedYear, selectedMonth, selectedWeek, limit, offset)
		if err != nil {
			return ReportSubmissionsView{}, err
		}
		view.Rows = rows
		view.TotalRows = totalRows

		for _, row := range rows {
			if row.SubmitStatus.Valid && row.SubmitStatus.String == "Submitted" && (!row.ReportStatus.Valid || !isFinalReportReviewStatus(row.ReportStatus.String)) {
				view.PendingCount++
			}
		}
	}

	view.CurrentURL = buildReportSubmissionsURLWithModeAndPage(view.SelectedFacility, view.SelectedDepartment, view.SelectedYear, view.SelectedMonth, view.SelectedWeek, view.SelectedStatus, view.ViewMode, view.Mode, view.Page)
	view.AllURL = buildReportSubmissionsURLWithModeAndPage(view.SelectedFacility, view.SelectedDepartment, view.SelectedYear, view.SelectedMonth, view.SelectedWeek, "all", view.ViewMode, view.Mode, 1)
	view.SubmittedURL = buildReportSubmissionsURLWithModeAndPage(view.SelectedFacility, view.SelectedDepartment, view.SelectedYear, view.SelectedMonth, view.SelectedWeek, "submitted", view.ViewMode, view.Mode, 1)
	view.PendingURL = buildReportSubmissionsURLWithModeAndPage(view.SelectedFacility, view.SelectedDepartment, view.SelectedYear, view.SelectedMonth, view.SelectedWeek, "pending", view.ViewMode, view.Mode, 1)
	view.ApprovedURL = buildReportSubmissionsURLWithModeAndPage(view.SelectedFacility, view.SelectedDepartment, view.SelectedYear, view.SelectedMonth, view.SelectedWeek, "approved", view.ViewMode, view.Mode, 1)
	view.DeclinedURL = buildReportSubmissionsURLWithModeAndPage(view.SelectedFacility, view.SelectedDepartment, view.SelectedYear, view.SelectedMonth, view.SelectedWeek, "declined", view.ViewMode, view.Mode, 1)
	view.DraftURL = buildReportSubmissionsURLWithModeAndPage(view.SelectedFacility, view.SelectedDepartment, view.SelectedYear, view.SelectedMonth, view.SelectedWeek, "draft", view.ViewMode, view.Mode, 1)
	view.ExportCSVURL = buildReportSubmissionsExportURLWithView("csv", view.SelectedFacility, view.SelectedDepartment, view.SelectedYear, view.SelectedMonth, view.SelectedWeek, view.SelectedStatus, view.ViewMode)
	view.ExportPDFURL = buildReportSubmissionsExportURLWithView("pdf", view.SelectedFacility, view.SelectedDepartment, view.SelectedYear, view.SelectedMonth, view.SelectedWeek, view.SelectedStatus, view.ViewMode)
	view.ClearFiltersURL = buildReportSubmissionsURLWithModeAndPage(0, 0, 0, 0, 0, "all", view.ViewMode, view.Mode, 1)
	view.FacilityModeURL = buildReportSubmissionsURLWithViewAndPage(0, 0, view.SelectedYear, view.SelectedMonth, view.SelectedWeek, view.SelectedStatus, "facility", 1)
	view.BackToWeeksURL = buildReportSubmissionsURLWithModeAndPage(view.SelectedFacility, view.SelectedDepartment, view.SelectedYear, view.SelectedMonth, 0, view.SelectedStatus, view.ViewMode, "weeks", 1)
	view.SubmitAllURL = "/reports/analysis/submit-all"
	view.FilterSummary = reportSubmissionFilterSummary(roleFromRights(sesDetails.Rights), selectedStatus, selectedWeekLabel)

	if view.PageSize > 0 {
		if view.TotalRows <= 0 {
			view.TotalPages = 1
		} else {
			view.TotalPages = (view.TotalRows + view.PageSize - 1) / view.PageSize
		}
		if view.Page < 1 {
			view.Page = 1
		}
		if view.Page > view.TotalPages {
			view.Page = view.TotalPages
		}
		if view.Page > 1 {
			view.PrevPageURL = buildReportSubmissionsURLWithModeAndPage(view.SelectedFacility, view.SelectedDepartment, view.SelectedYear, view.SelectedMonth, view.SelectedWeek, view.SelectedStatus, view.ViewMode, view.Mode, view.Page-1)
		}
		if view.Page < view.TotalPages {
			view.NextPageURL = buildReportSubmissionsURLWithModeAndPage(view.SelectedFacility, view.SelectedDepartment, view.SelectedYear, view.SelectedMonth, view.SelectedWeek, view.SelectedStatus, view.ViewMode, view.Mode, view.Page+1)
		}
	}

	return view, nil
}

func HandlerReportsAnalysis(c *gin.Context, db *sql.DB, sessionManager *scs.SessionManager) {
	sessionData, ok := Get_Session_Data(c, db, sessionManager, nil).(utilities.TemplateData)
	if !ok {
		log.Println("Failed to retrieve session data as TemplateData")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Unable to retrieve session data"})
		return
	}

	sesDetails, ok := sessionData.Ses.(utilities.SessionDetails)
	if !ok {
		log.Println("Failed to retrieve session details from TemplateData")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Unable to retrieve session details"})
		return
	}

	view, err := buildReportSubmissionsView(c, db, sesDetails, true)
	if err != nil {
		log.Printf("Error building report submissions view: %v", err)
		c.String(http.StatusInternalServerError, "Error retrieving report submissions")
		return
	}

	sessionData.Form = view
	utilities.GenerateHTML(c, sessionData, "base", "report-analysis")
}

func HandlerReportsAnalysisExport(c *gin.Context, db *sql.DB, sessionManager *scs.SessionManager) {
	sessionData, ok := Get_Session_Data(c, db, sessionManager, nil).(utilities.TemplateData)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Unable to retrieve session data"})
		return
	}

	sesDetails, ok := sessionData.Ses.(utilities.SessionDetails)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Unable to retrieve session details"})
		return
	}

	view, err := buildReportSubmissionsView(c, db, sesDetails, false)
	if err != nil {
		log.Printf("Error exporting report submissions: %v", err)
		c.String(http.StatusInternalServerError, "Error exporting report submissions")
		return
	}

	format := c.DefaultQuery("format", "csv")
	if format != "csv" && format != "pdf" {
		c.String(http.StatusBadRequest, "Invalid export format")
		return
	}

	headers := []string{"Facility", "Department", "Employee", "Week", "Entered On", "Submitted On", "Status", "Attendance", "Patients Reviewed", "Procedures"}
	csvRows := [][]string{}
	pdfLines := []string{
		view.ScopeTitle,
		fmt.Sprintf("Period: %s", view.SelectedWeekLabel),
		"",
	}

	for index, row := range view.Rows {
		status := reportSubmissionStatusLabel(row.SubmitStatus, row.ReportStatus)
		weekLabel := exportWeekRange(row.WeekStart, row.WeekStop)
		enteredOn := exportDateTime(row.EnteredOn)
		submittedOn := exportDateTime(row.SubmittedOn)

		csvRows = append(csvRows, []string{
			row.FacilityName,
			row.DepartmentName,
			row.EmployeeName,
			weekLabel,
			enteredOn,
			submittedOn,
			status,
			strconv.Itoa(row.Attendance),
			strconv.Itoa(row.PatientsReviewed),
			strconv.Itoa(row.Procedures),
		})

		pdfLines = append(pdfLines,
			fmt.Sprintf("%d. %s - %s", index+1, row.EmployeeName, row.FacilityName),
			fmt.Sprintf("   Department: %s | Week: %s", row.DepartmentName, weekLabel),
			fmt.Sprintf("   Status: %s | Entered: %s | Submitted: %s", status, enteredOn, submittedOn),
			fmt.Sprintf("   Attendance: %d | Patients Reviewed: %d | Procedures: %d", row.Attendance, row.PatientsReviewed, row.Procedures),
			"",
		)
	}

	filenameBase := buildReportSubmissionsExportFilename(view.SelectedFacility, view.SelectedDepartment, view.SelectedYear, view.SelectedMonth, view.SelectedWeek, view.SelectedStatus)
	if format == "pdf" {
		writeSimplePDFDownload(c, filenameBase+".pdf", view.ScopeTitle, pdfLines)
		return
	}
	writeCSVDownload(c, filenameBase+".csv", headers, csvRows)
}

func HandlerReportSubmissionView(c *gin.Context, db *sql.DB, sessionManager *scs.SessionManager) {
	sessionData, ok := Get_Session_Data(c, db, sessionManager, nil).(utilities.TemplateData)
	if !ok {
		log.Println("Failed to retrieve session data as TemplateData")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Unable to retrieve session data"})
		return
	}

	sesDetails, ok := sessionData.Ses.(utilities.SessionDetails)
	if !ok {
		log.Println("Failed to retrieve session details from TemplateData")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Unable to retrieve session details"})
		return
	}

	reportID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.String(http.StatusBadRequest, "Invalid report ID")
		return
	}

	scopeFacilityID := int64(0)
	if roleFromRights(sesDetails.Rights) == utilities.RoleFacilityAdmin {
		scopeFacilityID = sesDetails.HFID
	}

	reviewRole := sesDetails.Rights
	report, err := models.GetReportSubmissionByIDForReview(c.Request.Context(), db, reportID, scopeFacilityID, reviewRole)
	if err != nil {
		if err == sql.ErrNoRows {
			c.String(http.StatusNotFound, "Report submission not found")
			return
		}
		log.Printf("Error retrieving report submission: %v", err)
		c.String(http.StatusInternalServerError, "Unable to retrieve report submission")
		return
	}
	if roleFromRights(sesDetails.Rights) != utilities.RoleStaff {
		isDraft := report.SubmitStatus.String != "Submitted" && report.ReportStatus.String != "Approved" && report.ReportStatus.String != "Rejected" && report.ReportStatus.String != "Declined"
		if isDraft {
			c.String(http.StatusNotFound, "Report submission not found")
			return
		}
	}

	employee, err := models.EmployeeByID(c, db, int(report.EmployeeID))
	if err != nil {
		c.String(http.StatusInternalServerError, "Unable to load employee details")
		return
	}

	department, err := models.DepartmentByID(c, db, int(report.DepartmentID))
	if err != nil {
		c.String(http.StatusInternalServerError, "Unable to load department details")
		return
	}

	facility, err := models.FacilityByID(c, db, int(report.FacilityID))
	if err != nil {
		c.String(http.StatusInternalServerError, "Unable to load facility details")
		return
	}

	labels := resolveClinicianEntryLabels(c.Request.Context(), db)
	entryForm := ClinicianEntryView{
		EmployeeID:     report.EmployeeID,
		EmployeeName:   fmt.Sprintf("%s %s", employee.Fname.String, employee.Lname.String),
		DepartmentID:   report.DepartmentID,
		DepartmentName: department.DepartmentName.String,
		FacilityName:   facility.FacilityName,
		Labels:         labels,
		Sections:       buildClinicianEntrySections(c.Request.Context(), db, report.DepartmentID, labels),
		StartDate:      formatNullDate(report.WeekStart),
		StopDate:       formatNullDate(report.WeekStop),
		ReportID:       report.ReportID,
		Values:         clinicianEntryValuesFromReport(report),
		IsEdit:         true,
		StatusLabel:    reportSubmissionStatusLabel(report.SubmitStatus, report.ReportStatus),
		ReturnURL:      sanitizeReportSubmissionURL(c.Query("return_to")),
		ReadOnly:       true,
		WeekDays:       buildWeekDayChecks(report.WeekStart.Time, report.WeekStop.Time, report.DaysWorked.String),
	}
	applyDynamicReportValues(c.Request.Context(), db, report.ReportID, resolveClinicianEntryKeys(c.Request.Context(), db, report.DepartmentID), entryForm.Values)
	sessionData.Form = entryForm

	utilities.GenerateHTML(c, sessionData, "base", "clinician-entry")
}

func HandlerReportSubmissionApprove(c *gin.Context, db *sql.DB, sessionManager *scs.SessionManager) {
	sessionData, ok := Get_Session_Data(c, db, sessionManager, nil).(utilities.TemplateData)
	if !ok {
		log.Println("Failed to retrieve session data as TemplateData")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Unable to retrieve session data"})
		return
	}

	sesDetails, ok := sessionData.Ses.(utilities.SessionDetails)
	if !ok {
		log.Println("Failed to retrieve session details from TemplateData")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Unable to retrieve session details"})
		return
	}

	reportID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.String(http.StatusBadRequest, "Invalid report ID")
		return
	}

	updated, err := models.ApproveFacilityReport(c.Request.Context(), db, reportID, sesDetails.HFID, sesDetails.EmpID)
	if err != nil {
		log.Printf("Error approving report submission: %v", err)
		c.String(http.StatusInternalServerError, "Unable to approve report submission")
		return
	}
	if !updated {
		c.String(http.StatusForbidden, "Only pending submitted reports within your facility can be approved")
		return
	}

	c.Redirect(http.StatusFound, sanitizeReportSubmissionURL(c.PostForm("return_to")))
}

func HandlerReportSubmissionSubmitAll(c *gin.Context, db *sql.DB, sessionManager *scs.SessionManager) {
	sessionData, ok := Get_Session_Data(c, db, sessionManager, nil).(utilities.TemplateData)
	if !ok {
		log.Println("Failed to retrieve session data as TemplateData")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Unable to retrieve session data"})
		return
	}

	sesDetails, ok := sessionData.Ses.(utilities.SessionDetails)
	if !ok {
		log.Println("Failed to retrieve session details from TemplateData")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Unable to retrieve session details"})
		return
	}
	if roleFromRights(sesDetails.Rights) != utilities.RoleFacilityAdmin {
		c.String(http.StatusForbidden, "Only facility admins can submit all reports for a week")
		return
	}

	department, _ := strconv.Atoi(c.PostForm("department"))
	year, _ := strconv.Atoi(c.PostForm("year"))
	week, _ := strconv.Atoi(c.PostForm("week"))
	if year <= 0 || week <= 0 {
		c.String(http.StatusBadRequest, "Select a specific year and week before submitting all reports")
		return
	}

	weekStart := isoWeekStart(year, week)
	weekStop := weekStart.AddDate(0, 0, 6)

	missingOnDuty, err := models.CountOnDutyEmployeesMissingWeeklyReport(c.Request.Context(), db, sesDetails.HFID, weekStart, weekStop, department)
	if err != nil {
		log.Printf("Error validating missing on-duty reports: %v", err)
		c.String(http.StatusInternalServerError, "Unable to validate on-duty report coverage")
		return
	}
	if missingOnDuty > 0 {
		c.String(http.StatusForbidden, "Some on-duty staff still have no report for this week. Complete all on-duty entries first.")
		return
	}

	if _, err := models.EnsureOnLeaveZeroReports(c.Request.Context(), db, sesDetails.HFID, weekStart, weekStop, sesDetails.EmpID, department); err != nil {
		log.Printf("Error creating on-leave zero reports: %v", err)
		c.String(http.StatusInternalServerError, "Unable to prepare on-leave records")
		return
	}

	if _, err := models.SubmitFacilityReportsByFilter(c.Request.Context(), db, sesDetails.HFID, 0, year, 0, week, sesDetails.EmpID); err != nil {
		if errors.Is(err, models.ErrFacilityBatchRequiresApprovedReports) {
			c.String(http.StatusForbidden, "Approve all reports for this facility week before submitting the weekly batch to national review")
			return
		}
		log.Printf("Error submitting facility reports: %v", err)
		c.String(http.StatusInternalServerError, "Unable to submit facility reports")
		return
	}

	c.Redirect(http.StatusFound, sanitizeReportSubmissionURL(c.PostForm("return_to")))
}

func HandlerAdminFacilitySubmissionApprove(c *gin.Context, db *sql.DB, sessionManager *scs.SessionManager) {
	handleAdminFacilitySubmissionDecision(c, db, sessionManager, true)
}

func HandlerAdminFacilitySubmissionDecline(c *gin.Context, db *sql.DB, sessionManager *scs.SessionManager) {
	handleAdminFacilitySubmissionDecision(c, db, sessionManager, false)
}

func handleAdminFacilitySubmissionDecision(c *gin.Context, db *sql.DB, sessionManager *scs.SessionManager, approve bool) {
	sessionData, ok := Get_Session_Data(c, db, sessionManager, nil).(utilities.TemplateData)
	if !ok {
		log.Println("Failed to retrieve session data as TemplateData")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Unable to retrieve session data"})
		return
	}

	sesDetails, ok := sessionData.Ses.(utilities.SessionDetails)
	if !ok {
		log.Println("Failed to retrieve session details from TemplateData")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Unable to retrieve session details"})
		return
	}
	if roleFromRights(sesDetails.Rights) != utilities.RoleNationalAdmin {
		c.String(http.StatusForbidden, "Only national admins can approve or decline by facility")
		return
	}

	facilityID, err := strconv.Atoi(c.Param("facility"))
	if err != nil || facilityID <= 0 {
		c.String(http.StatusBadRequest, "Invalid facility")
		return
	}

	department, _ := strconv.Atoi(c.PostForm("department"))
	year, _ := strconv.Atoi(c.PostForm("year"))
	month, _ := strconv.Atoi(c.PostForm("month"))
	week, _ := strconv.Atoi(c.PostForm("week"))

	if approve {
		_, err = models.ApproveNationalFacilityReportsByFilter(c.Request.Context(), db, int64(facilityID), department, year, month, week, sesDetails.EmpID)
	} else {
		_, err = models.DeclineNationalFacilityReportsByFilter(c.Request.Context(), db, int64(facilityID), department, year, month, week, sesDetails.EmpID)
	}
	if err != nil {
		log.Printf("Error processing facility-level review: %v", err)
		c.String(http.StatusInternalServerError, "Unable to process facility-level review")
		return
	}

	c.Redirect(http.StatusFound, sanitizeReportSubmissionURL(c.PostForm("return_to")))
}

func HandlerReportSubmissionApproveAll(c *gin.Context, db *sql.DB, sessionManager *scs.SessionManager) {
	sessionData, ok := Get_Session_Data(c, db, sessionManager, nil).(utilities.TemplateData)
	if !ok {
		log.Println("Failed to retrieve session data as TemplateData")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Unable to retrieve session data"})
		return
	}

	sesDetails, ok := sessionData.Ses.(utilities.SessionDetails)
	if !ok {
		log.Println("Failed to retrieve session details from TemplateData")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Unable to retrieve session details"})
		return
	}

	department, _ := strconv.Atoi(c.PostForm("department"))
	year, _ := strconv.Atoi(c.PostForm("year"))
	month, _ := strconv.Atoi(c.PostForm("month"))
	week, _ := strconv.Atoi(c.PostForm("week"))

	_, err := models.ApproveFacilityReportsByFilter(c.Request.Context(), db, sesDetails.HFID, department, year, month, week, sesDetails.EmpID)
	if err != nil {
		log.Printf("Error approving filtered report submissions: %v", err)
		c.String(http.StatusInternalServerError, "Unable to approve the selected report submissions")
		return
	}

	c.Redirect(http.StatusFound, sanitizeReportSubmissionURL(c.PostForm("return_to")))
}

func HandlerReportSubmissionDecline(c *gin.Context, db *sql.DB, sessionManager *scs.SessionManager) {
	sessionData, ok := Get_Session_Data(c, db, sessionManager, nil).(utilities.TemplateData)
	if !ok {
		log.Println("Failed to retrieve session data as TemplateData")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Unable to retrieve session data"})
		return
	}

	sesDetails, ok := sessionData.Ses.(utilities.SessionDetails)
	if !ok {
		log.Println("Failed to retrieve session details from TemplateData")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Unable to retrieve session details"})
		return
	}

	reportID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.String(http.StatusBadRequest, "Invalid report ID")
		return
	}

	updated, err := models.DeclineFacilityReport(c.Request.Context(), db, reportID, sesDetails.HFID, sesDetails.EmpID)
	if err != nil {
		log.Printf("Error declining report submission: %v", err)
		c.String(http.StatusInternalServerError, "Unable to decline report submission")
		return
	}
	if !updated {
		c.String(http.StatusForbidden, "Only pending submitted reports within your facility can be declined")
		return
	}

	c.Redirect(http.StatusFound, sanitizeReportSubmissionURL(c.PostForm("return_to")))
}

func buildReportSubmissionsURL(facilityID int, departmentID int, year int, month int, week int, status string) string {
	if year < 0 {
		year = 0
	}
	if month < 0 {
		month = 0
	}
	if week < 0 {
		week = 0
	}

	params := []string{}
	if facilityID > 0 {
		params = append(params, fmt.Sprintf("facility=%d", facilityID))
	}
	if departmentID > 0 {
		params = append(params, fmt.Sprintf("department=%d", departmentID))
	}
	// Preserve explicit period scope (including 0 = all) so KPI links and list totals stay aligned.
	params = append(params, fmt.Sprintf("year=%d", year))
	params = append(params, fmt.Sprintf("month=%d", month))
	params = append(params, fmt.Sprintf("week=%d", week))
	if status != "" && status != "all" {
		params = append(params, "status="+status)
	}
	if len(params) == 0 {
		return "/reports/analysis"
	}
	return "/reports/analysis?" + strings.Join(params, "&")
}

func buildReportSubmissionsURLWithView(facilityID int, departmentID int, year int, month int, week int, status string, view string) string {
	base := buildReportSubmissionsURL(facilityID, departmentID, year, month, week, status)
	if view == "" || view == "staff" {
		return base
	}
	if strings.Contains(base, "?") {
		return base + "&view=" + view
	}
	return base + "?view=" + view
}

func buildReportSubmissionsURLWithViewAndPage(facilityID int, departmentID int, year int, month int, week int, status string, view string, page int) string {
	base := buildReportSubmissionsURLWithView(facilityID, departmentID, year, month, week, status, view)
	if page <= 1 {
		return base
	}
	if strings.Contains(base, "?") {
		return base + fmt.Sprintf("&page=%d", page)
	}
	return base + fmt.Sprintf("?page=%d", page)
}

func buildReportSubmissionsURLWithModeAndPage(facilityID int, departmentID int, year int, month int, week int, status string, view string, mode string, page int) string {
	base := buildReportSubmissionsURLWithViewAndPage(facilityID, departmentID, year, month, week, status, view, page)
	trimmed := strings.TrimSpace(strings.ToLower(mode))
	if trimmed == "" || trimmed == "weeks" {
		trimmed = ""
	}
	if trimmed == "" {
		return base
	}
	if strings.Contains(base, "?") {
		return base + "&mode=" + trimmed
	}
	return base + "?mode=" + trimmed
}

func buildFacilityWeekDrilldownURL(facilityID int, departmentID int, status string, row *models.FacilitySubmissionSummaryRow) string {
	if row == nil || !row.WeekStart.Valid {
		return buildReportSubmissionsURLWithModeAndPage(facilityID, departmentID, 0, 0, 0, status, "staff", "reports", 1)
	}
	year, week := row.WeekStart.Time.ISOWeek()
	month := int(row.WeekStart.Time.Month())
	return buildReportSubmissionsURLWithModeAndPage(facilityID, departmentID, year, month, week, status, "staff", "reports", 1)
}

func normalizeReportSubmissionMode(role string, value string) string {
	if role != utilities.RoleFacilityAdmin {
		return "reports"
	}
	trimmed := strings.ToLower(strings.TrimSpace(value))
	if trimmed == "reports" {
		return "reports"
	}
	return "weeks"
}

func buildReportSubmissionsExportURL(format string, facilityID int, departmentID int, year int, month int, week int, status string) string {
	if year < 0 {
		year = 0
	}
	if month < 0 {
		month = 0
	}
	if week < 0 {
		week = 0
	}

	params := []string{}
	if facilityID > 0 {
		params = append(params, fmt.Sprintf("facility=%d", facilityID))
	}
	if departmentID > 0 {
		params = append(params, fmt.Sprintf("department=%d", departmentID))
	}
	params = append(params, fmt.Sprintf("year=%d", year))
	params = append(params, fmt.Sprintf("month=%d", month))
	params = append(params, fmt.Sprintf("week=%d", week))
	if status != "" && status != "all" {
		params = append(params, "status="+status)
	}
	if format != "" {
		params = append(params, "format="+format)
	}
	if len(params) == 0 {
		return "/reports/analysis/export"
	}
	return "/reports/analysis/export?" + strings.Join(params, "&")
}

func buildReportSubmissionsExportURLWithView(format string, facilityID int, departmentID int, year int, month int, week int, status string, view string) string {
	base := buildReportSubmissionsExportURL(format, facilityID, departmentID, year, month, week, status)
	if view == "" || view == "staff" {
		return base
	}
	if strings.Contains(base, "?") {
		return base + "&view=" + view
	}
	return base + "?view=" + view
}

func buildReportSubmissionsExportFilename(facilityID int, departmentID int, year int, month int, week int, status string) string {
	name := "report-submissions"
	if facilityID > 0 {
		name += fmt.Sprintf("-facility-%d", facilityID)
	}
	if departmentID > 0 {
		name += fmt.Sprintf("-department-%d", departmentID)
	}
	if year > 0 {
		name += fmt.Sprintf("-%d", year)
	}
	if month > 0 {
		name += fmt.Sprintf("-month-%02d", month)
	}
	if week > 0 {
		name += fmt.Sprintf("-week-%02d", week)
	}
	if status != "" && status != "all" {
		name += "-" + status
	}
	return name
}

func sanitizeReportSubmissionURL(value string) string {
	if value == "" {
		return "/reports/analysis"
	}
	if strings.HasPrefix(value, "/reports/analysis") || strings.HasPrefix(value, "/reports/review") {
		return value
	}
	return "/reports/analysis"
}

func reportSubmissionStatusLabel(submitStatus sql.NullString, reportStatus sql.NullString) string {
	if reportStatus.Valid {
		switch reportStatus.String {
		case "Approved":
			return "Approved"
		case "Rejected", "Declined":
			return "Declined"
		}
	}
	if submitStatus.Valid && submitStatus.String == "Submitted" {
		return "Submitted"
	}
	return "Draft"
}

func parseOptionalIntQuery(c *gin.Context, key string) (int, bool) {
	value, ok := c.GetQuery(key)
	if !ok {
		return 0, false
	}
	parsed, _ := strconv.Atoi(value)
	return parsed, true
}

func resolveReportSubmissionPeriod(c *gin.Context, db *sql.DB, requestedYear int, hasYear bool, requestedMonth int, hasMonth bool, requestedWeek int, hasWeek bool) (int, int, int, []int, []models.DashboardFilterOption, []models.ClinicianWeekOption, string, error) {
	periods, err := models.GetDashboardReportPeriods(c.Request.Context(), db)
	if err != nil {
		return 0, 0, 0, nil, nil, nil, "", err
	}
	if len(periods) == 0 {
		periods = []time.Time{startOfWeekLocal(time.Now())}
	}

	latestStart := periods[0]
	latestYear, latestWeek := latestStart.ISOWeek()
	latestMonth := int(latestStart.Month())

	selectedYear := latestYear
	if hasYear {
		selectedYear = requestedYear
	}
	selectedMonth := latestMonth
	if hasMonth {
		selectedMonth = requestedMonth
	}
	selectedWeek := latestWeek
	if hasWeek {
		selectedWeek = requestedWeek
	}

	availableYears := []int{0}
	yearSeen := map[int]bool{}
	for _, period := range periods {
		year, _ := period.ISOWeek()
		if yearSeen[year] {
			continue
		}
		yearSeen[year] = true
		availableYears = append(availableYears, year)
	}
	sort.SliceStable(availableYears[1:], func(i, j int) bool {
		return availableYears[i+1] > availableYears[j+1]
	})

	availableMonths := []models.DashboardFilterOption{{ID: 0, Name: "All"}}
	monthSeen := map[int]bool{}
	for _, period := range periods {
		year, _ := period.ISOWeek()
		if selectedYear > 0 && year != selectedYear {
			continue
		}
		month := int(period.Month())
		if monthSeen[month] {
			continue
		}
		monthSeen[month] = true
		availableMonths = append(availableMonths, models.DashboardFilterOption{
			ID:   month,
			Name: time.Month(month).String(),
		})
	}
	sort.SliceStable(availableMonths[1:], func(i, j int) bool {
		return availableMonths[i+1].ID > availableMonths[j+1].ID
	})

	availableWeeks := []models.ClinicianWeekOption{{
		Year:  0,
		Week:  0,
		Label: "All",
	}}
	weekSeen := map[string]bool{}
	for _, period := range periods {
		year, week := period.ISOWeek()
		month := int(period.Month())
		if selectedYear > 0 && year != selectedYear {
			continue
		}
		if selectedMonth > 0 && month != selectedMonth {
			continue
		}
		key := fmt.Sprintf("%d-%d-%02d", year, month, week)
		if weekSeen[key] {
			continue
		}
		weekSeen[key] = true
		availableWeeks = append(availableWeeks, models.ClinicianWeekOption{
			Year:      year,
			Week:      week,
			StartDate: period.Format("2006-01-02"),
			EndDate:   period.AddDate(0, 0, 6).Format("2006-01-02"),
			Label:     fmt.Sprintf("Week %02d (%s - %s)", week, period.Format("02 Jan"), period.AddDate(0, 0, 6).Format("02 Jan")),
		})
	}
	sort.SliceStable(availableWeeks[1:], func(i, j int) bool {
		return availableWeeks[i+1].StartDate > availableWeeks[j+1].StartDate
	})

	selectedWeekLabel := "Across all records"
	switch {
	case selectedYear > 0 && selectedMonth > 0 && selectedWeek > 0:
		selectedWeekLabel = fmt.Sprintf("Week %02d in %s %d", selectedWeek, time.Month(selectedMonth).String(), selectedYear)
	case selectedYear > 0 && selectedMonth > 0:
		selectedWeekLabel = fmt.Sprintf("All weeks in %s %d", time.Month(selectedMonth).String(), selectedYear)
	case selectedYear > 0 && selectedWeek > 0:
		selectedWeekLabel = fmt.Sprintf("Week %02d across %d", selectedWeek, selectedYear)
	case selectedMonth > 0 && selectedWeek > 0:
		selectedWeekLabel = fmt.Sprintf("Week %02d across %s", selectedWeek, time.Month(selectedMonth).String())
	case selectedMonth > 0:
		selectedWeekLabel = fmt.Sprintf("All years in %s", time.Month(selectedMonth).String())
	case selectedYear > 0:
		selectedWeekLabel = fmt.Sprintf("All months in %d", selectedYear)
	}

	return selectedYear, selectedMonth, selectedWeek, availableYears, availableMonths, availableWeeks, selectedWeekLabel, nil
}

func containsFilterOption(options []models.DashboardFilterOption, target int) bool {
	if target == 0 {
		return true
	}
	for _, option := range options {
		if option.ID == target {
			return true
		}
	}
	return false
}

func normalizeReportSubmissionStatus(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "submitted":
		return "submitted"
	case "pending":
		return "pending"
	case "approved":
		return "approved"
	case "declined":
		return "declined"
	case "draft":
		return "draft"
	default:
		return "all"
	}
}

func normalizeReportSubmissionViewMode(role string, value string) string {
	trimmed := strings.ToLower(strings.TrimSpace(value))
	if role != utilities.RoleNationalAdmin {
		return "staff"
	}
	if trimmed == "staff" {
		return "staff"
	}
	return "facility"
}

func isoWeekStart(year int, week int) time.Time {
	base := time.Date(year, time.January, 4, 0, 0, 0, 0, time.Local)
	weekday := int(base.Weekday())
	if weekday == 0 {
		weekday = 7
	}
	week1Monday := base.AddDate(0, 0, -(weekday - 1))
	return week1Monday.AddDate(0, 0, (week-1)*7)
}

func reportSubmissionFilterSummary(role string, status string, periodLabel string) string {
	statusLabel := "all"
	switch status {
	case "submitted":
		statusLabel = "submitted"
	case "pending":
		statusLabel = "pending review"
	case "approved":
		statusLabel = "approved"
	case "declined":
		statusLabel = "declined"
	case "draft":
		statusLabel = "draft"
	}

	if role == utilities.RoleStaff {
		return fmt.Sprintf("Showing %s reports for %s.", statusLabel, periodLabel)
	}
	return fmt.Sprintf("Showing %s submissions for %s.", statusLabel, periodLabel)
}

func isFinalReportReviewStatus(status string) bool {
	switch status {
	case "Approved", "Rejected", "Declined":
		return true
	default:
		return false
	}
}
