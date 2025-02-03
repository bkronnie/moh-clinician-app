package handlers

import (
	"database/sql"
	"encoding/json"
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

// Handler used for entering weekly report for a single user
func SingleEntryForm(c *gin.Context, db *sql.DB, sessionManager *scs.SessionManager) {
	// Get session-specific data (e.g., user ID)
	//sessionData := Get_Session_Data(c, db, sessionManager, nil)
	//log.Printf("Session Data SingleEntryForm: %+v", sessionData)

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

	empID := sesDetails.EmpID       // Retrieve EmpID from the session
	departmentID := sesDetails.HDID // Retrieve HDID from the session
	log.Printf("UserID: %+d, DeptID: %+d", empID, departmentID)

	var zdata any
	idStr := c.Param("id")

	// Convert the 'id' to an integer
	id, err := strconv.Atoi(idStr)
	if err != nil {
		id = 0
	}

	if id > 0 {
		zdata = nil
	} else {
		dt, er := models.WeeklyreportByID(c, db, id)
		if er == nil {
			zdata = dt
		} else {
			zdata = nil
		}
	}

	// Get session data from the database or session manager
	data := Get_Session_Data(c, db, sessionManager, zdata)
	utilities.GenerateHTML(c, data, "base", "bulkcaptureform")
}

/*
func SingleEntryList(c *gin.Context, db *sql.DB, sessionManager *scs.SessionManager) {

}*/

func HandlerReportSave(c *gin.Context, db *sql.DB, sessionManager *scs.SessionManager) {
	/*
		start := c.PostForm("start")
		stop := c.PostForm("stop")
		qn_01 := c.PostForm("qn_01")
		qn_02 := c.PostForm("qn_02")
		qn_03 := c.PostForm("qn_03")
		qn_04 := c.PostForm("qn_04")
		qn_05 := c.PostForm("qn_05")
		qn_06 := c.PostForm("qn_06")
		qn_07 := c.PostForm("qn_07")
		comments := c.PostForm("comments")
	*/
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

	// Parse the start and stop dates from the form
	start := c.PostForm("start")
	stop := c.PostForm("stop")

	// Retrieve individual form fields into a slice of WeeklyReportExtended
	var reports []models.WeeklyReportExtended

	// Assume you have a list of employee IDs to loop through
	for empID := range c.PostFormMap("input") {
		// Create a new WeeklyReportExtended instance for each employee
		report := models.WeeklyReportExtended{
			Start: sql.NullTime{Time: parseDate(start), Valid: true},
			Stop:  sql.NullTime{Time: parseDate(stop), Valid: true},
			Emp:   sql.NullInt64{Int64: empIDToInt(empID), Valid: true}, // Convert empID from string to int64
			Qn01:  sql.NullInt64{Int64: parseInt(c.PostForm("input[" + empID + "][attendance]")), Valid: true},
			Qn02:  sql.NullInt64{Int64: parseInt(c.PostForm("input[" + empID + "][ward_rounds]")), Valid: true},
			Qn03:  sql.NullInt64{Int64: parseInt(c.PostForm("input[" + empID + "][patients_reviewed]")), Valid: true},
			Qn04:  sql.NullInt64{Int64: parseInt(c.PostForm("input[" + empID + "][theatre_days]")), Valid: true},
			Qn05:  sql.NullInt64{Int64: parseInt(c.PostForm("input[" + empID + "][elective]")), Valid: true},
			Qn06:  sql.NullInt64{Int64: parseInt(c.PostForm("input[" + empID + "][emergency]")), Valid: true},
			Qn07:  sql.NullInt64{Int64: parseInt(c.PostForm("input[" + empID + "][postmortems]")), Valid: true},
			Qn08:  sql.NullInt64{Int64: parseInt(c.PostForm("input[" + empID + "][OPD_clinics]")), Valid: true},
			Qn09:  sql.NullInt64{Int64: parseInt(c.PostForm("input[" + empID + "][OPD_patients]")), Valid: true},
			Qn10:  sql.NullInt64{Int64: parseInt(c.PostForm("input[" + empID + "][anc_patients]")), Valid: true},
			Qn11:  sql.NullInt64{Int64: parseInt(c.PostForm("input[" + empID + "][teaching_rounds]")), Valid: true},
			Qn12:  sql.NullInt64{Int64: parseInt(c.PostForm("input[" + empID + "][students_taught]")), Valid: true},
			Qn13:  sql.NullInt64{Int64: parseInt(c.PostForm("input[" + empID + "][mortality_reviews]")), Valid: true},
			Qn14:  sql.NullInt64{Int64: parseInt(c.PostForm("input[" + empID + "][maternal]")), Valid: true},
			Qn15:  sql.NullInt64{Int64: parseInt(c.PostForm("input[" + empID + "][perinatal]")), Valid: true},
			Qn16:  sql.NullInt64{Int64: parseInt(c.PostForm("input[" + empID + "][surgical]")), Valid: true},
			Qn17:  sql.NullInt64{Int64: parseInt(c.PostForm("input[" + empID + "][medical]")), Valid: true},
			Qn18:  sql.NullInt64{Int64: parseInt(c.PostForm("input[" + empID + "][paed]")), Valid: true},
			Qn19:  sql.NullInt64{Int64: parseInt(c.PostForm("input[" + empID + "][labs_requests]")), Valid: true},
			Qn20:  sql.NullInt64{Int64: parseInt(c.PostForm("input[" + empID + "][imaging_requests]")), Valid: true},
			Qn21:  sql.NullInt64{Int64: parseInt(c.PostForm("input[" + empID + "][lab_investigations]")), Valid: true},
			Qn22:  sql.NullInt64{Int64: parseInt(c.PostForm("input[" + empID + "][BS]")), Valid: true},
			Qn23:  sql.NullInt64{Int64: parseInt(c.PostForm("input[" + empID + "][HIV]")), Valid: true},
			Qn24:  sql.NullInt64{Int64: parseInt(c.PostForm("input[" + empID + "][malaria]")), Valid: true},
			Qn25:  sql.NullInt64{Int64: parseInt(c.PostForm("input[" + empID + "][TB]")), Valid: true},
			Qn26:  sql.NullInt64{Int64: parseInt(c.PostForm("input[" + empID + "][CBC]")), Valid: true},
			Qn27:  sql.NullInt64{Int64: parseInt(c.PostForm("input[" + empID + "][chemistry]")), Valid: true},
			Qn28:  sql.NullInt64{Int64: parseInt(c.PostForm("input[" + empID + "][hematology]")), Valid: true},
			Qn29:  sql.NullInt64{Int64: parseInt(c.PostForm("input[" + empID + "][urinalysis]")), Valid: true},
			Qn30:  sql.NullInt64{Int64: parseInt(c.PostForm("input[" + empID + "][gram_stain]")), Valid: true},
			Qn31:  sql.NullInt64{Int64: parseInt(c.PostForm("input[" + empID + "][culture]")), Valid: true},
			Qn32:  sql.NullInt64{Int64: parseInt(c.PostForm("input[" + empID + "][microbiology]")), Valid: true},
			Qn33:  sql.NullInt64{Int64: parseInt(c.PostForm("input[" + empID + "][sensitivity_tests]")), Valid: true},
			Qn34:  sql.NullInt64{Int64: parseInt(c.PostForm("input[" + empID + "][diagnostics]")), Valid: true},
			Qn35:  sql.NullInt64{Int64: parseInt(c.PostForm("input[" + empID + "][xrays]")), Valid: true},
			Qn36:  sql.NullInt64{Int64: parseInt(c.PostForm("input[" + empID + "][ct_scans]")), Valid: true},
			Qn37:  sql.NullInt64{Int64: parseInt(c.PostForm("input[" + empID + "][obstetrics_scans]")), Valid: true},
			Qn38:  sql.NullInt64{Int64: parseInt(c.PostForm("input[" + empID + "][abdominal_scans]")), Valid: true},
			//Qn39:           sql.NullInt64{Int64: parseInt(c.PostForm("input[" + empID + "][ct]")), Valid: true},
			EnteredByID:    sql.NullInt64{Int64: enteredByID, Valid: true},
			EntryCreatedOn: sql.NullTime{Time: time.Now(), Valid: true}, // Set created on time
		}
		// Append the report to the slice
		reports = append(reports, report)
	}

	// Insert each report into the database
	for _, report := range reports {
		if err := report.InsertNewRecord(c.Request.Context(), db); err != nil {
			// Handle error (you might want to log it and return a proper response)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save report"})
			return
		}
	}

	//redirect to report submissions
	c.Redirect(http.StatusFound, "/reports/submissions")

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

// Old handler fxn for bulk capture
func HandlerBulkCaptureList(c *gin.Context, db *sql.DB, sessionManager *scs.SessionManager) {
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

	hospitalID := sesDetails.HFID // Retrieve HFID from the session
	log.Printf("Hospital ID value from session: %+d", hospitalID)

	// Fetch distinct staff members using the new DistinctEmployeeListForCapture function
	staffList, err := models.WeeklyBulkCapture(c.Request.Context(), db, hospitalID, 0, reportingWeekStart, reportingWeekEnd)
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

	// Log the data being passed to the InsertLeave function
	//log.Printf("staffList Data: %+v", data)

	// Render the template with the staff data
	utilities.GenerateHTML(c, data, "base", "bulkcapturelist")
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
	//log.Printf("Hospital ID value from session: %d", facilityID)

	// Check if departmentID is provided
	departmentID := c.Query("departmentID") // Capture departmentID from the query

	if departmentID != "" {
		// Dynamic loading logic for department-specific data
		deptID, err := strconv.Atoi(departmentID)
		log.Printf("Dept Id from dropdown %d", deptID)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid department ID"})
			return
		}

		// Fetch staff and data points for the department
		staff, dataPoints, err := models.GetStaffAndDataPointsByDepartment(db, deptID, int(facilityID))
		if err != nil {
			log.Printf("Error fetching staff and data points: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch data"})
			return
		}

		// Return the data in JSON format for AJAX
		c.JSON(http.StatusOK, gin.H{"staff": staff, "dataPoints": dataPoints})
		return
	}

	//log.Printf("Executing GetFacilitiesAndDepartments fxn from HandlerBulkCaptureForm2")
	// Fetch facilities and departments for the dropdowns
	facilities, departments, err := models.GetFacilitiesAndDepartments(db)
	if err != nil {
		log.Printf("Error loading facilities or departments: %v", err)
		c.String(http.StatusInternalServerError, "Error loading facilities or departments")
		return
	}

	// Prepare data to pass to the template
	data := gin.H{
		"sessionData": sessionData,
		"facilities":  facilities,
		"departments": departments,
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

	userRightsID := sesDetails.Rights
	hospitalID := sesDetails.HFID // Retrieve HFID from the session

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
	flag := 0

	if c.FullPath() == "/reports/submissions" && userRightsID <= "2" {
		log.Println("Handler called for facilities submissions")
		flag = 1
		dts, err = models.WeeklyReportFacilities2(c.Request.Context(), db, hospitalID, yearint, monthint, flag, userRightsID)

	} else if c.FullPath() == "/reports/submissions/departments" || (c.FullPath() == "/reports/submissions" && userRightsID > "2") {
		log.Println("Handler called for departments submissions")
		log.Printf("Departments submissions facility id: %d", hospitalID)
		flag = 2
		week := c.Query("week")

		if userRightsID == "1" {
			facilityIDstr := c.Query("facility")
			facilityID, err := strconv.Atoi(facilityIDstr)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid facilityID"})
				return
			}
			hospitalID = int64(facilityID)
			log.Printf("Hospital ID UserRights 1: %d", hospitalID)
		}

		dts, err = models.WeeklyReportDepartments(c.Request.Context(), db, hospitalID, userRightsID, week, yearint, monthint)
	} else {
		log.Println("Invalid endpoint")
		c.JSON(http.StatusNotFound, gin.H{"error": "Invalid endpoint"})
		return
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

	// Define the reverse mapping for qn_* fields
	qnMapping := map[string]string{
		"firstname":     "staffname",
		"employeetitle": "title",
		"start":         "start",
		"qn_01":         "attendance",
		"qn_02":         "ward_rounds",
		"qn_03":         "patients_reviewed",
		"qn_04":         "theatre_days",
		"qn_05":         "elective",
		"qn_06":         "emergency",
		"qn_07":         "postmortems",
		"qn_08":         "OPD_clinics",
		"qn_09":         "OPD_patients",
		"qn_10":         "anc_patients",
		"qn_11":         "teaching_rounds",
		"qn_12":         "students_taught",
		"qn_13":         "mortality_reviews",
		"qn_14":         "maternal",
		"qn_15":         "perinatal",
		"qn_16":         "surgical",
		"qn_17":         "medical",
		"qn_18":         "pead",
		"qn_19":         "labs_requests",
		"qn_20":         "imaging_requests",
		"qn_21":         "investigations",
		"qn_22":         "BS",
		"qn_23":         "HIV",
		"qn_24":         "malaria",
		"qn_25":         "TB",
		"qn_26":         "CBC",
		"qn_27":         "chemistry",
		"qn_28":         "hematology",
		"qn_29":         "urinalysis",
		"qn_30":         "gram_stain",
		"qn_31":         "culture",
		"qn_32":         "microbiology",
		"qn_33":         "sensitivity_tests",
		"qn_34":         "diagnostics",
		"qn_35":         "xrays",
		"qn_36":         "ct_scans",
		"qn_37":         "obstetrics_scans",
		"qn_38":         "abdominal_scans",
	}

	// Transform the data points using the mapping
	transformedData := make([]map[string]interface{}, len(staffData))
	for i, staff := range staffData {
		transformedEntry := make(map[string]interface{})
		transformedEntry["staffname"] = staff.Fname
		transformedEntry["title"] = staff.EmpTitle
		transformedEntry["start"] = staff.Start

		// Apply the mapping to the struct fields explicitly
		for key, value := range qnMapping {
			switch key {
			case "qn_01":
				transformedEntry[value] = staff.Qn01.Int64
			case "qn_02":
				transformedEntry[value] = staff.Qn02.Int64
			case "qn_03":
				transformedEntry[value] = staff.Qn03.Int64
			case "qn_04":
				transformedEntry[value] = staff.Qn04.Int64
			case "qn_05":
				transformedEntry[value] = staff.Qn05.Int64
			case "qn_06":
				transformedEntry[value] = staff.Qn06.Int64
			case "qn_07":
				transformedEntry[value] = staff.Qn07.Int64
			case "qn_08":
				transformedEntry[value] = staff.Qn08.Int64
			case "qn_09":
				transformedEntry[value] = staff.Qn09.Int64
			case "qn_10":
				transformedEntry[value] = staff.Qn10.Int64
			case "qn_11":
				transformedEntry[value] = staff.Qn11.Int64
			case "qn_12":
				transformedEntry[value] = staff.Qn12.Int64
			case "qn_13":
				transformedEntry[value] = staff.Qn13.Int64
			case "qn_14":
				transformedEntry[value] = staff.Qn14.Int64
			case "qn_15":
				transformedEntry[value] = staff.Qn15.Int64
			case "qn_16":
				transformedEntry[value] = staff.Qn16.Int64
			case "qn_17":
				transformedEntry[value] = staff.Qn17.Int64
			case "qn_18":
				transformedEntry[value] = staff.Qn18.Int64
			case "qn_19":
				transformedEntry[value] = staff.Qn19.Int64
			case "qn_20":
				transformedEntry[value] = staff.Qn20.Int64
			case "qn_21":
				transformedEntry[value] = staff.Qn21.Int64
			case "qn_22":
				transformedEntry[value] = staff.Qn22.Int64
			case "qn_23":
				transformedEntry[value] = staff.Qn23.Int64
			case "qn_24":
				transformedEntry[value] = staff.Qn24.Int64
			case "qn_25":
				transformedEntry[value] = staff.Qn25.Int64
			case "qn_26":
				transformedEntry[value] = staff.Qn26.Int64
			case "qn_27":
				transformedEntry[value] = staff.Qn27.Int64
			case "qn_28":
				transformedEntry[value] = staff.Qn28.Int64
			case "qn_29":
				transformedEntry[value] = staff.Qn29.Int64
			case "qn_30":
				transformedEntry[value] = staff.Qn30.Int64
			case "qn_31":
				transformedEntry[value] = staff.Qn31.Int64
			case "qn_32":
				transformedEntry[value] = staff.Qn32.Int64
			case "qn_33":
				transformedEntry[value] = staff.Qn33.Int64
			case "qn_34":
				transformedEntry[value] = staff.Qn34.Int64
			case "qn_35":
				transformedEntry[value] = staff.Qn35.Int64
			case "qn_36":
				transformedEntry[value] = staff.Qn36.Int64
			case "qn_37":
				transformedEntry[value] = staff.Qn37.Int64
			case "qn_38":
				transformedEntry[value] = staff.Qn38.Int64
			}
		}

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
func HandlerReportsAnalysis(c *gin.Context, db *sql.DB, sessionManager *scs.SessionManager) {

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
