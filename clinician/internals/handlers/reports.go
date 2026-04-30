package handlers

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	"clinician/internals/models"
	"clinician/internals/utilities"

	"github.com/alexedwards/scs/v2"
	"github.com/gin-gonic/gin"
)

// WeekDayCheck represents one day in a reporting week for the attendance day-picker.
type WeekDayCheck struct {
	Name    string // "Monday", "Tuesday", etc.
	Short   string // "Mon", "Tue", etc.
	Date    string // "2006-01-02"
	Checked bool
}

type ClinicianEntryWeekOption struct {
	StartDate string
	Label     string
	URL       string
	Selected  bool
}

type ClinicianEntryField struct {
	Key      string
	Label    string
	ReadOnly bool
}

type ClinicianEntrySection struct {
	Title  string
	Fields []ClinicianEntryField
}

type ClinicianEntryView struct {
	EmployeeID     int64
	EmployeeName   string
	DepartmentID   int64
	DepartmentName string
	FacilityName   string
	Labels         map[string]string
	Sections       []ClinicianEntrySection
	PageTitle      string
	StartDate      string
	StopDate       string
	ReportID       int
	ActionURL      string
	SubmitLabel    string
	Values         map[string]string
	IsEdit         bool
	ReadOnly       bool
	StatusLabel    string
	ReturnURL      string
	WeekDays       []WeekDayCheck
	WeekOptions    []ClinicianEntryWeekOption
	SelectedWeek   string
	OnLeave        bool
	AttendanceOnly bool
}

type ClinicianReportHistoryView struct {
	Rows         []*models.ClinicianReportHistoryRow
	FilterStatus string
	FilterTitle  string
	Year         int
	Week         int
	PeriodLabel  string
	AllURL       string
	SubmittedURL string
	ApprovedURL  string
	DeclinedURL  string
	DraftURL     string
	CurrentURL   string
	ExportCSVURL string
	ExportPDFURL string
}

type FacilityReportReviewView struct {
	Rows         []*models.FacilityReportReviewRow
	FilterStatus string
	FilterTitle  string
	Year         int
	Week         int
	PeriodLabel  string
	AllURL       string
	PendingURL   string
	ApprovedURL  string
	DeclinedURL  string
	CurrentURL   string
	ExportCSVURL string
	ExportPDFURL string
}

type BulkEntryDepartmentTab struct {
	DepartmentID int
	Label        string
	Count        int
	URL          string
}

type BulkEntryRow struct {
	EmployeeID     int
	EmployeeName   string
	EmployeeTitle  string
	DepartmentName string
	Specialisation string
	EntryURL       string
}

func canAccessReportSubmissions(role string) bool {
	normalized := roleFromRights(role)
	return normalized == utilities.RoleNationalAdmin || normalized == utilities.RoleFacilityAdmin
}

func resolveReportSubmissionsRoute(path string, isNational bool) (flag int, isDepartment bool, ok bool) {
	switch path {
	case "/reports/submissions":
		if isNational {
			return 1, false, true
		}
		return 2, true, true
	case "/reports/submissions/departments":
		return 2, true, true
	default:
		return 0, false, false
	}
}

func resolveSubmissionsScopeFacilityID(isNational bool, currentHospitalID int64, facilityIDParam string) (int64, error) {
	if !isNational {
		if currentHospitalID <= 0 {
			return 0, errors.New("facility assignment is required for facility submissions")
		}
		return currentHospitalID, nil
	}

	if strings.TrimSpace(facilityIDParam) == "" {
		return 0, errors.New("facility is required for national department submissions")
	}

	facilityID, err := strconv.Atoi(facilityIDParam)
	if err != nil || facilityID <= 0 {
		return 0, errors.New("Invalid facilityID")
	}

	return int64(facilityID), nil
}

type BulkEntryView struct {
	ScopeTitle         string
	ScopeSubtitle      string
	FilterSummary      string
	SelectedDepartment int
	Tabs               []BulkEntryDepartmentTab
	Rows               []BulkEntryRow
	CurrentURL         string
}

