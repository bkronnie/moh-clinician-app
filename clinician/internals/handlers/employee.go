package handlers

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/alexedwards/scs/v2"
	"github.com/gin-gonic/gin"
	"github.com/moh/clinician/internals/models"
	"github.com/moh/clinician/internals/utilities"
)

type LeaveFormView struct {
	IsEdit       bool
	ActionURL    string
	SubmitLabel  string
	MinStartDate string
	Leave        *models.LeaveHistory
}

type LeaveHistoryView struct {
	Rows         []*models.LeaveHistory
	ExportCSVURL string
	ExportPDFURL string
	BackURL      string
}

type FacilityLeaveReviewView struct {
	Rows         []*models.FacilityLeaveReviewRow
	FilterStatus string
	FilterTitle  string
	AllURL       string
	PendingURL   string
	OnLeaveURL   string
	CurrentURL   string
	ExportCSVURL string
	ExportPDFURL string
}

type EmployeePageView struct {
	ActiveTab          string
	CurrentURL         string
	DashboardURL       string
	ListURL            string
	OnDutyURL          string
	OnLeaveURL         string
	ClearURL           string
	Message            string
	AddStaffURL        string
	Dashboard          models.DashboardData
	TopEmployees       []*models.Employee
	Employees          []*models.Employee
	OnDuty             []*models.Employee
	OnLeave            []*models.Employee
	Facilities         []*models.Facility
	Departments        []*models.Department
	SelectedFacility   int64
	SelectedDepartment int64
	SearchTerm         string
	ShowFacilityFilter bool
}

func normalizeEmployeeTab(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "list", "on_duty", "on_leave":
		return strings.ToLower(strings.TrimSpace(value))
	default:
		return "list"
	}
}

func buildEmployeeTabURL(tab string, facilityID, departmentID int64, searchTerm string, includeFacility bool) string {
	params := []string{"tab=" + tab}
	if includeFacility && facilityID > 0 {
		params = append(params, fmt.Sprintf("facility=%d", facilityID))
	}
	if departmentID > 0 {
		params = append(params, fmt.Sprintf("department=%d", departmentID))
	}
	trimmedSearch := strings.TrimSpace(searchTerm)
	if trimmedSearch != "" {
		params = append(params, "search="+url.QueryEscape(trimmedSearch))
	}
	return "/employee/list?" + strings.Join(params, "&")
}

// HandlerEmployeeForm renders the HTML form for adding a new employee with session data
func HandlerEmployeeForm(c *gin.Context, db *sql.DB, sessionManager *scs.SessionManager) {
	facilities, departments, err := models.GetFacilitiesAndDepartments(db)
	if err != nil {
		log.Printf("Error loading facilities or departments: %v", err)
		c.String(http.StatusInternalServerError, "Error loading facilities or departments")
		return
	}

	viewData := gin.H{
		"facilities":  facilities,
		"departments": departments,
		"employee":    &models.Employee{},
		"formAction":  "/employee/save",
		"formTitle":   "New Employee",
		"submitLabel": "Save Employee",
	}

	data := Get_Session_Data(c, db, sessionManager, viewData)
	utilities.GenerateHTML(c, data, "base", "employeeform")
}

func HandlerEmployeeEdit(c *gin.Context, db *sql.DB, sessionManager *scs.SessionManager) {
	employeeID, err := strconv.Atoi(strings.TrimSpace(c.Param("id")))
	if err != nil {
		c.String(http.StatusBadRequest, "Invalid employee ID")
		return
	}

	employee, err := models.EmployeeByID(c.Request.Context(), db, employeeID)
	if err != nil {
		c.String(http.StatusNotFound, "Employee not found")
		return
	}

	facilities, departments, err := models.GetFacilitiesAndDepartments(db)
	if err != nil {
		log.Printf("Error loading facilities or departments: %v", err)
		c.String(http.StatusInternalServerError, "Error loading facilities or departments")
		return
	}

	viewData := gin.H{
		"facilities":  facilities,
		"departments": departments,
		"employee":    employee,
		"formAction":  "/employee/save",
		"formTitle":   "Update Employee",
		"submitLabel": "Update Employee",
	}

	data := Get_Session_Data(c, db, sessionManager, viewData)
	utilities.GenerateHTML(c, data, "base", "employeeform")
}

// Handle function to save [Employee]
func HandlerEmployeeSave(c *gin.Context, db *sql.DB, sessionManager *scs.SessionManager) {
	var emp models.Employee

	err := utilities.DecodeFormData(c, &emp)
	if err != nil {
		log.Printf("Error decoding employee form: %v", err)
		c.String(http.StatusBadRequest, "Invalid employee data")
		return
	}

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

	if emp.EmpID > 0 {
		existing, err := models.EmployeeByID(c.Request.Context(), db, int(emp.EmpID))
		if err != nil {
			log.Printf("Error loading existing employee: %v", err)
			c.String(http.StatusNotFound, "Employee not found")
			return
		}

		existing.Fname = emp.Fname
		existing.Lname = emp.Lname
		existing.Oname = emp.Oname
		existing.EmpTitleID = emp.EmpTitleID
		existing.Specialisation = emp.Specialisation
		existing.EmpDepartment = emp.EmpDepartment
		existing.EmpFacility = emp.EmpFacility

		if err := existing.Update(c.Request.Context(), db); err != nil {
			log.Printf("Error updating employee: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update employee"})
			return
		}
	} else {
		emp.CreatedBy.Int64 = sesDetails.EmpID
		emp.CreatedBy.Valid = true
		emp.CreatedOn.Valid = true
		emp.CreatedOn.Time = time.Now()

		if err := emp.Insert(c.Request.Context(), db); err != nil {
			log.Printf("Error saving new employee: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save employee"})
			return
		}
	}

	c.Redirect(http.StatusFound, "/employee/list")
}

