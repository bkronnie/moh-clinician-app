package handlers

import (
	"database/sql"
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
	PendingCount       int
	ClearFiltersURL    string
}

func buildReportSubmissionsView(c *gin.Context, db *sql.DB, sesDetails utilities.SessionDetails) (ReportSubmissionsView, error) {
	requestedFacility, hasFacility := parseOptionalIntQuery(c, "facility")
	requestedDepartment, hasDepartment := parseOptionalIntQuery(c, "department")
	requestedYear, hasYear := parseOptionalIntQuery(c, "year")
	requestedMonth, hasMonth := parseOptionalIntQuery(c, "month")
	requestedWeek, hasWeek := parseOptionalIntQuery(c, "week")
	selectedStatus := normalizeReportSubmissionStatus(c.Query("status"))

	selectedYear, selectedMonth, selectedWeek, availableYears, availableMonths, availableWeeks, selectedWeekLabel, err := resolveReportSubmissionPeriod(c, db, requestedYear, hasYear, requestedMonth, hasMonth, requestedWeek, hasWeek)
	if err != nil {
		return ReportSubmissionsView{}, err
	}

	facilityOptions, err := models.GetDashboardFacilityOptions(c.Request.Context(), db)
	if err != nil {
		return ReportSubmissionsView{}, err
	}

	view := ReportSubmissionsView{
		Role:              sesDetails.Rights,
		SelectedStatus:    selectedStatus,
		SelectedYear:      selectedYear,
		SelectedMonth:     selectedMonth,
		SelectedWeek:      selectedWeek,
		SelectedWeekLabel: selectedWeekLabel,
		AvailableYears:    availableYears,
		AvailableMonths:   availableMonths,
		AvailableWeeks:    availableWeeks,
		CanApprove:        sesDetails.Rights == "approver",
		CanView:           sesDetails.Rights != "user",
	}

	scopeFacilityID := int64(0)
	scopeEmployeeID := int64(0)
	if sesDetails.Rights == "user" {
		scopeEmployeeID = sesDetails.EmpID
		view.ScopeTitle = "My Reports"
		view.ScopeSubtitle = "Review all of your saved, submitted, approved, and declined reports in one place using the same filtered reports screen used across the application."
		view.SelectedFacility = int(sesDetails.HFID)
		view.FacilityOptions = []models.DashboardFilterOption{{
			ID:   int(sesDetails.HFID),
			Name: sesDetails.HFName,
		}}
	} else if sesDetails.Rights == "approver" {
		scopeFacilityID = sesDetails.HFID
		view.SelectedFacility = int(sesDetails.HFID)
		view.FacilityOptions = []models.DashboardFilterOption{{
			ID:   int(sesDetails.HFID),
			Name: sesDetails.HFName,
		}}
		view.ScopeTitle = "Facility Report Submissions"
		view.ScopeSubtitle = "Review all report submissions for your facility, inspect the saved data in read-only mode, and approve pending submissions once they are verified."
	} else {
		if hasFacility {
			view.SelectedFacility = requestedFacility
		}
		view.FacilityOptions = facilityOptions
		view.ScopeTitle = "National Report Submissions"
		view.ScopeSubtitle = "Review report submissions across all hospital facilities using facility, year, month, and week filters. The default view opens on the latest reporting week in the database."
	}

	departmentOptions, err := models.GetReportSubmissionDepartmentOptions(c.Request.Context(), db, scopeFacilityID, view.SelectedFacility)
	if err != nil {
		return ReportSubmissionsView{}, err
	}
	view.DepartmentOptions = departmentOptions
	if sesDetails.Rights != "user" && hasDepartment && containsFilterOption(departmentOptions, requestedDepartment) {
		view.SelectedDepartment = requestedDepartment
	}

	rows, err := models.GetReportSubmissions(c.Request.Context(), db, scopeFacilityID, scopeEmployeeID, view.SelectedFacility, view.SelectedDepartment, selectedStatus, selectedYear, selectedMonth, selectedWeek)
	if err != nil {
		return ReportSubmissionsView{}, err
	}
	view.Rows = rows

	for _, row := range rows {
		if row.SubmitStatus.Valid && row.SubmitStatus.String == "Submitted" && (!row.ReportStatus.Valid || !isFinalReportReviewStatus(row.ReportStatus.String)) {
			view.PendingCount++
		}
	}

	view.CurrentURL = buildReportSubmissionsURL(view.SelectedFacility, view.SelectedDepartment, view.SelectedYear, view.SelectedMonth, view.SelectedWeek, view.SelectedStatus)
	view.AllURL = buildReportSubmissionsURL(view.SelectedFacility, view.SelectedDepartment, view.SelectedYear, view.SelectedMonth, view.SelectedWeek, "all")
	view.SubmittedURL = buildReportSubmissionsURL(view.SelectedFacility, view.SelectedDepartment, view.SelectedYear, view.SelectedMonth, view.SelectedWeek, "submitted")
	view.PendingURL = buildReportSubmissionsURL(view.SelectedFacility, view.SelectedDepartment, view.SelectedYear, view.SelectedMonth, view.SelectedWeek, "pending")
	view.ApprovedURL = buildReportSubmissionsURL(view.SelectedFacility, view.SelectedDepartment, view.SelectedYear, view.SelectedMonth, view.SelectedWeek, "approved")
	view.DeclinedURL = buildReportSubmissionsURL(view.SelectedFacility, view.SelectedDepartment, view.SelectedYear, view.SelectedMonth, view.SelectedWeek, "declined")
	view.DraftURL = buildReportSubmissionsURL(view.SelectedFacility, view.SelectedDepartment, view.SelectedYear, view.SelectedMonth, view.SelectedWeek, "draft")
	view.ExportCSVURL = buildReportSubmissionsExportURL("csv", view.SelectedFacility, view.SelectedDepartment, view.SelectedYear, view.SelectedMonth, view.SelectedWeek, view.SelectedStatus)
	view.ExportPDFURL = buildReportSubmissionsExportURL("pdf", view.SelectedFacility, view.SelectedDepartment, view.SelectedYear, view.SelectedMonth, view.SelectedWeek, view.SelectedStatus)
	view.ClearFiltersURL = buildReportSubmissionsURL(0, 0, 0, 0, 0, "all")
	view.FilterSummary = reportSubmissionFilterSummary(sesDetails.Rights, selectedStatus, selectedWeekLabel)

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

	view, err := buildReportSubmissionsView(c, db, sesDetails)
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

	view, err := buildReportSubmissionsView(c, db, sesDetails)
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
	if sesDetails.Rights == "approver" {
		scopeFacilityID = sesDetails.HFID
	}

	report, err := models.GetReportSubmissionByIDForReview(c.Request.Context(), db, reportID, scopeFacilityID)
	if err != nil {
		if err == sql.ErrNoRows {
			c.String(http.StatusNotFound, "Report submission not found")
			return
		}
		log.Printf("Error retrieving report submission: %v", err)
		c.String(http.StatusInternalServerError, "Unable to retrieve report submission")
		return
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

	sessionData.Form = ClinicianEntryView{
		EmployeeID:     report.EmployeeID,
		EmployeeName:   fmt.Sprintf("%s %s", employee.Fname.String, employee.Lname.String),
		DepartmentID:   report.DepartmentID,
		DepartmentName: department.DepartmentName.String,
		FacilityName:   facility.FacilityName,
		StartDate:      formatNullDate(report.WeekStart),
		StopDate:       formatNullDate(report.WeekStop),
		ReportID:       report.ReportID,
		Values:         clinicianEntryValuesFromReport(report),
		IsEdit:         true,
		StatusLabel:    reportSubmissionStatusLabel(report.SubmitStatus, report.ReportStatus),
		ReturnURL:      sanitizeReportSubmissionURL(c.Query("return_to")),
		ReadOnly:       true,
	}

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
	params := []string{}
	if facilityID > 0 {
		params = append(params, fmt.Sprintf("facility=%d", facilityID))
	}
	if departmentID > 0 {
		params = append(params, fmt.Sprintf("department=%d", departmentID))
	}
	if year > 0 {
		params = append(params, fmt.Sprintf("year=%d", year))
	}
	if month > 0 {
		params = append(params, fmt.Sprintf("month=%d", month))
	}
	if week > 0 {
		params = append(params, fmt.Sprintf("week=%d", week))
	}
	if status != "" && status != "all" {
		params = append(params, "status="+status)
	}
	if len(params) == 0 {
		return "/reports/analysis"
	}
	return "/reports/analysis?" + strings.Join(params, "&")
}

func buildReportSubmissionsExportURL(format string, facilityID int, departmentID int, year int, month int, week int, status string) string {
	params := []string{}
	if facilityID > 0 {
		params = append(params, fmt.Sprintf("facility=%d", facilityID))
	}
	if departmentID > 0 {
		params = append(params, fmt.Sprintf("department=%d", departmentID))
	}
	if year > 0 {
		params = append(params, fmt.Sprintf("year=%d", year))
	}
	if month > 0 {
		params = append(params, fmt.Sprintf("month=%d", month))
	}
	if week > 0 {
		params = append(params, fmt.Sprintf("week=%d", week))
	}
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
		return "Pending Review"
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

	if role == "user" {
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
