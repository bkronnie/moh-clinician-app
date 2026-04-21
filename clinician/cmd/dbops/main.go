package main

import (
	"bufio"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/moh/clinician/internals/models"

	_ "github.com/lib/pq"
)

type config struct {
	Ux string `json:"Ux"`
	Px string `json:"Px"`
	Dx string `json:"Dx"`
}

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(2)
	}

	root, err := findProjectRoot()
	if err != nil {
		fatalf("resolve project root: %v", err)
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

	switch os.Args[1] {
	case "inspect-weeklyreport":
		if err := runInspectWeeklyReport(db); err != nil {
			fatalf("inspect-weeklyreport: %v", err)
		}
	case "backdate-weeklyreport":
		if err := runBackdateWeeklyReport(db, os.Args[2:]); err != nil {
			fatalf("backdate-weeklyreport: %v", err)
		}
	case "verify-dashboard-kpis":
		if err := runVerifyDashboardKPIs(db); err != nil {
			fatalf("verify-dashboard-kpis: %v", err)
		}
	case "verify-staff-dashboard-all":
		if err := runVerifyStaffDashboardAll(db); err != nil {
			fatalf("verify-staff-dashboard-all: %v", err)
		}
	case "verify-staff-list-tables-all":
		if err := runVerifyStaffListTablesAll(db); err != nil {
			fatalf("verify-staff-list-tables-all: %v", err)
		}
	case "enforce-weeklyreport-integrity":
		if err := runEnforceWeeklyReportIntegrity(db, os.Args[2:]); err != nil {
			fatalf("enforce-weeklyreport-integrity: %v", err)
		}
	case "query":
		if err := runQuery(db, os.Args[2:]); err != nil {
			fatalf("query: %v", err)
		}
	default:
		printUsage()
		os.Exit(2)
	}
}

type checkResult struct {
	Name    string
	Passed  bool
	Details string
}

func runVerifyDashboardKPIs(db *sql.DB) error {
	ctx := contextWithTimeout()
	defer ctx.cancel()

	results := []checkResult{}

	results = append(results, checkRequiredTables(ctx.ctx, db))
	results = append(results, checkTableCounts(ctx.ctx, db))

	latestWeekStart, periodErr := latestDashboardPeriod(ctx.ctx, db)
	if periodErr != nil {
		results = append(results, checkResult{Name: "Dashboard periods", Passed: false, Details: periodErr.Error()})
		printCheckResults(results)
		return errors.New("dashboard verification failed")
	}
	latestYear, latestWeek := latestWeekStart.ISOWeek()
	latestMonth := int(latestWeekStart.Month())
	periodEnd := latestWeekStart.AddDate(0, 0, 6)
	results = append(results, checkResult{
		Name:    "Dashboard periods",
		Passed:  true,
		Details: fmt.Sprintf("latest_period_start=%s latest_period_end=%s", latestWeekStart.Format("2006-01-02"), periodEnd.Format("2006-01-02")),
	})

	results = append(results, checkNationalDashboardKPIs(ctx.ctx, db, latestWeekStart, periodEnd))

	facilityID, facilityFound, facilityErr := firstFacilityID(ctx.ctx, db)
	if facilityErr != nil {
		results = append(results, checkResult{Name: "Facility dashboard KPI consistency", Passed: false, Details: facilityErr.Error()})
	} else if !facilityFound {
		results = append(results, checkResult{Name: "Facility dashboard KPI consistency", Passed: true, Details: "skipped: no facilities found"})
	} else {
		results = append(results, checkFacilityDashboardKPIs(ctx.ctx, db, facilityID, latestWeekStart, periodEnd, latestYear, latestMonth, latestWeek))
	}

	empID, employeeFound, employeeErr := firstEmployeeID(ctx.ctx, db)
	if employeeErr != nil {
		results = append(results, checkResult{Name: "Staff dashboard KPI consistency", Passed: false, Details: employeeErr.Error()})
	} else if !employeeFound {
		results = append(results, checkResult{Name: "Staff dashboard KPI consistency", Passed: true, Details: "skipped: no employees found"})
	} else {
		results = append(results, checkStaffDashboardKPIs(ctx.ctx, db, empID))
	}

	printCheckResults(results)

	hasFailures := false
	for _, result := range results {
		if !result.Passed {
			hasFailures = true
			break
		}
	}
	if hasFailures {
		return errors.New("one or more dashboard/KPI checks failed")
	}

	return nil
}

func runVerifyStaffDashboardAll(db *sql.DB) error {
	ctx := contextWithTimeout()
	defer ctx.cancel()

	staffIDs, err := listStaffEmployeeIDs(ctx.ctx, db)
	if err != nil {
		return err
	}
	if len(staffIDs) == 0 {
		fmt.Println("staff_dashboard_verification")
		fmt.Println(strings.Repeat("=", 28))
		fmt.Println("staff_count=0")
		fmt.Println("status=PASS (no staff records found)")
		return nil
	}

	failures := []string{}
	checksRun := 0

	for _, employeeID := range staffIDs {
		// Scenario 1: all records (year/month/week = 0)
		snapshotAll, err := models.GetClinicianDashboardSnapshot(ctx.ctx, db, employeeID, 0, 0, 0)
		if err != nil {
			failures = append(failures, fmt.Sprintf("employee=%d scenario=all error=%v", employeeID, err))
			continue
		}
		checksRun++
		if mismatch := compareStaffSnapshotWithSQL(ctx.ctx, db, employeeID, "all", snapshotAll); mismatch != "" {
			failures = append(failures, mismatch)
		}

		if len(snapshotAll.AvailableYears) <= 1 {
			continue
		}

		latestYear := snapshotAll.AvailableYears[1]

		// Scenario 2: year filter
		snapshotYear, err := models.GetClinicianDashboardSnapshot(ctx.ctx, db, employeeID, latestYear, 0, 0)
		if err != nil {
			failures = append(failures, fmt.Sprintf("employee=%d scenario=year(%d) error=%v", employeeID, latestYear, err))
			continue
		}
		checksRun++
		if mismatch := compareStaffSnapshotWithSQL(ctx.ctx, db, employeeID, fmt.Sprintf("year(%d)", latestYear), snapshotYear); mismatch != "" {
			failures = append(failures, mismatch)
		}

		// Scenario 3: year + month filter (first non-zero month option if present)
		selectedMonth := 0
		for _, monthOption := range snapshotYear.AvailableMonths {
			if monthOption.ID > 0 {
				selectedMonth = monthOption.ID
				break
			}
		}
		if selectedMonth > 0 {
			snapshotMonth, err := models.GetClinicianDashboardSnapshot(ctx.ctx, db, employeeID, latestYear, selectedMonth, 0)
			if err != nil {
				failures = append(failures, fmt.Sprintf("employee=%d scenario=year(%d)-month(%d) error=%v", employeeID, latestYear, selectedMonth, err))
			} else {
				checksRun++
				if mismatch := compareStaffSnapshotWithSQL(ctx.ctx, db, employeeID, fmt.Sprintf("year(%d)-month(%d)", latestYear, selectedMonth), snapshotMonth); mismatch != "" {
					failures = append(failures, mismatch)
				}

				// Scenario 4: specific week within selected month if available
				selectedWeek := 0
				for _, weekOption := range snapshotMonth.AvailableWeeks {
					if weekOption.Week > 0 {
						selectedWeek = weekOption.Week
						break
					}
				}
				if selectedWeek > 0 {
					snapshotWeek, err := models.GetClinicianDashboardSnapshot(ctx.ctx, db, employeeID, latestYear, selectedMonth, selectedWeek)
					if err != nil {
						failures = append(failures, fmt.Sprintf("employee=%d scenario=year(%d)-month(%d)-week(%d) error=%v", employeeID, latestYear, selectedMonth, selectedWeek, err))
					} else {
						checksRun++
						if mismatch := compareStaffSnapshotWithSQL(ctx.ctx, db, employeeID, fmt.Sprintf("year(%d)-month(%d)-week(%d)", latestYear, selectedMonth, selectedWeek), snapshotWeek); mismatch != "" {
							failures = append(failures, mismatch)
						}
					}
				}
			}
		}
	}

	fmt.Println("staff_dashboard_verification")
	fmt.Println(strings.Repeat("=", 28))
	fmt.Printf("staff_count=%d\n", len(staffIDs))
	fmt.Printf("checks_run=%d\n", checksRun)
	fmt.Printf("failures=%d\n", len(failures))

	if len(failures) == 0 {
		fmt.Println("status=PASS")
		return nil
	}

	limit := len(failures)
	if limit > 25 {
		limit = 25
	}
	for i := 0; i < limit; i++ {
		fmt.Printf("failure_%02d=%s\n", i+1, failures[i])
	}
	if len(failures) > limit {
		fmt.Printf("failure_truncated=%d more\n", len(failures)-limit)
	}

	return errors.New("staff dashboard verification failed")
}