func HandlerEmployeeDelete(c *gin.Context, db *sql.DB, sessionManager *scs.SessionManager) {
	employeeID, err := strconv.Atoi(strings.TrimSpace(c.Param("id")))
	if err != nil {
		c.String(http.StatusBadRequest, "Invalid employee ID")
		return
	}

	employee, err := models.EmployeeByID(c.Request.Context(), db, employeeID)
	if err != nil {
		c.String(http.StatusNotFound, "Employee not found")
		return
	}

	if err := employee.Delete(c.Request.Context(), db); err != nil {
		log.Printf("Error deleting employee: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete employee"})
		return
	}

	c.Redirect(http.StatusFound, "/employee/list")
}

func HandlerEmployeeList(c *gin.Context, db *sql.DB, sessionManager *scs.SessionManager) {
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

	activeTab := normalizeEmployeeTab(c.DefaultQuery("tab", "list"))
	selectedFacilityInt, _ := parseOptionalIntQuery(c, "facility")
	selectedDepartmentInt, _ := parseOptionalIntQuery(c, "department")
	selectedFacility := int64(selectedFacilityInt)
	selectedDepartment := int64(selectedDepartmentInt)
	searchTerm := strings.TrimSpace(c.Query("search"))
	showFacilityFilter := utilities.RoleMatches(sesDetails.Rights, "National Admin")

	if !showFacilityFilter {
		selectedFacility = sesDetails.HFID
	}

	facilities, departments, err := models.GetFacilitiesAndDepartments(db)
	if err != nil {
		log.Printf("Error loading facility or department lists: %v", err)
		c.String(http.StatusInternalServerError, "Error loading filter options")
		return
	}

	if !showFacilityFilter {
		facilities = []*models.Facility{{FacilityID: selectedFacility, FacilityName: sesDetails.HFName}}
	}

	dashboard, err := models.GetDashboardData(db, int(selectedFacility))
	if err != nil {
		log.Printf("Error building employee dashboard: %v", err)
		c.String(http.StatusInternalServerError, "Error building employee dashboard")
		return
	}

	if !showFacilityFilter {
		dashboard.FacilitiesCount = 1
	}

	employees, err := models.Employees(c.Request.Context(), db, selectedFacility, "", selectedDepartment, searchTerm, 0)
	if err != nil {
		log.Printf("Error retrieving employee directory: %v", err)
		employees = nil
	} else {
		log.Printf("employee list tab=%s facility=%d department=%d search=%q rows=%d", activeTab, selectedFacility, selectedDepartment, searchTerm, len(employees))
	}

	onDutyEmployees, err := models.Employees(c.Request.Context(), db, selectedFacility, "on_duty", selectedDepartment, searchTerm, 0)
	if err != nil {
		log.Printf("Error retrieving on duty employees: %v", err)
		onDutyEmployees = nil
	}

	onLeaveEmployees, err := models.Employees(c.Request.Context(), db, selectedFacility, "on_leave", selectedDepartment, searchTerm, 0)
	if err != nil {
		log.Printf("Error retrieving on leave employees: %v", err)
		onLeaveEmployees = nil
	}

	topEmployees, err := models.Employees(c.Request.Context(), db, selectedFacility, "on_duty", 0, "", 5)
	if err != nil {
		log.Printf("Error retrieving top performing employees: %v", err)
		topEmployees = nil
	}

	view := EmployeePageView{
		ActiveTab:          activeTab,
		CurrentURL:         buildEmployeeTabURL(activeTab, selectedFacility, selectedDepartment, searchTerm, showFacilityFilter),
		ListURL:            buildEmployeeTabURL("list", selectedFacility, selectedDepartment, searchTerm, showFacilityFilter),
		OnDutyURL:          buildEmployeeTabURL("on_duty", selectedFacility, selectedDepartment, searchTerm, showFacilityFilter),
		OnLeaveURL:         buildEmployeeTabURL("on_leave", selectedFacility, selectedDepartment, searchTerm, showFacilityFilter),
		ClearURL:           buildEmployeeTabURL(activeTab, 0, 0, "", showFacilityFilter),
		Message:            strings.TrimSpace(c.Query("message")),
		AddStaffURL:        "/employee/staff/new",
		Dashboard:          dashboard,
		TopEmployees:       topEmployees,
		Employees:          employees,
		OnDuty:             onDutyEmployees,
		OnLeave:            onLeaveEmployees,
		Facilities:         facilities,
		Departments:        departments,
		SelectedFacility:   selectedFacility,
		SelectedDepartment: selectedDepartment,
		SearchTerm:         searchTerm,
		ShowFacilityFilter: showFacilityFilter,
	}

	data := Get_Session_Data(c, db, sessionManager, view)
	utilities.GenerateHTML(c, data, "base", "employees")
}

