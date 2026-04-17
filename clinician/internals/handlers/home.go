package handlers

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"sort"
	"strconv"
	"time"

	"github.com/alexedwards/scs/v2"
	"github.com/gin-gonic/gin"
	"github.com/moh/clinician/internals/models"
	"github.com/moh/clinician/internals/security"
	"github.com/moh/clinician/internals/utilities"
)

type DashboardKPIView struct {
	Title string
	Value int
	Unit  string
	Meta  string
	Link  string
}

type HomeViewModel struct {
	Role                string
	ScopeTitle          string
	ScopeSubtitle       string
	SelectedFacility    int
	FacilityOptions     []models.DashboardFilterOption
	SelectedDepartment  int
	DepartmentOptions   []models.DashboardFilterOption
	OfficerName         string
	OfficerID           int
	OfficerTitle        string
	OfficerFacility     string
	OfficerDepartment   string
	SelectedYear        int
	SelectedWeek        int
	SelectedWeekLabel   string
	AvailableYears      []int
	AvailableWeeks      []models.ClinicianWeekOption
	Stats               []map[string]interface{}
	KPIs                []DashboardKPIView
	SeriesTitle         string
	SeriesLabels        []string
	SeriesA             []int
	SeriesALabel        string
	SeriesB             []int
	SeriesBLabel        string
	PerformanceRows     []models.FacilityPerformanceRow
	DepartmentRows      []models.DepartmentProgressRow
	DepartmentAnalysis  []models.DepartmentAnalysisRow
	ClinicianAnalysis   []models.ClinicianAnalysisRow
	TopFacilities       []models.FacilityPerformanceRow
	AttentionFacilities []models.FacilityPerformanceRow
	PriorityDepartments []models.DepartmentAnalysisRow
	PriorityClinicians  []models.ClinicianAnalysisRow
	DataEntryURL        string
	LatestSubmission    string
	ReportHistoryURL    string
}

func HandlerHome2(c *gin.Context, db *sql.DB, sessionManager *scs.SessionManager) {

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

	// Retrieve HFID from session
	hospitalID := sesDetails.HFID
	log.Printf("Hospital ID value from session: %d", hospitalID)

	data := Get_Session_Data(c, db, sessionManager, nil)

	log.Printf("data variable passed in HandlerHome: %v", data)

	fmt.Println("home page loading with data")
	utilities.GenerateHTML(c, data, "base", "home")

}

func HandlerHome(c *gin.Context, db *sql.DB, sessionManager *scs.SessionManager) {
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

	log.Printf("Session Data HandlerHome: %+v", sesDetails)

	viewModel, err := buildHomeViewModel(c, db, sesDetails)
	if err != nil {
		log.Printf("Error fetching dashboard data: %v", err)
		c.String(http.StatusInternalServerError, "Error fetching dashboard data")
		return
	}

	sessionData.Form = viewModel
	log.Printf("data variable passed in HandlerHome: %v", sessionData)

	fmt.Println("home page loading with data")
	utilities.GenerateHTML(c, sessionData, "base", "home")

}

func HandlerLoginOut(c *gin.Context, db *sql.DB, sessionManager *scs.SessionManager) {
	// Clear the session to log the user out
	sessionManager.Destroy(c)

	// Optionally, you can set a flash message for feedback
	sessionManager.Put(c, "flash", "You have been logged out successfully.")

	// Redirect the user to the home page or a login page
	c.Redirect(302, "/login") // Change "/" to the appropriate route for your application
}

func GetPageData(f string) utilities.TemplateData {
	data := utilities.TemplateData{
		CurrentYear:     time.Now().Year(),
		Form:            nil,                            // or your form data
		Items:           []interface{}{},                // or your items
		Optionz:         map[string]map[string]string{}, // or your options
		Flash:           f,
		IsAuthenticated: true,              // or false based on your logic
		CSRFToken:       "your-csrf-token", // generate or retrieve CSRF token
	}
	return data
}

