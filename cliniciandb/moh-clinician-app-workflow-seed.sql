-- ============================================================
-- MOH Clinician App — Workflow Test Data Update Script
-- ============================================================
-- Target DB : warehouse
-- Schema    : clinician_app
-- Run as    : warehouse user (password: warehouse)
--
-- PURPOSE: Populates the new 3-stage review workflow columns and
-- fills missing employee profile data so all application features
-- can be exercised with realistic test data.
--
-- This script is IDEMPOTENT — safe to run multiple times.
-- It only UPDATEs existing records; it does not INSERT or DELETE.
--
-- WEEK DISTRIBUTION (26 weeks: 2026-01-05 to 2026-06-29):
-- ┌──────────┬─────────────────┬─────────────────────────────────────────┐
-- │ Week(s)  │ Start Dates     │ State                                   │
-- ├──────────┼─────────────────┼─────────────────────────────────────────┤
-- │  00–11   │ Jan 05 – Mar 22 │ Full pipeline — nationally approved ✓   │
-- │    12    │ Mar 30          │ Nationally submitted; national review:   │
-- │          │                 │   Hosp 4,9,13 → Declined (reopen test)  │
-- │          │                 │   all others  → Approved               │
-- │    13    │ Apr 06          │ Facility approved + nationally submitted │
-- │          │                 │   Hosp 5 → facility Declined (test)     │
-- │    14    │ Apr 13          │ Current week — individually reviewed;   │
-- │          │                 │   no national batch submission yet      │
-- │    15    │ Apr 20          │ Staff submitted; facility review pending │
-- │  16–25   │ Apr 27 – Jun 29 │ Future — draft / pre-populated          │
-- └──────────┴─────────────────┴─────────────────────────────────────────┘
--
-- FACILITY → ADMIN USER MAPPING used throughout:
--   1 Arua       → 114   9  Lira       → 100
--   2 Entebbe    → 104  10  Masaka     → 112
--   3 Fort Portal→ 109  11  Mbale      → 111
--   4 Gulu       → 108  12  Mbarara    → 110
--   5 Hoima      → 105  13  Moroto     → 106
--   6 Jinja      → 107  14  Mubende    → 113
--   7 Kabale     → 102  15  Soroti     → 103
--   8 Kayunga    →  99  16  Naguru     → 101
-- ============================================================

SET search_path TO clinician_app;

BEGIN;

-- ============================================================
-- SECTION 1 — FIX DATA QUALITY ISSUES
-- ============================================================

-- 1a. Fix the lone NULL report_status
UPDATE clinician_app.weeklyreport
SET    report_status = 'Draft'
WHERE  report_status IS NULL;

-- ============================================================
-- SECTION 2 — EMPLOYEE PROFILE DATA
-- ============================================================
-- Add employee_number (MO-0001 … MO-0080),
-- date_of_birth (spread 1975-1989), and
-- Ugandan mobile phone numbers (+256 7XX XXXXXXX).
-- Employee IDs run from 1001 to 1080.

UPDATE clinician_app.employees SET
    employee_number = 'MO-' || LPAD((id - 1000)::text, 4, '0'),
    date_of_birth   = '1975-01-01'::date
                      + ((((id - 1001) * 137 + 29) % (365 * 14)) || ' days')::interval,
    phone_number    = '+256 7'
                      || LPAD(
                             (((id - 1001) * 7823 + 1000000) % 90000000 + 10000000)::text,
                             8, '0')
WHERE  employee_number IS NULL;

-- ============================================================
-- SECTION 3 — STAFFLEAVE IMPROVEMENTS
-- ============================================================

-- 3a. Set reviewed_on for all reviewed leaves (Approved, Valid, etc.)
UPDATE clinician_app.staffleave SET
    reviewed_on = created_on + INTERVAL '2 days'
WHERE  reviewed_on  IS NULL
  AND  leave_status IN ('Approved', 'Valid', 'Rejected', 'Completed', 'Expired');

-- 3b. Assign approved_by where missing for reviewed leaves.
--     Look up the facility admin user for the employee's facility.
UPDATE clinician_app.staffleave sl SET
    approved_by = (
        SELECT u.id
        FROM   clinician_app.users      u
        JOIN   clinician_app.employees  ue ON ue.id = u.employees
        WHERE  ue.facility = (
                   SELECT e.facility
                   FROM   clinician_app.employees e
                   WHERE  e.id = sl.employee_id
               )
          AND  u.rights       = 'approver'
          AND  u.access_scope = 'facility'
        ORDER  BY u.id
        LIMIT  1
    )
WHERE  approved_by IS NULL
  AND  leave_status IN ('Approved', 'Valid', 'Completed');

