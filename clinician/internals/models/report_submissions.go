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
			COALESCE(w.qn_05, 0) + COALESCE(w.qn_06, 0) AS procedures
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
