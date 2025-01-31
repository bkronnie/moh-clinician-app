package handlers

import (
	"database/sql"
	"log"
	"net/http"
	"strconv"

	"github.com/alexedwards/scs/v2"
	"github.com/gin-gonic/gin"
	_ "github.com/lib/pq"
	"github.com/moh/clinician/internals/models"
	"github.com/moh/clinician/internals/utilities"
)

func SetupHandler(c *gin.Context, db *sql.DB) {

	firstname := c.PostForm("firstname")
	lastname := c.PostForm("lastname")
	username := c.PostForm("username")
	password := c.PostForm("password")
	cpassword := c.PostForm("cpassword")

	if firstname == "" || lastname == "" || username == "" || password == "" || cpassword == "" {
		utilities.Danger("All fields are required")
		c.Redirect(http.StatusFound, "/login")
		c.Abort()
		return
	}

	if password != cpassword {
		utilities.Danger("Passwords do not match")
		c.Redirect(http.StatusFound, "/login")
		c.Abort()
		return
	}

	employee := models.Employee{}

	employee.Fname.String = firstname
	employee.Fname.Valid = true
	employee.Lname.String = lastname
	employee.Lname.Valid = true

	var emp int64

	err := employee.Insert(c, db)

	if err != nil {
		emp = 0
	} else {
		emp = int64(employee.EmpID)
	}

	user := models.User{}
	user.Employees = sql.NullInt64{Int64: emp, Valid: true}
	user.Username = username
	user.Pssword = models.Encrypt(password)
	user.Rights = "admin"

	err = user.Insert(c, db)
	if err != nil {
		utilities.Danger("Setup failed: could not insert user - " + err.Error())
		c.Redirect(http.StatusFound, "/login")
		c.Abort()
		return
	}

	utilities.Danger("Setup complete successfully")
	c.Redirect(http.StatusFound, "/login")
	c.Abort()
}

func LoginHandler(c *gin.Context, db *sql.DB, sessionManager *scs.SessionManager) {
	// Extract form data (username and password)
	username := c.PostForm("username")
	password := c.PostForm("password")

	// Authenticate user
	user, err := models.UserByEmailPass(c.Request.Context(), db, username, password)
	if err != nil {
		utilities.Danger("Invalid username or password -" + err.Error())
		c.Redirect(http.StatusFound, "/login")
		c.Abort()
		return
	}

	if user.ID > 0 {
		// On successful login, set userID in the session
		userID := strconv.Itoa(user.ID)
		log.Printf("UserID: %s", userID)

		/*
					// Retrieve facilityID and rights for the user from the database
					var facilityID int
					var rights string
					query := `
			            SELECT u.rights, f.id
			            FROM public.users u
			            JOIN public.employees e ON u.employees = e.id
			            JOIN public.facilities f ON e.facility = f.id
			            JOIN public.departments d ON e.department = d.id
			            WHERE u.id = $1
			        `
					if err := db.QueryRowContext(c.Request.Context(), query, user.ID).Scan(&rights, &facilityID); err != nil {
						utilities.Danger("Failed to retrieve facility ID and rights - " + err.Error())
						c.Redirect(http.StatusFound, "/login")
						c.Abort()
						return
					}

					log.Printf("FacilityID: %d, Rights: %s", facilityID, rights) */

		// Store userID, facilityID, and rights in the session
		sessionCtx, er := sessionManager.Load(c.Request.Context(), "session_token")
		if er != nil {
			c.AbortWithError(http.StatusInternalServerError, err)
			return
		}

		sessionManager.Put(sessionCtx, "userID", userID)
		//sessionManager.Put(sessionCtx, "facilityID", facilityID)
		//sessionManager.Put(sessionCtx, "rights", rights)

		// Commit the session and get the session token
		token, _, err := sessionManager.Commit(sessionCtx)
		if err != nil {
			c.AbortWithError(http.StatusInternalServerError, err)
			return
		}

		// Set session token in cookie
		c.SetCookie("session_token", token, int(sessionManager.Lifetime.Seconds()), "/", "", sessionManager.Cookie.Secure, sessionManager.Cookie.HttpOnly)

		// Get session-specific data (e.g., user ID)
		sessionData := Get_Session_Data(c, db, sessionManager, nil)
		log.Printf("Session Data Login Handler: %+v", sessionData)

		// Redirect to the home page after successful login
		utilities.Info("Login successful")
		c.Redirect(http.StatusFound, "/")
		return
	} else {
		utilities.Danger("Login failure")
		c.Redirect(http.StatusFound, "/login")
		c.Abort()
	}
}

func AboutHandler(c *gin.Context) {

}

func DashboardHandler(c *gin.Context) {

}

func SettingsHandler(c *gin.Context) {

}
