package models

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"sort"
	"strings"
)

type ReportDataElement struct {
	ID          int64
	Position    int
	ElementKey  string
	ColumnName  string
	DisplayName string
	IsCore      bool
	IsActive    bool
}

type defaultReportDataElement struct {
	Position    int
	ElementKey  string
	ColumnName  string
	DisplayName string
	IsCore      bool
}

var reportIdentifierPattern = regexp.MustCompile(`^[a-z][a-z0-9_]{1,62}$`)

var defaultReportDataElements = []defaultReportDataElement{
	{1, "attendance", "attendance", "Attendance Days", true},
	{2, "ward_rounds", "ward_rounds", "Ward Rounds", true},
	{3, "patients_reviewed", "patients_reviewed", "Patients Reviewed", true},
	{4, "theatre_days", "theatre_days", "Theatre Days", true},
	{5, "elective", "elective", "Elective Procedures", true},
	{6, "emergency", "emergency", "Emergency Procedures", true},
	{7, "postmortems", "postmortems", "Postmortems", true},
	{8, "OPD_clinics", "opd_clinics", "OPD Clinics", true},
	{9, "OPD_patients", "opd_patients", "OPD Patients", true},
	{10, "anc_patients", "anc_patients", "ANC Patients", true},
	{11, "teaching_rounds", "teaching_rounds", "Teaching Rounds", true},
	{12, "students_taught", "students_taught", "Students Taught", true},
	{13, "mortality_reviews", "mortality_reviews", "Mortality Reviews", true},
	{14, "maternal", "maternal", "Maternal Reviews", true},
	{15, "perinatal", "perinatal", "Perinatal Reviews", true},
	{16, "surgical", "surgical", "Surgical Reviews", true},
	{17, "medical", "medical", "Medical Reviews", true},
	{18, "paed", "paed", "Paediatric Reviews", true},
	{19, "labs_requests", "labs_requests", "Lab Requests", true},
	{20, "imaging_requests", "imaging_requests", "Imaging Requests", true},
	{21, "lab_investigations", "lab_investigations", "Lab Investigations", true},
	{22, "BS", "bs", "BS Tests", true},
	{23, "HIV", "hiv", "HIV Tests", true},
	{24, "malaria", "malaria", "Malaria Cases Reviewed", true},
	{25, "TB", "tb", "TB Cases Reviewed", true},
	{26, "CBC", "cbc", "CBC Tests", true},
	{27, "chemistry", "chemistry", "Chemistry Tests", true},
	{28, "hematology", "hematology", "Hematology Tests", true},
	{29, "urinalysis", "urinalysis", "Urinalysis", true},
	{30, "gram_stain", "gram_stain", "Gram Stain", true},
	{31, "culture", "culture", "Culture", true},
	{32, "microbiology", "microbiology", "Microbiology", true},
	{33, "sensitivity_tests", "sensitivity_tests", "Sensitivity Tests", true},
	{34, "diagnostics", "diagnostics", "Diagnostics", true},
	{35, "xrays", "xrays", "X-Rays", true},
	{36, "ct_scans", "ct_scans", "CT Scans", true},
	{37, "obstetrics_scans", "obstetrics_scans", "Obstetrics Scans", true},
	{38, "abdominal_scans", "abdominal_scans", "Abdominal Scans", true},
	{39, "custom_metric_39", "custom_metric_39", "Custom Metric 39", false},
	{40, "custom_metric_40", "custom_metric_40", "Custom Metric 40", false},
}

