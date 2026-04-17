package handlers

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/alexedwards/scs/v2"
	"github.com/gin-gonic/gin"
	"github.com/moh/clinician/internals/models"
	"github.com/moh/clinician/internals/utilities"
)

type LeaveFormView struct {
	IsEdit      bool
	ActionURL   string
	SubmitLabel string
	Leave       *models.LeaveHistory
}

type LeaveHistoryView struct {
	Rows         []*models.LeaveHistory
	ExportCSVURL string
	ExportPDFURL string
}

type FacilityLeaveReviewView struct {
	Rows         []*models.FacilityLeaveReviewRow
	FilterStatus string
	FilterTitle  string
	AllURL       string
	PendingURL   string
	ApprovedURL  string
	DeclinedURL  string
	CurrentURL   string
	ExportCSVURL string
	ExportPDFURL string
}

// HandlerEmployeeForm renders the HTML form for adding a new employee with session data
func HandlerEmployeeForm(c *gin.Context, db *sql.DB, sessionManager *scs.SessionManager) {
	// Get session-specific data (e.g., user ID)
	sessionData := Get_Session_Data(c, db, sessionManager, nil)
	log.Printf("Session Data HandlerEmployeeForm: %+v", sessionData)

	// Get facilities and departments from the database
	facilities, departments, err := models.GetFacilitiesAndDepartments(db)
	if err != nil {
		log.Printf("Error loading facilities or departments: %v", err)
		c.String(http.StatusInternalServerError, "Error loading facilities or departments")
		return
	}
	//log.Printf("Facilities: %+v", facilities)
	//log.Printf("Departments: %+v", departments)

	// Get the 'Created by' details using the userID from session data
	userID := sessionManager.GetString(c.Request.Context(), "UserID")
	log.Printf("User ID from session: %s", userID)

	// Prepare data to pass to the template
	data := gin.H{
		"sessionData": sessionData,
		"facilities":  facilities,
		"departments": departments,
		//"createdByName":     createdByName,
		//"createdByUsername": createdByUsername,
		//"createdByID":       createdByID, // in case we need to store or use this ID
	}

	//log.Printf("User ID from session: %d", userID)

	// Render the employee form template with data
	utilities.GenerateHTML(c, data, "base", "employeeform")
}

// Handle function to save [Employee]
func HandlerEmployeeSave(c *gin.Context, db *sql.DB, sessionManager *scs.SessionManager) {

	var emp = models.Employee{}

	//log.Printf("Title: %v, Department: %v, Facility: %v\n", emp.EmpTitle, emp.EmpDepartment, emp.EmpFacility)

	err := utilities.DecodeFormData(c, &emp)
	if err != nil {
		fmt.Print("" + err.Error())
	}

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

	empID := sesDetails.EmpID // Retrieve HFID from the session
	log.Printf("UserID in employee save: %+d", empID)

	//log.Printf("Decoded Employee Data - Title: %v, Department: %v, Facility: %v", emp.EmpTitle, emp.EmpDepartment, emp.EmpFacility)

	//log.Printf("Title: %v, Department: %v, Facility: %v\n", emp.EmpTitle, emp.EmpDepartment, emp.EmpFacility)
	emp.CreatedBy.Int64 = empID
	emp.CreatedBy.Valid = true
	emp.CreatedOn.Valid = true

	emp.CreatedOn.Time = time.Now()

	// Log the data being passed to the Insert new staff function
	log.Printf("Employee struct data: %+v", emp)

	err = emp.Insert(c, db)
	if err != nil {
		log.Fatal(err)
	}

	// Redirect back to the home page
	c.Redirect(http.StatusFound, "/employee/list")
}

// Handler fxn displays the staff list for facilities
func HandlerEmployeeList(c *gin.Context, db *sql.DB, sessionManager *scs.SessionManager) {
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

	// Retrieve employees data filtered by HFID
	employees, err := models.Employees(c.Request.Context(), db, hospitalID, "", 0, 0)
	if err != nil {
		log.Printf("Error retrieving employees: %v", err)
		employees = nil
	}

	// Get session data and populate template
	data := Get_Session_Data(c, db, sessionManager, employees)
	utilities.GenerateHTML(c, data, "base", "employees") // Renders "employees" template
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

	// Retrieve employees data filtered by HFID
	employees, err := models.Employees(c.Request.Context(), db, hospitalID, "Valid", 0, 0)
	if err != nil {
		log.Printf("Error retrieving employees: %v", err)
		employees = nil
	}

	// Get session data and populate template
	data := Get_Session_Data(c, db, sessionManager, employees)
	utilities.GenerateHTML(c, data, "base", "staffonleavelist") // Renders "employees" template
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
		ActionURL:   "/leave/save",
		SubmitLabel: "Submit Leave Request",
	}

	if sesDetails.Rights == "admin" || sesDetails.Rights == "approver" {
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

	if sesDetails.Rights == "admin" || sesDetails.Rights == "approver" {
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
		IsEdit:      true,
		ActionURL:   fmt.Sprintf("/leave/update/%d", leave.ID),
		SubmitLabel: "Update Leave Request",
		Leave:       leave,
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

	view := LeaveHistoryView{
		Rows:         leaveHistory,
		ExportCSVURL: buildLeaveHistoryExportURL("csv", employeeID, employeeIDStr != ""),
		ExportPDFURL: buildLeaveHistoryExportURL("pdf", employeeID, employeeIDStr != ""),
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
	case "all", "pending", "approved", "declined":
	default:
		filterStatus = "pending"
	}

	rows, err := models.GetFacilityLeaveReview(c.Request.Context(), db, sesDetails.HFID, filterStatus)
	if err != nil {
		log.Printf("Error retrieving facility leave review data: %v", err)
		c.String(http.StatusInternalServerError, "Error retrieving leave requests")
		return
	}

	sessionData.Form = FacilityLeaveReviewView{
		Rows:         rows,
		FilterStatus: filterStatus,
		FilterTitle:  facilityLeaveReviewFilterTitle(filterStatus),
		AllURL:       buildFacilityLeaveReviewURL("all"),
		PendingURL:   buildFacilityLeaveReviewURL("pending"),
		ApprovedURL:  buildFacilityLeaveReviewURL("approved"),
		DeclinedURL:  buildFacilityLeaveReviewURL("declined"),
		CurrentURL:   buildFacilityLeaveReviewURL(filterStatus),
		ExportCSVURL: buildFacilityLeaveReviewExportURL("csv", filterStatus),
		ExportPDFURL: buildFacilityLeaveReviewExportURL("pdf", filterStatus),
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
	case "all", "pending", "approved", "declined":
	default:
		filterStatus = "pending"
	}

	rows, err := models.GetFacilityLeaveReview(c.Request.Context(), db, sesDetails.HFID, filterStatus)
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

func buildFacilityLeaveReviewURL(status string) string {
	if status == "" || status == "pending" {
		return "/leave/review"
	}
	return "/leave/review?status=" + status
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

func buildFacilityLeaveReviewExportURL(format, status string) string {
	params := []string{}
	if format != "" {
		params = append(params, "format="+format)
	}
	if status != "" && status != "pending" {
		params = append(params, "status="+status)
	}
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
	case "approved":
		return "Approved Leave Requests"
	case "declined":
		return "Declined Leave Requests"
	case "all":
		return "All Leave Requests"
	default:
		return "Pending Leave Requests"
	}
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
	default:
		return value.String
	}
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
