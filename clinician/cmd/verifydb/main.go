package main

import (
	"bufio"
	"database/sql"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	_ "github.com/lib/pq"
)

type config struct {
	Ux string `json:"Ux"`
	Px string `json:"Px"`
	Dx string `json:"Dx"`
}

type checkResult struct {
	Name    string
	Passed  bool
	Details string
}

func main() {
	strict := flag.Bool("strict", true, "Exit with non-zero code when warnings are found")
	fixLeaveStatus := flag.Bool("fix-leave-status", false, "Normalize legacy Expired leave status values to Completed before checks")
	flag.Parse()

	root, err := findAppRoot()
	if err != nil {
		fatalf("resolve app root: %v", err)
	}

	if err := loadDotEnv(filepath.Join(root, ".env")); err != nil {
		fatalf("load .env: %v", err)
	}

	cfg, err := loadConfig(root)
	if err != nil {
		fatalf("load config: %v", err)
	}
	applyEnvOverrides(&cfg)

	connStr, err := buildConnStr(cfg)
	if err != nil {
		fatalf("build db connection string: %v", err)
	}

	db, err := sql.Open("postgres", connStr)
	if err != nil {
		fatalf("open database: %v", err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		fatalf("ping database: %v", err)
	}

	schema := getenvDefault("DB_SCHEMA", "clinician_app")
	if *fixLeaveStatus {
		if err := normalizeLegacyLeaveStatuses(db, schema); err != nil {
			fatalf("normalize leave statuses: %v", err)
		}
	}

	results := runChecks(db, schema)
	printResults(results)

	hasFailures := false
	hasWarnings := false
	for _, result := range results {
		if !result.Passed {
			if strings.HasPrefix(result.Name, "WARN:") {
				hasWarnings = true
			} else {
				hasFailures = true
			}
		}
	}

	if hasFailures || (*strict && hasWarnings) {
		os.Exit(1)
	}
}

func normalizeLegacyLeaveStatuses(db *sql.DB, schema string) error {
	query := fmt.Sprintf(`
		UPDATE %s.staffleave
		SET leave_status = 'Completed'
		WHERE COALESCE(TRIM(leave_status), '') = 'Expired'
	`, schema)
	_, err := db.Exec(query)
	return err
}

func runChecks(db *sql.DB, schema string) []checkResult {
	results := []checkResult{}

	results = append(results, checkRequiredTables(db, schema))
	results = append(results, checkRequiredColumns(db, schema))
	results = append(results, checkCanonicalRights(db, schema))
	results = append(results, checkUserRoleAssignments(db, schema))
	results = append(results, checkEmployeeScopeAssignments(db, schema))
	results = append(results, checkWeeklyReportUniquenessAndSpan(db, schema))
	results = append(results, checkInvalidWorkflowStatuses(db, schema))

	return results
}

func checkRequiredTables(db *sql.DB, schema string) checkResult {
	required := []string{"users", "employees", "facilities", "departments", "rights", "employeerights", "weeklyreport", "staffleave"}
	missing := []string{}

	for _, table := range required {
		qualifiedName := fmt.Sprintf("%s.%s", schema, table)
		var regclass sql.NullString
		if err := db.QueryRow("SELECT to_regclass($1)", qualifiedName).Scan(&regclass); err != nil {
			return checkResult{Name: "Required tables", Passed: false, Details: err.Error()}
		}
		if !regclass.Valid || regclass.String == "" {
			missing = append(missing, qualifiedName)
		}
	}

	if len(missing) > 0 {
		return checkResult{Name: "Required tables", Passed: false, Details: "Missing: " + strings.Join(missing, ", ")}
	}
	return checkResult{Name: "Required tables", Passed: true, Details: "All required tables are present"}
}

func checkRequiredColumns(db *sql.DB, schema string) checkResult {
	requiredColumns := map[string][][]string{
		"users": {
			{"id"},
			{"username"},
			{"employees", "employeeid"},
			{"pssword", "password"},
		},
		"employees": {
			{"id"},
			{"facility"},
			{"department"},
			{"fname"},
			{"lname"},
		},
		"rights": {
			{"id"},
			{"rights", "r_name"},
		},
		"employeerights": {
			{"id"},
			{"employee"},
			{"rights"},
		},
		"weeklyreport": {
			{"id"},
			{"employee"},
			{"hospital"},
			{"department"},
			{"start"},
			{"submit_status"},
		},
		"staffleave": {
			{"leave_id"},
			{"employee_id"},
			{"leave_status"},
			{"created_on"},
			{"start_date"},
			{"end_date"},
		},
	}

	missing := []string{}
	for table, options := range requiredColumns {
		for _, option := range options {
			exists, err := hasAnyColumn(db, schema, table, option)
			if err != nil {
				return checkResult{Name: "Required columns", Passed: false, Details: err.Error()}
			}
			if !exists {
				missing = append(missing, fmt.Sprintf("%s.(%s)", table, strings.Join(option, "|")))
			}
		}
	}

	if len(missing) > 0 {
		return checkResult{Name: "Required columns", Passed: false, Details: "Missing: " + strings.Join(missing, ", ")}
	}

	return checkResult{Name: "Required columns", Passed: true, Details: "All required columns are present"}
}

func checkCanonicalRights(db *sql.DB, schema string) checkResult {
	expected := []string{"Facility Admin", "National Admin", "Staff"}
	actual := []string{}
	rightsColumn, err := detectFirstExistingColumn(db, schema, "rights", []string{"rights", "r_name"})
	if err != nil {
		return checkResult{Name: "Canonical rights", Passed: false, Details: err.Error()}
	}
	if rightsColumn == "" {
		return checkResult{Name: "Canonical rights", Passed: false, Details: "rights table is missing role-name column (rights or r_name)"}
	}

	query := fmt.Sprintf(`SELECT COALESCE(%s, '') FROM %s.rights ORDER BY %s`, rightsColumn, schema, rightsColumn)
	rows, err := db.Query(query)
	if err != nil {
		return checkResult{Name: "Canonical rights", Passed: false, Details: err.Error()}
	}
	defer rows.Close()

	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return checkResult{Name: "Canonical rights", Passed: false, Details: err.Error()}
		}
		actual = append(actual, strings.TrimSpace(name))
	}
	if err := rows.Err(); err != nil {
		return checkResult{Name: "Canonical rights", Passed: false, Details: err.Error()}
	}

	if len(actual) != len(expected) {
		return checkResult{Name: "Canonical rights", Passed: false, Details: fmt.Sprintf("Expected %v, found %v", expected, actual)}
	}
	for i := range expected {
		if actual[i] != expected[i] {
			return checkResult{Name: "Canonical rights", Passed: false, Details: fmt.Sprintf("Expected %v, found %v", expected, actual)}
		}
	}

	return checkResult{Name: "Canonical rights", Passed: true, Details: "rights table matches canonical role set"}
}