// Handler used for entering weekly report for a single user
func SingleEntryForm(c *gin.Context, db *sql.DB, sessionManager *scs.SessionManager) {
	sessionData, ok := Get_Session_Data(c, db, sessionManager, nil).(utilities.TemplateData)
	if !ok {
		log.Println("Failed to retrieve session data as TemplateData")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Unable to retrieve session data"})
		return
	}

	// Now access the SessionDetails from the TemplateData
	sesDetails, ok := sessionData.Ses.(utilities.SessionDetails)
	if !ok {
		log.Println("Failed to retrieve session details from TemplateData")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Unable to retrieve session details"})
		return
	}

	empID := sesDetails.EmpID
	entryTitle := "My Weekly Data Entry"

	requestedEmployeeID := 0
	approverSelfEntry := utilities.RoleMatches(sesDetails.Rights, "Facility Admin")
	if rawEmployeeID := strings.TrimSpace(c.Query("employee")); rawEmployeeID != "" {
		parsedEmployeeID, err := strconv.Atoi(rawEmployeeID)
		if err != nil || parsedEmployeeID <= 0 {
			c.String(http.StatusBadRequest, "Invalid employee selection")
			return
		}
		if !utilities.RoleMatches(sesDetails.Rights, "Facility Admin") {
			c.String(http.StatusForbidden, "Only facility admins can open staff bulk entry forms.")
			return
		}
		requestedEmployeeID = parsedEmployeeID
		empID = int64(parsedEmployeeID)
		entryTitle = "Bulk Staff Data Entry"
		approverSelfEntry = false
	}

	log.Printf("UserID: %+d", empID)

	employee, err := models.EmployeeByID(c, db, int(empID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Unable to load employee details"})
		return
	}
	if requestedEmployeeID > 0 && employee.EmpFacility != sesDetails.HFID {
		c.String(http.StatusForbidden, "This employee is outside your facility.")
		return
	}

	employeeName := formatEmployeeName(employee.Fname.String, employee.Lname.String, employee.Oname.String)
	labels := resolveClinicianEntryLabels(c.Request.Context(), db)

	reportID, _ := strconv.Atoi(c.Param("i"))
	returnURL := sanitizeHistoryReturnURL(c.Query("return_to"))

	if reportID > 0 {
		report, err := models.ClinicianEditableReportByID(c.Request.Context(), db, reportID, sesDetails.EmpID)
		if err != nil {
			if err == sql.ErrNoRows {
				c.String(http.StatusNotFound, "This report cannot be edited.")
				return
			}
			log.Printf("Error loading clinician report for editing: %v", err)
			c.String(http.StatusInternalServerError, "Error loading report")
			return
		}

		reportDepartment, err := models.DepartmentByID(c, db, int(report.DepartmentID))
		if err != nil {
			c.String(http.StatusInternalServerError, "Unable to load department details")
			return
		}

		facilityName := sesDetails.HFName
		if facilityName == "" && report.FacilityID > 0 {
			facility, err := models.FacilityByID(c, db, int(report.FacilityID))
			if err == nil {
				facilityName = facility.FacilityName
			}
		}

		weekDays := []WeekDayCheck{}
		onLeave := false
		if report.WeekStart.Valid && report.WeekStop.Valid {
			weekDays = buildWeekDayChecks(report.WeekStart.Time, report.WeekStop.Time, report.DaysWorked.String)
		}
		sectionKeys := resolveClinicianEntryKeys(c.Request.Context(), db, report.DepartmentID)
		entryForm := ClinicianEntryView{
			EmployeeID:     empID,
			EmployeeName:   employeeName,
			DepartmentID:   report.DepartmentID,
			DepartmentName: reportDepartment.DepartmentName.String,
			FacilityName:   facilityName,
			Labels:         labels,
			Sections:       buildClinicianEntrySections(c.Request.Context(), db, report.DepartmentID, labels),
			PageTitle:      entryTitle,
			StartDate:      formatNullDate(report.WeekStart),
			StopDate:       formatNullDate(report.WeekStop),
			ReportID:       report.ReportID,
			ActionURL:      fmt.Sprintf("/reports/update-self/%d", report.ReportID),
			SubmitLabel:    "Update Weekly Entry",
			Values:         clinicianEntryValuesFromReport(report),
			IsEdit:         true,
			StatusLabel:    clinicianEntryStatusLabel(report.HistoryStatus),
			ReturnURL:      returnURL,
			WeekDays:       weekDays,
			OnLeave:        onLeave,
			AttendanceOnly: false,
		}

		if report.WeekStart.Valid && report.WeekStop.Valid {
			if !isCompletedReportingWeekRange(report.WeekStart.Time, report.WeekStop.Time, time.Now()) {
				c.String(http.StatusForbidden, "You can only enter or update reports for completed reporting weeks.")
				return
			}
			onLeave, leaveErr := models.IsEmployeeOnLeaveForPeriod(c.Request.Context(), db, empID, report.WeekStart.Time, report.WeekStop.Time)
			if leaveErr != nil {
				c.String(http.StatusInternalServerError, "Unable to verify leave status")
				return
			}
			entryForm.OnLeave = onLeave
			if onLeave && !utilities.RoleMatches(sesDetails.Rights, "Facility Admin") {
				c.String(http.StatusForbidden, "Data entry is blocked because this staff member is on leave for the selected week.")
				return
			}
		}

		applyDynamicReportValues(c.Request.Context(), db, report.ReportID, sectionKeys, entryForm.Values)

		sessionData.Form = entryForm

		utilities.GenerateHTML(c, sessionData, "base", "clinician-entry")
		return
	}

	pendingWeeks, err := models.GetClinicianPendingEntryWeeks(c.Request.Context(), db, int(empID))
	if err != nil {
		log.Printf("Error loading pending entry weeks for employee %d: %v", empID, err)
		c.String(http.StatusInternalServerError, "Unable to load reporting weeks")
		return
	}

	selectedWeekStart, selectedWeekEnd, selectedWeekLabel := resolveClinicianEntryWeekSelection(c.Query("start"), pendingWeeks, time.Now())
	weekOptions := buildClinicianEntryWeekOptions(reportID, requestedEmployeeID, returnURL, pendingWeeks, selectedWeekStart)

	departmentID := employee.EmpDepartment
	department, err := models.DepartmentByID(c, db, int(departmentID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Unable to load department details"})
		return
	}

	facilityName := sesDetails.HFName
	if facilityName == "" && employee.EmpFacility > 0 {
		facility, err := models.FacilityByID(c, db, int(employee.EmpFacility))
		if err == nil {
			facilityName = facility.FacilityName
		}
	}

	onLeave, err := models.IsEmployeeOnLeaveForPeriod(c.Request.Context(), db, empID, selectedWeekStart, selectedWeekEnd)
	if err != nil {
		c.String(http.StatusInternalServerError, "Unable to verify leave status")
		return
	}
	if onLeave && !utilities.RoleMatches(sesDetails.Rights, "Facility Admin") {
		c.String(http.StatusForbidden, "Data entry is blocked because this staff member is on leave for the selected week.")
		return
	}
	weekDays := buildWeekDayChecks(selectedWeekStart, selectedWeekEnd, "")
	existingReport, existingErr := models.LatestClinicianReportByPeriod(c.Request.Context(), db, empID, selectedWeekStart, selectedWeekEnd)
	if existingErr != nil && existingErr != sql.ErrNoRows {
		log.Printf("Error loading existing weekly report for employee %d: %v", empID, existingErr)
		c.String(http.StatusInternalServerError, "Error loading report")
		return
	}
	if existingReport != nil {
		reportDepartmentName := department.DepartmentName.String
		if existingReport.DepartmentID > 0 && existingReport.DepartmentID != departmentID {
			reportDepartment, deptErr := models.DepartmentByID(c, db, int(existingReport.DepartmentID))
			if deptErr == nil {
				reportDepartmentName = reportDepartment.DepartmentName.String
			}
		}

		values := clinicianEntryValuesFromReport(existingReport)
		if approverSelfEntry {
			for key := range values {
				if key != "attendance" {
					values[key] = "0"
				}
			}
		}

		sections := buildClinicianEntrySections(c.Request.Context(), db, existingReport.DepartmentID, labels)
		if approverSelfEntry {
			sections = nil
		}

		entryForm := ClinicianEntryView{
			EmployeeID:     empID,
			EmployeeName:   employeeName,
			DepartmentID:   existingReport.DepartmentID,
			DepartmentName: reportDepartmentName,
			FacilityName:   facilityName,
			Labels:         labels,
			Sections:       sections,
			PageTitle:      entryTitle,
			StartDate:      formatNullDate(existingReport.WeekStart),
			StopDate:       formatNullDate(existingReport.WeekStop),
			ReportID:       existingReport.ReportID,
			ActionURL:      "/reports/zave",
			SubmitLabel:    "Update Weekly Entry",
			Values:         values,
			IsEdit:         true,
			ReadOnly:       !existingReport.Actionable,
			StatusLabel:    clinicianEntryStatusLabel(existingReport.HistoryStatus),
			ReturnURL:      returnURL,
			WeekDays:       buildWeekDayChecks(selectedWeekStart, selectedWeekEnd, existingReport.DaysWorked.String),
			WeekOptions:    weekOptions,
			SelectedWeek:   selectedWeekLabel,
			OnLeave:        onLeave,
			AttendanceOnly: approverSelfEntry,
		}

		if existingReport.Actionable {
			entryForm.SubmitLabel = "Update Existing Entry"
		} else {
			entryForm.SubmitLabel = "Already Submitted"
		}

		sectionKeys := resolveClinicianEntryKeys(c.Request.Context(), db, existingReport.DepartmentID)
		applyDynamicReportValues(c.Request.Context(), db, existingReport.ReportID, sectionKeys, entryForm.Values)

		sessionData.Form = entryForm
		utilities.GenerateHTML(c, sessionData, "base", "clinician-entry")
		return
	}

	defaultValues := defaultClinicianEntryValues()
	sections := buildClinicianEntrySections(c.Request.Context(), db, departmentID, labels)
	if approverSelfEntry {
		for key := range defaultValues {
			if key != "attendance" {
				defaultValues[key] = "0"
			}
		}
		sections = nil
	}

	sessionData.Form = ClinicianEntryView{
		EmployeeID:     empID,
		EmployeeName:   employeeName,
		DepartmentID:   departmentID,
		DepartmentName: department.DepartmentName.String,
		FacilityName:   facilityName,
		Labels:         labels,
		Sections:       sections,
		PageTitle:      entryTitle,
		StartDate:      selectedWeekStart.Format("2006-01-02"),
		StopDate:       selectedWeekEnd.Format("2006-01-02"),
		ActionURL:      "/reports/zave",
		SubmitLabel:    "Save Weekly Entry",
		Values:         defaultValues,
		ReturnURL:      returnURL,
		WeekDays:       weekDays,
		WeekOptions:    weekOptions,
		SelectedWeek:   selectedWeekLabel,
		OnLeave:        onLeave,
		AttendanceOnly: approverSelfEntry,
	}

	utilities.GenerateHTML(c, sessionData, "base", "clinician-entry")
}

/*
func SingleEntryList(c *gin.Context, db *sql.DB, sessionManager *scs.SessionManager) {

}*/

func HandlerReportSave(c *gin.Context, db *sql.DB, sessionManager *scs.SessionManager) {
}

// Handler to save BulkCapture
func HandlerReportZave(c *gin.Context, db *sql.DB, sessionManager *scs.SessionManager) {

	// Get session-specific data (e.g., user ID and HFID)
	sessionData, ok := Get_Session_Data(c, db, sessionManager, nil).(utilities.TemplateData)
	if !ok {
		log.Println("Failed to retrieve session data as TemplateData")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Unable to retrieve session data"})
		return
	}

	// Now access the SessionDetails from the TemplateData
	sesDetails, ok := sessionData.Ses.(utilities.SessionDetails)
	if !ok {
		log.Println("Failed to retrieve session details from TemplateData")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Unable to retrieve session details"})
		return
	}

	enteredByID := sesDetails.EmpID // Retrieve HFID from the session
	//log.Printf("Hospital ID value from session: %d", enteredByID)

	// Parse and validate the reporting period from the form.
	start := c.PostForm("start")
	stop := c.PostForm("stop")
	periodStart, err := parseISODate(start)
	if err != nil {
		c.String(http.StatusBadRequest, "Invalid report start date")
		return
	}
	periodStop, err := parseISODate(stop)
	if err != nil {
		c.String(http.StatusBadRequest, "Invalid report end date")
		return
	}
	if !isCompletedReportingWeekRange(periodStart, periodStop, time.Now()) {
		c.String(http.StatusForbidden, "Only completed reporting weeks can be entered.")
		return
	}

	// Retrieve individual form fields into a slice of WeeklyReportExtended
	var reports []models.WeeklyReportExtended
	allValuesByEmployee := map[int64]map[string]sql.NullInt64{}

	// Assume you have a list of employee IDs to loop through
	for empID := range c.PostFormMap("input") {
		targetEmployeeID := empIDToInt(empID)
		employee, err := models.EmployeeByID(c.Request.Context(), db, int(targetEmployeeID))
		if err != nil {
			c.String(http.StatusBadRequest, "Invalid employee selected for bulk entry")
			return
		}

		if utilities.RoleMatches(sesDetails.Rights, "Facility Admin") {
			if employee.EmpFacility != sesDetails.HFID {
				c.String(http.StatusForbidden, "You can only enter data for staff within your facility.")
				return
			}
		} else if targetEmployeeID != sesDetails.EmpID {
			c.String(http.StatusForbidden, "You can only save reports for your own record.")
			return
		}

		daysWorkedCSV, attendanceDays := collectDaysWorked(c, targetEmployeeID, periodStart, periodStop)
		entryKeys := resolveClinicianEntryKeys(c.Request.Context(), db, employee.EmpDepartment)
		values := collectClinicianEntryValues(c, targetEmployeeID, attendanceDays, entryKeys)
		if utilities.RoleMatches(sesDetails.Rights, "Facility Admin") && targetEmployeeID == sesDetails.EmpID {
			for key := range values {
				if key != "attendance" {
					values[key] = sql.NullInt64{}
				}
			}
		}

		onLeave, leaveErr := models.IsEmployeeOnLeaveForPeriod(c.Request.Context(), db, targetEmployeeID, periodStart, periodStop)
		if leaveErr != nil {
			c.String(http.StatusInternalServerError, "Unable to verify leave status")
			return
		}
		if onLeave {
			if utilities.RoleMatches(sesDetails.Rights, "Facility Admin") {
				values = nullableClinicianEntryValues()
				daysWorkedCSV = ""
			} else {
				c.String(http.StatusForbidden, "Staff on leave cannot enter reports during leave period.")
				return
			}
		}
		allValuesByEmployee[targetEmployeeID] = values

		// Create a new WeeklyReportExtended instance for each employee
		report := models.WeeklyReportExtended{
			Start: sql.NullTime{Time: periodStart, Valid: true},
			Stop:  sql.NullTime{Time: periodStop, Valid: true},
			Emp:   sql.NullInt64{Int64: targetEmployeeID, Valid: true},
			Qn01:  values["attendance"],
			Qn02:  values["ward_rounds"],
			Qn03:  values["patients_reviewed"],
			Qn04:  values["theatre_days"],
			Qn05:  values["elective"],
			Qn06:  values["emergency"],
			Qn07:  values["postmortems"],
			Qn08:  values["OPD_clinics"],
			Qn09:  values["OPD_patients"],
			Qn10:  values["anc_patients"],
			Qn11:  values["teaching_rounds"],
			Qn12:  values["students_taught"],
			Qn13:  values["mortality_reviews"],
			Qn14:  values["maternal"],
			Qn15:  values["perinatal"],
			Qn16:  values["surgical"],
			Qn17:  values["medical"],
			Qn18:  values["paed"],
			Qn19:  values["labs_requests"],
			Qn20:  values["imaging_requests"],
			Qn21:  values["lab_investigations"],
			Qn22:  values["BS"],
			Qn23:  values["HIV"],
			Qn24:  values["malaria"],
			Qn25:  values["TB"],
			Qn26:  values["CBC"],
			Qn27:  values["chemistry"],
			Qn28:  values["hematology"],
			Qn29:  values["urinalysis"],
			Qn30:  values["gram_stain"],
			Qn31:  values["culture"],
			Qn32:  values["microbiology"],
			Qn33:  values["sensitivity_tests"],
			Qn34:  values["diagnostics"],
			Qn35:  values["xrays"],
			Qn36:  values["ct_scans"],
			Qn37:  values["obstetrics_scans"],
			Qn38:  values["abdominal_scans"],
			//Qn39:           sql.NullInt64{Int64: parseInt(c.PostForm("input[" + empID + "][ct]")), Valid: true},
			EnteredByID:    sql.NullInt64{Int64: enteredByID, Valid: true},
			EntryCreatedOn: sql.NullTime{Time: time.Now(), Valid: true}, // Set created on time
			DaysWorked:     sql.NullString{String: daysWorkedCSV, Valid: strings.TrimSpace(daysWorkedCSV) != ""},
		}
		// Append the report to the slice
		reports = append(reports, report)
	}

	// Insert each report into the database, or update the existing record for the same staff/week.
	for _, report := range reports {
		existingReport, existingErr := models.LatestClinicianReportByPeriod(c.Request.Context(), db, report.Emp.Int64, periodStart, periodStop)
		if existingErr != nil && existingErr != sql.ErrNoRows {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to verify existing report"})
			return
		}

		if existingReport != nil {
			if !existingReport.Actionable {
				c.JSON(http.StatusConflict, gin.H{"error": "A submitted or approved report already exists for this week."})
				return
			}
			report.ID = existingReport.ReportID
			report.LastUpdateOn = sql.NullTime{Time: time.Now(), Valid: true}
			if err := report.Updatez(c.Request.Context(), db); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update existing report"})
				return
			}
		} else {
			if err := report.InsertNewRecord(c.Request.Context(), db); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save report"})
				return
			}
		}

		values := allValuesByEmployee[report.Emp.Int64]
		if err := models.UpdateWeeklyReportValuesByElementKeys(c.Request.Context(), db, report.ID, report.Emp.Int64, values); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to apply report data elements"})
			return
		}
	}

	// Redirect clinicians to their own report history for the reporting period they just saved.
	reportStart := parseDate(start)
	reportYear, reportWeek := reportStart.ISOWeek()
	reportMonth := int(reportStart.Month())
	returnURL := sanitizeHistoryReturnURL(c.PostForm("return_to"))
	if returnURL != "/reports/analysis" {
		c.Redirect(http.StatusFound, returnURL)
		return
	}
	c.Redirect(http.StatusFound, buildReportHistoryPeriodFilterQuery("all", reportYear, reportMonth, reportWeek))

	// Return a success response
	//c.JSON(http.StatusOK, gin.H{"message": "Reports saved successfully"})
}

// Helper function to parse date
func parseDate(dateStr string) time.Time {
	// Implement parsing logic here (e.g., using time.Parse)
	parsedTime, _ := time.Parse("2006-01-02", dateStr)
	return parsedTime
}

// Helper function to convert empID from string to int64
func empIDToInt(empID string) int64 {
	id, _ := strconv.ParseInt(empID, 10, 64)
	return id
}

// Helper function to parse an integer from string
func parseInt(val string) int64 {
	result, _ := strconv.ParseInt(val, 10, 64)
	return result
}

func HandlerClinicianReportHistory(c *gin.Context, db *sql.DB, sessionManager *scs.SessionManager) {
	filterStatus := strings.ToLower(strings.TrimSpace(c.DefaultQuery("status", "all")))
	switch filterStatus {
	case "all", "submitted", "approved", "declined", "draft":
	default:
		filterStatus = "all"
	}

	year, _ := strconv.Atoi(c.Query("year"))
	month, _ := strconv.Atoi(c.Query("month"))
	week, _ := strconv.Atoi(c.Query("week"))
	c.Redirect(http.StatusFound, buildReportSubmissionsURL(0, 0, year, month, week, filterStatus))
}

func HandlerClinicianReportHistoryExport(c *gin.Context, db *sql.DB, sessionManager *scs.SessionManager) {
	format := strings.ToLower(strings.TrimSpace(c.DefaultQuery("format", "csv")))
	if format != "csv" && format != "pdf" {
		c.String(http.StatusBadRequest, "Invalid export format")
		return
	}

	filterStatus := strings.ToLower(strings.TrimSpace(c.DefaultQuery("status", "all")))
	switch filterStatus {
	case "all", "submitted", "approved", "declined", "draft":
	default:
		filterStatus = "all"
	}

	year, _ := strconv.Atoi(c.Query("year"))
	month, _ := strconv.Atoi(c.Query("month"))
	week, _ := strconv.Atoi(c.Query("week"))
	c.Redirect(http.StatusFound, buildReportSubmissionsExportURL(format, 0, 0, year, month, week, filterStatus))
}

func HandlerClinicianReportEditForm(c *gin.Context, db *sql.DB, sessionManager *scs.SessionManager) {
	redirectURL := fmt.Sprintf("/reports/new/%s", c.Param("id"))
	returnURL := sanitizeHistoryReturnURL(c.Query("return_to"))
	if returnURL != "/reports/history" {
		redirectURL += "?return_to=" + returnURL
	}
	c.Redirect(http.StatusFound, redirectURL)
}

func HandlerClinicianReportUpdate(c *gin.Context, db *sql.DB, sessionManager *scs.SessionManager) {
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

	start, err := time.Parse("2006-01-02", c.PostForm("start"))
	if err != nil {
		c.String(http.StatusBadRequest, "Invalid start date")
		return
	}

	stop, err := time.Parse("2006-01-02", c.PostForm("stop"))
	if err != nil {
		c.String(http.StatusBadRequest, "Invalid stop date")
		return
	}

	onLeave, leaveErr := models.IsEmployeeOnLeaveForPeriod(c.Request.Context(), db, sesDetails.EmpID, start, stop)
	if leaveErr != nil {
		c.String(http.StatusInternalServerError, "Unable to verify leave status")
		return
	}
	if onLeave {
		c.String(http.StatusForbidden, "Staff on leave cannot update weekly reports for leave dates.")
		return
	}

	daysWorkedCSV, attendanceDays := collectDaysWorked(c, sesDetails.EmpID, start, stop)
	employee, err := models.EmployeeByID(c.Request.Context(), db, int(sesDetails.EmpID))
	if err != nil {
		c.String(http.StatusInternalServerError, "Unable to load employee details")
		return
	}

	entryKeys := resolveClinicianEntryKeys(c.Request.Context(), db, employee.EmpDepartment)
	values := collectClinicianEntryValues(c, sesDetails.EmpID, attendanceDays, entryKeys)

	updated, err := models.UpdateClinicianReport(
		c.Request.Context(),
		db,
		reportID,
		sesDetails.EmpID,
		start,
		stop,
		values,
		daysWorkedCSV,
	)
	if err != nil {
		log.Printf("Error updating clinician report: %v", err)
		c.String(http.StatusInternalServerError, "Unable to update report")
		return
	}
	if !updated {
		c.String(http.StatusForbidden, "This report can no longer be edited.")
		return
	}

	if err := models.UpdateWeeklyReportValuesByElementKeys(c.Request.Context(), db, reportID, sesDetails.EmpID, values); err != nil {
		c.String(http.StatusInternalServerError, "Unable to persist dynamic data elements")
		return
	}

	c.Redirect(http.StatusFound, sanitizeHistoryReturnURL(c.PostForm("return_to")))
}

