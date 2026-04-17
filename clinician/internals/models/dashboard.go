package models

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

type FacilityPerformanceRow struct {
	FacilityID        int
	FacilityName      string
	Clinicians        int
	EnteredThisWeek   int
	SubmittedThisWeek int
	PendingThisWeek   int
	PendingApproval   int
	ReportingRate     int
}

type DashboardFilterOption struct {
	ID   int
	Name string
}

type DepartmentProgressRow struct {
	DepartmentID      int
	DepartmentName    string
	Clinicians        int
	EnteredThisWeek   int
	SubmittedThisWeek int
	PendingThisWeek   int
	ReportingRate     int
}

type DepartmentAnalysisRow struct {
	DepartmentID      int
	DepartmentName    string
	Clinicians        int
	ReportsEntered    int
	ReportsSubmitted  int
	ReportingRate     int
	PendingReports    int
	AttendanceRecords int
	WardRounds        int
	PatientsReviewed  int
	Procedures        int
	AvgPatients       int
	AvgProcedures     int
}

type ClinicianAnalysisRow struct {
	EmployeeID        int
	EmployeeName      string
	Title             string
	DepartmentName    string
	ReportsEntered    int
	ReportsSubmitted  int
	AttendanceRecords int
	WardRounds        int
	PatientsReviewed  int
	Procedures        int
	SubmissionStatus  string
	LeaveStatus       string
}

type ClinicianTrendPoint struct {
	WeekLabel        string `json:"weekLabel"`
	StartDate        string `json:"startDate"`
	DaysWorked       int    `json:"daysWorked"`
	PatientsReviewed int    `json:"patientsReviewed"`
	Procedures       int    `json:"procedures"`
}

type ClinicianWeekOption struct {
	Year      int    `json:"year"`
	Week      int    `json:"week"`
	StartDate string `json:"startDate"`
	EndDate   string `json:"endDate"`
	Label     string `json:"label"`
}

type NationalDashboardSnapshot struct {
	TotalFacilities        int
	TotalClinicians        int
	ReportsEnteredThisWeek int
	SubmittedThisWeek      int
	PendingApproval        int
	FacilityPerformance    []FacilityPerformanceRow
}

type FacilityManagerDashboardSnapshot struct {
	FacilityName           string
	TotalClinicians        int
	ReportsEnteredThisWeek int
	SubmittedThisWeek      int
	PendingSubmission      int
	Departments            []DepartmentProgressRow
}

type ClinicianDashboardSnapshot struct {
	EmployeeID            int
	EmployeeName          string
	EmployeeTitle         string
	FacilityName          string
	DepartmentName        string
	SelectedYear          int
	SelectedWeek          int
	SelectedWeekStart     string
	SelectedWeekLabel     string
	WeeksEntered          int
	WeeksSubmitted        int
	DaysWorked            int
	WardRounds            int
	PatientsReviewed      int
	Procedures            int
	TotalReports          int
	SubmittedReports      int
	ApprovedReports       int
	DeclinedReports       int
	AttendanceRate        int
	OutputTarget          int
	AttendanceTarget      int
	WardRoundsTarget      int
	SubmissionProgress    int
	AvailableYears        []int
	AvailableWeeks        []ClinicianWeekOption
	RecentWeeks           []ClinicianTrendPoint
	SelectedWeekSubmitted bool
}