func SES_SET(c *gin.Context, db *sql.DB, sessionManager *scs.SessionManager) (utilities.SessionDetails, error) {

	id := security.GetCurrentUser(c, sessionManager)
	ses := utilities.SessionDetails{}

	if id == 0 {
		return ses, fmt.Errorf("no active user found")
	}

	user, err := models.UserByID(c, db, id)
	if err != nil {
		return ses, err
	}

	ses.Email = user.Username
	ses.EmpID = user.Employees.Int64
	ses.Rights = user.Rights
	ses.UserID = int64(user.ID)

	emp, err := models.EmployeeByID(c, db, int(user.Employees.Int64))
	if err != nil {
		return ses, err
	}

	ses.FName = emp.Fname.String
	ses.LName = emp.Lname.String

	hf, err := models.FacilityByID(c, db, int(emp.EmpFacility))
	if err != nil {
		return ses, err
	}

	ses.HFName = hf.FacilityName
	ses.HFID = int64(hf.FacilityID)

	hd, err := models.DepartmentByID(c, db, int(emp.EmpDepartment))
	if err != nil {
		return ses, err
	}

	ses.HDID = int64(hd.DeptID)

	return ses, nil

}

func Get_Session_Data(c *gin.Context, db *sql.DB, sessionManager *scs.SessionManager, pData any) any {
	id := security.GetCurrentUser(c, sessionManager)

	if id == 0 {
		return nil
	} else {

		data := GetPageData("your flash message")

		ses, er := SES_SET(c, db, sessionManager)
		if er != nil {
			log.Printf("failed to build session data: %v", er)
			data.Ses = utilities.SessionDetails{}
		} else {
			data.Ses = ses
		}

		data.Form = pData

		return data
	}
}

func buildHomeViewModel(c *gin.Context, db *sql.DB, ses utilities.SessionDetails) (HomeViewModel, error) {
	selectedFacility, _ := strconv.Atoi(c.Query("facility"))
	selectedDepartment, _ := strconv.Atoi(c.Query("department"))

	switch ses.Rights {
	case "admin":
		return buildNationalHome(c, db, selectedFacility, selectedDepartment)
	case "approver":
		return buildFacilityManagerHome(c, db, int(ses.HFID), ses.HFName, selectedDepartment)
	default:
		return buildClinicianHome(c, db, int(ses.EmpID), ses.HFName)
	}
}