var weeklyreportLegacyColumns = []struct {
	Legacy string
	Named  string
}{
	{"qn_01", "attendance"},
	{"qn_02", "ward_rounds"},
	{"qn_03", "patients_reviewed"},
	{"qn_04", "theatre_days"},
	{"qn_05", "elective"},
	{"qn_06", "emergency"},
	{"qn_07", "postmortems"},
	{"qn_08", "opd_clinics"},
	{"qn_09", "opd_patients"},
	{"qn_10", "anc_patients"},
	{"qn_11", "teaching_rounds"},
	{"qn_12", "students_taught"},
	{"qn_13", "mortality_reviews"},
	{"qn_14", "maternal"},
	{"qn_15", "perinatal"},
	{"qn_16", "surgical"},
	{"qn_17", "medical"},
	{"qn_18", "paed"},
	{"qn_19", "labs_requests"},
	{"qn_20", "imaging_requests"},
	{"qn_21", "lab_investigations"},
	{"qn_22", "bs"},
	{"qn_23", "hiv"},
	{"qn_24", "malaria"},
	{"qn_25", "tb"},
	{"qn_26", "cbc"},
	{"qn_27", "chemistry"},
	{"qn_28", "hematology"},
	{"qn_29", "urinalysis"},
	{"qn_30", "gram_stain"},
	{"qn_31", "culture"},
	{"qn_32", "microbiology"},
	{"qn_33", "sensitivity_tests"},
	{"qn_34", "diagnostics"},
	{"qn_35", "xrays"},
	{"qn_36", "ct_scans"},
	{"qn_37", "obstetrics_scans"},
	{"qn_38", "abdominal_scans"},
	{"qn_39", "custom_metric_39"},
	{"qn_40", "custom_metric_40"},
}

