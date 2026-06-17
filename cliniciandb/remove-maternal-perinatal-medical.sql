-- Remove maternal, perinatal, and medical data points from
-- Surgical, Internal Medicine, Paediatrics, and Orthopaedic departments.
-- These review categories are no longer collected for these departments.
BEGIN;
-- Surgery (dept_id = 1)
UPDATE clinician_app.department_roles
SET data_points = '["attendance","ward_rounds","patients_reviewed","elective","emergency","OPD_clinics","OPD_patients","teaching_rounds","students_taught","mortality_reviews","surgical","paed","labs_requests","imaging_requests"]'::jsonb
WHERE dept_id = 1;
-- Internal Medicine (dept_id = 2)
UPDATE clinician_app.department_roles
SET data_points = '["attendance","ward_rounds","patients_reviewed","OPD_clinics","OPD_patients","teaching_rounds","students_taught","mortality_reviews","surgical","paed","labs_requests","imaging_requests"]'::jsonb
WHERE dept_id = 2;
-- Paediatrics (dept_id = 3)
UPDATE clinician_app.department_roles
SET data_points = '["attendance","ward_rounds","patients_reviewed","elective","emergency","OPD_clinics","OPD_patients","teaching_rounds","students_taught","mortality_reviews","surgical","paed","labs_requests","imaging_requests"]'::jsonb
WHERE dept_id = 3;
-- Orthopaedic (identified by name because its dept_id is assigned at runtime)
UPDATE clinician_app.department_roles r
SET data_points = '["attendance","ward_rounds","patients_reviewed","elective","emergency","OPD_clinics","OPD_patients","teaching_rounds","students_taught","mortality_reviews","surgical","paed","labs_requests","imaging_requests"]'::jsonb
FROM clinician_app.departments d
WHERE r.dept_id = d.id
    AND d.d_name = 'Orthopaedic';
COMMIT;