// Handler fxn displays on leave staff list for facilities
func HandlerLeaveList(c *gin.Context, db *sql.DB, sessionManager *scs.SessionManager) {
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

	// Retrieve employees currently on leave using the updated filter signature
	employees, err := models.Employees(c.Request.Context(), db, hospitalID, "on_leave", 0, "", 0)
	if err != nil {
		log.Printf("Error retrieving employees: %v", err)
		employees = nil
	}

	// Get session data and populate template
	data := Get_Session_Data(c, db, sessionManager, employees)
	utilities.GenerateHTML(c, data, "base", "staffonleavelist")
}

// Handler fxn to display leave form for data entry
func HandlerEmployeeLeaveForm(c *gin.Context, db *sql.DB, sessionManager *scs.SessionManager) {
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

	formView := LeaveFormView{
		ActionURL:    "/leave/save",
		SubmitLabel:  "Submit Leave Request",
		MinStartDate: time.Now().In(time.Local).Format("2006-01-02"),
	}

	if utilities.RoleMatches(sesDetails.Rights, "National Admin") || utilities.RoleMatches(sesDetails.Rights, "Facility Admin") {
		formView.ActionURL = "/employee/leave/save"
	}

	data := Get_Session_Data(c, db, sessionManager, formView)
	utilities.GenerateHTML(c, data, "base", "leaveform")
}

// Handler fxn to save leave data to [staffleave]
func HandlerEmployeeLeaveSave(c *gin.Context, db *sql.DB, sessionManager *scs.SessionManager) {
	leave := models.LeaveHistory{}

	// Get session-specific data (e.g., user ID and HFID)
	sessionData, ok := Get_Session_Data(c, db, sessionManager, nil).(utilities.TemplateData)
	if !ok {
		log.Println("Failed to retrieve session data as TemplateData")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Unable to retrieve session data"})
		return
	}

	// Access the SessionDetails from the TemplateData
	sesDetails, ok := sessionData.Ses.(utilities.SessionDetails)
	if !ok {
		log.Println("Failed to retrieve session details from TemplateData")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Unable to retrieve session details"})
		return
	}

	// Parse form data
	if err := c.Request.ParseForm(); err != nil {
		log.Println("Failed to parse form data:", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid form data"})
		return
	}

	var employeeID int64
	redirectPath := "/leave/history"
	leaveStatus := sql.NullString{String: "Pending", Valid: true}
	approvedBy := sql.NullInt64{}

	if utilities.RoleMatches(sesDetails.Rights, "National Admin") || utilities.RoleMatches(sesDetails.Rights, "Facility Admin") {
		employeeIDValue := strings.TrimSpace(c.PostForm("employee_id"))
		if employeeIDValue == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Employee ID is required"})
			return
		}

		parsedEmployeeID, err := strconv.ParseInt(employeeIDValue, 10, 64)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid employee ID"})
			return
		}

		employeeID = parsedEmployeeID
		redirectPath = "/employee/list"
		leaveStatus = sql.NullString{String: "Valid", Valid: true}
		approvedBy = sql.NullInt64{Int64: sesDetails.EmpID, Valid: true}
	} else {
		employeeID = sesDetails.EmpID
	}

	leaveTypeID, _ := strconv.ParseInt(c.PostForm("leave_type_id"), 10, 64)
	startDate, err := time.Parse("2006-01-02", c.PostForm("start_date"))
	if err != nil {
		log.Println("Invalid start date:", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid start date format"})
		return
	}
	endDate, err := time.Parse("2006-01-02", c.PostForm("end_date"))
	if err != nil {
		log.Println("Invalid end date:", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid end date format"})
		return
	}
	comments := c.PostForm("comments")

	if endDate.Before(startDate) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "End date cannot be before start date"})
		return
	}

	if utilities.RoleMatches(sesDetails.Rights, "Staff") {
		today := time.Now().In(time.Local)
		today = time.Date(today.Year(), today.Month(), today.Day(), 0, 0, 0, 0, today.Location())
		if startDate.Before(today) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Start date cannot be before today"})
			return
		}
	}

	// Assign values to leave object
	leave.EmpID.Int64 = employeeID
	leave.EmpID.Valid = true
	leave.LeaveTypeID.Int64 = leaveTypeID
	leave.LeaveTypeID.Valid = true
	leave.StartDate.Time = startDate
	leave.StartDate.Valid = true
	leave.EndDate.Time = endDate
	leave.EndDate.Valid = true
	if comments != "" {
		leave.Comments.String = comments
		leave.Comments.Valid = true
	}
	leave.SubmittedBy = approvedBy
	leave.SubmissionDate.Time = time.Now()
	leave.SubmissionDate.Valid = true
	leave.LeaveStatus = leaveStatus

	// Log the data being passed to the InsertLeave function
	log.Printf("Data being passed to InsertLeave: %+v", leave)

	// Save leave data to the database
	if err := leave.InsertLeave(c.Request.Context(), db); err != nil {
		log.Println("Failed to insert leave data:", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to add leave record"})
		return
	}

	// Redirect back to the appropriate leave landing page
	c.Redirect(http.StatusFound, redirectPath)
}