func HandlerClinicianReportDelete(c *gin.Context, db *sql.DB, sessionManager *scs.SessionManager) {
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

	deleted, err := models.DeleteClinicianReport(c.Request.Context(), db, reportID, sesDetails.EmpID)
	if err != nil {
		log.Printf("Error deleting clinician report: %v", err)
		c.String(http.StatusInternalServerError, "Unable to delete report")
		return
	}
	if !deleted {
		c.String(http.StatusForbidden, "This report can no longer be deleted.")
		return
	}

	c.Redirect(http.StatusFound, sanitizeHistoryReturnURL(c.PostForm("return_to")))
}

func HandlerClinicianReportSubmit(c *gin.Context, db *sql.DB, sessionManager *scs.SessionManager) {
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

	report, fetchErr := models.GetReportSubmissionByIDForReview(c.Request.Context(), db, reportID, 0, sesDetails.Rights)
	if fetchErr != nil {
		c.String(http.StatusNotFound, "Report not found")
		return
	}
	if report.EmployeeID != sesDetails.EmpID {
		c.String(http.StatusForbidden, "You can only submit your own report")
		return
	}
	if report.WeekStart.Valid && report.WeekStop.Valid {
		onLeave, leaveErr := models.IsEmployeeOnLeaveForPeriod(c.Request.Context(), db, sesDetails.EmpID, report.WeekStart.Time, report.WeekStop.Time)
		if leaveErr != nil {
			c.String(http.StatusInternalServerError, "Unable to verify leave status")
			return
		}
		if onLeave {
			c.String(http.StatusForbidden, "Staff on leave cannot submit reports for leave periods.")
			return
		}
	}

	submitted, err := models.SubmitClinicianReport(c.Request.Context(), db, reportID, sesDetails.EmpID)
	if err != nil {
		log.Printf("Error submitting clinician report: %v", err)
		c.String(http.StatusInternalServerError, "Unable to submit report")
		return
	}
	if !submitted {
		c.String(http.StatusForbidden, "Only draft or declined reports can be submitted")
		return
	}

	c.Redirect(http.StatusFound, sanitizeHistoryReturnURL(c.PostForm("return_to")))
}

func HandlerFacilityReportReview(c *gin.Context, db *sql.DB, sessionManager *scs.SessionManager) {
	filterStatus := strings.ToLower(strings.TrimSpace(c.DefaultQuery("status", "pending")))
	switch filterStatus {
	case "all", "pending", "approved", "declined":
	default:
		filterStatus = "pending"
	}

	year, _ := strconv.Atoi(c.Query("year"))
	month, _ := strconv.Atoi(c.Query("month"))
	week, _ := strconv.Atoi(c.Query("week"))
	c.Redirect(http.StatusFound, buildReportSubmissionsURL(0, 0, year, month, week, filterStatus))
}

func HandlerFacilityReportReviewExport(c *gin.Context, db *sql.DB, sessionManager *scs.SessionManager) {
	format := strings.ToLower(strings.TrimSpace(c.DefaultQuery("format", "csv")))
	if format != "csv" && format != "pdf" {
		c.String(http.StatusBadRequest, "Invalid export format")
		return
	}

	filterStatus := strings.ToLower(strings.TrimSpace(c.DefaultQuery("status", "pending")))
	switch filterStatus {
	case "all", "pending", "approved", "declined":
	default:
		filterStatus = "pending"
	}

	year, _ := strconv.Atoi(c.Query("year"))
	month, _ := strconv.Atoi(c.Query("month"))
	week, _ := strconv.Atoi(c.Query("week"))
	c.Redirect(http.StatusFound, buildReportSubmissionsExportURL(format, 0, 0, year, month, week, filterStatus))
}

func HandlerFacilityReportApprove(c *gin.Context, db *sql.DB, sessionManager *scs.SessionManager) {
	handleFacilityReportDecision(c, db, sessionManager, true)
}

func HandlerFacilityReportDecline(c *gin.Context, db *sql.DB, sessionManager *scs.SessionManager) {
	handleFacilityReportDecision(c, db, sessionManager, false)
}

func handleFacilityReportDecision(c *gin.Context, db *sql.DB, sessionManager *scs.SessionManager, approve bool) {
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

	reportID, err := strconv.Atoi(strings.TrimSpace(c.Param("id")))
	if err != nil {
		c.String(http.StatusBadRequest, "Invalid report ID")
		return
	}

	var updated bool
	if approve {
		updated, err = models.ApproveFacilityReport(c.Request.Context(), db, reportID, sesDetails.HFID, sesDetails.EmpID)
	} else {
		updated, err = models.DeclineFacilityReport(c.Request.Context(), db, reportID, sesDetails.HFID, sesDetails.EmpID)
	}
	if err != nil {
		log.Printf("Error updating report review decision: %v", err)
		c.String(http.StatusInternalServerError, "Unable to update report status")
		return
	}
	if !updated {
		c.String(http.StatusForbidden, "Only pending submitted reports within your facility can be updated")
		return
	}

	c.Redirect(http.StatusFound, sanitizeFacilityReportReviewURL(c.PostForm("return_to")))
}

func defaultClinicianEntryValues() map[string]string {
	return map[string]string{
		"attendance":         "0",
		"ward_rounds":        "0",
		"patients_reviewed":  "0",
		"theatre_days":       "0",
		"elective":           "0",
		"emergency":          "0",
		"postmortems":        "0",
		"OPD_clinics":        "0",
		"OPD_patients":       "0",
		"anc_patients":       "0",
		"teaching_rounds":    "0",
		"students_taught":    "0",
		"mortality_reviews":  "0",
		"maternal":           "0",
		"perinatal":          "0",
		"surgical":           "0",
		"medical":            "0",
		"paed":               "0",
		"labs_requests":      "0",
		"imaging_requests":   "0",
		"lab_investigations": "0",
		"BS":                 "0",
		"HIV":                "0",
		"malaria":            "0",
		"TB":                 "0",
		"CBC":                "0",
		"chemistry":          "0",
		"hematology":         "0",
		"urinalysis":         "0",
		"gram_stain":         "0",
		"culture":            "0",
		"microbiology":       "0",
		"sensitivity_tests":  "0",
		"diagnostics":        "0",
		"xrays":              "0",
		"ct_scans":           "0",
		"obstetrics_scans":   "0",
		"abdominal_scans":    "0",
	}
}

func clinicianEntryCoreKeys() []string {
	return []string{
		"attendance",
		"ward_rounds",
		"patients_reviewed",
		"OPD_clinics",
		"OPD_patients",
		"teaching_rounds",
		"students_taught",
		"mortality_reviews",
		"labs_requests",
		"imaging_requests",
	}
}

func fallbackDepartmentDataPointKeys(departmentID int64) []string {
	switch departmentID {
	case 1:
		return []string{"theatre_days", "elective", "emergency", "postmortems", "xrays", "ct_scans"}
	case 2:
		return []string{"medical", "CBC", "chemistry", "hematology", "urinalysis"}
	case 3:
		return []string{"paed", "malaria", "TB", "CBC"}
	case 4:
		return []string{"theatre_days", "elective", "emergency", "anc_patients", "maternal", "perinatal", "obstetrics_scans", "abdominal_scans"}
	default:
		return []string{}
	}
}

func departmentMetricsTitle(departmentID int64, defaultName string) string {
	switch departmentID {
	case 1:
		return "Surgery Metrics"
	case 2:
		return "Internal Medicine Metrics"
	case 3:
		return "Paediatrics Metrics"
	case 4:
		return "Obstetrics and Gynaecology Metrics"
	default:
		if strings.TrimSpace(defaultName) == "" {
			return "Department Metrics"
		}
		return strings.TrimSpace(defaultName) + " Metrics"
	}
}

func buildClinicianEntrySections(ctx context.Context, db *sql.DB, departmentID int64, labels map[string]string) []ClinicianEntrySection {
	deptKeys, err := models.GetDepartmentRoleDataPoints(ctx, db, departmentID)
	if err != nil || len(deptKeys) == 0 {
		deptKeys = fallbackDepartmentDataPointKeys(departmentID)
	}

	core := map[string]struct{}{}
	for _, key := range clinicianEntryCoreKeys() {
		core[key] = struct{}{}
	}
	core["attendance"] = struct{}{}

	fields := make([]ClinicianEntryField, 0)
	seen := map[string]struct{}{}
	for _, key := range deptKeys {
		key = strings.TrimSpace(key)
		if key == "" {
			continue
		}
		if _, isCore := core[key]; isCore {
			continue
		}
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		label := strings.TrimSpace(labels[key])
		if label == "" {
			label = strings.ReplaceAll(strings.Title(strings.ReplaceAll(strings.ToLower(key), "_", " ")), " Opd", " OPD")
		}
		fields = append(fields, ClinicianEntryField{Key: key, Label: label})
	}

	if len(fields) == 0 {
		return []ClinicianEntrySection{}
	}

	return []ClinicianEntrySection{{
		Title:  departmentMetricsTitle(departmentID, ""),
		Fields: fields,
	}}
}

func resolveClinicianEntryKeys(ctx context.Context, db *sql.DB, departmentID int64) []string {
	keys := make([]string, 0)
	seen := map[string]struct{}{}

	for _, key := range clinicianEntryCoreKeys() {
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		keys = append(keys, key)
	}

	deptKeys, err := models.GetDepartmentRoleDataPoints(ctx, db, departmentID)
	if err != nil || len(deptKeys) == 0 {
		deptKeys = fallbackDepartmentDataPointKeys(departmentID)
	}
	for _, key := range deptKeys {
		key = strings.TrimSpace(key)
		if key == "" {
			continue
		}
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		keys = append(keys, key)
	}

	if _, ok := seen["attendance"]; !ok {
		keys = append(keys, "attendance")
	}

	return keys
}

func applyDynamicReportValues(ctx context.Context, db *sql.DB, reportID int, extraKeys []string, values map[string]string) {
	if reportID <= 0 {
		return
	}
	keys := make([]string, 0, len(values)+len(extraKeys))
	seen := map[string]struct{}{}
	for key := range values {
		seen[key] = struct{}{}
		keys = append(keys, key)
	}
	for _, key := range extraKeys {
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		keys = append(keys, key)
		if _, exists := values[key]; !exists {
			values[key] = "0"
		}
	}
	dynamicValues, err := models.GetWeeklyReportValuesByElementKeys(ctx, db, reportID, keys)
	if err != nil {
		return
	}
	for key, value := range dynamicValues {
		values[key] = strconv.FormatInt(value, 10)
	}
}

func defaultClinicianEntryLabels() map[string]string {
	return map[string]string{
		"attendance":         "Attendance Days (Auto)",
		"ward_rounds":        "Ward Rounds",
		"patients_reviewed":  "Patients Reviewed",
		"theatre_days":       "Theatre Days",
		"elective":           "Elective Procedures",
		"emergency":          "Emergency Procedures",
		"postmortems":        "Postmortems",
		"OPD_clinics":        "OPD Clinics",
		"OPD_patients":       "OPD Patients",
		"anc_patients":       "ANC Patients",
		"teaching_rounds":    "Teaching Rounds",
		"students_taught":    "Students Taught",
		"mortality_reviews":  "Mortality Reviews",
		"maternal":           "Maternal Reviews",
		"perinatal":          "Perinatal Reviews",
		"surgical":           "Surgical Reviews",
		"medical":            "Medical Reviews",
		"paed":               "Paediatric Reviews",
		"labs_requests":      "Lab Requests",
		"imaging_requests":   "Imaging Requests",
		"lab_investigations": "Lab Investigations",
		"BS":                 "BS Tests",
		"HIV":                "HIV Tests",
		"malaria":            "Malaria Cases Reviewed",
		"TB":                 "TB Cases Reviewed",
		"CBC":                "CBC Tests",
		"chemistry":          "Chemistry Tests",
		"hematology":         "Hematology Tests",
		"urinalysis":         "Urinalysis",
		"gram_stain":         "Gram Stain",
		"culture":            "Culture",
		"microbiology":       "Microbiology",
		"sensitivity_tests":  "Sensitivity Tests",
		"diagnostics":        "Diagnostics",
		"xrays":              "X-Rays",
		"ct_scans":           "CT Scans",
		"obstetrics_scans":   "Obstetrics Scans",
		"abdominal_scans":    "Abdominal Scans",
	}
}

