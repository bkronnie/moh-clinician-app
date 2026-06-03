-- =============================================================================
-- Migration: collapse 3-role system (National Admin, Facility Admin, Staff)
--            into 2-role system (Admin, Staff).
--
-- Rules after migration:
--   Admin  → any user with administrative access (national or facility level)
--   Staff  → regular clinicians / data-entry staff
--
-- The application distinguishes national-level Admins (no facility) from
-- facility-level Admins (has a facility) at session load time using HFID,
-- not by role label.
-- =============================================================================

BEGIN;

-- Step 1: Rename "National Admin" → "Admin"
UPDATE clinician_app.rights
SET rights = 'Admin'
WHERE rights = 'National Admin';

-- Step 2: Re-point all "Facility Admin" employeerights rows → "Admin"
UPDATE clinician_app.employeerights
SET rights = (SELECT id FROM clinician_app.rights WHERE rights = 'Admin')
WHERE rights = (SELECT id FROM clinician_app.rights WHERE rights = 'Facility Admin');

-- Step 3: Delete the now-redundant "Facility Admin" row
DELETE FROM clinician_app.rights WHERE rights = 'Facility Admin';

-- Step 4: Verify
SELECT id, rights FROM clinician_app.rights ORDER BY id;

-- Step 5: Check employeerights distribution
SELECT r.rights, COUNT(*) AS assigned_count
FROM clinician_app.employeerights er
JOIN clinician_app.rights r ON r.id = er.rights
GROUP BY r.rights
ORDER BY r.rights;

COMMIT;
