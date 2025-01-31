package handlers

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/alexedwards/scs/v2"
	"github.com/gin-gonic/gin"
	"github.com/moh/clinician/internals/models"
	"github.com/moh/clinician/internals/utilities"
)

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
	data := Get_Session_Data(c, db, sessionManager, "")
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

	// Retrieve form values
	employeeID, _ := strconv.ParseInt(c.PostForm("employee_id"), 10, 64)
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

	// Assign values to leave object
	leave.EmpID.Int64 = employeeID
	leave.EmpID.Valid = true
	leave.LeaveTypeID.Int64 = leaveTypeID
	leave.LeaveTypeID.Valid = true
	leave.StartDate.Time = startDate
	leave.StartDate.Valid = true
	leave.EndDate.Time = endDate
	leave.EndDate.Valid = true
	leave.Comments.String = comments
	leave.Comments.Valid = true
	leave.SubmittedBy.Int64 = sesDetails.EmpID
	leave.SubmittedBy.Valid = true
	leave.SubmissionDate.Time = time.Now()
	leave.SubmissionDate.Valid = true
	leave.LeaveStatus.String = "Approved"
	leave.LeaveStatus.Valid = true

	// Log the data being passed to the InsertLeave function
	log.Printf("Data being passed to InsertLeave: %+v", leave)

	// Save leave data to the database
	if err := leave.InsertLeave(c.Request.Context(), db); err != nil {
		log.Println("Failed to insert leave data:", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to add leave record"})
		return
	}

	// Redirect back to the employee list page
	c.Redirect(http.StatusFound, "/employee/list")
}

// Handler fxn to retrieve leave history for a staff member
func HandlerEmployeeLeaveHistory(c *gin.Context, db *sql.DB, sessionManager *scs.SessionManager) {
	// Retrieve the employee ID from the URL parameter
	employeeIDStr := c.Param("id")
	employeeID, err := strconv.Atoi(employeeIDStr)
	if err != nil {
		c.String(http.StatusBadRequest, "Invalid employee ID")
		return
	}

	// Fetch leave history from the database
	leaveHistory, err := models.GetHistory(db, employeeID)
	if err != nil {
		c.String(http.StatusInternalServerError, "Error retrieving leave history: %v", err)
		return
	}

	// Populate the template data
	data := Get_Session_Data(c, db, sessionManager, leaveHistory)

	// Render the leave history template
	utilities.GenerateHTML(c, data, "base", "leavehistory")
}