func EnsureReportDataElementSchema(ctx context.Context, db *sql.DB) error {
	if _, err := db.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS clinician_app.report_data_elements (
			id BIGSERIAL PRIMARY KEY,
			position INT NOT NULL,
			element_key TEXT NOT NULL UNIQUE,
			column_name TEXT NOT NULL UNIQUE,
			display_name TEXT NOT NULL,
			is_core BOOLEAN NOT NULL DEFAULT FALSE,
			is_active BOOLEAN NOT NULL DEFAULT TRUE,
			created_on TIMESTAMP NOT NULL DEFAULT NOW(),
			updated_on TIMESTAMP NOT NULL DEFAULT NOW()
		)
	`); err != nil {
		return err
	}

	if err := migrateLegacyWeeklyreportColumns(ctx, db); err != nil {
		return err
	}

	for _, item := range defaultReportDataElements {
		if _, err := db.ExecContext(ctx, `
			INSERT INTO clinician_app.report_data_elements (position, element_key, column_name, display_name, is_core, is_active, created_on, updated_on)
			VALUES ($1, $2, $3, $4, $5, TRUE, NOW(), NOW())
			ON CONFLICT (element_key)
			DO UPDATE SET
				position = EXCLUDED.position,
				column_name = EXCLUDED.column_name,
				display_name = EXCLUDED.display_name,
				is_core = EXCLUDED.is_core,
				is_active = TRUE,
				updated_on = NOW()
		`, item.Position, item.ElementKey, item.ColumnName, item.DisplayName, item.IsCore); err != nil {
			return err
		}

		if _, err := db.ExecContext(ctx, `ALTER TABLE clinician_app.weeklyreport ADD COLUMN IF NOT EXISTS `+item.ColumnName+` BIGINT`); err != nil {
			return err
		}
	}

	if _, err := db.ExecContext(ctx, `
		CREATE INDEX IF NOT EXISTS idx_report_data_elements_active_position
		ON clinician_app.report_data_elements(is_active, position)
	`); err != nil {
		return err
	}

	return nil
}

func migrateLegacyWeeklyreportColumns(ctx context.Context, db *sql.DB) error {
	for _, item := range weeklyreportLegacyColumns {
		stmt := fmt.Sprintf(`
			DO $$
			BEGIN
				IF EXISTS (
					SELECT 1 FROM information_schema.columns
					WHERE table_schema = 'clinician_app'
					  AND table_name = 'weeklyreport'
					  AND column_name = '%s'
				) AND NOT EXISTS (
					SELECT 1 FROM information_schema.columns
					WHERE table_schema = 'clinician_app'
					  AND table_name = 'weeklyreport'
					  AND column_name = '%s'
				) THEN
					ALTER TABLE clinician_app.weeklyreport RENAME COLUMN %s TO %s;
				END IF;
			END $$;
		`, item.Legacy, item.Named, item.Legacy, item.Named)
		if _, err := db.ExecContext(ctx, stmt); err != nil {
			return err
		}
	}
	return nil
}

func ListReportDataElements(ctx context.Context, db DB) ([]ReportDataElement, error) {
	rows, err := db.QueryContext(ctx, `
		SELECT id, position, element_key, column_name, display_name, is_core, is_active
		FROM clinician_app.report_data_elements
		ORDER BY position ASC, id ASC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]ReportDataElement, 0)
	for rows.Next() {
		var item ReportDataElement
		if err := rows.Scan(&item.ID, &item.Position, &item.ElementKey, &item.ColumnName, &item.DisplayName, &item.IsCore, &item.IsActive); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func GetReportDataElementDisplayMap(ctx context.Context, db DB) (map[string]string, error) {
	items, err := ListReportDataElements(ctx, db)
	if err != nil {
		return nil, err
	}

	labels := map[string]string{}
	for _, item := range items {
		if !item.IsActive {
			continue
		}
		labels[item.ElementKey] = item.DisplayName
	}
	return labels, nil
}

func SaveReportDataElement(ctx context.Context, db *sql.DB, actorUserID int64, actorEmployeeID int64, id int64, elementKey string, columnName string, displayName string) error {
	elementKey = strings.TrimSpace(elementKey)
	columnName = normalizeReportIdentifier(columnName)
	displayName = strings.TrimSpace(displayName)
	if displayName == "" {
		return errors.New("display name is required")
	}
	if id <= 0 {
		if elementKey == "" {
			return errors.New("element key is required")
		}
		if !reportIdentifierPattern.MatchString(normalizeReportIdentifier(elementKey)) {
			return errors.New("element key must be snake_case")
		}
		elementKey = strings.TrimSpace(elementKey)
	}
	if !reportIdentifierPattern.MatchString(columnName) {
		return errors.New("column name must be snake_case")
	}

	if err := EnsureCustomizationAuditSchema(ctx, db); err != nil {
		return err
	}
	if err := EnsureReportDataElementSchema(ctx, db); err != nil {
		return err
	}

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if id <= 0 {
		var maxPosition int
		if err := tx.QueryRowContext(ctx, `SELECT COALESCE(MAX(position), 0) FROM clinician_app.report_data_elements`).Scan(&maxPosition); err != nil {
			return err
		}
		position := maxPosition + 1

		if _, err := tx.ExecContext(ctx, `ALTER TABLE clinician_app.weeklyreport ADD COLUMN IF NOT EXISTS `+columnName+` BIGINT`); err != nil {
			return err
		}

		var newID int64
		if err := tx.QueryRowContext(ctx, `
			INSERT INTO clinician_app.report_data_elements (position, element_key, column_name, display_name, is_core, is_active, created_on, updated_on)
			VALUES ($1, $2, $3, $4, FALSE, TRUE, NOW(), NOW())
			RETURNING id
		`, position, elementKey, columnName, displayName).Scan(&newID); err != nil {
			return err
		}

		newSnapshot := map[string]interface{}{"id": newID, "key": elementKey, "column": columnName, "display_name": displayName}
		if err := insertCustomizationChange(ctx, tx, actorUserID, actorEmployeeID, "report_data_element", newID, "create", "Report data element created", map[string]interface{}{}, newSnapshot); err != nil {
			return err
		}
		return tx.Commit()
	}

	var current ReportDataElement
	if err := tx.QueryRowContext(ctx, `
		SELECT id, position, element_key, column_name, display_name, is_core, is_active
		FROM clinician_app.report_data_elements
		WHERE id = $1
	`, id).Scan(&current.ID, &current.Position, &current.ElementKey, &current.ColumnName, &current.DisplayName, &current.IsCore, &current.IsActive); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return errors.New("data element not found")
		}
		return err
	}

	newKey := current.ElementKey
	if !current.IsCore {
		candidateKey := strings.TrimSpace(elementKey)
		if candidateKey != "" {
			if !reportIdentifierPattern.MatchString(normalizeReportIdentifier(candidateKey)) {
				return errors.New("element key must be snake_case")
			}
			newKey = candidateKey
		}
	}

	newColumn := current.ColumnName
	if !current.IsCore {
		newColumn = columnName
	}

	if current.ColumnName != newColumn {
		if _, err := tx.ExecContext(ctx, `ALTER TABLE clinician_app.weeklyreport RENAME COLUMN `+current.ColumnName+` TO `+newColumn); err != nil {
			return err
		}
	}

	if current.DisplayName == displayName && current.ColumnName == newColumn && current.ElementKey == newKey {
		return nil
	}

	if _, err := tx.ExecContext(ctx, `
		UPDATE clinician_app.report_data_elements
		SET element_key = $1, column_name = $2, display_name = $3, is_active = TRUE, updated_on = NOW()
		WHERE id = $4
	`, newKey, newColumn, displayName, id); err != nil {
		return err
	}

	prevSnapshot := map[string]interface{}{"id": current.ID, "key": current.ElementKey, "column": current.ColumnName, "display_name": current.DisplayName}
	newSnapshot := map[string]interface{}{"id": current.ID, "key": newKey, "column": newColumn, "display_name": displayName}
	if err := insertCustomizationChange(ctx, tx, actorUserID, actorEmployeeID, "report_data_element", id, "update", "Report data element updated", prevSnapshot, newSnapshot); err != nil {
		return err
	}

	return tx.Commit()
}

