package utilities

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	//"github.com/moh/clinician/internals/models"
)

type TemplateData struct {
	CurrentYear         int
	Form                any
	Ses                 any
	Items               []interface{}
	Optionz             map[string]map[string]string
	Flash               string
	IsAuthenticated     bool
	CSRFToken           string // Add a CSRFToken field.
	NotifPendingLeave   int    // for managers: pending leave requests
	NotifPendingReports int    // for managers/admin: pending reports
	NotifMyLeave        int    // for clinicians: recently reviewed leave
	NotifMyReports      int    // for clinicians: recently reviewed reports
	NotifMyDataEntry    int    // for clinicians: pending reminder for last-week submission
}

type SessionDetails struct {
	Email  string `form:"email"`
	FName  string `form:"fname"`
	LName  string `form:"lname"`
	Right  int64  `form:"right"`
	EmpID  int64  `form:"employeeid"`
	Rights string `form:"rights"`
	UserID int64  `form:"userid"` //userID not employee ID
	HFName string `form:"hfname"` //facility name
	HFID   int64  `form:"hfid"`
	HDID   int64  `form:"hdid"` //facility id
}

// Initialize a template.FuncMap object and store it in a global variable. This is
// essentially a string-keyed map which acts as a lookup between the names of our
// custom template functions and the functions themselves.
var functions = template.FuncMap{
	"humanDate":         HumanDate,
	"IsNullStringEmpty": IsNullStringEmpty,
	"seq":               Seq,
	"session":           SessionFromTemplateData,
	"hasRole":           HasRole,
	"toJSON":            ToJSON,
	"navIcon":           NavIcon,
}

func HumanDate(t time.Time) string {
	return t.Format("02 Jan 2006 at 15:04")
}

func IsNullStringEmpty(nullable sql.NullString) bool {
	return !nullable.Valid || nullable.String == ""
}

func Seq(start, end int) []int {
	s := make([]int, end-start+1)
	for i := range s {
		s[i] = start + i
	}
	return s
}

func GetPath(parts ...string) string {
	return AppPath(parts...)
}

func GenerateHTML(c *gin.Context, zdata interface{}, filenames ...string) {
	var files []string
	for _, file := range filenames {
		files = append(files, GetPath("clinician", "ui", "html", fmt.Sprintf("%s.html", file)))
	}

	// Extend the global `functions` map with the `add` function
	funcs := template.FuncMap{
		"add":        add,
		"multiply":   multiply,
		"formatDate": formatDate,
		"dict":       dict, // Add the dict function here
	}

	// Merge custom functions with global `functions`
	for k, v := range functions {
		funcs[k] = v
	}

	// Parse templates using the updated functions map
	templates := template.Must(template.New("").Funcs(funcs).ParseFiles(files...))

	// Execute template
	if err := templates.ExecuteTemplate(c.Writer, "layout", zdata); err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
	}
}

func GenerateHTML1(c *gin.Context, zdata interface{}, filenames ...string) {
	var files []string
	for _, file := range filenames {
		files = append(files, GetPath("clinician", "ui", "html", fmt.Sprintf("%s.html", file)))
	}

	// Extend the global `functions` map with the `add` function
	funcs := template.FuncMap{
		"add":        add,
		"multiply":   multiply,
		"formatDate": formatDate,
		"dict":       dict, // Add the dict function here
	}

	// Merge custom functions with global `functions`
	for k, v := range functions {
		funcs[k] = v
	}

	// Parse templates using the updated functions map
	templates := template.Must(template.New("").Funcs(funcs).ParseFiles(files...))

	// Execute template
	if err := templates.ExecuteTemplate(c.Writer, "content", zdata); err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
	}
}

func GenerateHTML2(c *gin.Context, zdata interface{}, filenames ...string) {
	var files []string
	for _, file := range filenames {
		files = append(files, GetPath("clinician", "ui", "html", fmt.Sprintf("%s.html", file)))
	}

	// Extend the global `functions` map with necessary functions
	funcs := template.FuncMap{
		"add":        add,
		"multiply":   multiply,
		"formatDate": formatDate,
		"dict":       dict,
	}

	// Merge with global functions
	for k, v := range functions {
		funcs[k] = v
	}

	// Parse the provided template files
	templates := template.Must(template.New("").Funcs(funcs).ParseFiles(files...))

	// Render the first template file provided
	if err := templates.ExecuteTemplate(c.Writer, filenames[0], zdata); err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
	}
}

