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

	log.Printf("Session Data HandlerHome: %+v", sesDetails)

	// Retrieve HFID from session
	facilityID := sesDetails.HFID
	log.Printf("Hospital ID value from session: %d", facilityID)

	// Fetch dashboard data
	dashdata, err := models.GetDashboardData(db, int(facilityID))
	if err != nil {
		log.Printf("Error fetching dashboard data: %v", err)
		c.String(http.StatusInternalServerError, "Error fetching dashboard data")
		return
	}

	// Prepare the template data
	data := gin.H{
		"sessionData":     sessionData,
		"FacilitiesCount": dashdata.FacilitiesCount,
		"TotalStaff":      dashdata.TotalStaff,
		"StaffPresent":    dashdata.StaffPresent,
		"StaffOnLeave":    dashdata.StaffOnLeave,
		"HFName":          sesDetails.HFName, // Explicitly pass HFName to the template
	}

	//data := Get_Session_Data(c, db, sessionManager, nil)
	log.Printf("data variable passed in HandlerHome: %v", data)

	fmt.Println("home page loading with data")
	utilities.GenerateHTML(c, data, "base", "home")

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
