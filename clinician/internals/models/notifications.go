package models

import (
	"context"
	"database/sql"
)

// EnsureLeaveSchema adds the reviewed_on column to staffleave if it doesn't exist.
// This is safe to call repeatedly (ADD COLUMN IF NOT EXISTS).
func EnsureLeaveSchema(ctx context.Context, db *sql.DB) error {
	_, err := db.ExecContext(ctx, `
		ALTER TABLE clinician_app.staffleave
		ADD COLUMN IF NOT EXISTS reviewed_on TIMESTAMP
	`)
	if err != nil {
		return err
	}

	_, err = db.ExecContext(ctx, `
		UPDATE clinician_app.staffleave
		SET leave_status = 'Completed'
		WHERE leave_status = 'Expired'
	`)
	return err
}

// EnsureWeeklyReportSchema adds columns used by attendance day selection and submission tracking.
func EnsureWeeklyReportSchema(ctx context.Context, db *sql.DB) error {
	_, err := db.ExecContext(ctx, `
		ALTER TABLE clinician_app.weeklyreport
		ADD COLUMN IF NOT EXISTS days_worked TEXT,
		ADD COLUMN IF NOT EXISTS submitted_on TIMESTAMP,
		ADD COLUMN IF NOT EXISTS facility_review_status TEXT,
		ADD COLUMN IF NOT EXISTS facility_reviewed_by BIGINT,
		ADD COLUMN IF NOT EXISTS facility_reviewed_on TIMESTAMP,
		ADD COLUMN IF NOT EXISTS national_submission_status TEXT,
		ADD COLUMN IF NOT EXISTS national_submitted_by BIGINT,
		ADD COLUMN IF NOT EXISTS national_submitted_on TIMESTAMP,
		ADD COLUMN IF NOT EXISTS national_review_status TEXT,
		ADD COLUMN IF NOT EXISTS national_reviewed_by BIGINT,
		ADD COLUMN IF NOT EXISTS national_reviewed_on TIMESTAMP
	`)
	if err != nil {
		return err
	}

	return EnsureReportDataElementSchema(ctx, db)
}

// EnsurePerformanceIndexes adds non-destructive indexes for the query patterns used
// by dashboard, report review, leave review, and role data lookups.
func EnsurePerformanceIndexes(ctx context.Context, db *sql.DB) error {
	statements := []string{
		`CREATE EXTENSION IF NOT EXISTS btree_gist`,
		`CREATE INDEX IF NOT EXISTS idx_weeklyreport_hospital_start_status ON clinician_app.weeklyreport(hospital, start, submit_status, report_status)`,
		`CREATE INDEX IF NOT EXISTS idx_weeklyreport_employee_start_status ON clinician_app.weeklyreport(employee, start, submit_status, report_status)`,
		`DROP INDEX IF EXISTS clinician_app.ux_weeklyreport_employee_start_stop`,
		`CREATE UNIQUE INDEX IF NOT EXISTS ux_weeklyreport_employee_hospital_start ON clinician_app.weeklyreport(employee, hospital, start) WHERE employee IS NOT NULL AND hospital IS NOT NULL AND start IS NOT NULL`,
		`ALTER TABLE clinician_app.weeklyreport DROP CONSTRAINT IF EXISTS ck_weeklyreport_start_stop_7_days`,
		`ALTER TABLE clinician_app.weeklyreport DROP CONSTRAINT IF EXISTS ck_weeklyreport_start_stop_6_days`,
		`ALTER TABLE clinician_app.weeklyreport ADD CONSTRAINT ck_weeklyreport_start_stop_6_days CHECK (start IS NULL OR stop IS NULL OR stop = start + INTERVAL '6 days') NOT VALID`,
		`ALTER TABLE clinician_app.weeklyreport DROP CONSTRAINT IF EXISTS ck_weeklyreport_start_monday`,
		`ALTER TABLE clinician_app.weeklyreport ADD CONSTRAINT ck_weeklyreport_start_monday CHECK (start IS NULL OR EXTRACT(ISODOW FROM start) = 1) NOT VALID`,
		`ALTER TABLE clinician_app.weeklyreport DROP CONSTRAINT IF EXISTS ex_weeklyreport_employee_hospital_no_overlap`,
		`ALTER TABLE clinician_app.weeklyreport ADD CONSTRAINT ex_weeklyreport_employee_hospital_no_overlap EXCLUDE USING gist (employee WITH =, hospital WITH =, daterange(start, stop, '[]') WITH &&) WHERE (employee IS NOT NULL AND hospital IS NOT NULL AND start IS NOT NULL AND stop IS NOT NULL)`,
		`CREATE INDEX IF NOT EXISTS idx_weeklyreport_facility_week_review_flow ON clinician_app.weeklyreport(hospital, start, facility_review_status, national_submission_status, national_review_status)`,
		`CREATE INDEX IF NOT EXISTS idx_staffleave_employee_status_dates ON clinician_app.staffleave(employee_id, leave_status, start_date, end_date)`,
		`CREATE INDEX IF NOT EXISTS idx_staffleave_status_dates ON clinician_app.staffleave(leave_status, start_date, end_date)`,
		`CREATE INDEX IF NOT EXISTS idx_users_employees ON clinician_app.users(employees)`,
		`CREATE INDEX IF NOT EXISTS idx_department_roles_dept_role ON clinician_app.department_roles(dept_id, role_name)`,
	}

	for _, statement := range statements {
		if _, err := db.ExecContext(ctx, statement); err != nil {
			return err
		}
	}

	return nil
}