func add(a, b int) int {
	return a + b
}

// The multiply function
func multiply(a, b int) int {
	return a * b
}

func formatDate(t time.Time, layout string) string {
	fmt.Println(t.String())
	return t.Format(layout)
}

func getval(id int64, table string) string {

	switch table {
	case "facility":
		/* f, er := models.EmployeeByID(C, DB, int(id))
		if er == nil {
			return f.Fname
		} */
	case "department":
		// code for getting department name from department table based on id
	case "employee":

	}

	return ""
}

// Define the dict function that will be used in templates
func dict(keysAndValues ...interface{}) (map[string]interface{}, error) {
	if len(keysAndValues)%2 != 0 {
		return nil, fmt.Errorf("invalid number of arguments for dict")
	}

	result := make(map[string]interface{})
	for i := 0; i < len(keysAndValues); i += 2 {
		key, ok := keysAndValues[i].(string)
		if !ok {
			return nil, fmt.Errorf("key must be a string")
		}
		result[key] = keysAndValues[i+1]
	}
	return result, nil
}

func SessionFromTemplateData(root interface{}) SessionDetails {
	switch v := root.(type) {
	case TemplateData:
		return coerceSession(v.Ses)
	case *TemplateData:
		if v == nil {
			return SessionDetails{}
		}
		return coerceSession(v.Ses)
	case map[string]interface{}:
		if ses, ok := v["Ses"]; ok {
			return coerceSession(ses)
		}
		if ses, ok := v["sessionData"]; ok {
			return SessionFromTemplateData(ses)
		}
	}

	return SessionDetails{}
}

func HasRole(root interface{}, roles ...string) bool {
	ses := SessionFromTemplateData(root)
	if ses.Rights == "" {
		return false
	}

	for _, role := range roles {
		if strings.EqualFold(ses.Rights, role) {
			return true
		}
	}

	return false
}

func ToJSON(v interface{}) template.JS {
	b, err := json.Marshal(v)
	if err != nil {
		return template.JS("[]")
	}
	return template.JS(b)
}

