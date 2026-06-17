-- Fix Surgery (dept_id=1): sync all default rows to include major/minor procedures.
-- These keys were saved to some rows but not role_id=1 (the canonical read row).
UPDATE clinician_app.department_roles
SET data_points = '["attendance","ward_rounds","patients_reviewed","elective","elective_major_sugeries","elective_minor_sugeries","emergency","OPD_clinics","OPD_patients","teaching_rounds","students_taught","mortality_reviews","surgical","paed","labs_requests","imaging_requests"]'::jsonb
WHERE dept_id = 1
  AND LOWER(COALESCE(role_name, '')) = 'default';