func runVerifyStaffListTablesAll(db *sql.DB) error {
	ctx := contextWithTimeout()
	defer ctx.cancel()

	staffIDs, err := listStaffEmployeeIDs(ctx.ctx, db)
	if err != nil {
		return err
	}
	if len(staffIDs) == 0 {
		fmt.Println("staff_list_tables_verification")
		fmt.Println(strings.Repeat("=", 31))
		fmt.Println("staff_count=0")
		fmt.Println("status=PASS (no staff records found)")
		return nil
	}

	statuses := []string{"all", "submitted", "pending", "approved", "declined", "draft"}
	failures := []string{}
	checksRun := 0

	for _, employeeID := range staffIDs {
		year, month, week, err := mostRecentEmployeePeriod(ctx.ctx, db, employeeID)
		if err != nil {
			failures = append(failures, fmt.Sprintf("employee=%d period_lookup_error=%v", employeeID, err))
			continue
		}

		scenarios := []struct {
			name  string
			year  int
			month int
			week  int
		}{
			{name: "all_periods", year: 0, month: 0, week: 0},
			{name: fmt.Sprintf("year_%d", year), year: year, month: 0, week: 0},
			{name: fmt.Sprintf("year_%d_month_%d", year, month), year: year, month: month, week: 0},
			{name: fmt.Sprintf("year_%d_month_%d_week_%d", year, month, week), year: year, month: month, week: week},
		}

		for _, scenario := range scenarios {
			for _, status := range statuses {
				checksRun++
				if mismatch := verifyStaffReportSubmissionsListScenario(ctx.ctx, db, employeeID, status, scenario.year, scenario.month, scenario.week, scenario.name); mismatch != "" {
					failures = append(failures, mismatch)
				}
			}
		}
	}

	fmt.Println("staff_list_tables_verification")
	fmt.Println(strings.Repeat("=", 31))
	fmt.Printf("staff_count=%d\n", len(staffIDs))
	fmt.Printf("checks_run=%d\n", checksRun)
	fmt.Printf("failures=%d\n", len(failures))

	if len(failures) == 0 {
		fmt.Println("status=PASS")
		return nil
	}

	limit := len(failures)
	if limit > 30 {
		limit = 30
	}
	for i := 0; i < limit; i++ {
		fmt.Printf("failure_%02d=%s\n", i+1, failures[i])
	}
	if len(failures) > limit {
		fmt.Printf("failure_truncated=%d more\n", len(failures)-limit)
	}

	return errors.New("staff list-table verification failed")
}

func mostRecentEmployeePeriod(ctx context.Context, db *sql.DB, employeeID int) (int, int, int, error) {
	var year int
	var month int
	var week int
	err := db.QueryRowContext(ctx, `
		SELECT
			EXTRACT(ISOYEAR FROM start)::int,
			EXTRACT(MONTH FROM start)::int,
			EXTRACT(WEEK FROM start)::int
		FROM clinician_app.weeklyreport
		WHERE employee = $1
		ORDER BY start DESC
		LIMIT 1
	`, employeeID).Scan(&year, &month, &week)
	if err == sql.ErrNoRows {
		now := time.Now()
		y, w := now.ISOWeek()
		return y, int(now.Month()), w, nil
	}
	if err != nil {
		return 0, 0, 0, err
	}
	return year, month, week, nil
}