func checkUserRoleAssignments(db *sql.DB, schema string) checkResult {
	userEmployeeColumn, err := detectFirstExistingColumn(db, schema, "users", []string{"employees", "employeeid"})
	if err != nil {
		return checkResult{Name: "User role assignments", Passed: false, Details: err.Error()}
	}
	if userEmployeeColumn == "" {
		return checkResult{Name: "User role assignments", Passed: false, Details: "users table is missing employee mapping column (employees or employeeid)"}
	}

	query := fmt.Sprintf(`
		SELECT COUNT(*)
		FROM %s.users u
		LEFT JOIN %s.employeerights er ON er.employee = u.%s
		WHERE COALESCE(u.%s, 0) > 0 AND er.id IS NULL
	`, schema, schema, userEmployeeColumn, userEmployeeColumn)

	var missing int
	if err := db.QueryRow(query).Scan(&missing); err != nil {
		return checkResult{Name: "User role assignments", Passed: false, Details: err.Error()}
	}
	if missing > 0 {
		return checkResult{Name: "User role assignments", Passed: false, Details: fmt.Sprintf("%d users are missing employeerights mapping", missing)}
	}

	orphanQuery := fmt.Sprintf(`
		SELECT COUNT(*)
		FROM %s.employeerights er
		LEFT JOIN %s.rights r ON r.id = er.rights
		WHERE r.id IS NULL
	`, schema, schema)
	var orphaned int
	if err := db.QueryRow(orphanQuery).Scan(&orphaned); err != nil {
		return checkResult{Name: "User role assignments", Passed: false, Details: err.Error()}
	}
	if orphaned > 0 {
		return checkResult{Name: "User role assignments", Passed: false, Details: fmt.Sprintf("%d employeerights rows reference missing rights", orphaned)}
	}

	return checkResult{Name: "User role assignments", Passed: true, Details: "All user role mappings are valid"}
}

