package handlers

import (
	"database/sql"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"clinician/internals/models"
	"clinician/internals/security"
	"clinician/internals/utilities"

	"github.com/alexedwards/scs/v2"
	"github.com/gin-gonic/gin"
)

type RegistrationPageData struct {
	Error       string
	Message     string
	Values      map[string]string
	Facilities  []models.RegistrationOption
	Departments []models.RegistrationOption
	Titles      []models.SpecialistTitleOption
}

func HandlerRegistrationForm(c *gin.Context, db *sql.DB, sessionManager *scs.SessionManager) {
	if security.IsAuthenticated(c, sessionManager) {
		c.Redirect(http.StatusFound, "/")
		return
	}

	pageData, err := loadRegistrationPageData(c, db)
	if err != nil {
		c.String(http.StatusInternalServerError, "Error loading registration options")
		return
	}

	utilities.GenerateHTML(c, pageData, "register")
}

func HandlerRegistrationSave(c *gin.Context, db *sql.DB, sessionManager *scs.SessionManager) {
	if security.IsAuthenticated(c, sessionManager) {
		c.Redirect(http.StatusFound, "/")
		return
	}

	values := map[string]string{
		"firstname":      strings.TrimSpace(c.PostForm("firstname")),
		"lastname":       strings.TrimSpace(c.PostForm("lastname")),
		"othername":      strings.TrimSpace(c.PostForm("othername")),
		"employeeNumber": strings.TrimSpace(c.PostForm("employee_number")),
		"dateOfBirth":    strings.TrimSpace(c.PostForm("date_of_birth")),
		"phoneNumber":    strings.TrimSpace(c.PostForm("phone_number")),
		"email":          strings.TrimSpace(c.PostForm("email")),
		"facility":       strings.TrimSpace(c.PostForm("facility_id")),
		"department":     strings.TrimSpace(c.PostForm("department_id")),
		"title":          strings.TrimSpace(c.PostForm("title_id")),
		"specialisation": strings.TrimSpace(c.PostForm("specialisation")),
	}

	password := c.PostForm("password")
	confirmPassword := c.PostForm("confirm_password")

	pageData, err := loadRegistrationPageData(c, db)
	if err != nil {
		c.String(http.StatusInternalServerError, "Error loading registration options")
		return
	}
	pageData.Values = values

	if values["firstname"] == "" || values["lastname"] == "" || values["employeeNumber"] == "" || values["email"] == "" {
		pageData.Error = "First name, last name, medical officer ID, and email are required."
		utilities.GenerateHTML(c, pageData, "register")
		return
	}

	if password == "" || confirmPassword == "" {
		pageData.Error = "Password and confirm password are required."
		utilities.GenerateHTML(c, pageData, "register")
		return
	}

	if password != confirmPassword {
		pageData.Error = "Passwords do not match."
		utilities.GenerateHTML(c, pageData, "register")
		return
	}

	facilityID, departmentID, titleID, parseErr := parseRegistrationIDs(values["facility"], values["department"], values["title"])
	if parseErr != nil {
		pageData.Error = parseErr.Error()
		utilities.GenerateHTML(c, pageData, "register")
		return
	}

	var dob *time.Time
	if values["dateOfBirth"] != "" {
		parsedDOB, err := time.Parse("2006-01-02", values["dateOfBirth"])
		if err != nil {
			pageData.Error = "Date of birth must use the YYYY-MM-DD format."
			utilities.GenerateHTML(c, pageData, "register")
			return
		}
		dob = &parsedDOB
	}

	_, err = models.CreateClinicianSelfRegistration(c.Request.Context(), db, models.ClinicianRegistrationInput{
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
	})
	if err != nil {
		pageData.Error = err.Error()
		utilities.GenerateHTML(c, pageData, "register")
		return
	}

	c.Redirect(http.StatusFound, "/login?message=Account%20created%20successfully.%20Please%20sign%20in.")
}

func loadRegistrationPageData(c *gin.Context, db *sql.DB) (RegistrationPageData, error) {
	facilities, departments, titles, err := models.RegistrationOptions(c.Request.Context(), db)
	if err != nil {
		return RegistrationPageData{}, err
	}

	return RegistrationPageData{
		Message:     strings.TrimSpace(c.Query("message")),
		Values:      map[string]string{},
		Facilities:  facilities,
		Departments: departments,
		Titles:      titles,
	}, nil
}

func parseRegistrationIDs(facilityValue, departmentValue, titleValue string) (int64, int64, int64, error) {
	facilityID, err := parseInt64Field(facilityValue)
	if err != nil || facilityID == 0 {
		return 0, 0, 0, errors.New("Please select a facility.")
	}

	departmentID, err := parseInt64Field(departmentValue)
	if err != nil || departmentID == 0 {
		return 0, 0, 0, errors.New("Please select a department.")
	}

	titleID, err := parseInt64Field(titleValue)
	if err != nil || titleID == 0 {
		return 0, 0, 0, errors.New("Please select a specialist title.")
	}

	return facilityID, departmentID, titleID, nil
}

func parseInt64Field(value string) (int64, error) {
	if strings.TrimSpace(value) == "" {
		return 0, nil
	}

	return strconv.ParseInt(value, 10, 64)
}
