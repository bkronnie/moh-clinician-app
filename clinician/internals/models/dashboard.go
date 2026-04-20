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
	LastSubmittedOn   sql.NullTime
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
	FacilityName      string
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
	FacilityName      string
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
	WardRounds       int    `json:"wardRounds"`
	PatientsReviewed int    `json:"patientsReviewed"`
	Procedures       int    `json:"procedures"`
}

type ScopeTrendPoint struct {
	WeekLabel        string
	StartDate        string
	EnteredCount     int
	SubmittedCount   int
	PendingCount     int
	PatientsReviewed int
	Procedures       int
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
	NationalReportingRate  int
	FacilityPerformance    []FacilityPerformanceRow
}

func GetDashboardReportPeriods(ctx context.Context, db *sql.DB) ([]time.Time, error) {
	rows, err := db.QueryContext(ctx, `
		SELECT DISTINCT start
		FROM clinician_app.weeklyreport
		WHERE start IS NOT NULL
		ORDER BY start DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	periods := []time.Time{}
	for rows.Next() {
		var start time.Time
		if err := rows.Scan(&start); err != nil {
			return nil, err
		}
		periods = append(periods, start)
	}

	return periods, rows.Err()
}

func GetDashboardFacilityOptions(ctx context.Context, db *sql.DB) ([]DashboardFilterOption, error) {
	rows, err := db.QueryContext(ctx, `
		SELECT id, f_name
		FROM clinician_app.facilities
		ORDER BY f_name
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	options := []DashboardFilterOption{}
	for rows.Next() {
		var option DashboardFilterOption
		if err := rows.Scan(&option.ID, &option.Name); err != nil {
			return nil, err
		}
		options = append(options, option)
	}

	return options, rows.Err()
}

func GetDashboardDepartmentOptions(ctx context.Context, db *sql.DB, facilityID int) ([]DashboardFilterOption, error) {
	rows, err := db.QueryContext(ctx, `
		SELECT DISTINCT d.id, d.d_name
		FROM clinician_app.departments d
		JOIN clinician_app.employees e ON e.department = d.id
		WHERE ($1 = 0 OR e.facility = $1)
		ORDER BY d.d_name
	`, facilityID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	options := []DashboardFilterOption{}
	for rows.Next() {
		var option DashboardFilterOption
		if err := rows.Scan(&option.ID, &option.Name); err != nil {
			return nil, err
		}
		options = append(options, option)
	}

	return options, rows.Err()
}

func GetDashboardEmployeeOptions(ctx context.Context, db *sql.DB, facilityID int, departmentID int) ([]DashboardFilterOption, error) {
	rows, err := db.QueryContext(ctx, `
		SELECT
			e.id,
			TRIM(CONCAT(COALESCE(e.fname, ''), ' ', COALESCE(e.lname, ''))) AS employee_name
		FROM clinician_app.employees e
		WHERE ($1 = 0 OR e.facility = $1)
			AND ($2 = 0 OR e.department = $2)
		ORDER BY employee_name
	`, facilityID, departmentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	options := []DashboardFilterOption{}
	for rows.Next() {
		var option DashboardFilterOption
		if err := rows.Scan(&option.ID, &option.Name); err != nil {
			return nil, err
		}
		options = append(options, option)
	}

	return options, rows.Err()
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
	SelectedMonth         int
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
	AvailableMonths       []DashboardFilterOption
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

func GetNationalDashboardSnapshotByRange(ctx context.Context, db *sql.DB, periodStart, periodEnd time.Time, facilityID int, departmentID int, employeeID int) (NationalDashboardSnapshot, error) {
	var snapshot NationalDashboardSnapshot

	const sqlstr = `
		WITH employee_scope AS (
			SELECT
				e.id,
				e.facility
			FROM clinician_app.employees e
			WHERE ($3 = 0 OR e.facility = $3)
				AND ($4 = 0 OR e.department = $4)
				AND ($5 = 0 OR e.id = $5)
		),
		facility_base AS (
			SELECT
				f.id,
				f.f_name,
				COUNT(DISTINCT es.id) AS clinicians
			FROM clinician_app.facilities f
			LEFT JOIN employee_scope es ON es.facility = f.id
			WHERE ($3 = 0 OR f.id = $3)
			GROUP BY f.id, f.f_name
		),
		filtered_reports AS (
			SELECT
				w.hospital,
				w.employee,
				w.submit_status,
				w.report_status,
				w.created_on,
				COALESCE(w.patients_reviewed, 0) AS patients_reviewed,
				COALESCE(w.elective, 0) + COALESCE(w.emergency, 0) AS procedures
			FROM clinician_app.weeklyreport w
			JOIN employee_scope es ON es.id = w.employee
			WHERE w.start BETWEEN $1 AND $2
				AND ($3 = 0 OR w.hospital = $3)
		)
		SELECT
			fb.id,
			fb.f_name,
			fb.clinicians,
			COUNT(DISTINCT fr.employee) AS entered_reports,
			COUNT(DISTINCT CASE WHEN COALESCE(fr.submit_status, '') = 'Submitted' THEN fr.employee END) AS submitted_reports,
			COUNT(DISTINCT CASE
				WHEN COALESCE(fr.submit_status, '') = 'Submitted'
					AND COALESCE(fr.report_status, '') NOT IN ('Approved', 'Rejected', 'Declined')
				THEN fr.employee
			END) AS pending_approval,
			MAX(CASE WHEN COALESCE(fr.submit_status, '') = 'Submitted' THEN fr.created_on END) AS last_submitted_on
		FROM facility_base fb
		LEFT JOIN filtered_reports fr ON fr.hospital = fb.id
		GROUP BY fb.id, fb.f_name, fb.clinicians
		ORDER BY fb.f_name
	`

	rows, err := db.QueryContext(ctx, sqlstr, periodStart, periodEnd, facilityID, departmentID, employeeID)
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
			&row.LastSubmittedOn,
		); err != nil {
			return snapshot, err
		}

		row.PendingThisWeek = row.Clinicians - row.SubmittedThisWeek
		if row.PendingThisWeek < 0 {
			row.PendingThisWeek = 0
		}
		row.ReportingRate = percentageInt(row.SubmittedThisWeek, row.Clinicians)

		snapshot.TotalFacilities++
		snapshot.TotalClinicians += row.Clinicians
		snapshot.ReportsEnteredThisWeek += row.EnteredThisWeek
		snapshot.SubmittedThisWeek += row.SubmittedThisWeek
		snapshot.PendingApproval += row.PendingApproval
		snapshot.FacilityPerformance = append(snapshot.FacilityPerformance, row)
	}

	if err := rows.Err(); err != nil {
		return snapshot, err
	}

	snapshot.NationalReportingRate = percentageInt(snapshot.SubmittedThisWeek, snapshot.TotalClinicians)
	return snapshot, nil
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

func GetFacilityManagerDashboardSnapshotByRange(ctx context.Context, db *sql.DB, facilityID int, departmentID int, employeeID int, periodStart time.Time, periodEnd time.Time) (FacilityManagerDashboardSnapshot, error) {
	var snapshot FacilityManagerDashboardSnapshot

	const countQuery = `
		WITH employee_scope AS (
			SELECT
				e.id,
				e.department
			FROM clinician_app.employees e
			WHERE e.facility = $1
				AND ($2 = 0 OR e.department = $2)
				AND ($3 = 0 OR e.id = $3)
		),
		report_scope AS (
			SELECT
				w.employee,
				w.submit_status
			FROM clinician_app.weeklyreport w
			JOIN employee_scope es ON es.id = w.employee
			WHERE w.hospital = $1
				AND w.start BETWEEN $4 AND $5
		)
		SELECT
			f.f_name,
			COUNT(DISTINCT es.id) AS total_clinicians,
			COUNT(DISTINCT rs.employee) AS entered_this_period,
			COUNT(DISTINCT CASE WHEN COALESCE(rs.submit_status, '') = 'Submitted' THEN rs.employee END) AS submitted_this_period
		FROM clinician_app.facilities f
		LEFT JOIN employee_scope es ON TRUE
		LEFT JOIN report_scope rs ON rs.employee = es.id
		WHERE f.id = $1
		GROUP BY f.f_name
	`
	if err := db.QueryRowContext(ctx, countQuery, facilityID, departmentID, employeeID, periodStart, periodEnd).Scan(
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

	const departmentQuery = `
		WITH employee_scope AS (
			SELECT
				e.id,
				e.department
			FROM clinician_app.employees e
			WHERE e.facility = $1
				AND ($2 = 0 OR e.department = $2)
				AND ($3 = 0 OR e.id = $3)
		),
		report_scope AS (
			SELECT
				w.employee,
				w.department,
				w.submit_status
			FROM clinician_app.weeklyreport w
			JOIN employee_scope es ON es.id = w.employee
			WHERE w.hospital = $1
				AND w.start BETWEEN $4 AND $5
		)
		SELECT
			d.id,
			d.d_name,
			COUNT(DISTINCT es.id) AS clinicians,
			COUNT(DISTINCT rs.employee) AS entered_this_period,
			COUNT(DISTINCT CASE WHEN COALESCE(rs.submit_status, '') = 'Submitted' THEN rs.employee END) AS submitted_this_period
		FROM clinician_app.departments d
		LEFT JOIN employee_scope es ON es.department = d.id
		LEFT JOIN report_scope rs ON rs.department = d.id AND rs.employee = es.id
		WHERE ($2 = 0 OR d.id = $2)
		GROUP BY d.id, d.d_name
		HAVING COUNT(DISTINCT es.id) > 0
		ORDER BY d.d_name
	`

	rows, err := db.QueryContext(ctx, departmentQuery, facilityID, departmentID, employeeID, periodStart, periodEnd)
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

func GetClinicianDashboardSnapshot(ctx context.Context, db *sql.DB, employeeID int, selectedYear int, selectedMonth int, selectedWeek int) (ClinicianDashboardSnapshot, error) {
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

	availableMonths := []DashboardFilterOption{{ID: 0, Name: "All"}}
	monthSeen := map[int]bool{}
	for _, option := range allWeekOptions {
		if selectedYear > 0 && option.Year != selectedYear {
			continue
		}
		monthID := int(mustParseDate(option.StartDate).Month())
		if monthSeen[monthID] {
			continue
		}
		monthSeen[monthID] = true
		availableMonths = append(availableMonths, DashboardFilterOption{
			ID:   monthID,
			Name: time.Month(monthID).String(),
		})
	}

	if selectedYear != 0 && !containsInt(snapshot.AvailableYears, selectedYear) {
		selectedYear = 0
	}
	if selectedMonth != 0 && !containsMonthOptionModel(availableMonths, selectedMonth) {
		selectedMonth = 0
	}

	filteredWeeks := []ClinicianWeekOption{{
		Year:  0,
		Week:  0,
		Label: "All",
	}}
	for _, option := range allWeekOptions {
		if selectedYear > 0 && option.Year != selectedYear {
			continue
		}
		if selectedMonth > 0 && int(mustParseDate(option.StartDate).Month()) != selectedMonth {
			continue
		}
		filteredWeeks = append(filteredWeeks, option)
	}

	if selectedWeek != 0 {
		foundWeek := false
		for _, option := range filteredWeeks {
			if option.Week == selectedWeek {
				foundWeek = true
				break
			}
		}
		if !foundWeek {
			selectedWeek = 0
		}
	}

	snapshot.SelectedYear = selectedYear
	snapshot.SelectedMonth = selectedMonth
	snapshot.AvailableMonths = availableMonths
	snapshot.AvailableWeeks = filteredWeeks

	var selectedOption ClinicianWeekOption
	var hasSelectedOption bool
	if selectedWeek != 0 {
		selectedOption, hasSelectedOption = resolveWeekSelection(filteredWeeks[1:], selectedWeek)
		if !hasSelectedOption {
			selectedWeek = 0
		}
	}

	snapshot.SelectedWeek = selectedWeek
	switch {
	case selectedYear == 0 && selectedMonth == 0 && selectedWeek == 0:
		snapshot.SelectedWeek = 0
		snapshot.SelectedWeekLabel = "All records"
	case selectedWeek == 0 && selectedMonth > 0:
		snapshot.SelectedWeekLabel = fmt.Sprintf("All weeks in %s", time.Month(selectedMonth).String())
		if selectedYear > 0 {
			snapshot.SelectedWeekLabel += fmt.Sprintf(" %d", selectedYear)
		}
	case selectedWeek == 0 && selectedYear > 0:
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
				COALESCE(SUM(attendance), 0) AS days_worked,
				COALESCE(SUM(ward_rounds), 0) AS ward_rounds,
				COALESCE(SUM(patients_reviewed), 0) AS patients_reviewed,
				COALESCE(SUM(elective), 0) + COALESCE(SUM(emergency), 0) AS procedures
			FROM clinician_app.weeklyreport
			WHERE employee = $1
		`
		summaryArgs = []interface{}{employeeID}
	case selectedWeek == 0 && selectedMonth == 0:
		summaryQuery = `
			SELECT
				COUNT(DISTINCT start) AS weeks_entered,
				COUNT(DISTINCT CASE WHEN submit_status = 'Submitted' THEN start END) AS weeks_submitted,
				COALESCE(SUM(attendance), 0) AS days_worked,
				COALESCE(SUM(ward_rounds), 0) AS ward_rounds,
				COALESCE(SUM(patients_reviewed), 0) AS patients_reviewed,
				COALESCE(SUM(elective), 0) + COALESCE(SUM(emergency), 0) AS procedures
			FROM clinician_app.weeklyreport
			WHERE employee = $1 AND EXTRACT(ISOYEAR FROM start) = $2
		`
		summaryArgs = []interface{}{employeeID, selectedYear}
	case selectedWeek == 0:
		summaryQuery = `
			SELECT
				COUNT(DISTINCT start) AS weeks_entered,
				COUNT(DISTINCT CASE WHEN submit_status = 'Submitted' THEN start END) AS weeks_submitted,
				COALESCE(SUM(attendance), 0) AS days_worked,
				COALESCE(SUM(ward_rounds), 0) AS ward_rounds,
				COALESCE(SUM(patients_reviewed), 0) AS patients_reviewed,
				COALESCE(SUM(elective), 0) + COALESCE(SUM(emergency), 0) AS procedures
			FROM clinician_app.weeklyreport
			WHERE employee = $1 AND EXTRACT(ISOYEAR FROM start) = $2 AND EXTRACT(MONTH FROM start) = $3
		`
		summaryArgs = []interface{}{employeeID, selectedYear, selectedMonth}
	default:
		summaryQuery = `
			SELECT
				COUNT(DISTINCT start) AS weeks_entered,
				COUNT(DISTINCT CASE WHEN submit_status = 'Submitted' THEN start END) AS weeks_submitted,
				COALESCE(SUM(attendance), 0) AS days_worked,
				COALESCE(SUM(ward_rounds), 0) AS ward_rounds,
				COALESCE(SUM(patients_reviewed), 0) AS patients_reviewed,
				COALESCE(SUM(elective), 0) + COALESCE(SUM(emergency), 0) AS procedures
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

	reportSummary, err := GetClinicianReportHistorySummary(ctx, db, int64(employeeID), snapshot.SelectedYear, snapshot.SelectedMonth, snapshot.SelectedWeek)
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
				COALESCE(attendance, 0) AS days_worked,
				COALESCE(ward_rounds, 0) AS ward_rounds,
				COALESCE(patients_reviewed, 0) AS patients_reviewed,
				COALESCE(elective, 0) + COALESCE(emergency, 0) AS procedures
			FROM clinician_app.weeklyreport
			WHERE employee = $1
			ORDER BY start
		`
		trendArgs = []interface{}{employeeID}
	} else if selectedMonth == 0 {
		trendQuery = `
			SELECT
				start,
				COALESCE(attendance, 0) AS days_worked,
				COALESCE(ward_rounds, 0) AS ward_rounds,
				COALESCE(patients_reviewed, 0) AS patients_reviewed,
				COALESCE(elective, 0) + COALESCE(emergency, 0) AS procedures
			FROM clinician_app.weeklyreport
			WHERE employee = $1 AND EXTRACT(ISOYEAR FROM start) = $2
			ORDER BY start
		`
		trendArgs = []interface{}{employeeID, snapshot.SelectedYear}
	} else {
		trendQuery = `
			SELECT
				start,
				COALESCE(attendance, 0) AS days_worked,
				COALESCE(ward_rounds, 0) AS ward_rounds,
				COALESCE(patients_reviewed, 0) AS patients_reviewed,
				COALESCE(elective, 0) + COALESCE(emergency, 0) AS procedures
			FROM clinician_app.weeklyreport
			WHERE employee = $1 AND EXTRACT(ISOYEAR FROM start) = $2 AND EXTRACT(MONTH FROM start) = $3
			ORDER BY start
		`
		trendArgs = []interface{}{employeeID, snapshot.SelectedYear, snapshot.SelectedMonth}
	}

	rows, err := db.QueryContext(ctx, trendQuery, trendArgs...)
	if err != nil {
		return snapshot, err
	}
	defer rows.Close()

	for rows.Next() {
		var start time.Time
		var point ClinicianTrendPoint
		if err := rows.Scan(&start, &point.DaysWorked, &point.WardRounds, &point.PatientsReviewed, &point.Procedures); err != nil {
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

func containsMonthOptionModel(options []DashboardFilterOption, target int) bool {
	for _, option := range options {
		if option.ID == target {
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
				COALESCE(SUM(COALESCE(w.patients_reviewed, 0)), 0) AS patients_reviewed,
				COALESCE(SUM(COALESCE(w.elective, 0)), 0) + COALESCE(SUM(COALESCE(w.emergency, 0)), 0) AS procedures
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

func GetFacilityDepartmentAnalysisByRange(ctx context.Context, db *sql.DB, facilityID int, departmentID int, employeeID int, periodStart time.Time, periodEnd time.Time) ([]DepartmentAnalysisRow, error) {
	const sqlstr = `
		WITH employee_scope AS (
			SELECT
				e.id,
				e.department
			FROM clinician_app.employees e
			WHERE e.facility = $1
				AND ($2 = 0 OR e.department = $2)
				AND ($3 = 0 OR e.id = $3)
		),
		department_base AS (
			SELECT
				d.id AS department_id,
				d.d_name AS department_name,
				COUNT(DISTINCT es.id) AS clinicians
			FROM clinician_app.departments d
			LEFT JOIN employee_scope es ON es.department = d.id
			WHERE ($2 = 0 OR d.id = $2)
			GROUP BY d.id, d.d_name
		),
		report_stats AS (
			SELECT
				w.department AS department_id,
				COUNT(DISTINCT w.employee) AS reports_entered,
				COUNT(DISTINCT CASE WHEN COALESCE(w.submit_status, '') = 'Submitted' THEN w.employee END) AS reports_submitted,
				COALESCE(SUM(COALESCE(w.patients_reviewed, 0)), 0) AS patients_reviewed,
				COALESCE(SUM(COALESCE(w.elective, 0)), 0) + COALESCE(SUM(COALESCE(w.emergency, 0)), 0) AS procedures
			FROM clinician_app.weeklyreport w
			JOIN employee_scope es ON es.id = w.employee
			WHERE w.hospital = $1
				AND w.start BETWEEN $4 AND $5
			GROUP BY w.department
		),
		attendance_stats AS (
			SELECT
				a.department_id,
				COUNT(*) AS attendance_records
			FROM clinician_app.attendance_records a
			JOIN employee_scope es ON es.id = a.specialist_id
			WHERE a.facility_id = $1
				AND a.attendance_date BETWEEN $4 AND $5
			GROUP BY a.department_id
		),
		round_stats AS (
			SELECT
				w.department_id,
				COUNT(*) AS ward_rounds
			FROM clinician_app.ward_rounds w
			JOIN employee_scope es ON es.id = w.specialist_id
			WHERE w.facility_id = $1
				AND w.round_date BETWEEN $4 AND $5
			GROUP BY w.department_id
		)
		SELECT
			db.department_id,
			db.department_name,
			'' AS facility_name,
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

	rows, err := db.QueryContext(ctx, sqlstr, facilityID, departmentID, employeeID, periodStart, periodEnd)
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
			&row.FacilityName,
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

func GetNationalDepartmentAnalysisByRange(ctx context.Context, db *sql.DB, facilityID int, departmentID int, employeeID int, periodStart time.Time, periodEnd time.Time) ([]DepartmentAnalysisRow, error) {
	const sqlstr = `
		WITH employee_scope AS (
			SELECT
				e.id,
				e.department,
				e.facility
			FROM clinician_app.employees e
			WHERE ($1 = 0 OR e.facility = $1)
				AND ($2 = 0 OR e.department = $2)
				AND ($3 = 0 OR e.id = $3)
		),
		department_base AS (
			SELECT
				d.id AS department_id,
				COALESCE(d.d_name, '') AS department_name,
				es.facility AS facility_id,
				COALESCE(f.f_name, '') AS facility_name,
				COUNT(DISTINCT es.id) AS clinicians
			FROM employee_scope es
			JOIN clinician_app.departments d ON d.id = es.department
			JOIN clinician_app.facilities f ON f.id = es.facility
			GROUP BY d.id, d.d_name, es.facility, f.f_name
		),
		report_stats AS (
			SELECT
				w.hospital AS facility_id,
				w.department AS department_id,
				COUNT(DISTINCT w.employee) AS reports_entered,
				COUNT(DISTINCT CASE WHEN COALESCE(w.submit_status, '') = 'Submitted' THEN w.employee END) AS reports_submitted,
				COALESCE(SUM(COALESCE(w.patients_reviewed, 0)), 0) AS patients_reviewed,
				COALESCE(SUM(COALESCE(w.elective, 0)), 0) + COALESCE(SUM(COALESCE(w.emergency, 0)), 0) AS procedures
			FROM clinician_app.weeklyreport w
			JOIN employee_scope es ON es.id = w.employee
			WHERE w.start BETWEEN $4 AND $5
			GROUP BY w.hospital, w.department
		),
		attendance_stats AS (
			SELECT
				a.facility_id,
				a.department_id,
				COUNT(*) AS attendance_records
			FROM clinician_app.attendance_records a
			JOIN employee_scope es ON es.id = a.specialist_id
			WHERE a.attendance_date BETWEEN $4 AND $5
			GROUP BY a.facility_id, a.department_id
		),
		round_stats AS (
			SELECT
				w.facility_id,
				w.department_id,
				COUNT(*) AS ward_rounds
			FROM clinician_app.ward_rounds w
			JOIN employee_scope es ON es.id = w.specialist_id
			WHERE w.round_date BETWEEN $4 AND $5
			GROUP BY w.facility_id, w.department_id
		)
		SELECT
			db.department_id,
			db.department_name,
			db.facility_name,
			db.clinicians,
			COALESCE(rs.reports_entered, 0) AS reports_entered,
			COALESCE(rs.reports_submitted, 0) AS reports_submitted,
			COALESCE(att.attendance_records, 0) AS attendance_records,
			COALESCE(rnd.ward_rounds, 0) AS ward_rounds,
			COALESCE(rs.patients_reviewed, 0) AS patients_reviewed,
			COALESCE(rs.procedures, 0) AS procedures
		FROM department_base db
		LEFT JOIN report_stats rs ON rs.department_id = db.department_id AND rs.facility_id = db.facility_id
		LEFT JOIN attendance_stats att ON att.department_id = db.department_id AND att.facility_id = db.facility_id
		LEFT JOIN round_stats rnd ON rnd.department_id = db.department_id AND rnd.facility_id = db.facility_id
		WHERE db.clinicians > 0
		ORDER BY db.facility_name, db.department_name
	`

	rows, err := db.QueryContext(ctx, sqlstr, facilityID, departmentID, employeeID, periodStart, periodEnd)
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
			&row.FacilityName,
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
				COALESCE(SUM(COALESCE(w.patients_reviewed, 0)), 0) AS patients_reviewed,
				COALESCE(SUM(COALESCE(w.elective, 0)), 0) + COALESCE(SUM(COALESCE(w.emergency, 0)), 0) AS procedures
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
			'' AS facility_name,
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
			&row.FacilityName,
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

func GetNationalClinicianAnalysisByRange(ctx context.Context, db *sql.DB, facilityID int, departmentID int, employeeID int, periodStart time.Time, periodEnd time.Time) ([]ClinicianAnalysisRow, error) {
	const sqlstr = `
		WITH employee_scope AS (
			SELECT
				e.id,
				e.department,
				e.facility
			FROM clinician_app.employees e
			WHERE ($1 = 0 OR e.facility = $1)
				AND ($2 = 0 OR e.department = $2)
				AND ($3 = 0 OR e.id = $3)
		),
		report_stats AS (
			SELECT
				w.employee AS employee_id,
				COUNT(*) AS reports_entered,
				COUNT(*) FILTER (WHERE COALESCE(w.submit_status, '') = 'Submitted') AS reports_submitted,
				MAX(COALESCE(w.submit_status, '')) AS submit_status,
				MAX(COALESCE(w.report_status, '')) AS report_status,
				COALESCE(SUM(COALESCE(w.patients_reviewed, 0)), 0) AS patients_reviewed,
				COALESCE(SUM(COALESCE(w.elective, 0)), 0) + COALESCE(SUM(COALESCE(w.emergency, 0)), 0) AS procedures
			FROM clinician_app.weeklyreport w
			JOIN employee_scope es ON es.id = w.employee
			WHERE w.start BETWEEN $4 AND $5
			GROUP BY w.employee
		),
		attendance_stats AS (
			SELECT
				a.specialist_id AS employee_id,
				COUNT(*) AS attendance_records
			FROM clinician_app.attendance_records a
			JOIN employee_scope es ON es.id = a.specialist_id
			WHERE a.attendance_date BETWEEN $4 AND $5
			GROUP BY a.specialist_id
		),
		round_stats AS (
			SELECT
				w.specialist_id AS employee_id,
				COUNT(*) AS ward_rounds
			FROM clinician_app.ward_rounds w
			JOIN employee_scope es ON es.id = w.specialist_id
			WHERE w.round_date BETWEEN $4 AND $5
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
			JOIN employee_scope es ON es.id = s.employee_id
			ORDER BY s.employee_id, s.created_on DESC, s.leave_id DESC
		)
		SELECT
			e.id,
			TRIM(CONCAT(COALESCE(e.fname, ''), ' ', COALESCE(e.lname, ''))) AS employee_name,
			COALESCE(st.title, '') AS title_name,
			COALESCE(d.d_name, '') AS department_name,
			COALESCE(f.f_name, '') AS facility_name,
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
		JOIN employee_scope es ON es.id = e.id
		LEFT JOIN clinician_app.departments d ON d.id = e.department
		LEFT JOIN clinician_app.facilities f ON f.id = e.facility
		LEFT JOIN clinician_app.specialist_titles st ON st.id = e.title
		LEFT JOIN report_stats rs ON rs.employee_id = e.id
		LEFT JOIN attendance_stats att ON att.employee_id = e.id
		LEFT JOIN round_stats rnd ON rnd.employee_id = e.id
		LEFT JOIN latest_leave ll ON ll.employee_id = e.id
		ORDER BY f.f_name, d.d_name, employee_name
	`

	rows, err := db.QueryContext(ctx, sqlstr, facilityID, departmentID, employeeID, periodStart, periodEnd)
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
			&row.FacilityName,
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

func GetFacilityClinicianAnalysisByRange(ctx context.Context, db *sql.DB, facilityID int, departmentID int, employeeID int, periodStart time.Time, periodEnd time.Time) ([]ClinicianAnalysisRow, error) {
	const sqlstr = `
		WITH employee_scope AS (
			SELECT
				e.id,
				e.department
			FROM clinician_app.employees e
			WHERE e.facility = $1
				AND ($2 = 0 OR e.department = $2)
				AND ($3 = 0 OR e.id = $3)
		),
		report_stats AS (
			SELECT
				w.employee AS employee_id,
				COUNT(*) AS reports_entered,
				COUNT(*) FILTER (WHERE COALESCE(w.submit_status, '') = 'Submitted') AS reports_submitted,
				MAX(COALESCE(w.submit_status, '')) AS submit_status,
				MAX(COALESCE(w.report_status, '')) AS report_status,
				COALESCE(SUM(COALESCE(w.patients_reviewed, 0)), 0) AS patients_reviewed,
				COALESCE(SUM(COALESCE(w.elective, 0)), 0) + COALESCE(SUM(COALESCE(w.emergency, 0)), 0) AS procedures
			FROM clinician_app.weeklyreport w
			JOIN employee_scope es ON es.id = w.employee
			WHERE w.hospital = $1
				AND w.start BETWEEN $4 AND $5
			GROUP BY w.employee
		),
		attendance_stats AS (
			SELECT
				a.specialist_id AS employee_id,
				COUNT(*) AS attendance_records
			FROM clinician_app.attendance_records a
			JOIN employee_scope es ON es.id = a.specialist_id
			WHERE a.facility_id = $1
				AND a.attendance_date BETWEEN $4 AND $5
			GROUP BY a.specialist_id
		),
		round_stats AS (
			SELECT
				w.specialist_id AS employee_id,
				COUNT(*) AS ward_rounds
			FROM clinician_app.ward_rounds w
			JOIN employee_scope es ON es.id = w.specialist_id
			WHERE w.facility_id = $1
				AND w.round_date BETWEEN $4 AND $5
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
			JOIN employee_scope es ON es.id = s.employee_id
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
		JOIN employee_scope es ON es.id = e.id
		LEFT JOIN clinician_app.departments d ON d.id = e.department
		LEFT JOIN clinician_app.specialist_titles st ON st.id = e.title
		LEFT JOIN report_stats rs ON rs.employee_id = e.id
		LEFT JOIN attendance_stats att ON att.employee_id = e.id
		LEFT JOIN round_stats rnd ON rnd.employee_id = e.id
		LEFT JOIN latest_leave ll ON ll.employee_id = e.id
		ORDER BY d.d_name, employee_name
	`

	rows, err := db.QueryContext(ctx, sqlstr, facilityID, departmentID, employeeID, periodStart, periodEnd)
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

func GetScopeTrendPoints(ctx context.Context, db *sql.DB, scopeFacilityID int, facilityID int, departmentID int, employeeID int, year int, month int, week int) ([]ScopeTrendPoint, error) {
	const sqlstr = `
		WITH employee_scope AS (
			SELECT
				e.id,
				e.facility
			FROM clinician_app.employees e
			WHERE ($1 = 0 OR e.facility = $1)
				AND ($2 = 0 OR e.facility = $2)
				AND ($3 = 0 OR e.department = $3)
				AND ($4 = 0 OR e.id = $4)
		)
		SELECT
			w.start,
			COUNT(DISTINCT w.employee) AS entered_count,
			COUNT(DISTINCT CASE WHEN COALESCE(w.submit_status, '') = 'Submitted' THEN w.employee END) AS submitted_count,
			COUNT(DISTINCT CASE
				WHEN COALESCE(w.submit_status, '') = 'Submitted'
					AND COALESCE(w.report_status, '') NOT IN ('Approved', 'Rejected', 'Declined')
				THEN w.employee
			END) AS pending_count,
			COALESCE(SUM(COALESCE(w.patients_reviewed, 0)), 0) AS patients_reviewed,
			COALESCE(SUM(COALESCE(w.elective, 0)), 0) + COALESCE(SUM(COALESCE(w.emergency, 0)), 0) AS procedures
		FROM clinician_app.weeklyreport w
		JOIN employee_scope es ON es.id = w.employee
		WHERE ($5 = 0 OR EXTRACT(ISOYEAR FROM w.start) = $5)
			AND ($6 = 0 OR EXTRACT(MONTH FROM w.start) = $6)
			AND ($7 = 0 OR EXTRACT(WEEK FROM w.start) = $7)
		GROUP BY w.start
		ORDER BY w.start
	`

	rows, err := db.QueryContext(ctx, sqlstr, scopeFacilityID, facilityID, departmentID, employeeID, year, month, week)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := []ScopeTrendPoint{}
	for rows.Next() {
		var start time.Time
		var point ScopeTrendPoint
		if err := rows.Scan(&start, &point.EnteredCount, &point.SubmittedCount, &point.PendingCount, &point.PatientsReviewed, &point.Procedures); err != nil {
			return nil, err
		}
		point.StartDate = start.Format("2006-01-02")
		yearVal, weekVal := start.ISOWeek()
		point.WeekLabel = fmt.Sprintf("Wk %02d %d", weekVal, yearVal)
		items = append(items, point)
	}

	return items, rows.Err()
}

func GetClinicianMissingRequiredReportsCount(ctx context.Context, db *sql.DB, employeeID int) (int, error) {
	const sqlstr = `
		WITH employee_window AS (
			SELECT
				date_trunc('week', COALESCE(e.created_on, CURRENT_DATE::timestamp))::date AS start_week,
				date_trunc('week', CURRENT_DATE)::date AS end_week
			FROM clinician_app.employees e
			WHERE e.id = $1
		),
		required_weeks AS (
			SELECT gs::date AS week_start
			FROM employee_window ew,
				generate_series(ew.start_week, ew.end_week, INTERVAL '7 days') gs
		),
		on_leave_weeks AS (
			SELECT DISTINCT rw.week_start
			FROM required_weeks rw
			JOIN clinician_app.staffleave sl
				ON sl.employee_id = $1
			WHERE COALESCE(sl.leave_status, '') IN ('Approved', 'Valid')
				AND sl.start_date::date <= (rw.week_start + INTERVAL '6 days')::date
				AND COALESCE(sl.return_date::date, sl.end_date::date) >= rw.week_start
		),
		reported_weeks AS (
			SELECT DISTINCT w.start::date AS week_start
			FROM clinician_app.weeklyreport w
			WHERE w.employee = $1
		)
		SELECT COUNT(1)
		FROM required_weeks rw
		LEFT JOIN on_leave_weeks olw ON olw.week_start = rw.week_start
		LEFT JOIN reported_weeks rep ON rep.week_start = rw.week_start
		WHERE olw.week_start IS NULL
			AND rep.week_start IS NULL
	`

	var missing int
	err := db.QueryRowContext(ctx, sqlstr, employeeID).Scan(&missing)
	if err != nil {
		return 0, err
	}
	if missing < 0 {
		return 0, nil
	}
	return missing, nil
}

func GetClinicianPendingEntryWeeks(ctx context.Context, db *sql.DB, employeeID int) ([]ClinicianWeekOption, error) {
	const sqlstr = `
		WITH last_submitted AS (
			SELECT MAX(w.start)::date AS week_start
			FROM clinician_app.weeklyreport w
			WHERE w.employee = $1
				AND COALESCE(w.submit_status, '') = 'Submitted'
		),
		employee_window AS (
			SELECT date_trunc('week', COALESCE(e.created_on, CURRENT_DATE::timestamp))::date AS created_week
			FROM clinician_app.employees e
			WHERE e.id = $1
		),
		bounds AS (
			SELECT
				COALESCE(
					(SELECT (ls.week_start + INTERVAL '7 days')::date FROM last_submitted ls WHERE ls.week_start IS NOT NULL),
					(SELECT ew.created_week FROM employee_window ew)
				) AS start_week,
				(date_trunc('week', CURRENT_DATE)::date - INTERVAL '7 days')::date AS end_week
		),
		candidate_weeks AS (
			SELECT gs::date AS week_start
			FROM bounds b,
				generate_series(b.start_week, b.end_week, INTERVAL '7 days') gs
			WHERE b.start_week IS NOT NULL
				AND b.start_week <= b.end_week
		),
		on_leave_weeks AS (
			SELECT DISTINCT cw.week_start
			FROM candidate_weeks cw
			JOIN clinician_app.staffleave sl ON sl.employee_id = $1
			WHERE COALESCE(sl.leave_status, '') IN ('Approved', 'Valid')
				AND sl.start_date::date <= (cw.week_start + INTERVAL '6 days')::date
				AND COALESCE(sl.return_date::date, sl.end_date::date) >= cw.week_start
		),
		submitted_weeks AS (
			SELECT DISTINCT w.start::date AS week_start
			FROM clinician_app.weeklyreport w
			WHERE w.employee = $1
				AND COALESCE(w.submit_status, '') = 'Submitted'
		)
		SELECT cw.week_start
		FROM candidate_weeks cw
		LEFT JOIN on_leave_weeks olw ON olw.week_start = cw.week_start
		LEFT JOIN submitted_weeks sw ON sw.week_start = cw.week_start
		WHERE olw.week_start IS NULL
			AND sw.week_start IS NULL
		ORDER BY cw.week_start DESC
	`

	rows, err := db.QueryContext(ctx, sqlstr, employeeID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	weeks := make([]ClinicianWeekOption, 0)
	for rows.Next() {
		var weekStart time.Time
		if err := rows.Scan(&weekStart); err != nil {
			return nil, err
		}
		weekStop := weekStart.AddDate(0, 0, 6)
		yearVal, weekVal := weekStart.ISOWeek()
		weeks = append(weeks, ClinicianWeekOption{
			Year:      yearVal,
			Week:      weekVal,
			StartDate: weekStart.Format("2006-01-02"),
			EndDate:   weekStop.Format("2006-01-02"),
			Label:     fmt.Sprintf("Week %02d (%s - %s)", weekVal, weekStart.Format("02 Jan 2006"), weekStop.Format("02 Jan 2006")),
		})
	}

	return weeks, rows.Err()
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
