BEGIN;
-- ============================================================
-- 1. Insert 7 missing departments (guard with WHERE NOT EXISTS)
-- ============================================================
INSERT INTO clinician_app.departments (d_name)
SELECT v.d_name
FROM (
        VALUES ('ENT'),
            ('Orthopaedic'),
            ('Ophthalmology'),
            ('Pathology'),
            ('Pharmacy'),
            ('Radiologist'),
            ('Laboratory')
    ) AS v(d_name)
WHERE NOT EXISTS (
        SELECT 1
        FROM clinician_app.departments d
        WHERE d.d_name = v.d_name
    );
-- ============================================================
-- 2. Fix existing department_roles data_points
--    (remove orphaned 'investigations' key, add mortality breakdown)
-- ============================================================
-- Surgery (dept_id = 1)
UPDATE clinician_app.department_roles
SET data_points = '["attendance","ward_rounds","patients_reviewed","elective","emergency","OPD_clinics","OPD_patients","teaching_rounds","students_taught","mortality_reviews","maternal","perinatal","surgical","medical","paed","labs_requests","imaging_requests"]'::jsonb
WHERE dept_id = 1;
-- Internal Medicine (dept_id = 2)
UPDATE clinician_app.department_roles
SET data_points = '["attendance","ward_rounds","patients_reviewed","OPD_clinics","OPD_patients","teaching_rounds","students_taught","mortality_reviews","maternal","perinatal","surgical","medical","paed","labs_requests","imaging_requests"]'::jsonb
WHERE dept_id = 2;
-- Paediatrics (dept_id = 3)
UPDATE clinician_app.department_roles
SET data_points = '["attendance","ward_rounds","patients_reviewed","elective","emergency","OPD_clinics","OPD_patients","teaching_rounds","students_taught","mortality_reviews","maternal","perinatal","surgical","medical","paed","labs_requests","imaging_requests"]'::jsonb
WHERE dept_id = 3;
-- Obstetrics and Gynaecology (dept_id = 4)
UPDATE clinician_app.department_roles
SET data_points = '["attendance","ward_rounds","patients_reviewed","elective","emergency","anc_patients","OPD_clinics","OPD_patients","teaching_rounds","students_taught","mortality_reviews","maternal","perinatal","surgical","medical","paed","labs_requests","imaging_requests"]'::jsonb
WHERE dept_id = 4;
-- Admin/Hospital Director (dept_id = 5): ensure one row exists with attendance only
INSERT INTO clinician_app.department_roles (dept_id, role_name, data_points)
SELECT 5,
    'default',
    '["attendance"]'::jsonb
WHERE NOT EXISTS (
        SELECT 1
        FROM clinician_app.department_roles
        WHERE dept_id = 5
    );
UPDATE clinician_app.department_roles
SET data_points = '["attendance"]'::jsonb
WHERE dept_id = 5;
-- ============================================================
-- 3. Insert default department_roles for the 7 new departments
--    (WHERE NOT EXISTS guards against re-running)
-- ============================================================
-- ENT (clinical cadre — same structure as Surgery)
INSERT INTO clinician_app.department_roles (dept_id, role_name, data_points)
SELECT d.id,
    'default',
    '["attendance","ward_rounds","patients_reviewed","elective","emergency","OPD_clinics","OPD_patients","teaching_rounds","students_taught","mortality_reviews","maternal","perinatal","surgical","medical","paed","labs_requests","imaging_requests"]'::jsonb
FROM clinician_app.departments d
WHERE d.d_name = 'ENT'
    AND NOT EXISTS (
        SELECT 1
        FROM clinician_app.department_roles r
        WHERE r.dept_id = d.id
    );
-- Orthopaedic (clinical cadre)
INSERT INTO clinician_app.department_roles (dept_id, role_name, data_points)
SELECT d.id,
    'default',
    '["attendance","ward_rounds","patients_reviewed","elective","emergency","OPD_clinics","OPD_patients","teaching_rounds","students_taught","mortality_reviews","maternal","perinatal","surgical","medical","paed","labs_requests","imaging_requests"]'::jsonb
FROM clinician_app.departments d
WHERE d.d_name = 'Orthopaedic'
    AND NOT EXISTS (
        SELECT 1
        FROM clinician_app.department_roles r
        WHERE r.dept_id = d.id
    );
-- Ophthalmology (clinical cadre)
INSERT INTO clinician_app.department_roles (dept_id, role_name, data_points)
SELECT d.id,
    'default',
    '["attendance","ward_rounds","patients_reviewed","elective","emergency","OPD_clinics","OPD_patients","teaching_rounds","students_taught","mortality_reviews","maternal","perinatal","surgical","medical","paed","labs_requests","imaging_requests"]'::jsonb
FROM clinician_app.departments d
WHERE d.d_name = 'Ophthalmology'
    AND NOT EXISTS (
        SELECT 1
        FROM clinician_app.department_roles r
        WHERE r.dept_id = d.id
    );
-- Pathology (non-clinical: no core clinical fields)
INSERT INTO clinician_app.department_roles (dept_id, role_name, data_points)
SELECT d.id,
    'default',
    '["attendance","teaching_rounds","students_taught","postmortems"]'::jsonb
FROM clinician_app.departments d
WHERE d.d_name = 'Pathology'
    AND NOT EXISTS (
        SELECT 1
        FROM clinician_app.department_roles r
        WHERE r.dept_id = d.id
    );
-- Pharmacy (non-clinical)
INSERT INTO clinician_app.department_roles (dept_id, role_name, data_points)
SELECT d.id,
    'default',
    '["attendance","teaching_rounds","students_taught","prescriptions_received","patients_prescribed"]'::jsonb
FROM clinician_app.departments d
WHERE d.d_name = 'Pharmacy'
    AND NOT EXISTS (
        SELECT 1
        FROM clinician_app.department_roles r
        WHERE r.dept_id = d.id
    );
-- Radiologist (non-clinical)
INSERT INTO clinician_app.department_roles (dept_id, role_name, data_points)
SELECT d.id,
    'default',
    '["attendance","diagnostics","xrays","ct_scans","obstetrics_scans","abdominal_scans"]'::jsonb
FROM clinician_app.departments d
WHERE d.d_name = 'Radiologist'
    AND NOT EXISTS (
        SELECT 1
        FROM clinician_app.department_roles r
        WHERE r.dept_id = d.id
    );
-- Laboratory (non-clinical)
INSERT INTO clinician_app.department_roles (dept_id, role_name, data_points)
SELECT d.id,
    'default',
    '["attendance","lab_investigations","BS","HIV","malaria","TB","CBC","chemistry","hematology","urinalysis","gram_stain","culture","microbiology","sensitivity_tests"]'::jsonb
FROM clinician_app.departments d
WHERE d.d_name = 'Laboratory'
    AND NOT EXISTS (
        SELECT 1
        FROM clinician_app.department_roles r
        WHERE r.dept_id = d.id
    );
-- ============================================================
-- Verify (run separately after migration if needed)
-- ============================================================
-- SELECT d.id, d.d_name, dr.role_name, dr.data_points
-- FROM clinician_app.departments d
-- LEFT JOIN clinician_app.department_roles dr ON dr.dept_id = d.id
-- ORDER BY d.id, dr.role_name;
COMMIT;