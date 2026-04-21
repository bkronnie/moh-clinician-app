package handlers

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/alexedwards/scs/v2"
	"github.com/gin-gonic/gin"
	"github.com/moh/clinician/internals/models"
	"github.com/moh/clinician/internals/security"
	"github.com/moh/clinician/internals/utilities"
)

type DashboardKPIView struct {
	Title   string
	Value   int
	Unit    string
	Meta    string
	Link    string
	Tooltip string
}

type DashboardTabView struct {
	ID    string
	Label string
	URL   string
}

type DashboardChartView struct {
	ID           string
	Title        string
	Subtitle     string
	ChartType    string
	Labels       []string
	SeriesA      []int
	SeriesALabel string
	SeriesB      []int
	SeriesBLabel string
}

type DashboardActionView struct {
	Label string
	URL   string
	Tone  string
}

type DashboardOptionView struct {
	Value string
	Label string
}

type DashboardFilterFieldView struct {
	Name     string
	ID       string
	Label    string
	Options  []DashboardOptionView
	Selected string
}

type DashboardBadgeView struct {
	Text string
	Tone string
}

type DashboardMetricView struct {
	Label string
	Value string
}

type DashboardSummaryCellView struct {
	Label    string
	Value    string
	Subvalue string
	Badge    *DashboardBadgeView
}

type DashboardSummaryRowView struct {
	Cells []DashboardSummaryCellView
}

type DashboardSummarySectionView struct {
	TabID        string
	TabLabel     string
	Title        string
	Subtitle     string
	EmptyMessage string
	Rows         []DashboardSummaryRowView
}

type DashboardDetailRowView struct {
	Title    string
	Subtitle string
	Badges   []DashboardBadgeView
	Metrics  []DashboardMetricView
}

type DashboardDetailSectionView struct {
	TabID        string
	TabLabel     string
	Title        string
	Subtitle     string
	EmptyMessage string
	Rows         []DashboardDetailRowView
}

type HomeViewModel struct {
	Role                string
	ScopeTitle          string
	ScopeSubtitle       string
	FilterSummary       string
	SelectedFacility    int
	FacilityOptions     []models.DashboardFilterOption
	SelectedDepartment  int
	DepartmentOptions   []models.DashboardFilterOption
	SelectedEmployee    int
	EmployeeOptions     []models.DashboardFilterOption
	OfficerName         string
	OfficerID           int
	OfficerTitle        string
	OfficerFacility     string
	OfficerDepartment   string
	SelectedYear        int
	SelectedMonth       int
	SelectedWeek        int
	SelectedWeekLabel   string
	AvailableYears      []int
	AvailableMonths     []models.DashboardFilterOption
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
	TopDepartments      []models.DepartmentAnalysisRow
	PriorityDepartments []models.DepartmentAnalysisRow
	TopClinicians       []models.ClinicianAnalysisRow
	PriorityClinicians  []models.ClinicianAnalysisRow
	DataEntryURL        string
	LatestSubmission    string
	ReportHistoryURL    string
	IdentityTitle       string
	IdentityName        string
	IdentityBadges      []DashboardBadgeView
	IdentityNote        string
	FilterTitle         string
	FilterHint          string
	FilterFormID        string
	FilterFields        []DashboardFilterFieldView
	ClearFiltersURL     string
	PrimaryActions      []DashboardActionView
	OverviewCharts      []DashboardChartView
	SummarySections     []DashboardSummarySectionView
	DetailSections      []DashboardDetailSectionView
	ActiveTab           string
	Tabs                []DashboardTabView
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
	standardizeHomeViewModel(&viewModel)

	sessionData.Form = viewModel
	log.Printf("data variable passed in HandlerHome: %v", sessionData)

	fmt.Println("home page loading with data")
	utilities.GenerateHTML(c, sessionData, "base", "home")

}

// roleFromRights normalizes a stored rights label to a canonical access role name.
func roleFromRights(rights string) string {
	return utilities.NormalizeRoleKey(rights)
}

func hasEmployeeMapping(user *models.User) bool {
	if user == nil {
		return false
	}
	return user.Employees.Valid && user.Employees.Int64 > 0
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
	roleID, roleName, err := models.UserRoleByID(c, db, user.ID)
	if err != nil {
		return ses, err
	}
	ses.Right = roleID
	ses.Rights = roleName
	ses.UserID = int64(user.ID)

	if !hasEmployeeMapping(user) {
		log.Printf("SES_SET: user %d has no employee mapping; continuing with role-only session", user.ID)
		return ses, nil
	}

	emp, err := models.EmployeeByID(c, db, int(user.Employees.Int64))
	if err != nil {
		log.Printf("SES_SET: failed to load employee profile for user %d (employee_id=%d): %v", user.ID, user.Employees.Int64, err)
		return ses, nil
	}

	ses.FName = emp.Fname.String
	ses.LName = emp.Lname.String
	ses.Title = emp.EmpTitle.String

	// National users can legitimately have no facility/department assignment.
	// Keep session construction resilient so role-based navigation still renders.
	if emp.EmpFacility > 0 {
		hf, hfErr := models.FacilityByID(c, db, int(emp.EmpFacility))
		if hfErr != nil {
			log.Printf("SES_SET: failed to resolve facility for employee %d (facility_id=%d): %v", emp.EmpID, emp.EmpFacility, hfErr)
		} else {
			ses.HFName = hf.FacilityName
			ses.HFID = int64(hf.FacilityID)
		}
	}

	if emp.EmpDepartment > 0 {
		hd, hdErr := models.DepartmentByID(c, db, int(emp.EmpDepartment))
		if hdErr != nil {
			log.Printf("SES_SET: failed to resolve department for employee %d (department_id=%d): %v", emp.EmpID, emp.EmpDepartment, hdErr)
		} else {
			ses.HDID = int64(hd.DeptID)
		}
	}

	return ses, nil

}

func Get_Session_Data(c *gin.Context, db *sql.DB, sessionManager *scs.SessionManager, pData any) any {
	id := security.GetCurrentUser(c, sessionManager)
	if id == 0 {
		return nil
	}

	flash := sessionManager.PopString(c.Request.Context(), "flash")
	data := GetPageData(flash)

	ses, er := SES_SET(c, db, sessionManager)
	if er != nil {
		log.Printf("failed to build session data: %v", er)
		data.Ses = utilities.SessionDetails{}
	} else {
		data.Ses = ses

		ctx := c.Request.Context()

		// Expire any approved leaves whose end_date has passed
		if ses.HFID > 0 {
			if err := models.ExpireOldLeave(ctx, db, ses.HFID); err != nil {
				log.Printf("leave expiry: %v", err)
			}
		}

		// Load notification counts by role
		switch roleFromRights(ses.Rights) {
		case utilities.RoleFacilityAdmin:
			data.NotifPendingLeave, _ = models.GetPendingLeaveCount(ctx, db, ses.HFID)
			data.NotifPendingReports, _ = models.GetPendingReportCount(ctx, db, ses.HFID)
			missingReports, _ := models.GetClinicianMissingRequiredReportsCount(ctx, db, int(ses.EmpID))
			data.NotifMyDataEntry = missingReports
			if missingReports > 0 {
				data.NotifMyDataState = "overdue"
			} else {
				data.NotifMyDataState = "current"
			}
		case utilities.RoleNationalAdmin:
			data.NotifPendingReports, _ = models.GetPendingReportCount(ctx, db, 0)
		case utilities.RoleStaff:
			data.NotifMyLeave, _ = models.GetMyRecentLeaveUpdates(ctx, db, ses.EmpID)
			data.NotifMyReports, _ = models.GetMyRecentReportUpdates(ctx, db, ses.EmpID)
			missingReports, _ := models.GetClinicianMissingRequiredReportsCount(ctx, db, int(ses.EmpID))
			data.NotifMyDataEntry = missingReports
			if missingReports > 0 {
				data.NotifMyDataState = "overdue"
			} else {
				data.NotifMyDataState = "current"
			}
		}
	}

	data.Form = pData
	return data
}

func buildHomeViewModel(c *gin.Context, db *sql.DB, ses utilities.SessionDetails) (HomeViewModel, error) {
	selectedFacility, _ := parseOptionalIntQuery(c, "facility")
	selectedDepartment, _ := parseOptionalIntQuery(c, "department")
	selectedEmployee, _ := parseOptionalIntQuery(c, "employee")
	selectedYear, hasYear := parseOptionalIntQuery(c, "year")
	selectedMonth, hasMonth := parseOptionalIntQuery(c, "month")
	selectedWeek, hasWeek := parseOptionalIntQuery(c, "week")
	activeTab := strings.TrimSpace(c.Query("tab"))

	switch roleFromRights(ses.Rights) {
	case utilities.RoleNationalAdmin:
		return buildNationalHome(c, db, selectedFacility, selectedDepartment, selectedEmployee, selectedYear, hasYear, selectedMonth, hasMonth, selectedWeek, hasWeek, activeTab)
	case utilities.RoleFacilityAdmin:
		return buildFacilityManagerHome(c, db, int(ses.HFID), ses.HFName, selectedDepartment, selectedEmployee, selectedYear, hasYear, selectedMonth, hasMonth, selectedWeek, hasWeek, activeTab)
	default:
		return buildClinicianHome(c, db, int(ses.EmpID), ses.HFName, selectedYear, selectedMonth, selectedWeek, activeTab)
	}
}