func DeleteReportDataElement(ctx context.Context, db *sql.DB, actorUserID int64, actorEmployeeID int64, id int64) error {
	if id <= 0 {
		return errors.New("invalid data element id")
	}

	if err := EnsureCustomizationAuditSchema(ctx, db); err != nil {
		return err
	}
	if err := EnsureReportDataElementSchema(ctx, db); err != nil {
		return err
	}

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var current ReportDataElement
	if err := tx.QueryRowContext(ctx, `
		SELECT id, position, element_key, column_name, display_name, is_core, is_active
		FROM clinician_app.report_data_elements
		WHERE id = $1
	`, id).Scan(&current.ID, &current.Position, &current.ElementKey, &current.ColumnName, &current.DisplayName, &current.IsCore, &current.IsActive); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return errors.New("data element not found")
		}
		return err
	}

	if current.IsCore {
		return errors.New("core data elements cannot be deleted")
	}

	if _, err := tx.ExecContext(ctx, `ALTER TABLE clinician_app.weeklyreport DROP COLUMN IF EXISTS `+current.ColumnName); err != nil {
		return err
	}

	if _, err := tx.ExecContext(ctx, `DELETE FROM clinician_app.report_data_elements WHERE id = $1`, id); err != nil {
		return err
	}

	prevSnapshot := map[string]interface{}{"id": current.ID, "key": current.ElementKey, "column": current.ColumnName, "display_name": current.DisplayName}
	if err := insertCustomizationChange(ctx, tx, actorUserID, actorEmployeeID, "report_data_element", id, "delete", "Report data element deleted", prevSnapshot, map[string]interface{}{}); err != nil {
		return err
	}

	remaining, err := ListReportDataElements(ctx, tx)
	if err != nil {
		return err
	}

	sort.SliceStable(remaining, func(i, j int) bool {
		if remaining[i].Position == remaining[j].Position {
			return remaining[i].ID < remaining[j].ID
		}
		return remaining[i].Position < remaining[j].Position
	})
	for idx, item := range remaining {
		newPos := idx + 1
		if item.Position == newPos {
			continue
		}
		if _, err := tx.ExecContext(ctx, `UPDATE clinician_app.report_data_elements SET position = $1, updated_on = NOW() WHERE id = $2`, newPos, item.ID); err != nil {
			return err
		}
	}

	return tx.Commit()
}

func normalizeReportIdentifier(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

func ReportDataElementSchemaSnapshot(ctx context.Context, db DB) (string, error) {
	items, err := ListReportDataElements(ctx, db)
	if err != nil {
		return "", err
	}
	payload, err := json.Marshal(items)
	if err != nil {
		return "", err
	}
	return string(payload), nil
}
