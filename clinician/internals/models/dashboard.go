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
}

type DepartmentProgressRow struct {
	DepartmentID      int
	DepartmentName    string
	Clinicians        int
	EnteredThisWeek   int
	SubmittedThisWeek int
	PendingThisWeek   int
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
			COUNT(DISTINCT CASE WHEN w.start = date_trunc('week', CURRENT_DATE)::date AND w.submit_status = 'Submitted' THEN w.employee END) AS submitted_this_week
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
		); err != nil {
			return snapshot, err
		}

		row.PendingThisWeek = row.Clinicians - row.SubmittedThisWeek
		if row.PendingThisWeek < 0 {
			row.PendingThisWeek = 0
		}

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
	var latestOption ClinicianWeekOption
	hasLatestOption := false

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
		if !hasLatestOption {
			latestOption = option
			hasLatestOption = true
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
		latestOption = option
		hasLatestOption = true
	}

	if selectedYear == 0 && hasLatestOption {
		selectedYear = latestOption.Year
	}
	if !containsInt(snapshot.AvailableYears, selectedYear) {
		if hasLatestOption {
			selectedYear = latestOption.Year
		} else {
			selectedYear = snapshot.AvailableYears[0]
		}
	}
	snapshot.SelectedYear = selectedYear
	snapshot.AvailableWeeks = allWeekOptions

	defaultWeek := selectedWeek
	if defaultWeek == 0 && hasLatestOption && selectedYear == latestOption.Year {
		defaultWeek = latestOption.Week
	}

	selectedOption, ok := resolveWeekSelection(weekOptionsByYear[selectedYear], defaultWeek)
	if !ok {
		return snapshot, fmt.Errorf("no reporting weeks available for employee %d", employeeID)
	}

	snapshot.SelectedWeek = selectedOption.Week
	snapshot.SelectedWeekStart = selectedOption.StartDate
	snapshot.SelectedWeekLabel = fmt.Sprintf(
		"Week %02d, %d (%s - %s)",
		selectedOption.Week,
		selectedOption.Year,
		mustParseDate(selectedOption.StartDate).Format("02 Jan 2006"),
		mustParseDate(selectedOption.EndDate).Format("02 Jan 2006"),
	)

	summaryQuery := `
		SELECT
			COUNT(id) AS weeks_entered,
			COUNT(CASE WHEN submit_status = 'Submitted' THEN 1 END) AS weeks_submitted,
			COALESCE(SUM(qn_01), 0) AS days_worked,
			COALESCE(SUM(qn_02), 0) AS ward_rounds,
			COALESCE(SUM(qn_03), 0) AS patients_reviewed,
			COALESCE(SUM(qn_05), 0) + COALESCE(SUM(qn_06), 0) AS procedures
		FROM clinician_app.weeklyreport
		WHERE employee = $1 AND start = $2::date
	`
	if err := db.QueryRowContext(ctx, summaryQuery, employeeID, selectedOption.StartDate).Scan(
		&snapshot.WeeksEntered,
		&snapshot.WeeksSubmitted,
		&snapshot.DaysWorked,
		&snapshot.WardRounds,
		&snapshot.PatientsReviewed,
		&snapshot.Procedures,
	); err != nil {
		return snapshot, err
	}

	snapshot.SelectedWeekSubmitted = snapshot.WeeksSubmitted > 0

	if snapshot.WeeksEntered > 0 {
		snapshot.AttendanceRate = int(float64(snapshot.DaysWorked) / 5.0 * 100)
		snapshot.SubmissionProgress = int(float64(snapshot.WeeksSubmitted) / float64(snapshot.WeeksEntered) * 100)
	}

	if snapshot.AttendanceRate > 100 {
		snapshot.AttendanceRate = 100
	}
	if snapshot.SubmissionProgress > 100 {
		snapshot.SubmissionProgress = 100
	}

	snapshot.OutputTarget = getDashboardTarget(ctx, db, 1, 40)
	snapshot.AttendanceTarget = getDashboardTarget(ctx, db, 2, 95)
	snapshot.WardRoundsTarget = getDashboardTarget(ctx, db, 3, 25)

	trendQuery := `
		SELECT
			start,
			COALESCE(qn_01, 0) AS days_worked,
			COALESCE(qn_03, 0) AS patients_reviewed,
			COALESCE(qn_05, 0) + COALESCE(qn_06, 0) AS procedures
		FROM clinician_app.weeklyreport
		WHERE employee = $1 AND EXTRACT(ISOYEAR FROM start) = $2
		ORDER BY start
	`

	rows, err := db.QueryContext(ctx, trendQuery, employeeID, snapshot.SelectedYear)
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
		_, week := start.ISOWeek()
		point.WeekLabel = fmt.Sprintf("Wk %02d", week)
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