func buildNationalHome(c *gin.Context, db *sql.DB, selectedFacilityID int, selectedDepartmentID int, selectedEmployeeID int, requestedYear int, hasYear bool, requestedMonth int, hasMonth bool, requestedWeek int, hasWeek bool, activeTab string) (HomeViewModel, error) {
	selectedYear, selectedMonth, selectedWeek, availableYears, availableMonths, availableWeeks, selectedWeekLabel, periodStart, periodEnd, err := resolveNationalDashboardPeriod(c, db, requestedYear, hasYear, requestedMonth, hasMonth, requestedWeek, hasWeek)
	if err != nil {
		return HomeViewModel{}, err
	}

	snapshot, err := models.GetNationalDashboardSnapshotByRange(c.Request.Context(), db, periodStart, periodEnd, selectedFacilityID, selectedDepartmentID, selectedEmployeeID)
	if err != nil {
		return HomeViewModel{}, err
	}

	facilityOptions, err := models.GetDashboardFacilityOptions(c.Request.Context(), db)
	if err != nil {
		return HomeViewModel{}, err
	}
	departmentOptions, err := models.GetDashboardDepartmentOptions(c.Request.Context(), db, selectedFacilityID)
	if err != nil {
		return HomeViewModel{}, err
	}
	employeeOptions, err := models.GetDashboardEmployeeOptions(c.Request.Context(), db, selectedFacilityID, selectedDepartmentID)
	if err != nil {
		return HomeViewModel{}, err
	}

	pendingSubmission := 0
	for _, row := range snapshot.FacilityPerformance {
		pendingSubmission += row.PendingThisWeek
	}

	scopeSuffix := "all facilities"
	if selectedFacilityID > 0 {
		for _, option := range facilityOptions {
			if option.ID == selectedFacilityID {
				scopeSuffix = option.Name
				break
			}
		}
	}

	view := HomeViewModel{
		Role:               utilities.RoleNationalAdmin,
		ScopeTitle:         "National Reporting Dashboard",
		ScopeSubtitle:      "Track national reporting performance across facilities using the latest reporting period stored in the database, then narrow the view with year, month, week, and facility filters.",
		FilterSummary:      fmt.Sprintf("%s | Scope: %s", selectedWeekLabel, scopeSuffix),
		PerformanceRows:    snapshot.FacilityPerformance,
		SelectedFacility:   selectedFacilityID,
		SelectedDepartment: selectedDepartmentID,
		DepartmentOptions:  departmentOptions,
		SelectedEmployee:   selectedEmployeeID,
		EmployeeOptions:    employeeOptions,
		SelectedYear:       selectedYear,
		SelectedMonth:      selectedMonth,
		SelectedWeek:       selectedWeek,
		SelectedWeekLabel:  selectedWeekLabel,
		AvailableYears:     availableYears,
		AvailableMonths:    availableMonths,
		AvailableWeeks:     availableWeeks,
		FacilityOptions:    facilityOptions,
		ActiveTab:          activeTab,
	}

	view.Stats = []map[string]interface{}{
		{"Title": "Scope", "Value": scopeSuffix, "Tone": "stat-primary"},
		{"Title": "Facilities in Scope", "Value": snapshot.TotalFacilities, "Tone": "stat-info"},
		{"Title": "Clinicians in Scope", "Value": snapshot.TotalClinicians, "Tone": "stat-primary"},
		{"Title": "Reporting Rate", "Value": fmt.Sprintf("%d%%", snapshot.NationalReportingRate), "Tone": "stat-success"},
	}

	view.TopFacilities = topFacilityPerformers(snapshot.FacilityPerformance, 5)
	view.AttentionFacilities = facilitiesNeedingSupport(snapshot.FacilityPerformance, 5)

	departmentAnalysis, err := models.GetNationalDepartmentAnalysisByRange(c.Request.Context(), db, selectedFacilityID, selectedDepartmentID, selectedEmployeeID, periodStart, periodEnd)
	if err != nil {
		return HomeViewModel{}, err
	}
	view.DepartmentAnalysis = departmentAnalysis
	view.TopDepartments = topDepartments(departmentAnalysis, 5)
	view.PriorityDepartments = priorityDepartments(departmentAnalysis, 5)

	clinicianAnalysis, err := models.GetNationalClinicianAnalysisByRange(c.Request.Context(), db, selectedFacilityID, selectedDepartmentID, selectedEmployeeID, periodStart, periodEnd)
	if err != nil {
		return HomeViewModel{}, err
	}
	view.ClinicianAnalysis = clinicianAnalysis
	view.TopClinicians = topClinicians(clinicianAnalysis, 5)
	view.PriorityClinicians = priorityClinicians(clinicianAnalysis, 5)
	approvedCount, declinedCount, onLeaveCount := summarizeClinicianStatus(clinicianAnalysis)
	firstPassApprovalRate := percentage(approvedCount, approvedCount+declinedCount)

	view.KPIs = []DashboardKPIView{
		{
			Title: "National Reporting Rate",
			Value: snapshot.NationalReportingRate,
			Unit:  "% reporting rate",
			Meta:  selectedWeekLabel,
		},
		{
			Title: "First-Pass Approval Rate",
			Value: firstPassApprovalRate,
			Unit:  "% approved first-pass",
			Meta:  selectedWeekLabel,
		},
		{
			Title: "Facilities in Scope",
			Value: snapshot.TotalFacilities,
			Unit:  "facilities",
			Meta:  "Filtered national view",
		},
		{
			Title: "Clinicians in Scope",
			Value: snapshot.TotalClinicians,
			Unit:  "clinicians",
			Meta:  "Reporting population",
		},
		{
			Title: "Reports Entered",
			Value: snapshot.ReportsEnteredThisWeek,
			Unit:  "entered reports",
			Meta:  selectedWeekLabel,
		},
		{
			Title: "Reports Submitted",
			Value: snapshot.SubmittedThisWeek,
			Unit:  "submitted reports",
			Meta:  selectedWeekLabel,
		},
		{
			Title: "Pending Approval",
			Value: snapshot.PendingApproval,
			Unit:  "submitted reports",
			Meta:  "Awaiting approval",
		},
		{
			Title: "Missing Submissions",
			Value: pendingSubmission,
			Unit:  "clinicians",
			Meta:  "Still not submitted",
		},
		{
			Title: "Staff On Leave",
			Value: onLeaveCount,
			Unit:  "clinicians",
			Meta:  "Current leave status",
		},
	}

	trendPoints, err := models.GetScopeTrendPoints(c.Request.Context(), db, 0, selectedFacilityID, selectedDepartmentID, selectedEmployeeID, selectedYear, selectedMonth, 0)
	if err != nil {
		return HomeViewModel{}, err
	}
	if len(trendPoints) > 0 {
		view.OverviewCharts = append(view.OverviewCharts,
			buildScopeTrendChart("National Reporting Trend", "Submitted reports compared with pending approval across the selected time range.", trendPoints),
		)
	}
	if len(view.PerformanceRows) > 0 {
		view.OverviewCharts = append(view.OverviewCharts,
			buildFacilityComparisonChart("Facility Comparison", "Submitted reports versus missing submissions by facility for the selected period.", view.PerformanceRows),
		)
	}
	if len(view.DepartmentAnalysis) > 0 {
		view.OverviewCharts = append(view.OverviewCharts,
			buildDepartmentWorkloadChart("Department Workload Comparison", "Compare patient reviews and ward rounds between departments in the current national scope.", view.DepartmentAnalysis),
		)
	}

	return view, nil
}