func resolveClinicianEntryLabels(ctx context.Context, db *sql.DB) map[string]string {
	labels := defaultClinicianEntryLabels()
	custom, err := models.GetReportDataElementDisplayMap(ctx, db)
	if err != nil {
		return labels
	}
	for key, name := range custom {
		key = strings.TrimSpace(key)
		name = strings.TrimSpace(name)
		if key == "" || name == "" {
			continue
		}
		labels[key] = name
	}
	return labels
}

func clinicianEntryValuesFromReport(report *models.ClinicianReportHistoryRow) map[string]string {
	values := defaultClinicianEntryValues()
	values["attendance"] = formatNullInt(report.Qn01)
	values["ward_rounds"] = formatNullInt(report.Qn02)
	values["patients_reviewed"] = formatNullInt(report.Qn03)
	values["theatre_days"] = formatNullInt(report.Qn04)
	values["elective"] = formatNullInt(report.Qn05)
	values["emergency"] = formatNullInt(report.Qn06)
	values["postmortems"] = formatNullInt(report.Qn07)
	values["OPD_clinics"] = formatNullInt(report.Qn08)
	values["OPD_patients"] = formatNullInt(report.Qn09)
	values["anc_patients"] = formatNullInt(report.Qn10)
	values["teaching_rounds"] = formatNullInt(report.Qn11)
	values["students_taught"] = formatNullInt(report.Qn12)
	values["mortality_reviews"] = formatNullInt(report.Qn13)
	values["maternal"] = formatNullInt(report.Qn14)
	values["perinatal"] = formatNullInt(report.Qn15)
	values["surgical"] = formatNullInt(report.Qn16)
	values["medical"] = formatNullInt(report.Qn17)
	values["paed"] = formatNullInt(report.Qn18)
	values["labs_requests"] = formatNullInt(report.Qn19)
	values["imaging_requests"] = formatNullInt(report.Qn20)
	values["lab_investigations"] = formatNullInt(report.Qn21)
	values["BS"] = formatNullInt(report.Qn22)
	values["HIV"] = formatNullInt(report.Qn23)
	values["malaria"] = formatNullInt(report.Qn24)
	values["TB"] = formatNullInt(report.Qn25)
	values["CBC"] = formatNullInt(report.Qn26)
	values["chemistry"] = formatNullInt(report.Qn27)
	values["hematology"] = formatNullInt(report.Qn28)
	values["urinalysis"] = formatNullInt(report.Qn29)
	values["gram_stain"] = formatNullInt(report.Qn30)
	values["culture"] = formatNullInt(report.Qn31)
	values["microbiology"] = formatNullInt(report.Qn32)
	values["sensitivity_tests"] = formatNullInt(report.Qn33)
	values["diagnostics"] = formatNullInt(report.Qn34)
	values["xrays"] = formatNullInt(report.Qn35)
	values["ct_scans"] = formatNullInt(report.Qn36)
	values["obstetrics_scans"] = formatNullInt(report.Qn37)
	values["abdominal_scans"] = formatNullInt(report.Qn38)
	if report.DaysWorked.Valid && strings.TrimSpace(report.DaysWorked.String) != "" {
		selectedDates := splitDaysWorkedCSV(report.DaysWorked.String)
		values["attendance"] = strconv.Itoa(len(selectedDates))
	}
	return values
}

func buildWeekDayChecks(start, stop time.Time, daysWorkedCSV string) []WeekDayCheck {
	selected := map[string]bool{}
	for _, item := range splitDaysWorkedCSV(daysWorkedCSV) {
		selected[item] = true
	}

	weekDays := []WeekDayCheck{}
	for day := normalizeDateOnly(start); !day.After(normalizeDateOnly(stop)); day = day.AddDate(0, 0, 1) {
		isoDate := day.Format("2006-01-02")
		weekDays = append(weekDays, WeekDayCheck{
			Name:    day.Weekday().String(),
			Short:   day.Format("Mon"),
			Date:    isoDate,
			Checked: selected[isoDate],
		})
	}

	return weekDays
}

func splitDaysWorkedCSV(daysWorkedCSV string) []string {
	parts := strings.Split(strings.TrimSpace(daysWorkedCSV), ",")
	items := make([]string, 0, len(parts))
	for _, p := range parts {
		item := strings.TrimSpace(p)
		if item == "" {
			continue
		}
		items = append(items, item)
	}
	return items
}

func normalizeDateOnly(value time.Time) time.Time {
	return time.Date(value.Year(), value.Month(), value.Day(), 0, 0, 0, 0, time.Local)
}

func collectDaysWorked(c *gin.Context, employeeID int64, weekStart, weekStop time.Time) (string, int64) {
	key := fmt.Sprintf("input[%d][days_worked][]", employeeID)
	selectedRaw := c.PostFormArray(key)
	seen := map[string]bool{}
	selected := make([]string, 0, len(selectedRaw))

	for _, item := range selectedRaw {
		parsed, err := parseISODate(item)
		if err != nil {
			continue
		}
		dateOnly := normalizeDateOnly(parsed)
		if dateOnly.Before(normalizeDateOnly(weekStart)) || dateOnly.After(normalizeDateOnly(weekStop)) {
			continue
		}
		isoDate := dateOnly.Format("2006-01-02")
		if seen[isoDate] {
			continue
		}
		seen[isoDate] = true
		selected = append(selected, isoDate)
	}

	sort.Strings(selected)
	return strings.Join(selected, ","), int64(len(selected))
}

func nullableClinicianEntryValues() map[string]sql.NullInt64 {
	zero := map[string]sql.NullInt64{}
	for key := range defaultClinicianEntryValues() {
		zero[key] = sql.NullInt64{}
	}
	return zero
}

func collectClinicianEntryValues(c *gin.Context, employeeID int64, attendance int64, allowedKeys []string) map[string]sql.NullInt64 {
	field := func(name string) sql.NullInt64 {
		raw := strings.TrimSpace(c.PostForm(fmt.Sprintf("input[%d][%s]", employeeID, name)))
		if raw == "" {
			return sql.NullInt64{}
		}
		parsed, err := strconv.ParseInt(raw, 10, 64)
		if err != nil {
			return sql.NullInt64{}
		}
		return sql.NullInt64{Int64: parsed, Valid: true}
	}

	values := nullableClinicianEntryValues()
	attendanceValue := attendance
	if attendanceValue == 0 {
		rawAttendance := field("attendance")
		if rawAttendance.Valid {
			attendanceValue = rawAttendance.Int64
		}
	}
	for _, key := range allowedKeys {
		key = strings.TrimSpace(key)
		if key == "" || key == "attendance" {
			continue
		}
		values[key] = field(key)
	}
	values["attendance"] = sql.NullInt64{Int64: attendanceValue, Valid: true}
	return values
}

func buildReportHistoryPeriodQuery(year, month, week int) string {
	params := []string{}
	if year > 0 {
		params = append(params, fmt.Sprintf("year=%d", year))
	}
	if month > 0 {
		params = append(params, fmt.Sprintf("month=%d", month))
	}
	if week > 0 {
		params = append(params, fmt.Sprintf("week=%d", week))
	}
	if len(params) == 0 {
		return ""
	}
	return "?" + strings.Join(params, "&")
}

func buildReportHistoryPeriodFilterQuery(status string, year, month, week int) string {
	if year < 0 {
		year = 0
	}
	if month < 0 {
		month = 0
	}
	if week < 0 {
		week = 0
	}

	values := url.Values{}
	values.Set("status", normalizeReportSubmissionStatus(status))
	values.Set("year", strconv.Itoa(year))
	values.Set("month", strconv.Itoa(month))
	values.Set("week", strconv.Itoa(week))

	return "/reports/analysis?" + values.Encode()
}

func buildFacilityReportReviewURL(status string, year, month, week int) string {
	return buildReportSubmissionsURL(0, 0, year, month, week, status)
}

func buildReportHistoryExportURL(format, status string, year, month, week int) string {
	if year < 0 {
		year = 0
	}
	if month < 0 {
		month = 0
	}
	if week < 0 {
		week = 0
	}

	values := url.Values{}
	values.Set("format", strings.ToLower(strings.TrimSpace(format)))
	values.Set("status", normalizeReportSubmissionStatus(status))
	values.Set("year", strconv.Itoa(year))
	values.Set("month", strconv.Itoa(month))
	values.Set("week", strconv.Itoa(week))

	return "/reports/analysis/export?" + values.Encode()
}

func buildFacilityReportReviewExportURL(format, status string, year, month, week int) string {
	return buildReportSubmissionsExportURL(format, 0, 0, year, month, week, status)
}

func buildReportHistoryExportFilename(filterStatus string, year, week int) string {
	name := "clinician-report-history"
	if filterStatus != "" && filterStatus != "all" {
		name += "-" + filterStatus
	}
	if year > 0 {
		name += fmt.Sprintf("-%d", year)
	}
	if week > 0 {
		name += fmt.Sprintf("-week-%02d", week)
	}
	return name
}

func buildFacilityReportReviewExportFilename(filterStatus string, year, week int) string {
	name := "facility-report-review"
	if filterStatus != "" && filterStatus != "pending" {
		name += "-" + filterStatus
	}
	if year > 0 {
		name += fmt.Sprintf("-%d", year)
	}
	if week > 0 {
		name += fmt.Sprintf("-week-%02d", week)
	}
	return name
}

func clinicianReportHistoryFilterTitle(filterStatus string) string {
	switch filterStatus {
	case "submitted":
		return "Submitted Reports"
	case "approved":
		return "Approved Reports"
	case "declined":
		return "Declined Reports"
	case "draft":
		return "Draft Reports"
	default:
		return "All Reports"
	}
}

func facilityReportReviewFilterTitle(filterStatus string) string {
	switch filterStatus {
	case "approved":
		return "Approved Submitted Reports"
	case "declined":
		return "Declined Submitted Reports"
	case "all":
		return "All Submitted Reports"
	default:
		return "Pending Submitted Reports"
	}
}

func clinicianReportHistoryPeriodLabel(year, week int) string {
	switch {
	case year <= 0:
		return "Across all records"
	case week <= 0:
		return fmt.Sprintf("Across all weeks in %d", year)
	default:
		return fmt.Sprintf("Week %02d in %d", week, year)
	}
}

func clinicianEntryStatusLabel(historyStatus string) string {
	switch historyStatus {
	case "declined":
		return "Declined"
	case "submitted":
		return "Submitted"
	default:
		return "Draft"
	}
}

func sanitizeHistoryReturnURL(value string) string {
	if value == "" {
		return "/reports/analysis"
	}
	if strings.HasPrefix(value, "/reports/history") || strings.HasPrefix(value, "/reports/analysis") || strings.HasPrefix(value, "/reports/bulk") {
		return value
	}
	return "/reports/analysis"
}

func formatEmployeeName(firstName, lastName, otherName string) string {
	parts := []string{}
	for _, part := range []string{firstName, lastName, otherName} {
		part = strings.TrimSpace(part)
		if part != "" {
			parts = append(parts, part)
		}
	}
	if len(parts) == 0 {
		return "Unknown staff member"
	}
	return strings.Join(parts, " ")
}

func buildBulkEntryURL(departmentID int) string {
	params := url.Values{}
	if departmentID > 0 {
		params.Set("department", strconv.Itoa(departmentID))
	}
	encoded := params.Encode()
	if encoded == "" {
		return "/reports/bulk"
	}
	return "/reports/bulk?" + encoded
}

func buildBulkEntryFormURL(employeeID, departmentID int) string {
	params := url.Values{}
	params.Set("employee", strconv.Itoa(employeeID))
	params.Set("return_to", buildBulkEntryURL(departmentID))
	return "/reports/new/0?" + params.Encode()
}

func sanitizeFacilityReportReviewURL(value string) string {
	if value == "" {
		return "/reports/analysis"
	}
	if strings.HasPrefix(value, "/reports/review") || strings.HasPrefix(value, "/reports/analysis") {
		return value
	}
	return "/reports/analysis"
}

func exportWeekRange(start, stop sql.NullTime) string {
	if start.Valid && stop.Valid {
		return fmt.Sprintf("%s - %s", start.Time.Format("02 Jan 2006"), stop.Time.Format("02 Jan 2006"))
	}
	if start.Valid {
		return start.Time.Format("02 Jan 2006")
	}
	if stop.Valid {
		return stop.Time.Format("02 Jan 2006")
	}
	return "-"
}

func exportDateTime(value sql.NullTime) string {
	if !value.Valid {
		return "-"
	}
	return value.Time.Format("02 Jan 2006 15:04")
}

func exportReportSubmissionStatus(value sql.NullString) string {
	if !value.Valid {
		return "Draft"
	}
	switch value.String {
	case "Submitted":
		return "Submitted"
	default:
		return "Draft"
	}
}

func exportReportApprovalStatus(value sql.NullString) string {
	if !value.Valid {
		return "Pending Review"
	}
	switch value.String {
	case "Approved":
		return "Approved"
	case "Rejected", "Declined":
		return "Declined"
	default:
		return "Pending Review"
	}
}

func formatNullDate(value sql.NullTime) string {
	if !value.Valid {
		return ""
	}
	return value.Time.Format("2006-01-02")
}

func formatNullInt(value sql.NullInt64) string {
	if !value.Valid {
		return "0"
	}
	return strconv.FormatInt(value.Int64, 10)
}