func checkEmployeeScopeAssignments(db *sql.DB, schema string) checkResult {
	query := fmt.Sprintf(`
		SELECT COUNT(*)
		FROM %s.employees
		WHERE COALESCE(facility, 0) <= 0 OR COALESCE(department, 0) <= 0
	`, schema)
	var invalid int
	if err := db.QueryRow(query).Scan(&invalid); err != nil {
		return checkResult{Name: "Employee scope assignments", Passed: false, Details: err.Error()}
	}
	if invalid > 0 {
		return checkResult{Name: "Employee scope assignments", Passed: false, Details: fmt.Sprintf("%d employees are missing facility/department assignment", invalid)}
	}

	return checkResult{Name: "Employee scope assignments", Passed: true, Details: "All employees have facility and department assignments"}
}

func checkWeeklyReportUniquenessAndSpan(db *sql.DB, schema string) checkResult {
	duplicateQuery := fmt.Sprintf(`
		WITH dupes AS (
			SELECT employee, hospital, start, COUNT(*) AS cnt
			FROM %s.weeklyreport
			WHERE employee IS NOT NULL AND hospital IS NOT NULL AND start IS NOT NULL
			GROUP BY employee, hospital, start
			HAVING COUNT(*) > 1
		)
		SELECT COUNT(*)::bigint, COALESCE(SUM(cnt - 1), 0)::bigint FROM dupes
	`, schema)

	var duplicateGroups int64
	var duplicateRows int64
	if err := db.QueryRow(duplicateQuery).Scan(&duplicateGroups, &duplicateRows); err != nil {
		return checkResult{Name: "Weekly report uniqueness/span", Passed: false, Details: err.Error()}
	}

	spanQuery := fmt.Sprintf(`
		SELECT COUNT(*)::bigint
		FROM %s.weeklyreport
		WHERE start IS NOT NULL
			AND stop IS NOT NULL
			AND stop <> start + INTERVAL '6 days'
	`, schema)
	var invalidSpan int64
	if err := db.QueryRow(spanQuery).Scan(&invalidSpan); err != nil {
		return checkResult{Name: "Weekly report uniqueness/span", Passed: false, Details: err.Error()}
	}

	overlapQuery := fmt.Sprintf(`
		SELECT COUNT(*)::bigint
		FROM %s.weeklyreport a
		JOIN %s.weeklyreport b
			ON a.id < b.id
			AND a.employee = b.employee
			AND a.hospital = b.hospital
			AND a.start IS NOT NULL AND a.stop IS NOT NULL
			AND b.start IS NOT NULL AND b.stop IS NOT NULL
			AND daterange(a.start, a.stop, '[]') && daterange(b.start, b.stop, '[]')
	`, schema, schema)
	var overlapConflicts int64
	if err := db.QueryRow(overlapQuery).Scan(&overlapConflicts); err != nil {
		return checkResult{Name: "Weekly report uniqueness/span", Passed: false, Details: err.Error()}
	}

	indexQuery := fmt.Sprintf(`
		SELECT EXISTS (
			SELECT 1
			FROM pg_indexes
			WHERE schemaname = %s
				AND tablename = 'weeklyreport'
				AND indexname = 'ux_weeklyreport_employee_hospital_start'
		)
	`, quoteLiteral(schema))
	var hasIndex bool
	if err := db.QueryRow(indexQuery).Scan(&hasIndex); err != nil {
		return checkResult{Name: "Weekly report uniqueness/span", Passed: false, Details: err.Error()}
	}

	exclusionConstraintQuery := fmt.Sprintf(`
		SELECT EXISTS (
			SELECT 1
			FROM pg_constraint
			WHERE conname = 'ex_weeklyreport_employee_hospital_no_overlap'
				AND conrelid = %s::regclass
		)
	`, quoteLiteral(schema+".weeklyreport"))
	var hasExclusionConstraint bool
	if err := db.QueryRow(exclusionConstraintQuery).Scan(&hasExclusionConstraint); err != nil {
		return checkResult{Name: "Weekly report uniqueness/span", Passed: false, Details: err.Error()}
	}

	if duplicateGroups > 0 || invalidSpan > 0 || overlapConflicts > 0 || !hasIndex || !hasExclusionConstraint {
		return checkResult{
			Name:    "Weekly report uniqueness/span",
			Passed:  false,
			Details: fmt.Sprintf("duplicate_groups=%d duplicate_rows=%d invalid_start_stop_span_rows=%d overlap_conflict_pairs=%d has_index=%t has_exclusion_constraint=%t", duplicateGroups, duplicateRows, invalidSpan, overlapConflicts, hasIndex, hasExclusionConstraint),
		}
	}

	return checkResult{
		Name:    "Weekly report uniqueness/span",
		Passed:  true,
		Details: fmt.Sprintf("duplicate_groups=%d invalid_start_stop_span_rows=%d overlap_conflict_pairs=%d has_index=%t has_exclusion_constraint=%t", duplicateGroups, invalidSpan, overlapConflicts, hasIndex, hasExclusionConstraint),
	}
}

