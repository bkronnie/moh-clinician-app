package models

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

type ClinicianReportHistoryRow struct {
	ReportID         int
	EmployeeID       int64
	DepartmentID     int64
	FacilityID       int64
	WeekStart        sql.NullTime
	WeekStop         sql.NullTime
	EnteredOn        sql.NullTime
	SubmitStatus     sql.NullString
	ReportStatus     sql.NullString
	HistoryStatus    string
	Actionable       bool
	Attendance       int
	PatientsReviewed int
	Procedures       int
	Qn01             sql.NullInt64
	Qn02             sql.NullInt64
	Qn03             sql.NullInt64
	Qn04             sql.NullInt64
	Qn05             sql.NullInt64
	Qn06             sql.NullInt64
	Qn07             sql.NullInt64
	Qn08             sql.NullInt64
	Qn09             sql.NullInt64
	Qn10             sql.NullInt64
	Qn11             sql.NullInt64
	Qn12             sql.NullInt64
	Qn13             sql.NullInt64
	Qn14             sql.NullInt64
	Qn15             sql.NullInt64
	Qn16             sql.NullInt64
	Qn17             sql.NullInt64
	Qn18             sql.NullInt64
	Qn19             sql.NullInt64
	Qn20             sql.NullInt64
	Qn21             sql.NullInt64
	Qn22             sql.NullInt64
	Qn23             sql.NullInt64
	Qn24             sql.NullInt64
	Qn25             sql.NullInt64
	Qn26             sql.NullInt64
	Qn27             sql.NullInt64
	Qn28             sql.NullInt64
	Qn29             sql.NullInt64
	Qn30             sql.NullInt64
	Qn31             sql.NullInt64
	Qn32             sql.NullInt64
	Qn33             sql.NullInt64
	Qn34             sql.NullInt64
	Qn35             sql.NullInt64
	Qn36             sql.NullInt64
	Qn37             sql.NullInt64
	Qn38             sql.NullInt64
}

type ClinicianReportHistorySummary struct {
	TotalReports     int
	SubmittedReports int
	ApprovedReports  int
	DeclinedReports  int
	DraftReports     int
}

type FacilityReportReviewRow struct {
	ReportID         int
	EmployeeID       int64
	EmployeeName     string
	DepartmentName   string
	WeekStart        sql.NullTime
	WeekStop         sql.NullTime
	SubmittedOn      sql.NullTime
	SubmitStatus     sql.NullString
	ReportStatus     sql.NullString
	PatientsReviewed int
	Procedures       int
	Actionable       bool
}

type FacilityReportReviewSummary struct {
	PendingReports    int
	ApprovedReports   int
	DeclinedReports   int
	SubmittedThisWeek int
}

