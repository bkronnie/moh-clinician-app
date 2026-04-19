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

type CustomizationFacility struct {
	ID    int64
	Name  string
	Level string
}

type CustomizationDepartment struct {
	ID   int64
	Name string
}

type CustomizationRole struct {
	ID   int64
	Name string
}

type CustomizationClinicalRole struct {
	ID    int64
	Title string
}

type CustomizationChange struct {
	ID        int64
	Entity    string
	EntityID  int64
	Action    string
	Summary   string
	ChangedBy string
	ChangedOn time.Time
}

type CustomizationView struct {
	Facilities    []CustomizationFacility
	Departments   []CustomizationDepartment
	Roles         []CustomizationRole
	ClinicalRoles []CustomizationClinicalRole
	History       []CustomizationChange
	Success       string
	Error         string
}

var immutableRoleNames = map[string]struct{}{
	"admin":    {},
	"approver": {},
	"user":     {},
}

func EnsureCustomizationAuditSchema(ctx context.Context, db *sql.DB) error {
	_, err := db.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS clinician_app.customization_change_log (
			id BIGSERIAL PRIMARY KEY,
			entity_type TEXT NOT NULL,
			entity_id BIGINT,
			action TEXT NOT NULL,
			change_summary TEXT NOT NULL,
			previous_snapshot JSONB NOT NULL DEFAULT '{}'::jsonb,
			new_snapshot JSONB NOT NULL DEFAULT '{}'::jsonb,
			changed_by_user BIGINT REFERENCES clinician_app.users(id),
			changed_by_employee BIGINT REFERENCES clinician_app.employees(id),
			changed_on TIMESTAMP NOT NULL DEFAULT NOW()
		)
	`)
	if err != nil {
		return err
	}

	_, err = db.ExecContext(ctx, `
		CREATE INDEX IF NOT EXISTS idx_customization_change_log_on_changed_on
		ON clinician_app.customization_change_log(changed_on DESC)
	`)
	if err != nil {
		return err
	}

	_, err = db.ExecContext(ctx, `
		CREATE INDEX IF NOT EXISTS idx_customization_change_log_entity
		ON clinician_app.customization_change_log(entity_type, entity_id, changed_on DESC)
	`)
	return err
}

func GetCustomizationView(ctx context.Context, db DB, historyLimit int) (CustomizationView, error) {
	if historyLimit <= 0 {
		historyLimit = 100
	}

	facilities, err := ListCustomizationFacilities(ctx, db)
	if err != nil {
		return CustomizationView{}, err
	}

	departments, err := ListCustomizationDepartments(ctx, db)
	if err != nil {
		return CustomizationView{}, err
	}

	roles, err := ListCustomizationRoles(ctx, db)
	if err != nil {
		return CustomizationView{}, err
	}

	clinicalRoles, err := ListCustomizationClinicalRoles(ctx, db)
	if err != nil {
		return CustomizationView{}, err
	}

	history, err := ListCustomizationHistory(ctx, db, historyLimit)
	if err != nil {
		return CustomizationView{}, err
	}

	return CustomizationView{
		Facilities:    facilities,
		Departments:   departments,
		Roles:         roles,
		ClinicalRoles: clinicalRoles,
		History:       history,
	}, nil
}

func ListCustomizationFacilities(ctx context.Context, db DB) ([]CustomizationFacility, error) {
	rows, err := db.QueryContext(ctx, `
		SELECT id, COALESCE(f_name, ''), COALESCE(f_level, '')
		FROM clinician_app.facilities
		ORDER BY f_name ASC, id ASC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]CustomizationFacility, 0)
	for rows.Next() {
		var item CustomizationFacility
		if err := rows.Scan(&item.ID, &item.Name, &item.Level); err != nil {
			return nil, err
		}
		items = append(items, item)
	}

	return items, rows.Err()
}

func ListCustomizationDepartments(ctx context.Context, db DB) ([]CustomizationDepartment, error) {
	rows, err := db.QueryContext(ctx, `
		SELECT id, COALESCE(d_name, '')
		FROM clinician_app.departments
		ORDER BY d_name ASC, id ASC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]CustomizationDepartment, 0)
	for rows.Next() {
		var item CustomizationDepartment
		if err := rows.Scan(&item.ID, &item.Name); err != nil {
			return nil, err
		}
		items = append(items, item)
	}

	return items, rows.Err()
}

func ListCustomizationRoles(ctx context.Context, db DB) ([]CustomizationRole, error) {
	rows, err := db.QueryContext(ctx, `
		SELECT id, COALESCE(rights, '')
		FROM clinician_app.rights
		ORDER BY rights ASC, id ASC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]CustomizationRole, 0)
	for rows.Next() {
		var item CustomizationRole
		if err := rows.Scan(&item.ID, &item.Name); err != nil {
			return nil, err
		}
		items = append(items, item)
	}

	return items, rows.Err()
}