func buildFacilityManagerHome(c *gin.Context, db *sql.DB, facilityID int, facilityName string, selectedDepartmentID int, selectedEmployeeID int, requestedYear int, hasYear bool, requestedMonth int, hasMonth bool, requestedWeek int, hasWeek bool, activeTab string) (HomeViewModel, error) {
	selectedYear, selectedMonth, selectedWeek, availableYears, availableMonths, availableWeeks, selectedWeekLabel, periodStart, periodEnd, err := resolveNationalDashboardPeriod(c, db, requestedYear, hasYear, requestedMonth, hasMonth, requestedWeek, hasWeek)
	if err != nil {
		return HomeViewModel{}, err
	}

	snapshot, err := models.GetFacilityManagerDashboardSnapshotByRange(c.Request.Context(), db, facilityID, selectedDepartmentID, selectedEmployeeID, periodStart, periodEnd)
	if err != nil {
		return HomeViewModel{}, err
	}

	departmentOptions, err := models.GetDashboardDepartmentOptions(c.Request.Context(), db, facilityID)
	if err != nil {
		return HomeViewModel{}, err
	}
	employeeOptions, err := models.GetDashboardEmployeeOptions(c.Request.Context(), db, facilityID, selectedDepartmentID)
	if err != nil {
		return HomeViewModel{}, err
	}

	departmentAnalysis, err := models.GetFacilityDepartmentAnalysisByRange(c.Request.Context(), db, facilityID, selectedDepartmentID, selectedEmployeeID, periodStart, periodEnd)
	if err != nil {
		return HomeViewModel{}, err
	}
	clinicianAnalysis, err := models.GetFacilityClinicianAnalysisByRange(c.Request.Context(), db, facilityID, selectedDepartmentID, selectedEmployeeID, periodStart, periodEnd)
	if err != nil {
		return HomeViewModel{}, err
	}

	reportSummary, err := models.GetFacilityReportReviewSummary(c.Request.Context(), db, int64(facilityID))
	if err != nil {
		return HomeViewModel{}, err
	}
	leaveSummary, err := models.GetFacilityLeaveReviewSummary(c.Request.Context(), db, int64(facilityID), int64(selectedDepartmentID), selectedYear, selectedMonth, selectedWeek)
	if err != nil {
		return HomeViewModel{}, err
	}

	trendPoints, err := models.GetScopeTrendPoints(c.Request.Context(), db, facilityID, 0, selectedDepartmentID, selectedEmployeeID, selectedYear, selectedMonth, 0)
	if err != nil {
		return HomeViewModel{}, err
	}

	reportingRate := percentage(snapshot.SubmittedThisWeek, snapshot.TotalClinicians)
	departmentSummary := summarizeDepartmentAnalysis(departmentAnalysis)
	scopeLabel := facilityName
	if selectedDepartmentID > 0 {
		for _, option := range departmentOptions {
			if option.ID == selectedDepartmentID {
				scopeLabel = fmt.Sprintf("%s | %s", facilityName, option.Name)
				break
			}
		}
	}

	view := HomeViewModel{
		Role:               utilities.RoleFacilityAdmin,
		ScopeTitle:         fmt.Sprintf("%s Facility Dashboard", facilityName),
		ScopeSubtitle:      "Monitor department performance, staff activity, and review queues for the current facility scope.",
		FilterSummary:      fmt.Sprintf("%s | Scope: %s", selectedWeekLabel, scopeLabel),
		DepartmentRows:     snapshot.Departments,
		DepartmentAnalysis: departmentAnalysis,
		ClinicianAnalysis:  clinicianAnalysis,
		SelectedDepartment: selectedDepartmentID,
		DepartmentOptions:  departmentOptions,
		SelectedEmployee:   selectedEmployeeID,
		EmployeeOptions:    employeeOptions,
		SelectedYear:       selectedYear,
		SelectedMonth:      selectedMonth,
		SelectedWeek:       selectedWeek,
		SelectedWeekLabel:  selectedWeekLabel,
		AvailableYears:     availableYears,
		AvailableMonths:    availableMonths,
		AvailableWeeks:     availableWeeks,
		ActiveTab:          activeTab,
	}

	view.Stats = []map[string]interface{}{
		{"Title": "Facility Reporting Rate", "Value": fmt.Sprintf("%d%%", reportingRate), "Tone": "stat-success"},
		{"Title": "Clinicians in Scope", "Value": snapshot.TotalClinicians, "Tone": "stat-primary"},
		{"Title": "Departments in Scope", "Value": len(snapshot.Departments), "Tone": "stat-info"},
		{"Title": "Review Queue", "Value": reportSummary.PendingReports + leaveSummary.PendingLeaves, "Tone": "stat-warning"},
	}

	view.KPIs = []DashboardKPIView{
		{Title: "Facility Reporting Rate", Value: reportingRate, Unit: "% reporting rate", Meta: selectedWeekLabel},
		{Title: "Reports Entered", Value: snapshot.ReportsEnteredThisWeek, Unit: "entered reports", Meta: selectedWeekLabel},
		{Title: "Reports Submitted", Value: snapshot.SubmittedThisWeek, Unit: "submitted reports", Meta: selectedWeekLabel},
		{Title: "Pending Submission", Value: snapshot.PendingSubmission, Unit: "clinicians", Meta: "Still not submitted"},
		{Title: "Pending Report Approvals", Value: reportSummary.PendingReports, Unit: "reports", Meta: "Current review queue", Link: buildReportSubmissionsURL(0, 0, selectedYear, selectedMonth, selectedWeek, "pending")},
		{Title: "First-Pass Approval Rate", Value: percentage(reportSummary.ApprovedReports, reportSummary.ApprovedReports+reportSummary.DeclinedReports), Unit: "% approved first-pass", Meta: selectedWeekLabel},
		{Title: "Staff On Leave", Value: countCliniciansOnLeave(clinicianAnalysis), Unit: "clinicians", Meta: "Current leave status"},
		{Title: "Pending Leave Requests", Value: leaveSummary.PendingLeaves, Unit: "leave requests", Meta: "Current review queue", Link: buildFacilityLeaveReviewURL("pending", selectedDepartmentID, selectedYear, selectedMonth, selectedWeek)},
		{Title: "Ward Rounds", Value: departmentSummary.WardRounds, Unit: "rounds", Meta: selectedWeekLabel},
		{Title: "Patients Reviewed", Value: departmentSummary.PatientsReviewed, Unit: "patients", Meta: selectedWeekLabel},
		{Title: "Procedures", Value: departmentSummary.Procedures, Unit: "procedures", Meta: selectedWeekLabel},
	}

	view.TopDepartments = topDepartments(departmentAnalysis, 5)
	view.PriorityDepartments = priorityDepartments(departmentAnalysis, 5)
	view.TopClinicians = topClinicians(clinicianAnalysis, 5)
	view.PriorityClinicians = priorityClinicians(clinicianAnalysis, 5)
	if len(snapshot.Departments) > 0 {
		view.OverviewCharts = append(view.OverviewCharts,
			buildDepartmentProgressChart("Department Submission Comparison", "Compare entered reports against submitted reports across departments in the current scope.", snapshot.Departments),
		)
	}
	if len(departmentAnalysis) > 0 {
		view.OverviewCharts = append(view.OverviewCharts,
			buildDepartmentWorkloadChart("Department Clinical Workload", "Compare patient reviews and ward rounds by department to spot workload differences.", departmentAnalysis),
		)
	}
	if len(trendPoints) > 0 {
		view.OverviewCharts = append(view.OverviewCharts,
			buildScopeTrendChart("Facility Trend", "Submitted reports compared with pending approval across the selected period.", trendPoints),
		)
	}

	return view, nil
}

func buildClinicianHome(c *gin.Context, db *sql.DB, employeeID int, facilityName string, selectedYear int, selectedMonth int, selectedWeek int, activeTab string) (HomeViewModel, error) {
	snapshot, err := models.GetClinicianDashboardSnapshot(c.Request.Context(), db, employeeID, selectedYear, selectedMonth, selectedWeek)
	if err != nil {
		return HomeViewModel{}, err
	}

	missingRequiredReports, err := models.GetClinicianMissingRequiredReportsCount(c.Request.Context(), db, employeeID)
	if err != nil {
		return HomeViewModel{}, err
	}

	entryStatus := "Current"
	entryTone := "stat-success"
	if missingRequiredReports > 0 {
		entryStatus = "Overdue"
		entryTone = "stat-warning"
	}

	view := HomeViewModel{
		Role:              utilities.RoleStaff,
		ScopeTitle:        "My Dashboard",
		ScopeSubtitle:     "Review your clinical activity, reporting progress, and trends using year, month, and week filters.",
		FilterSummary:     snapshot.SelectedWeekLabel,
		OfficerName:       snapshot.EmployeeName,
		OfficerID:         snapshot.EmployeeID,
		OfficerTitle:      snapshot.EmployeeTitle,
		OfficerFacility:   snapshot.FacilityName,
		OfficerDepartment: snapshot.DepartmentName,
		SelectedYear:      snapshot.SelectedYear,
		SelectedMonth:     snapshot.SelectedMonth,
		SelectedWeek:      snapshot.SelectedWeek,
		SelectedWeekLabel: snapshot.SelectedWeekLabel,
		AvailableYears:    snapshot.AvailableYears,
		AvailableMonths:   snapshot.AvailableMonths,
		AvailableWeeks:    snapshot.AvailableWeeks,
		DataEntryURL:      "/reports/new/0",
		ReportHistoryURL:  buildReportHistoryPeriodFilterQuery("all", snapshot.SelectedYear, snapshot.SelectedMonth, snapshot.SelectedWeek),
		ActiveTab:         activeTab,
	}

	view.KPIs = []DashboardKPIView{
		{
			Title: "Total Reports",
			Value: snapshot.TotalReports,
			Unit:  "reports",
			Meta:  addReportPeriodSuffix(snapshot.SelectedWeekLabel),
			Link:  buildReportHistoryPeriodFilterQuery("all", snapshot.SelectedYear, snapshot.SelectedMonth, snapshot.SelectedWeek),
		},
		{
			Title: "Submitted",
			Value: snapshot.SubmittedReports,
			Unit:  "reports",
			Meta:  addReportPeriodSuffix(snapshot.SelectedWeekLabel),
			Link:  buildReportHistoryPeriodFilterQuery("submitted", snapshot.SelectedYear, snapshot.SelectedMonth, snapshot.SelectedWeek),
		},
		{
			Title: "Approved",
			Value: snapshot.ApprovedReports,
			Unit:  "reports",
			Meta:  addReportPeriodSuffix(snapshot.SelectedWeekLabel),
			Link:  buildReportHistoryPeriodFilterQuery("approved", snapshot.SelectedYear, snapshot.SelectedMonth, snapshot.SelectedWeek),
		},
		{
			Title: "Declined",
			Value: snapshot.DeclinedReports,
			Unit:  "reports",
			Meta:  addReportPeriodSuffix(snapshot.SelectedWeekLabel),
			Link:  buildReportHistoryPeriodFilterQuery("declined", snapshot.SelectedYear, snapshot.SelectedMonth, snapshot.SelectedWeek),
		},
		{
			Title: "Submission Progress",
			Value: snapshot.SubmissionProgress,
			Unit:  "% submitted",
			Meta:  addReportPeriodSuffix(snapshot.SelectedWeekLabel),
		},
		{
			Title: "First-Pass Approval Rate",
			Value: percentage(snapshot.ApprovedReports, snapshot.ApprovedReports+snapshot.DeclinedReports),
			Unit:  "% approved first-pass",
			Meta:  addReportPeriodSuffix(snapshot.SelectedWeekLabel),
		},
		{
			Title: "Days Worked",
			Value: snapshot.DaysWorked,
			Unit:  "days",
			Meta:  addReportPeriodSuffix(snapshot.SelectedWeekLabel),
		},
		{
			Title: "Patients per Day",
			Value: ratioPerUnit(snapshot.PatientsReviewed, snapshot.DaysWorked),
			Unit:  "patients/day",
			Meta:  addReportPeriodSuffix(snapshot.SelectedWeekLabel),
		},
		{
			Title: "Procedures per Day",
			Value: ratioPerUnit(snapshot.Procedures, snapshot.DaysWorked),
			Unit:  "procedures/day",
			Meta:  addReportPeriodSuffix(snapshot.SelectedWeekLabel),
		},
		{
			Title: "Ward Rounds",
			Value: snapshot.WardRounds,
			Unit:  "rounds",
			Meta:  addReportPeriodSuffix(snapshot.SelectedWeekLabel),
		},
		{
			Title: "Patients Reviewed",
			Value: snapshot.PatientsReviewed,
			Unit:  "patients",
			Meta:  addReportPeriodSuffix(snapshot.SelectedWeekLabel),
		},
		{
			Title: "Procedures",
			Value: snapshot.Procedures,
			Unit:  "procedures",
			Meta:  addReportPeriodSuffix(snapshot.SelectedWeekLabel),
		},
	}

	view.Stats = []map[string]interface{}{
		{"Title": "Entry Status", "Value": entryStatus, "Tone": entryTone},
		{"Title": "Missing Required Reports", "Value": missingRequiredReports, "Tone": entryTone},
		{"Title": "Attendance Rate", "Value": fmt.Sprintf("%d%%", snapshot.AttendanceRate), "Tone": "stat-success"},
		{"Title": "Submission Progress", "Value": fmt.Sprintf("%d%%", snapshot.SubmissionProgress), "Tone": "stat-primary"},
	}

	if len(snapshot.RecentWeeks) > 0 {
		view.OverviewCharts = append(view.OverviewCharts,
			buildClinicianWorkloadChart("Clinical Workload Trend", "Patients reviewed compared with procedures across your selected reporting range.", snapshot.RecentWeeks),
			buildClinicianActivityChart("Attendance and Ward Rounds Trend", "Track working days against ward rounds to spot consistency over time.", snapshot.RecentWeeks),
		)
	}
	view.LatestSubmission = fmt.Sprintf("%s: %s", snapshot.SelectedWeekLabel, entryStatus)

	_ = facilityName

	return view, nil
}