func beginningOfWeek(now time.Time) time.Time {
	offset := (int(now.Weekday()) + 6) % 7
	weekStart := now.AddDate(0, 0, -offset)
	return time.Date(weekStart.Year(), weekStart.Month(), weekStart.Day(), 0, 0, 0, 0, weekStart.Location())
}

func previousReportingWeekRange(now time.Time) (time.Time, time.Time) {
	thisWeekStart := beginningOfWeek(now)
	previousWeekStart := thisWeekStart.AddDate(0, 0, -7)
	return previousWeekStart, previousWeekStart.AddDate(0, 0, 6)
}

func sameCalendarDate(a, b time.Time) bool {
	return a.Year() == b.Year() && a.Month() == b.Month() && a.Day() == b.Day()
}

func isPreviousReportingWeekRange(start, stop, now time.Time) bool {
	expectedStart, expectedStop := previousReportingWeekRange(now)
	return sameCalendarDate(start, expectedStart) && sameCalendarDate(stop, expectedStop)
}

func isCompletedReportingWeekRange(start, stop, now time.Time) bool {
	startDate := normalizeDateOnly(start)
	stopDate := normalizeDateOnly(stop)
	if !sameCalendarDate(stopDate, startDate.AddDate(0, 0, 6)) {
		return false
	}
	if startDate.Weekday() != time.Monday {
		return false
	}
	thisWeekStart := beginningOfWeek(now)
	return startDate.Before(thisWeekStart)
}

func resolveClinicianEntryWeekSelection(requestedStart string, pendingWeeks []models.ClinicianWeekOption, now time.Time) (time.Time, time.Time, string) {
	defaultStart, defaultStop := previousReportingWeekRange(now)
	defaultLabel := fmt.Sprintf("Week %02d (%s - %s)", func() int {
		_, wk := defaultStart.ISOWeek()
		return wk
	}(), defaultStart.Format("02 Jan 2006"), defaultStop.Format("02 Jan 2006"))

	if len(pendingWeeks) == 0 {
		return defaultStart, defaultStop, defaultLabel
	}

	selected := pendingWeeks[0]
	requestedStart = strings.TrimSpace(requestedStart)
	if requestedStart != "" {
		for _, option := range pendingWeeks {
			if option.StartDate == requestedStart {
				selected = option
				break
			}
		}
	}

	start, err := parseISODate(selected.StartDate)
	if err != nil {
		return defaultStart, defaultStop, defaultLabel
	}
	stop := start.AddDate(0, 0, 6)
	label := strings.TrimSpace(selected.Label)
	if label == "" {
		_, weekVal := start.ISOWeek()
		label = fmt.Sprintf("Week %02d (%s - %s)", weekVal, start.Format("02 Jan 2006"), stop.Format("02 Jan 2006"))
	}

	return start, stop, label
}

func buildClinicianEntryWeekOptions(reportID int, employeeID int, returnURL string, pendingWeeks []models.ClinicianWeekOption, selectedStart time.Time) []ClinicianEntryWeekOption {
	if reportID > 0 {
		return nil
	}
	options := make([]ClinicianEntryWeekOption, 0, len(pendingWeeks))
	selectedDate := selectedStart.Format("2006-01-02")
	for _, option := range pendingWeeks {
		params := url.Values{}
		if employeeID > 0 {
			params.Set("employee", strconv.Itoa(employeeID))
		}
		if strings.TrimSpace(returnURL) != "" {
			params.Set("return_to", returnURL)
		}
		params.Set("start", option.StartDate)
		optionLabel := strings.TrimSpace(option.Label)
		if option.StartDate != "" && option.EndDate != "" {
			optionLabel = fmt.Sprintf("%s to %s", formatISODateToDMY(option.StartDate), formatISODateToDMY(option.EndDate))
		}
		options = append(options, ClinicianEntryWeekOption{
			StartDate: option.StartDate,
			Label:     optionLabel,
			URL:       "/reports/new/0?" + params.Encode(),
			Selected:  option.StartDate == selectedDate,
		})
	}
	return options
}

func formatISODateToDMY(value string) string {
	parsed, err := time.Parse("2006-01-02", strings.TrimSpace(value))
	if err != nil {
		return strings.TrimSpace(value)
	}
	return parsed.Format("02-01-2006")
}

func parseISODate(value string) (time.Time, error) {
	parsed, err := time.Parse("2006-01-02", strings.TrimSpace(value))
	if err != nil {
		return time.Time{}, err
	}
	return time.Date(parsed.Year(), parsed.Month(), parsed.Day(), 0, 0, 0, 0, time.Local), nil
}

func buildBulkEntryWeekOptions(now time.Time, maxWeeks int) []models.ClinicianWeekOption {
	if maxWeeks <= 0 {
		maxWeeks = 12
	}
	start, _ := previousReportingWeekRange(now)
	options := make([]models.ClinicianWeekOption, 0, maxWeeks)
	for i := 0; i < maxWeeks; i++ {
		weekStart := start.AddDate(0, 0, -(7 * i))
		weekStop := weekStart.AddDate(0, 0, 6)
		yearVal, weekVal := weekStart.ISOWeek()
		options = append(options, models.ClinicianWeekOption{
			Year:      yearVal,
			Week:      weekVal,
			StartDate: weekStart.Format("2006-01-02"),
			EndDate:   weekStop.Format("2006-01-02"),
			Label:     fmt.Sprintf("Week %02d (%s - %s)", weekVal, weekStart.Format("02 Jan 2006"), weekStop.Format("02 Jan 2006")),
		})
	}
	return options
}

func resolveBulkEntryWeekSelection(requestedStart string, options []models.ClinicianWeekOption) (time.Time, time.Time, string, error) {
	if len(options) == 0 {
		return time.Time{}, time.Time{}, "", errors.New("no available weeks")
	}
	selected := options[0]
	requestedStart = strings.TrimSpace(requestedStart)
	if requestedStart != "" {
		for _, option := range options {
			if option.StartDate == requestedStart {
				selected = option
				break
			}
		}
	}
	start, err := parseISODate(selected.StartDate)
	if err != nil {
		return time.Time{}, time.Time{}, "", err
	}
	stop := start.AddDate(0, 0, 6)
	label := strings.TrimSpace(selected.Label)
	if label == "" {
		_, weekVal := start.ISOWeek()
		label = fmt.Sprintf("Week %02d (%s - %s)", weekVal, start.Format("02 Jan 2006"), stop.Format("02 Jan 2006"))
	}
	return start, stop, label, nil
}

func parseBulkEntryWeekRange(startRaw, stopRaw string, now time.Time) (time.Time, time.Time, error) {
	startRaw = strings.TrimSpace(startRaw)
	stopRaw = strings.TrimSpace(stopRaw)
	if startRaw == "" || stopRaw == "" {
		start, stop := previousReportingWeekRange(now)
		return start, stop, nil
	}
	start, err := parseISODate(startRaw)
	if err != nil {
		return time.Time{}, time.Time{}, err
	}
	stop, err := parseISODate(stopRaw)
	if err != nil {
		return time.Time{}, time.Time{}, err
	}
	if !sameCalendarDate(stop, start.AddDate(0, 0, 6)) {
		return time.Time{}, time.Time{}, errors.New("invalid week range")
	}
	if !isCompletedReportingWeekRange(start, stop, now) {
		return time.Time{}, time.Time{}, errors.New("only completed weeks are allowed")
	}
	return start, stop, nil
}

func HandlerBulkCaptureList(c *gin.Context, db *sql.DB, sessionManager *scs.SessionManager) {
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

	if !utilities.RoleMatches(sesDetails.Rights, "Facility Admin") {
		c.String(http.StatusForbidden, "Bulk entry is only available to facility admins.")
		return
	}

	selectedDepartment, _ := strconv.Atoi(c.Query("department"))
	staffList, err := models.WeeklyBulkCapture(c.Request.Context(), db, sesDetails.HFID, 0, "", "")
	if err != nil {
		log.Printf("Error loading bulk entry staff list: %v", err)
		c.String(http.StatusInternalServerError, "Unable to load facility staff for bulk entry.")
		return
	}

	departmentCounts := map[int]int{}
	departmentLabels := map[int]string{}
	uniqueStaff := make([]*models.WeeklyReportExtended, 0, len(staffList))
	seenEmployees := make(map[int]bool, len(staffList))

	for _, item := range staffList {
		if item == nil || seenEmployees[item.EmpID] {
			continue
		}
		seenEmployees[item.EmpID] = true
		uniqueStaff = append(uniqueStaff, item)

		departmentID := item.DeptID
		departmentCounts[departmentID]++
		label := strings.TrimSpace(item.DepartmentName.String)
		if label == "" {
			label = "Unassigned"
		}
		departmentLabels[departmentID] = label
	}

	tabs := []BulkEntryDepartmentTab{{
		DepartmentID: 0,
		Label:        "All Staff",
		Count:        len(uniqueStaff),
		URL:          buildBulkEntryURL(0),
	}}

	departmentIDs := make([]int, 0, len(departmentLabels))
	for departmentID := range departmentLabels {
		departmentIDs = append(departmentIDs, departmentID)
	}
	sort.Slice(departmentIDs, func(i, j int) bool {
		return strings.ToLower(departmentLabels[departmentIDs[i]]) < strings.ToLower(departmentLabels[departmentIDs[j]])
	})
	for _, departmentID := range departmentIDs {
		tabs = append(tabs, BulkEntryDepartmentTab{
			DepartmentID: departmentID,
			Label:        departmentLabels[departmentID],
			Count:        departmentCounts[departmentID],
			URL:          buildBulkEntryURL(departmentID),
		})
	}

	rows := make([]BulkEntryRow, 0, len(uniqueStaff))
	for _, item := range uniqueStaff {
		if selectedDepartment > 0 && item.DeptID != selectedDepartment {
			continue
		}
		departmentName := strings.TrimSpace(item.DepartmentName.String)
		if departmentName == "" {
			departmentName = "Unassigned"
		}
		rows = append(rows, BulkEntryRow{
			EmployeeID:     item.EmpID,
			EmployeeName:   formatEmployeeName(item.Fname.String, item.Lname.String, item.Oname.String),
			EmployeeTitle:  strings.TrimSpace(item.EmpTitle.String),
			DepartmentName: departmentName,
			Specialisation: strings.TrimSpace(item.Specialisation.String),
			EntryURL:       buildBulkEntryFormURL(item.EmpID, selectedDepartment),
		})
	}

	sort.Slice(rows, func(i, j int) bool {
		if rows[i].DepartmentName != rows[j].DepartmentName {
			return rows[i].DepartmentName < rows[j].DepartmentName
		}
		return rows[i].EmployeeName < rows[j].EmployeeName
	})

	filterSummary := fmt.Sprintf("Showing all %d staff records in %s.", len(rows), sesDetails.HFName)
	if selectedDepartment > 0 {
		if label, ok := departmentLabels[selectedDepartment]; ok {
			filterSummary = fmt.Sprintf("Showing %d staff records in %s for %s.", len(rows), label, sesDetails.HFName)
		}
	}

	sessionData.Form = BulkEntryView{
		ScopeTitle:         "Facility Bulk Entry",
		ScopeSubtitle:      "Select a department tab, then open a staff record to enter weekly data for that clinician.",
		FilterSummary:      filterSummary,
		SelectedDepartment: selectedDepartment,
		Tabs:               tabs,
		Rows:               rows,
		CurrentURL:         buildBulkEntryURL(selectedDepartment),
	}
	utilities.GenerateHTML(c, sessionData, "base", "bulk-entry")
}

// Handler to generate capture form to collect data
func HandlerBulkCaptureForm(c *gin.Context, db *sql.DB, sessionManager *scs.SessionManager) {
	// Get session-specific data (e.g., user ID and HFID)
	sessionData, ok := Get_Session_Data(c, db, sessionManager, nil).(utilities.TemplateData)
	if !ok {
		log.Println("Failed to retrieve session data as TemplateData")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Unable to retrieve session data"})
		return
	}

	// Now access the SessionDetails from the TemplateData
	sesDetails, ok := sessionData.Ses.(utilities.SessionDetails)
	if !ok {
		log.Println("Failed to retrieve session details from TemplateData")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Unable to retrieve session details"})
		return
	}

	//log.Printf("Session Data HandlerReportFacilities: %+v", sesDetails)

	// Retrieve reporting week parameters from the request
	reportingWeekStart := c.Query("reporting_week_start")
	reportingWeekEnd := c.Query("reporting_week_end")

	// Parse empID from URL
	//empIDStr := c.Query("empID")
	//log.Printf("Hospital ID value from session: %s", empIDStr)

	hospitalID := sesDetails.HFID // Retrieve HFID from the session
	//log.Printf("Hospital ID value from session: %d", hospitalID)

	facilities, departments, err := models.GetFacilitiesAndDepartments(db)
	if err != nil {
		log.Printf("Error loading facilities or departments: %v", err)
		c.String(http.StatusInternalServerError, "Error loading facilities or departments")
		return
	}
	//log.Printf("Facilities: %+v", facilities)
	//log.Printf("Departments: %+v", departments)

	// Get the 'Created by' details using the userID from session data
	//userID := sessionManager.GetString(c.Request.Context(), "UserID")
	//log.Printf("User ID from session: %s", userID)

	// Fetch distinct staff members using the new DistinctEmployeeListForCapture function
	staffList, err := models.WeeklyBulkCapture(c.Request.Context(), db, hospitalID, 0, reportingWeekStart, reportingWeekEnd)
	if err != nil {
		staffList = nil
	}

	// Prepare data to pass to the template
	data := gin.H{
		"sessionData": sessionData,
		"facilities":  facilities,
		"departments": departments,
		"zdata":       staffList,
	}

	// Render the template with the staff data
	utilities.GenerateHTML(c, data, "base", "bulkcapturelanding")
}