func buildNationalHome(c *gin.Context, db *sql.DB, selectedFacilityID int, selectedDepartmentID int) (HomeViewModel, error) {
	snapshot, err := models.GetNationalDashboardSnapshot(c.Request.Context(), db)
	if err != nil {
		return HomeViewModel{}, err
	}

	view := HomeViewModel{
		Role:               "admin",
		ScopeTitle:         "National Performance Overview",
		ScopeSubtitle:      "Monitor facility participation, submissions, and approval follow-up across the referral network. Select a facility to drill down into department and clinician performance.",
		SeriesTitle:        "Weekly Facility Submission Snapshot",
		SeriesALabel:       "Entered",
		SeriesBLabel:       "Submitted",
		PerformanceRows:    snapshot.FacilityPerformance,
		SelectedFacility:   selectedFacilityID,
		SelectedDepartment: selectedDepartmentID,
	}

	for _, row := range snapshot.FacilityPerformance {
		view.FacilityOptions = append(view.FacilityOptions, models.DashboardFilterOption{
			ID:   row.FacilityID,
			Name: row.FacilityName,
		})
	}

	filteredRows := snapshot.FacilityPerformance
	if selectedFacilityID > 0 {
		for _, row := range snapshot.FacilityPerformance {
			if row.FacilityID != selectedFacilityID {
				continue
			}

			filteredRows = []models.FacilityPerformanceRow{row}
			view.ScopeTitle = fmt.Sprintf("%s Performance Overview", row.FacilityName)
			view.ScopeSubtitle = "National administrator view filtered to a single referral facility."
			view.Stats = []map[string]interface{}{
				{"Title": "Selected Facility", "Value": row.FacilityName, "Tone": "stat-primary"},
				{"Title": "Reporting Rate", "Value": fmt.Sprintf("%d%%", row.ReportingRate), "Tone": "stat-success"},
				{"Title": "Pending Approval", "Value": row.PendingApproval, "Tone": "stat-warning"},
				{"Title": "Missing Submissions", "Value": row.PendingThisWeek, "Tone": "stat-info"},
			}
			break
		}
	}

	if selectedFacilityID == 0 {
		view.SelectedFacility = 0
		view.Stats = []map[string]interface{}{
			{"Title": "Referral Facilities", "Value": snapshot.TotalFacilities, "Tone": "stat-primary"},
			{"Title": "Clinicians Tracked", "Value": snapshot.TotalClinicians, "Tone": "stat-success"},
			{"Title": "Reports Entered This Week", "Value": snapshot.ReportsEnteredThisWeek, "Tone": "stat-info"},
			{"Title": "Pending Approval", "Value": snapshot.PendingApproval, "Tone": "stat-warning"},
		}
		view.TopFacilities = topFacilityPerformers(snapshot.FacilityPerformance, 3)
		view.AttentionFacilities = facilitiesNeedingSupport(snapshot.FacilityPerformance, 3)
	}

	view.PerformanceRows = filteredRows

	for _, row := range filteredRows {
		view.SeriesLabels = append(view.SeriesLabels, row.FacilityName)
		view.SeriesA = append(view.SeriesA, row.EnteredThisWeek)
		view.SeriesB = append(view.SeriesB, row.SubmittedThisWeek)
	}

	if selectedFacilityID > 0 {
		departmentAnalysis, err := models.GetFacilityDepartmentAnalysis(c.Request.Context(), db, selectedFacilityID)
		if err != nil {
			return HomeViewModel{}, err
		}

		for _, row := range departmentAnalysis {
			view.DepartmentOptions = append(view.DepartmentOptions, models.DashboardFilterOption{
				ID:   row.DepartmentID,
				Name: row.DepartmentName,
			})
		}

		if selectedDepartmentID > 0 {
			filteredDepartments := make([]models.DepartmentAnalysisRow, 0, len(departmentAnalysis))
			for _, row := range departmentAnalysis {
				if row.DepartmentID == selectedDepartmentID {
					filteredDepartments = append(filteredDepartments, row)
					view.ScopeTitle = fmt.Sprintf("%s: %s", view.ScopeTitle, row.DepartmentName)
					view.ScopeSubtitle = "National administrator drill-down focused on a single department inside the selected facility."
				}
			}
			if len(filteredDepartments) > 0 {
				departmentAnalysis = filteredDepartments
			} else {
				view.SelectedDepartment = 0
			}
		}

		view.DepartmentAnalysis = departmentAnalysis

		clinicianAnalysis, err := models.GetFacilityClinicianAnalysis(c.Request.Context(), db, selectedFacilityID, view.SelectedDepartment)
		if err != nil {
			return HomeViewModel{}, err
		}
		view.ClinicianAnalysis = clinicianAnalysis

		if len(view.DepartmentAnalysis) > 0 {
			summary := summarizeDepartmentAnalysis(view.DepartmentAnalysis)
			scopeName := "Selected Facility"
			scopeValue := view.ScopeTitle
			if selectedDepartmentID > 0 && len(view.DepartmentAnalysis) == 1 {
				scopeName = "Selected Department"
				scopeValue = view.DepartmentAnalysis[0].DepartmentName
			}
			view.Stats = []map[string]interface{}{
				{"Title": scopeName, "Value": scopeValue, "Tone": "stat-primary"},
				{"Title": "Reporting Rate", "Value": fmt.Sprintf("%d%%", percentage(summary.Submitted, summary.Clinicians)), "Tone": "stat-success"},
				{"Title": "Ward Rounds", "Value": summary.WardRounds, "Tone": "stat-info"},
				{"Title": "Patients Reviewed", "Value": summary.PatientsReviewed, "Tone": "stat-info"},
			}
		}
		view.PriorityDepartments = priorityDepartments(view.DepartmentAnalysis, 3)
		view.PriorityClinicians = priorityClinicians(view.ClinicianAnalysis, 5)
	}

	return view, nil
}