-- 3c. Ensure return_date is set for reviewed leaves
UPDATE clinician_app.staffleave SET
    return_date = end_date + INTERVAL '1 day'
WHERE  return_date IS NULL
  AND  end_date    IS NOT NULL
  AND  leave_status IN ('Approved', 'Valid', 'Completed');

-- ============================================================
-- SECTION 4 — WEEKLYREPORT: SHARED FIELDS
-- ============================================================

-- 4a. days_worked: default 5-day working week for all records
UPDATE clinician_app.weeklyreport SET days_worked = '1,2,3,4,5'
WHERE  days_worked IS NULL;

-- 4b. submitted_on and submitted_by for all staff-submitted reports
--     Time is staggered by report id so not all land at exactly the same moment.
UPDATE clinician_app.weeklyreport SET
    submitted_on = stop + (((id % 8) + 1)::text || ' hours')::interval,
    submitted_by = employee - 1000   -- user_id = employee_id − 1000
WHERE  submitted_on IS NULL
  AND  submit_status  = 'Submitted'
  AND  report_status IN ('Submitted', 'Approved', 'Declined');

-- 4c. approved_by (individual-report facility approval) for all Approved records
UPDATE clinician_app.weeklyreport SET
    approved_by = CASE hospital
        WHEN  1 THEN 114 WHEN  2 THEN 104 WHEN  3 THEN 109 WHEN  4 THEN 108
        WHEN  5 THEN 105 WHEN  6 THEN 107 WHEN  7 THEN 102 WHEN  8 THEN  99
        WHEN  9 THEN 100 WHEN 10 THEN 112 WHEN 11 THEN 111 WHEN 12 THEN 110
        WHEN 13 THEN 106 WHEN 14 THEN 113 WHEN 15 THEN 103 ELSE            101
    END
WHERE  approved_by   IS NULL
  AND  report_status  = 'Approved';

-- ============================================================
-- SECTION 5 — WEEKS 00–11  (Jan 05 – Mar 22)
--             FULL PIPELINE: all nationally approved
-- ============================================================

-- 5a. Upgrade any lingering 'Submitted' records in old weeks to 'Approved'
--     (old data before the workflow columns were added)
UPDATE clinician_app.weeklyreport SET
    report_status   = 'Approved',
    approved_by     = CASE hospital
        WHEN  1 THEN 114 WHEN  2 THEN 104 WHEN  3 THEN 109 WHEN  4 THEN 108
        WHEN  5 THEN 105 WHEN  6 THEN 107 WHEN  7 THEN 102 WHEN  8 THEN  99
        WHEN  9 THEN 100 WHEN 10 THEN 112 WHEN 11 THEN 111 WHEN 12 THEN 110
        WHEN 13 THEN 106 WHEN 14 THEN 113 WHEN 15 THEN 103 ELSE            101
    END
WHERE  start        < '2026-03-30'
  AND  report_status = 'Submitted';

-- 5b. Populate full 3-stage workflow for all approved records before week 12
UPDATE clinician_app.weeklyreport SET
    facility_review_status     = 'Approved',
    facility_reviewed_by       = CASE hospital
        WHEN  1 THEN 114 WHEN  2 THEN 104 WHEN  3 THEN 109 WHEN  4 THEN 108
        WHEN  5 THEN 105 WHEN  6 THEN 107 WHEN  7 THEN 102 WHEN  8 THEN  99
        WHEN  9 THEN 100 WHEN 10 THEN 112 WHEN 11 THEN 111 WHEN 12 THEN 110
        WHEN 13 THEN 106 WHEN 14 THEN 113 WHEN 15 THEN 103 ELSE            101
    END,
    facility_reviewed_on       = stop + INTERVAL '1 day 09:00:00',
    national_submission_status = 'Submitted',
    national_submitted_by      = CASE hospital
        WHEN  1 THEN 114 WHEN  2 THEN 104 WHEN  3 THEN 109 WHEN  4 THEN 108
        WHEN  5 THEN 105 WHEN  6 THEN 107 WHEN  7 THEN 102 WHEN  8 THEN  99
        WHEN  9 THEN 100 WHEN 10 THEN 112 WHEN 11 THEN 111 WHEN 12 THEN 110
        WHEN 13 THEN 106 WHEN 14 THEN 113 WHEN 15 THEN 103 ELSE            101
    END,
    national_submitted_on      = stop + INTERVAL '2 days 10:00:00',
    national_review_status     = 'Approved',
    national_reviewed_by       = 1,
    national_reviewed_on       = stop + INTERVAL '4 days 14:00:00',
    last_updated_on            = stop + INTERVAL '4 days 14:05:00'
WHERE  start               < '2026-03-30'
  AND  report_status        = 'Approved'
  AND  facility_review_status IS NULL;

