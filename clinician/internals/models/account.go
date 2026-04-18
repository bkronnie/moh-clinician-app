package models

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
)

type AccountProfile struct {
	UserID         int64
	EmployeeID     int64
	Email          string
	FirstName      string
	LastName       string
	OtherName      string
	EmployeeNumber string
	DateOfBirth    sql.NullTime
	PhoneNumber    string
	FacilityID     int64
	FacilityName   string
	DepartmentID   int64
	DepartmentName string
	TitleID        int64
	TitleName      string
	Specialisation string
}

type AccountProfileInput struct {
	Email          string
	FirstName      string
	LastName       string
	OtherName      string
	EmployeeNumber string
	DateOfBirth    *time.Time
	PhoneNumber    string
	FacilityID     int64
	DepartmentID   int64
	TitleID        int64
	Specialisation string
}

type AccountProfileChange struct {
	ID        int64
	ChangedOn time.Time
	ChangedBy string
	Summary   string
}

type accountProfileSnapshot struct {
	Email          string `json:"email"`
	FirstName      string `json:"first_name"`
	LastName       string `json:"last_name"`
	OtherName      string `json:"other_name"`
	EmployeeNumber string `json:"employee_number"`
	DateOfBirth    string `json:"date_of_birth"`
	PhoneNumber    string `json:"phone_number"`
	FacilityID     int64  `json:"facility_id"`
	FacilityName   string `json:"facility_name"`
	DepartmentID   int64  `json:"department_id"`
	DepartmentName string `json:"department_name"`
	TitleID        int64  `json:"title_id"`
	TitleName      string `json:"title_name"`
	Specialisation string `json:"specialisation"`
}

func EnsureEmployeeProfileChangeLog(ctx context.Context, db DB) error {
	const sqlstr = `
		CREATE TABLE IF NOT EXISTS clinician_app.employee_profile_changes (
			id BIGSERIAL PRIMARY KEY,
			employee_id BIGINT NOT NULL REFERENCES clinician_app.employees(id),
			changed_by_user BIGINT REFERENCES clinician_app.users(id),
			changed_by_employee BIGINT REFERENCES clinician_app.employees(id),
			changed_on TIMESTAMP NOT NULL DEFAULT NOW(),
			change_summary TEXT NOT NULL,
			previous_snapshot JSONB NOT NULL DEFAULT '{}'::jsonb,
			new_snapshot JSONB NOT NULL DEFAULT '{}'::jsonb
		)
	`
	if _, err := db.ExecContext(ctx, sqlstr); err != nil {
		return err
	}

	const indexSQL = `
		CREATE INDEX IF NOT EXISTS idx_employee_profile_changes_employee_changed_on
		ON clinician_app.employee_profile_changes(employee_id, changed_on DESC)
	`
	_, err := db.ExecContext(ctx, indexSQL)
	return err
}

func GetAccountProfile(ctx context.Context, db DB, userID int64) (AccountProfile, error) {
	const sqlstr = `
		SELECT
			u.id,
			e.id,
			COALESCE(u.username, ''),
			COALESCE(e.fname, ''),
			COALESCE(e.lname, ''),
			COALESCE(e.oname, ''),
			COALESCE(e.employee_number, ''),
			e.date_of_birth,
			COALESCE(e.phone_number, ''),
			COALESCE(e.facility, 0),
			COALESCE(f.f_name, ''),
			COALESCE(e.department, 0),
			COALESCE(d.d_name, ''),
			COALESCE(e.title, 0),
			COALESCE(st.title, ''),
			COALESCE(e.specialisation, '')
		FROM clinician_app.users u
		JOIN clinician_app.employees e ON e.id = u.employees
		LEFT JOIN clinician_app.facilities f ON f.id = e.facility
		LEFT JOIN clinician_app.departments d ON d.id = e.department
		LEFT JOIN clinician_app.specialist_titles st ON st.id = e.title
		WHERE u.id = $1
	`

	var profile AccountProfile
	if err := db.QueryRowContext(ctx, sqlstr, userID).Scan(
		&profile.UserID,
		&profile.EmployeeID,
		&profile.Email,
		&profile.FirstName,
		&profile.LastName,
		&profile.OtherName,
		&profile.EmployeeNumber,
		&profile.DateOfBirth,
		&profile.PhoneNumber,
		&profile.FacilityID,
		&profile.FacilityName,
		&profile.DepartmentID,
		&profile.DepartmentName,
		&profile.TitleID,
		&profile.TitleName,
		&profile.Specialisation,
	); err != nil {
		return AccountProfile{}, err
	}

	return profile, nil
}