func buildFacilityManagerHome(c *gin.Context, db *sql.DB, facilityID int, facilityName string, selectedDepartmentID int) (HomeViewModel, error) {
	snapshot, err := models.GetFacilityManagerDashboardSnapshot(c.Request.Context(), db, facilityID)
	if err != nil {
		return HomeViewModel{}, err
	}

	reportSummary, err := models.GetFacilityReportReviewSummary(c.Request.Context(), db, int64(facilityID))
	if err != nil {
		return HomeViewModel{}, err
	}

	leaveSummary, err := models.GetFacilityLeaveReviewSummary(c.Request.Context(), db, int64(facilityID))
	if err != nil {
		return HomeViewModel{}, err
	}

	view := HomeViewModel{
		Role:               "approver",
		ScopeTitle:         fmt.Sprintf("%s Follow-up Dashboard", facilityName),
		ScopeSubtitle:      "Review submitted reports and leave requests across your facility, then act on anything still pending.",
		SeriesTitle:        "Department Submission Status",
		SeriesALabel:       "Entered",
		SeriesBLabel:       "Submitted",
		DepartmentRows:     snapshot.Departments,
		SelectedDepartment: selectedDepartmentID,
	}

	for _, row := range snapshot.Departments {
		view.DepartmentOptions = append(view.DepartmentOptions, models.DashboardFilterOption{
			ID:   row.DepartmentID,
			Name: row.DepartmentName,
		})
	}

	filteredRows := snapshot.Departments
	if selectedDepartmentID > 0 {
		for _, row := range snapshot.Departments {
			if row.DepartmentID != selectedDepartmentID {
				continue
			}

			filteredRows = []models.DepartmentProgressRow{row}
			view.ScopeTitle = fmt.Sprintf("%s: %s", facilityName, row.DepartmentName)
			view.ScopeSubtitle = "Facility administrator view filtered to a single department."
			view.Stats = []map[string]interface{}{
				{"Title": "Selected Department", "Value": row.DepartmentName, "Tone": "stat-primary"},
				{"Title": "Clinicians in Department", "Value": row.Clinicians, "Tone": "stat-primary"},
				{"Title": "Entered This Week", "Value": row.EnteredThisWeek, "Tone": "stat-info"},
				{"Title": "Pending Follow-up", "Value": row.PendingThisWeek, "Tone": "stat-warning"},
			}
			break
		}
	}

	if len(view.Stats) == 0 {
		view.SelectedDepartment = 0
		view.Stats = []map[string]interface{}{
			{"Title": "Clinicians in Facility", "Value": snapshot.TotalClinicians, "Tone": "stat-primary"},
			{"Title": "Reports Entered This Week", "Value": snapshot.ReportsEnteredThisWeek, "Tone": "stat-info"},
			{"Title": "Facility Review Queue", "Value": reportSummary.PendingReports + leaveSummary.PendingLeaves, "Tone": "stat-warning"},
		}
	}

	currentYear, currentWeek := time.Now().ISOWeek()
	view.KPIs = []DashboardKPIView{
		{
			Title: "Report Submissions Pending Approval",
			Value: reportSummary.PendingReports,
			Unit:  "pending reports",
			Meta:  "Needs facility review",
			Link:  buildFacilityReportReviewURL("pending", 0, 0),
		},
		{
			Title: "Leave Requests Pending Approval",
			Value: leaveSummary.PendingLeaves,
			Unit:  "pending leave requests",
			Meta:  "Needs facility review",
			Link:  buildFacilityLeaveReviewURL("pending"),
		},
		{
			Title: "Approved Report Submissions",
			Value: reportSummary.ApprovedReports,
			Unit:  "approved reports",
			Meta:  "Facility history",
			Link:  buildFacilityReportReviewURL("approved", 0, 0),
		},
		{
			Title: "Declined Report Submissions",
			Value: reportSummary.DeclinedReports,
			Unit:  "declined reports",
			Meta:  "Facility history",
			Link:  buildFacilityReportReviewURL("declined", 0, 0),
		},
		{
			Title: "Approved Leave Requests",
			Value: leaveSummary.ApprovedLeaves,
			Unit:  "approved leave requests",
			Meta:  "Facility history",
			Link:  buildFacilityLeaveReviewURL("approved"),
		},
		{
			Title: "Declined Leave Requests",
			Value: leaveSummary.DeclinedLeaves,
			Unit:  "declined leave requests",
			Meta:  "Facility history",
			Link:  buildFacilityLeaveReviewURL("declined"),
		},
		{
			Title: "Reports Submitted This Week",
			Value: reportSummary.SubmittedThisWeek,
			Unit:  "submitted this week",
			Meta:  "Current reporting week",
			Link:  buildFacilityReportReviewURL("all", currentYear, currentWeek),
		},
	}

	view.DepartmentRows = filteredRows

	for _, row := range filteredRows {
		view.SeriesLabels = append(view.SeriesLabels, row.DepartmentName)
		view.SeriesA = append(view.SeriesA, row.EnteredThisWeek)
		view.SeriesB = append(view.SeriesB, row.SubmittedThisWeek)
	}

	departmentAnalysis, err := models.GetFacilityDepartmentAnalysis(c.Request.Context(), db, facilityID)
	if err != nil {
		return HomeViewModel{}, err
	}

	if selectedDepartmentID > 0 {
		filteredDepartmentAnalysis := make([]models.DepartmentAnalysisRow, 0, len(departmentAnalysis))
		for _, row := range departmentAnalysis {
			if row.DepartmentID == selectedDepartmentID {
				filteredDepartmentAnalysis = append(filteredDepartmentAnalysis, row)
				break
			}
		}
		if len(filteredDepartmentAnalysis) > 0 {
			departmentAnalysis = filteredDepartmentAnalysis
		}
	}
	view.DepartmentAnalysis = departmentAnalysis

	clinicianAnalysis, err := models.GetFacilityClinicianAnalysis(c.Request.Context(), db, facilityID, selectedDepartmentID)
	if err != nil {
		return HomeViewModel{}, err
	}
	view.ClinicianAnalysis = clinicianAnalysis

	if len(view.DepartmentAnalysis) > 0 {
		summary := summarizeDepartmentAnalysis(view.DepartmentAnalysis)
		scopeName := "Facility Reporting Rate"
		scopeValue := fmt.Sprintf("%d%%", percentage(summary.Submitted, summary.Clinicians))
		if selectedDepartmentID > 0 && len(view.DepartmentAnalysis) == 1 {
			scopeName = "Department Reporting Rate"
		}
		view.Stats = []map[string]interface{}{
			{"Title": scopeName, "Value": scopeValue, "Tone": "stat-success"},
			{"Title": "Total Ward Rounds", "Value": summary.WardRounds, "Tone": "stat-info"},
			{"Title": "Patients Reviewed", "Value": summary.PatientsReviewed, "Tone": "stat-info"},
			{"Title": "Procedures", "Value": summary.Procedures, "Tone": "stat-primary"},
		}
	}

	view.PriorityDepartments = priorityDepartments(view.DepartmentAnalysis, 3)
	view.PriorityClinicians = priorityClinicians(view.ClinicianAnalysis, 5)

	return view, nil
}

