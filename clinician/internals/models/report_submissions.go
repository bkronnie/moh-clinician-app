package models

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"
)

var ErrFacilityBatchRequiresApprovedReports = errors.New("all reports for the selected facility week must be locally approved before national submission")

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
	DecisionOn       sql.NullTime
	SubmitStatus     sql.NullString
	ReportStatus     sql.NullString
	Attendance       int
	PatientsReviewed int
	Procedures       int
	Actionable       bool
	IsOnLeave        bool
	Missing          bool
}

type FacilitySubmissionSummaryRow struct {
	FacilityID      int64
	FacilityName    string
	WeekStart       sql.NullTime
	WeekStop        sql.NullTime
	OnDutyCount     int
	OnLeaveCount    int
	SubmittedCount  int
	ApprovedCount   int
	DeclinedCount   int
	PendingCount    int
	MissingCount    int
	SubmittedOn     sql.NullTime
	StatusOn        sql.NullTime
	StatusBy        string
	ApprovalStatus  string
	SubmissionState string
	CanApprove      bool
}

func GetReportSubmissions(ctx context.Context, db *sql.DB, scopeFacilityID int64, scopeEmployeeID int64, viewerEmployeeID int64, filterFacilityID int, filterDepartmentID int, filterStatus string, year int, month int, week int) ([]*ReportSubmissionListRow, error) {
	items, _, err := GetReportSubmissionsPaged(ctx, db, scopeFacilityID, scopeEmployeeID, viewerEmployeeID, filterFacilityID, filterDepartmentID, filterStatus, year, month, week, 0, 0)
	return items, err
}

