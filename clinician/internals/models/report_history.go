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
	DaysWorked       sql.NullString
	SubmittedOn      sql.NullTime
	Qn01             sql.NullInt64
	Qn02             sql.NullInt64
	Qn03             sql.NullInt64
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
				COALESCE(SUM(w.attendance), 0) AS attendance,
				COALESCE(SUM(w.patients_reviewed), 0) AS patients_reviewed,
				COALESCE(SUM(w.elective), 0) + COALESCE(SUM(w.emergency), 0) AS procedures
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

func GetClinicianReportHistorySummary(ctx context.Context, db *sql.DB, employeeID int64, year int, month int, week int) (ClinicianReportHistorySummary, error) {
	var summary ClinicianReportHistorySummary

	whereClause := `WHERE w.employee = $1`
	args := []interface{}{employeeID}
	argPos := 2

	if year > 0 {
		whereClause += fmt.Sprintf(" AND EXTRACT(ISOYEAR FROM w.start) = $%d", argPos)
		args = append(args, year)
		argPos++
	}

	if month > 0 {
		whereClause += fmt.Sprintf(" AND EXTRACT(MONTH FROM w.start) = $%d", argPos)
		args = append(args, month)
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
			w.attendance, w.ward_rounds, w.patients_reviewed, w.elective, w.emergency, w.postmortems, w.opd_clinics, w.opd_patients, w.anc_patients,
			w.teaching_rounds, w.students_taught, w.mortality_reviews, w.maternal, w.perinatal, w.surgical, w.medical, w.paed, w.labs_requests, w.imaging_requests,
			w.lab_investigations, w.bs, w.hiv, w.malaria, w.tb, w.cbc, w.chemistry, w.hematology, w.urinalysis, w.gram_stain,
			w.culture, w.microbiology, w.sensitivity_tests, w.diagnostics, w.xrays, w.ct_scans, w.obstetrics_scans, w.abdominal_scans,
			COALESCE(w.days_worked, ''),
			w.submitted_on
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
	var daysWorkedText string
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
		&row.Qn01, &row.Qn02, &row.Qn03, &row.Qn05, &row.Qn06, &row.Qn07, &row.Qn08, &row.Qn09, &row.Qn10,
		&row.Qn11, &row.Qn12, &row.Qn13, &row.Qn14, &row.Qn15, &row.Qn16, &row.Qn17, &row.Qn18, &row.Qn19, &row.Qn20,
		&row.Qn21, &row.Qn22, &row.Qn23, &row.Qn24, &row.Qn25, &row.Qn26, &row.Qn27, &row.Qn28, &row.Qn29, &row.Qn30,
		&row.Qn31, &row.Qn32, &row.Qn33, &row.Qn34, &row.Qn35, &row.Qn36, &row.Qn37, &row.Qn38,
		&daysWorkedText,
		&row.SubmittedOn,
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
	if daysWorkedText != "" {
		row.DaysWorked = sql.NullString{String: daysWorkedText, Valid: true}
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

func LatestClinicianReportByPeriod(ctx context.Context, db *sql.DB, employeeID int64, weekStart time.Time, weekStop time.Time) (*ClinicianReportHistoryRow, error) {
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
			w.attendance, w.ward_rounds, w.patients_reviewed, w.elective, w.emergency, w.postmortems, w.opd_clinics, w.opd_patients, w.anc_patients,
			w.teaching_rounds, w.students_taught, w.mortality_reviews, w.maternal, w.perinatal, w.surgical, w.medical, w.paed, w.labs_requests, w.imaging_requests,
			w.lab_investigations, w.bs, w.hiv, w.malaria, w.tb, w.cbc, w.chemistry, w.hematology, w.urinalysis, w.gram_stain,
			w.culture, w.microbiology, w.sensitivity_tests, w.diagnostics, w.xrays, w.ct_scans, w.obstetrics_scans, w.abdominal_scans,
			COALESCE(w.days_worked, ''),
			w.submitted_on
		FROM clinician_app.weeklyreport w
		WHERE w.employee = $1
			AND (
				w.start::date = $2::date
				OR (
					w.start IS NOT NULL
					AND w.stop IS NOT NULL
					AND daterange(w.start, w.stop, '[]') && daterange($2::date, $3::date, '[]')
				)
			)
		ORDER BY CASE WHEN w.start::date = $2::date THEN 0 ELSE 1 END,
			CASE WHEN w.stop::date = $3::date THEN 0 ELSE 1 END,
			w.id DESC
		LIMIT 1
	`

	row := &ClinicianReportHistoryRow{}
	var submitStatusText string
	var reportStatusText string
	var daysWorkedText string
	err := db.QueryRowContext(ctx, sqlstr, employeeID, weekStart, weekStop).Scan(
		&row.ReportID,
		&row.EmployeeID,
		&row.DepartmentID,
		&row.FacilityID,
		&row.WeekStart,
		&row.WeekStop,
		&row.EnteredOn,
		&submitStatusText,
		&reportStatusText,
		&row.Qn01, &row.Qn02, &row.Qn03, &row.Qn05, &row.Qn06, &row.Qn07, &row.Qn08, &row.Qn09, &row.Qn10,
		&row.Qn11, &row.Qn12, &row.Qn13, &row.Qn14, &row.Qn15, &row.Qn16, &row.Qn17, &row.Qn18, &row.Qn19, &row.Qn20,
		&row.Qn21, &row.Qn22, &row.Qn23, &row.Qn24, &row.Qn25, &row.Qn26, &row.Qn27, &row.Qn28, &row.Qn29, &row.Qn30,
		&row.Qn31, &row.Qn32, &row.Qn33, &row.Qn34, &row.Qn35, &row.Qn36, &row.Qn37, &row.Qn38,
		&daysWorkedText,
		&row.SubmittedOn,
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
	if daysWorkedText != "" {
		row.DaysWorked = sql.NullString{String: daysWorkedText, Valid: true}
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

func UpdateClinicianReport(ctx context.Context, db *sql.DB, reportID int, employeeID int64, start, stop time.Time, values map[string]sql.NullInt64, daysWorked string) (bool, error) {
	const sqlstr = `
		UPDATE clinician_app.weeklyreport SET
			start = $1,
			stop = $2,
			attendance = $3, ward_rounds = $4, patients_reviewed = $5, elective = $6, emergency = $7, postmortems = $8, opd_clinics = $9, opd_patients = $10, anc_patients = $11,
			teaching_rounds = $12, students_taught = $13, mortality_reviews = $14, maternal = $15, perinatal = $16, surgical = $17, medical = $18, paed = $19, labs_requests = $20, imaging_requests = $21,
			lab_investigations = $22, bs = $23, hiv = $24, malaria = $25, tb = $26, cbc = $27, chemistry = $28, hematology = $29, urinalysis = $30, gram_stain = $31,
			culture = $32, microbiology = $33, sensitivity_tests = $34, diagnostics = $35, xrays = $36, ct_scans = $37, obstetrics_scans = $38, abdominal_scans = $39,
			days_worked = $40,
			last_updated_on = $41
		WHERE id = $42
			AND employee = $43
			AND (
				COALESCE(submit_status, '') <> 'Submitted'
				OR COALESCE(report_status, '') IN ('Rejected', 'Declined')
			)
			AND COALESCE(report_status, '') <> 'Approved'
	`

	nullableValue := func(key string) interface{} {
		value, ok := values[key]
		if !ok || !value.Valid {
			return nil
		}
		return value.Int64
	}

	result, err := db.ExecContext(
		ctx,
		sqlstr,
		start,
		stop,
		nullableValue("attendance"), nullableValue("ward_rounds"), nullableValue("patients_reviewed"), nullableValue("elective"), nullableValue("emergency"), nullableValue("postmortems"), nullableValue("OPD_clinics"), nullableValue("OPD_patients"), nullableValue("anc_patients"),
		nullableValue("teaching_rounds"), nullableValue("students_taught"), nullableValue("mortality_reviews"), nullableValue("maternal"), nullableValue("perinatal"), nullableValue("surgical"), nullableValue("medical"), nullableValue("paed"), nullableValue("labs_requests"), nullableValue("imaging_requests"),
		nullableValue("lab_investigations"), nullableValue("BS"), nullableValue("HIV"), nullableValue("malaria"), nullableValue("TB"), nullableValue("CBC"), nullableValue("chemistry"), nullableValue("hematology"), nullableValue("urinalysis"), nullableValue("gram_stain"),
		nullableValue("culture"), nullableValue("microbiology"), nullableValue("sensitivity_tests"), nullableValue("diagnostics"), nullableValue("xrays"), nullableValue("ct_scans"), nullableValue("obstetrics_scans"), nullableValue("abdominal_scans"),
		daysWorked,
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

// InsertClinicianReport inserts a new record from a populated ClinicianReportHistoryRow.
// FacilityID and DepartmentID are resolved from the employee record when zero.
// row.ReportID is set to the newly assigned ID on success.
func InsertClinicianReport(ctx context.Context, db *sql.DB, row *ClinicianReportHistoryRow) error {
	if !row.WeekStart.Valid {
		return fmt.Errorf("report start date is required")
	}

	facilityID := row.FacilityID
	departmentID := row.DepartmentID
	if facilityID == 0 || departmentID == 0 {
		const empQuery = `SELECT facility, department FROM clinician_app.employees WHERE id = $1`
		if err := db.QueryRowContext(ctx, empQuery, row.EmployeeID).Scan(&facilityID, &departmentID); err != nil {
			return fmt.Errorf("resolving employee facility/department: %w", err)
		}
	}

	var newID int
	if err := db.QueryRowContext(ctx, `SELECT COALESCE(MAX(id), 0) + 1 FROM clinician_app.weeklyreport`).Scan(&newID); err != nil {
		return fmt.Errorf("generating report id: %w", err)
	}

	nullInt := func(n sql.NullInt64) interface{} {
		if n.Valid {
			return n.Int64
		}
		return nil
	}

	start := row.WeekStart.Time
	stop := start
	if row.WeekStop.Valid {
		stop = row.WeekStop.Time
	}

	daysWorked := ""
	if row.DaysWorked.Valid {
		daysWorked = row.DaysWorked.String
	}

	createdOn := time.Now()
	if row.EnteredOn.Valid {
		createdOn = row.EnteredOn.Time
	}

	const sqlstr = `INSERT INTO clinician_app.weeklyreport (` +
		`id, hospital, department, employee, start, stop, ` +
		`attendance, ward_rounds, patients_reviewed, elective, emergency, postmortems, opd_clinics, opd_patients, anc_patients, ` +
		`teaching_rounds, students_taught, mortality_reviews, maternal, perinatal, surgical, medical, paed, labs_requests, imaging_requests, ` +
		`lab_investigations, bs, hiv, malaria, tb, cbc, chemistry, hematology, urinalysis, gram_stain, ` +
		`culture, microbiology, sensitivity_tests, diagnostics, xrays, ct_scans, obstetrics_scans, abdominal_scans, ` +
		`created_on, days_worked` +
		`) VALUES (` +
		`$1, $2, $3, $4, $5, $6, ` +
		`$7, $8, $9, $10, $11, $12, $13, $14, $15, ` +
		`$16, $17, $18, $19, $20, $21, $22, $23, $24, $25, ` +
		`$26, $27, $28, $29, $30, $31, $32, $33, $34, $35, ` +
		`$36, $37, $38, $39, $40, $41, $42, $43, ` +
		`$44, $45` +
		`)`

	_, err := db.ExecContext(ctx, sqlstr,
		newID, facilityID, departmentID, row.EmployeeID, start, stop,
		nullInt(row.Qn01), nullInt(row.Qn02), nullInt(row.Qn03), nullInt(row.Qn05), nullInt(row.Qn06), nullInt(row.Qn07), nullInt(row.Qn08), nullInt(row.Qn09), nullInt(row.Qn10),
		nullInt(row.Qn11), nullInt(row.Qn12), nullInt(row.Qn13), nullInt(row.Qn14), nullInt(row.Qn15), nullInt(row.Qn16), nullInt(row.Qn17), nullInt(row.Qn18), nullInt(row.Qn19), nullInt(row.Qn20),
		nullInt(row.Qn21), nullInt(row.Qn22), nullInt(row.Qn23), nullInt(row.Qn24), nullInt(row.Qn25), nullInt(row.Qn26), nullInt(row.Qn27), nullInt(row.Qn28), nullInt(row.Qn29), nullInt(row.Qn30),
		nullInt(row.Qn31), nullInt(row.Qn32), nullInt(row.Qn33), nullInt(row.Qn34), nullInt(row.Qn35), nullInt(row.Qn36), nullInt(row.Qn37), nullInt(row.Qn38),
		createdOn, daysWorked,
	)
	if err != nil {
		return err
	}
	row.ReportID = newID
	return nil
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
			COALESCE(w.patients_reviewed, 0) AS patients_reviewed,
			COALESCE(w.elective, 0) + COALESCE(w.emergency, 0) AS procedures
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
		SET
			report_status = 'Approved',
			facility_review_status = 'Approved',
			facility_reviewed_by = $3,
			facility_reviewed_on = NOW(),
			approved_by = $3,
			last_updated_on = NOW()
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
		SET
			report_status = 'Declined',
			facility_review_status = 'Declined',
			facility_reviewed_by = $3,
			facility_reviewed_on = NOW(),
			approved_by = $3,
			last_updated_on = NOW()
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
