package utilities

import (
	"database/sql"
	"fmt"
	"html/template"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/gin-gonic/gin"
	//"github.com/moh/clinician/internals/models"
)

type TemplateData struct {
	CurrentYear     int
	Form            any
	Ses             any
	Items           []interface{}
	Optionz         map[string]map[string]string
	Flash           string
	IsAuthenticated bool
	CSRFToken       string // Add a CSRFToken field.
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

func GetPath(topFile string) (rtn string) {
	var dirAbsPath string

	ex, err := os.Executable()
	if err == nil {
		dirAbsPath = filepath.Dir(ex)
		rtn = dirAbsPath + "/" + topFile
	} else {
		rtn = topFile
	}

	return
}

func GenerateHTML(c *gin.Context, zdata interface{}, filenames ...string) {
	var files []string
	for _, file := range filenames {
		files = append(files, GetPath(fmt.Sprintf("../../ui/html/%s.html", file)))
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
		files = append(files, GetPath(fmt.Sprintf("../../ui/html/%s.html", file)))
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
		files = append(files, GetPath(fmt.Sprintf("../../ui/html/%s.html", file)))
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
