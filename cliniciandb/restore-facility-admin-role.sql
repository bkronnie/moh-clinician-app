-- Migration: Restore "Facility Admin" as a separate DB role.
--
-- After the rights-to-admin-migration.sql merged all admins into a single
-- "Admin" entry, we cannot distinguish national-level from facility-level
-- admins using HFID alone (because Ministry of Health employees also have
-- HFID > 0). This migration restores the distinction:
--
--   "Admin"          → national-level admins (facility.f_level = 'National')
--   "Facility Admin" → facility-level admins  (all other facilities)
--   "Staff"          → unchanged

BEGIN;

-- 1. Re-add the Facility Admin role.
INSERT INTO clinician_app.rights (rights) VALUES ('Facility Admin');

-- 2. Re-map all currently-Admin employees whose facility is NOT national-level
--    to "Facility Admin".  Employees with no facility or with a National
--    facility (Ministry of Health) keep the "Admin" (national) role.
UPDATE clinician_app.employeerights
SET rights = (SELECT id FROM clinician_app.rights WHERE rights = 'Facility Admin')
WHERE employee IN (
    SELECT e.id
    FROM clinician_app.employees e
    JOIN clinician_app.facilities f ON f.id = e.facility
    WHERE f.f_level <> 'National'
)
AND rights = (SELECT id FROM clinician_app.rights WHERE rights = 'Admin');

-- Verification
SELECT id, rights FROM clinician_app.rights ORDER BY rights;

SELECT r.rights, COUNT(*) AS assigned_count
FROM clinician_app.employeerights er
JOIN clinician_app.rights r ON r.id = er.rights
GROUP BY r.rights
ORDER BY r.rights;

COMMIT;