func verifyStaffReportSubmissionsListScenario(ctx context.Context, db *sql.DB, employeeID int, status string, year int, month int, week int, scenario string) string {
	rows, totalRows, err := models.GetReportSubmissionsPaged(ctx, db, 0, int64(employeeID), int64(employeeID), 0, 0, status, year, month, week, 0, 0)
	if err != nil {
		return fmt.Sprintf("employee=%d scenario=%s status=%s query_error=%v", employeeID, scenario, status, err)
	}

	var expectedTotal int
	expectedSQL := `
		SELECT COUNT(*)
		FROM clinician_app.weeklyreport w
		WHERE w.employee = $1
	`
	args := []interface{}{employeeID}
	argPos := 2

	if year > 0 {
		expectedSQL += fmt.Sprintf(" AND EXTRACT(ISOYEAR FROM w.start) = $%d", argPos)
		args = append(args, year)
		argPos++
	}
	if month > 0 {
		expectedSQL += fmt.Sprintf(" AND EXTRACT(MONTH FROM w.start) = $%d", argPos)
		args = append(args, month)
		argPos++
	}
	if week > 0 {
		expectedSQL += fmt.Sprintf(" AND EXTRACT(WEEK FROM w.start) = $%d", argPos)
		args = append(args, week)
		argPos++
	}

	switch status {
	case "submitted", "pending":
		expectedSQL += " AND COALESCE(w.submit_status, '') = 'Submitted'"
		expectedSQL += " AND COALESCE(w.report_status, '') NOT IN ('Approved', 'Rejected', 'Declined')"
	case "approved":
		expectedSQL += " AND COALESCE(w.report_status, '') = 'Approved'"
	case "declined":
		expectedSQL += " AND COALESCE(w.report_status, '') IN ('Rejected', 'Declined')"
	case "draft":
		expectedSQL += " AND COALESCE(w.submit_status, '') <> 'Submitted'"
		expectedSQL += " AND COALESCE(w.report_status, '') NOT IN ('Approved', 'Rejected', 'Declined')"
	}

	if err := db.QueryRowContext(ctx, expectedSQL, args...).Scan(&expectedTotal); err != nil {
		return fmt.Sprintf("employee=%d scenario=%s status=%s expected_sql_error=%v", employeeID, scenario, status, err)
	}

	if totalRows != expectedTotal || len(rows) != expectedTotal {
		return fmt.Sprintf("employee=%d scenario=%s status=%s total_mismatch actual_total=%d actual_len=%d expected=%d", employeeID, scenario, status, totalRows, len(rows), expectedTotal)
	}

	for _, row := range rows {
		if row == nil {
			return fmt.Sprintf("employee=%d scenario=%s status=%s nil_row_returned", employeeID, scenario, status)
		}
		if int(row.EmployeeID) != employeeID {
			return fmt.Sprintf("employee=%d scenario=%s status=%s employee_scope_leak row_employee=%d", employeeID, scenario, status, row.EmployeeID)
		}
		submitStatus := ""
		if row.SubmitStatus.Valid {
			submitStatus = strings.TrimSpace(row.SubmitStatus.String)
		}
		reportStatus := ""
		if row.ReportStatus.Valid {
			reportStatus = strings.TrimSpace(row.ReportStatus.String)
		}
		switch status {
		case "submitted", "pending":
			if submitStatus != "Submitted" || reportStatus == "Approved" || reportStatus == "Rejected" || reportStatus == "Declined" {
				return fmt.Sprintf("employee=%d scenario=%s status=%s row_status_mismatch submit=%q report=%q", employeeID, scenario, status, submitStatus, reportStatus)
			}
		case "approved":
			if reportStatus != "Approved" {
				return fmt.Sprintf("employee=%d scenario=%s status=%s row_status_mismatch report=%q", employeeID, scenario, status, reportStatus)
			}
		case "declined":
			if reportStatus != "Rejected" && reportStatus != "Declined" {
				return fmt.Sprintf("employee=%d scenario=%s status=%s row_status_mismatch report=%q", employeeID, scenario, status, reportStatus)
			}
		case "draft":
			if submitStatus == "Submitted" || reportStatus == "Approved" || reportStatus == "Rejected" || reportStatus == "Declined" {
				return fmt.Sprintf("employee=%d scenario=%s status=%s row_status_mismatch submit=%q report=%q", employeeID, scenario, status, submitStatus, reportStatus)
			}
		}
	}

	return ""
}

