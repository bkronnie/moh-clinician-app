-- ============================================================
-- Migration: weekly -> daily submissions
-- Run this script once against the production database.
-- It is safe to run inside a transaction; if anything fails
-- the whole script is rolled back.
-- ============================================================
BEGIN;
-- 1. Remove the constraint that forced stop = start + 6 days.
--    This allows daily records where stop = start.
ALTER TABLE clinician_app.weeklyreport DROP CONSTRAINT IF EXISTS ck_weeklyreport_start_stop_6_days;
-- 2. Remove the constraint that forced start to fall on a Monday.
--    Daily entries can fall on any day of the week.
ALTER TABLE clinician_app.weeklyreport DROP CONSTRAINT IF EXISTS ck_weeklyreport_start_monday;
-- 3. Convert all existing weekly records to single-day records.
--    Each record's aggregate values are preserved on the week's
--    start date (Monday).  Administrators can adjust historical
--    values in the UI if needed.
UPDATE clinician_app.weeklyreport
SET stop = start;
-- 4. Enforce the new daily invariant: every record covers exactly
--    one calendar day.
ALTER TABLE clinician_app.weeklyreport
ADD CONSTRAINT ck_weeklyreport_daily CHECK (stop = start);
-- 4. The existing overlap exclusion constraint
--    (ex_weeklyreport_employee_hospital_no_overlap) and the
--    unique index (ux_weeklyreport_employee_hospital_start) both
--    continue to work correctly for single-day ranges and still
--    prevent duplicate records for the same employee/facility/date.
--    No changes to those constraints are needed.
-- 5. The days_worked column is now semantically redundant (a daily
--    record represents exactly one working day) but is preserved
--    for backward compatibility with the existing application code.
--    It can be dropped in a future migration after the codebase
--    no longer references it.
COMMIT;