func GetNationalDashboardSnapshot(ctx context.Context, db *sql.DB) (NationalDashboardSnapshot, error) {
	var snapshot NationalDashboardSnapshot

	countQuery := `
		SELECT
			(SELECT COUNT(*) FROM clinician_app.facilities) AS total_facilities,
			(SELECT COUNT(*) FROM clinician_app.employees) AS total_clinicians,
			(SELECT COUNT(*) FROM clinician_app.weeklyreport WHERE start = date_trunc('week', CURRENT_DATE)::date) AS entered_this_week,
			(SELECT COUNT(*) FROM clinician_app.weeklyreport WHERE start = date_trunc('week', CURRENT_DATE)::date AND submit_status = 'Submitted') AS submitted_this_week,
			(SELECT COUNT(*) FROM clinician_app.weeklyreport WHERE start = date_trunc('week', CURRENT_DATE)::date AND submit_status = 'Submitted' AND COALESCE(report_status, '') <> 'Approved') AS pending_approval
	`
	if err := db.QueryRowContext(ctx, countQuery).Scan(
		&snapshot.TotalFacilities,
		&snapshot.TotalClinicians,
		&snapshot.ReportsEnteredThisWeek,
		&snapshot.SubmittedThisWeek,
		&snapshot.PendingApproval,
	); err != nil {
		return snapshot, err
	}

	performanceQuery := `
		SELECT
			f.id,
			f.f_name,
			COUNT(DISTINCT e.id) AS clinicians,
			COUNT(DISTINCT CASE WHEN w.start = date_trunc('week', CURRENT_DATE)::date THEN w.employee END) AS entered_this_week,
			COUNT(DISTINCT CASE WHEN w.start = date_trunc('week', CURRENT_DATE)::date AND w.submit_status = 'Submitted' THEN w.employee END) AS submitted_this_week,
			COUNT(DISTINCT CASE
				WHEN w.start = date_trunc('week', CURRENT_DATE)::date
				AND w.submit_status = 'Submitted'
				AND COALESCE(w.report_status, '') NOT IN ('Approved', 'Rejected', 'Declined')
				THEN w.employee
			END) AS pending_approval
		FROM clinician_app.facilities f
		LEFT JOIN clinician_app.employees e ON e.facility = f.id
		LEFT JOIN clinician_app.weeklyreport w ON w.hospital = f.id
		GROUP BY f.id, f.f_name
		ORDER BY f.f_name
	`

	rows, err := db.QueryContext(ctx, performanceQuery)
	if err != nil {
		return snapshot, err
	}
	defer rows.Close()

	for rows.Next() {
		var row FacilityPerformanceRow
		if err := rows.Scan(
			&row.FacilityID,
			&row.FacilityName,
			&row.Clinicians,
			&row.EnteredThisWeek,
			&row.SubmittedThisWeek,
			&row.PendingApproval,
		); err != nil {
			return snapshot, err
		}

		row.PendingThisWeek = row.Clinicians - row.SubmittedThisWeek
		if row.PendingThisWeek < 0 {
			row.PendingThisWeek = 0
		}
		row.ReportingRate = percentageInt(row.SubmittedThisWeek, row.Clinicians)

		snapshot.FacilityPerformance = append(snapshot.FacilityPerformance, row)
	}

	return snapshot, rows.Err()
}

