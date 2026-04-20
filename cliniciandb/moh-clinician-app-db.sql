-- Text-format PostgreSQL bootstrap script for the MOH Clinician App.
--
-- This file is plain SQL for use in IDE SQL editors.
-- It must be run while connected to the target database: clinician
--
-- Before running this file, create the role and database separately as an admin:
--   CREATE ROLE clinician_app LOGIN PASSWORD 'root';
--   CREATE DATABASE clinician OWNER clinician_app;
--
-- Then connect your SQL editor to database: clinician
-- and run the rest of this file.
--
-- Notes:
-- 1. This is a readable SQL reconstruction for the app's expected schema.
-- 2. The original file `moh-clinician-app-db.sql` appears to be a non-text/custom dump.

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1
        FROM pg_roles
        WHERE rolname = 'clinician_app'
    ) THEN
        CREATE ROLE clinician_app LOGIN PASSWORD 'root';
    END IF;
END
$$;

CREATE SCHEMA IF NOT EXISTS clinician_app;
GRANT USAGE ON SCHEMA clinician_app TO clinician_app;
GRANT CREATE ON SCHEMA clinician_app TO clinician_app;

CREATE TABLE IF NOT EXISTS clinician_app.lg (
    id BIGSERIAL PRIMARY KEY,
    lg_name TEXT,
    lg_type TEXT
);

CREATE TABLE IF NOT EXISTS clinician_app.facilities (
    id BIGSERIAL PRIMARY KEY,
    f_name TEXT NOT NULL UNIQUE,
    f_level TEXT NOT NULL DEFAULT '',
    f_lg BIGINT REFERENCES clinician_app.lg(id),
    created_by BIGINT,
    created_on TIMESTAMP
);

CREATE TABLE IF NOT EXISTS clinician_app.departments (
    id BIGSERIAL PRIMARY KEY,
    d_name TEXT NOT NULL UNIQUE
);

CREATE TABLE IF NOT EXISTS clinician_app.specialist_titles (
    id BIGINT PRIMARY KEY,
    title TEXT NOT NULL UNIQUE
);

CREATE TABLE IF NOT EXISTS clinician_app.rights (
    id BIGSERIAL PRIMARY KEY,
    rights TEXT NOT NULL UNIQUE
);

CREATE TABLE IF NOT EXISTS clinician_app.employees (
    id BIGINT PRIMARY KEY,
    fname TEXT,
    lname TEXT,
    oname TEXT,
    specialisation TEXT,
    department BIGINT REFERENCES clinician_app.departments(id),
    facility BIGINT REFERENCES clinician_app.facilities(id),
    created_by BIGINT,
    created_on TIMESTAMP,
    title BIGINT REFERENCES clinician_app.specialist_titles(id)
);

ALTER TABLE clinician_app.employees
    ADD COLUMN IF NOT EXISTS employee_number TEXT,
    ADD COLUMN IF NOT EXISTS date_of_birth DATE,
    ADD COLUMN IF NOT EXISTS phone_number TEXT;

CREATE UNIQUE INDEX IF NOT EXISTS idx_employees_employee_number_unique
    ON clinician_app.employees(employee_number)
    WHERE employee_number IS NOT NULL;

CREATE TABLE IF NOT EXISTS clinician_app.employeerights (
    id BIGSERIAL PRIMARY KEY,
    employee BIGINT REFERENCES clinician_app.employees(id),
    rights BIGINT NOT NULL REFERENCES clinician_app.rights(id)
);

CREATE TABLE IF NOT EXISTS clinician_app.users (
    id BIGSERIAL PRIMARY KEY,
    username TEXT NOT NULL UNIQUE,
    pssword TEXT NOT NULL,
    employees BIGINT REFERENCES clinician_app.employees(id),
    created_by BIGINT,
    created_on TIMESTAMP
);