func NavIcon(name string) template.HTML {
	icons := map[string]string{
		"brand":         `<svg viewBox="0 0 24 24" aria-hidden="true" focusable="false"><path d="M12 2l7 4v6c0 5.1-3.4 8.9-7 10-3.6-1.1-7-4.9-7-10V6l7-4z"></path><path d="M12 7v10"></path><path d="M8 11h8"></path></svg>`,
		"dashboard":     `<svg viewBox="0 0 24 24" aria-hidden="true" focusable="false"><rect x="3" y="3" width="8" height="8" rx="2"></rect><rect x="13" y="3" width="8" height="5" rx="2"></rect><rect x="13" y="10" width="8" height="11" rx="2"></rect><rect x="3" y="13" width="8" height="8" rx="2"></rect></svg>`,
		"entry":         `<svg viewBox="0 0 24 24" aria-hidden="true" focusable="false"><path d="M14 4h4a2 2 0 0 1 2 2v12a2 2 0 0 1-2 2H6a2 2 0 0 1-2-2v-4"></path><path d="M13 5l6 6"></path><path d="M3 21l3.5-.7L18 8.8a1.5 1.5 0 0 0 0-2.1l-.7-.7a1.5 1.5 0 0 0-2.1 0L3.7 17.5 3 21z"></path></svg>`,
		"weekly":        `<svg viewBox="0 0 24 24" aria-hidden="true" focusable="false"><rect x="3" y="5" width="18" height="16" rx="2"></rect><path d="M8 3v4"></path><path d="M16 3v4"></path><path d="M3 10h18"></path><path d="M8 14h3"></path><path d="M13 14h3"></path><path d="M8 18h8"></path></svg>`,
		"reports":       `<svg viewBox="0 0 24 24" aria-hidden="true" focusable="false"><path d="M5 19V9"></path><path d="M12 19V5"></path><path d="M19 19v-7"></path><path d="M3 19h18"></path></svg>`,
		"employees":     `<svg viewBox="0 0 24 24" aria-hidden="true" focusable="false"><path d="M16 21v-2a4 4 0 0 0-4-4H7a4 4 0 0 0-4 4v2"></path><circle cx="9.5" cy="8" r="3"></circle><path d="M20 21v-2a4 4 0 0 0-3-3.87"></path><path d="M14.5 5.2a3 3 0 0 1 0 5.6"></path></svg>`,
		"account":       `<svg viewBox="0 0 24 24" aria-hidden="true" focusable="false"><circle cx="12" cy="8" r="4"></circle><path d="M4 20a8 8 0 0 1 16 0"></path></svg>`,
		"leave":         `<svg viewBox="0 0 24 24" aria-hidden="true" focusable="false"><rect x="3" y="5" width="18" height="16" rx="2"></rect><path d="M8 3v4"></path><path d="M16 3v4"></path><path d="M3 10h18"></path><path d="M8 14h8"></path></svg>`,
		"history":       `<svg viewBox="0 0 24 24" aria-hidden="true" focusable="false"><path d="M3 12a9 9 0 1 0 3-6.7"></path><path d="M3 4v5h5"></path><path d="M12 7v6l4 2"></path></svg>`,
		"facilities":    `<svg viewBox="0 0 24 24" aria-hidden="true" focusable="false"><path d="M3 21h18"></path><path d="M5 21V7l7-4 7 4v14"></path><path d="M9 10h1"></path><path d="M14 10h1"></path><path d="M9 14h1"></path><path d="M14 14h1"></path><path d="M11 21v-4h2v4"></path></svg>`,
		"departments":   `<svg viewBox="0 0 24 24" aria-hidden="true" focusable="false"><path d="M4 6h16"></path><path d="M4 12h10"></path><path d="M4 18h16"></path><circle cx="18" cy="12" r="2"></circle></svg>`,
		"customization": `<svg viewBox="0 0 24 24" aria-hidden="true" focusable="false"><circle cx="12" cy="12" r="3"></circle><path d="M19.4 15a1.6 1.6 0 0 0 .3 1.8l.1.1a2 2 0 1 1-2.8 2.8l-.1-.1a1.6 1.6 0 0 0-1.8-.3 1.6 1.6 0 0 0-1 1.5V21a2 2 0 1 1-4 0v-.2a1.6 1.6 0 0 0-1-1.5 1.6 1.6 0 0 0-1.8.3l-.1.1a2 2 0 0 1-2.8-2.8l.1-.1a1.6 1.6 0 0 0 .3-1.8 1.6 1.6 0 0 0-1.5-1H3a2 2 0 1 1 0-4h.2a1.6 1.6 0 0 0 1.5-1 1.6 1.6 0 0 0-.3-1.8l-.1-.1a2 2 0 1 1 2.8-2.8l.1.1a1.6 1.6 0 0 0 1.8.3h0a1.6 1.6 0 0 0 1-1.5V3a2 2 0 1 1 4 0v.2a1.6 1.6 0 0 0 1 1.5h0a1.6 1.6 0 0 0 1.8-.3l.1-.1a2 2 0 1 1 2.8 2.8l-.1.1a1.6 1.6 0 0 0-.3 1.8v0a1.6 1.6 0 0 0 1.5 1H21a2 2 0 1 1 0 4h-.2a1.6 1.6 0 0 0-1.5 1z"></path></svg>`,
		"submissions":   `<svg viewBox="0 0 24 24" aria-hidden="true" focusable="false"><path d="M12 3v12"></path><path d="M7 10l5 5 5-5"></path><rect x="4" y="17" width="16" height="4" rx="1.5"></rect></svg>`,
		"logout":        `<svg viewBox="0 0 24 24" aria-hidden="true" focusable="false"><path d="M15 3h4a2 2 0 0 1 2 2v14a2 2 0 0 1-2 2h-4"></path><path d="M10 17l5-5-5-5"></path><path d="M15 12H3"></path></svg>`,
	}

	if icon, ok := icons[name]; ok {
		return template.HTML(icon)
	}

	return template.HTML(icons["dashboard"])
}

func coerceSession(v interface{}) SessionDetails {
	switch s := v.(type) {
	case SessionDetails:
		return s
	case *SessionDetails:
		if s == nil {
			return SessionDetails{}
		}
		return *s
	default:
		return SessionDetails{}
	}
}