func GetClinicianReportHistory(ctx context.Context, db *sql.DB, employeeID int64, filterStatus string, year int, week int) ([]*ClinicianReportHistoryRow, error) {
	whereClause := `WHERE w.employee = $1`
	args := []interface{}{employeeID}
	argPos := 2

	if year > 0 {
		whereClause += fmt.Sprintf(" AND EXTRACT(ISOYEAR FROM w.start) = $%d", argPos)
		args = append(args, year)
		argPos++
	}

	if week > 0 {
		whereClause += fmt.Sprintf(" AND EXTRACT(WEEK FROM w.start) = $%d", argPos)
		args = append(args, week)
		argPos++
	}

	sqlstr := `
		WITH report_groups AS (
			SELECT
				MAX(w.id) AS report_id,
				MAX(w.employee) AS employee_id,
				MAX(w.department) AS department_id,
				MAX(w.hospital) AS facility_id,
				MIN(w.start) AS week_start,
				MAX(w.stop) AS week_stop,
				MAX(w.created_on) AS entered_on,
				MAX(COALESCE(w.submit_status, '')) AS submit_status_text,
				MAX(COALESCE(w.report_status, '')) AS report_status_text,
				COALESCE(SUM(w.qn_01), 0) AS attendance,
				COALESCE(SUM(w.qn_03), 0) AS patients_reviewed,
				COALESCE(SUM(w.qn_05), 0) + COALESCE(SUM(w.qn_06), 0) AS procedures
			FROM clinician_app.weeklyreport w
			` + whereClause + `
			GROUP BY w.start
		)
		SELECT
			report_id,
			employee_id,
			department_id,
			facility_id,
			week_start,
			week_stop,
			entered_on,
			submit_status_text,
			report_status_text,
			CASE
				WHEN report_status_text = 'Approved' THEN 'approved'
				WHEN report_status_text IN ('Rejected', 'Declined') THEN 'declined'
				WHEN submit_status_text = 'Submitted' THEN 'submitted'
				ELSE 'draft'
			END AS history_status,
			attendance,
			patients_reviewed,
			procedures
		FROM report_groups
	`

	switch filterStatus {
	case "submitted":
		sqlstr += ` WHERE submit_status_text = 'Submitted' AND COALESCE(report_status_text, '') NOT IN ('Approved', 'Rejected', 'Declined')`
	case "approved":
		sqlstr += ` WHERE report_status_text = 'Approved'`
	case "declined":
		sqlstr += ` WHERE report_status_text IN ('Rejected', 'Declined')`
	case "draft":
		sqlstr += ` WHERE COALESCE(submit_status_text, '') <> 'Submitted' AND COALESCE(report_status_text, '') NOT IN ('Approved', 'Rejected', 'Declined')`
	}

	sqlstr += ` ORDER BY week_start DESC`

	rows, err := db.QueryContext(ctx, sqlstr, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	history := []*ClinicianReportHistoryRow{}
	for rows.Next() {
		row := &ClinicianReportHistoryRow{}
		var submitStatusText string
		var reportStatusText string
		if err := rows.Scan(
			&row.ReportID,
			&row.EmployeeID,
			&row.DepartmentID,
			&row.FacilityID,
			&row.WeekStart,
			&row.WeekStop,
			&row.EnteredOn,
			&submitStatusText,
			&reportStatusText,
			&row.HistoryStatus,
			&row.Attendance,
			&row.PatientsReviewed,
			&row.Procedures,
		); err != nil {
			return nil, err
		}

		if submitStatusText != "" {
			row.SubmitStatus = sql.NullString{String: submitStatusText, Valid: true}
		}
		if reportStatusText != "" {
			row.ReportStatus = sql.NullString{String: reportStatusText, Valid: true}
		}
		row.Actionable = row.HistoryStatus == "draft" || row.HistoryStatus == "declined"
		history = append(history, row)
	}

	return history, rows.Err()
}

func GetClinicianReportHistorySummary(ctx context.Context, db *sql.DB, employeeID int64, year int, week int) (ClinicianReportHistorySummary, error) {
	var summary ClinicianReportHistorySummary

	whereClause := `WHERE w.employee = $1`
	args := []interface{}{employeeID}
	argPos := 2

	if year > 0 {
		whereClause += fmt.Sprintf(" AND EXTRACT(ISOYEAR FROM w.start) = $%d", argPos)
		args = append(args, year)
		argPos++
	}

	if week > 0 {
		whereClause += fmt.Sprintf(" AND EXTRACT(WEEK FROM w.start) = $%d", argPos)
		args = append(args, week)
		argPos++
	}

	sqlstr := `
		WITH report_groups AS (
			SELECT
				MIN(w.start) AS week_start,
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
			COUNT(*) FILTER (WHERE report_status_text IN ('Rejected', 'Declined')) AS declined_reports,
			COUNT(*) FILTER (
				WHERE COALESCE(submit_status_text, '') <> 'Submitted'
				AND COALESCE(report_status_text, '') NOT IN ('Approved', 'Rejected', 'Declined')
			) AS draft_reports
		FROM report_groups
	`

	err := db.QueryRowContext(ctx, sqlstr, args...).Scan(
		&summary.TotalReports,
		&summary.SubmittedReports,
		&summary.ApprovedReports,
		&summary.DeclinedReports,
		&summary.DraftReports,
	)

	return summary, err
}

func ClinicianEditableReportByID(ctx context.Context, db *sql.DB, reportID int, employeeID int64) (*ClinicianReportHistoryRow, error) {
	const sqlstr = `
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
		WHERE w.id = $1
			AND w.employee = $2
			AND (
				COALESCE(w.submit_status, '') <> 'Submitted'
				OR COALESCE(w.report_status, '') IN ('Rejected', 'Declined')
			)
			AND COALESCE(w.report_status, '') <> 'Approved'
	`

	row := &ClinicianReportHistoryRow{}
	var submitStatusText string
	var reportStatusText string
	err := db.QueryRowContext(ctx, sqlstr, reportID, employeeID).Scan(
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
	if row.ReportStatus.Valid && (row.ReportStatus.String == "Rejected" || row.ReportStatus.String == "Declined") {
		row.HistoryStatus = "declined"
	} else if row.SubmitStatus.Valid && row.SubmitStatus.String == "Submitted" {
		row.HistoryStatus = "submitted"
	} else {
		row.HistoryStatus = "draft"
	}
	row.Actionable = row.HistoryStatus == "draft" || row.HistoryStatus == "declined"

	return row, nil
}

func UpdateClinicianReport(ctx context.Context, db *sql.DB, reportID int, employeeID int64, start, stop time.Time, values map[string]int64) (bool, error) {
	const sqlstr = `
		UPDATE clinician_app.weeklyreport SET
			start = $1,
			stop = $2,
			qn_01 = $3, qn_02 = $4, qn_03 = $5, qn_04 = $6, qn_05 = $7, qn_06 = $8, qn_07 = $9, qn_08 = $10, qn_09 = $11, qn_10 = $12,
			qn_11 = $13, qn_12 = $14, qn_13 = $15, qn_14 = $16, qn_15 = $17, qn_16 = $18, qn_17 = $19, qn_18 = $20, qn_19 = $21, qn_20 = $22,
			qn_21 = $23, qn_22 = $24, qn_23 = $25, qn_24 = $26, qn_25 = $27, qn_26 = $28, qn_27 = $29, qn_28 = $30, qn_29 = $31, qn_30 = $32,
			qn_31 = $33, qn_32 = $34, qn_33 = $35, qn_34 = $36, qn_35 = $37, qn_36 = $38, qn_37 = $39, qn_38 = $40,
			last_updated_on = $41
		WHERE id = $42
			AND employee = $43
			AND (
				COALESCE(submit_status, '') <> 'Submitted'
				OR COALESCE(report_status, '') IN ('Rejected', 'Declined')
			)
			AND COALESCE(report_status, '') <> 'Approved'
	`

	result, err := db.ExecContext(
		ctx,
		sqlstr,
		start,
		stop,
		values["attendance"], values["ward_rounds"], values["patients_reviewed"], values["theatre_days"], values["elective"], values["emergency"], values["postmortems"], values["OPD_clinics"], values["OPD_patients"], values["anc_patients"],
		values["teaching_rounds"], values["students_taught"], values["mortality_reviews"], values["maternal"], values["perinatal"], values["surgical"], values["medical"], values["paed"], values["labs_requests"], values["imaging_requests"],
		values["lab_investigations"], values["BS"], values["HIV"], values["malaria"], values["TB"], values["CBC"], values["chemistry"], values["hematology"], values["urinalysis"], values["gram_stain"],
		values["culture"], values["microbiology"], values["sensitivity_tests"], values["diagnostics"], values["xrays"], values["ct_scans"], values["obstetrics_scans"], values["abdominal_scans"],
		time.Now(),
		reportID,
		employeeID,
	)
	if err != nil {
		return false, err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return false, err
	}

	return rowsAffected > 0, nil
}

func DeleteClinicianReport(ctx context.Context, db *sql.DB, reportID int, employeeID int64) (bool, error) {
	const sqlstr = `
		DELETE FROM clinician_app.weeklyreport
		WHERE id = $1
			AND employee = $2
			AND (
				COALESCE(submit_status, '') <> 'Submitted'
				OR COALESCE(report_status, '') IN ('Rejected', 'Declined')
			)
			AND COALESCE(report_status, '') <> 'Approved'
	`

	result, err := db.ExecContext(ctx, sqlstr, reportID, employeeID)
	if err != nil {
		return false, err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return false, err
	}

	return rowsAffected > 0, nil
}

func GetFacilityReportReview(ctx context.Context, db *sql.DB, facilityID int64, filterStatus string, year int, week int) ([]*FacilityReportReviewRow, error) {
	whereClause := `WHERE w.hospital = $1 AND COALESCE(w.submit_status, '') = 'Submitted'`
	args := []interface{}{facilityID}
	argPos := 2

	if year > 0 {
		whereClause += fmt.Sprintf(" AND EXTRACT(ISOYEAR FROM w.start) = $%d", argPos)
		args = append(args, year)
		argPos++
	}

	if week > 0 {
		whereClause += fmt.Sprintf(" AND EXTRACT(WEEK FROM w.start) = $%d", argPos)
		args = append(args, week)
		argPos++
	}

	switch filterStatus {
	case "pending":
		whereClause += ` AND COALESCE(w.report_status, '') NOT IN ('Approved', 'Rejected', 'Declined')`
	case "approved":
		whereClause += ` AND COALESCE(w.report_status, '') = 'Approved'`
	case "declined":
		whereClause += ` AND COALESCE(w.report_status, '') IN ('Rejected', 'Declined')`
	default:
		filterStatus = "all"
	}

	sqlstr := `
		SELECT
			w.id,
			w.employee,
			TRIM(CONCAT(COALESCE(e.fname, ''), ' ', COALESCE(e.lname, ''))) AS employee_name,
			COALESCE(d.d_name, '') AS department_name,
			w.start,
			w.stop,
			w.created_on,
			COALESCE(w.submit_status, ''),
			COALESCE(w.report_status, ''),
			COALESCE(w.qn_03, 0) AS patients_reviewed,
			COALESCE(w.qn_05, 0) + COALESCE(w.qn_06, 0) AS procedures
		FROM clinician_app.weeklyreport w
		JOIN clinician_app.employees e ON e.id = w.employee
		LEFT JOIN clinician_app.departments d ON d.id = w.department
		` + whereClause + `
		ORDER BY w.start DESC, employee_name
	`

	rows, err := db.QueryContext(ctx, sqlstr, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := []*FacilityReportReviewRow{}
	for rows.Next() {
		item := &FacilityReportReviewRow{}
		var submitStatusText string
		var reportStatusText string
		if err := rows.Scan(
			&item.ReportID,
			&item.EmployeeID,
			&item.EmployeeName,
			&item.DepartmentName,
			&item.WeekStart,
			&item.WeekStop,
			&item.SubmittedOn,
			&submitStatusText,
			&reportStatusText,
			&item.PatientsReviewed,
			&item.Procedures,
		); err != nil {
			return nil, err
		}

		item.SubmitStatus = sql.NullString{String: submitStatusText, Valid: submitStatusText != ""}
		item.ReportStatus = sql.NullString{String: reportStatusText, Valid: reportStatusText != ""}
		item.Actionable = reportStatusText == ""
		items = append(items, item)
	}

	return items, rows.Err()
}

func ApproveFacilityReport(ctx context.Context, db *sql.DB, reportID int, facilityID int64, approverID int64) (bool, error) {
	const sqlstr = `
		UPDATE clinician_app.weeklyreport
		SET report_status = 'Approved', approved_by = $3
		WHERE id = $1
			AND hospital = $2
			AND COALESCE(submit_status, '') = 'Submitted'
			AND COALESCE(report_status, '') NOT IN ('Approved', 'Rejected', 'Declined')
	`

	result, err := db.ExecContext(ctx, sqlstr, reportID, facilityID, approverID)
	if err != nil {
		return false, err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return false, err
	}

	return rowsAffected > 0, nil
}

func DeclineFacilityReport(ctx context.Context, db *sql.DB, reportID int, facilityID int64, approverID int64) (bool, error) {
	const sqlstr = `
		UPDATE clinician_app.weeklyreport
		SET report_status = 'Declined', approved_by = $3
		WHERE id = $1
			AND hospital = $2
			AND COALESCE(submit_status, '') = 'Submitted'
			AND COALESCE(report_status, '') NOT IN ('Approved', 'Rejected', 'Declined')
	`

	result, err := db.ExecContext(ctx, sqlstr, reportID, facilityID, approverID)
	if err != nil {
		return false, err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return false, err
	}

	return rowsAffected > 0, nil
}

func GetFacilityReportReviewSummary(ctx context.Context, db *sql.DB, facilityID int64) (FacilityReportReviewSummary, error) {
	var summary FacilityReportReviewSummary

	const sqlstr = `
		SELECT
			COUNT(*) FILTER (
				WHERE COALESCE(submit_status, '') = 'Submitted'
				AND COALESCE(report_status, '') NOT IN ('Approved', 'Rejected', 'Declined')
			) AS pending_reports,
			COUNT(*) FILTER (
				WHERE COALESCE(submit_status, '') = 'Submitted'
				AND COALESCE(report_status, '') = 'Approved'
			) AS approved_reports,
			COUNT(*) FILTER (
				WHERE COALESCE(submit_status, '') = 'Submitted'
				AND COALESCE(report_status, '') IN ('Rejected', 'Declined')
			) AS declined_reports,
			COUNT(*) FILTER (
				WHERE COALESCE(submit_status, '') = 'Submitted'
				AND start = date_trunc('week', CURRENT_DATE)::date
			) AS submitted_this_week
		FROM clinician_app.weeklyreport
		WHERE hospital = $1
	`

	err := db.QueryRowContext(ctx, sqlstr, facilityID).Scan(
		&summary.PendingReports,
		&summary.ApprovedReports,
		&summary.DeclinedReports,
		&summary.SubmittedThisWeek,
	)

	return summary, err
}