func GetReportSubmissionsPaged(ctx context.Context, db *sql.DB, scopeFacilityID int64, scopeEmployeeID int64, viewerEmployeeID int64, filterFacilityID int, filterDepartmentID int, filterStatus string, year int, month int, week int, limit int, offset int) ([]*ReportSubmissionListRow, int, error) {
	args := []interface{}{}
	whereParts := []string{}
	argPos := 1
	adminMode := scopeFacilityID == 0 && scopeEmployeeID == 0
	facilityApproverMode := scopeFacilityID > 0 && scopeEmployeeID == 0
	staffViewerMode := scopeEmployeeID > 0
	submitStatusExpr := "COALESCE(w.submit_status, '')"
	reportStatusExpr := "COALESCE(w.report_status, '')"
	decisionOnExpr := `CASE
				WHEN COALESCE(w.report_status, '') IN ('Approved', 'Rejected', 'Declined')
				THEN COALESCE(w.facility_reviewed_on, w.last_updated_on, w.created_on)
				ELSE NULL
			END`
	submittedOnExpr := `CASE
				WHEN COALESCE(w.submit_status, '') = 'Submitted'
				THEN COALESCE(w.submitted_on, w.last_updated_on, w.created_on)
				ELSE NULL
			END`

	effectiveFacilityID := filterFacilityID
	if scopeFacilityID > 0 {
		effectiveFacilityID = int(scopeFacilityID)
	}
	if adminMode {
		submitStatusExpr = "COALESCE(w.national_submission_status, '')"
		reportStatusExpr = "COALESCE(w.national_review_status, '')"
		decisionOnExpr = `CASE
				WHEN COALESCE(w.national_review_status, '') IN ('Approved', 'Rejected', 'Declined')
				THEN COALESCE(w.national_reviewed_on, w.last_updated_on, w.created_on)
				ELSE NULL
			END`
		submittedOnExpr = `CASE
				WHEN COALESCE(w.national_submission_status, '') = 'Submitted'
				THEN COALESCE(w.national_submitted_on, w.last_updated_on, w.created_on)
				ELSE NULL
			END`
	} else if facilityApproverMode {
		// Facility approvers should see national submission/review lifecycle once a batch is sent upward.
		// The viewer's own row (the facility admin) should NOT show "Submitted" merely because they saved
		// their personal entry — for the admin, "Submitted" means the facility batch has been escalated to
		// national. Until then their row is reported as Draft so the dashboard reflects the true upstream state.
		viewerSelfClause := ""
		if viewerEmployeeID > 0 {
			viewerSelfClause = fmt.Sprintf("WHEN w.employee = %d AND COALESCE(w.national_submission_status, '') <> 'Submitted' THEN '' ", viewerEmployeeID)
		}
		submitStatusExpr = `CASE
				WHEN COALESCE(w.national_submission_status, '') = 'Submitted' THEN 'Submitted'
				` + viewerSelfClause + `
				ELSE COALESCE(w.submit_status, '')
			END`
		reportStatusExpr = `CASE
				WHEN COALESCE(w.national_review_status, '') IN ('Approved', 'Rejected', 'Declined')
				THEN COALESCE(w.national_review_status, '')
				WHEN COALESCE(w.national_submission_status, '') = 'Submitted'
				THEN ''
				ELSE COALESCE(w.report_status, '')
			END`
		decisionOnExpr = `CASE
				WHEN COALESCE(w.national_review_status, '') IN ('Approved', 'Rejected', 'Declined')
				THEN COALESCE(w.national_reviewed_on, w.last_updated_on, w.created_on)
				WHEN COALESCE(w.report_status, '') IN ('Approved', 'Rejected', 'Declined')
				THEN COALESCE(w.facility_reviewed_on, w.last_updated_on, w.created_on)
				ELSE NULL
			END`
		submittedOnExpr = `CASE
				WHEN COALESCE(w.national_submission_status, '') = 'Submitted'
				THEN COALESCE(w.national_submitted_on, w.last_updated_on, w.created_on)
				WHEN COALESCE(w.submit_status, '') = 'Submitted'
				THEN COALESCE(w.submitted_on, w.last_updated_on, w.created_on)
				ELSE NULL
			END`
	} else if staffViewerMode {
		submitStatusExpr = `CASE
				WHEN COALESCE(w.national_submission_status, '') = 'Submitted' THEN 'Submitted'
				ELSE COALESCE(w.submit_status, '')
			END`
		reportStatusExpr = `CASE
				WHEN COALESCE(w.national_review_status, '') IN ('Approved', 'Rejected', 'Declined')
				THEN COALESCE(w.national_review_status, '')
				WHEN COALESCE(w.national_submission_status, '') = 'Submitted'
				THEN ''
				ELSE COALESCE(w.report_status, '')
			END`
		decisionOnExpr = `CASE
				WHEN COALESCE(w.national_review_status, '') IN ('Approved', 'Rejected', 'Declined')
				THEN COALESCE(w.national_reviewed_on, w.last_updated_on, w.created_on)
				WHEN COALESCE(w.report_status, '') IN ('Approved', 'Rejected', 'Declined')
				THEN COALESCE(w.facility_reviewed_on, w.last_updated_on, w.created_on)
				ELSE NULL
			END`
		submittedOnExpr = `CASE
				WHEN COALESCE(w.national_submission_status, '') = 'Submitted'
				THEN COALESCE(w.national_submitted_on, w.last_updated_on, w.created_on)
				WHEN COALESCE(w.submit_status, '') = 'Submitted'
				THEN COALESCE(w.submitted_on, w.last_updated_on, w.created_on)
				ELSE NULL
			END`
	}
	if scopeEmployeeID > 0 {
		whereParts = append(whereParts, fmt.Sprintf("w.employee = $%d", argPos))
		args = append(args, scopeEmployeeID)
		argPos++
	} else if adminMode {
		whereParts = append(whereParts, "(COALESCE(w.national_submission_status, '') = 'Submitted' OR COALESCE(w.national_review_status, '') IN ('Approved', 'Rejected', 'Declined'))")
	}
	// Facility approver mode: drafts of all staff in scope are visible so that
	// "All" really means every row for this facility. Cross-facility leakage is
	// still prevented by the hospital filter below.

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
		// "Submitted" means the clinician (or, for national, the facility) ever
		// submitted the report — regardless of any later approve/decline outcome.
		if facilityApproverMode {
			whereParts = append(whereParts, "(COALESCE(w.submit_status, '') = 'Submitted' OR COALESCE(w.national_submission_status, '') = 'Submitted')")
		} else {
			whereParts = append(whereParts, submitStatusExpr+" = 'Submitted'")
		}
	case "pending":
		// "Pending" means submitted but not yet finalized in this role's review queue.
		if facilityApproverMode {
			whereParts = append(whereParts, "COALESCE(w.submit_status, '') = 'Submitted'")
			whereParts = append(whereParts, "COALESCE(w.report_status, '') NOT IN ('Approved', 'Rejected', 'Declined')")
			whereParts = append(whereParts, "COALESCE(w.national_submission_status, '') <> 'Submitted'")
		} else {
			whereParts = append(whereParts, submitStatusExpr+" = 'Submitted'")
			whereParts = append(whereParts, reportStatusExpr+" NOT IN ('Approved', 'Rejected', 'Declined')")
		}
	case "approved":
		if facilityApproverMode {
			// Include facility-approved rows even after they are escalated to national review.
			whereParts = append(whereParts, "("+reportStatusExpr+" = 'Approved' OR COALESCE(w.report_status, '') = 'Approved')")
		} else {
			whereParts = append(whereParts, reportStatusExpr+" = 'Approved'")
		}
	case "declined":
		if facilityApproverMode {
			whereParts = append(whereParts, "("+reportStatusExpr+" IN ('Rejected', 'Declined') OR COALESCE(w.report_status, '') IN ('Rejected', 'Declined'))")
		} else {
			whereParts = append(whereParts, reportStatusExpr+" IN ('Rejected', 'Declined')")
		}
	case "draft":
		if adminMode {
			whereParts = append(whereParts, "1 = 0")
		} else {
			whereParts = append(whereParts, "COALESCE(w.submit_status, '') <> 'Submitted'")
			whereParts = append(whereParts, "COALESCE(w.report_status, '') NOT IN ('Approved', 'Rejected', 'Declined')")
		}
	}
	// Period filters: when a specific ISO week is selected it already pins the
	// period, so the month predicate is dropped to avoid year-boundary mismatches
	// where ISO year/week and calendar month disagree. When no week is chosen,
	// the year/month predicates use calendar year/month for intuitive UX.
	if year > 0 {
		if week > 0 {
			whereParts = append(whereParts, fmt.Sprintf("EXTRACT(ISOYEAR FROM w.start) = $%d", argPos))
		} else {
			whereParts = append(whereParts, fmt.Sprintf("EXTRACT(YEAR FROM w.start) = $%d", argPos))
		}
		args = append(args, year)
		argPos++
	}
	if month > 0 && week == 0 {
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

	countSQL := `
		SELECT COUNT(1)
		FROM clinician_app.weeklyreport w
		JOIN clinician_app.employees e ON e.id = w.employee
		LEFT JOIN clinician_app.facilities f ON f.id = w.hospital
		LEFT JOIN clinician_app.departments d ON d.id = w.department
		` + whereClause

	var totalCount int
	if err := db.QueryRowContext(ctx, countSQL, args...).Scan(&totalCount); err != nil {
		return nil, 0, err
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
			` + submittedOnExpr + ` AS submitted_on,
			` + decisionOnExpr + ` AS decision_on,
			` + submitStatusExpr + `,
			` + reportStatusExpr + `,
			COALESCE(w.national_submission_status, '') AS national_submission_status,
			COALESCE(w.attendance, 0) AS attendance,
			COALESCE(w.patients_reviewed, 0) AS patients_reviewed,
			COALESCE(w.elective, 0) + COALESCE(w.emergency, 0) AS procedures,
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

	queryArgs := append([]interface{}{}, args...)
	if limit > 0 {
		offsetValue := offset
		if offsetValue < 0 {
			offsetValue = 0
		}
		sqlstr += fmt.Sprintf(" LIMIT $%d OFFSET $%d", len(queryArgs)+1, len(queryArgs)+2)
		queryArgs = append(queryArgs, limit, offsetValue)
	}

	rows, err := db.QueryContext(ctx, sqlstr, queryArgs...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	items := []*ReportSubmissionListRow{}
	for rows.Next() {
		item := &ReportSubmissionListRow{}
		var submitStatusText string
		var reportStatusText string
		var nationalSubmissionStatusText string
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
			&item.DecisionOn,
			&submitStatusText,
			&reportStatusText,
			&nationalSubmissionStatusText,
			&item.Attendance,
			&item.PatientsReviewed,
			&item.Procedures,
			&item.IsOnLeave,
		); err != nil {
			return nil, 0, err
		}

		item.SubmitStatus = sql.NullString{String: submitStatusText, Valid: submitStatusText != ""}
		item.ReportStatus = sql.NullString{String: reportStatusText, Valid: reportStatusText != ""}
		if scopeEmployeeID > 0 {
			item.Actionable = submitStatusText != "Submitted" || reportStatusText == "Rejected" || reportStatusText == "Declined"
		} else if adminMode {
			item.Actionable = submitStatusText == "Submitted" && !isFinalReportSubmissionStatus(reportStatusText)
		} else if facilityApproverMode {
			item.Actionable = nationalSubmissionStatusText != "Submitted" && submitStatusText == "Submitted" && !isFinalReportSubmissionStatus(reportStatusText)
		} else {
			item.Actionable = submitStatusText == "Submitted" && !isFinalReportSubmissionStatus(reportStatusText)
		}
		items = append(items, item)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, err
	}

	return items, totalCount, nil
}

// GetMissingStaffForWeek returns synthetic ReportSubmissionListRow entries
// (Missing=true, ReportID=0) for every employee in the given facility (and
// optional department) who has no weeklyreport row for the specified week.
func GetMissingStaffForWeek(ctx context.Context, db *sql.DB, facilityID int, departmentID int, weekStart time.Time, weekStop time.Time) ([]*ReportSubmissionListRow, error) {
	if facilityID <= 0 {
		return []*ReportSubmissionListRow{}, nil
	}
	args := []interface{}{facilityID, weekStart, weekStop}
	departmentFilter := ""
	if departmentID > 0 {
		departmentFilter = " AND e.department = $4"
		args = append(args, departmentID)
	}
	query := `
		SELECT
			e.id,
			TRIM(CONCAT(COALESCE(e.fname, ''), ' ', COALESCE(e.lname, ''))) AS employee_name,
			e.facility,
			COALESCE(f.f_name, '') AS facility_name,
			COALESCE(e.department, 0) AS department_id,
			COALESCE(d.d_name, '') AS department_name,
			EXISTS (
				SELECT 1
				FROM clinician_app.staffleave sl
				WHERE sl.employee_id = e.id
				  AND COALESCE(sl.leave_status, '') IN ('Approved', 'Valid')
				  AND sl.start_date::date <= $3::date
				  AND sl.end_date::date >= $2::date
			) AS is_on_leave
		FROM clinician_app.employees e
		LEFT JOIN clinician_app.facilities f ON f.id = e.facility
		LEFT JOIN clinician_app.departments d ON d.id = e.department
		WHERE e.facility = $1
		` + departmentFilter + `
		  AND NOT EXISTS (
			SELECT 1
			FROM clinician_app.weeklyreport w
			WHERE w.employee = e.id
			  AND w.hospital = e.facility
			  AND w.start::date = $2::date
			  AND w.stop::date = $3::date
		  )
		ORDER BY employee_name
	`
	rows, err := db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := []*ReportSubmissionListRow{}
	for rows.Next() {
		item := &ReportSubmissionListRow{Missing: true}
		if err := rows.Scan(
			&item.EmployeeID,
			&item.EmployeeName,
			&item.FacilityID,
			&item.FacilityName,
			&item.DepartmentID,
			&item.DepartmentName,
			&item.IsOnLeave,
		); err != nil {
			return nil, err
		}
		item.WeekStart = sql.NullTime{Time: weekStart, Valid: true}
		item.WeekStop = sql.NullTime{Time: weekStop, Valid: true}
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

// FacilityWeekReadiness summarizes the gate state for a facility-week so the UI
// can show/hide the "Submit Weekly Facility Report" action and explain why it
// is blocked. Counts span every weeklyreport row for the period (regardless of
// the currently selected status tab).
type FacilityWeekReadiness struct {
	TotalCount       int // weeklyreport rows for the period
	DraftCount       int // submit_status <> 'Submitted'
	PendingCount     int // submit_status = 'Submitted' AND not yet finally reviewed
	ApprovedCount    int // report_status = 'Approved'
	DeclinedCount    int // report_status IN ('Rejected','Declined')
	UnapprovedExists bool
}

// GetFacilityWeekReadiness returns the per-status counts that gate the
// "Submit to National" action for a facility week (and optional department).
// When selfEmployeeID > 0, that employee's own still-draft row is excluded
// from the draft count because the submit-to-national flow promotes it
// automatically via selfQuery in SubmitFacilityReportsByFilter.
func GetFacilityWeekReadiness(ctx context.Context, db *sql.DB, facilityID int, departmentID int, year int, month int, week int, selfEmployeeID int64) (FacilityWeekReadiness, error) {
	out := FacilityWeekReadiness{}
	if facilityID <= 0 {
		return out, nil
	}
	args := []interface{}{facilityID}
	parts := []string{"w.hospital = $1"}
	pos := 2
	if departmentID > 0 {
		parts = append(parts, fmt.Sprintf("w.department = $%d", pos))
		args = append(args, departmentID)
		pos++
	}
	if year > 0 {
		parts = append(parts, fmt.Sprintf("EXTRACT(ISOYEAR FROM w.start) = $%d", pos))
		args = append(args, year)
		pos++
	}
	if month > 0 {
		parts = append(parts, fmt.Sprintf("EXTRACT(MONTH FROM w.start) = $%d", pos))
		args = append(args, month)
		pos++
	}
	if week > 0 {
		parts = append(parts, fmt.Sprintf("EXTRACT(WEEK FROM w.start) = $%d", pos))
		args = append(args, week)
		pos++
	}
	selfDraftPred := "FALSE"
	if selfEmployeeID > 0 {
		selfDraftPred = fmt.Sprintf("(w.employee = $%d AND COALESCE(w.submit_status, '') <> 'Submitted')", pos)
		args = append(args, selfEmployeeID)
		pos++
	}
	query := `
		SELECT
			COUNT(*) AS total_count,
			COUNT(*) FILTER (
				WHERE COALESCE(w.submit_status, '') <> 'Submitted'
				  AND NOT (` + selfDraftPred + `)
			) AS draft_count,
			COUNT(*) FILTER (
				WHERE COALESCE(w.submit_status, '') = 'Submitted'
				  AND COALESCE(w.report_status, '') NOT IN ('Approved', 'Rejected', 'Declined')
			) AS pending_count,
			COUNT(*) FILTER (WHERE COALESCE(w.report_status, '') = 'Approved') AS approved_count,
			COUNT(*) FILTER (WHERE COALESCE(w.report_status, '') IN ('Rejected', 'Declined')) AS declined_count
		FROM clinician_app.weeklyreport w
		WHERE ` + strings.Join(parts, " AND ")
	if err := db.QueryRowContext(ctx, query, args...).Scan(&out.TotalCount, &out.DraftCount, &out.PendingCount, &out.ApprovedCount, &out.DeclinedCount); err != nil {
		return out, err
	}
	out.UnapprovedExists = out.DraftCount > 0 || out.PendingCount > 0
	return out, nil
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

func GetReportSubmissionByIDForReview(ctx context.Context, db *sql.DB, reportID int, scopeFacilityID int64, reviewRole string) (*ClinicianReportHistoryRow, error) {
	args := []interface{}{reportID}
	whereClause := `WHERE w.id = $1`
	submitStatusExpr := "COALESCE(w.submit_status, '')"
	reportStatusExpr := "COALESCE(w.report_status, '')"
	submittedOnExpr := "w.submitted_on"
	if strings.EqualFold(reviewRole, "National Admin") {
		submitStatusExpr = "COALESCE(w.national_submission_status, '')"
		reportStatusExpr = "COALESCE(w.national_review_status, '')"
		submittedOnExpr = "w.national_submitted_on"
	}

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
			` + submitStatusExpr + `,
			` + reportStatusExpr + `,
			w.attendance, w.ward_rounds, w.patients_reviewed, w.elective, w.emergency, w.postmortems, w.opd_clinics, w.opd_patients, w.anc_patients,
			w.teaching_rounds, w.students_taught, w.mortality_reviews, w.maternal, w.perinatal, w.surgical, w.medical, w.paed, w.labs_requests, w.imaging_requests,
			w.lab_investigations, w.bs, w.hiv, w.malaria, w.tb, w.cbc, w.chemistry, w.hematology, w.urinalysis, w.gram_stain,
			w.culture, w.microbiology, w.sensitivity_tests, w.diagnostics, w.xrays, w.ct_scans, w.obstetrics_scans, w.abdominal_scans,
			COALESCE(w.days_worked, ''),
			COALESCE(w.absence_reason, ''),
			` + submittedOnExpr + `
		FROM clinician_app.weeklyreport w
		` + whereClause

	row := &ClinicianReportHistoryRow{}
	var submitStatusText string
	var reportStatusText string
	var daysWorkedText string
	var absenceReasonText string
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
		&row.Qn01, &row.Qn02, &row.Qn03, &row.Qn05, &row.Qn06, &row.Qn07, &row.Qn08, &row.Qn09, &row.Qn10,
		&row.Qn11, &row.Qn12, &row.Qn13, &row.Qn14, &row.Qn15, &row.Qn16, &row.Qn17, &row.Qn18, &row.Qn19, &row.Qn20,
		&row.Qn21, &row.Qn22, &row.Qn23, &row.Qn24, &row.Qn25, &row.Qn26, &row.Qn27, &row.Qn28, &row.Qn29, &row.Qn30,
		&row.Qn31, &row.Qn32, &row.Qn33, &row.Qn34, &row.Qn35, &row.Qn36, &row.Qn37, &row.Qn38,
		&daysWorkedText,
		&absenceReasonText,
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
	if absenceReasonText != "" {
		row.AbsenceReason = sql.NullString{String: absenceReasonText, Valid: true}
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
			facility_review_status = NULL,
			facility_reviewed_by = NULL,
			facility_reviewed_on = NULL,
			national_submission_status = NULL,
			national_submitted_by = NULL,
			national_submitted_on = NULL,
			national_review_status = NULL,
			national_reviewed_by = NULL,
			national_reviewed_on = NULL,
			submitted_on = $4,
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
	// Exclude the approver's own still-draft row: it is intentionally left
	// alone until the national batch submission promotes it (see
	// SubmitFacilityReportsByFilter / selfQuery).
	whereParts = append(whereParts, fmt.Sprintf("NOT (employee = $%d AND COALESCE(submit_status, '') <> 'Submitted')", approverArg))

	sqlstr := `
		UPDATE clinician_app.weeklyreport
		SET
			submit_status = 'Submitted',
			submitted_by = COALESCE(submitted_by, $` + fmt.Sprintf("%d", approverArg) + `),
			submitted_on = COALESCE(submitted_on, $` + fmt.Sprintf("%d", timeArg) + `),
			report_status = 'Approved',
			facility_review_status = 'Approved',
			facility_reviewed_by = $` + fmt.Sprintf("%d", approverArg) + `,
			facility_reviewed_on = $` + fmt.Sprintf("%d", timeArg) + `,
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
			facility_review_status = 'Declined',
			facility_reviewed_by = $` + fmt.Sprintf("%d", approverArg) + `,
			facility_reviewed_on = $` + fmt.Sprintf("%d", timeArg) + `,
			approved_by = $` + fmt.Sprintf("%d", approverArg) + `,
			last_updated_on = $` + fmt.Sprintf("%d", timeArg) + `
		WHERE ` + strings.Join(whereParts, " AND ")

	result, err := db.ExecContext(ctx, sqlstr, args...)
	if err != nil {
		return 0, err
	}

	return result.RowsAffected()
}

// ApproveFacilityReportDirect approves any report (including drafts) within a facility.
// When approving a draft it also marks it as submitted.
func ApproveFacilityReportDirect(ctx context.Context, db *sql.DB, reportID int, facilityID int64, approverID int64) (bool, error) {
	const sqlstr = `
		UPDATE clinician_app.weeklyreport
		SET
			submit_status = 'Submitted',
			submitted_on = CASE WHEN COALESCE(submit_status,'') <> 'Submitted' THEN NOW() ELSE submitted_on END,
			report_status = 'Approved',
			facility_review_status = 'Approved',
			facility_reviewed_by = $3,
			facility_reviewed_on = NOW(),
			approved_by = $3,
			last_updated_on = NOW()
		WHERE id = $1
			AND hospital = $2
			AND COALESCE(report_status, '') NOT IN ('Approved', 'Rejected', 'Declined')
	`
	result, err := db.ExecContext(ctx, sqlstr, reportID, facilityID, approverID)
	if err != nil {
		return false, err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return false, err
	}
	return rows > 0, nil
}

// DeclineFacilityReportDirect declines any report (including drafts) within a facility.
func DeclineFacilityReportDirect(ctx context.Context, db *sql.DB, reportID int, facilityID int64, approverID int64) (bool, error) {
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
			AND COALESCE(report_status, '') NOT IN ('Approved', 'Rejected', 'Declined')
	`
	result, err := db.ExecContext(ctx, sqlstr, reportID, facilityID, approverID)
	if err != nil {
		return false, err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return false, err
	}
	return rows > 0, nil
}

// ApproveFacilityReportsByDate approves (and auto-submits if draft) all
// non-reviewed reports for a specific date within a facility.
// departmentID = 0 means all departments.
func ApproveFacilityReportsByDate(ctx context.Context, db *sql.DB, facilityID int64, departmentID int, date time.Time, approverID int64) (int64, error) {
	args := []interface{}{facilityID, date, approverID}
	deptFilter := ""
	if departmentID > 0 {
		deptFilter = " AND department = $4"
		args = append(args, departmentID)
	}

	sqlstr := `
		UPDATE clinician_app.weeklyreport
		SET
			submit_status = 'Submitted',
			submitted_on = CASE WHEN COALESCE(submit_status,'') <> 'Submitted' THEN NOW() ELSE submitted_on END,
			report_status = 'Approved',
			facility_review_status = 'Approved',
			facility_reviewed_by = $3,
			facility_reviewed_on = NOW(),
			approved_by = $3,
			last_updated_on = NOW()
		WHERE hospital = $1
			AND start::date = $2::date
			AND COALESCE(report_status, '') NOT IN ('Approved', 'Rejected', 'Declined')
	` + deptFilter

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

	// Daily-report model: a staff member is "covered" for the week when, for
	// every calendar day in [weekStart, weekStop] they are not on leave, a
	// weeklyreport row exists for that day (start::date = day). We count the
	// number of (employee, day) gaps and treat any gap > 0 as missing.
	query := `
		WITH days AS (
			SELECT generate_series($2::date, $3::date, INTERVAL '1 day')::date AS day
		),
		on_duty_days AS (
			SELECT e.id AS employee_id, d.day
			FROM clinician_app.employees e
			CROSS JOIN days d
			WHERE e.facility = $1
			` + departmentFilter + `
			  AND NOT EXISTS (
				SELECT 1
				FROM clinician_app.staffleave sl
				WHERE sl.employee_id = e.id
				  AND COALESCE(sl.leave_status, '') IN ('Approved', 'Valid')
				  AND sl.start_date::date <= d.day
				  AND sl.end_date::date >= d.day
			  )
		)
		SELECT COUNT(*)
		FROM on_duty_days odd
		WHERE NOT EXISTS (
			SELECT 1
			FROM clinician_app.weeklyreport w
			WHERE w.employee = odd.employee_id
			  AND w.hospital = $1
			  AND w.start::date = odd.day
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
			attendance, ward_rounds, patients_reviewed, elective, emergency, postmortems, opd_clinics, opd_patients, anc_patients,
			teaching_rounds, students_taught, mortality_reviews, maternal, perinatal, surgical, medical, paed, labs_requests, imaging_requests,
			lab_investigations, bs, hiv, malaria, tb, cbc, chemistry, hematology, urinalysis, gram_stain,
			culture, microbiology, sensitivity_tests, diagnostics, xrays, ct_scans, obstetrics_scans, abdominal_scans,
			submit_status, report_status, submitted_by, submitted_on, created_on, last_updated_on, days_worked
		)
		SELECT
			e.facility,
			e.id,
			e.department,
			$2,
			$3,
			NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,
			NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,
			NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,
			NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,
			'Submitted', NULL, $4, $5, $5, $5, NULL
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
	_ = departmentID
	periodArgs := []interface{}{facilityID}
	periodWhereParts := []string{
		"hospital = $1",
	}
	argPos := 2
	if year > 0 {
		periodWhereParts = append(periodWhereParts, fmt.Sprintf("EXTRACT(ISOYEAR FROM start) = $%d", argPos))
		periodArgs = append(periodArgs, year)
		argPos++
	}
	if month > 0 {
		periodWhereParts = append(periodWhereParts, fmt.Sprintf("EXTRACT(MONTH FROM start) = $%d", argPos))
		periodArgs = append(periodArgs, month)
		argPos++
	}
	if week > 0 {
		periodWhereParts = append(periodWhereParts, fmt.Sprintf("EXTRACT(WEEK FROM start) = $%d", argPos))
		periodArgs = append(periodArgs, week)
		argPos++
	}

	filterArgs := append([]interface{}{}, periodArgs...)
	// Update target: only rows actually marked Submitted are escalated upward.
	whereParts := append([]string{}, periodWhereParts...)
	whereParts = append(whereParts,
		"COALESCE(submit_status, '') = 'Submitted'",
	)
	// Validation target: any in-scope row that is NOT fully facility-approved
	// (including unsubmitted Drafts) must block the national escalation so the
	// admin is forced to clear them before re-trying.
	validationWhereParts := append([]string{}, periodWhereParts...)

	updateArgs := append([]interface{}{}, filterArgs...)
	updateArgs = append(updateArgs, submittedBy)
	submittedByArg := argPos
	updateArgs = append(updateArgs, time.Now())
	nowArg := argPos + 1
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()

	// If the facility admin has a self row still in draft/declined for the selected week,
	// promote it to locally submitted+approved so the national batch stays consistent.
	selfArgs := append([]interface{}{}, periodArgs...)
	selfArgs = append(selfArgs, submittedBy)
	selfActorArg := len(periodArgs) + 1
	selfArgs = append(selfArgs, time.Now())
	selfTimeArg := selfActorArg + 1
	selfQuery := `
		UPDATE clinician_app.weeklyreport
		SET
			submit_status = 'Submitted',
			report_status = 'Approved',
			submitted_by = $` + fmt.Sprintf("%d", selfActorArg) + `,
			submitted_on = $` + fmt.Sprintf("%d", selfTimeArg) + `,
			facility_review_status = 'Approved',
			facility_reviewed_by = $` + fmt.Sprintf("%d", selfActorArg) + `,
			facility_reviewed_on = $` + fmt.Sprintf("%d", selfTimeArg) + `,
			approved_by = $` + fmt.Sprintf("%d", selfActorArg) + `,
			last_updated_on = $` + fmt.Sprintf("%d", selfTimeArg) + `
		WHERE ` + strings.Join(periodWhereParts, " AND ") + `
		  AND employee = $` + fmt.Sprintf("%d", selfActorArg) + `
		  AND (
			COALESCE(submit_status, '') <> 'Submitted'
			OR COALESCE(report_status, '') <> 'Approved'
			OR COALESCE(facility_review_status, '') <> 'Approved'
		  )`
	if _, err := tx.ExecContext(ctx, selfQuery, selfArgs...); err != nil {
		return 0, err
	}

	validationQuery := `
		SELECT COUNT(*)
		FROM clinician_app.weeklyreport
		WHERE ` + strings.Join(validationWhereParts, " AND ") + `
		  AND (
			COALESCE(facility_review_status, '') <> 'Approved'
			OR COALESCE(report_status, '') <> 'Approved'
			OR COALESCE(submit_status, '') <> 'Submitted'
		  )`

	var pendingLocalApproval int
	if err := tx.QueryRowContext(ctx, validationQuery, periodArgs...).Scan(&pendingLocalApproval); err != nil {
		return 0, err
	}
	if pendingLocalApproval > 0 {
		return 0, ErrFacilityBatchRequiresApprovedReports
	}

	query := `
		UPDATE clinician_app.weeklyreport
		SET
			national_submission_status = 'Submitted',
			national_submitted_by = $` + fmt.Sprintf("%d", submittedByArg) + `,
			national_submitted_on = $` + fmt.Sprintf("%d", nowArg) + `,
			national_review_status = NULL,
			national_reviewed_by = NULL,
			national_reviewed_on = NULL,
			last_updated_on = $` + fmt.Sprintf("%d", nowArg) + `
		WHERE ` + strings.Join(whereParts, " AND ") + `
		  AND COALESCE(facility_review_status, '') = 'Approved'
		  AND (
			COALESCE(national_submission_status, '') <> 'Submitted'
			OR COALESCE(national_review_status, '') IN ('Rejected', 'Declined')
		  )`

	result, err := tx.ExecContext(ctx, query, updateArgs...)
	if err != nil {
		return 0, err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return 0, err
	}
	if err := tx.Commit(); err != nil {
		return 0, err
	}

	return rowsAffected, nil
}

// SubmitFacilityReportsByDay escalates all locally-approved reports for a
// specific calendar day (start::date = date) to national_submission_status =
// 'Submitted'. Mirrors SubmitFacilityReportsByFilter but scoped to one day.
// Any existing row that is not yet facility-approved blocks the operation.
// The admin's own draft for that day is self-promoted before validation.
func SubmitFacilityReportsByDay(ctx context.Context, db *sql.DB, facilityID int64, departmentID int, date time.Time, submittedBy int64) (int64, error) {
	periodArgs := []interface{}{facilityID, date}
	periodWhereParts := []string{
		"hospital = $1",
		"start::date = $2::date",
	}
	argPos := 3
	if departmentID > 0 {
		periodWhereParts = append(periodWhereParts, fmt.Sprintf("department = $%d", argPos))
		periodArgs = append(periodArgs, departmentID)
		argPos++
	}

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()

	// Self-promote the admin's own draft for that day so the validation below passes.
	selfArgs := append([]interface{}{}, periodArgs...)
	selfArgs = append(selfArgs, submittedBy)
	selfActorArg := argPos
	selfArgs = append(selfArgs, time.Now())
	selfTimeArg := argPos + 1
	selfQuery := `
		UPDATE clinician_app.weeklyreport
		SET
			submit_status = 'Submitted',
			report_status = 'Approved',
			submitted_by = $` + fmt.Sprintf("%d", selfActorArg) + `,
			submitted_on = $` + fmt.Sprintf("%d", selfTimeArg) + `,
			facility_review_status = 'Approved',
			facility_reviewed_by = $` + fmt.Sprintf("%d", selfActorArg) + `,
			facility_reviewed_on = $` + fmt.Sprintf("%d", selfTimeArg) + `,
			approved_by = $` + fmt.Sprintf("%d", selfActorArg) + `,
			last_updated_on = $` + fmt.Sprintf("%d", selfTimeArg) + `
		WHERE ` + strings.Join(periodWhereParts, " AND ") + `
		  AND employee = $` + fmt.Sprintf("%d", selfActorArg) + `
		  AND (
			COALESCE(submit_status, '') <> 'Submitted'
			OR COALESCE(report_status, '') <> 'Approved'
			OR COALESCE(facility_review_status, '') <> 'Approved'
		  )`
	if _, err := tx.ExecContext(ctx, selfQuery, selfArgs...); err != nil {
		return 0, err
	}

	// Reject if any existing row for this day is not fully facility-approved.
	var pendingLocalApproval int
	validationQuery := `
		SELECT COUNT(*)
		FROM clinician_app.weeklyreport
		WHERE ` + strings.Join(periodWhereParts, " AND ") + `
		  AND (
			COALESCE(facility_review_status, '') <> 'Approved'
			OR COALESCE(report_status, '') <> 'Approved'
			OR COALESCE(submit_status, '') <> 'Submitted'
		  )`
	if err := tx.QueryRowContext(ctx, validationQuery, periodArgs...).Scan(&pendingLocalApproval); err != nil {
		return 0, err
	}
	if pendingLocalApproval > 0 {
		return 0, ErrFacilityBatchRequiresApprovedReports
	}

	// Escalate approved rows to national.
	updateArgs := append([]interface{}{}, periodArgs...)
	updateArgs = append(updateArgs, submittedBy)
	submittedByArg := argPos
	updateArgs = append(updateArgs, time.Now())
	nowArg := argPos + 1
	updateWhereParts := append([]string{}, periodWhereParts...)
	updateWhereParts = append(updateWhereParts, "COALESCE(submit_status, '') = 'Submitted'")
	query := `
		UPDATE clinician_app.weeklyreport
		SET
			national_submission_status = 'Submitted',
			national_submitted_by = $` + fmt.Sprintf("%d", submittedByArg) + `,
			national_submitted_on = $` + fmt.Sprintf("%d", nowArg) + `,
			national_review_status = NULL,
			national_reviewed_by = NULL,
			national_reviewed_on = NULL,
			last_updated_on = $` + fmt.Sprintf("%d", nowArg) + `
		WHERE ` + strings.Join(updateWhereParts, " AND ") + `
		  AND COALESCE(facility_review_status, '') = 'Approved'
		  AND (
			COALESCE(national_submission_status, '') <> 'Submitted'
			OR COALESCE(national_review_status, '') IN ('Rejected', 'Declined')
		  )`
	result, err := tx.ExecContext(ctx, query, updateArgs...)
	if err != nil {
		return 0, err
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return 0, err
	}
	if err := tx.Commit(); err != nil {
		return 0, err
	}
	return rowsAffected, nil
}

func ApproveNationalFacilityReportsByFilter(ctx context.Context, db *sql.DB, facilityID int64, departmentID int, year int, month int, week int, approverID int64) (int64, error) {
	_ = departmentID
	args := []interface{}{facilityID}
	whereParts := []string{
		"hospital = $1",
		"COALESCE(national_submission_status, '') = 'Submitted'",
		"COALESCE(national_review_status, '') NOT IN ('Approved', 'Rejected', 'Declined')",
	}
	argPos := 2

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

	query := `
		UPDATE clinician_app.weeklyreport
		SET
			report_status = 'Approved',
			facility_review_status = 'Approved',
			national_review_status = 'Approved',
			national_reviewed_by = $` + fmt.Sprintf("%d", approverArg) + `,
			national_reviewed_on = $` + fmt.Sprintf("%d", timeArg) + `,
			approved_by = $` + fmt.Sprintf("%d", approverArg) + `,
			last_updated_on = $` + fmt.Sprintf("%d", timeArg) + `
		WHERE ` + strings.Join(whereParts, " AND ")

	result, err := db.ExecContext(ctx, query, args...)
	if err != nil {
		return 0, err
	}

	return result.RowsAffected()
}

func DeclineNationalFacilityReportsByFilter(ctx context.Context, db *sql.DB, facilityID int64, departmentID int, year int, month int, week int, approverID int64) (int64, error) {
	_ = departmentID
	args := []interface{}{facilityID}
	whereParts := []string{
		"hospital = $1",
		"COALESCE(national_submission_status, '') = 'Submitted'",
		"COALESCE(national_review_status, '') NOT IN ('Approved', 'Rejected', 'Declined')",
	}
	argPos := 2

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

	query := `
		UPDATE clinician_app.weeklyreport
		SET
			report_status = 'Declined',
			facility_review_status = 'Declined',
			national_review_status = 'Declined',
			national_reviewed_by = $` + fmt.Sprintf("%d", approverArg) + `,
			national_reviewed_on = $` + fmt.Sprintf("%d", timeArg) + `,
			approved_by = $` + fmt.Sprintf("%d", approverArg) + `,
			last_updated_on = $` + fmt.Sprintf("%d", timeArg) + `
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
				COUNT(*) FILTER (WHERE COALESCE(w.national_submission_status, '') = 'Submitted') AS submitted_count,
				COUNT(*) FILTER (WHERE COALESCE(w.national_review_status, '') = 'Approved') AS approved_count,
				COUNT(*) FILTER (WHERE COALESCE(w.national_review_status, '') IN ('Rejected', 'Declined')) AS declined_count,
				COUNT(*) FILTER (
					WHERE COALESCE(w.national_submission_status, '') = 'Submitted'
					  AND COALESCE(w.national_review_status, '') NOT IN ('Approved', 'Rejected', 'Declined')
				) AS pending_count,
				MAX(CASE WHEN COALESCE(w.national_submission_status, '') = 'Submitted' THEN COALESCE(w.national_submitted_on, w.last_updated_on, w.created_on) END) AS submitted_on
			FROM clinician_app.weeklyreport w
			GROUP BY w.hospital, w.start
		)
		SELECT
			fw.facility_id,
			COALESCE(f.f_name, '') AS facility_name,
			fw.week_start,
			fw.week_stop,
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
			&item.WeekStart,
			&item.WeekStop,
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

		item.CanApprove = item.ApprovalStatus == "Submitted"

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

func GetFacilityWeeklySubmissionSummaries(ctx context.Context, db *sql.DB, facilityID int, departmentID int, selectedStatus string, year int, month int, week int) ([]*FacilitySubmissionSummaryRow, error) {
	if facilityID <= 0 {
		return []*FacilitySubmissionSummaryRow{}, nil
	}

	weekWhereParts := []string{"w.hospital = $1"}
	args := []interface{}{facilityID}
	argPos := 2

	if departmentID > 0 {
		weekWhereParts = append(weekWhereParts, fmt.Sprintf("w.department = $%d", argPos))
		args = append(args, departmentID)
		argPos++
	}
	if year > 0 {
		weekWhereParts = append(weekWhereParts, fmt.Sprintf("EXTRACT(ISOYEAR FROM w.start) = $%d", argPos))
		args = append(args, year)
		argPos++
	}
	if month > 0 {
		weekWhereParts = append(weekWhereParts, fmt.Sprintf("EXTRACT(MONTH FROM w.start) = $%d", argPos))
		args = append(args, month)
		argPos++
	}
	if week > 0 {
		weekWhereParts = append(weekWhereParts, fmt.Sprintf("EXTRACT(WEEK FROM w.start) = $%d", argPos))
		args = append(args, week)
		argPos++
	}

	employeeWhereParts := []string{"e.facility = $1"}
	if departmentID > 0 {
		employeeWhereParts = append(employeeWhereParts, fmt.Sprintf("e.department = $%d", 2))
	}

	weekWhereClause := "WHERE " + strings.Join(weekWhereParts, " AND ")
	employeeWhereClause := "WHERE " + strings.Join(employeeWhereParts, " AND ")

	query := `
		WITH facility_weeks AS (
			SELECT DISTINCT
				w.hospital AS facility_id,
				date_trunc('week', w.start)::date AS week_start,
				(date_trunc('week', w.start) + INTERVAL '6 days')::date AS week_stop
			FROM clinician_app.weeklyreport w
			` + weekWhereClause + `
		), on_leave AS (
			SELECT fw.facility_id, fw.week_start, COUNT(DISTINCT e.id) AS on_leave_count
			FROM facility_weeks fw
			JOIN clinician_app.employees e ON e.facility = fw.facility_id
			JOIN clinician_app.staffleave sl ON sl.employee_id = e.id
			WHERE COALESCE(sl.leave_status, '') IN ('Approved', 'Valid')
			  AND sl.start_date::date <= fw.week_stop
			  AND sl.end_date::date >= fw.week_start
			  ` + strings.Replace(employeeWhereClause, "WHERE ", "AND ", 1) + `
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
			  ` + strings.Replace(employeeWhereClause, "WHERE ", "AND ", 1) + `
			GROUP BY fw.facility_id, fw.week_start
		), submissions AS (
			SELECT
				w.hospital AS facility_id,
				date_trunc('week', w.start)::date AS week_start,
				COUNT(*) FILTER (WHERE COALESCE(w.submit_status, '') = 'Submitted') AS submitted_count,
				COUNT(*) FILTER (WHERE COALESCE(w.report_status, '') = 'Approved') AS approved_count,
				COUNT(*) FILTER (WHERE COALESCE(w.report_status, '') IN ('Rejected', 'Declined')) AS declined_count,
				COUNT(*) FILTER (
					WHERE COALESCE(w.submit_status, '') = 'Submitted'
					  AND COALESCE(w.report_status, '') NOT IN ('Approved', 'Rejected', 'Declined')
				) AS pending_count,
				MAX(CASE WHEN COALESCE(w.submit_status, '') = 'Submitted' THEN COALESCE(w.submitted_on, w.last_updated_on, w.created_on) END) AS submitted_on,
				COUNT(*) FILTER (WHERE COALESCE(w.national_submission_status, '') = 'Submitted') AS national_submitted_count,
				COUNT(*) FILTER (WHERE COALESCE(w.national_review_status, '') = 'Approved') AS national_approved_count,
				COUNT(*) FILTER (WHERE COALESCE(w.national_review_status, '') IN ('Rejected', 'Declined')) AS national_declined_count
			FROM clinician_app.weeklyreport w
			` + weekWhereClause + `
			GROUP BY w.hospital, date_trunc('week', w.start)::date
		), national_submit_events AS (
			SELECT DISTINCT ON (w.hospital, date_trunc('week', w.start)::date)
				w.hospital AS facility_id,
				date_trunc('week', w.start)::date AS week_start,
				COALESCE(w.national_submitted_on, w.last_updated_on, w.created_on) AS submitted_on,
				TRIM(CONCAT(COALESCE(e.fname, ''), ' ', COALESCE(e.lname, ''))) AS submitted_by
			FROM clinician_app.weeklyreport w
			LEFT JOIN clinician_app.employees e ON e.id = w.national_submitted_by
			` + weekWhereClause + `
			  AND COALESCE(w.national_submission_status, '') = 'Submitted'
			ORDER BY w.hospital, date_trunc('week', w.start)::date, COALESCE(w.national_submitted_on, w.last_updated_on, w.created_on) DESC, w.id DESC
		), national_review_events AS (
			SELECT DISTINCT ON (w.hospital, date_trunc('week', w.start)::date)
				w.hospital AS facility_id,
				date_trunc('week', w.start)::date AS week_start,
				COALESCE(w.national_reviewed_on, w.last_updated_on, w.created_on) AS reviewed_on,
				TRIM(CONCAT(COALESCE(e.fname, ''), ' ', COALESCE(e.lname, ''))) AS reviewed_by
			FROM clinician_app.weeklyreport w
			LEFT JOIN clinician_app.employees e ON e.id = w.national_reviewed_by
			` + weekWhereClause + `
			  AND COALESCE(w.national_review_status, '') IN ('Approved', 'Rejected', 'Declined')
			ORDER BY w.hospital, date_trunc('week', w.start)::date, COALESCE(w.national_reviewed_on, w.last_updated_on, w.created_on) DESC, w.id DESC
		)
		SELECT
			fw.facility_id,
			COALESCE(f.f_name, '') AS facility_name,
			fw.week_start,
			fw.week_stop,
			COALESCE(od.on_duty_count, 0) AS on_duty_count,
			COALESCE(ol.on_leave_count, 0) AS on_leave_count,
			COALESCE(s.submitted_count, 0) AS submitted_count,
			COALESCE(s.approved_count, 0) AS approved_count,
			COALESCE(s.declined_count, 0) AS declined_count,
			COALESCE(s.pending_count, 0) AS pending_count,
			GREATEST(COALESCE(od.on_duty_count, 0) + COALESCE(ol.on_leave_count, 0) - COALESCE(s.submitted_count, 0), 0) AS missing_count,
			s.submitted_on,
			COALESCE(s.national_submitted_count, 0) AS national_submitted_count,
			COALESCE(s.national_approved_count, 0) AS national_approved_count,
			COALESCE(s.national_declined_count, 0) AS national_declined_count,
			nse.submitted_on AS national_submitted_on,
			COALESCE(nse.submitted_by, '') AS national_submitted_by,
			nre.reviewed_on AS national_reviewed_on,
			COALESCE(nre.reviewed_by, '') AS national_reviewed_by
		FROM facility_weeks fw
		LEFT JOIN clinician_app.facilities f ON f.id = fw.facility_id
		LEFT JOIN on_duty od ON od.facility_id = fw.facility_id AND od.week_start = fw.week_start
		LEFT JOIN on_leave ol ON ol.facility_id = fw.facility_id AND ol.week_start = fw.week_start
		LEFT JOIN submissions s ON s.facility_id = fw.facility_id AND s.week_start = fw.week_start
		LEFT JOIN national_submit_events nse ON nse.facility_id = fw.facility_id AND nse.week_start = fw.week_start
		LEFT JOIN national_review_events nre ON nre.facility_id = fw.facility_id AND nre.week_start = fw.week_start
		ORDER BY fw.week_start DESC
	`

	rows, err := db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := []*FacilitySubmissionSummaryRow{}
	for rows.Next() {
		item := &FacilitySubmissionSummaryRow{}
		var nationalSubmittedCount int
		var nationalApprovedCount int
		var nationalDeclinedCount int
		var nationalSubmittedOn sql.NullTime
		var nationalSubmittedBy string
		var nationalReviewedOn sql.NullTime
		var nationalReviewedBy string
		if err := rows.Scan(
			&item.FacilityID,
			&item.FacilityName,
			&item.WeekStart,
			&item.WeekStop,
			&item.OnDutyCount,
			&item.OnLeaveCount,
			&item.SubmittedCount,
			&item.ApprovedCount,
			&item.DeclinedCount,
			&item.PendingCount,
			&item.MissingCount,
			&item.SubmittedOn,
			&nationalSubmittedCount,
			&nationalApprovedCount,
			&nationalDeclinedCount,
			&nationalSubmittedOn,
			&nationalSubmittedBy,
			&nationalReviewedOn,
			&nationalReviewedBy,
		); err != nil {
			return nil, err
		}

		// Expected non-leave daily reports for the week (7 days per on-duty staff).
		expectedReports := (item.OnDutyCount - item.OnLeaveCount) * 7
		if expectedReports < 0 {
			expectedReports = 0
		}
		fullyNationalSubmitted := nationalSubmittedCount > 0 &&
			expectedReports > 0 &&
			nationalSubmittedCount >= expectedReports

		switch {
		case nationalApprovedCount > 0 && nationalApprovedCount >= expectedReports && expectedReports > 0:
			item.ApprovalStatus = "Approved"
			item.StatusOn = nationalReviewedOn
			item.StatusBy = strings.TrimSpace(nationalReviewedBy)
		case nationalDeclinedCount > 0:
			item.ApprovalStatus = "Declined"
			item.StatusOn = nationalReviewedOn
			item.StatusBy = strings.TrimSpace(nationalReviewedBy)
		case fullyNationalSubmitted:
			item.ApprovalStatus = "Submitted"
			item.StatusOn = nationalSubmittedOn
			item.StatusBy = strings.TrimSpace(nationalSubmittedBy)
		default:
			// Partial submissions (some days submitted, others not) are treated
			// as Draft so the weekly action button continues to read
			// "Submit Week" until every non-leave report has been escalated.
			item.ApprovalStatus = "Draft"
		}

		if item.MissingCount > 0 || item.SubmittedCount == 0 {
			item.SubmissionState = "Incomplete"
		} else {
			item.SubmissionState = "Complete"
		}

		item.CanApprove = item.ApprovalStatus == "Submitted"

		if selectedStatus != "all" {
			switch selectedStatus {
			case "pending":
				if item.ApprovalStatus != "Submitted" {
					continue
				}
			case "submitted":
				if item.ApprovalStatus != "Submitted" {
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
				if item.ApprovalStatus != "Draft" {
					continue
				}
			}
		}

		items = append(items, item)
	}

	return items, rows.Err()
}

// WeekDayRow holds per-day submission stats for a single day within an ISO week.
type WeekDayRow struct {
	Day          time.Time
	DayName      string
	TotalStaff   int
	OnLeaveCount int
	OnDutyCount  int
	Submitted    int
	Approved     int
	NotSubmitted int
	Percentage   int
}

// DayStaffRow holds a single staff member's report submission status for a given day.
type DayStaffRow struct {
	ReportID       int
	EmployeeID     int64
	EmployeeName   string
	DepartmentName string
	HasSubmitted   bool
	SubmitStatus   string
	ReportStatus   string
	IsOnLeave      bool
}

// GetWeekDailyBreakdown returns one WeekDayRow per day (Monday–Sunday) for the ISO
// week that starts at weekStart. Each row shows on-duty count, submitted count, and
// the derived not-submitted count and submission percentage.
func GetWeekDailyBreakdown(ctx context.Context, db *sql.DB, facilityID int, departmentID int, weekStart time.Time) ([]*WeekDayRow, error) {
	if facilityID <= 0 {
		return buildEmptyWeekDays(weekStart), nil
	}

	weekStop := weekStart.AddDate(0, 0, 6)

	args := []interface{}{facilityID, weekStart, weekStop}
	argPos := 4
	employeeDeptFilter := ""
	reportDeptFilter := ""
	if departmentID > 0 {
		employeeDeptFilter = fmt.Sprintf(" AND e.department = $%d", argPos)
		reportDeptFilter = fmt.Sprintf(" AND w.department = $%d", argPos)
		args = append(args, departmentID)
	}

	query := `
		WITH day_series AS (
			SELECT generate_series($2::date, $3::date, '1 day'::interval)::date AS day
		), employee_base AS (
			SELECT COUNT(DISTINCT e.id) AS total_staff
			FROM clinician_app.employees e
			WHERE e.facility = $1
			` + employeeDeptFilter + `
		), day_submissions AS (
			SELECT
				w.start::date AS report_day,
				COUNT(*) AS submitted_count,
				COUNT(*) FILTER (WHERE COALESCE(w.report_status,'') = 'Approved') AS approved_count
			FROM clinician_app.weeklyreport w
			WHERE w.hospital = $1
			  AND w.start::date >= $2::date
			  AND w.start::date <= $3::date
			` + reportDeptFilter + `
			GROUP BY w.start::date
		), day_on_leave AS (
			SELECT ds.day, COUNT(DISTINCT e.id) AS on_leave_count
			FROM day_series ds
			JOIN clinician_app.employees e ON e.facility = $1` + employeeDeptFilter + `
			JOIN clinician_app.staffleave sl ON sl.employee_id = e.id
			WHERE COALESCE(sl.leave_status, '') IN ('Approved', 'Valid')
			  AND sl.start_date::date <= ds.day
			  AND sl.end_date::date >= ds.day
			GROUP BY ds.day
		)
		SELECT
			ds.day,
			eb.total_staff,
			COALESCE(dl.on_leave_count, 0) AS on_leave_count,
			GREATEST(eb.total_staff - COALESCE(dl.on_leave_count, 0), 0) AS on_duty_count,
			COALESCE(dsub.submitted_count, 0) AS submitted_count,
			COALESCE(dsub.approved_count, 0) AS approved_count
		FROM day_series ds
		CROSS JOIN employee_base eb
		LEFT JOIN day_submissions dsub ON dsub.report_day = ds.day
		LEFT JOIN day_on_leave dl ON dl.day = ds.day
		ORDER BY ds.day
	`

	rows, err := db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	dayNames := [8]string{"", "Monday", "Tuesday", "Wednesday", "Thursday", "Friday", "Saturday", "Sunday"}
	items := []*WeekDayRow{}
	for rows.Next() {
		item := &WeekDayRow{}
		if err := rows.Scan(&item.Day, &item.TotalStaff, &item.OnLeaveCount, &item.OnDutyCount, &item.Submitted, &item.Approved); err != nil {
			return nil, err
		}
		wd := int(item.Day.Weekday()) // Sunday=0
		if wd == 0 {
			wd = 7
		}
		item.DayName = dayNames[wd]
		item.NotSubmitted = item.OnDutyCount - item.Submitted
		if item.NotSubmitted < 0 {
			item.NotSubmitted = 0
		}
		if item.OnDutyCount > 0 {
			item.Percentage = item.Submitted * 100 / item.OnDutyCount
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

func buildEmptyWeekDays(weekStart time.Time) []*WeekDayRow {
	dayNames := [7]string{"Monday", "Tuesday", "Wednesday", "Thursday", "Friday", "Saturday", "Sunday"}
	items := make([]*WeekDayRow, 7)
	for i := 0; i < 7; i++ {
		items[i] = &WeekDayRow{
			Day:     weekStart.AddDate(0, 0, i),
			DayName: dayNames[i],
		}
	}
	return items
}

// GetDayStaffStatus returns every staff member at the facility with their
// weeklyreport submission status for the given date. Staff with no report row
// have HasSubmitted=false. The department filter is optional (0 = all).
func GetDayStaffStatus(ctx context.Context, db *sql.DB, facilityID int, departmentID int, date time.Time) ([]*DayStaffRow, error) {
	if facilityID <= 0 {
		return []*DayStaffRow{}, nil
	}

	args := []interface{}{facilityID, date}
	deptFilter := ""
	if departmentID > 0 {
		deptFilter = fmt.Sprintf(" AND e.department = $%d", 3)
		args = append(args, departmentID)
	}

	query := `
		SELECT
			e.id AS employee_id,
			TRIM(CONCAT(COALESCE(e.fname, ''), ' ', COALESCE(e.lname, ''))) AS employee_name,
			COALESCE(d.d_name, '') AS department_name,
			(w.id IS NOT NULL) AS has_submitted,
			COALESCE(w.submit_status, '') AS submit_status,
			COALESCE(w.report_status, '') AS report_status,
			COALESCE(w.id, 0) AS report_id,
			EXISTS (
				SELECT 1 FROM clinician_app.staffleave sl
				WHERE sl.employee_id = e.id
				  AND COALESCE(sl.leave_status, '') IN ('Approved', 'Valid')
				  AND sl.start_date::date <= $2::date
				  AND sl.end_date::date >= $2::date
			) AS is_on_leave
		FROM clinician_app.employees e
		LEFT JOIN clinician_app.departments d ON d.id = e.department
		LEFT JOIN clinician_app.weeklyreport w
			ON w.employee = e.id
			AND w.hospital = $1
			AND w.start::date = $2::date
		WHERE e.facility = $1
		` + deptFilter + `
		ORDER BY employee_name
	`

	rows, err := db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := []*DayStaffRow{}
	for rows.Next() {
		item := &DayStaffRow{}
		if err := rows.Scan(
			&item.EmployeeID,
			&item.EmployeeName,
			&item.DepartmentName,
			&item.HasSubmitted,
			&item.SubmitStatus,
			&item.ReportStatus,
			&item.ReportID,
			&item.IsOnLeave,
		); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}