// Handler fxn to display bulk entry form dynamically
func HandlerBulkCaptureForm2(c *gin.Context, db *sql.DB, sessionManager *scs.SessionManager) {

	// Get session-specific data (e.g., user ID and HFID)
	sessionData, ok := Get_Session_Data(c, db, sessionManager, nil).(utilities.TemplateData)
	if !ok {
		log.Println("Failed to retrieve session data as TemplateData")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Unable to retrieve session data"})
		return
	}

	// Now access the SessionDetails from the TemplateData
	sesDetails, ok := sessionData.Ses.(utilities.SessionDetails)
	if !ok {
		log.Println("Failed to retrieve session details from TemplateData")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Unable to retrieve session details"})
		return
	}

	//log.Printf("Session Data HandlerReportFacilities: %+v", sesDetails)

	// Parse empID from URL
	//empIDStr := c.Query("empID")
	//log.Printf("Hospital ID value from session: %s", empIDStr)

	facilityID := sesDetails.HFID // Retrieve HFID from the session

	departmentID := c.Query("departmentID")
	if departmentID != "" || c.Query("week_start") != "" || c.Query("week_end") != "" {
		deptID := 0
		if strings.TrimSpace(departmentID) != "" {
			parsedDeptID, err := strconv.Atoi(departmentID)
			if err != nil || parsedDeptID <= 0 {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid department ID"})
				return
			}
			deptID = parsedDeptID
		}

		weekStart, weekStop, err := parseBulkEntryWeekRange(c.Query("week_start"), c.Query("week_end"), time.Now())
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid week selection"})
			return
		}

		staffList, err := models.WeeklyBulkCapture(c.Request.Context(), db, facilityID, int64(deptID), weekStart.Format("2006-01-02"), weekStop.Format("2006-01-02"))
		if err != nil {
			log.Printf("Error loading bulk staff list: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch staff"})
			return
		}

		labels := resolveClinicianEntryLabels(c.Request.Context(), db)
		entryKeys := make([]string, 0)
		seenKeys := map[string]struct{}{}
		for _, item := range staffList {
			if item == nil {
				continue
			}
			staffKeys := resolveClinicianEntryKeys(c.Request.Context(), db, int64(item.DeptID))
			for _, key := range staffKeys {
				key = strings.TrimSpace(key)
				if key == "" {
					continue
				}
				if _, exists := seenKeys[key]; exists {
					continue
				}
				seenKeys[key] = struct{}{}
				entryKeys = append(entryKeys, key)
			}
		}

		type bulkStaffRow struct {
			EmployeeID int64  `json:"employeeID"`
			Name       string `json:"name"`
			Title      string `json:"title"`
		}
		type bulkDataPoint struct {
			Key   string `json:"key"`
			Label string `json:"label"`
		}

		rows := make([]bulkStaffRow, 0, len(staffList))
		for _, item := range staffList {
			if item == nil {
				continue
			}
			rows = append(rows, bulkStaffRow{
				EmployeeID: int64(item.EmpID),
				Name:       formatEmployeeName(item.Fname.String, item.Lname.String, item.Oname.String),
				Title:      strings.TrimSpace(item.EmpTitle.String),
			})
		}

		points := make([]bulkDataPoint, 0, len(entryKeys))
		for _, key := range entryKeys {
			key = strings.TrimSpace(key)
			if key == "" {
				continue
			}
			label := strings.TrimSpace(labels[key])
			if label == "" {
				label = strings.ReplaceAll(strings.Title(strings.ReplaceAll(strings.ToLower(key), "_", " ")), " Opd", " OPD")
			}
			points = append(points, bulkDataPoint{Key: key, Label: label})
		}

		c.JSON(http.StatusOK, gin.H{"staff": rows, "dataPoints": points})
		return
	}

	facilities, departments, err := models.GetFacilitiesAndDepartments(db)
	if err != nil {
		log.Printf("Error loading facilities or departments: %v", err)
		c.String(http.StatusInternalServerError, "Error loading facilities or departments")
		return
	}

	weekOptions := buildBulkEntryWeekOptions(time.Now(), 16)
	selectedWeekStart, selectedWeekStop, selectedWeekLabel, err := resolveBulkEntryWeekSelection(c.Query("week_start"), weekOptions)
	if err != nil {
		c.String(http.StatusBadRequest, "Invalid week selection")
		return
	}

	// Prepare data to pass to the template
	data := gin.H{
		"sessionData": sessionData,
		"facilities":  facilities,
		"departments": departments,
		"weekOptions": weekOptions,
		"weekStart":   selectedWeekStart.Format("2006-01-02"),
		"weekEnd":     selectedWeekStop.Format("2006-01-02"),
		"weekLabel":   selectedWeekLabel,
		"zdata":       nil, // Default empty data on initial load
	}

	// Render the template
	utilities.GenerateHTML(c, data, "base", "bulkcapturelanding")
}

// Handler to generate capture form to collect data - Recode this for single entry form
func HandlerBulkCaptureFormID(c *gin.Context, db *sql.DB, sessionManager *scs.SessionManager) {
	// Get session-specific data (e.g., user ID and HFID)
	sessionData, ok := Get_Session_Data(c, db, sessionManager, nil).(utilities.TemplateData)
	if !ok {
		log.Println("Failed to retrieve session data as TemplateData")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Unable to retrieve session data"})
		return
	}

	// Now access the SessionDetails from the TemplateData
	sesDetails, ok := sessionData.Ses.(utilities.SessionDetails)
	if !ok {
		log.Println("Failed to retrieve session details from TemplateData")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Unable to retrieve session details"})
		return
	}

	log.Printf("Session Data HandlerReportFacilities: %+v", sesDetails)

	// Retrieve reporting week parameters from the request
	reportingWeekStart := c.Query("reporting_week_start")
	reportingWeekEnd := c.Query("reporting_week_end")

	// Parse empID from URL
	empIDStr := c.Query("empID")
	log.Printf("Hospital ID value from session: %s", empIDStr)
	empID, err := strconv.ParseInt(empIDStr, 10, 64)
	if err != nil {
		log.Printf("Invalid empID: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid empID"})
		return
	}

	hospitalID := sesDetails.HFID // Retrieve HFID from the session
	log.Printf("Hospital ID value from session: %d", hospitalID)

	// Fetch distinct staff members using the new DistinctEmployeeListForCapture function
	staffList, err := models.WeeklyBulkCaptureID(c.Request.Context(), db, hospitalID, empID, reportingWeekStart, reportingWeekEnd)
	if err != nil {
		staffList = nil
	}

	// Variable to hold the staff data for the template
	var zdata any

	if err == nil {
		zdata = staffList // Pass the staff data (staffList) to zdata
	} else {
		zdata = nil // In case of error, pass nil
	}

	// Get the session data and pass the staff data (zdata) to the template
	data := Get_Session_Data(c, db, sessionManager, zdata)

	// Render the template with the staff data
	utilities.GenerateHTML(c, data, "base", "bulkcaptureform-old2")
}

// Handler to view submitted reports for approval by facility
func HandlerReportFacilities(c *gin.Context, db *sql.DB, sessionManager *scs.SessionManager) {
	// Get session-specific data (e.g., user ID and HFID)
	sessionData, ok := Get_Session_Data(c, db, sessionManager, nil).(utilities.TemplateData)
	if !ok {
		log.Println("Failed to retrieve session data as TemplateData")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Unable to retrieve session data"})
		return
	}

	// Now access the SessionDetails from the TemplateData
	sesDetails, ok := sessionData.Ses.(utilities.SessionDetails)
	if !ok {
		log.Println("Failed to retrieve session details from TemplateData")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Unable to retrieve session details"})
		return
	}

	//log.Printf("Session Data HandlerReportFacilities: %+v", sesDetails)

	var zdata any

	hospitalID := sesDetails.HFID // Retrieve HFID from the session
	//departmentID := sesDetails.HDID
	//employeeID := sesDetails.EmpID
	//log.Printf("Hospital ID: %d, Department ID: %d, Employee ID: %d", hospitalID, departmentID, employeeID)

	// Pass the hospitalID to the query
	dts, er := models.WeeklyReportFacilities(c, db, hospitalID)
	if er == nil {
		zdata = dts
	} else {
		zdata = nil
	}

	// Use the TemplateData for rendering
	data := Get_Session_Data(c, db, sessionManager, zdata)

	utilities.GenerateHTML(c, data, "base", "submissions")
}

// Handler to view submitted reports for approval by facility or department
func HandlerReportFacilities2(c *gin.Context, db *sql.DB, sessionManager *scs.SessionManager) {
	// Get session-specific data (e.g., user ID and HFID)
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

	userRole := roleFromRights(sesDetails.Rights)
	isNational := userRole == utilities.RoleNationalAdmin
	hospitalID := sesDetails.HFID // Retrieve HFID from the session

	if !canAccessReportSubmissions(userRole) {
		c.JSON(http.StatusForbidden, gin.H{"error": "access denied"})
		return
	}

	if !isNational && hospitalID <= 0 {
		c.JSON(http.StatusForbidden, gin.H{"error": "facility assignment is required for facility submissions"})
		return
	}

	log.Printf("Hospital ID: %d", hospitalID)

	// Determine current year and month
	currentYear := time.Now().Year()
	currentMonth := time.Now().Month()

	// Retrieve query parameters for filtering
	year := c.Query("year")
	month := c.Query("month")

	if year == "" {
		year = fmt.Sprintf("%d", currentYear)
	}

	if month == "" && year == fmt.Sprintf("%d", currentYear) {
		month = fmt.Sprintf("%d", currentMonth)
	}

	yearint, err := strconv.Atoi(year)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid year"})
		return
	}

	monthint, err := strconv.Atoi(month)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid month"})
		return
	}

	// Detect the URL path and call the corresponding function
	var dts []*models.WeeklyReportSubmission
	flag, isDepartmentRoute, ok := resolveReportSubmissionsRoute(c.FullPath(), isNational)
	if !ok {
		log.Println("Invalid endpoint")
		c.JSON(http.StatusNotFound, gin.H{"error": "Invalid endpoint"})
		return
	}

	if !isDepartmentRoute {
		log.Println("Handler called for facilities submissions")
		dts, err = models.WeeklyReportFacilities2(c.Request.Context(), db, hospitalID, yearint, monthint, flag, userRole)

	} else {
		log.Println("Handler called for departments submissions")
		log.Printf("Departments submissions facility id: %d", hospitalID)
		week := c.Query("week")

		resolvedHospitalID, scopeErr := resolveSubmissionsScopeFacilityID(isNational, hospitalID, c.Query("facility"))
		if scopeErr != nil {
			status := http.StatusBadRequest
			if !isNational {
				status = http.StatusForbidden
			}
			c.JSON(status, gin.H{"error": scopeErr.Error()})
			return
		}
		hospitalID = resolvedHospitalID
		if isNational {
			log.Printf("National department submissions facility id: %d", hospitalID)
		}

		dts, err = models.WeeklyReportDepartments(c.Request.Context(), db, hospitalID, userRole, week, yearint, monthint)
	}

	//dts, err := models.WeeklyReportFacilities2(c.Request.Context(), db, hospitalID, yearint, monthint, flag)
	if err != nil {
		log.Printf("Error retrieving reports: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch reports"})
		return
	}

	/*if err != nil {
		log.Printf("Error retrieving reports: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch reports"})
		return
	}*/

	// Use the TemplateData for rendering
	data := Get_Session_Data(c, db, sessionManager, dts)
	utilities.GenerateHTML(c, data, "base", "submissions_facilities")
}

// Handler to call WeeklyreportList() fxn to filter and populate table based on facility id and start date of report.
func HandlerReportList(c *gin.Context, db *sql.DB, sessionManager *scs.SessionManager) {
	var zdata any

	// Retrieve facility ID from the path parameter and start date from the query parameter
	facility := c.Query("facility")
	department := c.Query("dept")
	start := c.Query("start")

	// Log for debugging purposes
	//log.Printf("Filtering by Facility ID: %s, Start Date: %s, Dept ID: %s", facility, start, department)

	// Call WeeklyreportList with the filtering parameters
	dts, err := models.WeeklyreportList(c.Request.Context(), db, facility, department, start, "", 0, 0)
	if err == nil {
		zdata = dts
	} else {
		zdata = nil
	}

	log.Printf("ReportList data from db: %v", zdata)

	// Pass filtered data to the HTML template
	data := Get_Session_Data(c, db, sessionManager, zdata)
	utilities.GenerateHTML(c, data, "base", "reports")
}

