package handlers

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/alexedwards/scs/v2"
	"github.com/gin-gonic/gin"
	"github.com/moh/clinician/internals/models"
	"github.com/moh/clinician/internals/security"
	"github.com/moh/clinician/internals/utilities"
)

type DashboardKPIView struct {
	Title    string
	Value    int
	Target   int
	Unit     string
	Progress int
}

type HomeViewModel struct {
	Role                string
	ScopeTitle          string
	ScopeSubtitle       string
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
	DataEntryURL        string
	LatestSubmission    string
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
		fmt.Printf("x3: " + err.Error())
	} else {
		ses.FName = emp.Fname.String
		ses.LName = emp.Lname.String
	}

	hf, err := models.FacilityByID(c, db, int(emp.EmpFacility))
	if err != nil {
		fmt.Printf("x2: " + err.Error())
	} else {
		ses.HFName = hf.FacilityName
		ses.HFID = int64(hf.FacilityID)
	}

	hd, err := models.DepartmentByID(c, db, int(emp.EmpDepartment))
	if err != nil {
		fmt.Printf("x2: " + err.Error())
	} else {
		//ses.HFName = hd.Department
		ses.HDID = int64(hd.DeptID)
	}

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
			fmt.Printf("x1: " + er.Error())
		} else {
			data.Ses = ses
		}

		data.Form = pData

		return data
	}
}

func buildHomeViewModel(c *gin.Context, db *sql.DB, ses utilities.SessionDetails) (HomeViewModel, error) {
	switch ses.Rights {
	case "admin":
		return buildNationalHome(c, db)
	case "approver":
		return buildFacilityManagerHome(c, db, int(ses.HFID), ses.HFName)
	default:
		return buildClinicianHome(c, db, int(ses.EmpID), ses.HFName)
	}
}

func buildNationalHome(c *gin.Context, db *sql.DB) (HomeViewModel, error) {
	snapshot, err := models.GetNationalDashboardSnapshot(c.Request.Context(), db)
	if err != nil {
		return HomeViewModel{}, err
	}

	view := HomeViewModel{
		Role:          "admin",
		ScopeTitle:    "National Performance Overview",
		ScopeSubtitle: "Monitor facility participation, submissions, and approval follow-up across the referral network.",
		SeriesTitle:   "Weekly Facility Submission Snapshot",
		SeriesALabel:  "Entered",
		SeriesBLabel:  "Submitted",
		PerformanceRows: snapshot.FacilityPerformance,
	}

	view.Stats = []map[string]interface{}{
		{"Title": "Referral Facilities", "Value": snapshot.TotalFacilities, "Tone": "stat-primary"},
		{"Title": "Clinicians Tracked", "Value": snapshot.TotalClinicians, "Tone": "stat-success"},
		{"Title": "Reports Entered This Week", "Value": snapshot.ReportsEnteredThisWeek, "Tone": "stat-info"},
		{"Title": "Pending Approval", "Value": snapshot.PendingApproval, "Tone": "stat-warning"},
	}

	for _, row := range snapshot.FacilityPerformance {
		view.SeriesLabels = append(view.SeriesLabels, row.FacilityName)
		view.SeriesA = append(view.SeriesA, row.EnteredThisWeek)
		view.SeriesB = append(view.SeriesB, row.SubmittedThisWeek)
	}

	return view, nil
}

func buildFacilityManagerHome(c *gin.Context, db *sql.DB, facilityID int, facilityName string) (HomeViewModel, error) {
	snapshot, err := models.GetFacilityManagerDashboardSnapshot(c.Request.Context(), db, facilityID)
	if err != nil {
		return HomeViewModel{}, err
	}

	view := HomeViewModel{
		Role:            "approver",
		ScopeTitle:      fmt.Sprintf("%s Follow-up Dashboard", facilityName),
		ScopeSubtitle:   "Track who has entered data, who has submitted, and which departments still need follow-up this week.",
		SeriesTitle:     "Department Submission Status",
		SeriesALabel:    "Entered",
		SeriesBLabel:    "Submitted",
		DepartmentRows:  snapshot.Departments,
	}

	view.Stats = []map[string]interface{}{
		{"Title": "Clinicians in Facility", "Value": snapshot.TotalClinicians, "Tone": "stat-primary"},
		{"Title": "Entered This Week", "Value": snapshot.ReportsEnteredThisWeek, "Tone": "stat-info"},
		{"Title": "Submitted This Week", "Value": snapshot.SubmittedThisWeek, "Tone": "stat-success"},
		{"Title": "Pending Follow-up", "Value": snapshot.PendingSubmission, "Tone": "stat-warning"},
	}

	for _, row := range snapshot.Departments {
		view.SeriesLabels = append(view.SeriesLabels, row.DepartmentName)
		view.SeriesA = append(view.SeriesA, row.EnteredThisWeek)
		view.SeriesB = append(view.SeriesB, row.SubmittedThisWeek)
	}

	return view, nil
}

func buildClinicianHome(c *gin.Context, db *sql.DB, employeeID int, facilityName string) (HomeViewModel, error) {
	snapshot, err := models.GetClinicianDashboardSnapshot(c.Request.Context(), db, employeeID)
	if err != nil {
		return HomeViewModel{}, err
	}

	view := HomeViewModel{
		Role:           "user",
		ScopeTitle:     "My Weekly KPI Dashboard",
		ScopeSubtitle:  "Review your recent performance, compare against targets, and open your weekly data entry form.",
		DataEntryURL:   "/reports/new/0",
		SeriesTitle:    "My Recent Reporting Trend",
		SeriesALabel:   "Patients Reviewed",
		SeriesBLabel:   "Procedures",
	}

	view.KPIs = []DashboardKPIView{
		{
			Title:    "Specialist Output",
			Value:    snapshot.PatientsReviewed,
			Target:   snapshot.OutputTarget,
			Unit:     "patients",
			Progress: cappedProgress(snapshot.PatientsReviewed, snapshot.OutputTarget),
		},
		{
			Title:    "Attendance Rate",
			Value:    snapshot.AttendanceRate,
			Target:   snapshot.AttendanceTarget,
			Unit:     "%",
			Progress: cappedProgress(snapshot.AttendanceRate, snapshot.AttendanceTarget),
		},
		{
			Title:    "Ward Round Coverage",
			Value:    snapshot.WardRounds,
			Target:   snapshot.WardRoundsTarget,
			Unit:     "rounds",
			Progress: cappedProgress(snapshot.WardRounds, snapshot.WardRoundsTarget),
		},
	}

	view.Stats = []map[string]interface{}{
		{"Title": "Weeks Entered", "Value": snapshot.WeeksEntered, "Tone": "stat-primary"},
		{"Title": "Weeks Submitted", "Value": snapshot.WeeksSubmitted, "Tone": "stat-success"},
		{"Title": "Patients Reviewed", "Value": snapshot.PatientsReviewed, "Tone": "stat-info"},
		{"Title": "Procedures Logged", "Value": snapshot.Procedures, "Tone": "stat-warning"},
	}

	for _, point := range snapshot.RecentWeeks {
		view.SeriesLabels = append(view.SeriesLabels, point.WeekLabel)
		view.SeriesA = append(view.SeriesA, point.PatientsReviewed)
		view.SeriesB = append(view.SeriesB, point.Procedures)
	}

	if snapshot.LatestWeekLabel != "" {
		status := "Not submitted yet"
		if snapshot.LatestWeekSubmitted {
			status = "Submitted"
		}
		view.LatestSubmission = fmt.Sprintf("%s: %s", snapshot.LatestWeekLabel, status)
	}

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