func buildClinicianHome(c *gin.Context, db *sql.DB, employeeID int, facilityName string) (HomeViewModel, error) {
	selectedYear, _ := strconv.Atoi(c.Query("year"))
	selectedWeek, _ := strconv.Atoi(c.Query("week"))

	snapshot, err := models.GetClinicianDashboardSnapshot(c.Request.Context(), db, employeeID, selectedYear, selectedWeek)
	if err != nil {
		return HomeViewModel{}, err
	}

	entryStatus := "Not entered"
	if snapshot.WeeksEntered > 0 {
		entryStatus = "Entered"
	}

	submissionStatus := "Not submitted"
	if snapshot.SelectedWeek == 0 {
		switch {
		case snapshot.WeeksEntered == 0:
			submissionStatus = "Not submitted"
		case snapshot.WeeksSubmitted == snapshot.WeeksEntered:
			submissionStatus = "Submitted"
		default:
			submissionStatus = "Partially submitted"
		}
	} else if snapshot.SelectedWeekSubmitted {
		submissionStatus = "Submitted"
	} else if snapshot.WeeksEntered > 0 {
		submissionStatus = "Saved as draft"
	}

	view := HomeViewModel{
		Role:              "user",
		ScopeTitle:        "My Weekly KPI Dashboard",
		ScopeSubtitle:     "Choose a reporting year and week to review your performance for that reporting period.",
		OfficerName:       snapshot.EmployeeName,
		OfficerID:         snapshot.EmployeeID,
		OfficerTitle:      snapshot.EmployeeTitle,
		OfficerFacility:   snapshot.FacilityName,
		OfficerDepartment: snapshot.DepartmentName,
		SelectedYear:      snapshot.SelectedYear,
		SelectedWeek:      snapshot.SelectedWeek,
		SelectedWeekLabel: snapshot.SelectedWeekLabel,
		AvailableYears:    snapshot.AvailableYears,
		AvailableWeeks:    snapshot.AvailableWeeks,
		DataEntryURL:      "/reports/new/0",
		SeriesTitle:       "My Weekly Trend",
		SeriesALabel:      "Patients Reviewed",
		SeriesBLabel:      "Procedures",
		ReportHistoryURL:  buildReportHistoryPeriodFilterQuery("all", snapshot.SelectedYear, snapshot.SelectedWeek),
	}

	if snapshot.SelectedYear > 0 {
		view.SeriesTitle = fmt.Sprintf("My %d Weekly Trend", snapshot.SelectedYear)
	} else {
		view.SeriesTitle = "My Weekly Trend Across All Records"
	}

	view.KPIs = []DashboardKPIView{
		{
			Title: "Total Reports",
			Value: snapshot.TotalReports,
			Unit:  "reports",
			Meta:  addReportPeriodSuffix(snapshot.SelectedWeekLabel),
			Link:  buildReportHistoryPeriodFilterQuery("all", snapshot.SelectedYear, snapshot.SelectedWeek),
		},
		{
			Title: "Submitted",
			Value: snapshot.SubmittedReports,
			Unit:  "reports",
			Meta:  addReportPeriodSuffix(snapshot.SelectedWeekLabel),
			Link:  buildReportHistoryPeriodFilterQuery("submitted", snapshot.SelectedYear, snapshot.SelectedWeek),
		},
		{
			Title: "Approved",
			Value: snapshot.ApprovedReports,
			Unit:  "reports",
			Meta:  addReportPeriodSuffix(snapshot.SelectedWeekLabel),
			Link:  buildReportHistoryPeriodFilterQuery("approved", snapshot.SelectedYear, snapshot.SelectedWeek),
		},
		{
			Title: "Declined",
			Value: snapshot.DeclinedReports,
			Unit:  "reports",
			Meta:  addReportPeriodSuffix(snapshot.SelectedWeekLabel),
			Link:  buildReportHistoryPeriodFilterQuery("declined", snapshot.SelectedYear, snapshot.SelectedWeek),
		},
	}

	view.Stats = []map[string]interface{}{
		{"Title": "Entry Status", "Value": entryStatus, "Tone": "stat-info"},
		{"Title": "Patients Reviewed", "Value": snapshot.PatientsReviewed, "Tone": "stat-info"},
		{"Title": "Submission Status", "Value": submissionStatus, "Tone": "stat-warning"},
	}

	for _, point := range snapshot.RecentWeeks {
		view.SeriesLabels = append(view.SeriesLabels, point.WeekLabel)
		view.SeriesA = append(view.SeriesA, point.PatientsReviewed)
		view.SeriesB = append(view.SeriesB, point.Procedures)
	}

	view.LatestSubmission = fmt.Sprintf("%s: %s", snapshot.SelectedWeekLabel, submissionStatus)

	_ = facilityName

	return view, nil
}