DO $$
BEGIN
    IF EXISTS (
        SELECT 1
        FROM information_schema.columns
        WHERE table_schema = 'clinician_app'
          AND table_name = 'users'
          AND column_name = 'rights'
          AND data_type IN ('text', 'character varying')
    ) THEN
        INSERT INTO clinician_app.employeerights (employee, rights)
        SELECT u.employees, COALESCE(r.id, s.id)
        FROM clinician_app.users u
        CROSS JOIN (
            SELECT id
            FROM clinician_app.rights
            WHERE rights = 'Staff'
            LIMIT 1
        ) s
        LEFT JOIN clinician_app.rights r
            ON LOWER(TRIM(u.rights)) = LOWER(TRIM(r.rights))
        WHERE u.employees IS NOT NULL
          AND NOT EXISTS (
              SELECT 1
              FROM clinician_app.employeerights er
              WHERE er.employee = u.employees
          );
    END IF;

    IF EXISTS (
        SELECT 1
        FROM information_schema.columns
        WHERE table_schema = 'clinician_app'
          AND table_name = 'users'
          AND column_name = 'rights'
          AND data_type IN ('bigint', 'integer', 'smallint')
    ) THEN
        INSERT INTO clinician_app.employeerights (employee, rights)
        SELECT u.employees, COALESCE(u.rights, s.id)
        FROM clinician_app.users u
        CROSS JOIN (
            SELECT id
            FROM clinician_app.rights
            WHERE rights = 'Staff'
            LIMIT 1
        ) s
        WHERE u.employees IS NOT NULL
          AND NOT EXISTS (
              SELECT 1
              FROM clinician_app.employeerights er
              WHERE er.employee = u.employees
          );

        ALTER TABLE clinician_app.users DROP CONSTRAINT IF EXISTS users_rights_fk;
        ALTER TABLE clinician_app.users DROP CONSTRAINT IF EXISTS users_rights_fkey;
        ALTER TABLE clinician_app.users DROP COLUMN rights;
    END IF;

    IF EXISTS (
        SELECT 1
        FROM information_schema.columns
        WHERE table_schema = 'clinician_app'
          AND table_name = 'users'
          AND column_name = 'rights_legacy'
    ) THEN
        ALTER TABLE clinician_app.users DROP COLUMN rights_legacy;
    END IF;

    IF EXISTS (
        SELECT 1
        FROM information_schema.columns
        WHERE table_schema = 'clinician_app'
          AND table_name = 'users'
          AND column_name = 'access_scope'
    ) THEN
        ALTER TABLE clinician_app.users DROP COLUMN access_scope;
    END IF;
END
$$;

CREATE TABLE IF NOT EXISTS clinician_app.employee_profile_changes (
    id BIGSERIAL PRIMARY KEY,
    employee_id BIGINT NOT NULL REFERENCES clinician_app.employees(id),
    changed_by_user BIGINT REFERENCES clinician_app.users(id),
    changed_by_employee BIGINT REFERENCES clinician_app.employees(id),
    changed_on TIMESTAMP NOT NULL DEFAULT NOW(),
    change_summary TEXT NOT NULL,
    previous_snapshot JSONB NOT NULL DEFAULT '{}'::jsonb,
    new_snapshot JSONB NOT NULL DEFAULT '{}'::jsonb
);

CREATE INDEX IF NOT EXISTS idx_employee_profile_changes_employee_changed_on
    ON clinician_app.employee_profile_changes(employee_id, changed_on DESC);

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
);

CREATE INDEX IF NOT EXISTS idx_customization_change_log_on_changed_on
    ON clinician_app.customization_change_log(changed_on DESC);

CREATE INDEX IF NOT EXISTS idx_customization_change_log_entity
    ON clinician_app.customization_change_log(entity_type, entity_id, changed_on DESC);

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
);

CREATE INDEX IF NOT EXISTS idx_report_data_elements_active_position
    ON clinician_app.report_data_elements(is_active, position);

CREATE TABLE IF NOT EXISTS clinician_app.indicators (
    id BIGSERIAL PRIMARY KEY,
    indicator TEXT NOT NULL,
    created_by BIGINT,
    created_on TIMESTAMP
);

CREATE TABLE IF NOT EXISTS clinician_app.targets (
    id BIGSERIAL PRIMARY KEY,
    indicator BIGINT REFERENCES clinician_app.indicators(id),
    target BIGINT,
    created_by BIGINT,
    created_on TIMESTAMP
);