func quoteLiteral(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "''") + "'"
}

func checkInvalidWorkflowStatuses(db *sql.DB, schema string) checkResult {
	warnings := []string{}

	invalidReportStatuses, err := readDistinctStatuses(db, fmt.Sprintf(`
		SELECT DISTINCT COALESCE(TRIM(submit_status), '')
		FROM %s.weeklyreport
		WHERE COALESCE(TRIM(submit_status), '') <> ''
			AND COALESCE(TRIM(submit_status), '') NOT IN ('Draft', 'Submitted', 'Approved', 'Declined', 'Rejected', 'Pending')
	`, schema))
	if err != nil {
		return checkResult{Name: "Workflow statuses", Passed: false, Details: err.Error()}
	}
	if len(invalidReportStatuses) > 0 {
		warnings = append(warnings, "weeklyreport.submit_status contains unexpected values: "+strings.Join(invalidReportStatuses, ", "))
	}

	invalidLeaveStatuses, err := readDistinctStatuses(db, fmt.Sprintf(`
		SELECT DISTINCT COALESCE(TRIM(leave_status), '')
		FROM %s.staffleave
		WHERE COALESCE(TRIM(leave_status), '') <> ''
			AND COALESCE(TRIM(leave_status), '') NOT IN ('Draft', 'Pending', 'Approved', 'Valid', 'Rejected', 'Declined', 'Completed')
	`, schema))
	if err != nil {
		return checkResult{Name: "Workflow statuses", Passed: false, Details: err.Error()}
	}
	if len(invalidLeaveStatuses) > 0 {
		warnings = append(warnings, "staffleave.leave_status contains unexpected values: "+strings.Join(invalidLeaveStatuses, ", "))
	}

	if len(warnings) > 0 {
		return checkResult{Name: "WARN: Workflow statuses", Passed: false, Details: strings.Join(warnings, " | ")}
	}

	return checkResult{Name: "Workflow statuses", Passed: true, Details: "Workflow status values are within expected sets"}
}