func GetFacilityManagerDashboardSnapshot(ctx context.Context, db *sql.DB, facilityID int) (FacilityManagerDashboardSnapshot, error) {
	var snapshot FacilityManagerDashboardSnapshot

	countQuery := `
		SELECT
			f.f_name,
			COUNT(DISTINCT e.id) AS total_clinicians,
			COUNT(DISTINCT CASE WHEN w.start = date_trunc('week', CURRENT_DATE)::date THEN w.employee END) AS entered_this_week,
			COUNT(DISTINCT CASE WHEN w.start = date_trunc('week', CURRENT_DATE)::date AND w.submit_status = 'Submitted' THEN w.employee END) AS submitted_this_week
		FROM clinician_app.facilities f
		LEFT JOIN clinician_app.employees e ON e.facility = f.id
		LEFT JOIN clinician_app.weeklyreport w ON w.hospital = f.id
		WHERE f.id = $1
		GROUP BY f.f_name
	`
	if err := db.QueryRowContext(ctx, countQuery, facilityID).Scan(
		&snapshot.FacilityName,
		&snapshot.TotalClinicians,
		&snapshot.ReportsEnteredThisWeek,
		&snapshot.SubmittedThisWeek,
	); err != nil {
		return snapshot, err
	}

	snapshot.PendingSubmission = snapshot.TotalClinicians - snapshot.SubmittedThisWeek
	if snapshot.PendingSubmission < 0 {
		snapshot.PendingSubmission = 0
	}

	departmentQuery := `
		SELECT
			d.id,
			d.d_name,
			COUNT(DISTINCT e.id) AS clinicians,
			COUNT(DISTINCT CASE WHEN w.start = date_trunc('week', CURRENT_DATE)::date THEN w.employee END) AS entered_this_week,
			COUNT(DISTINCT CASE WHEN w.start = date_trunc('week', CURRENT_DATE)::date AND w.submit_status = 'Submitted' THEN w.employee END) AS submitted_this_week
		FROM clinician_app.departments d
		LEFT JOIN clinician_app.employees e ON e.department = d.id AND e.facility = $1
		LEFT JOIN clinician_app.weeklyreport w ON w.department = d.id AND w.hospital = $1
		GROUP BY d.id, d.d_name
		ORDER BY d.d_name
	`

	rows, err := db.QueryContext(ctx, departmentQuery, facilityID)
	if err != nil {
		return snapshot, err
	}
	defer rows.Close()

	for rows.Next() {
		var row DepartmentProgressRow
		if err := rows.Scan(
			&row.DepartmentID,
			&row.DepartmentName,
			&row.Clinicians,
			&row.EnteredThisWeek,
			&row.SubmittedThisWeek,
		); err != nil {
			return snapshot, err
		}

		row.PendingThisWeek = row.Clinicians - row.SubmittedThisWeek
		if row.PendingThisWeek < 0 {
			row.PendingThisWeek = 0
		}
		row.ReportingRate = percentageInt(row.SubmittedThisWeek, row.Clinicians)

		snapshot.Departments = append(snapshot.Departments, row)
	}

	return snapshot, rows.Err()
}