-- ============================================================
-- SECTION 6 — WEEK 12  (Mar 30 – Apr 05)
--             Nationally submitted; mixed national review outcome.
--             Hospitals 4 (Gulu), 9 (Lira), 13 (Moroto) → DECLINED
--             All others → APPROVED
-- ============================================================

-- 6a. Declined hospitals — cascade: report_status = 'Declined'
UPDATE clinician_app.weeklyreport SET
    report_status              = 'Declined',
    facility_review_status     = 'Approved',
    facility_reviewed_by       = CASE hospital
                                     WHEN 4 THEN 108
                                     WHEN 9 THEN 100
                                     ELSE        106   -- 13 → 106
                                 END,
    facility_reviewed_on       = '2026-04-07 09:00:00',
    national_submission_status = 'Submitted',
    national_submitted_by      = CASE hospital
                                     WHEN 4 THEN 108
                                     WHEN 9 THEN 100
                                     ELSE        106
                                 END,
    national_submitted_on      = '2026-04-07 11:00:00',
    national_review_status     = 'Declined',
    national_reviewed_by       = 1,
    national_reviewed_on       = '2026-04-09 15:00:00',
    last_updated_on            = '2026-04-09 15:05:00'
WHERE  start      = '2026-03-30'
  AND  hospital  IN (4, 9, 13);

-- 6b. Approved hospitals — full pipeline
UPDATE clinician_app.weeklyreport SET
    facility_review_status     = 'Approved',
    facility_reviewed_by       = CASE hospital
        WHEN  1 THEN 114 WHEN  2 THEN 104 WHEN  3 THEN 109
        WHEN  5 THEN 105 WHEN  6 THEN 107 WHEN  7 THEN 102 WHEN  8 THEN  99
        WHEN 10 THEN 112 WHEN 11 THEN 111 WHEN 12 THEN 110
        WHEN 14 THEN 113 WHEN 15 THEN 103 ELSE 101
    END,
    facility_reviewed_on       = '2026-04-07 09:30:00',
    national_submission_status = 'Submitted',
    national_submitted_by      = CASE hospital
        WHEN  1 THEN 114 WHEN  2 THEN 104 WHEN  3 THEN 109
        WHEN  5 THEN 105 WHEN  6 THEN 107 WHEN  7 THEN 102 WHEN  8 THEN  99
        WHEN 10 THEN 112 WHEN 11 THEN 111 WHEN 12 THEN 110
        WHEN 14 THEN 113 WHEN 15 THEN 103 ELSE 101
    END,
    national_submitted_on      = '2026-04-07 11:30:00',
    national_review_status     = 'Approved',
    national_reviewed_by       = 1,
    national_reviewed_on       = '2026-04-09 16:00:00',
    last_updated_on            = '2026-04-09 16:05:00'
WHERE  start       = '2026-03-30'
  AND  hospital   NOT IN (4, 9, 13)
  AND  report_status IN ('Approved', 'Submitted')
  AND  facility_review_status IS NULL;

-- Upgrade Submitted → Approved for nationally-approved hospitals
UPDATE clinician_app.weeklyreport SET
    report_status   = 'Approved',
    last_updated_on = '2026-04-09 16:10:00'
WHERE  start                    = '2026-03-30'
  AND  hospital                NOT IN (4, 9, 13)
  AND  report_status            = 'Submitted'
  AND  national_review_status   = 'Approved';

-- ============================================================
-- SECTION 7 — WEEK 13  (Apr 06 – Apr 12)
--             Facility approved + nationally submitted.
--             Hospital 5 (Hoima) → facility DECLINED (for testing).
-- ============================================================

-- 7a. Hospital 5 — facility declined; reports revert to Declined
UPDATE clinician_app.weeklyreport SET
    report_status          = 'Declined',
    facility_review_status = 'Declined',
    facility_reviewed_by   = 105,
    facility_reviewed_on   = '2026-04-14 11:00:00',
    last_updated_on        = '2026-04-14 11:05:00'
WHERE  start    = '2026-04-06'
  AND  hospital = 5
  AND  facility_review_status IS NULL;