func standardizeHomeViewModel(view *HomeViewModel) {
	view.FilterFormID = "dashboardFilters"

	if len(view.OverviewCharts) == 0 && len(view.SeriesLabels) > 0 {
		view.OverviewCharts = append(view.OverviewCharts, DashboardChartView{
			ID:           makeChartID("overview", view.SeriesTitle),
			Title:        view.SeriesTitle,
			Labels:       view.SeriesLabels,
			SeriesA:      view.SeriesA,
			SeriesALabel: view.SeriesALabel,
			SeriesB:      view.SeriesB,
			SeriesBLabel: view.SeriesBLabel,
		})
	}

	switch view.Role {
	case utilities.RoleStaff:
		view.IdentityTitle = "Medical Officer Profile"
		view.IdentityName = view.OfficerName
		view.IdentityBadges = []DashboardBadgeView{
			{Text: fmt.Sprintf("ID: %d", view.OfficerID), Tone: "info"},
			{Text: view.OfficerTitle, Tone: "info"},
			{Text: view.OfficerDepartment, Tone: "info"},
			{Text: view.OfficerFacility, Tone: "info"},
		}
		view.IdentityNote = view.LatestSubmission
		view.FilterTitle = "Dashboard Filters"
		view.FilterHint = "Change a filter and the dashboard refreshes immediately."
		view.FilterFields = []DashboardFilterFieldView{
			{
				Name:     "year",
				ID:       "dashboardYear",
				Label:    "Year",
				Options:  intOptionsToFilterOptions(view.AvailableYears, true),
				Selected: strconv.Itoa(view.SelectedYear),
			},
			{
				Name:     "month",
				ID:       "dashboardMonth",
				Label:    "Month",
				Options:  dashboardOptionsToFilterOptions(view.AvailableMonths),
				Selected: strconv.Itoa(view.SelectedMonth),
			},
			{
				Name:     "week",
				ID:       "dashboardWeek",
				Label:    "Week",
				Options:  weekOptionsToFilterOptions(view.AvailableWeeks, true),
				Selected: strconv.Itoa(view.SelectedWeek),
			},
		}
		view.PrimaryActions = []DashboardActionView{
			{Label: "Open My Weekly Entry Form", URL: view.DataEntryURL, Tone: "primary"},
			{Label: "View Reports", URL: view.ReportHistoryURL, Tone: "secondary"},
		}
		view.SummarySections = append(view.SummarySections,
			withSummaryTab(buildClinicianPeriodSummarySection(*view), "activity", "Activity"),
		)
		view.DetailSections = append(view.DetailSections,
			withDetailTab(buildClinicianBenchmarkSection(*view), "benchmarks", "Benchmarks"),
		)
	case utilities.RoleNationalAdmin:
		view.FilterTitle = "National Filters"
		view.FilterHint = "National dashboards default to the latest reporting year, month, and week in the database."
		view.FilterFields = []DashboardFilterFieldView{
			{
				Name:     "facility",
				ID:       "dashboardFacility",
				Label:    "Facility",
				Options:  dashboardFilterOptionsWithAll(view.FacilityOptions, "All facilities"),
				Selected: strconv.Itoa(view.SelectedFacility),
			},
			{
				Name:     "year",
				ID:       "dashboardAdminYear",
				Label:    "Year",
				Options:  intOptionsToFilterOptions(view.AvailableYears, true),
				Selected: strconv.Itoa(view.SelectedYear),
			},
			{
				Name:     "month",
				ID:       "dashboardAdminMonth",
				Label:    "Month",
				Options:  dashboardOptionsToFilterOptions(view.AvailableMonths),
				Selected: strconv.Itoa(view.SelectedMonth),
			},
			{
				Name:     "week",
				ID:       "dashboardAdminWeek",
				Label:    "Week",
				Options:  weekOptionsToFilterOptions(view.AvailableWeeks, true),
				Selected: strconv.Itoa(view.SelectedWeek),
			},
			{
				Name:     "department",
				ID:       "dashboardAdminDepartment",
				Label:    "Department",
				Options:  dashboardFilterOptionsWithAll(view.DepartmentOptions, "All departments"),
				Selected: strconv.Itoa(view.SelectedDepartment),
			},
			{
				Name:     "employee",
				ID:       "dashboardAdminEmployee",
				Label:    "Employee",
				Options:  dashboardFilterOptionsWithAll(view.EmployeeOptions, "All staff"),
				Selected: strconv.Itoa(view.SelectedEmployee),
			},
		}
		view.SummarySections = append(view.SummarySections,
			withSummaryTab(buildNationalPerformanceSection(view.PerformanceRows, view.SelectedWeekLabel), "facilities", "Facilities"),
			withSummaryTab(buildFacilityTopPerformanceSection("Top 5 Performing Facilities", "Highest-performing facilities in the selected national scope.", view.TopFacilities), "facilities", "Facilities"),
			withSummaryTab(buildClinicianTopPerformanceSection("Top 5 Performing Staff Nationwide", "Highest-performing staff in the selected national scope.", view.TopClinicians), "clinicians", "Clinicians"),
		)
		view.DetailSections = append(view.DetailSections,
			withDetailTab(buildFacilitySupportSection("Facilities Requiring Support", "Facilities with the clearest reporting gaps in the current national scope.", view.AttentionFacilities), "facilities", "Facilities"),
		)
		if len(view.DepartmentAnalysis) > 0 {
			view.DetailSections = append(view.DetailSections,
				withDetailTab(buildDepartmentAnalysisSection("Top Performing Departments", "Departments leading national reporting and workload output in the current scope.", view.TopDepartments), "departments", "Departments"),
				withDetailTab(buildDepartmentAnalysisSection("Departments Requiring Follow-up", "Departments with weaker reporting or higher pending submission gaps in the current scope.", view.PriorityDepartments), "departments", "Departments"),
				withDetailTab(buildDepartmentAnalysisSection("Department Comparison", "Nationwide department reporting, workload, and attendance indicators for the selected scope.", view.DepartmentAnalysis), "departments", "Departments"),
			)
		}
		if len(view.ClinicianAnalysis) > 0 {
			view.DetailSections = append(view.DetailSections,
				withDetailTab(buildClinicianAnalysisSection("Staff Requiring Follow-up", "Staff whose reporting status or leave status requires follow-up in the current national scope.", view.PriorityClinicians), "clinicians", "Clinicians"),
				withDetailTab(buildClinicianAnalysisSection("Clinician Drill-down", "Clinician-level reporting and workload indicators for the current drill-down selection.", view.ClinicianAnalysis), "clinicians", "Clinicians"),
			)
		}
	case utilities.RoleFacilityAdmin:
		view.FilterTitle = "Facility Filters"
		view.FilterHint = "Facility dashboards refresh automatically and can be narrowed to a department or an individual employee."
		view.FilterFields = []DashboardFilterFieldView{
			{
				Name:     "year",
				ID:       "dashboardFacilityYear",
				Label:    "Year",
				Options:  intOptionsToFilterOptions(view.AvailableYears, true),
				Selected: strconv.Itoa(view.SelectedYear),
			},
			{
				Name:     "month",
				ID:       "dashboardFacilityMonth",
				Label:    "Month",
				Options:  dashboardOptionsToFilterOptions(view.AvailableMonths),
				Selected: strconv.Itoa(view.SelectedMonth),
			},
			{
				Name:     "week",
				ID:       "dashboardFacilityWeek",
				Label:    "Week",
				Options:  weekOptionsToFilterOptions(view.AvailableWeeks, true),
				Selected: strconv.Itoa(view.SelectedWeek),
			},
			{
				Name:     "department",
				ID:       "dashboardDepartment",
				Label:    "Department",
				Options:  dashboardFilterOptionsWithAll(view.DepartmentOptions, "All departments"),
				Selected: strconv.Itoa(view.SelectedDepartment),
			},
			{
				Name:     "employee",
				ID:       "dashboardEmployee",
				Label:    "Employee",
				Options:  dashboardFilterOptionsWithAll(view.EmployeeOptions, "All staff"),
				Selected: strconv.Itoa(view.SelectedEmployee),
			},
		}
		view.SummarySections = append(view.SummarySections,
			withSummaryTab(buildDepartmentFollowUpSection(view.DepartmentRows), "departments", "Departments"),
			withSummaryTab(buildDepartmentTopPerformanceSection("Top 5 Performing Departments", "Highest-performing departments in the current facility scope.", view.TopDepartments), "departments", "Departments"),
			withSummaryTab(buildClinicianTopPerformanceSection("Top 5 Performing Staff", "Highest-performing staff in the current facility scope.", view.TopClinicians), "staff", "Staff"),
		)
		view.DetailSections = append(view.DetailSections,
			withDetailTab(buildDepartmentAnalysisSection("Department Analysis", "Compare departments using reporting rate, ward rounds, attendance, and workload indicators.", view.DepartmentAnalysis), "departments", "Departments"),
			withDetailTab(buildDepartmentAnalysisSection("Priority Departments", "Departments that need the fastest admin follow-up in the current facility scope.", view.PriorityDepartments), "departments", "Departments"),
			withDetailTab(buildClinicianAnalysisSection("Staff Analysis", "Clinician-level reporting and workload indicators within the facility.", view.ClinicianAnalysis), "staff", "Staff"),
			withDetailTab(buildClinicianAnalysisSection("Staff Requiring Follow-up", "Staff whose reporting status or leave status requires attention.", view.PriorityClinicians), "staff", "Staff"),
		)
	}

	populateDashboardTabs(view)
	applyKPIDrilldowns(view)
	view.ClearFiltersURL = buildDashboardClearURL(*view)
}