func readDistinctStatuses(db *sql.DB, query string) ([]string, error) {
	rows, err := db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	values := []string{}
	for rows.Next() {
		var value string
		if err := rows.Scan(&value); err != nil {
			return nil, err
		}
		values = append(values, value)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	sort.Strings(values)
	return values, nil
}

func hasAnyColumn(db *sql.DB, schema, table string, columns []string) (bool, error) {
	for _, col := range columns {
		var exists bool
		err := db.QueryRow(`
			SELECT EXISTS (
				SELECT 1
				FROM information_schema.columns
				WHERE table_schema = $1 AND table_name = $2 AND column_name = $3
			)
		`, schema, table, col).Scan(&exists)
		if err != nil {
			return false, err
		}
		if exists {
			return true, nil
		}
	}
	return false, nil
}

func detectFirstExistingColumn(db *sql.DB, schema, table string, columns []string) (string, error) {
	for _, col := range columns {
		exists, err := hasAnyColumn(db, schema, table, []string{col})
		if err != nil {
			return "", err
		}
		if exists {
			return col, nil
		}
	}
	return "", nil
}

func printResults(results []checkResult) {
	fmt.Println("Database workflow verification")
	fmt.Println(strings.Repeat("=", 32))
	for _, result := range results {
		status := "PASS"
		if !result.Passed {
			status = "FAIL"
			if strings.HasPrefix(result.Name, "WARN:") {
				status = "WARN"
			}
		}
		fmt.Printf("[%s] %s\n", status, result.Name)
		if strings.TrimSpace(result.Details) != "" {
			fmt.Printf("  %s\n", result.Details)
		}
	}
}

func loadConfig(root string) (config, error) {
	paths := []string{
		filepath.Join(root, "config.json"),
		filepath.Join(root, "clinician", "cmd", "web", "config.json"),
	}

	for _, path := range paths {
		file, err := os.Open(path)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return config{}, fmt.Errorf("open %s: %w", path, err)
		}

		var cfg config
		decodeErr := json.NewDecoder(file).Decode(&cfg)
		_ = file.Close()
		if decodeErr != nil {
			return config{}, fmt.Errorf("decode %s: %w", path, decodeErr)
		}
		return cfg, nil
	}

	return config{}, errors.New("config.json not found in expected locations")
}

func applyEnvOverrides(cfg *config) {
	if value := os.Getenv("DB_USER"); value != "" {
		cfg.Ux = value
	}
	if value := os.Getenv("DB_PASSWORD"); value != "" {
		cfg.Px = value
	}
	if value := os.Getenv("DB_NAME"); value != "" {
		cfg.Dx = value
	}
}

func buildConnStr(cfg config) (string, error) {
	host := getenvDefault("DB_HOST", "127.0.0.1")
	port := getenvDefault("DB_PORT", "5432")
	sslmode := getenvDefault("DB_SSLMODE", "disable")
	schema := getenvDefault("DB_SCHEMA", "clinician_app")
	user := strings.TrimSpace(cfg.Ux)
	password := cfg.Px
	dbname := strings.TrimSpace(cfg.Dx)

	if user == "" {
		return "", errors.New("database user is missing; set DB_USER in .env")
	}
	if dbname == "" {
		return "", errors.New("database name is missing; set DB_NAME in .env")
	}

	return fmt.Sprintf(
		"host=%s port=%s user=%s password='%s' dbname=%s sslmode=%s search_path=%s",
		host,
		port,
		user,
		password,
		dbname,
		sslmode,
		schema,
	), nil
}

func loadDotEnv(path string) error {
	file, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		value := strings.Trim(strings.TrimSpace(parts[1]), "\"")
		if key == "" {
			continue
		}
		if _, exists := os.LookupEnv(key); !exists {
			_ = os.Setenv(key, value)
		}
	}

	return scanner.Err()
}

func findAppRoot() (string, error) {
	if root := os.Getenv("CLINICIAN_APP_ROOT"); root != "" {
		return filepath.Abs(root)
	}

	wd, err := os.Getwd()
	if err != nil {
		return "", err
	}

	current := wd
	for {
		if dirExists(filepath.Join(current, "clinician", "ui", "html")) {
			return current, nil
		}
		if fileExists(filepath.Join(current, "config.json")) && dirExists(filepath.Join(current, "clinician")) {
			return current, nil
		}
		parent := filepath.Dir(current)
		if parent == current {
			break
		}
		current = parent
	}

	return "", errors.New("could not locate application root")
}

func getenvDefault(key string, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func dirExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

func fatalf(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(2)
}