func listStaffEmployeeIDs(ctx context.Context, db *sql.DB) ([]int, error) {
	rows, err := db.QueryContext(ctx, `
		WITH latest AS (
			SELECT DISTINCT ON (er.employee)
				er.employee,
				COALESCE(r.rights, '') AS role_name
			FROM clinician_app.employeerights er
			JOIN clinician_app.rights r ON r.id = er.rights
			ORDER BY er.employee, er.id DESC
		)
		SELECT employee
		FROM latest
		WHERE LOWER(TRIM(role_name)) = 'staff'
		ORDER BY employee
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	ids := []int{}
	for rows.Next() {
		var id int
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

func compareStaffSnapshotWithSQL(ctx context.Context, db *sql.DB, employeeID int, scenario string, snapshot models.ClinicianDashboardSnapshot) string {
	var weeksEntered int
	var weeksSubmitted int
	var daysWorked int
	var wardRounds int
	var patientsReviewed int
	var procedures int

	if snapshot.SelectedYear == 0 {
		if err := db.QueryRowContext(ctx, `
			SELECT
				COUNT(DISTINCT start),
				COUNT(DISTINCT CASE WHEN submit_status = 'Submitted' THEN start END),
				COALESCE(SUM(attendance), 0),
				COALESCE(SUM(ward_rounds), 0),
				COALESCE(SUM(patients_reviewed), 0),
				COALESCE(SUM(elective), 0) + COALESCE(SUM(emergency), 0)
			FROM clinician_app.weeklyreport
			WHERE employee = $1
		`, employeeID).Scan(&weeksEntered, &weeksSubmitted, &daysWorked, &wardRounds, &patientsReviewed, &procedures); err != nil {
			return fmt.Sprintf("employee=%d scenario=%s sql_error=%v", employeeID, scenario, err)
		}
	} else if snapshot.SelectedWeek > 0 && strings.TrimSpace(snapshot.SelectedWeekStart) != "" {
		if err := db.QueryRowContext(ctx, `
			SELECT
				COUNT(DISTINCT start),
				COUNT(DISTINCT CASE WHEN submit_status = 'Submitted' THEN start END),
				COALESCE(SUM(attendance), 0),
				COALESCE(SUM(ward_rounds), 0),
				COALESCE(SUM(patients_reviewed), 0),
				COALESCE(SUM(elective), 0) + COALESCE(SUM(emergency), 0)
			FROM clinician_app.weeklyreport
			WHERE employee = $1 AND start = $2::date
		`, employeeID, snapshot.SelectedWeekStart).Scan(&weeksEntered, &weeksSubmitted, &daysWorked, &wardRounds, &patientsReviewed, &procedures); err != nil {
			return fmt.Sprintf("employee=%d scenario=%s sql_error=%v", employeeID, scenario, err)
		}
	} else if snapshot.SelectedMonth > 0 {
		if err := db.QueryRowContext(ctx, `
			SELECT
				COUNT(DISTINCT start),
				COUNT(DISTINCT CASE WHEN submit_status = 'Submitted' THEN start END),
				COALESCE(SUM(attendance), 0),
				COALESCE(SUM(ward_rounds), 0),
				COALESCE(SUM(patients_reviewed), 0),
				COALESCE(SUM(elective), 0) + COALESCE(SUM(emergency), 0)
			FROM clinician_app.weeklyreport
			WHERE employee = $1
				AND EXTRACT(ISOYEAR FROM start) = $2
				AND EXTRACT(MONTH FROM start) = $3
		`, employeeID, snapshot.SelectedYear, snapshot.SelectedMonth).Scan(&weeksEntered, &weeksSubmitted, &daysWorked, &wardRounds, &patientsReviewed, &procedures); err != nil {
			return fmt.Sprintf("employee=%d scenario=%s sql_error=%v", employeeID, scenario, err)
		}
	} else {
		if err := db.QueryRowContext(ctx, `
			SELECT
				COUNT(DISTINCT start),
				COUNT(DISTINCT CASE WHEN submit_status = 'Submitted' THEN start END),
				COALESCE(SUM(attendance), 0),
				COALESCE(SUM(ward_rounds), 0),
				COALESCE(SUM(patients_reviewed), 0),
				COALESCE(SUM(elective), 0) + COALESCE(SUM(emergency), 0)
			FROM clinician_app.weeklyreport
			WHERE employee = $1
				AND EXTRACT(ISOYEAR FROM start) = $2
		`, employeeID, snapshot.SelectedYear).Scan(&weeksEntered, &weeksSubmitted, &daysWorked, &wardRounds, &patientsReviewed, &procedures); err != nil {
			return fmt.Sprintf("employee=%d scenario=%s sql_error=%v", employeeID, scenario, err)
		}
	}

	var totalReports int
	var submittedReports int
	var approvedReports int
	var declinedReports int

	whereClause := `WHERE w.employee = $1`
	args := []interface{}{employeeID}
	argPos := 2
	if snapshot.SelectedYear > 0 {
		whereClause += fmt.Sprintf(" AND EXTRACT(ISOYEAR FROM w.start) = $%d", argPos)
		args = append(args, snapshot.SelectedYear)
		argPos++
	}
	if snapshot.SelectedMonth > 0 {
		whereClause += fmt.Sprintf(" AND EXTRACT(MONTH FROM w.start) = $%d", argPos)
		args = append(args, snapshot.SelectedMonth)
		argPos++
	}
	if snapshot.SelectedWeek > 0 {
		whereClause += fmt.Sprintf(" AND EXTRACT(WEEK FROM w.start) = $%d", argPos)
		args = append(args, snapshot.SelectedWeek)
		argPos++
	}

	sqlstr := `
		WITH report_groups AS (
			SELECT
				w.start,
				MAX(COALESCE(w.submit_status, '')) AS submit_status_text,
				MAX(COALESCE(w.report_status, '')) AS report_status_text
			FROM clinician_app.weeklyreport w
			` + whereClause + `
			GROUP BY w.start
		)
		SELECT
			COUNT(*) AS total_reports,
			COUNT(*) FILTER (
				WHERE submit_status_text = 'Submitted'
				AND COALESCE(report_status_text, '') NOT IN ('Approved', 'Rejected', 'Declined')
			) AS submitted_reports,
			COUNT(*) FILTER (WHERE report_status_text = 'Approved') AS approved_reports,
			COUNT(*) FILTER (WHERE report_status_text IN ('Rejected', 'Declined')) AS declined_reports
		FROM report_groups
	`

	if err := db.QueryRowContext(ctx, sqlstr, args...).Scan(&totalReports, &submittedReports, &approvedReports, &declinedReports); err != nil {
		return fmt.Sprintf("employee=%d scenario=%s report_summary_sql_error=%v", employeeID, scenario, err)
	}

	if snapshot.WeeksEntered != weeksEntered ||
		snapshot.WeeksSubmitted != weeksSubmitted ||
		snapshot.DaysWorked != daysWorked ||
		snapshot.WardRounds != wardRounds ||
		snapshot.PatientsReviewed != patientsReviewed ||
		snapshot.Procedures != procedures ||
		snapshot.TotalReports != totalReports ||
		snapshot.SubmittedReports != submittedReports ||
		snapshot.ApprovedReports != approvedReports ||
		snapshot.DeclinedReports != declinedReports {
		return fmt.Sprintf("employee=%d scenario=%s actual: weeks_entered=%d weeks_submitted=%d days=%d rounds=%d patients=%d procedures=%d total=%d submitted=%d approved=%d declined=%d | expected: weeks_entered=%d weeks_submitted=%d days=%d rounds=%d patients=%d procedures=%d total=%d submitted=%d approved=%d declined=%d", employeeID, scenario, snapshot.WeeksEntered, snapshot.WeeksSubmitted, snapshot.DaysWorked, snapshot.WardRounds, snapshot.PatientsReviewed, snapshot.Procedures, snapshot.TotalReports, snapshot.SubmittedReports, snapshot.ApprovedReports, snapshot.DeclinedReports, weeksEntered, weeksSubmitted, daysWorked, wardRounds, patientsReviewed, procedures, totalReports, submittedReports, approvedReports, declinedReports)
	}

	return ""
}

type runContext struct {
	ctx    context.Context
	cancel context.CancelFunc
}

func contextWithTimeout() runContext {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	return runContext{ctx: ctx, cancel: cancel}
}

func checkRequiredTables(ctx context.Context, db *sql.DB) checkResult {
	required := []string{
		"clinician_app.facilities",
		"clinician_app.departments",
		"clinician_app.employees",
		"clinician_app.weeklyreport",
		"clinician_app.staffleave",
		"clinician_app.attendance_records",
		"clinician_app.ward_rounds",
		"clinician_app.targets",
		"clinician_app.rights",
		"clinician_app.employeerights",
	}

	missing := []string{}
	for _, table := range required {
		var regclass sql.NullString
		if err := db.QueryRowContext(ctx, "SELECT to_regclass($1)", table).Scan(&regclass); err != nil {
			return checkResult{Name: "Required tables", Passed: false, Details: err.Error()}
		}
		if !regclass.Valid || regclass.String == "" {
			missing = append(missing, table)
		}
	}

	if len(missing) > 0 {
		return checkResult{Name: "Required tables", Passed: false, Details: "missing: " + strings.Join(missing, ", ")}
	}
	return checkResult{Name: "Required tables", Passed: true, Details: "all required tables found"}
}

func checkTableCounts(ctx context.Context, db *sql.DB) checkResult {
	tables := []string{"facilities", "departments", "employees", "weeklyreport", "staffleave", "attendance_records", "ward_rounds", "targets", "rights", "employeerights"}
	parts := []string{}
	for _, table := range tables {
		var count int64
		query := fmt.Sprintf("SELECT COUNT(*)::bigint FROM clinician_app.%s", table)
		if err := db.QueryRowContext(ctx, query).Scan(&count); err != nil {
			return checkResult{Name: "Table row counts", Passed: false, Details: fmt.Sprintf("%s: %v", table, err)}
		}
		parts = append(parts, fmt.Sprintf("%s=%d", table, count))
	}
	return checkResult{Name: "Table row counts", Passed: true, Details: strings.Join(parts, " | ")}
}

func latestDashboardPeriod(ctx context.Context, db *sql.DB) (time.Time, error) {
	periods, err := models.GetDashboardReportPeriods(ctx, db)
	if err != nil {
		return time.Time{}, err
	}
	if len(periods) == 0 {
		return time.Time{}, errors.New("no report periods found in weeklyreport.start")
	}
	return periods[0], nil
}

func checkNationalDashboardKPIs(ctx context.Context, db *sql.DB, periodStart time.Time, periodEnd time.Time) checkResult {
	snapshot, err := models.GetNationalDashboardSnapshotByRange(ctx, db, periodStart, periodEnd, 0, 0, 0)
	if err != nil {
		return checkResult{Name: "National dashboard KPI consistency", Passed: false, Details: err.Error()}
	}

	var expectedFacilities int
	var expectedClinicians int
	var expectedEntered int
	var expectedSubmitted int
	var expectedPending int

	const sqlstr = `
		WITH employee_scope AS (
			SELECT id, facility
			FROM clinician_app.employees
		),
		filtered_reports AS (
			SELECT w.employee, w.submit_status, w.report_status
			FROM clinician_app.weeklyreport w
			JOIN employee_scope es ON es.id = w.employee
			WHERE w.start BETWEEN $1 AND $2
		)
		SELECT
			(SELECT COUNT(*) FROM clinician_app.facilities) AS total_facilities,
			(SELECT COUNT(*) FROM employee_scope) AS total_clinicians,
			COUNT(DISTINCT fr.employee) AS entered_reports,
			COUNT(DISTINCT CASE WHEN COALESCE(fr.submit_status, '') = 'Submitted' THEN fr.employee END) AS submitted_reports,
			COUNT(DISTINCT CASE
				WHEN COALESCE(fr.submit_status, '') = 'Submitted'
					AND COALESCE(fr.report_status, '') NOT IN ('Approved', 'Rejected', 'Declined')
				THEN fr.employee
			END) AS pending_approval
		FROM filtered_reports fr
	`

	if err := db.QueryRowContext(ctx, sqlstr, periodStart, periodEnd).Scan(
		&expectedFacilities,
		&expectedClinicians,
		&expectedEntered,
		&expectedSubmitted,
		&expectedPending,
	); err != nil {
		return checkResult{Name: "National dashboard KPI consistency", Passed: false, Details: err.Error()}
	}

	expectedRate := percentageInt(expectedSubmitted, expectedClinicians)
	if snapshot.TotalFacilities != expectedFacilities ||
		snapshot.TotalClinicians != expectedClinicians ||
		snapshot.ReportsEnteredThisWeek != expectedEntered ||
		snapshot.SubmittedThisWeek != expectedSubmitted ||
		snapshot.PendingApproval != expectedPending ||
		snapshot.NationalReportingRate != expectedRate {
		return checkResult{
			Name:   "National dashboard KPI consistency",
			Passed: false,
			Details: fmt.Sprintf("actual: facilities=%d clinicians=%d entered=%d submitted=%d pending=%d rate=%d | expected: facilities=%d clinicians=%d entered=%d submitted=%d pending=%d rate=%d",
				snapshot.TotalFacilities, snapshot.TotalClinicians, snapshot.ReportsEnteredThisWeek, snapshot.SubmittedThisWeek, snapshot.PendingApproval, snapshot.NationalReportingRate,
				expectedFacilities, expectedClinicians, expectedEntered, expectedSubmitted, expectedPending, expectedRate,
			),
		}
	}

	return checkResult{
		Name:    "National dashboard KPI consistency",
		Passed:  true,
		Details: fmt.Sprintf("facilities=%d clinicians=%d entered=%d submitted=%d pending=%d rate=%d", snapshot.TotalFacilities, snapshot.TotalClinicians, snapshot.ReportsEnteredThisWeek, snapshot.SubmittedThisWeek, snapshot.PendingApproval, snapshot.NationalReportingRate),
	}
}

func firstFacilityID(ctx context.Context, db *sql.DB) (int, bool, error) {
	var id int
	err := db.QueryRowContext(ctx, `SELECT id FROM clinician_app.facilities ORDER BY id LIMIT 1`).Scan(&id)
	if err == sql.ErrNoRows {
		return 0, false, nil
	}
	if err != nil {
		return 0, false, err
	}
	return id, true, nil
}

func checkFacilityDashboardKPIs(ctx context.Context, db *sql.DB, facilityID int, periodStart time.Time, periodEnd time.Time, year int, month int, week int) checkResult {
	snapshot, err := models.GetFacilityManagerDashboardSnapshotByRange(ctx, db, facilityID, 0, 0, periodStart, periodEnd)
	if err != nil {
		return checkResult{Name: "Facility dashboard KPI consistency", Passed: false, Details: err.Error()}
	}

	var expectedClinicians int
	var expectedEntered int
	var expectedSubmitted int
	const sqlstr = `
		WITH employee_scope AS (
			SELECT id
			FROM clinician_app.employees
			WHERE facility = $1
		),
		report_scope AS (
			SELECT employee, submit_status
			FROM clinician_app.weeklyreport
			WHERE hospital = $1 AND start BETWEEN $2 AND $3
		)
		SELECT
			(SELECT COUNT(*) FROM employee_scope) AS total_clinicians,
			COUNT(DISTINCT rs.employee) AS entered_this_period,
			COUNT(DISTINCT CASE WHEN COALESCE(rs.submit_status, '') = 'Submitted' THEN rs.employee END) AS submitted_this_period
		FROM employee_scope es
		LEFT JOIN report_scope rs ON rs.employee = es.id
	`

	if err := db.QueryRowContext(ctx, sqlstr, facilityID, periodStart, periodEnd).Scan(&expectedClinicians, &expectedEntered, &expectedSubmitted); err != nil {
		return checkResult{Name: "Facility dashboard KPI consistency", Passed: false, Details: err.Error()}
	}
	expectedPending := expectedClinicians - expectedSubmitted
	if expectedPending < 0 {
		expectedPending = 0
	}

	if snapshot.TotalClinicians != expectedClinicians || snapshot.ReportsEnteredThisWeek != expectedEntered || snapshot.SubmittedThisWeek != expectedSubmitted || snapshot.PendingSubmission != expectedPending {
		return checkResult{
			Name:    "Facility dashboard KPI consistency",
			Passed:  false,
			Details: fmt.Sprintf("facility=%d actual: clinicians=%d entered=%d submitted=%d pending=%d | expected: clinicians=%d entered=%d submitted=%d pending=%d", facilityID, snapshot.TotalClinicians, snapshot.ReportsEnteredThisWeek, snapshot.SubmittedThisWeek, snapshot.PendingSubmission, expectedClinicians, expectedEntered, expectedSubmitted, expectedPending),
		}
	}

	reportSummary, err := models.GetFacilityReportReviewSummary(ctx, db, int64(facilityID))
	if err != nil {
		return checkResult{Name: "Facility dashboard KPI consistency", Passed: false, Details: fmt.Sprintf("facility summary: %v", err)}
	}

	leaveSummary, err := models.GetFacilityLeaveReviewSummary(ctx, db, int64(facilityID), 0, year, month, week)
	if err != nil {
		return checkResult{Name: "Facility dashboard KPI consistency", Passed: false, Details: fmt.Sprintf("leave summary: %v", err)}
	}

	return checkResult{
		Name:    "Facility dashboard KPI consistency",
		Passed:  true,
		Details: fmt.Sprintf("facility=%d clinicians=%d entered=%d submitted=%d pending=%d review_pending=%d leave_pending=%d", facilityID, snapshot.TotalClinicians, snapshot.ReportsEnteredThisWeek, snapshot.SubmittedThisWeek, snapshot.PendingSubmission, reportSummary.PendingReports, leaveSummary.PendingLeaves),
	}
}

func firstEmployeeID(ctx context.Context, db *sql.DB) (int, bool, error) {
	var id int
	err := db.QueryRowContext(ctx, `
		SELECT e.id
		FROM clinician_app.employees e
		WHERE EXISTS (
			SELECT 1
			FROM clinician_app.weeklyreport w
			WHERE w.employee = e.id
		)
		ORDER BY e.id
		LIMIT 1
	`).Scan(&id)
	if err == sql.ErrNoRows {
		err = db.QueryRowContext(ctx, `SELECT id FROM clinician_app.employees ORDER BY id LIMIT 1`).Scan(&id)
	}
	if err == sql.ErrNoRows {
		return 0, false, nil
	}
	if err != nil {
		return 0, false, err
	}
	return id, true, nil
}

func checkStaffDashboardKPIs(ctx context.Context, db *sql.DB, employeeID int) checkResult {
	snapshot, err := models.GetClinicianDashboardSnapshot(ctx, db, employeeID, 0, 0, 0)
	if err != nil {
		return checkResult{Name: "Staff dashboard KPI consistency", Passed: false, Details: err.Error()}
	}

	var weeksEntered int
	var weeksSubmitted int
	var daysWorked int
	var wardRounds int
	var patientsReviewed int
	var procedures int

	if err := db.QueryRowContext(ctx, `
		SELECT
			COUNT(DISTINCT start),
			COUNT(DISTINCT CASE WHEN submit_status = 'Submitted' THEN start END),
			COALESCE(SUM(attendance), 0),
			COALESCE(SUM(ward_rounds), 0),
			COALESCE(SUM(patients_reviewed), 0),
			COALESCE(SUM(elective), 0) + COALESCE(SUM(emergency), 0)
		FROM clinician_app.weeklyreport
		WHERE employee = $1
	`, employeeID).Scan(&weeksEntered, &weeksSubmitted, &daysWorked, &wardRounds, &patientsReviewed, &procedures); err != nil {
		return checkResult{Name: "Staff dashboard KPI consistency", Passed: false, Details: err.Error()}
	}

	var totalReports int
	var submittedReports int
	var approvedReports int
	var declinedReports int
	if err := db.QueryRowContext(ctx, `
		WITH report_groups AS (
			SELECT
				w.start,
				MAX(COALESCE(w.submit_status, '')) AS submit_status_text,
				MAX(COALESCE(w.report_status, '')) AS report_status_text
			FROM clinician_app.weeklyreport w
			WHERE w.employee = $1
			GROUP BY w.start
		)
		SELECT
			COUNT(*) AS total_reports,
			COUNT(*) FILTER (
				WHERE submit_status_text = 'Submitted'
				AND COALESCE(report_status_text, '') NOT IN ('Approved', 'Rejected', 'Declined')
			) AS submitted_reports,
			COUNT(*) FILTER (WHERE report_status_text = 'Approved') AS approved_reports,
			COUNT(*) FILTER (WHERE report_status_text IN ('Rejected', 'Declined')) AS declined_reports
		FROM report_groups
	`, employeeID).Scan(&totalReports, &submittedReports, &approvedReports, &declinedReports); err != nil {
		return checkResult{Name: "Staff dashboard KPI consistency", Passed: false, Details: err.Error()}
	}

	if snapshot.WeeksEntered != weeksEntered ||
		snapshot.WeeksSubmitted != weeksSubmitted ||
		snapshot.DaysWorked != daysWorked ||
		snapshot.WardRounds != wardRounds ||
		snapshot.PatientsReviewed != patientsReviewed ||
		snapshot.Procedures != procedures ||
		snapshot.TotalReports != totalReports ||
		snapshot.SubmittedReports != submittedReports ||
		snapshot.ApprovedReports != approvedReports ||
		snapshot.DeclinedReports != declinedReports {
		return checkResult{
			Name:    "Staff dashboard KPI consistency",
			Passed:  false,
			Details: fmt.Sprintf("employee=%d actual: weeks_entered=%d weeks_submitted=%d days=%d rounds=%d patients=%d procedures=%d total=%d submitted=%d approved=%d declined=%d | expected: weeks_entered=%d weeks_submitted=%d days=%d rounds=%d patients=%d procedures=%d total=%d submitted=%d approved=%d declined=%d", employeeID, snapshot.WeeksEntered, snapshot.WeeksSubmitted, snapshot.DaysWorked, snapshot.WardRounds, snapshot.PatientsReviewed, snapshot.Procedures, snapshot.TotalReports, snapshot.SubmittedReports, snapshot.ApprovedReports, snapshot.DeclinedReports, weeksEntered, weeksSubmitted, daysWorked, wardRounds, patientsReviewed, procedures, totalReports, submittedReports, approvedReports, declinedReports),
		}
	}

	return checkResult{
		Name:    "Staff dashboard KPI consistency",
		Passed:  true,
		Details: fmt.Sprintf("employee=%d weeks_entered=%d submitted_reports=%d approved_reports=%d declined_reports=%d", employeeID, snapshot.WeeksEntered, snapshot.SubmittedReports, snapshot.ApprovedReports, snapshot.DeclinedReports),
	}
}

func percentageInt(numerator int, denominator int) int {
	if denominator <= 0 {
		return 0
	}
	value := int(float64(numerator) / float64(denominator) * 100)
	if value < 0 {
		return 0
	}
	if value > 100 {
		return 100
	}
	return value
}

func printCheckResults(results []checkResult) {
	fmt.Println("dashboard_kpi_verification")
	fmt.Println(strings.Repeat("=", 30))
	for _, result := range results {
		status := "PASS"
		if !result.Passed {
			status = "FAIL"
		}
		fmt.Printf("[%s] %s\n", status, result.Name)
		if strings.TrimSpace(result.Details) != "" {
			fmt.Printf("  %s\n", result.Details)
		}
	}
}

func runInspectWeeklyReport(db *sql.DB) error {
	const sqlstr = `
		SELECT
			COUNT(*)::bigint AS total_rows,
			MIN(start) AS min_start,
			MAX(start) AS max_start,
			SUM(CASE WHEN start >= CURRENT_DATE THEN 1 ELSE 0 END)::bigint AS starts_today_or_future
		FROM clinician_app.weeklyreport
	`

	var totalRows int64
	var minStart sql.NullTime
	var maxStart sql.NullTime
	var startsTodayOrFuture int64

	if err := db.QueryRow(sqlstr).Scan(&totalRows, &minStart, &maxStart, &startsTodayOrFuture); err != nil {
		return err
	}

	fmt.Printf("total_rows=%d\n", totalRows)
	fmt.Printf("min_start=%s\n", formatNullDate(minStart))
	fmt.Printf("max_start=%s\n", formatNullDate(maxStart))
	fmt.Printf("starts_today_or_future=%d\n", startsTodayOrFuture)
	return nil
}

func runBackdateWeeklyReport(db *sql.DB, args []string) error {
	fs := flag.NewFlagSet("backdate-weeklyreport", flag.ContinueOnError)
	fs.SetOutput(os.Stdout)
	daysAgo := fs.Int("days-ago", 1, "base offset in days before today")
	dryRun := fs.Bool("dry-run", false, "show how many rows would be affected without updating")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *daysAgo < 1 {
		return errors.New("--days-ago must be >= 1")
	}

	if *dryRun {
		var count int64
		if err := db.QueryRow(`SELECT COUNT(*)::bigint FROM clinician_app.weeklyreport`).Scan(&count); err != nil {
			return err
		}
		fmt.Printf("dry_run_rows=%d\n", count)
		fmt.Printf("dry_run_note=unique(employee,start,stop) is preserved by row-number day offsets per employee\n")
		return nil
	}

	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	const updateSQL = `
		WITH ranked AS (
			SELECT
				id,
				employee,
				ROW_NUMBER() OVER (
					PARTITION BY employee
					ORDER BY COALESCE(start, CURRENT_DATE), id
				) AS rn
			FROM clinician_app.weeklyreport
		), targets AS (
			SELECT
				id,
				(CURRENT_DATE - ($1::int + rn - 1))::date AS target_date,
				(CURRENT_DATE - ($1::int + rn - 1))::timestamp AS target_ts
			FROM ranked
		)
		UPDATE clinician_app.weeklyreport w
		SET
			start = t.target_date,
			stop = t.target_date,
			created_on = CASE WHEN w.created_on IS NULL THEN NULL ELSE t.target_ts END,
			last_updated_on = CASE WHEN w.last_updated_on IS NULL THEN NULL ELSE t.target_ts END,
			submitted_on = CASE WHEN w.submitted_on IS NULL THEN NULL ELSE t.target_ts END,
			facility_reviewed_on = CASE WHEN w.facility_reviewed_on IS NULL THEN NULL ELSE t.target_ts END,
			national_submitted_on = CASE WHEN w.national_submitted_on IS NULL THEN NULL ELSE t.target_ts END,
			national_reviewed_on = CASE WHEN w.national_reviewed_on IS NULL THEN NULL ELSE t.target_ts END
		FROM targets t
		WHERE w.id = t.id
	`

	result, err := tx.Exec(updateSQL, *daysAgo)
	if err != nil {
		return err
	}
	rowsUpdated, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return err
	}

	fmt.Printf("rows_updated=%d\n", rowsUpdated)
	return runInspectWeeklyReport(db)
}

func runEnforceWeeklyReportIntegrity(db *sql.DB, args []string) error {
	fs := flag.NewFlagSet("enforce-weeklyreport-integrity", flag.ContinueOnError)
	fs.SetOutput(os.Stdout)
	fix := fs.Bool("fix", false, "apply fixes (dedupe overlaps, normalize weekly windows, and enforce constraints)")
	if err := fs.Parse(args); err != nil {
		return err
	}

	if *fix {
		tx, err := db.Begin()
		if err != nil {
			return err
		}
		defer tx.Rollback()

		if _, err := tx.Exec(`CREATE EXTENSION IF NOT EXISTS btree_gist`); err != nil {
			return err
		}

		if _, err := tx.Exec(`ALTER TABLE clinician_app.weeklyreport DROP CONSTRAINT IF EXISTS ck_weeklyreport_start_stop_7_days`); err != nil {
			return err
		}

		if _, err := tx.Exec(`ALTER TABLE clinician_app.weeklyreport DROP CONSTRAINT IF EXISTS ck_weeklyreport_start_stop_6_days`); err != nil {
			return err
		}

		if _, err := tx.Exec(`DROP INDEX IF EXISTS clinician_app.ux_weeklyreport_employee_hospital_start`); err != nil {
			return err
		}

		if _, err := tx.Exec(`
			WITH ranked AS (
				SELECT
					id,
					employee,
					hospital,
					ROW_NUMBER() OVER (
						PARTITION BY employee, hospital
						ORDER BY COALESCE(start, CURRENT_DATE)::date DESC, id DESC
					) AS rn
				FROM clinician_app.weeklyreport
				WHERE employee IS NOT NULL
					AND hospital IS NOT NULL
			), anchors AS (
				SELECT
					employee,
					hospital,
					MAX(COALESCE(start, CURRENT_DATE)::date) AS anchor_start
				FROM clinician_app.weeklyreport
				WHERE employee IS NOT NULL
					AND hospital IS NOT NULL
				GROUP BY employee, hospital
			), targets AS (
				SELECT
					r.id,
					(a.anchor_start - ((r.rn - 1)::int * INTERVAL '7 days'))::date AS target_start
				FROM ranked r
				JOIN anchors a
					ON a.employee = r.employee
					AND a.hospital = r.hospital
			)
			UPDATE clinician_app.weeklyreport w
			SET
				start = t.target_start,
				stop = t.target_start + INTERVAL '6 days'
			FROM targets t
			WHERE w.id = t.id
		`); err != nil {
			return err
		}

		if _, err := tx.Exec(`
			UPDATE clinician_app.weeklyreport
			SET stop = start + INTERVAL '6 days'
			WHERE start IS NOT NULL
				AND (stop IS NULL OR stop <> start + INTERVAL '6 days')
		`); err != nil {
			return err
		}

		if _, err := tx.Exec(`
			WITH ranked AS (
				SELECT
					id,
					ROW_NUMBER() OVER (
						PARTITION BY employee, hospital, start
						ORDER BY
							CASE COALESCE(report_status, '')
								WHEN 'Approved' THEN 4
								WHEN 'Submitted' THEN 3
								WHEN 'Declined' THEN 2
								WHEN 'Rejected' THEN 2
								ELSE 1
							END DESC,
							CASE COALESCE(submit_status, '') WHEN 'Submitted' THEN 1 ELSE 0 END DESC,
							COALESCE(last_updated_on, created_on, TIMESTAMP 'epoch') DESC,
							id DESC
					) AS rn
				FROM clinician_app.weeklyreport
				WHERE employee IS NOT NULL
					AND hospital IS NOT NULL
					AND start IS NOT NULL
			)
			DELETE FROM clinician_app.weeklyreport w
			USING ranked r
			WHERE w.id = r.id
				AND r.rn > 1
		`); err != nil {
			return err
		}

		if _, err := tx.Exec(`DROP INDEX IF EXISTS clinician_app.ux_weeklyreport_employee_start_stop`); err != nil {
			return err
		}

		if _, err := tx.Exec(`
			CREATE UNIQUE INDEX IF NOT EXISTS ux_weeklyreport_employee_hospital_start
			ON clinician_app.weeklyreport(employee, hospital, start)
			WHERE employee IS NOT NULL AND hospital IS NOT NULL AND start IS NOT NULL
		`); err != nil {
			return err
		}

		if _, err := tx.Exec(`
			ALTER TABLE clinician_app.weeklyreport DROP CONSTRAINT IF EXISTS ck_weeklyreport_start_stop_7_days
		`); err != nil {
			return err
		}

		if _, err := tx.Exec(`ALTER TABLE clinician_app.weeklyreport DROP CONSTRAINT IF EXISTS ck_weeklyreport_start_stop_6_days`); err != nil {
			return err
		}

		if _, err := tx.Exec(`
			ALTER TABLE clinician_app.weeklyreport
			ADD CONSTRAINT ck_weeklyreport_start_stop_6_days
			CHECK (start IS NULL OR stop IS NULL OR stop = start + INTERVAL '6 days')
		`); err != nil {
			return err
		}

		if _, err := tx.Exec(`ALTER TABLE clinician_app.weeklyreport DROP CONSTRAINT IF EXISTS ex_weeklyreport_employee_hospital_no_overlap`); err != nil {
			return err
		}

		if _, err := tx.Exec(`
			ALTER TABLE clinician_app.weeklyreport
			ADD CONSTRAINT ex_weeklyreport_employee_hospital_no_overlap
			EXCLUDE USING gist (
				employee WITH =,
				hospital WITH =,
				daterange(start, stop, '[]') WITH &&
			)
			WHERE (employee IS NOT NULL AND hospital IS NOT NULL AND start IS NOT NULL AND stop IS NOT NULL)
		`); err != nil {
			return err
		}

		if err := tx.Commit(); err != nil {
			return err
		}
	}

	var duplicateGroups int64
	var duplicateRows int64
	if err := db.QueryRow(`
		WITH dupes AS (
			SELECT employee, hospital, start, COUNT(*) AS cnt
			FROM clinician_app.weeklyreport
			WHERE employee IS NOT NULL AND hospital IS NOT NULL AND start IS NOT NULL
			GROUP BY employee, hospital, start
			HAVING COUNT(*) > 1
		)
		SELECT COUNT(*)::bigint, COALESCE(SUM(cnt - 1), 0)::bigint FROM dupes
	`).Scan(&duplicateGroups, &duplicateRows); err != nil {
		return err
	}

	var invalidSpan int64
	if err := db.QueryRow(`
		SELECT COUNT(*)::bigint
		FROM clinician_app.weeklyreport
		WHERE start IS NOT NULL
			AND stop IS NOT NULL
			AND stop <> start + INTERVAL '6 days'
	`).Scan(&invalidSpan); err != nil {
		return err
	}

	var overlapConflicts int64
	if err := db.QueryRow(`
		SELECT COUNT(*)::bigint
		FROM clinician_app.weeklyreport a
		JOIN clinician_app.weeklyreport b
			ON a.id < b.id
			AND a.employee = b.employee
			AND a.hospital = b.hospital
			AND a.start IS NOT NULL AND a.stop IS NOT NULL
			AND b.start IS NOT NULL AND b.stop IS NOT NULL
			AND daterange(a.start, a.stop, '[]') && daterange(b.start, b.stop, '[]')
	`).Scan(&overlapConflicts); err != nil {
		return err
	}

	var hasIndex bool
	if err := db.QueryRow(`
		SELECT EXISTS (
			SELECT 1
			FROM pg_indexes
			WHERE schemaname = 'clinician_app'
				AND tablename = 'weeklyreport'
				AND indexname = 'ux_weeklyreport_employee_hospital_start'
		)
	`).Scan(&hasIndex); err != nil {
		return err
	}

	var hasConstraint bool
	if err := db.QueryRow(`
		SELECT EXISTS (
			SELECT 1
			FROM pg_constraint
			WHERE conname = 'ck_weeklyreport_start_stop_6_days'
				AND conrelid = 'clinician_app.weeklyreport'::regclass
		)
	`).Scan(&hasConstraint); err != nil {
		return err
	}

	var hasOverlapConstraint bool
	if err := db.QueryRow(`
		SELECT EXISTS (
			SELECT 1
			FROM pg_constraint
			WHERE conname = 'ex_weeklyreport_employee_hospital_no_overlap'
				AND conrelid = 'clinician_app.weeklyreport'::regclass
		)
	`).Scan(&hasOverlapConstraint); err != nil {
		return err
	}

	fmt.Printf("duplicate_groups=%d\n", duplicateGroups)
	fmt.Printf("duplicate_rows=%d\n", duplicateRows)
	fmt.Printf("invalid_start_stop_span_rows=%d\n", invalidSpan)
	fmt.Printf("overlap_conflict_pairs=%d\n", overlapConflicts)
	fmt.Printf("has_unique_index_employee_hospital_start=%t\n", hasIndex)
	fmt.Printf("has_start_stop_6_days_constraint=%t\n", hasConstraint)
	fmt.Printf("has_no_overlap_exclusion_constraint=%t\n", hasOverlapConstraint)

	if duplicateGroups > 0 || invalidSpan > 0 || overlapConflicts > 0 || !hasIndex || !hasConstraint || !hasOverlapConstraint {
		return errors.New("weeklyreport integrity check failed; run with --fix to enforce")
	}

	return nil
}

func runQuery(db *sql.DB, args []string) error {
	fs := flag.NewFlagSet("query", flag.ContinueOnError)
	fs.SetOutput(os.Stdout)
	sqlText := fs.String("sql", "", "SQL statement to run")
	if err := fs.Parse(args); err != nil {
		return err
	}

	statement := strings.TrimSpace(*sqlText)
	if statement == "" {
		return errors.New("--sql is required")
	}

	lower := strings.ToLower(statement)
	isSelect := strings.HasPrefix(lower, "select") || strings.HasPrefix(lower, "with")
	if !isSelect {
		result, err := db.Exec(statement)
		if err != nil {
			return err
		}
		affected, err := result.RowsAffected()
		if err != nil {
			return err
		}
		fmt.Printf("rows_affected=%d\n", affected)
		return nil
	}

	rows, err := db.Query(statement)
	if err != nil {
		return err
	}
	defer rows.Close()

	cols, err := rows.Columns()
	if err != nil {
		return err
	}
	fmt.Printf("columns=%s\n", strings.Join(cols, ","))

	count := 0
	for rows.Next() {
		values := make([]interface{}, len(cols))
		ptrs := make([]interface{}, len(cols))
		for i := range values {
			ptrs[i] = &values[i]
		}
		if err := rows.Scan(ptrs...); err != nil {
			return err
		}

		parts := make([]string, 0, len(cols))
		for i, col := range cols {
			parts = append(parts, fmt.Sprintf("%s=%s", col, stringifySQLValue(values[i])))
		}
		fmt.Println(strings.Join(parts, " | "))
		count++
	}
	if err := rows.Err(); err != nil {
		return err
	}

	fmt.Printf("row_count=%d\n", count)
	return nil
}

func stringifySQLValue(v interface{}) string {
	switch t := v.(type) {
	case nil:
		return "NULL"
	case []byte:
		return string(t)
	case time.Time:
		return t.Format(time.RFC3339)
	default:
		return fmt.Sprintf("%v", t)
	}
}

func formatNullDate(value sql.NullTime) string {
	if !value.Valid {
		return "NULL"
	}
	return value.Time.Format("2006-01-02")
}

func printUsage() {
	fmt.Println("Usage:")
	fmt.Println("  go run ./cmd/dbops inspect-weeklyreport")
	fmt.Println("  go run ./cmd/dbops backdate-weeklyreport [--days-ago 1] [--dry-run]")
	fmt.Println("  go run ./cmd/dbops verify-dashboard-kpis")
	fmt.Println("  go run ./cmd/dbops verify-staff-dashboard-all")
	fmt.Println("  go run ./cmd/dbops verify-staff-list-tables-all")
	fmt.Println("  go run ./cmd/dbops enforce-weeklyreport-integrity [--fix]")
	fmt.Println("  go run ./cmd/dbops query --sql \"SELECT COUNT(*) FROM clinician_app.weeklyreport\"")
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

func findProjectRoot() (string, error) {
	if root := os.Getenv("CLINICIAN_APP_ROOT"); root != "" {
		return filepath.Abs(root)
	}

	wd, err := os.Getwd()
	if err != nil {
		return "", err
	}

	candidates := []string{}
	current := wd
	for {
		candidates = append(candidates, current)
		parent := filepath.Dir(current)
		if parent == current {
			break
		}
		current = parent
	}

	for _, dir := range candidates {
		if fileExists(filepath.Join(dir, "config.json")) && dirExists(filepath.Join(dir, "clinician")) {
			return dir, nil
		}
	}

	for _, dir := range candidates {
		if fileExists(filepath.Join(dir, "go.work")) && dirExists(filepath.Join(dir, "clinician")) {
			return dir, nil
		}
	}

	for _, dir := range candidates {
		if fileExists(filepath.Join(dir, ".env")) && dirExists(filepath.Join(dir, "clinician")) {
			return dir, nil
		}
	}

	sort.Strings(candidates)
	return "", errors.New("could not locate project root (expected config.json/go.work + clinician directory)")
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