func applyKPIDrilldowns(view *HomeViewModel) {
	facilitiesURL := dashboardTabURLForID(*view, "facilities")
	departmentsURL := dashboardTabURLForID(*view, "departments")
	cliniciansURL := dashboardTabURLForID(*view, "clinicians")
	staffURL := dashboardTabURLForID(*view, "staff")
	activityURL := dashboardTabURLForID(*view, "activity")
	benchmarksURL := dashboardTabURLForID(*view, "benchmarks")

	for i := range view.KPIs {
		kpi := &view.KPIs[i]

		switch view.Role {
		case utilities.RoleNationalAdmin:
			switch kpi.Title {
			case "National Reporting Rate", "Facilities in Scope", "Missing Submissions":
				kpi.Link = facilitiesURL
				kpi.Tooltip = "Open facility performance table"
			case "Clinicians in Scope", "First-Pass Approval Rate", "Staff On Leave":
				kpi.Link = cliniciansURL
				kpi.Tooltip = "Open nationwide clinician drill-down table"
			case "Reports Entered", "Reports Submitted":
				kpi.Link = buildReportSubmissionsURL(view.SelectedFacility, view.SelectedDepartment, view.SelectedYear, view.SelectedMonth, view.SelectedWeek, "all")
				kpi.Tooltip = "Open report submissions table"
			case "Pending Approval":
				kpi.Link = buildReportSubmissionsURL(view.SelectedFacility, view.SelectedDepartment, view.SelectedYear, view.SelectedMonth, view.SelectedWeek, "pending")
				kpi.Tooltip = "Open pending approval submissions"
			}
		case utilities.RoleFacilityAdmin:
			switch kpi.Title {
			case "Reports Entered", "Reports Submitted", "Facility Reporting Rate", "First-Pass Approval Rate", "Staff On Leave":
				kpi.Link = buildReportSubmissionsURL(0, view.SelectedDepartment, view.SelectedYear, view.SelectedMonth, view.SelectedWeek, "all")
				kpi.Tooltip = "Open facility report submissions table"
			case "Pending Submission", "Ward Rounds", "Patients Reviewed", "Procedures":
				kpi.Link = departmentsURL
				kpi.Tooltip = "Open department analysis table"
			case "Pending Report Approvals":
				kpi.Link = buildReportSubmissionsURL(0, view.SelectedDepartment, view.SelectedYear, view.SelectedMonth, view.SelectedWeek, "pending")
				kpi.Tooltip = "Open pending report approvals"
			case "Pending Leave Requests":
				kpi.Link = buildFacilityLeaveReviewURL("pending", view.SelectedDepartment, view.SelectedYear, view.SelectedMonth, view.SelectedWeek)
				kpi.Tooltip = "Open pending leave requests table"
			}
		default:
			switch kpi.Title {
			case "Total Reports":
				kpi.Link = buildReportHistoryPeriodFilterQuery("all", view.SelectedYear, view.SelectedMonth, view.SelectedWeek)
				kpi.Tooltip = "Open my report history"
			case "Submitted":
				kpi.Link = buildReportHistoryPeriodFilterQuery("submitted", view.SelectedYear, view.SelectedMonth, view.SelectedWeek)
				kpi.Tooltip = "Open submitted reports"
			case "Approved":
				kpi.Link = buildReportHistoryPeriodFilterQuery("approved", view.SelectedYear, view.SelectedMonth, view.SelectedWeek)
				kpi.Tooltip = "Open approved reports"
			case "Declined":
				kpi.Link = buildReportHistoryPeriodFilterQuery("declined", view.SelectedYear, view.SelectedMonth, view.SelectedWeek)
				kpi.Tooltip = "Open declined reports"
			case "Submission Progress", "First-Pass Approval Rate":
				kpi.Link = buildReportHistoryPeriodFilterQuery("all", view.SelectedYear, view.SelectedMonth, view.SelectedWeek)
				kpi.Tooltip = "Open my report history"
			case "Patients per Day", "Procedures per Day":
				kpi.Link = activityURL
				kpi.Tooltip = "Open activity breakdown table"
			case "Days Worked", "Ward Rounds", "Patients Reviewed", "Procedures":
				kpi.Link = activityURL
				kpi.Tooltip = "Open activity breakdown table"
			}
		}

		if kpi.Link == "" {
			if view.Role == utilities.RoleNationalAdmin && departmentsURL != "" {
				kpi.Link = departmentsURL
			} else if view.Role == utilities.RoleFacilityAdmin && staffURL != "" {
				kpi.Link = staffURL
			} else if view.Role == utilities.RoleStaff && benchmarksURL != "" {
				kpi.Link = benchmarksURL
			}
		}

		if strings.TrimSpace(kpi.Tooltip) == "" {
			if strings.TrimSpace(kpi.Link) != "" {
				kpi.Tooltip = "Click to view detailed table for this KPI"
			} else {
				kpi.Tooltip = "Metric for the current dashboard filter scope"
			}
		}
	}
}

func dashboardTabURLForID(view HomeViewModel, tabID string) string {
	for _, tab := range view.Tabs {
		if tab.ID == tabID {
			return tab.URL
		}
	}
	if tabID == "" {
		return ""
	}
	return buildDashboardTabURL(view, tabID)
}

func withSummaryTab(section DashboardSummarySectionView, tabID string, tabLabel string) DashboardSummarySectionView {
	section.TabID = tabID
	section.TabLabel = tabLabel
	return section
}

func withDetailTab(section DashboardDetailSectionView, tabID string, tabLabel string) DashboardDetailSectionView {
	section.TabID = tabID
	section.TabLabel = tabLabel
	return section
}

func populateDashboardTabs(view *HomeViewModel) {
	tabs := []DashboardTabView{{ID: "overview", Label: "Overview"}}
	seen := map[string]bool{"overview": true}

	for _, section := range view.SummarySections {
		if section.TabID == "" || seen[section.TabID] {
			continue
		}
		tabs = append(tabs, DashboardTabView{ID: section.TabID, Label: section.TabLabel})
		seen[section.TabID] = true
	}
	for _, section := range view.DetailSections {
		if section.TabID == "" || seen[section.TabID] {
			continue
		}
		tabs = append(tabs, DashboardTabView{ID: section.TabID, Label: section.TabLabel})
		seen[section.TabID] = true
	}

	for i := range tabs {
		tabs[i].URL = buildDashboardTabURL(*view, tabs[i].ID)
	}

	view.Tabs = tabs
	if !seen[view.ActiveTab] {
		view.ActiveTab = "overview"
	}
}

func buildDashboardTabURL(view HomeViewModel, tabID string) string {
	values := url.Values{}
	values.Set("tab", tabID)
	values.Set("year", strconv.Itoa(view.SelectedYear))
	values.Set("month", strconv.Itoa(view.SelectedMonth))
	values.Set("week", strconv.Itoa(view.SelectedWeek))

	if view.SelectedFacility >= 0 {
		values.Set("facility", strconv.Itoa(view.SelectedFacility))
	}
	if view.SelectedDepartment >= 0 {
		values.Set("department", strconv.Itoa(view.SelectedDepartment))
	}
	if view.SelectedEmployee >= 0 {
		values.Set("employee", strconv.Itoa(view.SelectedEmployee))
	}

	return "/?" + values.Encode()
}

func buildDashboardClearURL(view HomeViewModel) string {
	resetView := view
	resetView.SelectedYear = 0
	resetView.SelectedMonth = 0
	resetView.SelectedWeek = 0
	resetView.SelectedDepartment = 0
	resetView.SelectedEmployee = 0
	if view.Role == utilities.RoleNationalAdmin {
		resetView.SelectedFacility = 0
	}
	if resetView.ActiveTab == "" {
		resetView.ActiveTab = "overview"
	}
	return buildDashboardTabURL(resetView, resetView.ActiveTab)
}

func makeChartID(prefix string, title string) string {
	slug := strings.ToLower(title)
	replacer := strings.NewReplacer(" ", "-", "/", "-", "_", "-", ",", "", ".", "", "(", "", ")", "")
	slug = replacer.Replace(slug)
	slug = strings.Trim(slug, "-")
	if slug == "" {
		slug = "chart"
	}
	return prefix + "-" + slug
}

func buildScopeTrendChart(title string, subtitle string, points []models.ScopeTrendPoint) DashboardChartView {
	chart := DashboardChartView{
		ID:           makeChartID("scope", title),
		Title:        title,
		Subtitle:     subtitle,
		ChartType:    "line",
		SeriesALabel: "Submitted",
		SeriesBLabel: "Pending Approval",
	}
	for _, point := range points {
		chart.Labels = append(chart.Labels, point.WeekLabel)
		chart.SeriesA = append(chart.SeriesA, point.SubmittedCount)
		chart.SeriesB = append(chart.SeriesB, point.PendingCount)
	}
	return chart
}

func buildFacilityComparisonChart(title string, subtitle string, rows []models.FacilityPerformanceRow) DashboardChartView {
	chart := DashboardChartView{
		ID:           makeChartID("facility", title),
		Title:        title,
		Subtitle:     subtitle,
		ChartType:    "bar",
		SeriesALabel: "Submitted",
		SeriesBLabel: "Missing",
	}
	for _, row := range rows {
		chart.Labels = append(chart.Labels, row.FacilityName)
		chart.SeriesA = append(chart.SeriesA, row.SubmittedThisWeek)
		chart.SeriesB = append(chart.SeriesB, row.PendingThisWeek)
	}
	return chart
}