func ListCustomizationClinicalRoles(ctx context.Context, db DB) ([]CustomizationClinicalRole, error) {
	rows, err := db.QueryContext(ctx, `
		SELECT id, COALESCE(title, '')
		FROM clinician_app.specialist_titles
		ORDER BY id ASC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]CustomizationClinicalRole, 0)
	for rows.Next() {
		var item CustomizationClinicalRole
		if err := rows.Scan(&item.ID, &item.Title); err != nil {
			return nil, err
		}
		items = append(items, item)
	}

	return items, rows.Err()
}

func ListCustomizationHistory(ctx context.Context, db DB, limit int) ([]CustomizationChange, error) {
	if limit <= 0 {
		limit = 100
	}

	rows, err := db.QueryContext(ctx, `
		SELECT
			c.id,
			c.entity_type,
			COALESCE(c.entity_id, 0),
			c.action,
			c.change_summary,
			COALESCE(NULLIF(TRIM(CONCAT(COALESCE(e.fname, ''), ' ', COALESCE(e.lname, ''))), ''), 'System') AS changed_by,
			c.changed_on
		FROM clinician_app.customization_change_log c
		LEFT JOIN clinician_app.employees e ON e.id = c.changed_by_employee
		ORDER BY c.changed_on DESC, c.id DESC
		LIMIT $1
	`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]CustomizationChange, 0)
	for rows.Next() {
		var item CustomizationChange
		if err := rows.Scan(
			&item.ID,
			&item.Entity,
			&item.EntityID,
			&item.Action,
			&item.Summary,
			&item.ChangedBy,
			&item.ChangedOn,
		); err != nil {
			return nil, err
		}
		items = append(items, item)
	}

	return items, rows.Err()
}

func SaveCustomizationFacility(ctx context.Context, db *sql.DB, actorUserID int64, actorEmployeeID int64, id int64, name string, level string) error {
	name = strings.TrimSpace(name)
	level = strings.TrimSpace(level)
	if name == "" {
		return errors.New("facility name is required")
	}
	if level == "" {
		return errors.New("facility level is required")
	}

	if err := EnsureCustomizationAuditSchema(ctx, db); err != nil {
		return err
	}

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if id <= 0 {
		var newID int64
		if err := tx.QueryRowContext(ctx, `
			INSERT INTO clinician_app.facilities (f_name, f_level, created_by, created_on)
			VALUES ($1, $2, $3, NOW())
			RETURNING id
		`, name, level, actorEmployeeID).Scan(&newID); err != nil {
			return err
		}

		newSnapshot := map[string]interface{}{"id": newID, "name": name, "level": level}
		if err := insertCustomizationChange(ctx, tx, actorUserID, actorEmployeeID, "facility", newID, "create", "Facility created", map[string]interface{}{}, newSnapshot); err != nil {
			return err
		}
		return tx.Commit()
	}

	var currentName string
	var currentLevel string
	err = tx.QueryRowContext(ctx, `
		SELECT COALESCE(f_name, ''), COALESCE(f_level, '')
		FROM clinician_app.facilities
		WHERE id = $1
	`, id).Scan(&currentName, &currentLevel)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return errors.New("facility not found")
		}
		return err
	}

	if currentName == name && currentLevel == level {
		return nil
	}

	if _, err := tx.ExecContext(ctx, `
		UPDATE clinician_app.facilities
		SET f_name = $1, f_level = $2
		WHERE id = $3
	`, name, level, id); err != nil {
		return err
	}

	prevSnapshot := map[string]interface{}{"id": id, "name": currentName, "level": currentLevel}
	newSnapshot := map[string]interface{}{"id": id, "name": name, "level": level}
	if err := insertCustomizationChange(ctx, tx, actorUserID, actorEmployeeID, "facility", id, "update", "Facility updated", prevSnapshot, newSnapshot); err != nil {
		return err
	}

	return tx.Commit()
}

func DeleteCustomizationFacility(ctx context.Context, db *sql.DB, actorUserID int64, actorEmployeeID int64, id int64) error {
	if id <= 0 {
		return errors.New("invalid facility id")
	}

	if err := EnsureCustomizationAuditSchema(ctx, db); err != nil {
		return err
	}

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var currentName string
	var currentLevel string
	err = tx.QueryRowContext(ctx, `
		SELECT COALESCE(f_name, ''), COALESCE(f_level, '')
		FROM clinician_app.facilities
		WHERE id = $1
	`, id).Scan(&currentName, &currentLevel)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return errors.New("facility not found")
		}
		return err
	}

	var employeeRefs int
	if err := tx.QueryRowContext(ctx, `SELECT COUNT(1) FROM clinician_app.employees WHERE facility = $1`, id).Scan(&employeeRefs); err != nil {
		return err
	}
	var reportRefs int
	if err := tx.QueryRowContext(ctx, `SELECT COUNT(1) FROM clinician_app.weeklyreport WHERE hospital = $1`, id).Scan(&reportRefs); err != nil {
		return err
	}

	if employeeRefs > 0 || reportRefs > 0 {
		return fmt.Errorf("cannot delete facility; in use by %d employees and %d reports", employeeRefs, reportRefs)
	}

	if _, err := tx.ExecContext(ctx, `DELETE FROM clinician_app.facilities WHERE id = $1`, id); err != nil {
		return err
	}

	prevSnapshot := map[string]interface{}{"id": id, "name": currentName, "level": currentLevel}
	if err := insertCustomizationChange(ctx, tx, actorUserID, actorEmployeeID, "facility", id, "delete", "Facility deleted", prevSnapshot, map[string]interface{}{}); err != nil {
		return err
	}

	return tx.Commit()
}

func SaveCustomizationDepartment(ctx context.Context, db *sql.DB, actorUserID int64, actorEmployeeID int64, id int64, name string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return errors.New("department name is required")
	}

	if err := EnsureCustomizationAuditSchema(ctx, db); err != nil {
		return err
	}

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if id <= 0 {
		var newID int64
		if err := tx.QueryRowContext(ctx, `
			INSERT INTO clinician_app.departments (d_name)
			VALUES ($1)
			RETURNING id
		`, name).Scan(&newID); err != nil {
			return err
		}

		newSnapshot := map[string]interface{}{"id": newID, "name": name}
		if err := insertCustomizationChange(ctx, tx, actorUserID, actorEmployeeID, "department", newID, "create", "Department created", map[string]interface{}{}, newSnapshot); err != nil {
			return err
		}
		return tx.Commit()
	}

	var currentName string
	err = tx.QueryRowContext(ctx, `
		SELECT COALESCE(d_name, '')
		FROM clinician_app.departments
		WHERE id = $1
	`, id).Scan(&currentName)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return errors.New("department not found")
		}
		return err
	}

	if currentName == name {
		return nil
	}

	if _, err := tx.ExecContext(ctx, `
		UPDATE clinician_app.departments
		SET d_name = $1
		WHERE id = $2
	`, name, id); err != nil {
		return err
	}

	prevSnapshot := map[string]interface{}{"id": id, "name": currentName}
	newSnapshot := map[string]interface{}{"id": id, "name": name}
	if err := insertCustomizationChange(ctx, tx, actorUserID, actorEmployeeID, "department", id, "update", "Department updated", prevSnapshot, newSnapshot); err != nil {
		return err
	}

	return tx.Commit()
}

func DeleteCustomizationDepartment(ctx context.Context, db *sql.DB, actorUserID int64, actorEmployeeID int64, id int64) error {
	if id <= 0 {
		return errors.New("invalid department id")
	}

	if err := EnsureCustomizationAuditSchema(ctx, db); err != nil {
		return err
	}

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var currentName string
	err = tx.QueryRowContext(ctx, `
		SELECT COALESCE(d_name, '')
		FROM clinician_app.departments
		WHERE id = $1
	`, id).Scan(&currentName)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return errors.New("department not found")
		}
		return err
	}

	var employeeRefs int
	if err := tx.QueryRowContext(ctx, `SELECT COUNT(1) FROM clinician_app.employees WHERE department = $1`, id).Scan(&employeeRefs); err != nil {
		return err
	}
	var reportRefs int
	if err := tx.QueryRowContext(ctx, `SELECT COUNT(1) FROM clinician_app.weeklyreport WHERE department = $1`, id).Scan(&reportRefs); err != nil {
		return err
	}
	var roleRefs int
	if err := tx.QueryRowContext(ctx, `SELECT COUNT(1) FROM clinician_app.department_roles WHERE dept_id = $1`, id).Scan(&roleRefs); err != nil {
		return err
	}

	if employeeRefs > 0 || reportRefs > 0 || roleRefs > 0 {
		return fmt.Errorf("cannot delete department; in use by %d employees, %d reports, %d role mappings", employeeRefs, reportRefs, roleRefs)
	}

	if _, err := tx.ExecContext(ctx, `DELETE FROM clinician_app.departments WHERE id = $1`, id); err != nil {
		return err
	}

	prevSnapshot := map[string]interface{}{"id": id, "name": currentName}
	if err := insertCustomizationChange(ctx, tx, actorUserID, actorEmployeeID, "department", id, "delete", "Department deleted", prevSnapshot, map[string]interface{}{}); err != nil {
		return err
	}

	return tx.Commit()
}

func SaveCustomizationRole(ctx context.Context, db *sql.DB, actorUserID int64, actorEmployeeID int64, id int64, name string) error {
	name = strings.ToLower(strings.TrimSpace(name))
	if name == "" {
		return errors.New("role name is required")
	}

	if err := EnsureCustomizationAuditSchema(ctx, db); err != nil {
		return err
	}

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if id <= 0 {
		var newID int64
		if err := tx.QueryRowContext(ctx, `
			INSERT INTO clinician_app.rights (rights)
			VALUES ($1)
			RETURNING id
		`, name).Scan(&newID); err != nil {
			return err
		}

		newSnapshot := map[string]interface{}{"id": newID, "name": name}
		if err := insertCustomizationChange(ctx, tx, actorUserID, actorEmployeeID, "role", newID, "create", "Role created", map[string]interface{}{}, newSnapshot); err != nil {
			return err
		}
		return tx.Commit()
	}

	var currentName string
	err = tx.QueryRowContext(ctx, `
		SELECT COALESCE(rights, '')
		FROM clinician_app.rights
		WHERE id = $1
	`, id).Scan(&currentName)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return errors.New("role not found")
		}
		return err
	}

	currentNorm := strings.ToLower(strings.TrimSpace(currentName))
	if _, locked := immutableRoleNames[currentNorm]; locked && currentNorm != name {
		return fmt.Errorf("core role '%s' cannot be renamed", currentNorm)
	}

	if currentNorm == name {
		return nil
	}

	if _, err := tx.ExecContext(ctx, `
		UPDATE clinician_app.rights
		SET rights = $1
		WHERE id = $2
	`, name, id); err != nil {
		return err
	}

	prevSnapshot := map[string]interface{}{"id": id, "name": currentName}
	newSnapshot := map[string]interface{}{"id": id, "name": name}
	if err := insertCustomizationChange(ctx, tx, actorUserID, actorEmployeeID, "role", id, "update", "Role updated", prevSnapshot, newSnapshot); err != nil {
		return err
	}

	return tx.Commit()
}

func DeleteCustomizationRole(ctx context.Context, db *sql.DB, actorUserID int64, actorEmployeeID int64, id int64) error {
	if id <= 0 {
		return errors.New("invalid role id")
	}

	if err := EnsureCustomizationAuditSchema(ctx, db); err != nil {
		return err
	}

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var currentName string
	err = tx.QueryRowContext(ctx, `
		SELECT COALESCE(rights, '')
		FROM clinician_app.rights
		WHERE id = $1
	`, id).Scan(&currentName)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return errors.New("role not found")
		}
		return err
	}

	currentNorm := strings.ToLower(strings.TrimSpace(currentName))
	if _, locked := immutableRoleNames[currentNorm]; locked {
		return fmt.Errorf("core role '%s' cannot be deleted", currentNorm)
	}

	var roleRefs int
	if err := tx.QueryRowContext(ctx, `SELECT COUNT(1) FROM clinician_app.employeerights WHERE rights = $1`, id).Scan(&roleRefs); err != nil {
		return err
	}
	if roleRefs > 0 {
		return fmt.Errorf("cannot delete role; used by %d employee-right mappings", roleRefs)
	}

	if _, err := tx.ExecContext(ctx, `DELETE FROM clinician_app.rights WHERE id = $1`, id); err != nil {
		return err
	}

	prevSnapshot := map[string]interface{}{"id": id, "name": currentName}
	if err := insertCustomizationChange(ctx, tx, actorUserID, actorEmployeeID, "role", id, "delete", "Role deleted", prevSnapshot, map[string]interface{}{}); err != nil {
		return err
	}

	return tx.Commit()
}

func SaveCustomizationClinicalRole(ctx context.Context, db *sql.DB, actorUserID int64, actorEmployeeID int64, id int64, title string) error {
	title = strings.TrimSpace(title)
	if title == "" {
		return errors.New("clinical role title is required")
	}

	if err := EnsureCustomizationAuditSchema(ctx, db); err != nil {
		return err
	}

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if id <= 0 {
		var nextID int64
		if err := tx.QueryRowContext(ctx, `SELECT COALESCE(MAX(id), 0) + 1 FROM clinician_app.specialist_titles`).Scan(&nextID); err != nil {
			return err
		}

		if _, err := tx.ExecContext(ctx, `
			INSERT INTO clinician_app.specialist_titles (id, title)
			VALUES ($1, $2)
		`, nextID, title); err != nil {
			return err
		}

		newSnapshot := map[string]interface{}{"id": nextID, "title": title}
		if err := insertCustomizationChange(ctx, tx, actorUserID, actorEmployeeID, "clinical_role", nextID, "create", "Clinical role created", map[string]interface{}{}, newSnapshot); err != nil {
			return err
		}

		return tx.Commit()
	}

	var currentTitle string
	err = tx.QueryRowContext(ctx, `
		SELECT COALESCE(title, '')
		FROM clinician_app.specialist_titles
		WHERE id = $1
	`, id).Scan(&currentTitle)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return errors.New("clinical role not found")
		}
		return err
	}

	if currentTitle == title {
		return nil
	}

	if _, err := tx.ExecContext(ctx, `
		UPDATE clinician_app.specialist_titles
		SET title = $1
		WHERE id = $2
	`, title, id); err != nil {
		return err
	}

	prevSnapshot := map[string]interface{}{"id": id, "title": currentTitle}
	newSnapshot := map[string]interface{}{"id": id, "title": title}
	if err := insertCustomizationChange(ctx, tx, actorUserID, actorEmployeeID, "clinical_role", id, "update", "Clinical role updated", prevSnapshot, newSnapshot); err != nil {
		return err
	}

	return tx.Commit()
}

func DeleteCustomizationClinicalRole(ctx context.Context, db *sql.DB, actorUserID int64, actorEmployeeID int64, id int64) error {
	if id <= 0 {
		return errors.New("invalid clinical role id")
	}

	if err := EnsureCustomizationAuditSchema(ctx, db); err != nil {
		return err
	}

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var currentTitle string
	err = tx.QueryRowContext(ctx, `
		SELECT COALESCE(title, '')
		FROM clinician_app.specialist_titles
		WHERE id = $1
	`, id).Scan(&currentTitle)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return errors.New("clinical role not found")
		}
		return err
	}

	var employeeRefs int
	if err := tx.QueryRowContext(ctx, `SELECT COUNT(1) FROM clinician_app.employees WHERE title = $1`, id).Scan(&employeeRefs); err != nil {
		return err
	}
	if employeeRefs > 0 {
		return fmt.Errorf("cannot delete clinical role; used by %d employees", employeeRefs)
	}

	if _, err := tx.ExecContext(ctx, `DELETE FROM clinician_app.specialist_titles WHERE id = $1`, id); err != nil {
		return err
	}

	prevSnapshot := map[string]interface{}{"id": id, "title": currentTitle}
	if err := insertCustomizationChange(ctx, tx, actorUserID, actorEmployeeID, "clinical_role", id, "delete", "Clinical role deleted", prevSnapshot, map[string]interface{}{}); err != nil {
		return err
	}

	return tx.Commit()
}

func insertCustomizationChange(ctx context.Context, tx *sql.Tx, actorUserID int64, actorEmployeeID int64, entityType string, entityID int64, action string, summary string, previous map[string]interface{}, next map[string]interface{}) error {
	if previous == nil {
		previous = map[string]interface{}{}
	}
	if next == nil {
		next = map[string]interface{}{}
	}

	prevBytes, err := json.Marshal(previous)
	if err != nil {
		return err
	}
	nextBytes, err := json.Marshal(next)
	if err != nil {
		return err
	}

	_, err = tx.ExecContext(ctx, `
		INSERT INTO clinician_app.customization_change_log (
			entity_type,
			entity_id,
			action,
			change_summary,
			previous_snapshot,
			new_snapshot,
			changed_by_user,
			changed_by_employee,
			changed_on
		) VALUES ($1, $2, $3, $4, $5::jsonb, $6::jsonb, $7, $8, NOW())
	`, entityType, entityID, action, summary, string(prevBytes), string(nextBytes), actorUserID, actorEmployeeID)

	return err
}