// HandlerEmployeeLeaveEditForm renders the edit form for a pending clinician leave request.
func HandlerEmployeeLeaveEditForm(c *gin.Context, db *sql.DB, sessionManager *scs.SessionManager) {
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

	leaveID, err := strconv.Atoi(strings.TrimSpace(c.Param("id")))
	if err != nil {
		c.String(http.StatusBadRequest, "Invalid leave ID")
		return
	}

	leave, err := models.PendingLeaveByID(c.Request.Context(), db, leaveID, sesDetails.EmpID)
	if err != nil {
		c.String(http.StatusNotFound, "Pending leave request not found")
		return
	}

	formView := LeaveFormView{
		IsEdit:       true,
		ActionURL:    fmt.Sprintf("/leave/update/%d", leave.ID),
		SubmitLabel:  "Update Leave Request",
		MinStartDate: time.Now().In(time.Local).Format("2006-01-02"),
		Leave:        leave,
	}

	data := Get_Session_Data(c, db, sessionManager, formView)
	utilities.GenerateHTML(c, data, "base", "leaveform")
}

// HandlerEmployeeLeaveUpdate updates a clinician's own pending leave request.
func HandlerEmployeeLeaveUpdate(c *gin.Context, db *sql.DB, sessionManager *scs.SessionManager) {
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

	leaveID, err := strconv.Atoi(strings.TrimSpace(c.Param("id")))
	if err != nil {
		c.String(http.StatusBadRequest, "Invalid leave ID")
		return
	}

	if err := c.Request.ParseForm(); err != nil {
		log.Println("Failed to parse form data:", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid form data"})
		return
	}

	leaveTypeID, err := strconv.ParseInt(c.PostForm("leave_type_id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid leave type"})
		return
	}

	startDate, err := time.Parse("2006-01-02", c.PostForm("start_date"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid start date format"})
		return
	}

	endDate, err := time.Parse("2006-01-02", c.PostForm("end_date"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid end date format"})
		return
	}

	if endDate.Before(startDate) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "End date cannot be before start date"})
		return
	}

	today := time.Now().In(time.Local)
	today = time.Date(today.Year(), today.Month(), today.Day(), 0, 0, 0, 0, today.Location())
	if startDate.Before(today) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Start date cannot be before today"})
		return
	}

	leave := models.LeaveHistory{
		ID:          leaveID,
		EmpID:       sql.NullInt64{Int64: sesDetails.EmpID, Valid: true},
		LeaveTypeID: sql.NullInt64{Int64: leaveTypeID, Valid: true},
		StartDate:   sql.NullTime{Time: startDate, Valid: true},
		EndDate:     sql.NullTime{Time: endDate, Valid: true},
	}

	comments := strings.TrimSpace(c.PostForm("comments"))
	if comments != "" {
		leave.Comments = sql.NullString{String: comments, Valid: true}
	}

	updated, err := leave.UpdatePendingLeave(c.Request.Context(), db)
	if err != nil {
		log.Println("Failed to update leave data:", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update leave record"})
		return
	}

	if !updated {
		c.String(http.StatusForbidden, "Only pending leave requests can be edited")
		return
	}

	c.Redirect(http.StatusFound, "/leave/history")
}