func buildDepartmentProgressChart(title string, subtitle string, rows []models.DepartmentProgressRow) DashboardChartView {
	chart := DashboardChartView{
		ID:           makeChartID("department-progress", title),
		Title:        title,
		Subtitle:     subtitle,
		ChartType:    "bar",
		SeriesALabel: "Entered",
		SeriesBLabel: "Submitted",
	}
	for _, row := range rows {
		chart.Labels = append(chart.Labels, row.DepartmentName)
		chart.SeriesA = append(chart.SeriesA, row.EnteredThisWeek)
		chart.SeriesB = append(chart.SeriesB, row.SubmittedThisWeek)
	}
	return chart
}

func buildDepartmentWorkloadChart(title string, subtitle string, rows []models.DepartmentAnalysisRow) DashboardChartView {
	chart := DashboardChartView{
		ID:           makeChartID("department-workload", title),
		Title:        title,
		Subtitle:     subtitle,
		ChartType:    "bar",
		SeriesALabel: "Patients Reviewed",
		SeriesBLabel: "Ward Rounds",
	}
	for _, row := range rows {
		chart.Labels = append(chart.Labels, row.DepartmentName)
		chart.SeriesA = append(chart.SeriesA, row.PatientsReviewed)
		chart.SeriesB = append(chart.SeriesB, row.WardRounds)
	}
	return chart
}

func buildDepartmentReportingChart(title string, subtitle string, rows []models.DepartmentAnalysisRow) DashboardChartView {
	chart := DashboardChartView{
		ID:           makeChartID("department-reporting", title),
		Title:        title,
		Subtitle:     subtitle,
		ChartType:    "bar",
		SeriesALabel: "Submitted",
		SeriesBLabel: "Pending",
	}
	for _, row := range rows {
		chart.Labels = append(chart.Labels, row.DepartmentName)
		chart.SeriesA = append(chart.SeriesA, row.ReportsSubmitted)
		chart.SeriesB = append(chart.SeriesB, row.PendingReports)
	}
	return chart
}

func buildClinicianPerformanceChart(title string, subtitle string, rows []models.ClinicianAnalysisRow) DashboardChartView {
	chart := DashboardChartView{
		ID:           makeChartID("clinician-performance", title),
		Title:        title,
		Subtitle:     subtitle,
		ChartType:    "bar",
		SeriesALabel: "Reports Submitted",
		SeriesBLabel: "Patients Reviewed",
	}
	for _, row := range rows {
		chart.Labels = append(chart.Labels, row.EmployeeName)
		chart.SeriesA = append(chart.SeriesA, row.ReportsSubmitted)
		chart.SeriesB = append(chart.SeriesB, row.PatientsReviewed)
	}
	return chart
}

func buildClinicianWorkloadChart(title string, subtitle string, rows []models.ClinicianTrendPoint) DashboardChartView {
	chart := DashboardChartView{
		ID:           makeChartID("clinician-workload", title),
		Title:        title,
		Subtitle:     subtitle,
		ChartType:    "line",
		SeriesALabel: "Patients Reviewed",
		SeriesBLabel: "Procedures",
	}
	for _, row := range rows {
		chart.Labels = append(chart.Labels, row.WeekLabel)
		chart.SeriesA = append(chart.SeriesA, row.PatientsReviewed)
		chart.SeriesB = append(chart.SeriesB, row.Procedures)
	}
	return chart
}

func buildClinicianActivityChart(title string, subtitle string, rows []models.ClinicianTrendPoint) DashboardChartView {
	chart := DashboardChartView{
		ID:           makeChartID("clinician-activity", title),
		Title:        title,
		Subtitle:     subtitle,
		ChartType:    "line",
		SeriesALabel: "Days Worked",
		SeriesBLabel: "Ward Rounds",
	}
	for _, row := range rows {
		chart.Labels = append(chart.Labels, row.WeekLabel)
		chart.SeriesA = append(chart.SeriesA, row.DaysWorked)
		chart.SeriesB = append(chart.SeriesB, row.WardRounds)
	}
	return chart
}

func buildClinicianPeriodSummarySection(view HomeViewModel) DashboardSummarySectionView {
	return DashboardSummarySectionView{
		Title:        "Selected Period Summary",
		Subtitle:     "A quick summary of your reporting and clinical workload for the current dashboard filter.",
		EmptyMessage: "No activity summary is available for this selection.",
		Rows: []DashboardSummaryRowView{
			{
				Cells: []DashboardSummaryCellView{
					{Label: "Period", Value: view.SelectedWeekLabel},
					{Label: "Attendance Rate", Value: findStatValue(view.Stats, "Attendance Rate")},
					{Label: "Submission Progress", Value: findStatValue(view.Stats, "Submission Progress")},
					{Label: "Total Reports", Value: findKPIValue(view.KPIs, "Total Reports")},
					{Label: "Submitted", Value: findKPIValue(view.KPIs, "Submitted")},
					{Label: "Ward Rounds", Value: findKPIValue(view.KPIs, "Ward Rounds")},
					{Label: "Patients Reviewed", Value: findKPIValue(view.KPIs, "Patients Reviewed")},
					{Label: "Procedures", Value: findKPIValue(view.KPIs, "Procedures")},
				},
			},
		},
	}
}

func buildClinicianBenchmarkSection(view HomeViewModel) DashboardDetailSectionView {
	return DashboardDetailSectionView{
		Title:        "Performance Benchmarks",
		Subtitle:     "Key performance signals for reporting consistency and clinical output in the selected period.",
		EmptyMessage: "No benchmark data is available for this selection.",
		Rows: []DashboardDetailRowView{
			{
				Title:    "Reporting Consistency",
				Subtitle: "Submission quality and reporting outcome indicators.",
				Metrics: []DashboardMetricView{
					{Label: "Total Reports", Value: findKPIValue(view.KPIs, "Total Reports")},
					{Label: "Submitted", Value: findKPIValue(view.KPIs, "Submitted")},
					{Label: "Approved", Value: findKPIValue(view.KPIs, "Approved")},
					{Label: "Declined", Value: findKPIValue(view.KPIs, "Declined")},
					{Label: "Submission Progress", Value: findStatValue(view.Stats, "Submission Progress")},
				},
			},
			{
				Title:    "Clinical Output",
				Subtitle: "Workload indicators captured from the reporting form for the current period.",
				Metrics: []DashboardMetricView{
					{Label: "Days Worked", Value: findKPIValue(view.KPIs, "Days Worked")},
					{Label: "Ward Rounds", Value: findKPIValue(view.KPIs, "Ward Rounds")},
					{Label: "Patients Reviewed", Value: findKPIValue(view.KPIs, "Patients Reviewed")},
					{Label: "Procedures", Value: findKPIValue(view.KPIs, "Procedures")},
					{Label: "Attendance Rate", Value: findStatValue(view.Stats, "Attendance Rate")},
				},
			},
		},
	}
}

func findKPIValue(items []DashboardKPIView, title string) string {
	for _, item := range items {
		if item.Title == title {
			return strconv.Itoa(item.Value)
		}
	}
	return "0"
}

func findStatValue(items []map[string]interface{}, title string) string {
	for _, item := range items {
		if item["Title"] == title {
			return fmt.Sprintf("%v", item["Value"])
		}
	}
	return "-"
}

func intOptionsToFilterOptions(values []int, includeAllLabel bool) []DashboardOptionView {
	options := make([]DashboardOptionView, 0, len(values))
	for _, value := range values {
		label := strconv.Itoa(value)
		if value == 0 && includeAllLabel {
			label = "All"
		}
		options = append(options, DashboardOptionView{
			Value: strconv.Itoa(value),
			Label: label,
		})
	}
	return options
}

func dashboardOptionsToFilterOptions(options []models.DashboardFilterOption) []DashboardOptionView {
	items := make([]DashboardOptionView, 0, len(options))
	for _, option := range options {
		items = append(items, DashboardOptionView{
			Value: strconv.Itoa(option.ID),
			Label: option.Name,
		})
	}
	return items
}

func dashboardFilterOptionsWithAll(options []models.DashboardFilterOption, allLabel string) []DashboardOptionView {
	items := []DashboardOptionView{{Value: "0", Label: allLabel}}
	for _, option := range options {
		if option.ID == 0 {
			continue
		}
		items = append(items, DashboardOptionView{
			Value: strconv.Itoa(option.ID),
			Label: option.Name,
		})
	}
	return items
}

func weekOptionsToFilterOptions(options []models.ClinicianWeekOption, includeAllLabel bool) []DashboardOptionView {
	items := make([]DashboardOptionView, 0, len(options))
	for _, option := range options {
		label := option.Label
		if option.Week == 0 && includeAllLabel {
			label = "All"
		}
		items = append(items, DashboardOptionView{
			Value: strconv.Itoa(option.Week),
			Label: label,
		})
	}
	return items
}

func buildNationalPerformanceSection(rows []models.FacilityPerformanceRow, periodLabel string) DashboardSummarySectionView {
	section := DashboardSummarySectionView{
		Title:        "Facility Reporting Performance",
		Subtitle:     fmt.Sprintf("Facility-by-facility reporting performance for %s.", periodLabel),
		EmptyMessage: "No facility performance records are available for this selection.",
	}

	for _, row := range rows {
		rateTone := "danger"
		if row.ReportingRate >= 80 {
			rateTone = "success"
		} else if row.ReportingRate >= 50 {
			rateTone = "warning"
		}

		lastSubmitted := "-"
		if row.LastSubmittedOn.Valid {
			lastSubmitted = utilities.HumanDate(row.LastSubmittedOn.Time)
		}

		section.Rows = append(section.Rows, DashboardSummaryRowView{
			Cells: []DashboardSummaryCellView{
				{Label: "Facility", Value: row.FacilityName, Subvalue: fmt.Sprintf("Facility ID %d", row.FacilityID)},
				{Label: "Clinicians", Value: strconv.Itoa(row.Clinicians)},
				{Label: "Entered", Value: strconv.Itoa(row.EnteredThisWeek)},
				{Label: "Submitted", Value: strconv.Itoa(row.SubmittedThisWeek)},
				{Label: "Missing", Value: strconv.Itoa(row.PendingThisWeek)},
				{Label: "Pending Approval", Value: strconv.Itoa(row.PendingApproval)},
				{Label: "Reporting Rate", Value: fmt.Sprintf("%d%%", row.ReportingRate), Badge: &DashboardBadgeView{Text: fmt.Sprintf("%d%%", row.ReportingRate), Tone: rateTone}},
				{Label: "Last Submitted", Value: lastSubmitted},
			},
		})
	}

	return section
}