func GetClinicianDashboardSnapshot(ctx context.Context, db *sql.DB, employeeID int, selectedYear int, selectedWeek int) (ClinicianDashboardSnapshot, error) {
	var snapshot ClinicianDashboardSnapshot

	employeeQuery := `
		SELECT
			e.id,
			CONCAT(e.fname, ' ', e.lname) AS employee_name,
			COALESCE(st.title, '') AS employee_title,
			COALESCE(f.f_name, '') AS facility_name,
			COALESCE(d.d_name, '') AS department_name
		FROM clinician_app.employees e
		LEFT JOIN clinician_app.specialist_titles st ON st.id = e.title
		LEFT JOIN clinician_app.facilities f ON f.id = e.facility
		LEFT JOIN clinician_app.departments d ON d.id = e.department
		WHERE e.id = $1
	`
	if err := db.QueryRowContext(ctx, employeeQuery, employeeID).Scan(
		&snapshot.EmployeeID,
		&snapshot.EmployeeName,
		&snapshot.EmployeeTitle,
		&snapshot.FacilityName,
		&snapshot.DepartmentName,
	); err != nil {
		return snapshot, err
	}

	periodRows, err := db.QueryContext(ctx, `
		SELECT DISTINCT start
		FROM clinician_app.weeklyreport
		WHERE employee = $1
		ORDER BY start DESC
	`, employeeID)
	if err != nil {
		return snapshot, err
	}
	defer periodRows.Close()

	yearSeen := map[int]bool{}
	weekOptionsByYear := map[int][]ClinicianWeekOption{}
	var allWeekOptions []ClinicianWeekOption
	for periodRows.Next() {
		var start time.Time
		if err := periodRows.Scan(&start); err != nil {
			return snapshot, err
		}

		year, week := start.ISOWeek()
		option := ClinicianWeekOption{
			Year:      year,
			Week:      week,
			StartDate: start.Format("2006-01-02"),
			EndDate:   start.AddDate(0, 0, 6).Format("2006-01-02"),
			Label:     fmt.Sprintf("Week %02d (%s - %s)", week, start.Format("02 Jan"), start.AddDate(0, 0, 6).Format("02 Jan")),
		}

		if !yearSeen[year] {
			snapshot.AvailableYears = append(snapshot.AvailableYears, year)
			yearSeen[year] = true
		}

		weekOptionsByYear[year] = append(weekOptionsByYear[year], option)
		allWeekOptions = append(allWeekOptions, option)
	}
	if err := periodRows.Err(); err != nil {
		return snapshot, err
	}

	if len(allWeekOptions) == 0 {
		currentWeekStart := startOfWeek(time.Now())
		year, week := currentWeekStart.ISOWeek()
		option := ClinicianWeekOption{
			Year:      year,
			Week:      week,
			StartDate: currentWeekStart.Format("2006-01-02"),
			EndDate:   currentWeekStart.AddDate(0, 0, 6).Format("2006-01-02"),
			Label:     fmt.Sprintf("Week %02d (%s - %s)", week, currentWeekStart.Format("02 Jan"), currentWeekStart.AddDate(0, 0, 6).Format("02 Jan")),
		}

		snapshot.AvailableYears = []int{year}
		weekOptionsByYear[year] = []ClinicianWeekOption{option}
		allWeekOptions = []ClinicianWeekOption{option}
	}

	snapshot.AvailableYears = append([]int{0}, snapshot.AvailableYears...)

	if selectedYear != 0 && !containsInt(snapshot.AvailableYears, selectedYear) {
		selectedYear = 0
	}
	snapshot.SelectedYear = selectedYear
	snapshot.AvailableWeeks = allWeekOptions

	var selectedOption ClinicianWeekOption
	var hasSelectedOption bool
	if selectedYear != 0 && selectedWeek != 0 {
		selectedOption, hasSelectedOption = resolveWeekSelection(weekOptionsByYear[selectedYear], selectedWeek)
		if !hasSelectedOption {
			return snapshot, fmt.Errorf("no reporting weeks available for employee %d", employeeID)
		}
	}

	snapshot.SelectedWeek = selectedWeek
	switch {
	case selectedYear == 0:
		snapshot.SelectedWeek = 0
		snapshot.SelectedWeekLabel = "All records"
	case selectedWeek == 0:
		snapshot.SelectedWeekLabel = fmt.Sprintf("All weeks in %d", selectedYear)
	default:
		snapshot.SelectedWeek = selectedOption.Week
		snapshot.SelectedWeekStart = selectedOption.StartDate
		snapshot.SelectedWeekLabel = fmt.Sprintf(
			"Week %02d, %d (%s - %s)",
			selectedOption.Week,
			selectedOption.Year,
			mustParseDate(selectedOption.StartDate).Format("02 Jan 2006"),
			mustParseDate(selectedOption.EndDate).Format("02 Jan 2006"),
		)
	}

	var summaryQuery string
	var summaryArgs []interface{}
	switch {
	case selectedYear == 0:
		summaryQuery = `
			SELECT
				COUNT(DISTINCT start) AS weeks_entered,
				COUNT(DISTINCT CASE WHEN submit_status = 'Submitted' THEN start END) AS weeks_submitted,
				COALESCE(SUM(qn_01), 0) AS days_worked,
				COALESCE(SUM(qn_02), 0) AS ward_rounds,
				COALESCE(SUM(qn_03), 0) AS patients_reviewed,
				COALESCE(SUM(qn_05), 0) + COALESCE(SUM(qn_06), 0) AS procedures
			FROM clinician_app.weeklyreport
			WHERE employee = $1
		`
		summaryArgs = []interface{}{employeeID}
	case selectedWeek == 0:
		summaryQuery = `
			SELECT
				COUNT(DISTINCT start) AS weeks_entered,
				COUNT(DISTINCT CASE WHEN submit_status = 'Submitted' THEN start END) AS weeks_submitted,
				COALESCE(SUM(qn_01), 0) AS days_worked,
				COALESCE(SUM(qn_02), 0) AS ward_rounds,
				COALESCE(SUM(qn_03), 0) AS patients_reviewed,
				COALESCE(SUM(qn_05), 0) + COALESCE(SUM(qn_06), 0) AS procedures
			FROM clinician_app.weeklyreport
			WHERE employee = $1 AND EXTRACT(ISOYEAR FROM start) = $2
		`
		summaryArgs = []interface{}{employeeID, selectedYear}
	default:
		summaryQuery = `
			SELECT
				COUNT(DISTINCT start) AS weeks_entered,
				COUNT(DISTINCT CASE WHEN submit_status = 'Submitted' THEN start END) AS weeks_submitted,
				COALESCE(SUM(qn_01), 0) AS days_worked,
				COALESCE(SUM(qn_02), 0) AS ward_rounds,
				COALESCE(SUM(qn_03), 0) AS patients_reviewed,
				COALESCE(SUM(qn_05), 0) + COALESCE(SUM(qn_06), 0) AS procedures
			FROM clinician_app.weeklyreport
			WHERE employee = $1 AND start = $2::date
		`
		summaryArgs = []interface{}{employeeID, selectedOption.StartDate}
	}

	if err := db.QueryRowContext(ctx, summaryQuery, summaryArgs...).Scan(
		&snapshot.WeeksEntered,
		&snapshot.WeeksSubmitted,
		&snapshot.DaysWorked,
		&snapshot.WardRounds,
		&snapshot.PatientsReviewed,
		&snapshot.Procedures,
	); err != nil {
		return snapshot, err
	}

	snapshot.SelectedWeekSubmitted = snapshot.WeeksEntered > 0 && snapshot.WeeksEntered == snapshot.WeeksSubmitted

	if snapshot.WeeksEntered > 0 {
		snapshot.AttendanceRate = int(float64(snapshot.DaysWorked) / 5.0 * 100)
		snapshot.SubmissionProgress = int(float64(snapshot.WeeksSubmitted) / float64(snapshot.WeeksEntered) * 100)
	}

	reportSummary, err := GetClinicianReportHistorySummary(ctx, db, int64(employeeID), snapshot.SelectedYear, snapshot.SelectedWeek)
	if err != nil {
		return snapshot, err
	}
	snapshot.TotalReports = reportSummary.TotalReports
	snapshot.SubmittedReports = reportSummary.SubmittedReports
	snapshot.ApprovedReports = reportSummary.ApprovedReports
	snapshot.DeclinedReports = reportSummary.DeclinedReports

	if snapshot.AttendanceRate > 100 {
		snapshot.AttendanceRate = 100
	}
	if snapshot.SubmissionProgress > 100 {
		snapshot.SubmissionProgress = 100
	}

	snapshot.OutputTarget = getDashboardTarget(ctx, db, 1, 40)
	snapshot.AttendanceTarget = getDashboardTarget(ctx, db, 2, 95)
	snapshot.WardRoundsTarget = getDashboardTarget(ctx, db, 3, 25)

	var trendQuery string
	var trendArgs []interface{}
	if selectedYear == 0 {
		trendQuery = `
			SELECT
				start,
				COALESCE(qn_01, 0) AS days_worked,
				COALESCE(qn_03, 0) AS patients_reviewed,
				COALESCE(qn_05, 0) + COALESCE(qn_06, 0) AS procedures
			FROM clinician_app.weeklyreport
			WHERE employee = $1
			ORDER BY start
		`
		trendArgs = []interface{}{employeeID}
	} else {
		trendQuery = `
			SELECT
				start,
				COALESCE(qn_01, 0) AS days_worked,
				COALESCE(qn_03, 0) AS patients_reviewed,
				COALESCE(qn_05, 0) + COALESCE(qn_06, 0) AS procedures
			FROM clinician_app.weeklyreport
			WHERE employee = $1 AND EXTRACT(ISOYEAR FROM start) = $2
			ORDER BY start
		`
		trendArgs = []interface{}{employeeID, snapshot.SelectedYear}
	}

	rows, err := db.QueryContext(ctx, trendQuery, trendArgs...)
	if err != nil {
		return snapshot, err
	}
	defer rows.Close()

	for rows.Next() {
		var start time.Time
		var point ClinicianTrendPoint
		if err := rows.Scan(&start, &point.DaysWorked, &point.PatientsReviewed, &point.Procedures); err != nil {
			return snapshot, err
		}

		point.StartDate = start.Format("2006-01-02")
		year, week := start.ISOWeek()
		if selectedYear == 0 {
			point.WeekLabel = fmt.Sprintf("Wk %02d %d", week, year)
		} else {
			point.WeekLabel = fmt.Sprintf("Wk %02d", week)
		}
		snapshot.RecentWeeks = append(snapshot.RecentWeeks, point)
	}

	return snapshot, rows.Err()
}