// HandlerEmployeeLeaveDelete deletes a clinician's own pending leave request.
func HandlerEmployeeLeaveDelete(c *gin.Context, db *sql.DB, sessionManager *scs.SessionManager) {
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

	leaveID, err := strconv.Atoi(strings.TrimSpace(c.Param("id")))
	if err != nil {
		c.String(http.StatusBadRequest, "Invalid leave ID")
		return
	}

	deleted, err := models.DeletePendingLeave(c.Request.Context(), db, leaveID, sesDetails.EmpID)
	if err != nil {
		log.Println("Failed to delete leave data:", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete leave record"})
		return
	}

	if !deleted {
		c.String(http.StatusForbidden, "Only pending leave requests can be deleted")
		return
	}

	c.Redirect(http.StatusFound, "/leave/history")
}

// Handler fxn to retrieve leave history for a staff member
func HandlerEmployeeLeaveHistory(c *gin.Context, db *sql.DB, sessionManager *scs.SessionManager) {
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

	employeeID := int(sesDetails.EmpID)
	employeeIDStr := strings.TrimSpace(c.Param("id"))
	backURL := sanitizeEmployeeListURL(c.Query("return_to"))
	if employeeIDStr != "" {
		parsedEmployeeID, err := strconv.Atoi(employeeIDStr)
		if err != nil {
			c.String(http.StatusBadRequest, "Invalid employee ID")
			return
		}
		employeeID = parsedEmployeeID
	}

	// Fetch leave history from the database
	leaveHistory, err := models.GetHistory(db, employeeID)
	if err != nil {
		c.String(http.StatusInternalServerError, "Error retrieving leave history: %v", err)
		return
	}

	// Draft leave records are visible only to the owning staff member.
	if employeeIDStr != "" {
		filtered := make([]*models.LeaveHistory, 0, len(leaveHistory))
		for _, row := range leaveHistory {
			if row == nil || isDraftLeaveStatus(row.LeaveStatus) {
				continue
			}
			filtered = append(filtered, row)
		}
		leaveHistory = filtered
	}

	view := LeaveHistoryView{
		Rows:         leaveHistory,
		ExportCSVURL: buildLeaveHistoryExportURL("csv", employeeID, employeeIDStr != ""),
		ExportPDFURL: buildLeaveHistoryExportURL("pdf", employeeID, employeeIDStr != ""),
		BackURL:      backURL,
	}

	sessionData.Form = view
	utilities.GenerateHTML(c, sessionData, "base", "leavehistory")
}

func HandlerEmployeeLeaveHistoryExport(c *gin.Context, db *sql.DB, sessionManager *scs.SessionManager) {
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

	format := strings.ToLower(strings.TrimSpace(c.DefaultQuery("format", "csv")))
	if format != "csv" && format != "pdf" {
		c.String(http.StatusBadRequest, "Invalid export format")
		return
	}

	employeeID := int(sesDetails.EmpID)
	employeeIDStr := strings.TrimSpace(c.Param("id"))
	isEmployeeRoute := employeeIDStr != ""
	if isEmployeeRoute {
		parsedEmployeeID, err := strconv.Atoi(employeeIDStr)
		if err != nil {
			c.String(http.StatusBadRequest, "Invalid employee ID")
			return
		}
		employeeID = parsedEmployeeID
	}

	rows, err := models.GetHistory(db, employeeID)
	if err != nil {
		log.Printf("Error exporting leave history: %v", err)
		c.String(http.StatusInternalServerError, "Error exporting leave history")
		return
	}

	// Draft leave records are visible only to the owning staff member.
	if isEmployeeRoute {
		filtered := make([]*models.LeaveHistory, 0, len(rows))
		for _, row := range rows {
			if row == nil || isDraftLeaveStatus(row.LeaveStatus) {
				continue
			}
			filtered = append(filtered, row)
		}
		rows = filtered
	}

	headers := []string{"Application ID", "Leave Type", "Submitted", "Start Date", "End Date", "Status", "Comments"}
	csvRows := [][]string{}
	pdfLines := []string{
		fmt.Sprintf("Employee ID: %d", employeeID),
		"",
	}

	for index, row := range rows {
		status := exportLeaveStatus(row.LeaveStatus)
		csvRows = append(csvRows, []string{
			strconv.Itoa(row.ID),
			exportLeaveType(row.LeaveTypeName),
			exportDateTime(row.SubmissionDate),
			exportLeaveDate(row.StartDate),
			exportLeaveDate(row.EndDate),
			status,
			exportLeaveComment(row.Comments),
		})

		pdfLines = append(pdfLines,
			fmt.Sprintf("%d. %s", index+1, exportLeaveType(row.LeaveTypeName)),
			fmt.Sprintf("   Application ID: %d | Submitted: %s | Status: %s", row.ID, exportDateTime(row.SubmissionDate), status),
			fmt.Sprintf("   Period: %s - %s", exportLeaveDate(row.StartDate), exportLeaveDate(row.EndDate)),
			fmt.Sprintf("   Comments: %s", exportLeaveComment(row.Comments)),
			"",
		)
	}

	filenameBase := buildLeaveHistoryExportFilename(employeeID, isEmployeeRoute)
	if format == "pdf" {
		writeSimplePDFDownload(c, filenameBase+".pdf", "Leave History", pdfLines)
		return
	}
	writeCSVDownload(c, filenameBase+".csv", headers, csvRows)
}

func HandlerFacilityLeaveReview(c *gin.Context, db *sql.DB, sessionManager *scs.SessionManager) {
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

	filterStatus := strings.ToLower(strings.TrimSpace(c.DefaultQuery("status", "pending")))
	switch filterStatus {
	case "all", "pending", "on_leave":
	default:
		filterStatus = "pending"
	}
	selectedDepartment, _ := parseOptionalIntQuery(c, "department")
	selectedYear, _ := parseOptionalIntQuery(c, "year")
	selectedMonth, _ := parseOptionalIntQuery(c, "month")
	selectedWeek, _ := parseOptionalIntQuery(c, "week")

	rows, err := models.GetFacilityLeaveReview(
		c.Request.Context(),
		db,
		sesDetails.HFID,
		int64(selectedDepartment),
		selectedYear,
		selectedMonth,
		selectedWeek,
		filterStatus,
	)
	if err != nil {
		log.Printf("Error retrieving facility leave review data: %v", err)
		c.String(http.StatusInternalServerError, "Error retrieving leave requests")
		return
	}

	sessionData.Form = FacilityLeaveReviewView{
		Rows:         rows,
		FilterStatus: filterStatus,
		FilterTitle:  facilityLeaveReviewFilterTitle(filterStatus),
		AllURL:       buildFacilityLeaveReviewURL("all", selectedDepartment, selectedYear, selectedMonth, selectedWeek),
		PendingURL:   buildFacilityLeaveReviewURL("pending", selectedDepartment, selectedYear, selectedMonth, selectedWeek),
		OnLeaveURL:   buildFacilityLeaveReviewURL("on_leave", selectedDepartment, selectedYear, selectedMonth, selectedWeek),
		CurrentURL:   buildFacilityLeaveReviewURL(filterStatus, selectedDepartment, selectedYear, selectedMonth, selectedWeek),
		ExportCSVURL: buildFacilityLeaveReviewExportURL("csv", filterStatus, selectedDepartment, selectedYear, selectedMonth, selectedWeek),
		ExportPDFURL: buildFacilityLeaveReviewExportURL("pdf", filterStatus, selectedDepartment, selectedYear, selectedMonth, selectedWeek),
	}

	utilities.GenerateHTML(c, sessionData, "base", "facility-leave-review")
}

func HandlerFacilityLeaveReviewExport(c *gin.Context, db *sql.DB, sessionManager *scs.SessionManager) {
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

	format := strings.ToLower(strings.TrimSpace(c.DefaultQuery("format", "csv")))
	if format != "csv" && format != "pdf" {
		c.String(http.StatusBadRequest, "Invalid export format")
		return
	}

	filterStatus := strings.ToLower(strings.TrimSpace(c.DefaultQuery("status", "pending")))
	switch filterStatus {
	case "all", "pending", "on_leave":
	default:
		filterStatus = "pending"
	}
	selectedDepartment, _ := parseOptionalIntQuery(c, "department")
	selectedYear, _ := parseOptionalIntQuery(c, "year")
	selectedMonth, _ := parseOptionalIntQuery(c, "month")
	selectedWeek, _ := parseOptionalIntQuery(c, "week")

	rows, err := models.GetFacilityLeaveReview(
		c.Request.Context(),
		db,
		sesDetails.HFID,
		int64(selectedDepartment),
		selectedYear,
		selectedMonth,
		selectedWeek,
		filterStatus,
	)
	if err != nil {
		log.Printf("Error exporting facility leave review: %v", err)
		c.String(http.StatusInternalServerError, "Error exporting leave requests")
		return
	}

	headers := []string{"Employee", "Department", "Leave Type", "Leave Period", "Submitted", "Status", "Comments", "Actionable"}
	csvRows := [][]string{}
	pdfLines := []string{
		fmt.Sprintf("Facility: %s", sesDetails.HFName),
		fmt.Sprintf("Filter: %s", facilityLeaveReviewFilterTitle(filterStatus)),
		"",
	}

	for index, row := range rows {
		status := exportLeaveStatus(row.LeaveStatus)
		actionable := "No"
		if row.Actionable {
			actionable = "Yes"
		}
		period := exportWeekRange(row.StartDate, row.EndDate)

		csvRows = append(csvRows, []string{
			row.EmployeeName,
			row.DepartmentName,
			row.LeaveTypeName,
			period,
			exportDateTime(row.SubmissionDate),
			status,
			exportLeaveComment(row.Comments),
			actionable,
		})

		pdfLines = append(pdfLines,
			fmt.Sprintf("%d. %s - %s", index+1, row.EmployeeName, row.DepartmentName),
			fmt.Sprintf("   Leave Type: %s | Period: %s | Submitted: %s", row.LeaveTypeName, period, exportDateTime(row.SubmissionDate)),
			fmt.Sprintf("   Status: %s | Actionable: %s", status, actionable),
			fmt.Sprintf("   Comments: %s", exportLeaveComment(row.Comments)),
			"",
		)
	}

	filenameBase := buildFacilityLeaveReviewExportFilename(filterStatus)
	if format == "pdf" {
		writeSimplePDFDownload(c, filenameBase+".pdf", "Facility Leave Requests Review", pdfLines)
		return
	}
	writeCSVDownload(c, filenameBase+".csv", headers, csvRows)
}

func HandlerFacilityLeaveApprove(c *gin.Context, db *sql.DB, sessionManager *scs.SessionManager) {
	handleFacilityLeaveDecision(c, db, sessionManager, true)
}

func HandlerFacilityLeaveDecline(c *gin.Context, db *sql.DB, sessionManager *scs.SessionManager) {
	handleFacilityLeaveDecision(c, db, sessionManager, false)
}

func handleFacilityLeaveDecision(c *gin.Context, db *sql.DB, sessionManager *scs.SessionManager, approve bool) {
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

	leaveID, err := strconv.Atoi(strings.TrimSpace(c.Param("id")))
	if err != nil {
		c.String(http.StatusBadRequest, "Invalid leave ID")
		return
	}

	var updated bool
	if approve {
		updated, err = models.ApproveFacilityLeave(c.Request.Context(), db, leaveID, sesDetails.HFID, sesDetails.EmpID)
	} else {
		updated, err = models.DeclineFacilityLeave(c.Request.Context(), db, leaveID, sesDetails.HFID, sesDetails.EmpID)
	}
	if err != nil {
		log.Printf("Error updating leave review decision: %v", err)
		c.String(http.StatusInternalServerError, "Unable to update leave request")
		return
	}
	if !updated {
		c.String(http.StatusForbidden, "Only pending leave requests within your facility can be updated")
		return
	}

	c.Redirect(http.StatusFound, sanitizeFacilityLeaveReviewURL(c.PostForm("return_to")))
}

func buildFacilityLeaveReviewURL(status string, departmentID int, year int, month int, week int) string {
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
	if status != "" && status != "pending" {
		params = append(params, "status="+status)
	}
	if departmentID > 0 {
		params = append(params, fmt.Sprintf("department=%d", departmentID))
	}
	params = append(params, fmt.Sprintf("year=%d", year))
	params = append(params, fmt.Sprintf("month=%d", month))
	params = append(params, fmt.Sprintf("week=%d", week))
	if len(params) == 0 {
		return "/leave/review"
	}
	return "/leave/review?" + strings.Join(params, "&")
}

func buildLeaveHistoryExportURL(format string, employeeID int, isEmployeeRoute bool) string {
	base := "/leave/history/export"
	if isEmployeeRoute {
		base = fmt.Sprintf("/employee/%d/leave/history/export", employeeID)
	}
	if format == "" {
		return base
	}
	return base + "?format=" + format
}

func buildFacilityLeaveReviewExportURL(format, status string, departmentID int, year int, month int, week int) string {
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
	if format != "" {
		params = append(params, "format="+format)
	}
	if status != "" && status != "pending" {
		params = append(params, "status="+status)
	}
	if departmentID > 0 {
		params = append(params, fmt.Sprintf("department=%d", departmentID))
	}
	params = append(params, fmt.Sprintf("year=%d", year))
	params = append(params, fmt.Sprintf("month=%d", month))
	params = append(params, fmt.Sprintf("week=%d", week))
	if len(params) == 0 {
		return "/leave/review/export"
	}
	return "/leave/review/export?" + strings.Join(params, "&")
}

func buildLeaveHistoryExportFilename(employeeID int, isEmployeeRoute bool) string {
	if isEmployeeRoute {
		return fmt.Sprintf("employee-%d-leave-history", employeeID)
	}
	return "my-leave-history"
}

func buildFacilityLeaveReviewExportFilename(status string) string {
	name := "facility-leave-review"
	if status != "" && status != "pending" {
		name += "-" + status
	}
	return name
}

func facilityLeaveReviewFilterTitle(status string) string {
	switch status {
	case "pending":
		return "Pending Leave Requests"
	case "on_leave":
		return "Staff Currently On Leave"
	default:
		return "All Leave Requests"
	}
}

func isDraftLeaveStatus(value sql.NullString) bool {
	if !value.Valid {
		return true
	}
	trimmed := strings.TrimSpace(value.String)
	return trimmed == "" || strings.EqualFold(trimmed, "draft")
}

func sanitizeFacilityLeaveReviewURL(value string) string {
	if value == "" {
		return "/leave/review"
	}
	if strings.HasPrefix(value, "/leave/review") {
		return value
	}
	return "/leave/review"
}

func exportLeaveStatus(value sql.NullString) string {
	if !value.Valid {
		return "-"
	}
	switch value.String {
	case "Valid":
		return "Approved"
	case "Rejected":
		return "Declined"
	case "Expired":
		return "Completed"
	default:
		return value.String
	}
}

func sanitizeEmployeeListURL(value string) string {
	if value == "" {
		return "/employee/list"
	}
	if strings.HasPrefix(value, "/employee/list") || strings.HasPrefix(value, "/employee/leave/staffonleave") {
		return value
	}
	return "/employee/list"
}

func exportLeaveType(value sql.NullString) string {
	if !value.Valid || strings.TrimSpace(value.String) == "" {
		return "Office Leave"
	}
	return value.String
}

func exportLeaveDate(value sql.NullTime) string {
	if !value.Valid {
		return "-"
	}
	return value.Time.Format("02 Jan 2006")
}

func exportLeaveComment(value sql.NullString) string {
	if !value.Valid || strings.TrimSpace(value.String) == "" {
		return "-"
	}
	return value.String
}

// AddStaffView is the view model for the admin create-staff form.
type AddStaffView struct {
	Error          string
	Values         map[string]string
	Facilities     []models.RegistrationOption
	Departments    []models.RegistrationOption
	Titles         []models.SpecialistTitleOption
	FacilityFixed  bool
	ShowRolePicker bool
}

// HandlerAdminAddUserForm renders the admin form for creating a new staff account.
func HandlerAdminAddUserForm(c *gin.Context, db *sql.DB, sessionManager *scs.SessionManager) {
	sessionData, ok := Get_Session_Data(c, db, sessionManager, nil).(utilities.TemplateData)
	if !ok {
		c.String(http.StatusInternalServerError, "session error")
		return
	}
	sesDetails, ok := sessionData.Ses.(utilities.SessionDetails)
	if !ok {
		c.String(http.StatusInternalServerError, "session error")
		return
	}

	isNationalAdmin := utilities.RoleMatches(sesDetails.Rights, "National Admin")
	isAdmin := isNationalAdmin || utilities.RoleMatches(sesDetails.Rights, "Facility Admin")

	facilities, departments, titles, err := models.RegistrationOptions(c.Request.Context(), db)
	if err != nil {
		log.Printf("HandlerAdminAddUserForm: RegistrationOptions error: %v", err)
		c.String(http.StatusInternalServerError, "Error loading form options")
		return
	}

	if !isNationalAdmin {
		facilities = []models.RegistrationOption{{ID: sesDetails.HFID, Name: sesDetails.HFName}}
	}

	view := AddStaffView{
		Values:         map[string]string{},
		Facilities:     facilities,
		Departments:    departments,
		Titles:         titles,
		FacilityFixed:  !isNationalAdmin,
		ShowRolePicker: isAdmin,
	}

	data := Get_Session_Data(c, db, sessionManager, view)
	utilities.GenerateHTML(c, data, "base", "add-staff")
}

// HandlerAdminAddUserSave handles the POST form submission for admin-created staff accounts.
func HandlerAdminAddUserSave(c *gin.Context, db *sql.DB, sessionManager *scs.SessionManager) {
	sessionData, ok := Get_Session_Data(c, db, sessionManager, nil).(utilities.TemplateData)
	if !ok {
		c.String(http.StatusInternalServerError, "session error")
		return
	}
	sesDetails, ok := sessionData.Ses.(utilities.SessionDetails)
	if !ok {
		c.String(http.StatusInternalServerError, "session error")
		return
	}

	isNationalAdmin := utilities.RoleMatches(sesDetails.Rights, "National Admin")
	isAdmin := isNationalAdmin || utilities.RoleMatches(sesDetails.Rights, "Facility Admin")

	values := map[string]string{
		"firstname":      strings.TrimSpace(c.PostForm("firstname")),
		"lastname":       strings.TrimSpace(c.PostForm("lastname")),
		"othername":      strings.TrimSpace(c.PostForm("othername")),
		"employeeNumber": strings.TrimSpace(c.PostForm("employee_number")),
		"dateOfBirth":    strings.TrimSpace(c.PostForm("date_of_birth")),
		"phoneNumber":    strings.TrimSpace(c.PostForm("phone_number")),
		"email":          strings.TrimSpace(c.PostForm("email")),
		"specialisation": strings.TrimSpace(c.PostForm("specialisation")),
		"facility":       strings.TrimSpace(c.PostForm("facility_id")),
		"department":     strings.TrimSpace(c.PostForm("department_id")),
		"title":          strings.TrimSpace(c.PostForm("title_id")),
		"role":           strings.TrimSpace(c.PostForm("role")),
	}

	// Security: Facility Admin may only create accounts within their own facility.
	if !isNationalAdmin {
		values["facility"] = fmt.Sprintf("%d", sesDetails.HFID)
	}

	password := c.PostForm("password")
	confirmPassword := c.PostForm("confirm_password")

	renderError := func(errMsg string) {
		facilities, departments, titles, err := models.RegistrationOptions(c.Request.Context(), db)
		if err != nil {
			log.Printf("HandlerAdminAddUserSave renderError: RegistrationOptions error: %v", err)
			c.String(http.StatusInternalServerError, "Error loading form options")
			return
		}
		if !isNationalAdmin {
			facilities = []models.RegistrationOption{{ID: sesDetails.HFID, Name: sesDetails.HFName}}
		}
		view := AddStaffView{
			Error:          errMsg,
			Values:         values,
			Facilities:     facilities,
			Departments:    departments,
			Titles:         titles,
			FacilityFixed:  !isNationalAdmin,
			ShowRolePicker: isAdmin,
		}
		data := Get_Session_Data(c, db, sessionManager, view)
		utilities.GenerateHTML(c, data, "base", "add-staff")
	}

	if values["firstname"] == "" || values["lastname"] == "" || values["employeeNumber"] == "" || values["email"] == "" {
		renderError("First name, last name, medical officer ID, and email are required.")
		return
	}
	if password == "" {
		renderError("An initial password is required.")
		return
	}
	if password != confirmPassword {
		renderError("Passwords do not match.")
		return
	}

	facilityID, departmentID, titleID, parseErr := parseRegistrationIDs(values["facility"], values["department"], values["title"])
	if parseErr != nil {
		renderError(parseErr.Error())
		return
	}

	var dob *time.Time
	if values["dateOfBirth"] != "" {
		parsedDOB, err := time.Parse("2006-01-02", values["dateOfBirth"])
		if err != nil {
			renderError("Date of birth must use the YYYY-MM-DD format.")
			return
		}
		dob = &parsedDOB
	}

	roleName := "Staff"
	if isAdmin && values["role"] == "Admin" {
		// A Facility Admin creating "Admin" produces a Facility Admin account.
		// A National Admin creating "Admin" produces a National Admin account.
		if utilities.RoleMatches(sesDetails.Rights, "Facility Admin") {
			roleName = "Facility Admin"
		} else {
			roleName = "Admin"
		}
	}

	_, err := models.CreateClinicianAdminRegistration(c.Request.Context(), db, models.ClinicianRegistrationInput{
		FirstName:      values["firstname"],
		LastName:       values["lastname"],
		OtherName:      values["othername"],
		EmployeeNumber: values["employeeNumber"],
		DateOfBirth:    dob,
		PhoneNumber:    values["phoneNumber"],
		Email:          values["email"],
		PasswordHash:   models.Encrypt(password),
		FacilityID:     facilityID,
		DepartmentID:   departmentID,
		TitleID:        titleID,
		Specialisation: values["specialisation"],
	}, sesDetails.EmpID, roleName)
	if err != nil {
		renderError(err.Error())
		return
	}

	successMsg := url.QueryEscape(values["firstname"] + " " + values["lastname"] + " has been added successfully.")
	c.Redirect(http.StatusFound, "/employee/list?tab=list&message="+successMsg)
}