-- 7b. All other hospitals — facility approved + nationally submitted
UPDATE clinician_app.weeklyreport SET
    facility_review_status     = 'Approved',
    facility_reviewed_by       = CASE hospital
        WHEN  1 THEN 114 WHEN  2 THEN 104 WHEN  3 THEN 109 WHEN  4 THEN 108
        WHEN  6 THEN 107 WHEN  7 THEN 102 WHEN  8 THEN  99
        WHEN  9 THEN 100 WHEN 10 THEN 112 WHEN 11 THEN 111 WHEN 12 THEN 110
        WHEN 13 THEN 106 WHEN 14 THEN 113 WHEN 15 THEN 103 ELSE 101
    END,
    facility_reviewed_on       = '2026-04-14 09:00:00',
    national_submission_status = 'Submitted',
    national_submitted_by      = CASE hospital
        WHEN  1 THEN 114 WHEN  2 THEN 104 WHEN  3 THEN 109 WHEN  4 THEN 108
        WHEN  6 THEN 107 WHEN  7 THEN 102 WHEN  8 THEN  99
        WHEN  9 THEN 100 WHEN 10 THEN 112 WHEN 11 THEN 111 WHEN 12 THEN 110
        WHEN 13 THEN 106 WHEN 14 THEN 113 WHEN 15 THEN 103 ELSE 101
    END,
    national_submitted_on      = '2026-04-14 10:00:00',
    last_updated_on            = '2026-04-14 10:05:00'
WHERE  start      = '2026-04-06'
  AND  hospital  != 5
  AND  report_status IN ('Approved', 'Submitted')
  AND  facility_review_status IS NULL;

-- Upgrade Submitted → Approved for nationally submitted hospitals
UPDATE clinician_app.weeklyreport SET
    report_status   = 'Approved',
    last_updated_on = '2026-04-14 10:10:00'
WHERE  start                       = '2026-04-06'
  AND  hospital                   != 5
  AND  report_status               = 'Submitted'
  AND  national_submission_status  = 'Submitted';

-- ============================================================
-- SECTION 8 — WEEK 14  (Apr 13 – Apr 19)  CURRENT WEEK
--             Individual reports reviewed at facility level.
--             No national batch submission yet.
-- ============================================================

-- 8a. Mark Approved individual reports as facility-reviewed
UPDATE clinician_app.weeklyreport SET
    facility_review_status = 'Approved',
    facility_reviewed_by   = CASE hospital
        WHEN  1 THEN 114 WHEN  2 THEN 104 WHEN  3 THEN 109 WHEN  4 THEN 108
        WHEN  5 THEN 105 WHEN  6 THEN 107 WHEN  7 THEN 102 WHEN  8 THEN  99
        WHEN  9 THEN 100 WHEN 10 THEN 112 WHEN 11 THEN 111 WHEN 12 THEN 110
        WHEN 13 THEN 106 WHEN 14 THEN 113 WHEN 15 THEN 103 ELSE 101
    END,
    facility_reviewed_on   = '2026-04-17 09:00:00',
    last_updated_on        = '2026-04-17 09:05:00'
WHERE  start          = '2026-04-13'
  AND  report_status  = 'Approved'
  AND  facility_review_status IS NULL;

-- 8b. Submitted (staff) reports for current week remain unreviewed at facility level
--     (facility_review_status stays NULL — these are in the review queue).

-- ============================================================
-- SECTION 9 — WEEK 15 (Apr 20) and beyond
--             Staff submitted or draft; no workflow action needed.
-- ============================================================
-- No changes — leave data as-is (Submitted / Draft).

-- ============================================================
-- SECTION 10 — GRANT PERMISSIONS (safety net for any new rows)
-- ============================================================
GRANT ALL PRIVILEGES ON ALL TABLES    IN SCHEMA clinician_app TO warehouse;
GRANT ALL PRIVILEGES ON ALL SEQUENCES IN SCHEMA clinician_app TO warehouse;

COMMIT;

-- ============================================================
-- VERIFICATION QUERIES
-- Run these after committing to confirm results.
-- ============================================================
SELECT
    start,
    COALESCE(facility_review_status,     '(none)') AS fac_review,
    COALESCE(national_submission_status, '(none)') AS nat_submit,
    COALESCE(national_review_status,     '(none)') AS nat_review,
    report_status,
    COUNT(*) AS cnt
FROM  clinician_app.weeklyreport
GROUP BY 1, 2, 3, 4, 5
ORDER BY 1, 2, 3, 4, 5;

SELECT
    'employees with employee_number' AS check_item,
    COUNT(*) FILTER (WHERE employee_number IS NOT NULL) AS populated,
    COUNT(*) AS total
FROM clinician_app.employees
UNION ALL
SELECT
    'staffleave with reviewed_on', COUNT(*) FILTER (WHERE reviewed_on IS NOT NULL), COUNT(*)
FROM clinician_app.staffleave
UNION ALL
SELECT
    'weeklyreport with days_worked', COUNT(*) FILTER (WHERE days_worked IS NOT NULL), COUNT(*)
FROM clinician_app.weeklyreport
UNION ALL
SELECT
    'weeklyreport with submitted_on', COUNT(*) FILTER (WHERE submitted_on IS NOT NULL), COUNT(*)
FROM clinician_app.weeklyreport;