func containsInt(values []int, target int) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

func resolveWeekSelection(options []ClinicianWeekOption, selectedWeek int) (ClinicianWeekOption, bool) {
	if len(options) == 0 {
		return ClinicianWeekOption{}, false
	}

	for _, option := range options {
		if option.Week == selectedWeek {
			return option, true
		}
	}

	return options[0], true
}

func startOfWeek(value time.Time) time.Time {
	weekday := int(value.Weekday())
	if weekday == 0 {
		weekday = 7
	}

	year, month, day := value.Date()
	return time.Date(year, month, day-(weekday-1), 0, 0, 0, 0, value.Location())
}

func mustParseDate(value string) time.Time {
	parsed, err := time.Parse("2006-01-02", value)
	if err != nil {
		return time.Time{}
	}
	return parsed
}

func getDashboardTarget(ctx context.Context, db *sql.DB, indicatorID int, fallback int) int {
	query := `SELECT target FROM clinician_app.targets WHERE indicator = $1 ORDER BY id LIMIT 1`
	var target sql.NullInt64
	if err := db.QueryRowContext(ctx, query, indicatorID).Scan(&target); err != nil {
		return fallback
	}
	if !target.Valid {
		return fallback
	}
	return int(target.Int64)
}

func GetFacilityDepartmentAnalysis(ctx context.Context, db *sql.DB, facilityID int) ([]DepartmentAnalysisRow, error) {
	const sqlstr = `
		WITH week_window AS (
			SELECT
				date_trunc('week', CURRENT_DATE)::date AS week_start,
				(date_trunc('week', CURRENT_DATE)::date + INTERVAL '6 days')::date AS week_end
		),
		department_base AS (
			SELECT
				d.id AS department_id,
				d.d_name AS department_name,
				COUNT(DISTINCT e.id) AS clinicians
			FROM clinician_app.departments d
			LEFT JOIN clinician_app.employees e
				ON e.department = d.id
				AND e.facility = $1
			GROUP BY d.id, d.d_name
		),
		report_stats AS (
			SELECT
				w.department AS department_id,
				COUNT(DISTINCT w.employee) AS reports_entered,
				COUNT(DISTINCT CASE WHEN COALESCE(w.submit_status, '') = 'Submitted' THEN w.employee END) AS reports_submitted,
				COALESCE(SUM(COALESCE(w.qn_03, 0)), 0) AS patients_reviewed,
				COALESCE(SUM(COALESCE(w.qn_05, 0)), 0) + COALESCE(SUM(COALESCE(w.qn_06, 0)), 0) AS procedures
			FROM clinician_app.weeklyreport w
			JOIN week_window ww ON w.start = ww.week_start
			WHERE w.hospital = $1
			GROUP BY w.department
		),
		attendance_stats AS (
			SELECT
				a.department_id,
				COUNT(*) AS attendance_records
			FROM clinician_app.attendance_records a
			JOIN week_window ww ON a.attendance_date BETWEEN ww.week_start AND ww.week_end
			WHERE a.facility_id = $1
			GROUP BY a.department_id
		),
		round_stats AS (
			SELECT
				w.department_id,
				COUNT(*) AS ward_rounds
			FROM clinician_app.ward_rounds w
			JOIN week_window ww ON w.round_date BETWEEN ww.week_start AND ww.week_end
			WHERE w.facility_id = $1
			GROUP BY w.department_id
		)
		SELECT
			db.department_id,
			db.department_name,
			db.clinicians,
			COALESCE(rs.reports_entered, 0) AS reports_entered,
			COALESCE(rs.reports_submitted, 0) AS reports_submitted,
			COALESCE(att.attendance_records, 0) AS attendance_records,
			COALESCE(rnd.ward_rounds, 0) AS ward_rounds,
			COALESCE(rs.patients_reviewed, 0) AS patients_reviewed,
			COALESCE(rs.procedures, 0) AS procedures
		FROM department_base db
		LEFT JOIN report_stats rs ON rs.department_id = db.department_id
		LEFT JOIN attendance_stats att ON att.department_id = db.department_id
		LEFT JOIN round_stats rnd ON rnd.department_id = db.department_id
		WHERE db.clinicians > 0
		ORDER BY db.department_name
	`

	rows, err := db.QueryContext(ctx, sqlstr, facilityID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := []DepartmentAnalysisRow{}
	for rows.Next() {
		var row DepartmentAnalysisRow
		if err := rows.Scan(
			&row.DepartmentID,
			&row.DepartmentName,
			&row.Clinicians,
			&row.ReportsEntered,
			&row.ReportsSubmitted,
			&row.AttendanceRecords,
			&row.WardRounds,
			&row.PatientsReviewed,
			&row.Procedures,
		); err != nil {
			return nil, err
		}

		row.ReportingRate = percentageInt(row.ReportsSubmitted, row.Clinicians)
		row.PendingReports = row.Clinicians - row.ReportsSubmitted
		if row.PendingReports < 0 {
			row.PendingReports = 0
		}
		row.AvgPatients = averageInt(row.PatientsReviewed, row.ReportsEntered)
		row.AvgProcedures = averageInt(row.Procedures, row.ReportsEntered)
		items = append(items, row)
	}

	return items, rows.Err()
}

func GetFacilityClinicianAnalysis(ctx context.Context, db *sql.DB, facilityID int, departmentID int) ([]ClinicianAnalysisRow, error) {
	const sqlstr = `
		WITH week_window AS (
			SELECT
				date_trunc('week', CURRENT_DATE)::date AS week_start,
				(date_trunc('week', CURRENT_DATE)::date + INTERVAL '6 days')::date AS week_end
		),
		report_stats AS (
			SELECT
				w.employee AS employee_id,
				COUNT(*) AS reports_entered,
				COUNT(*) FILTER (WHERE COALESCE(w.submit_status, '') = 'Submitted') AS reports_submitted,
				MAX(COALESCE(w.submit_status, '')) AS submit_status,
				MAX(COALESCE(w.report_status, '')) AS report_status,
				COALESCE(SUM(COALESCE(w.qn_03, 0)), 0) AS patients_reviewed,
				COALESCE(SUM(COALESCE(w.qn_05, 0)), 0) + COALESCE(SUM(COALESCE(w.qn_06, 0)), 0) AS procedures
			FROM clinician_app.weeklyreport w
			JOIN week_window ww ON w.start = ww.week_start
			WHERE w.hospital = $1
			GROUP BY w.employee
		),
		attendance_stats AS (
			SELECT
				a.specialist_id AS employee_id,
				COUNT(*) AS attendance_records
			FROM clinician_app.attendance_records a
			JOIN week_window ww ON a.attendance_date BETWEEN ww.week_start AND ww.week_end
			WHERE a.facility_id = $1
			GROUP BY a.specialist_id
		),
		round_stats AS (
			SELECT
				w.specialist_id AS employee_id,
				COUNT(*) AS ward_rounds
			FROM clinician_app.ward_rounds w
			JOIN week_window ww ON w.round_date BETWEEN ww.week_start AND ww.week_end
			WHERE w.facility_id = $1
			GROUP BY w.specialist_id
		),
		latest_leave AS (
			SELECT DISTINCT ON (s.employee_id)
				s.employee_id,
				COALESCE(s.leave_status, '') AS leave_status,
				s.start_date,
				s.end_date,
				s.return_date
			FROM clinician_app.staffleave s
			JOIN clinician_app.employees e ON e.id = s.employee_id
			WHERE e.facility = $1
			ORDER BY s.employee_id, s.created_on DESC, s.leave_id DESC
		)
		SELECT
			e.id,
			TRIM(CONCAT(COALESCE(e.fname, ''), ' ', COALESCE(e.lname, ''))) AS employee_name,
			COALESCE(st.title, '') AS title_name,
			COALESCE(d.d_name, '') AS department_name,
			COALESCE(rs.reports_entered, 0) AS reports_entered,
			COALESCE(rs.reports_submitted, 0) AS reports_submitted,
			COALESCE(att.attendance_records, 0) AS attendance_records,
			COALESCE(rnd.ward_rounds, 0) AS ward_rounds,
			COALESCE(rs.patients_reviewed, 0) AS patients_reviewed,
			COALESCE(rs.procedures, 0) AS procedures,
			CASE
				WHEN COALESCE(rs.report_status, '') = 'Approved' THEN 'Approved'
				WHEN COALESCE(rs.report_status, '') IN ('Rejected', 'Declined') THEN 'Declined'
				WHEN COALESCE(rs.submit_status, '') = 'Submitted' THEN 'Submitted'
				WHEN COALESCE(rs.reports_entered, 0) > 0 THEN 'Draft'
				ELSE 'Not Entered'
			END AS submission_status,
			CASE
				WHEN COALESCE(ll.leave_status, '') IN ('Valid', 'Approved')
					AND CURRENT_DATE BETWEEN COALESCE(ll.start_date, CURRENT_DATE) AND COALESCE(ll.return_date, ll.end_date, CURRENT_DATE)
				THEN 'On Leave'
				ELSE 'Active'
			END AS leave_status
		FROM clinician_app.employees e
		LEFT JOIN clinician_app.departments d ON d.id = e.department
		LEFT JOIN clinician_app.specialist_titles st ON st.id = e.title
		LEFT JOIN report_stats rs ON rs.employee_id = e.id
		LEFT JOIN attendance_stats att ON att.employee_id = e.id
		LEFT JOIN round_stats rnd ON rnd.employee_id = e.id
		LEFT JOIN latest_leave ll ON ll.employee_id = e.id
		WHERE e.facility = $1
			AND ($2 = 0 OR e.department = $2)
		ORDER BY d.d_name, employee_name
	`

	rows, err := db.QueryContext(ctx, sqlstr, facilityID, departmentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := []ClinicianAnalysisRow{}
	for rows.Next() {
		var row ClinicianAnalysisRow
		if err := rows.Scan(
			&row.EmployeeID,
			&row.EmployeeName,
			&row.Title,
			&row.DepartmentName,
			&row.ReportsEntered,
			&row.ReportsSubmitted,
			&row.AttendanceRecords,
			&row.WardRounds,
			&row.PatientsReviewed,
			&row.Procedures,
			&row.SubmissionStatus,
			&row.LeaveStatus,
		); err != nil {
			return nil, err
		}
		items = append(items, row)
	}

	return items, rows.Err()
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

func averageInt(total int, count int) int {
	if count <= 0 {
		return 0
	}

	value := int(float64(total) / float64(count))
	if value < 0 {
		return 0
	}
	return value
}