// Handler to call WeeklyreportList() fxn to filter and populate table based on facility id and start date of report.
func HandlerReportList2(c *gin.Context, db *sql.DB, sessionManager *scs.SessionManager) {
	var zdata any

	// Retrieve facility ID from the path parameter and start date from the query parameter
	facility := c.Query("facility")
	department := c.Query("dept")
	start := c.Query("start")

	// Log for debugging purposes
	//log.Printf("Filtering by Facility ID: %s, Start Date: %s, Dept ID: %s", facility, start, department)

	// Call WeeklyreportList with the filtering parameters
	dts, err := models.WeeklyreportList(c.Request.Context(), db, facility, department, start, "", 0, 0)
	if err == nil {
		zdata = dts
	} else {
		zdata = nil
	}

	log.Printf("ReportList data from db: %v", zdata)

	// Pass filtered data to the HTML template
	data := Get_Session_Data(c, db, sessionManager, zdata)
	utilities.GenerateHTML(c, data, "base", "viewreport2")
}

// Handler to retrieve a specific weekly report by report ID and populate the viewreport template.
func HandlerReportView(c *gin.Context, db *sql.DB, sessionManager *scs.SessionManager) {
	var zdata any

	// Get session-specific data (e.g., user ID and HFID)
	sessionData, ok := Get_Session_Data(c, db, sessionManager, nil).(utilities.TemplateData)
	if !ok {
		log.Println("Failed to retrieve session data as TemplateData")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Unable to retrieve session data"})
		return
	}

	// Now access the SessionDetails from the TemplateData
	sesDetails, ok := sessionData.Ses.(utilities.SessionDetails)
	if !ok {
		log.Println("Failed to retrieve session details from TemplateData")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Unable to retrieve session details"})
		return
	}

	// Retrieve report ID from the path parameter
	reportID := c.Param("id")
	//departmentID := c.Query("department")

	// Log for debugging purposes
	log.Printf("Fetching report by ID: %s", reportID)

	// Convert reportID to integer
	id, err := strconv.Atoi(reportID)
	if err != nil {
		log.Printf("Invalid report ID: %s", reportID)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid report ID"})
		return
	}

	facilityID := sesDetails.HFID // Retrieve HFID from the session
	log.Printf("Hospital ID value from session: %d", facilityID)

	// Check if departmentID is provided
	departmentID := c.Query("department") // Capture departmentID from the query

	if departmentID != "" {
		deptID, err := strconv.Atoi(departmentID)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid department ID"})
			return
		}

		log.Printf("Executing GetFacilitiesAndDepartments fxn from HandlerBulkCaptureForm2")

		staff, dataPoints, err := models.GetStaffAndDataPointsByDepartment(db, deptID, int(facilityID))
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch staff and data points"})
			return
		}

		// Convert dataPoints into a usable JSON format if needed
		/*jsonDataPoints, err := json.Marshal(dataPoints)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to process data points"})
			return
		}*/

		// Return as JSON for AJAX
		c.JSON(http.StatusOK, gin.H{"staff": staff, "dataPoints": dataPoints})
		return
	}

	// Call WeeklyreportByID with the report ID
	report, err := models.WeeklyreportByID(c.Request.Context(), db, id)
	if err == nil {
		zdata = []*models.WeeklyReportExtended{report}
	} else {
		log.Printf("Error retrieving report data: %v", err)
		zdata = nil
	}

	log.Printf("Session Data HandlerReportFacilities: %+v", report)

	// Pass retrieved report data to the HTML template
	data := Get_Session_Data(c, db, sessionManager, zdata)
	utilities.GenerateHTML(c, data, "base", "viewreport2")
}

func HandlerReportView4(c *gin.Context, db *sql.DB, sessionManager *scs.SessionManager) {
	// Get session-specific data (e.g., user ID and HFID)
	sessionData, ok := Get_Session_Data(c, db, sessionManager, nil).(utilities.TemplateData)
	if !ok {
		log.Println("Failed to retrieve session data as TemplateData")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Unable to retrieve session data"})
		return
	}

	// Now access the SessionDetails from the TemplateData
	sesDetails, ok := sessionData.Ses.(utilities.SessionDetails)
	if !ok {
		log.Println("Failed to retrieve session details from TemplateData")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Unable to retrieve session details"})
		return
	}

	//log.Printf("Session Data HandlerReportFacilities: %+v", sesDetails)

	// Parse empID from URL
	empIDStr := c.Query("empID")
	log.Printf("Employee ID value from session: %s", empIDStr)

	facilityID := sesDetails.HFID // Retrieve HFID from the session
	log.Printf("Hospital ID value from session: %d", facilityID)

	// Check if departmentID is provided
	departmentID := c.Query("department") // Capture departmentID from the query

	rptID := c.Param("id")
	reportID, err := strconv.Atoi(rptID)

	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid department ID"})
		return
	}
	log.Printf("Report ID from URL %d", reportID)
	log.Printf("Department ID from URL %s", departmentID)

	if departmentID != "" {
		// Dynamic loading logic for department-specific data
		deptID, err := strconv.Atoi(departmentID)
		log.Printf("Dept Id from HandlerReportView %d", deptID)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid department ID"})
			return
		}

		// Fetch staff and data points for the department
		/*staff, dataPoints, err := models.GetReportAndDataPointsByDepartment(db, reportID, deptID)
		if err != nil {
			log.Printf("Error fetching staff and data points: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch data"})
			return
		}

		// Return the data in JSON format for AJAX
		c.JSON(http.StatusOK, gin.H{"staff": staff, "dataPoints": dataPoints})
		return*/
	}

	/*
		log.Printf("Executing GetFacilitiesAndDepartments fxn from HandlerReportView2")
		// Fetch facilities and departments for the dropdowns
		facilities, departments, err := models.GetFacilitiesAndDepartments(db)
		if err != nil {
			log.Printf("Error loading facilities or departments: %v", err)
			c.String(http.StatusInternalServerError, "Error loading facilities or departments")
			return
		}*/

	// Prepare data to pass to the template
	data := gin.H{
		"sessionData": sessionData,
		//"facilities":  facilities,
		//"departments": departments,
		"zdata": nil, // Default empty data on initial load
	}

	// Render the template
	utilities.GenerateHTML(c, data, "base", "viewreport2")
}

/*func HandlerReportView3(c *gin.Context, db *sql.DB, sessionManager *scs.SessionManager) {
	sessionData, ok := Get_Session_Data(c, db, sessionManager, nil).(utilities.TemplateData)
	if !ok {
		log.Println("Failed to retrieve session data")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Session error"})
		return
	}

	departmentID := c.Query("departmentID")
	if departmentID != "" {
		deptID, err := strconv.Atoi(departmentID)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid department ID"})
			return
		}

		reportID, err := strconv.Atoi(c.Query("reportID")) // Add report ID if applicable
		if err != nil {
			reportID = 0 // Default to 0 if not provided
		}

		staff, dataPoints, err := models.GetReportAndDataPointsByDepartment(db, reportID, deptID)
		if err != nil {
			log.Printf("Error fetching data: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Data fetch error"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"staff": staff, "dataPoints": dataPoints})
		return
	}

	facilities, departments, err := models.GetFacilitiesAndDepartments(db)
	if err != nil {
		log.Printf("Error fetching dropdown data: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Dropdown fetch error"})
		return
	}

	data := gin.H{
		"sessionData": sessionData,
		"facilities":  facilities,
		"departments": departments,
	}

	utilities.GenerateHTML(c, data, "base", "viewreport2")
}*/

func HandlerReportRem(c *gin.Context, db *sql.DB, sessionManager *scs.SessionManager) {

}

func HandlerReportExport(c *gin.Context, db *sql.DB, sessionManager *scs.SessionManager) {

}

// Handler to update edited entries in the bulk capture report submission list
func HandlerReportEntryUpdate(c *gin.Context, db *sql.DB, sessionManager *scs.SessionManager) {
	// Parse the start and stop dates from the form
	// start := c.PostForm("start")
	// stop := c.PostForm("stop")

	// Retrieve individual form fields into a slice of WeeklyReportExtended
	var reports []models.WeeklyReportExtended

	// Assume you have a list of employee IDs to loop through
	for empID := range c.PostFormMap("input") {
		// Fetch the report ID for this employee directly from the form
		reportIDStr := c.PostForm(fmt.Sprintf("input[%s][reportID]", empID))
		reportID := parseInt(reportIDStr) // Parse reportID from string to int64

		log.Printf("Update ReportID: %d", reportID)

		// Create a new WeeklyReportExtended instance and populate fields from the form
		weeklyReport := models.WeeklyReportExtended{
			ID: int(reportID), // Use report ID
			// Start:          sql.NullTime{Time: parseDate(start), Valid: true},
			// Stop:           sql.NullTime{Time: parseDate(stop), Valid: true},
			Emp:          sql.NullInt64{Int64: empIDToInt(empID), Valid: true},
			Qn01:         sql.NullInt64{Int64: parseInt(c.PostForm("input[" + empID + "][attendance]")), Valid: true},
			Qn02:         sql.NullInt64{Int64: parseInt(c.PostForm("input[" + empID + "][ward_rounds]")), Valid: true},
			Qn03:         sql.NullInt64{Int64: parseInt(c.PostForm("input[" + empID + "][patients_reviewed]")), Valid: true},
			Qn04:         sql.NullInt64{Int64: parseInt(c.PostForm("input[" + empID + "][theatre_days]")), Valid: true},
			Qn05:         sql.NullInt64{Int64: parseInt(c.PostForm("input[" + empID + "][elective]")), Valid: true},
			Qn06:         sql.NullInt64{Int64: parseInt(c.PostForm("input[" + empID + "][emergency]")), Valid: true},
			Qn07:         sql.NullInt64{Int64: parseInt(c.PostForm("input[" + empID + "][postmortems]")), Valid: true},
			Qn08:         sql.NullInt64{Int64: parseInt(c.PostForm("input[" + empID + "][OPD_clinics]")), Valid: true},
			Qn09:         sql.NullInt64{Int64: parseInt(c.PostForm("input[" + empID + "][OPD_patients]")), Valid: true},
			Qn10:         sql.NullInt64{Int64: parseInt(c.PostForm("input[" + empID + "][anc_patients]")), Valid: true},
			Qn11:         sql.NullInt64{Int64: parseInt(c.PostForm("input[" + empID + "][teaching_rounds]")), Valid: true},
			Qn12:         sql.NullInt64{Int64: parseInt(c.PostForm("input[" + empID + "][students_taught]")), Valid: true},
			Qn13:         sql.NullInt64{Int64: parseInt(c.PostForm("input[" + empID + "][mortality_reviews]")), Valid: true},
			Qn14:         sql.NullInt64{Int64: parseInt(c.PostForm("input[" + empID + "][maternal]")), Valid: true},
			Qn15:         sql.NullInt64{Int64: parseInt(c.PostForm("input[" + empID + "][perinatal]")), Valid: true},
			Qn16:         sql.NullInt64{Int64: parseInt(c.PostForm("input[" + empID + "][surgical]")), Valid: true},
			Qn17:         sql.NullInt64{Int64: parseInt(c.PostForm("input[" + empID + "][medical]")), Valid: true},
			Qn18:         sql.NullInt64{Int64: parseInt(c.PostForm("input[" + empID + "][pead]")), Valid: true},
			Qn19:         sql.NullInt64{Int64: parseInt(c.PostForm("input[" + empID + "][labs_requests]")), Valid: true},
			Qn20:         sql.NullInt64{Int64: parseInt(c.PostForm("input[" + empID + "][imaging_requests]")), Valid: true},
			Qn21:         sql.NullInt64{Int64: parseInt(c.PostForm("input[" + empID + "][investigations]")), Valid: true},
			Qn22:         sql.NullInt64{Int64: parseInt(c.PostForm("input[" + empID + "][BS]")), Valid: true},
			Qn23:         sql.NullInt64{Int64: parseInt(c.PostForm("input[" + empID + "][HIV]")), Valid: true},
			Qn24:         sql.NullInt64{Int64: parseInt(c.PostForm("input[" + empID + "][malaria]")), Valid: true},
			Qn25:         sql.NullInt64{Int64: parseInt(c.PostForm("input[" + empID + "][TB]")), Valid: true},
			Qn26:         sql.NullInt64{Int64: parseInt(c.PostForm("input[" + empID + "][CBC]")), Valid: true},
			Qn27:         sql.NullInt64{Int64: parseInt(c.PostForm("input[" + empID + "][chemistry]")), Valid: true},
			Qn28:         sql.NullInt64{Int64: parseInt(c.PostForm("input[" + empID + "][hematology]")), Valid: true},
			Qn29:         sql.NullInt64{Int64: parseInt(c.PostForm("input[" + empID + "][urinalysis]")), Valid: true},
			Qn30:         sql.NullInt64{Int64: parseInt(c.PostForm("input[" + empID + "][gram_stain]")), Valid: true},
			Qn31:         sql.NullInt64{Int64: parseInt(c.PostForm("input[" + empID + "][culture]")), Valid: true},
			Qn32:         sql.NullInt64{Int64: parseInt(c.PostForm("input[" + empID + "][microbiology]")), Valid: true},
			Qn33:         sql.NullInt64{Int64: parseInt(c.PostForm("input[" + empID + "][sensitivity_tests]")), Valid: true},
			Qn34:         sql.NullInt64{Int64: parseInt(c.PostForm("input[" + empID + "][diagnostics]")), Valid: true},
			Qn35:         sql.NullInt64{Int64: parseInt(c.PostForm("input[" + empID + "][xrays]")), Valid: true},
			Qn36:         sql.NullInt64{Int64: parseInt(c.PostForm("input[" + empID + "][ct_scans]")), Valid: true},
			Qn37:         sql.NullInt64{Int64: parseInt(c.PostForm("input[" + empID + "][obstetrics_scans]")), Valid: true},
			Qn38:         sql.NullInt64{Int64: parseInt(c.PostForm("input[" + empID + "][abdominal_scans]")), Valid: true},
			LastUpdateOn: sql.NullTime{Time: time.Now(), Valid: true}, // Set updated time
		}

		// Append the report to the slice
		reports = append(reports, weeklyReport)

		log.Printf("Updatez reports: %v", reports)
	}

	// Update each report in the database
	// Update each report in the database
	for _, report := range reports {

		if err := report.Updatez(c.Request.Context(), db); err != nil {
			// Handle error (you might want to log it and return a proper response)
			log.Printf("Error updating report for employee %d: %v", report.ID, err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update report"})
			return
		}
	}

	// Reload the same URL with the original query parameters
	redirectURL := c.Request.Referer()
	if redirectURL == "" {
		// Fallback to a default URL if referer is not available
		redirectURL = "/reports/list"
	}

	// Redirect back to the referring URL to refresh the page
	c.Redirect(http.StatusFound, redirectURL)

	// Return a success response
	// c.JSON(http.StatusOK, gin.H{"message": "Reports updated successfully"})
}

// Handler to approve weekly report entries.
func HandlerReportApprove(c *gin.Context, db *sql.DB, sessionManager *scs.SessionManager) {
	// Define a structure to parse the incoming JSON payload
	/*var requestBody struct {
		ReportIDs []int `json:"reportIDs"`
	}*/

	// Get session-specific data (e.g., user ID and HFID)
	sessionData, ok := Get_Session_Data(c, db, sessionManager, nil).(utilities.TemplateData)
	if !ok {
		log.Println("Failed to retrieve session data as TemplateData")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Unable to retrieve session data"})
		return
	}

	// Now access the SessionDetails from the TemplateData
	sesDetails, ok := sessionData.Ses.(utilities.SessionDetails)
	if !ok {
		log.Println("Failed to retrieve session details from TemplateData")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Unable to retrieve session details"})
		return
	}

	submittedByID := sesDetails.EmpID // ID of person submitting report at department level
	approvedByID := sesDetails.EmpID  // ID of person approving report at facility level

	//log.Printf("Hospital ID value from session: %d", enteredByID)
	log.Printf("WE are here in HandlerReportApprove")
	var requestBody struct {
		ReportIDs    json.RawMessage `json:"reportIDs"`
		FacilityID   string          `json:"facility"`
		DepartmentID string          `json:"department"`
		Start        string          `json:"start"`
	}

	//log.Printf("Debug - Facility: %s, Dept: %s, start : %s", facilityID, departmentID, start)

	// Parse JSON body into requestBody
	if err := c.ShouldBindJSON(&requestBody); err != nil {
		log.Printf("Error parsing request body: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format"})
		return
	}

	// Process ReportIDs: Handle both object (`{}`) and array (`[]int`)
	var reportIDs []int
	if len(requestBody.ReportIDs) > 0 {
		if err := json.Unmarshal(requestBody.ReportIDs, &reportIDs); err != nil {
			// Handle case where it's not an array (e.g., `{}`)
			log.Printf("ReportIDs is not an array, attempting fallback: %v", err)

			// Fallback: Unmarshal as a generic map if it's an object
			var reportIDsMap map[string]interface{}
			if err := json.Unmarshal(requestBody.ReportIDs, &reportIDsMap); err != nil {
				log.Printf("Error parsing reportIDs as map: %v", err)
				c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid reportIDs format"})
				return
			}

			// Optionally, process reportIDsMap to extract specific values if needed
			log.Printf("Parsed reportIDs as map: %v", reportIDsMap)
			reportIDs = []int{} // Treat empty object `{}` as an empty array
		}
	}

	// Debugging
	//log.Printf("Debug - Facility: %s, Dept: %s, Start: %s, ReportIDs: %v", requestBody.FacilityID, requestBody.DepartmentID, requestBody.Start, requestBody.ReportIDs)

	flag := 0

	if c.FullPath() == "/reports/approve" {
		log.Println("Handler called for facilities submissions")
		flag = 1
		// Call ApproveAll to update all entries with the provided IDs
		if err := models.ApproveSubmitReport(db, flag, int(approvedByID), requestBody.FacilityID, requestBody.DepartmentID, requestBody.Start, reportIDs); err != nil {
			log.Printf("Error updating reports: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to approve reports"})
			return
		}
	} else if c.FullPath() == "/reports/submit" {
		log.Println("Handler called for departments submissions")
		flag = 2

		// Call ApproveAll to update all entries with the provided IDs
		if err := models.ApproveSubmitReport(db, flag, int(submittedByID), requestBody.FacilityID, requestBody.DepartmentID, requestBody.Start, reportIDs); err != nil {
			log.Printf("Error updating reports: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to approve reports"})
			return
		}
	} else {
		log.Println("Invalid endpoint")
		c.JSON(http.StatusNotFound, gin.H{"error": "Invalid endpoint"})
		return
	}

	// Call ApproveAll to update all entries with the provided IDs
	/*if err := models.ApproveSubmitReport(db, requestBody.ReportIDs, flag); err != nil {
		log.Printf("Error updating reports: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to approve reports"})
		return
	}*/

	// Respond with success
	c.JSON(http.StatusOK, gin.H{"message": "All reports approved successfully"})
}