func GetRecentAccountProfileChanges(ctx context.Context, db DB, employeeID int64, limit int) ([]AccountProfileChange, error) {
	if limit <= 0 {
		limit = 8
	}

	if err := EnsureEmployeeProfileChangeLog(ctx, db); err != nil {
		return nil, err
	}

	const sqlstr = `
		SELECT
			c.id,
			c.changed_on,
			COALESCE(NULLIF(TRIM(CONCAT(COALESCE(e.fname, ''), ' ', COALESCE(e.lname, ''))), ''), 'System'),
			c.change_summary
		FROM clinician_app.employee_profile_changes c
		LEFT JOIN clinician_app.employees e ON e.id = c.changed_by_employee
		WHERE c.employee_id = $1
		ORDER BY c.changed_on DESC, c.id DESC
		LIMIT $2
	`

	rows, err := db.QueryContext(ctx, sqlstr, employeeID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	changes := []AccountProfileChange{}
	for rows.Next() {
		var item AccountProfileChange
		if err := rows.Scan(&item.ID, &item.ChangedOn, &item.ChangedBy, &item.Summary); err != nil {
			return nil, err
		}
		changes = append(changes, item)
	}

	return changes, rows.Err()
}

func UpdateOwnAccountProfile(ctx context.Context, db *sql.DB, actorUserID int64, actorEmployeeID int64, input AccountProfileInput) error {
	if err := EnsureEmployeeProfileChangeLog(ctx, db); err != nil {
		return err
	}

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	current, err := getAccountProfileForUpdate(ctx, tx, actorUserID)
	if err != nil {
		return err
	}

	if err := validateAccountProfileUpdate(ctx, tx, current, input); err != nil {
		return err
	}

	next := current
	next.Email = strings.ToLower(strings.TrimSpace(input.Email))
	next.FirstName = strings.TrimSpace(input.FirstName)
	next.LastName = strings.TrimSpace(input.LastName)
	next.OtherName = strings.TrimSpace(input.OtherName)
	next.EmployeeNumber = strings.TrimSpace(input.EmployeeNumber)
	next.PhoneNumber = strings.TrimSpace(input.PhoneNumber)
	next.FacilityID = input.FacilityID
	next.DepartmentID = input.DepartmentID
	next.TitleID = input.TitleID
	next.Specialisation = strings.TrimSpace(input.Specialisation)
	next.DateOfBirth = sql.NullTime{}
	if input.DateOfBirth != nil {
		next.DateOfBirth = sql.NullTime{Time: *input.DateOfBirth, Valid: true}
	}

	if next.FacilityID > 0 {
		next.FacilityName, err = lookupFacilityName(ctx, tx, next.FacilityID)
		if err != nil {
			return err
		}
	}
	if next.DepartmentID > 0 {
		next.DepartmentName, err = lookupDepartmentName(ctx, tx, next.DepartmentID)
		if err != nil {
			return err
		}
	}
	if next.TitleID > 0 {
		next.TitleName, err = lookupTitleName(ctx, tx, next.TitleID)
		if err != nil {
			return err
		}
	} else {
		next.TitleName = ""
	}

	if _, err := tx.ExecContext(ctx, `
		UPDATE clinician_app.users
		SET username = $1
		WHERE id = $2
	`, next.Email, current.UserID); err != nil {
		return err
	}

	var dateOfBirth interface{}
	if next.DateOfBirth.Valid {
		dateOfBirth = next.DateOfBirth.Time
	}

	var titleValue interface{}
	if next.TitleID > 0 {
		titleValue = next.TitleID
	}

	if _, err := tx.ExecContext(ctx, `
		UPDATE clinician_app.employees
		SET
			fname = $1,
			lname = $2,
			oname = NULLIF($3, ''),
			specialisation = NULLIF($4, ''),
			department = $5,
			facility = $6,
			title = $7,
			employee_number = NULLIF($8, ''),
			date_of_birth = $9,
			phone_number = NULLIF($10, '')
		WHERE id = $11
	`,
		next.FirstName,
		next.LastName,
		next.OtherName,
		next.Specialisation,
		next.DepartmentID,
		next.FacilityID,
		titleValue,
		next.EmployeeNumber,
		dateOfBirth,
		next.PhoneNumber,
		current.EmployeeID,
	); err != nil {
		return err
	}

	changes := describeAccountProfileChanges(current, next)
	if len(changes) > 0 {
		previousSnapshot, err := json.Marshal(snapshotAccountProfile(current))
		if err != nil {
			return err
		}
		newSnapshot, err := json.Marshal(snapshotAccountProfile(next))
		if err != nil {
			return err
		}

		if _, err := tx.ExecContext(ctx, `
			INSERT INTO clinician_app.employee_profile_changes (
				employee_id,
				changed_by_user,
				changed_by_employee,
				change_summary,
				previous_snapshot,
				new_snapshot
			) VALUES ($1, $2, $3, $4, $5::jsonb, $6::jsonb)
		`,
			current.EmployeeID,
			actorUserID,
			actorEmployeeID,
			strings.Join(changes, "; "),
			string(previousSnapshot),
			string(newSnapshot),
		); err != nil {
			return err
		}
	}

	return tx.Commit()
}

func getAccountProfileForUpdate(ctx context.Context, db DB, userID int64) (AccountProfile, error) {
	const sqlstr = `
		SELECT
			u.id,
			e.id,
			COALESCE(u.username, ''),
			COALESCE(e.fname, ''),
			COALESCE(e.lname, ''),
			COALESCE(e.oname, ''),
			COALESCE(e.employee_number, ''),
			e.date_of_birth,
			COALESCE(e.phone_number, ''),
			COALESCE(e.facility, 0),
			COALESCE(f.f_name, ''),
			COALESCE(e.department, 0),
			COALESCE(d.d_name, ''),
			COALESCE(e.title, 0),
			COALESCE(st.title, ''),
			COALESCE(e.specialisation, '')
		FROM clinician_app.users u
		JOIN clinician_app.employees e ON e.id = u.employees
		LEFT JOIN clinician_app.facilities f ON f.id = e.facility
		LEFT JOIN clinician_app.departments d ON d.id = e.department
		LEFT JOIN clinician_app.specialist_titles st ON st.id = e.title
		WHERE u.id = $1
		FOR UPDATE OF u, e
	`

	var profile AccountProfile
	if err := db.QueryRowContext(ctx, sqlstr, userID).Scan(
		&profile.UserID,
		&profile.EmployeeID,
		&profile.Email,
		&profile.FirstName,
		&profile.LastName,
		&profile.OtherName,
		&profile.EmployeeNumber,
		&profile.DateOfBirth,
		&profile.PhoneNumber,
		&profile.FacilityID,
		&profile.FacilityName,
		&profile.DepartmentID,
		&profile.DepartmentName,
		&profile.TitleID,
		&profile.TitleName,
		&profile.Specialisation,
	); err != nil {
		return AccountProfile{}, err
	}

	return profile, nil
}

func validateAccountProfileUpdate(ctx context.Context, db DB, current AccountProfile, input AccountProfileInput) error {
	email := strings.ToLower(strings.TrimSpace(input.Email))
	firstName := strings.TrimSpace(input.FirstName)
	lastName := strings.TrimSpace(input.LastName)

	switch {
	case firstName == "":
		return fmt.Errorf("first name is required")
	case lastName == "":
		return fmt.Errorf("last name is required")
	case email == "":
		return fmt.Errorf("email address is required")
	case input.FacilityID == 0:
		return fmt.Errorf("please select a facility")
	case input.DepartmentID == 0:
		return fmt.Errorf("please select a department")
	}

	var count int
	if err := db.QueryRowContext(ctx, `
		SELECT COUNT(1)
		FROM clinician_app.users
		WHERE LOWER(username) = LOWER($1) AND id <> $2
	`, email, current.UserID).Scan(&count); err != nil {
		return err
	}
	if count > 0 {
		return fmt.Errorf("an account with that email already exists")
	}

	employeeNumber := strings.TrimSpace(input.EmployeeNumber)
	if employeeNumber != "" {
		if err := db.QueryRowContext(ctx, `
			SELECT COUNT(1)
			FROM clinician_app.employees
			WHERE employee_number = $1 AND id <> $2
		`, employeeNumber, current.EmployeeID).Scan(&count); err != nil {
			return err
		}
		if count > 0 {
			return fmt.Errorf("that medical officer ID is already registered")
		}
	}

	return nil
}

func lookupFacilityName(ctx context.Context, db DB, facilityID int64) (string, error) {
	var name string
	if err := db.QueryRowContext(ctx, `
		SELECT COALESCE(f_name, '')
		FROM clinician_app.facilities
		WHERE id = $1
	`, facilityID).Scan(&name); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", fmt.Errorf("please select a valid facility")
		}
		return "", err
	}
	return name, nil
}

