package models

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"
)

type ReportSubmissionListRow struct {
	ReportID         int
	EmployeeID       int64
	EmployeeName     string
	FacilityID       int64
	FacilityName     string
	DepartmentID     int64
	DepartmentName   string
	WeekStart        sql.NullTime
	WeekStop         sql.NullTime
	EnteredOn        sql.NullTime
	SubmittedOn      sql.NullTime
	SubmitStatus     sql.NullString
	ReportStatus     sql.NullString
	Attendance       int
	PatientsReviewed int
	Procedures       int
	Actionable       bool
	IsOnLeave        bool
}

type FacilitySubmissionSummaryRow struct {
	FacilityID      int64
	FacilityName    string
	OnDutyCount     int
	OnLeaveCount    int
	SubmittedCount  int
	ApprovedCount   int
	DeclinedCount   int
	PendingCount    int
	MissingCount    int
	SubmittedOn     sql.NullTime
	ApprovalStatus  string
	SubmissionState string
	CanApprove      bool
}

func GetReportSubmissions(ctx context.Context, db *sql.DB, scopeFacilityID int64, scopeEmployeeID int64, filterFacilityID int, filterDepartmentID int, filterStatus string, year int, month int, week int) ([]*ReportSubmissionListRow, error) {
	args := []interface{}{}
	whereParts := []string{}
	argPos := 1

	effectiveFacilityID := filterFacilityID
	if scopeFacilityID > 0 {
		effectiveFacilityID = int(scopeFacilityID)
	}
	if scopeEmployeeID > 0 {
		whereParts = append(whereParts, fmt.Sprintf("w.employee = $%d", argPos))
		args = append(args, scopeEmployeeID)
		argPos++
	} else {
		// Draft rows are private to the creating staff member and are hidden from review roles.
		whereParts = append(whereParts, "NOT (COALESCE(w.submit_status, '') <> 'Submitted' AND COALESCE(w.report_status, '') NOT IN ('Approved', 'Rejected', 'Declined'))")
	}

	if effectiveFacilityID > 0 {
		whereParts = append(whereParts, fmt.Sprintf("w.hospital = $%d", argPos))
		args = append(args, effectiveFacilityID)
		argPos++
	}
	if filterDepartmentID > 0 {
		whereParts = append(whereParts, fmt.Sprintf("w.department = $%d", argPos))
		args = append(args, filterDepartmentID)
		argPos++
	}
	switch filterStatus {
	case "submitted":
		whereParts = append(whereParts, "COALESCE(w.submit_status, '') = 'Submitted'")
		whereParts = append(whereParts, "COALESCE(w.report_status, '') NOT IN ('Approved', 'Rejected', 'Declined')")
	case "pending":
		whereParts = append(whereParts, "COALESCE(w.submit_status, '') = 'Submitted'")
		whereParts = append(whereParts, "COALESCE(w.report_status, '') NOT IN ('Approved', 'Rejected', 'Declined')")
	case "approved":
		whereParts = append(whereParts, "COALESCE(w.report_status, '') = 'Approved'")
	case "declined":
		whereParts = append(whereParts, "COALESCE(w.report_status, '') IN ('Rejected', 'Declined')")
	case "draft":
		whereParts = append(whereParts, "COALESCE(w.submit_status, '') <> 'Submitted'")
		whereParts = append(whereParts, "COALESCE(w.report_status, '') NOT IN ('Approved', 'Rejected', 'Declined')")
	}
	if year > 0 {
		whereParts = append(whereParts, fmt.Sprintf("EXTRACT(ISOYEAR FROM w.start) = $%d", argPos))
		args = append(args, year)
		argPos++
	}
	if month > 0 {
		whereParts = append(whereParts, fmt.Sprintf("EXTRACT(MONTH FROM w.start) = $%d", argPos))
		args = append(args, month)
		argPos++
	}
	if week > 0 {
		whereParts = append(whereParts, fmt.Sprintf("EXTRACT(WEEK FROM w.start) = $%d", argPos))
		args = append(args, week)
		argPos++
	}

	whereClause := ""
	if len(whereParts) > 0 {
		whereClause = "WHERE " + strings.Join(whereParts, " AND ")
	}

	sqlstr := `
		SELECT
			w.id,
			w.employee,
			TRIM(CONCAT(COALESCE(e.fname, ''), ' ', COALESCE(e.lname, ''))) AS employee_name,
			w.hospital,
			COALESCE(f.f_name, '') AS facility_name,
			w.department,
			COALESCE(d.d_name, '') AS department_name,
			w.start,
			w.stop,
			w.created_on,
			CASE
				WHEN COALESCE(w.submit_status, '') = 'Submitted'
				THEN COALESCE(w.last_updated_on, w.created_on)
				ELSE NULL
			END AS submitted_on,
			COALESCE(w.submit_status, ''),
			COALESCE(w.report_status, ''),
			COALESCE(w.qn_01, 0) AS attendance,
			COALESCE(w.qn_03, 0) AS patients_reviewed,
			COALESCE(w.qn_05, 0) + COALESCE(w.qn_06, 0) AS procedures,
			EXISTS (
				SELECT 1
				FROM clinician_app.staffleave sl
				WHERE sl.employee_id = w.employee
				  AND COALESCE(sl.leave_status, '') IN ('Approved', 'Valid')
				  AND sl.start_date::date <= w.stop::date
				  AND sl.end_date::date >= w.start::date
			) AS is_on_leave
		FROM clinician_app.weeklyreport w
		JOIN clinician_app.employees e ON e.id = w.employee
		LEFT JOIN clinician_app.facilities f ON f.id = w.hospital
		LEFT JOIN clinician_app.departments d ON d.id = w.department
		` + whereClause + `
		ORDER BY w.start DESC, facility_name, department_name, employee_name
	`

	rows, err := db.QueryContext(ctx, sqlstr, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := []*ReportSubmissionListRow{}
	for rows.Next() {
		item := &ReportSubmissionListRow{}
		var submitStatusText string
		var reportStatusText string
		if err := rows.Scan(
			&item.ReportID,
			&item.EmployeeID,
			&item.EmployeeName,
			&item.FacilityID,
			&item.FacilityName,
			&item.DepartmentID,
			&item.DepartmentName,
			&item.WeekStart,
			&item.WeekStop,
			&item.EnteredOn,
			&item.SubmittedOn,
			&submitStatusText,
			&reportStatusText,
			&item.Attendance,
			&item.PatientsReviewed,
			&item.Procedures,
			&item.IsOnLeave,
		); err != nil {
			return nil, err
		}

		item.SubmitStatus = sql.NullString{String: submitStatusText, Valid: submitStatusText != ""}
		item.ReportStatus = sql.NullString{String: reportStatusText, Valid: reportStatusText != ""}
		if scopeEmployeeID > 0 {
			item.Actionable = submitStatusText != "Submitted" || reportStatusText == "Rejected" || reportStatusText == "Declined"
		} else {
			item.Actionable = submitStatusText == "Submitted" && !isFinalReportSubmissionStatus(reportStatusText)
		}
		items = append(items, item)
	}

	return items, rows.Err()
}

func isFinalReportSubmissionStatus(status string) bool {
	switch status {
	case "Approved", "Rejected", "Declined":
		return true
	default:
		return false
	}
}

func GetReportSubmissionDepartmentOptions(ctx context.Context, db *sql.DB, scopeFacilityID int64, filterFacilityID int) ([]DashboardFilterOption, error) {
	args := []interface{}{}
	whereParts := []string{}
	argPos := 1

	effectiveFacilityID := filterFacilityID
	if scopeFacilityID > 0 {
		effectiveFacilityID = int(scopeFacilityID)
	}
	if effectiveFacilityID > 0 {
		whereParts = append(whereParts, fmt.Sprintf("w.hospital = $%d", argPos))
		args = append(args, effectiveFacilityID)
		argPos++
	}

	whereClause := ""
	if len(whereParts) > 0 {
		whereClause = "WHERE " + strings.Join(whereParts, " AND ")
	}

	rows, err := db.QueryContext(ctx, `
		SELECT DISTINCT d.id, d.d_name
		FROM clinician_app.weeklyreport w
		JOIN clinician_app.departments d ON d.id = w.department
		`+whereClause+`
		ORDER BY d.d_name
	`, args...)
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

func GetReportSubmissionByIDForReview(ctx context.Context, db *sql.DB, reportID int, scopeFacilityID int64) (*ClinicianReportHistoryRow, error) {
	args := []interface{}{reportID}
	whereClause := `WHERE w.id = $1`

	if scopeFacilityID > 0 {
		whereClause += ` AND w.hospital = $2`
		args = append(args, scopeFacilityID)
	}

	sqlstr := `
		SELECT
			w.id,
			w.employee,
			w.department,
			w.hospital,
			w.start,
			w.stop,
			w.created_on,
			COALESCE(w.submit_status, ''),
			COALESCE(w.report_status, ''),
			w.qn_01, w.qn_02, w.qn_03, w.qn_04, w.qn_05, w.qn_06, w.qn_07, w.qn_08, w.qn_09, w.qn_10,
			w.qn_11, w.qn_12, w.qn_13, w.qn_14, w.qn_15, w.qn_16, w.qn_17, w.qn_18, w.qn_19, w.qn_20,
			w.qn_21, w.qn_22, w.qn_23, w.qn_24, w.qn_25, w.qn_26, w.qn_27, w.qn_28, w.qn_29, w.qn_30,
			w.qn_31, w.qn_32, w.qn_33, w.qn_34, w.qn_35, w.qn_36, w.qn_37, w.qn_38
		FROM clinician_app.weeklyreport w
		` + whereClause

	row := &ClinicianReportHistoryRow{}
	var submitStatusText string
	var reportStatusText string
	err := db.QueryRowContext(ctx, sqlstr, args...).Scan(
		&row.ReportID,
		&row.EmployeeID,
		&row.DepartmentID,
		&row.FacilityID,
		&row.WeekStart,
		&row.WeekStop,
		&row.EnteredOn,
		&submitStatusText,
		&reportStatusText,
		&row.Qn01, &row.Qn02, &row.Qn03, &row.Qn04, &row.Qn05, &row.Qn06, &row.Qn07, &row.Qn08, &row.Qn09, &row.Qn10,
		&row.Qn11, &row.Qn12, &row.Qn13, &row.Qn14, &row.Qn15, &row.Qn16, &row.Qn17, &row.Qn18, &row.Qn19, &row.Qn20,
		&row.Qn21, &row.Qn22, &row.Qn23, &row.Qn24, &row.Qn25, &row.Qn26, &row.Qn27, &row.Qn28, &row.Qn29, &row.Qn30,
		&row.Qn31, &row.Qn32, &row.Qn33, &row.Qn34, &row.Qn35, &row.Qn36, &row.Qn37, &row.Qn38,
	)
	if err != nil {
		return nil, err
	}

	if submitStatusText != "" {
		row.SubmitStatus = sql.NullString{String: submitStatusText, Valid: true}
	}
	if reportStatusText != "" {
		row.ReportStatus = sql.NullString{String: reportStatusText, Valid: true}
	}

	return row, nil
}

func SubmitClinicianReport(ctx context.Context, db *sql.DB, reportID int, employeeID int64) (bool, error) {
	const sqlstr = `
		UPDATE clinician_app.weeklyreport
		SET
			submit_status = 'Submitted',
			report_status = NULL,
			submitted_by = $3,
			approved_by = NULL,
			last_updated_on = $4
		WHERE id = $1
			AND employee = $2
			AND (
				COALESCE(submit_status, '') <> 'Submitted'
				OR COALESCE(report_status, '') IN ('Rejected', 'Declined')
			)
			AND COALESCE(report_status, '') <> 'Approved'
	`

	result, err := db.ExecContext(ctx, sqlstr, reportID, employeeID, employeeID, time.Now())
	if err != nil {
		return false, err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return false, err
	}

	return rowsAffected > 0, nil
}

func ApproveFacilityReportsByFilter(ctx context.Context, db *sql.DB, facilityID int64, departmentID int, year int, month int, week int, approverID int64) (int64, error) {
	args := []interface{}{facilityID}
	whereParts := []string{
		"hospital = $1",
		"COALESCE(submit_status, '') = 'Submitted'",
		"COALESCE(report_status, '') NOT IN ('Approved', 'Rejected', 'Declined')",
	}
	argPos := 2

	if departmentID > 0 {
		whereParts = append(whereParts, fmt.Sprintf("department = $%d", argPos))
		args = append(args, departmentID)
		argPos++
	}
	if year > 0 {
		whereParts = append(whereParts, fmt.Sprintf("EXTRACT(ISOYEAR FROM start) = $%d", argPos))
		args = append(args, year)
		argPos++
	}
	if month > 0 {
		whereParts = append(whereParts, fmt.Sprintf("EXTRACT(MONTH FROM start) = $%d", argPos))
		args = append(args, month)
		argPos++
	}
	if week > 0 {
		whereParts = append(whereParts, fmt.Sprintf("EXTRACT(WEEK FROM start) = $%d", argPos))
		args = append(args, week)
		argPos++
	}

	args = append(args, approverID)
	approverArg := argPos
	args = append(args, time.Now())
	timeArg := argPos + 1

	sqlstr := `
		UPDATE clinician_app.weeklyreport
		SET
			report_status = 'Approved',
			approved_by = $` + fmt.Sprintf("%d", approverArg) + `,
			last_updated_on = $` + fmt.Sprintf("%d", timeArg) + `
		WHERE ` + strings.Join(whereParts, " AND ")

	result, err := db.ExecContext(ctx, sqlstr, args...)
	if err != nil {
		return 0, err
	}

	return result.RowsAffected()
}

func DeclineFacilityReportsByFilter(ctx context.Context, db *sql.DB, facilityID int64, departmentID int, year int, month int, week int, approverID int64) (int64, error) {
	args := []interface{}{facilityID}
	whereParts := []string{
		"hospital = $1",
		"COALESCE(submit_status, '') = 'Submitted'",
		"COALESCE(report_status, '') NOT IN ('Approved', 'Rejected', 'Declined')",
	}
	argPos := 2

	if departmentID > 0 {
		whereParts = append(whereParts, fmt.Sprintf("department = $%d", argPos))
		args = append(args, departmentID)
		argPos++
	}
	if year > 0 {
		whereParts = append(whereParts, fmt.Sprintf("EXTRACT(ISOYEAR FROM start) = $%d", argPos))
		args = append(args, year)
		argPos++
	}
	if month > 0 {
		whereParts = append(whereParts, fmt.Sprintf("EXTRACT(MONTH FROM start) = $%d", argPos))
		args = append(args, month)
		argPos++
	}
	if week > 0 {
		whereParts = append(whereParts, fmt.Sprintf("EXTRACT(WEEK FROM start) = $%d", argPos))
		args = append(args, week)
		argPos++
	}

	args = append(args, approverID)
	approverArg := argPos
	args = append(args, time.Now())
	timeArg := argPos + 1

	sqlstr := `
		UPDATE clinician_app.weeklyreport
		SET
			report_status = 'Declined',
			approved_by = $` + fmt.Sprintf("%d", approverArg) + `,
			last_updated_on = $` + fmt.Sprintf("%d", timeArg) + `
		WHERE ` + strings.Join(whereParts, " AND ")

	result, err := db.ExecContext(ctx, sqlstr, args...)
	if err != nil {
		return 0, err
	}

	return result.RowsAffected()
}

func IsEmployeeOnLeaveForPeriod(ctx context.Context, db *sql.DB, employeeID int64, weekStart time.Time, weekStop time.Time) (bool, error) {
	var exists bool
	err := db.QueryRowContext(ctx, `
		SELECT EXISTS (
			SELECT 1
			FROM clinician_app.staffleave sl
			WHERE sl.employee_id = $1
			  AND COALESCE(sl.leave_status, '') IN ('Approved', 'Valid')
			  AND sl.start_date::date <= $3::date
			  AND sl.end_date::date >= $2::date
		)
	`, employeeID, weekStart, weekStop).Scan(&exists)
	return exists, err
}

func CountOnDutyEmployeesMissingWeeklyReport(ctx context.Context, db *sql.DB, facilityID int64, weekStart time.Time, weekStop time.Time, departmentID int) (int, error) {
	args := []interface{}{facilityID, weekStart, weekStop}
	departmentFilter := ""
	if departmentID > 0 {
		departmentFilter = " AND e.department = $4"
		args = append(args, departmentID)
	}

	query := `
		SELECT COUNT(*)
		FROM clinician_app.employees e
		WHERE e.facility = $1
		` + departmentFilter + `
		  AND NOT EXISTS (
			SELECT 1
			FROM clinician_app.staffleave sl
			WHERE sl.employee_id = e.id
			  AND COALESCE(sl.leave_status, '') IN ('Approved', 'Valid')
			  AND sl.start_date::date <= $3::date
			  AND sl.end_date::date >= $2::date
		  )
		  AND NOT EXISTS (
			SELECT 1
			FROM clinician_app.weeklyreport w
			WHERE w.employee = e.id
			  AND w.hospital = $1
			  AND w.start::date = $2::date
			  AND w.stop::date = $3::date
		  )
	`

	var count int
	err := db.QueryRowContext(ctx, query, args...).Scan(&count)
	return count, err
}

func EnsureOnLeaveZeroReports(ctx context.Context, db *sql.DB, facilityID int64, weekStart time.Time, weekStop time.Time, submittedBy int64, departmentID int) (int64, error) {
	args := []interface{}{facilityID, weekStart, weekStop, submittedBy, time.Now()}
	departmentFilter := ""
	if departmentID > 0 {
		departmentFilter = " AND e.department = $6"
		args = append(args, departmentID)
	}

	query := `
		INSERT INTO clinician_app.weeklyreport (
			hospital, employee, department, start, stop,
			qn_01, qn_02, qn_03, qn_04, qn_05, qn_06, qn_07, qn_08, qn_09, qn_10,
			qn_11, qn_12, qn_13, qn_14, qn_15, qn_16, qn_17, qn_18, qn_19, qn_20,
			qn_21, qn_22, qn_23, qn_24, qn_25, qn_26, qn_27, qn_28, qn_29, qn_30,
			qn_31, qn_32, qn_33, qn_34, qn_35, qn_36, qn_37, qn_38,
			submit_status, report_status, submitted_by, created_on, last_updated_on
		)
		SELECT
			e.facility,
			e.id,
			e.department,
			$2,
			$3,
			0,0,0,0,0,0,0,0,0,0,
			0,0,0,0,0,0,0,0,0,0,
			0,0,0,0,0,0,0,0,0,0,
			0,0,0,0,0,0,0,0,
			'Submitted', NULL, $4, $5, $5
		FROM clinician_app.employees e
		WHERE e.facility = $1
		` + departmentFilter + `
		  AND EXISTS (
			SELECT 1
			FROM clinician_app.staffleave sl
			WHERE sl.employee_id = e.id
			  AND COALESCE(sl.leave_status, '') IN ('Approved', 'Valid')
			  AND sl.start_date::date <= $3::date
			  AND sl.end_date::date >= $2::date
		  )
		  AND NOT EXISTS (
			SELECT 1
			FROM clinician_app.weeklyreport w
			WHERE w.employee = e.id
			  AND w.hospital = e.facility
			  AND w.start::date = $2::date
			  AND w.stop::date = $3::date
		  )
	`

	result, err := db.ExecContext(ctx, query, args...)
	if err != nil {
		return 0, err
	}

	return result.RowsAffected()
}

func SubmitFacilityReportsByFilter(ctx context.Context, db *sql.DB, facilityID int64, departmentID int, year int, month int, week int, submittedBy int64) (int64, error) {
	args := []interface{}{facilityID}
	whereParts := []string{
		"hospital = $1",
		"(COALESCE(submit_status, '') <> 'Submitted' OR COALESCE(report_status, '') IN ('Rejected', 'Declined'))",
		"COALESCE(report_status, '') <> 'Approved'",
	}
	argPos := 2

	if departmentID > 0 {
		whereParts = append(whereParts, fmt.Sprintf("department = $%d", argPos))
		args = append(args, departmentID)
		argPos++
	}
	if year > 0 {
		whereParts = append(whereParts, fmt.Sprintf("EXTRACT(ISOYEAR FROM start) = $%d", argPos))
		args = append(args, year)
		argPos++
	}
	if month > 0 {
		whereParts = append(whereParts, fmt.Sprintf("EXTRACT(MONTH FROM start) = $%d", argPos))
		args = append(args, month)
		argPos++
	}
	if week > 0 {
		whereParts = append(whereParts, fmt.Sprintf("EXTRACT(WEEK FROM start) = $%d", argPos))
		args = append(args, week)
		argPos++
	}

	args = append(args, submittedBy)
	submittedByArg := argPos
	args = append(args, time.Now())
	nowArg := argPos + 1

	query := `
		UPDATE clinician_app.weeklyreport
		SET
			submit_status = 'Submitted',
			report_status = NULL,
			submitted_by = $` + fmt.Sprintf("%d", submittedByArg) + `,
			approved_by = NULL,
			last_updated_on = $` + fmt.Sprintf("%d", nowArg) + `
		WHERE ` + strings.Join(whereParts, " AND ")

	result, err := db.ExecContext(ctx, query, args...)
	if err != nil {
		return 0, err
	}

	return result.RowsAffected()
}

func GetFacilitySubmissionSummaries(ctx context.Context, db *sql.DB, facilityID int, selectedStatus string, year int, month int, week int) ([]*FacilitySubmissionSummaryRow, error) {
	whereParts := []string{}
	args := []interface{}{}
	argPos := 1

	if facilityID > 0 {
		whereParts = append(whereParts, fmt.Sprintf("e.facility = $%d", argPos))
		args = append(args, facilityID)
		argPos++
	}
	if year > 0 {
		whereParts = append(whereParts, fmt.Sprintf("EXTRACT(ISOYEAR FROM w.start) = $%d", argPos))
		args = append(args, year)
		argPos++
	}
	if month > 0 {
		whereParts = append(whereParts, fmt.Sprintf("EXTRACT(MONTH FROM w.start) = $%d", argPos))
		args = append(args, month)
		argPos++
	}
	if week > 0 {
		whereParts = append(whereParts, fmt.Sprintf("EXTRACT(WEEK FROM w.start) = $%d", argPos))
		args = append(args, week)
		argPos++
	}

	whereClause := ""
	if len(whereParts) > 0 {
		whereClause = "WHERE " + strings.Join(whereParts, " AND ")
	}

	query := `
		WITH facility_weeks AS (
			SELECT DISTINCT e.facility AS facility_id, w.start::date AS week_start, w.stop::date AS week_stop
			FROM clinician_app.weeklyreport w
			JOIN clinician_app.employees e ON e.id = w.employee
			` + whereClause + `
		), on_leave AS (
			SELECT fw.facility_id, fw.week_start, COUNT(DISTINCT e.id) AS on_leave_count
			FROM facility_weeks fw
			JOIN clinician_app.employees e ON e.facility = fw.facility_id
			JOIN clinician_app.staffleave sl ON sl.employee_id = e.id
			WHERE COALESCE(sl.leave_status, '') IN ('Approved', 'Valid')
			  AND sl.start_date::date <= fw.week_stop
			  AND sl.end_date::date >= fw.week_start
			GROUP BY fw.facility_id, fw.week_start
		), on_duty AS (
			SELECT fw.facility_id, fw.week_start, COUNT(DISTINCT e.id) AS on_duty_count
			FROM facility_weeks fw
			JOIN clinician_app.employees e ON e.facility = fw.facility_id
			LEFT JOIN clinician_app.staffleave sl
			  ON sl.employee_id = e.id
			 AND COALESCE(sl.leave_status, '') IN ('Approved', 'Valid')
			 AND sl.start_date::date <= fw.week_stop
			 AND sl.end_date::date >= fw.week_start
			WHERE sl.employee_id IS NULL
			GROUP BY fw.facility_id, fw.week_start
		), submissions AS (
			SELECT
				w.hospital AS facility_id,
				w.start::date AS week_start,
				COUNT(*) FILTER (WHERE COALESCE(w.submit_status, '') = 'Submitted') AS submitted_count,
				COUNT(*) FILTER (WHERE COALESCE(w.report_status, '') = 'Approved') AS approved_count,
				COUNT(*) FILTER (WHERE COALESCE(w.report_status, '') IN ('Rejected', 'Declined')) AS declined_count,
				COUNT(*) FILTER (
					WHERE COALESCE(w.submit_status, '') = 'Submitted'
					  AND COALESCE(w.report_status, '') NOT IN ('Approved', 'Rejected', 'Declined')
				) AS pending_count,
				MAX(CASE WHEN COALESCE(w.submit_status, '') = 'Submitted' THEN COALESCE(w.last_updated_on, w.created_on) END) AS submitted_on
			FROM clinician_app.weeklyreport w
			GROUP BY w.hospital, w.start
		)
		SELECT
			fw.facility_id,
			COALESCE(f.f_name, '') AS facility_name,
			COALESCE(od.on_duty_count, 0) AS on_duty_count,
			COALESCE(ol.on_leave_count, 0) AS on_leave_count,
			COALESCE(s.submitted_count, 0) AS submitted_count,
			COALESCE(s.approved_count, 0) AS approved_count,
			COALESCE(s.declined_count, 0) AS declined_count,
			COALESCE(s.pending_count, 0) AS pending_count,
			GREATEST(COALESCE(od.on_duty_count, 0) + COALESCE(ol.on_leave_count, 0) - COALESCE(s.submitted_count, 0), 0) AS missing_count,
			s.submitted_on
		FROM facility_weeks fw
		LEFT JOIN clinician_app.facilities f ON f.id = fw.facility_id
		LEFT JOIN on_duty od ON od.facility_id = fw.facility_id AND od.week_start = fw.week_start
		LEFT JOIN on_leave ol ON ol.facility_id = fw.facility_id AND ol.week_start = fw.week_start
		LEFT JOIN submissions s ON s.facility_id = fw.facility_id AND s.week_start = fw.week_start
		ORDER BY fw.week_start DESC, facility_name
	`

	rows, err := db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := []*FacilitySubmissionSummaryRow{}
	for rows.Next() {
		item := &FacilitySubmissionSummaryRow{}
		if err := rows.Scan(
			&item.FacilityID,
			&item.FacilityName,
			&item.OnDutyCount,
			&item.OnLeaveCount,
			&item.SubmittedCount,
			&item.ApprovedCount,
			&item.DeclinedCount,
			&item.PendingCount,
			&item.MissingCount,
			&item.SubmittedOn,
		); err != nil {
			return nil, err
		}

		switch {
		case item.PendingCount > 0:
			item.ApprovalStatus = "Pending"
		case item.DeclinedCount > 0:
			item.ApprovalStatus = "Declined"
		case item.ApprovedCount > 0:
			item.ApprovalStatus = "Approved"
		default:
			item.ApprovalStatus = "Pending"
		}

		if item.MissingCount > 0 || item.SubmittedCount == 0 {
			item.SubmissionState = "Incomplete"
		} else {
			item.SubmissionState = "Complete"
		}

		item.CanApprove = item.PendingCount > 0

		if selectedStatus != "all" {
			switch selectedStatus {
			case "pending":
				if item.ApprovalStatus != "Pending" {
					continue
				}
			case "approved":
				if item.ApprovalStatus != "Approved" {
					continue
				}
			case "declined":
				if item.ApprovalStatus != "Declined" {
					continue
				}
			case "draft":
				if item.SubmissionState != "Incomplete" {
					continue
				}
			}
		}

		items = append(items, item)
	}

	return items, rows.Err()
}