// JSON Data Handler
/*func HandlerReportData(c *gin.Context, db *sql.DB, sessionManager *scs.SessionManager) {
	log.Printf("HandlerReportData fxn invoked")
	reportIDStr := c.Query("id")
	departmentIDStr := c.Query("department")

	reportID, err := strconv.Atoi(reportIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid report ID"})
		return
	}

	deptID, err := strconv.Atoi(departmentIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid department ID"})
		return
	}

	log.Printf("HandlerReportData fxn reportid %d, deptid %d", reportID, deptID)

	// Fetch data points and staff
	staff, dataPoints, err := models.GetReportAndDataPointsByDepartment(db, reportID, deptID)
	if err != nil {
		log.Printf("Error fetching data: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch data"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"staff": staff, "dataPoints": dataPoints})
}*/

func HandlerReportData2(c *gin.Context, db *sql.DB, sessionManager *scs.SessionManager) {
	log.Println("HandlerReportData function invoked")

	facilityIDStr := c.Query("facility")
	departmentIDStr := c.Query("department")
	startDate := c.Query("start")

	log.Printf("For the json %s, %s, %s", facilityIDStr, departmentIDStr, startDate)

	facilityID, err := strconv.Atoi(facilityIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid report ID"})
		return
	}

	departmentID, err := strconv.Atoi(departmentIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid department ID"})
		return
	}

	//log.Printf("Fetching report data for Report ID: %d, Department ID: %d", reportID, departmentID)

	// Fetch data points and staff for the report and department
	staffData, dataPoints, err := models.GetReportAndDataPointsByDepartment(db, startDate, facilityID, departmentID)
	if err != nil {
		log.Printf("Error fetching data: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch data"})
		return
	}

	//log.Printf("Staff: %v \nData points: %v", staffData, dataPoints)

	// Transform the data points into their named data-element keys.
	transformedData := make([]map[string]interface{}, len(staffData))
	for i, staff := range staffData {
		transformedEntry := make(map[string]interface{})
		transformedEntry["staffname"] = staff.Fname
		transformedEntry["title"] = staff.EmpTitle
		transformedEntry["start"] = staff.Start
		transformedEntry["attendance"] = staff.Qn01.Int64
		transformedEntry["ward_rounds"] = staff.Qn02.Int64
		transformedEntry["patients_reviewed"] = staff.Qn03.Int64
		transformedEntry["theatre_days"] = staff.Qn04.Int64
		transformedEntry["elective"] = staff.Qn05.Int64
		transformedEntry["emergency"] = staff.Qn06.Int64
		transformedEntry["postmortems"] = staff.Qn07.Int64
		transformedEntry["OPD_clinics"] = staff.Qn08.Int64
		transformedEntry["OPD_patients"] = staff.Qn09.Int64
		transformedEntry["anc_patients"] = staff.Qn10.Int64
		transformedEntry["teaching_rounds"] = staff.Qn11.Int64
		transformedEntry["students_taught"] = staff.Qn12.Int64
		transformedEntry["mortality_reviews"] = staff.Qn13.Int64
		transformedEntry["maternal"] = staff.Qn14.Int64
		transformedEntry["perinatal"] = staff.Qn15.Int64
		transformedEntry["surgical"] = staff.Qn16.Int64
		transformedEntry["medical"] = staff.Qn17.Int64
		transformedEntry["pead"] = staff.Qn18.Int64
		transformedEntry["labs_requests"] = staff.Qn19.Int64
		transformedEntry["imaging_requests"] = staff.Qn20.Int64
		transformedEntry["investigations"] = staff.Qn21.Int64
		transformedEntry["BS"] = staff.Qn22.Int64
		transformedEntry["HIV"] = staff.Qn23.Int64
		transformedEntry["malaria"] = staff.Qn24.Int64
		transformedEntry["TB"] = staff.Qn25.Int64
		transformedEntry["CBC"] = staff.Qn26.Int64
		transformedEntry["chemistry"] = staff.Qn27.Int64
		transformedEntry["hematology"] = staff.Qn28.Int64
		transformedEntry["urinalysis"] = staff.Qn29.Int64
		transformedEntry["gram_stain"] = staff.Qn30.Int64
		transformedEntry["culture"] = staff.Qn31.Int64
		transformedEntry["microbiology"] = staff.Qn32.Int64
		transformedEntry["sensitivity_tests"] = staff.Qn33.Int64
		transformedEntry["diagnostics"] = staff.Qn34.Int64
		transformedEntry["xrays"] = staff.Qn35.Int64
		transformedEntry["ct_scans"] = staff.Qn36.Int64
		transformedEntry["obstetrics_scans"] = staff.Qn37.Int64
		transformedEntry["abdominal_scans"] = staff.Qn38.Int64

		// Add additional fields from staff to transformedEntry if needed
		transformedEntry["EmpID"] = staff.EmpID
		transformedEntry["ID"] = staff.ID

		transformedData[i] = transformedEntry
	}

	// Send the transformed data to the client
	c.JSON(http.StatusOK, gin.H{
		"staff":      transformedData,
		"dataPoints": dataPoints,
	})
}

// Template Rendering Handler
func HandlerReportView2(c *gin.Context, db *sql.DB, sessionManager *scs.SessionManager) {
	//log.Printf("HandlerReportView2 invoked.")
	sessionData, ok := Get_Session_Data(c, db, sessionManager, nil).(utilities.TemplateData)
	if !ok {
		log.Println("Failed to retrieve session data as TemplateData")
		c.String(http.StatusInternalServerError, "Session error")
		return
	}

	data := gin.H{
		"sessionData": sessionData,
	}

	utilities.GenerateHTML(c, data, "base", "viewreport2")
}

// Handler used for entering weekly report for a single user
func LegacyHandlerReportsAnalysis(c *gin.Context, db *sql.DB, sessionManager *scs.SessionManager) {

	// Get session-specific data (e.g., user ID and HFID)
	sessionData, ok := Get_Session_Data(c, db, sessionManager, nil).(utilities.TemplateData)
	if !ok {
		log.Println("Failed to retrieve session data as TemplateData")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Unable to retrieve session data"})
		return
	}

	// Now access the SessionDetails from the TemplateData
	sesDetails, ok := sessionData.Ses.(utilities.SessionDetails)
	if !ok {
		log.Println("Failed to retrieve session details from TemplateData")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Unable to retrieve session details"})
		return
	}

	//log.Printf("Session Data HandlerReportFacilities: %+v", sesDetails)

	hospitalID := sesDetails.HFID // Retrieve HFID from the session

	// Call GetFacilitiesList function to retrieve facilities
	facilities, err := models.GetFacilitiesList(c.Request.Context(), db, hospitalID)
	if err != nil {
		log.Printf("Error fetching facilities: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error fetching facilities"})
		return
	}

	// Log the facilities data in the debug console
	//log.Printf("Fetched facilities: %+v", facilities)

	// Get session data from the database or session manager
	data := Get_Session_Data(c, db, sessionManager, facilities)

	// Render the HTML using the utility function
	utilities.GenerateHTML(c, data, "base", "report-analysis")
}

// Handler to display summary report in an HTML table
func HandlerReportSummaries(c *gin.Context, db *sql.DB, sessionManager *scs.SessionManager) {
	log.Printf("HandlerReportsSummary executed")

	// Fetch facility from query parameters
	hfidStr := c.Query("hfid")

	hfid, err := strconv.Atoi(hfidStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid report ID"})
		return
	}

	// Fetch summarized report data from the database
	reports, err := models.GetWeeklyReportSummary(c.Request.Context(), db, hfid)
	if err != nil {
		log.Printf("Error fetching summary data: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error fetching report summary"})
		return
	}

	//log.Printf("Weekly Report Summary Data: %+v", reports)

	// Prepare data for rendering
	data := Get_Session_Data(c, db, sessionManager, reports)

	// Render HTML using the utility function
	utilities.GenerateHTML(c, data, "base", "summary-table")

}