// EnsureCustomizationSchema provisions audit storage for national customization changes.
func EnsureCustomizationSchema(ctx context.Context, db *sql.DB) error {
	if err := EnsureCustomizationAuditSchema(ctx, db); err != nil {
		return err
	}
	return EnsureRoleMetricTargetSchema(ctx, db)
}

// ExpireOldLeave marks approved/valid leaves whose end_date is before today as 'Completed'.
// Pass facilityID = 0 to expire across all facilities.
func ExpireOldLeave(ctx context.Context, db DB, facilityID int64) error {
	// Keep the terminal leave status canonical.
	if _, err := db.ExecContext(ctx, `
		UPDATE clinician_app.staffleave
		SET leave_status = 'Completed'
		WHERE leave_status = 'Expired'
	`); err != nil {
		return err
	}

	if facilityID > 0 {
		_, err := db.ExecContext(ctx, `
			UPDATE clinician_app.staffleave sl
			SET leave_status = 'Completed'
			FROM clinician_app.employees e
			WHERE sl.employee_id = e.id
			  AND sl.leave_status IN ('Approved', 'Valid')
			  AND sl.end_date < CURRENT_DATE
			  AND e.facility = $1
		`, facilityID)
		return err
	}
	_, err := db.ExecContext(ctx, `
		UPDATE clinician_app.staffleave
		SET leave_status = 'Completed'
		WHERE leave_status IN ('Approved', 'Valid')
		  AND end_date < CURRENT_DATE
	`)
	return err
}

// GetPendingLeaveCount returns the count of unreviewed (Pending) leave requests for a facility.
func GetPendingLeaveCount(ctx context.Context, db DB, facilityID int64) (int, error) {
	var count int
	err := db.QueryRowContext(ctx, `
		SELECT COUNT(*)
		FROM clinician_app.staffleave sl
		JOIN clinician_app.employees e ON e.id = sl.employee_id
		WHERE e.facility = $1
		  AND COALESCE(sl.leave_status, '') = 'Pending'
	`, facilityID).Scan(&count)
	if err != nil {
		return 0, nil
	}
	return count, nil
}

// GetPendingReportCount returns the count of submitted-but-unreviewed reports.
// Pass facilityID = 0 to count nationally (for admin).
func GetPendingReportCount(ctx context.Context, db DB, facilityID int64) (int, error) {
	var count int
	var err error
	if facilityID > 0 {
		err = db.QueryRowContext(ctx, `
			SELECT COUNT(*)
			FROM clinician_app.weeklyreport w
			WHERE w.hospital = $1
			  AND COALESCE(w.submit_status, '') = 'Submitted'
			  AND COALESCE(w.report_status, '') NOT IN ('Approved', 'Valid', 'Rejected', 'Declined')
		`, facilityID).Scan(&count)
	} else {
		err = db.QueryRowContext(ctx, `
			WITH expected AS (
				SELECT e.facility AS facility_id, COUNT(*) AS total_staff
				FROM clinician_app.employees e
				GROUP BY e.facility
			), weekly AS (
				SELECT
					w.hospital AS facility_id,
					COUNT(*) FILTER (WHERE COALESCE(w.submit_status, '') = 'Submitted') AS submitted_count,
					COUNT(*) FILTER (
						WHERE COALESCE(w.submit_status, '') = 'Submitted'
						  AND COALESCE(w.report_status, '') NOT IN ('Approved', 'Valid', 'Rejected', 'Declined')
					) AS pending_count
				FROM clinician_app.weeklyreport w
				WHERE w.start = date_trunc('week', CURRENT_DATE)::date
				GROUP BY w.hospital
			)
			SELECT COUNT(*)
			FROM weekly w
			JOIN expected e ON e.facility_id = w.facility_id
			WHERE w.submitted_count >= e.total_staff
			  AND w.pending_count > 0
		`).Scan(&count)
	}
	if err != nil {
		return 0, nil
	}
	return count, nil
}