func buildDepartmentFollowUpSection(rows []models.DepartmentProgressRow) DashboardSummarySectionView {
	section := DashboardSummarySectionView{
		Title:        "Department Follow-up Status",
		Subtitle:     "Department-level follow-up for review work inside your facility.",
		EmptyMessage: "No department records are available for this selection.",
	}

	for _, row := range rows {
		section.Rows = append(section.Rows, DashboardSummaryRowView{
			Cells: []DashboardSummaryCellView{
				{Label: "Department", Value: row.DepartmentName},
				{Label: "Clinicians", Value: strconv.Itoa(row.Clinicians)},
				{Label: "Entered", Value: strconv.Itoa(row.EnteredThisWeek)},
				{Label: "Submitted", Value: strconv.Itoa(row.SubmittedThisWeek)},
				{Label: "Pending", Value: strconv.Itoa(row.PendingThisWeek)},
				{Label: "Reporting Rate", Value: fmt.Sprintf("%d%%", row.ReportingRate)},
			},
		})
	}

	return section
}

func buildFacilityTopPerformanceSection(title string, subtitle string, rows []models.FacilityPerformanceRow) DashboardSummarySectionView {
	section := DashboardSummarySectionView{
		Title:        title,
		Subtitle:     subtitle,
		EmptyMessage: "No facility performance records are available for this selection.",
	}

	for _, row := range rows {
		section.Rows = append(section.Rows, DashboardSummaryRowView{
			Cells: []DashboardSummaryCellView{
				{Label: "Facility", Value: row.FacilityName},
				{Label: "Reporting Rate", Value: fmt.Sprintf("%d%%", row.ReportingRate)},
				{Label: "Submitted", Value: strconv.Itoa(row.SubmittedThisWeek)},
				{Label: "Missing", Value: strconv.Itoa(row.PendingThisWeek)},
				{Label: "Pending Approval", Value: strconv.Itoa(row.PendingApproval)},
			},
		})
	}

	return section
}

func buildDepartmentTopPerformanceSection(title string, subtitle string, rows []models.DepartmentAnalysisRow) DashboardSummarySectionView {
	section := DashboardSummarySectionView{
		Title:        title,
		Subtitle:     subtitle,
		EmptyMessage: "No department performance records are available for this selection.",
	}

	for _, row := range rows {
		section.Rows = append(section.Rows, DashboardSummaryRowView{
			Cells: []DashboardSummaryCellView{
				{Label: "Department", Value: row.DepartmentName},
				{Label: "Reporting Rate", Value: fmt.Sprintf("%d%%", row.ReportingRate)},
				{Label: "Submitted", Value: strconv.Itoa(row.ReportsSubmitted)},
				{Label: "Pending", Value: strconv.Itoa(row.PendingReports)},
				{Label: "Patients Reviewed", Value: strconv.Itoa(row.PatientsReviewed)},
			},
		})
	}

	return section
}

func buildClinicianTopPerformanceSection(title string, subtitle string, rows []models.ClinicianAnalysisRow) DashboardSummarySectionView {
	section := DashboardSummarySectionView{
		Title:        title,
		Subtitle:     subtitle,
		EmptyMessage: "No staff performance records are available for this selection.",
	}

	for _, row := range rows {
		section.Rows = append(section.Rows, DashboardSummaryRowView{
			Cells: []DashboardSummaryCellView{
				{Label: "Staff", Value: row.EmployeeName, Subvalue: row.DepartmentName},
				{Label: "Submitted", Value: strconv.Itoa(row.ReportsSubmitted)},
				{Label: "Patients Reviewed", Value: strconv.Itoa(row.PatientsReviewed)},
				{Label: "Procedures", Value: strconv.Itoa(row.Procedures)},
				{Label: "Status", Value: row.SubmissionStatus},
			},
		})
	}

	return section
}

func buildFacilityFocusSection(title string, subtitle string, rows []models.FacilityPerformanceRow) DashboardDetailSectionView {
	section := DashboardDetailSectionView{
		Title:        title,
		Subtitle:     subtitle,
		EmptyMessage: "No facility records are available for this view.",
	}

	for _, row := range rows {
		tone := "danger"
		if row.ReportingRate >= 80 {
			tone = "success"
		} else if row.ReportingRate >= 50 {
			tone = "warning"
		}

		section.Rows = append(section.Rows, DashboardDetailRowView{
			Title:    row.FacilityName,
			Subtitle: fmt.Sprintf("%d clinicians tracked this week", row.Clinicians),
			Badges:   []DashboardBadgeView{{Text: fmt.Sprintf("%d%% reporting rate", row.ReportingRate), Tone: tone}},
			Metrics: []DashboardMetricView{
				{Label: "Submitted", Value: strconv.Itoa(row.SubmittedThisWeek)},
				{Label: "Entered", Value: strconv.Itoa(row.EnteredThisWeek)},
				{Label: "Pending Approval", Value: strconv.Itoa(row.PendingApproval)},
				{Label: "Missing Submissions", Value: strconv.Itoa(row.PendingThisWeek)},
			},
		})
	}

	return section
}

func buildFacilitySupportSection(title string, subtitle string, rows []models.FacilityPerformanceRow) DashboardDetailSectionView {
	return buildFacilityFocusSection(title, subtitle, rows)
}

func buildDepartmentAnalysisSection(title string, subtitle string, rows []models.DepartmentAnalysisRow) DashboardDetailSectionView {
	section := DashboardDetailSectionView{
		Title:        title,
		Subtitle:     subtitle,
		EmptyMessage: "No department analysis records are available for this selection.",
	}

	for _, row := range rows {
		tone := "danger"
		if row.ReportingRate >= 80 {
			tone = "success"
		} else if row.ReportingRate >= 50 {
			tone = "warning"
		}

		section.Rows = append(section.Rows, DashboardDetailRowView{
			Title:    row.DepartmentName,
			Subtitle: departmentSubtitle(row),
			Badges:   []DashboardBadgeView{{Text: fmt.Sprintf("%d%% reporting rate", row.ReportingRate), Tone: tone}},
			Metrics: []DashboardMetricView{
				{Label: "Reports Entered", Value: strconv.Itoa(row.ReportsEntered)},
				{Label: "Reports Submitted", Value: strconv.Itoa(row.ReportsSubmitted)},
				{Label: "Pending Reports", Value: strconv.Itoa(row.PendingReports)},
				{Label: "Ward Rounds", Value: strconv.Itoa(row.WardRounds)},
				{Label: "Patients Reviewed", Value: strconv.Itoa(row.PatientsReviewed)},
				{Label: "Procedures", Value: strconv.Itoa(row.Procedures)},
				{Label: "Avg Patients / Report", Value: strconv.Itoa(row.AvgPatients)},
				{Label: "Avg Procedures / Report", Value: strconv.Itoa(row.AvgProcedures)},
				{Label: "Attendance Records", Value: strconv.Itoa(row.AttendanceRecords)},
			},
		})
	}

	return section
}

func buildClinicianAnalysisSection(title string, subtitle string, rows []models.ClinicianAnalysisRow) DashboardDetailSectionView {
	section := DashboardDetailSectionView{
		Title:        title,
		Subtitle:     subtitle,
		EmptyMessage: "No clinician records are available for this selection.",
	}

	for _, row := range rows {
		submissionTone := "warning"
		switch row.SubmissionStatus {
		case "Approved":
			submissionTone = "success"
		case "Submitted":
			submissionTone = "info"
		case "Declined":
			submissionTone = "danger"
		}
		leaveTone := "success"
		if row.LeaveStatus == "On Leave" {
			leaveTone = "warning"
		}

		subtitleText := "Clinician"
		if row.Title != "" {
			subtitleText = row.Title
		}
		if row.DepartmentName != "" {
			subtitleText += " - " + row.DepartmentName
		}
		if row.FacilityName != "" {
			subtitleText += " - " + row.FacilityName
		}

		section.Rows = append(section.Rows, DashboardDetailRowView{
			Title:    row.EmployeeName,
			Subtitle: subtitleText,
			Badges: []DashboardBadgeView{
				{Text: row.SubmissionStatus, Tone: submissionTone},
				{Text: row.LeaveStatus, Tone: leaveTone},
			},
			Metrics: []DashboardMetricView{
				{Label: "Reports Entered", Value: strconv.Itoa(row.ReportsEntered)},
				{Label: "Reports Submitted", Value: strconv.Itoa(row.ReportsSubmitted)},
				{Label: "Attendance Records", Value: strconv.Itoa(row.AttendanceRecords)},
				{Label: "Ward Rounds", Value: strconv.Itoa(row.WardRounds)},
				{Label: "Patients Reviewed", Value: strconv.Itoa(row.PatientsReviewed)},
				{Label: "Procedures", Value: strconv.Itoa(row.Procedures)},
			},
		})
	}

	return section
}