func cappedProgress(value, target int) int {
	if target <= 0 {
		return 0
	}

	progress := int(float64(value) / float64(target) * 100)
	if progress > 100 {
		return 100
	}
	if progress < 0 {
		return 0
	}
	return progress
}

type departmentAnalysisSummary struct {
	Clinicians       int
	Entered          int
	Submitted        int
	Attendance       int
	WardRounds       int
	PatientsReviewed int
	Procedures       int
}

func summarizeDepartmentAnalysis(rows []models.DepartmentAnalysisRow) departmentAnalysisSummary {
	var summary departmentAnalysisSummary
	for _, row := range rows {
		summary.Clinicians += row.Clinicians
		summary.Entered += row.ReportsEntered
		summary.Submitted += row.ReportsSubmitted
		summary.Attendance += row.AttendanceRecords
		summary.WardRounds += row.WardRounds
		summary.PatientsReviewed += row.PatientsReviewed
		summary.Procedures += row.Procedures
	}
	return summary
}

func percentage(numerator int, denominator int) int {
	if denominator <= 0 {
		return 0
	}

	value := int(float64(numerator) / float64(denominator) * 100)
	if value < 0 {
		return 0
	}
	if value > 100 {
		return 100
	}
	return value
}

func topFacilityPerformers(rows []models.FacilityPerformanceRow, limit int) []models.FacilityPerformanceRow {
	if len(rows) == 0 || limit <= 0 {
		return nil
	}

	cloned := append([]models.FacilityPerformanceRow(nil), rows...)
	sort.SliceStable(cloned, func(i, j int) bool {
		if cloned[i].ReportingRate != cloned[j].ReportingRate {
			return cloned[i].ReportingRate > cloned[j].ReportingRate
		}
		if cloned[i].SubmittedThisWeek != cloned[j].SubmittedThisWeek {
			return cloned[i].SubmittedThisWeek > cloned[j].SubmittedThisWeek
		}
		return cloned[i].PendingApproval < cloned[j].PendingApproval
	})

	if len(cloned) > limit {
		cloned = cloned[:limit]
	}
	return cloned
}