CREATE TABLE IF NOT EXISTS clinician_app.leavetypes (
    leave_type_id BIGINT PRIMARY KEY,
    leave_type_name TEXT NOT NULL UNIQUE,
    description TEXT,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS clinician_app.staffleave (
    leave_id BIGINT PRIMARY KEY,
    employee_id BIGINT NOT NULL REFERENCES clinician_app.employees(id),
    start_date DATE,
    end_date DATE,
    leave_status TEXT,
    approved_by BIGINT,
    created_on TIMESTAMP,
    notes TEXT,
    leave_type_id BIGINT REFERENCES clinician_app.leavetypes(leave_type_id),
    return_date DATE
);

ALTER TABLE clinician_app.staffleave
    ADD COLUMN IF NOT EXISTS reviewed_on TIMESTAMP;

CREATE OR REPLACE VIEW clinician_app.staffleave_view AS
SELECT
    s.leave_id,
    s.employee_id,
    s.start_date,
    s.end_date,
    s.leave_status,
    s.approved_by,
    s.created_on,
    s.notes,
    s.leave_type_id,
    s.return_date
FROM clinician_app.staffleave s;

CREATE TABLE IF NOT EXISTS clinician_app.department_roles (
    role_id BIGSERIAL PRIMARY KEY,
    dept_id BIGINT NOT NULL REFERENCES clinician_app.departments(id),
    role_name TEXT NOT NULL,
    data_points JSONB NOT NULL DEFAULT '[]'::jsonb
);

CREATE TABLE IF NOT EXISTS clinician_app.weeklyreport (
    id BIGINT PRIMARY KEY,
    hospital BIGINT REFERENCES clinician_app.facilities(id),
    department BIGINT REFERENCES clinician_app.departments(id),
    employee BIGINT REFERENCES clinician_app.employees(id),
    start DATE,
    stop DATE,
    attendance BIGINT,
    ward_rounds BIGINT,
    patients_reviewed BIGINT,
    theatre_days BIGINT,
    elective BIGINT,
    emergency BIGINT,
    postmortems BIGINT,
    opd_clinics BIGINT,
    opd_patients BIGINT,
    anc_patients BIGINT,
    teaching_rounds BIGINT,
    students_taught BIGINT,
    mortality_reviews BIGINT,
    maternal BIGINT,
    perinatal BIGINT,
    surgical BIGINT,
    medical BIGINT,
    paed BIGINT,
    labs_requests BIGINT,
    imaging_requests BIGINT,
    lab_investigations BIGINT,
    bs BIGINT,
    hiv BIGINT,
    malaria BIGINT,
    tb BIGINT,
    cbc BIGINT,
    chemistry BIGINT,
    hematology BIGINT,
    urinalysis BIGINT,
    gram_stain BIGINT,
    culture BIGINT,
    microbiology BIGINT,
    sensitivity_tests BIGINT,
    diagnostics BIGINT,
    xrays BIGINT,
    ct_scans BIGINT,
    obstetrics_scans BIGINT,
    abdominal_scans BIGINT,
    custom_metric_39 BIGINT,
    custom_metric_40 BIGINT,
    created_on TIMESTAMP,
    entered_by BIGINT,
    report_status TEXT DEFAULT 'Draft',
    last_updated_on TIMESTAMP,
    submitted_by BIGINT,
    approved_by BIGINT,
    facility_review_status TEXT,
    facility_reviewed_by BIGINT,
    facility_reviewed_on TIMESTAMP,
    national_submission_status TEXT,
    national_submitted_by BIGINT,
    national_submitted_on TIMESTAMP,
    national_review_status TEXT,
    national_reviewed_by BIGINT,
    national_reviewed_on TIMESTAMP,
    submit_status TEXT
);

    -- Track which days staff worked during the reporting week and when reports were submitted.
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
        ADD COLUMN IF NOT EXISTS national_reviewed_on TIMESTAMP;

CREATE TABLE IF NOT EXISTS clinician_app.attendance_records (
    id BIGSERIAL PRIMARY KEY,
    attendance_date DATE,
    specialist_id BIGINT REFERENCES clinician_app.employees(id),
    department_id BIGINT REFERENCES clinician_app.departments(id),
    attendance_type TEXT,
    facility_id BIGINT REFERENCES clinician_app.facilities(id)
);

CREATE TABLE IF NOT EXISTS clinician_app.surgeries (
    id BIGSERIAL PRIMARY KEY,
    surgery_date DATE,
    surgery_type TEXT,
    department_id BIGINT REFERENCES clinician_app.departments(id),
    patient_id BIGINT,
    surgeries_count BIGINT,
    specialist_id BIGINT REFERENCES clinician_app.employees(id),
    facility_id BIGINT REFERENCES clinician_app.facilities(id)
);

CREATE TABLE IF NOT EXISTS clinician_app.ward_rounds (
    id BIGSERIAL PRIMARY KEY,
    round_date DATE,
    department_id BIGINT REFERENCES clinician_app.departments(id),
    patients_reviewed BIGINT,
    specialist_id BIGINT REFERENCES clinician_app.employees(id),
    facility_id BIGINT REFERENCES clinician_app.facilities(id)
);

CREATE TABLE IF NOT EXISTS clinician_app.investigations (
    id BIGSERIAL PRIMARY KEY,
    request_date DATE,
    investigation_type TEXT,
    test_type TEXT,
    result_status TEXT,
    result TEXT,
    specialist_id BIGINT REFERENCES clinician_app.employees(id),
    facility_id BIGINT REFERENCES clinician_app.facilities(id)
);

CREATE INDEX IF NOT EXISTS idx_users_username ON clinician_app.users(username);
CREATE INDEX IF NOT EXISTS idx_users_employees ON clinician_app.users(employees);
CREATE INDEX IF NOT EXISTS idx_employees_facility ON clinician_app.employees(facility);
CREATE INDEX IF NOT EXISTS idx_employees_department ON clinician_app.employees(department);
CREATE INDEX IF NOT EXISTS idx_staffleave_employee_id ON clinician_app.staffleave(employee_id);
CREATE INDEX IF NOT EXISTS idx_staffleave_employee_status_dates ON clinician_app.staffleave(employee_id, leave_status, start_date, end_date);
CREATE INDEX IF NOT EXISTS idx_staffleave_status_dates ON clinician_app.staffleave(leave_status, start_date, end_date);
CREATE INDEX IF NOT EXISTS idx_weeklyreport_employee_start ON clinician_app.weeklyreport(employee, start);
CREATE UNIQUE INDEX IF NOT EXISTS ux_weeklyreport_employee_start_stop
ON clinician_app.weeklyreport(employee, start, stop)
WHERE employee IS NOT NULL AND start IS NOT NULL AND stop IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_weeklyreport_facility_dept_start ON clinician_app.weeklyreport(hospital, department, start);
CREATE INDEX IF NOT EXISTS idx_weeklyreport_hospital_start_status ON clinician_app.weeklyreport(hospital, start, submit_status, report_status);
CREATE INDEX IF NOT EXISTS idx_weeklyreport_employee_start_status ON clinician_app.weeklyreport(employee, start, submit_status, report_status);
CREATE INDEX IF NOT EXISTS idx_weeklyreport_facility_week_review_flow ON clinician_app.weeklyreport(hospital, start, facility_review_status, national_submission_status, national_review_status);
CREATE INDEX IF NOT EXISTS idx_department_roles_dept_role ON clinician_app.department_roles(dept_id, role_name);

INSERT INTO clinician_app.specialist_titles (id, title) VALUES
    (1, 'Medical Officer(SG)'),
    (2, 'Medical Officer'),
    (3, 'Medical Officer(Specialist)'),
    (4, 'Senior Consultant'),
    (5, 'Consultant'),
    (6, 'Senior Nursing Officer'),
    (7, 'Nursing Officer')
ON CONFLICT (id) DO NOTHING;

INSERT INTO clinician_app.leavetypes (leave_type_id, leave_type_name, description) VALUES
    (1, 'Annual Leave', 'Annual leave'),
    (2, 'Sick Leave', 'Sick leave'),
    (3, 'Maternity Leave', 'Maternity leave'),
    (4, 'Paternity Leave', 'Paternity leave'),
    (5, 'Bereavement Leave', 'Bereavement leave'),
    (6, 'Unpaid Leave', 'Unpaid leave'),
    (7, 'Study Leave', 'Study leave'),
    (8, 'Field Activities Leave', 'Field activities leave'),
    (9, 'Emergency Leave', 'Emergency leave')
ON CONFLICT (leave_type_id) DO NOTHING;

INSERT INTO clinician_app.rights (rights) VALUES
    ('National Admin'),
    ('Staff'),
    ('Facility Admin')
ON CONFLICT (rights) DO NOTHING;

INSERT INTO clinician_app.departments (id, d_name) VALUES
    (1, 'Surgery'),
    (2, 'Internal Medicine'),
    (3, 'Paediatrics'),
    (4, 'Obstetrics and Gynaecology')
ON CONFLICT (id) DO NOTHING;

INSERT INTO clinician_app.department_roles (dept_id, role_name, data_points) VALUES
    (
        1,
        'default',
        '[
            "attendance","ward_rounds","patients_reviewed","theatre_days",
            "elective","emergency","postmortems","OPD_clinics","OPD_patients",
            "teaching_rounds","students_taught","mortality_reviews","labs_requests",
            "imaging_requests","investigations","xrays","ct_scans"
        ]'::jsonb
    ),
    (
        2,
        'default',
        '[
            "attendance","ward_rounds","patients_reviewed","OPD_clinics",
            "OPD_patients","teaching_rounds","students_taught","mortality_reviews",
            "medical","labs_requests","imaging_requests","investigations",
            "CBC","chemistry","hematology","urinalysis"
        ]'::jsonb
    ),
    (
        3,
        'default',
        '[
            "attendance","ward_rounds","patients_reviewed","OPD_clinics",
            "OPD_patients","teaching_rounds","students_taught","mortality_reviews",
            "paed","labs_requests","imaging_requests","investigations",
            "malaria","TB","CBC"
        ]'::jsonb
    ),
    (
        4,
        'default',
        '[
            "attendance","ward_rounds","patients_reviewed","theatre_days",
            "elective","emergency","anc_patients","maternal","perinatal",
            "OPD_clinics","OPD_patients","teaching_rounds","students_taught",
            "obstetrics_scans","abdominal_scans"
        ]'::jsonb
    )
ON CONFLICT DO NOTHING;

GRANT ALL PRIVILEGES ON ALL TABLES IN SCHEMA clinician_app TO clinician_app;
GRANT ALL PRIVILEGES ON ALL SEQUENCES IN SCHEMA clinician_app TO clinician_app;

