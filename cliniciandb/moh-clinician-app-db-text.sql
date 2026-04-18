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
    rights BIGINT REFERENCES clinician_app.rights(id)
);

CREATE TABLE IF NOT EXISTS clinician_app.users (
    id BIGSERIAL PRIMARY KEY,
    username TEXT NOT NULL UNIQUE,
    pssword TEXT NOT NULL,
    employees BIGINT REFERENCES clinician_app.employees(id),
    created_by BIGINT,
    created_on TIMESTAMP,
    rights TEXT NOT NULL DEFAULT 'user'
);

ALTER TABLE clinician_app.users
    ADD COLUMN IF NOT EXISTS access_scope TEXT NOT NULL DEFAULT 'individual';

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1
        FROM pg_constraint
        WHERE conname = 'users_access_scope_check'
          AND conrelid = 'clinician_app.users'::regclass
    ) THEN
        ALTER TABLE clinician_app.users
            ADD CONSTRAINT users_access_scope_check
            CHECK (access_scope IN ('national', 'facility', 'individual'));
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
    qn_01 BIGINT,
    qn_02 BIGINT,
    qn_03 BIGINT,
    qn_04 BIGINT,
    qn_05 BIGINT,
    qn_06 BIGINT,
    qn_07 BIGINT,
    qn_08 BIGINT,
    qn_09 BIGINT,
    qn_10 BIGINT,
    qn_11 BIGINT,
    qn_12 BIGINT,
    qn_13 BIGINT,
    qn_14 BIGINT,
    qn_15 BIGINT,
    qn_16 BIGINT,
    qn_17 BIGINT,
    qn_18 BIGINT,
    qn_19 BIGINT,
    qn_20 BIGINT,
    qn_21 BIGINT,
    qn_22 BIGINT,
    qn_23 BIGINT,
    qn_24 BIGINT,
    qn_25 BIGINT,
    qn_26 BIGINT,
    qn_27 BIGINT,
    qn_28 BIGINT,
    qn_29 BIGINT,
    qn_30 BIGINT,
    qn_31 BIGINT,
    qn_32 BIGINT,
    qn_33 BIGINT,
    qn_34 BIGINT,
    qn_35 BIGINT,
    qn_36 BIGINT,
    qn_37 BIGINT,
    qn_38 BIGINT,
    qn_39 BIGINT,
    qn_40 BIGINT,
    created_on TIMESTAMP,
    entered_by BIGINT,
    report_status TEXT DEFAULT 'Draft',
    last_updated_on TIMESTAMP,
    submitted_by BIGINT,
    approved_by BIGINT,
    submit_status TEXT
);

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
CREATE INDEX IF NOT EXISTS idx_employees_facility ON clinician_app.employees(facility);
CREATE INDEX IF NOT EXISTS idx_employees_department ON clinician_app.employees(department);
CREATE INDEX IF NOT EXISTS idx_staffleave_employee_id ON clinician_app.staffleave(employee_id);
CREATE INDEX IF NOT EXISTS idx_weeklyreport_employee_start ON clinician_app.weeklyreport(employee, start);
CREATE INDEX IF NOT EXISTS idx_weeklyreport_facility_dept_start ON clinician_app.weeklyreport(hospital, department, start);

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
    ('admin'),
    ('user'),
    ('approver')
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

