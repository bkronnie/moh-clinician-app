package handlers

import (
	"database/sql"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/alexedwards/scs/v2"
	"github.com/gin-gonic/gin"
	"github.com/moh/clinician/internals/models"
	"github.com/moh/clinician/internals/utilities"
)

type AccountEditView struct {
	Error       string
	Message     string
	Values      map[string]string
	Facilities  []models.RegistrationOption
	Departments []models.RegistrationOption
	Titles      []models.SpecialistTitleOption
	Changes     []models.AccountProfileChange
}

func HandlerAccountEdit(c *gin.Context, db *sql.DB, sessionManager *scs.SessionManager) {
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

	view, err := buildAccountEditView(c, db, sesDetails.UserID, "")
	if err != nil {
		c.String(http.StatusInternalServerError, "Error loading account details")
		return
	}

	if strings.TrimSpace(c.Query("updated")) == "1" {
		view.Message = "Account details updated. Future submissions will follow your current facility and department assignment."
	}

	sessionData.Form = view
	utilities.GenerateHTML(c, sessionData, "base", "account-edit")
}

func HandlerAccountUpdate(c *gin.Context, db *sql.DB, sessionManager *scs.SessionManager) {
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

	values := map[string]string{
		"firstname":      strings.TrimSpace(c.PostForm("firstname")),
		"lastname":       strings.TrimSpace(c.PostForm("lastname")),
		"othername":      strings.TrimSpace(c.PostForm("othername")),
		"employeeNumber": strings.TrimSpace(c.PostForm("employee_number")),
		"dateOfBirth":    strings.TrimSpace(c.PostForm("date_of_birth")),
		"phoneNumber":    strings.TrimSpace(c.PostForm("phone_number")),
		"email":          strings.ToLower(strings.TrimSpace(c.PostForm("email"))),
		"facility":       strings.TrimSpace(c.PostForm("facility_id")),
		"department":     strings.TrimSpace(c.PostForm("department_id")),
		"title":          strings.TrimSpace(c.PostForm("title_id")),
		"specialisation": strings.TrimSpace(c.PostForm("specialisation")),
	}

	view, err := buildAccountEditView(c, db, sesDetails.UserID, "")
	if err != nil {
		c.String(http.StatusInternalServerError, "Error loading account details")
		return
	}
	view.Values = values

	facilityID, err := parseOptionalAccountInt64(values["facility"])
	if err != nil {
		view.Error = "Please select a valid facility."
		renderAccountEdit(c, sessionData, view)
		return
	}

	departmentID, err := parseOptionalAccountInt64(values["department"])
	if err != nil {
		view.Error = "Please select a valid department."
		renderAccountEdit(c, sessionData, view)
		return
	}

	titleID, err := parseOptionalAccountInt64(values["title"])
	if err != nil {
		view.Error = "Please select a valid specialist title."
		renderAccountEdit(c, sessionData, view)
		return
	}

	var dateOfBirth *time.Time
	if values["dateOfBirth"] != "" {
		parsedDOB, err := time.Parse("2006-01-02", values["dateOfBirth"])
		if err != nil {
			view.Error = "Date of birth must use the YYYY-MM-DD format."
			renderAccountEdit(c, sessionData, view)
			return
		}
		dateOfBirth = &parsedDOB
	}

	err = models.UpdateOwnAccountProfile(c.Request.Context(), db, sesDetails.UserID, sesDetails.EmpID, models.AccountProfileInput{
		Email:          values["email"],
		FirstName:      values["firstname"],
		LastName:       values["lastname"],
		OtherName:      values["othername"],
		EmployeeNumber: values["employeeNumber"],
		DateOfBirth:    dateOfBirth,
		PhoneNumber:    values["phoneNumber"],
		FacilityID:     facilityID,
		DepartmentID:   departmentID,
		TitleID:        titleID,
		Specialisation: values["specialisation"],
	})
	if err != nil {
		view.Error = err.Error()
		renderAccountEdit(c, sessionData, view)
		return
	}

	c.Redirect(http.StatusFound, "/account/edit?updated=1")
}

func buildAccountEditView(c *gin.Context, db *sql.DB, userID int64, errorMessage string) (AccountEditView, error) {
	profile, err := models.GetAccountProfile(c.Request.Context(), db, userID)
	if err != nil {
		return AccountEditView{}, err
	}

	facilities, departments, titles, err := models.RegistrationOptions(c.Request.Context(), db)
	if err != nil {
		return AccountEditView{}, err
	}

	changes, err := models.GetRecentAccountProfileChanges(c.Request.Context(), db, profile.EmployeeID, 8)
	if err != nil {
		return AccountEditView{}, err
	}

	return AccountEditView{
		Error:       errorMessage,
		Values:      accountProfileValues(profile),
		Facilities:  facilities,
		Departments: departments,
		Titles:      titles,
		Changes:     changes,
	}, nil
}

func accountProfileValues(profile models.AccountProfile) map[string]string {
	values := map[string]string{
		"firstname":      profile.FirstName,
		"lastname":       profile.LastName,
		"othername":      profile.OtherName,
		"employeeNumber": profile.EmployeeNumber,
		"phoneNumber":    profile.PhoneNumber,
		"email":          profile.Email,
		"facility":       strconv.FormatInt(profile.FacilityID, 10),
		"department":     strconv.FormatInt(profile.DepartmentID, 10),
		"title":          strconv.FormatInt(profile.TitleID, 10),
		"specialisation": profile.Specialisation,
	}
	if profile.DateOfBirth.Valid {
		values["dateOfBirth"] = profile.DateOfBirth.Time.Format("2006-01-02")
	}
	return values
}

func renderAccountEdit(c *gin.Context, sessionData utilities.TemplateData, view AccountEditView) {
	sessionData.Form = view
	utilities.GenerateHTML(c, sessionData, "base", "account-edit")
}

func parseOptionalAccountInt64(value string) (int64, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return 0, nil
	}
	return strconv.ParseInt(value, 10, 64)
}