func lookupDepartmentName(ctx context.Context, db DB, departmentID int64) (string, error) {
	var name string
	if err := db.QueryRowContext(ctx, `
		SELECT COALESCE(d_name, '')
		FROM clinician_app.departments
		WHERE id = $1
	`, departmentID).Scan(&name); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", fmt.Errorf("please select a valid department")
		}
		return "", err
	}
	return name, nil
}

func lookupTitleName(ctx context.Context, db DB, titleID int64) (string, error) {
	var name string
	if err := db.QueryRowContext(ctx, `
		SELECT COALESCE(title, '')
		FROM clinician_app.specialist_titles
		WHERE id = $1
	`, titleID).Scan(&name); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", fmt.Errorf("please select a valid specialist title")
		}
		return "", err
	}
	return name, nil
}

func snapshotAccountProfile(profile AccountProfile) accountProfileSnapshot {
	dateOfBirth := ""
	if profile.DateOfBirth.Valid {
		dateOfBirth = profile.DateOfBirth.Time.Format("2006-01-02")
	}

	return accountProfileSnapshot{
		Email:          profile.Email,
		FirstName:      profile.FirstName,
		LastName:       profile.LastName,
		OtherName:      profile.OtherName,
		EmployeeNumber: profile.EmployeeNumber,
		DateOfBirth:    dateOfBirth,
		PhoneNumber:    profile.PhoneNumber,
		FacilityID:     profile.FacilityID,
		FacilityName:   profile.FacilityName,
		DepartmentID:   profile.DepartmentID,
		DepartmentName: profile.DepartmentName,
		TitleID:        profile.TitleID,
		TitleName:      profile.TitleName,
		Specialisation: profile.Specialisation,
	}
}

