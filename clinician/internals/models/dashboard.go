package models

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

type FacilityPerformanceRow struct {
	FacilityID         int
	FacilityName       string
	Clinicians         int
	EnteredThisWeek    int
	SubmittedThisWeek  int
	PendingThisWeek    int
}

type DepartmentProgressRow struct {
	DepartmentID       int
	DepartmentName     string
	Clinicians         int
	EnteredThisWeek    int
	SubmittedThisWeek  int
	PendingThisWeek    int
}

type ClinicianTrendPoint struct {
	WeekLabel         string `json:"weekLabel"`
	StartDate         string `json:"startDate"`
	DaysWorked        int    `json:"daysWorked"`
	PatientsReviewed  int    `json:"patientsReviewed"`
	Procedures        int    `json:"procedures"`
}

type NationalDashboardSnapshot struct {
	TotalFacilities       int
	TotalClinicians       int
	ReportsEnteredThisWeek int
	SubmittedThisWeek     int
	PendingApproval       int
	FacilityPerformance   []FacilityPerformanceRow
}

type FacilityManagerDashboardSnapshot struct {
	FacilityName         string
	TotalClinicians      int
	ReportsEnteredThisWeek int
	SubmittedThisWeek    int
	PendingSubmission    int
	Departments          []DepartmentProgressRow
}

type ClinicianDashboardSnapshot struct {
	EmployeeName            string
	FacilityName            string
	DepartmentName          string
	WeeksEntered            int
	WeeksSubmitted          int
	DaysWorked              int
	WardRounds              int
	PatientsReviewed        int
	Procedures              int
	AttendanceRate          int
	OutputTarget            int
	AttendanceTarget        int
	WardRoundsTarget        int
	SubmissionProgress      int
	RecentWeeks             []ClinicianTrendPoint
	LatestWeekLabel         string
	LatestWeekSubmitted     bool
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

func GetClinicianDashboardSnapshot(ctx context.Context, db *sql.DB, employeeID int) (ClinicianDashboardSnapshot, error) {
	var snapshot ClinicianDashboardSnapshot

	summaryQuery := `
		SELECT
			CONCAT(e.fname, ' ', e.lname) AS employee_name,
			f.f_name AS facility_name,
			d.d_name AS department_name,
			COUNT(w.id) AS weeks_entered,
			COUNT(CASE WHEN w.submit_status = 'Submitted' THEN 1 END) AS weeks_submitted,
			COALESCE(SUM(w.qn_01), 0) AS days_worked,
			COALESCE(SUM(w.qn_02), 0) AS ward_rounds,
			COALESCE(SUM(w.qn_03), 0) AS patients_reviewed,
			COALESCE(SUM(w.qn_05), 0) + COALESCE(SUM(w.qn_06), 0) AS procedures
		FROM clinician_app.employees e
		LEFT JOIN clinician_app.facilities f ON f.id = e.facility
		LEFT JOIN clinician_app.departments d ON d.id = e.department
		LEFT JOIN clinician_app.weeklyreport w ON w.employee = e.id
		WHERE e.id = $1
		GROUP BY e.fname, e.lname, f.f_name, d.d_name
	`
	if err := db.QueryRowContext(ctx, summaryQuery, employeeID).Scan(
		&snapshot.EmployeeName,
		&snapshot.FacilityName,
		&snapshot.DepartmentName,
		&snapshot.WeeksEntered,
		&snapshot.WeeksSubmitted,
		&snapshot.DaysWorked,
		&snapshot.WardRounds,
		&snapshot.PatientsReviewed,
		&snapshot.Procedures,
	); err != nil {
		return snapshot, err
	}

	if snapshot.WeeksEntered > 0 {
		snapshot.AttendanceRate = int(float64(snapshot.DaysWorked) / float64(snapshot.WeeksEntered*5) * 100)
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
			COALESCE(qn_05, 0) + COALESCE(qn_06, 0) AS procedures,
			COALESCE(submit_status, '') = 'Submitted' AS submitted
		FROM clinician_app.weeklyreport
		WHERE employee = $1
		ORDER BY start DESC
		LIMIT 8
	`

	rows, err := db.QueryContext(ctx, trendQuery, employeeID)
	if err != nil {
		return snapshot, err
	}
	defer rows.Close()

	var reversed []ClinicianTrendPoint
	var latestDate time.Time
	for rows.Next() {
		var start time.Time
		var point ClinicianTrendPoint
		var submitted bool
		if err := rows.Scan(&start, &point.DaysWorked, &point.PatientsReviewed, &point.Procedures, &submitted); err != nil {
			return snapshot, err
		}

		if latestDate.IsZero() || start.After(latestDate) {
			latestDate = start
			snapshot.LatestWeekSubmitted = submitted
		}

		point.StartDate = start.Format("2006-01-02")
		point.WeekLabel = fmt.Sprintf("Week of %s", start.Format("02 Jan"))
		reversed = append(reversed, point)
	}

	for i := len(reversed) - 1; i >= 0; i-- {
		snapshot.RecentWeeks = append(snapshot.RecentWeeks, reversed[i])
	}

	if !latestDate.IsZero() {
		snapshot.LatestWeekLabel = fmt.Sprintf("Week of %s", latestDate.Format("02 Jan 2006"))
	}

	return snapshot, rows.Err()
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