func departmentSubtitle(row models.DepartmentAnalysisRow) string {
	parts := []string{fmt.Sprintf("%d clinicians tracked", row.Clinicians)}
	if row.FacilityName != "" {
		parts = append(parts, row.FacilityName)
	}
	return strings.Join(parts, " - ")
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

func ratioPerUnit(numerator int, denominator int) int {
	if denominator <= 0 {
		return 0
	}
	value := int(float64(numerator)/float64(denominator) + 0.5)
	if value < 0 {
		return 0
	}
	return value
}

func summarizeClinicianStatus(rows []models.ClinicianAnalysisRow) (approved int, declined int, onLeave int) {
	for _, row := range rows {
		switch strings.ToLower(strings.TrimSpace(row.SubmissionStatus)) {
		case "approved":
			approved++
		case "declined":
			declined++
		}
		if strings.EqualFold(strings.TrimSpace(row.LeaveStatus), "On Leave") {
			onLeave++
		}
	}
	return approved, declined, onLeave
}

func countCliniciansOnLeave(rows []models.ClinicianAnalysisRow) int {
	_, _, onLeave := summarizeClinicianStatus(rows)
	return onLeave
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

func topDepartments(rows []models.DepartmentAnalysisRow, limit int) []models.DepartmentAnalysisRow {
	if len(rows) == 0 || limit <= 0 {
		return nil
	}

	cloned := append([]models.DepartmentAnalysisRow(nil), rows...)
	sort.SliceStable(cloned, func(i, j int) bool {
		if cloned[i].ReportingRate != cloned[j].ReportingRate {
			return cloned[i].ReportingRate > cloned[j].ReportingRate
		}
		if cloned[i].ReportsSubmitted != cloned[j].ReportsSubmitted {
			return cloned[i].ReportsSubmitted > cloned[j].ReportsSubmitted
		}
		if cloned[i].PatientsReviewed != cloned[j].PatientsReviewed {
			return cloned[i].PatientsReviewed > cloned[j].PatientsReviewed
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

func topClinicians(rows []models.ClinicianAnalysisRow, limit int) []models.ClinicianAnalysisRow {
	if len(rows) == 0 || limit <= 0 {
		return nil
	}

	cloned := append([]models.ClinicianAnalysisRow(nil), rows...)
	sort.SliceStable(cloned, func(i, j int) bool {
		leftApproved := 0
		if cloned[i].SubmissionStatus == "Approved" {
			leftApproved = 1
		}
		rightApproved := 0
		if cloned[j].SubmissionStatus == "Approved" {
			rightApproved = 1
		}
		if leftApproved != rightApproved {
			return leftApproved > rightApproved
		}
		if cloned[i].ReportsSubmitted != cloned[j].ReportsSubmitted {
			return cloned[i].ReportsSubmitted > cloned[j].ReportsSubmitted
		}
		if cloned[i].PatientsReviewed != cloned[j].PatientsReviewed {
			return cloned[i].PatientsReviewed > cloned[j].PatientsReviewed
		}
		if cloned[i].Procedures != cloned[j].Procedures {
			return cloned[i].Procedures > cloned[j].Procedures
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

func resolveNationalDashboardPeriod(c *gin.Context, db *sql.DB, requestedYear int, hasYear bool, requestedMonth int, hasMonth bool, requestedWeek int, hasWeek bool) (int, int, int, []int, []models.DashboardFilterOption, []models.ClinicianWeekOption, string, time.Time, time.Time, error) {
	periods, err := models.GetDashboardReportPeriods(c.Request.Context(), db)
	if err != nil {
		return 0, 0, 0, nil, nil, nil, "", time.Time{}, time.Time{}, err
	}

	if len(periods) == 0 {
		periods = []time.Time{startOfWeekLocal(time.Now())}
	}

	latestStart := periods[0]
	defaultYear, defaultWeek := latestStart.ISOWeek()
	defaultMonth := int(latestStart.Month())

	yearSeen := map[int]bool{}
	availableYears := []int{0}
	for _, period := range periods {
		year, _ := period.ISOWeek()
		if yearSeen[year] {
			continue
		}
		yearSeen[year] = true
		availableYears = append(availableYears, year)
	}
	sort.SliceStable(availableYears[1:], func(i, j int) bool { return availableYears[i+1] > availableYears[j+1] })

	selectedYear := defaultYear
	if hasYear {
		selectedYear = requestedYear
	}
	if selectedYear != 0 && !containsIntLocal(availableYears, selectedYear) {
		selectedYear = defaultYear
	}

	monthSeen := map[int]bool{}
	availableMonths := []models.DashboardFilterOption{{ID: 0, Name: "All"}}
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
	sort.SliceStable(availableMonths[1:], func(i, j int) bool { return availableMonths[i+1].ID > availableMonths[j+1].ID })

	selectedMonth := defaultMonth
	if hasMonth {
		selectedMonth = requestedMonth
	}
	if selectedMonth != 0 && !containsMonthOption(availableMonths, selectedMonth) {
		if !hasMonth && selectedYear == defaultYear {
			selectedMonth = defaultMonth
		}
		if !containsMonthOption(availableMonths, selectedMonth) && len(availableMonths) > 0 {
			selectedMonth = 0
		}
	}
	if selectedYear == 0 && !hasMonth {
		selectedMonth = 0
	}

	availableWeeks := []models.ClinicianWeekOption{{
		Year:  0,
		Week:  0,
		Label: "All",
	}}
	for _, period := range periods {
		year, week := period.ISOWeek()
		if selectedYear > 0 && year != selectedYear {
			continue
		}
		if selectedMonth > 0 && int(period.Month()) != selectedMonth {
			continue
		}

		availableWeeks = append(availableWeeks, models.ClinicianWeekOption{
			Year:      year,
			Week:      week,
			StartDate: period.Format("2006-01-02"),
			EndDate:   period.AddDate(0, 0, 6).Format("2006-01-02"),
			Label:     fmt.Sprintf("Week %02d (%s - %s)", week, period.Format("02 Jan"), period.AddDate(0, 0, 6).Format("02 Jan")),
		})
	}
	sort.SliceStable(availableWeeks, func(i, j int) bool {
		if i == 0 || j == 0 {
			return i < j
		}
		return availableWeeks[i].StartDate > availableWeeks[j].StartDate
	})

	if len(availableWeeks) == 1 {
		start := latestStart
		year, week := start.ISOWeek()
		availableWeeks = append(availableWeeks, models.ClinicianWeekOption{
			Year:      year,
			Week:      week,
			StartDate: start.Format("2006-01-02"),
			EndDate:   start.AddDate(0, 0, 6).Format("2006-01-02"),
			Label:     fmt.Sprintf("Week %02d (%s - %s)", week, start.Format("02 Jan"), start.AddDate(0, 0, 6).Format("02 Jan")),
		})
		selectedYear = year
		if !hasMonth {
			selectedMonth = int(start.Month())
		}
	}

	selectedWeek := defaultWeek
	if hasWeek {
		selectedWeek = requestedWeek
	}
	var selectedWeekOption models.ClinicianWeekOption
	var ok bool
	if selectedWeek > 0 {
		selectedWeekOption, ok = resolveWeekSelectionLocal(availableWeeks[1:], selectedWeek)
		if !ok {
			selectedWeek = 0
		}
	}
	if !hasWeek {
		if selectedYear == defaultYear && selectedMonth == defaultMonth {
			selectedWeek = defaultWeek
			selectedWeekOption, ok = resolveWeekSelectionLocal(availableWeeks[1:], selectedWeek)
		}
		if !ok && len(availableWeeks) > 1 {
			selectedWeekOption = availableWeeks[1]
			selectedWeek = selectedWeekOption.Week
		}
	}

	filteredPeriods := []time.Time{}
	for _, period := range periods {
		year, week := period.ISOWeek()
		if selectedYear > 0 && year != selectedYear {
			continue
		}
		if selectedMonth > 0 && int(period.Month()) != selectedMonth {
			continue
		}
		if selectedWeek > 0 && week != selectedWeek {
			continue
		}
		filteredPeriods = append(filteredPeriods, period)
	}
	if len(filteredPeriods) == 0 {
		filteredPeriods = []time.Time{latestStart}
	}

	periodStart := filteredPeriods[len(filteredPeriods)-1]
	periodEnd := filteredPeriods[0].AddDate(0, 0, 6)
	selectedWeekLabel := "Across all records"
	switch {
	case selectedYear > 0 && selectedMonth > 0 && selectedWeek > 0:
		selectedWeekLabel = fmt.Sprintf(
			"Week %02d, %d (%s - %s)",
			selectedWeekOption.Week,
			selectedWeekOption.Year,
			mustParseDateLocal(selectedWeekOption.StartDate).Format("02 Jan 2006"),
			mustParseDateLocal(selectedWeekOption.EndDate).Format("02 Jan 2006"),
		)
	case selectedYear > 0 && selectedMonth > 0:
		selectedWeekLabel = fmt.Sprintf("All weeks in %s %d", time.Month(selectedMonth).String(), selectedYear)
	case selectedYear > 0:
		selectedWeekLabel = fmt.Sprintf("All months in %d", selectedYear)
	case selectedMonth > 0:
		selectedWeekLabel = fmt.Sprintf("All years in %s", time.Month(selectedMonth).String())
	}

	return selectedYear, selectedMonth, selectedWeek, availableYears, availableMonths, availableWeeks, selectedWeekLabel, periodStart, periodEnd, nil
}

func containsMonthOption(options []models.DashboardFilterOption, target int) bool {
	for _, option := range options {
		if option.ID == target {
			return true
		}
	}
	return false
}

func containsIntLocal(values []int, target int) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

func resolveWeekSelectionLocal(options []models.ClinicianWeekOption, selectedWeek int) (models.ClinicianWeekOption, bool) {
	if len(options) == 0 {
		return models.ClinicianWeekOption{}, false
	}

	for _, option := range options {
		if option.Week == selectedWeek {
			return option, true
		}
	}

	return options[0], true
}

func startOfWeekLocal(value time.Time) time.Time {
	weekday := int(value.Weekday())
	if weekday == 0 {
		weekday = 7
	}

	year, month, day := value.Date()
	return time.Date(year, month, day-(weekday-1), 0, 0, 0, 0, value.Location())
}

func mustParseDateLocal(value string) time.Time {
	parsed, err := time.Parse("2006-01-02", value)
	if err != nil {
		return time.Time{}
	}
	return parsed
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