func describeAccountProfileChanges(current AccountProfile, next AccountProfile) []string {
	changes := []string{}

	appendChange := func(label string, before string, after string) {
		if normalizeChangeValue(before) == normalizeChangeValue(after) {
			return
		}
		changes = append(changes, fmt.Sprintf("%s: %s -> %s", label, presentChangeValue(before), presentChangeValue(after)))
	}

	appendChange("Email", current.Email, next.Email)
	appendChange("First name", current.FirstName, next.FirstName)
	appendChange("Last name", current.LastName, next.LastName)
	appendChange("Other name", current.OtherName, next.OtherName)
	appendChange("Medical officer ID", current.EmployeeNumber, next.EmployeeNumber)
	appendChange("Phone number", current.PhoneNumber, next.PhoneNumber)
	appendChange("Facility", current.FacilityName, next.FacilityName)
	appendChange("Department", current.DepartmentName, next.DepartmentName)
	appendChange("Title", current.TitleName, next.TitleName)
	appendChange("Specialisation", current.Specialisation, next.Specialisation)

	currentDOB := ""
	if current.DateOfBirth.Valid {
		currentDOB = current.DateOfBirth.Time.Format("2006-01-02")
	}
	nextDOB := ""
	if next.DateOfBirth.Valid {
		nextDOB = next.DateOfBirth.Time.Format("2006-01-02")
	}
	appendChange("Date of birth", currentDOB, nextDOB)

	return changes
}

func normalizeChangeValue(value string) string {
	return strings.TrimSpace(strings.ToLower(value))
}

func presentChangeValue(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "Not set"
	}
	return value
}