// GetMyRecentLeaveUpdates returns the count of leave requests for an employee
// that were reviewed (approved/declined) within the last 7 days.
func GetMyRecentLeaveUpdates(ctx context.Context, db DB, empID int64) (int, error) {
	var count int
	err := db.QueryRowContext(ctx, `
		WITH latest_change AS (
			SELECT reviewed_on
			FROM clinician_app.staffleave
			WHERE employee_id = $1
			  AND COALESCE(leave_status, '') IN ('Approved', 'Valid', 'Rejected', 'Declined')
			  AND reviewed_on IS NOT NULL
			ORDER BY reviewed_on DESC
			LIMIT 1
		)
		SELECT COALESCE(
			(
				SELECT CASE WHEN reviewed_on >= NOW() - INTERVAL '7 days' THEN 1 ELSE 0 END
				FROM latest_change
			),
			0
		)
	`, empID).Scan(&count)
	if err != nil {
		return 0, nil
	}
	return count, nil
}

// GetMyRecentReportUpdates returns the count of reports for an employee
// that were reviewed (approved/declined) within the last 7 days.
func GetMyRecentReportUpdates(ctx context.Context, db DB, empID int64) (int, error) {
	var count int
	err := db.QueryRowContext(ctx, `
		SELECT COUNT(DISTINCT w.id)
		FROM clinician_app.weeklyreport w
		WHERE w.employee = $1
		  AND (
			(
				COALESCE(w.report_status, '') IN ('Approved', 'Rejected', 'Declined')
				AND COALESCE(w.facility_reviewed_on, w.last_updated_on, w.created_on) >= NOW() - INTERVAL '7 days'
			)
			OR (
				COALESCE(w.national_review_status, '') IN ('Approved', 'Rejected', 'Declined')
				AND COALESCE(w.national_reviewed_on, w.last_updated_on, w.created_on) >= NOW() - INTERVAL '7 days'
			)
		  )
	`, empID).Scan(&count)
	if err != nil {
		return 0, nil
	}
	return count, nil
}

// GetMyDataEntryReminderCount returns 1 when a clinician still needs to submit
// last week's report (and is not on approved leave for that week), else 0.
func GetMyDataEntryReminderCount(ctx context.Context, db DB, empID int64) (int, error) {
	var count int
	err := db.QueryRowContext(ctx, `
		WITH due_week AS (
			SELECT
				(date_trunc('week', CURRENT_DATE)::date - INTERVAL '7 days')::date AS week_start,
				(date_trunc('week', CURRENT_DATE)::date - INTERVAL '1 day')::date AS week_stop
		),
		due_on_leave AS (
			SELECT EXISTS (
				SELECT 1
				FROM clinician_app.staffleave sl
				JOIN due_week d ON TRUE
				WHERE sl.employee_id = $1
				  AND COALESCE(sl.leave_status, '') IN ('Approved', 'Valid')
				  AND sl.start_date::date <= d.week_stop
				  AND sl.end_date::date >= d.week_start
			) AS is_on_leave
		),
		due_submitted AS (
			SELECT EXISTS (
				SELECT 1
				FROM clinician_app.weeklyreport w
				JOIN due_week d ON TRUE
				WHERE w.employee = $1
				  AND w.start::date = d.week_start
				  AND w.stop::date = d.week_stop
				  AND COALESCE(w.submit_status, '') = 'Submitted'
			) AS is_submitted
		)
		SELECT CASE
			WHEN ol.is_on_leave THEN 0
			WHEN ds.is_submitted THEN 0
			ELSE 1
		END
		FROM due_on_leave ol
		CROSS JOIN due_submitted ds
	`, empID).Scan(&count)
	if err != nil {
		return 0, nil
	}
	return count, nil
}