func facilitiesNeedingSupport(rows []models.FacilityPerformanceRow, limit int) []models.FacilityPerformanceRow {
	if len(rows) == 0 || limit <= 0 {
		return nil
	}

	cloned := append([]models.FacilityPerformanceRow(nil), rows...)
	sort.SliceStable(cloned, func(i, j int) bool {
		if cloned[i].ReportingRate != cloned[j].ReportingRate {
			return cloned[i].ReportingRate < cloned[j].ReportingRate
		}
		if cloned[i].PendingApproval != cloned[j].PendingApproval {
			return cloned[i].PendingApproval > cloned[j].PendingApproval
		}
		return cloned[i].PendingThisWeek > cloned[j].PendingThisWeek
	})

	if len(cloned) > limit {
		cloned = cloned[:limit]
	}
	return cloned
}

func priorityDepartments(rows []models.DepartmentAnalysisRow, limit int) []models.DepartmentAnalysisRow {
	if len(rows) == 0 || limit <= 0 {
		return nil
	}

	cloned := append([]models.DepartmentAnalysisRow(nil), rows...)
	sort.SliceStable(cloned, func(i, j int) bool {
		if cloned[i].ReportingRate != cloned[j].ReportingRate {
			return cloned[i].ReportingRate < cloned[j].ReportingRate
		}
		if cloned[i].PendingReports != cloned[j].PendingReports {
			return cloned[i].PendingReports > cloned[j].PendingReports
		}
		return cloned[i].DepartmentName < cloned[j].DepartmentName
	})

	if len(cloned) > limit {
		cloned = cloned[:limit]
	}
	return cloned
}

func priorityClinicians(rows []models.ClinicianAnalysisRow, limit int) []models.ClinicianAnalysisRow {
	if len(rows) == 0 || limit <= 0 {
		return nil
	}

	cloned := append([]models.ClinicianAnalysisRow(nil), rows...)
	sort.SliceStable(cloned, func(i, j int) bool {
		leftPriority := clinicianPriorityScore(cloned[i])
		rightPriority := clinicianPriorityScore(cloned[j])
		if leftPriority != rightPriority {
			return leftPriority > rightPriority
		}
		if cloned[i].PatientsReviewed != cloned[j].PatientsReviewed {
			return cloned[i].PatientsReviewed < cloned[j].PatientsReviewed
		}
		return cloned[i].EmployeeName < cloned[j].EmployeeName
	})

	if len(cloned) > limit {
		cloned = cloned[:limit]
	}
	return cloned
}

func clinicianPriorityScore(row models.ClinicianAnalysisRow) int {
	switch row.SubmissionStatus {
	case "Declined":
		return 5
	case "Draft":
		return 4
	case "Not Entered":
		if row.LeaveStatus == "On Leave" {
			return 1
		}
		return 3
	case "Submitted":
		return 2
	default:
		return 0
	}
}

func addReportPeriodSuffix(selectedWeekLabel string) string {
	if selectedWeekLabel == "" {
		return "Across all records"
	}
	return selectedWeekLabel
}

func LogoutHandler(c *gin.Context, sessionManager *scs.SessionManager) {

	// Get the context with the session
	sessionCtx, err := sessionManager.Load(c.Request.Context(), "session_token")
	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	// Destroy the session
	err = sessionManager.Destroy(sessionCtx)
	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	// Clear the session token cookie
	c.SetCookie("session_token", "", -1, "/", "", sessionManager.Cookie.Secure, sessionManager.Cookie.HttpOnly)

	// Redirect to the login page or another page after logout
	utilities.Info("Logout successful")
	c.Redirect(http.StatusFound, "/login")
}
