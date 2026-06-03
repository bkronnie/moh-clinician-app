-- =============================================================================
-- Seed: Entebbe Regional Referral Hospital — May 2026 daily report records
--
-- Facility: 2 (Entebbe Regional Referral Hospital)
-- Employees:
--   1006  Moses Kato          dept 5 (Admin)                  user 6
--   1007  Allan Ssembatya     dept 1 (Surgery)                user 7
--   1008  Juliet Nalwadda     dept 2 (Internal Medicine)      user 8
--   1009  Mark Nsubuga        dept 3 (Paediatrics)            user 9
--   1010  Allen Nakato        dept 4 (Obstetrics & Gynae)     user 10
--
-- Status tiers:
--   May 1  (Fri)       : Submitted + Approved
--   May 4–8  (Week 1)  : Submitted + Approved
--   May 11–15 (Week 2) : Submitted + Approved
--   May 18–22 (Week 3) : Submitted (not yet approved)
--   May 25–29 (Week 4) : Pending (not yet submitted)
-- =============================================================================
BEGIN;
-- Step 1: Drop residual weekly-only constraints that block daily inserts
--         (these were left NOT VALID by the daily migration but still enforce on new rows)
ALTER TABLE clinician_app.weeklyreport DROP CONSTRAINT IF EXISTS ck_weeklyreport_start_monday;
ALTER TABLE clinician_app.weeklyreport DROP CONSTRAINT IF EXISTS ck_weeklyreport_start_stop_6_days;
-- Step 2: Truncate the report table (cascade through any dependent FKs)
TRUNCATE clinician_app.weeklyreport RESTART IDENTITY CASCADE;
-- Step 3: Create a temporary sequence to generate unique IDs
--         (weeklyreport.id has no attached sequence; the app uses MAX(id)+1)
DROP SEQUENCE IF EXISTS clinician_app._seed_id_seq;
CREATE SEQUENCE clinician_app._seed_id_seq START 1;
-- =============================================================================
-- Step 4: Insert records for each employee using generate_series
-- =============================================================================
-- Helper: all working weekdays in May 2026
-- May 1 (Fri), May 4–8, May 11–15, May 18–22, May 25–29 = 21 records each
-- -------------------------------------------------------------------------
-- Admin — Moses Kato (emp 1006, dept 5, user 6) — attendance only
-- -------------------------------------------------------------------------
INSERT INTO clinician_app.weeklyreport (
        id,
        hospital,
        department,
        employee,
        start,
        stop,
        attendance,
        days_worked,
        created_on,
        last_updated_on,
        entered_by,
        submitted_by,
        approved_by,
        submit_status,
        report_status
    )
SELECT nextval('clinician_app._seed_id_seq'),
    2,
    5,
    1006,
    d::date,
    d::date,
    1,
    to_char(d, 'Dy'),
    d::date::timestamp,
    d::date::timestamp,
    6,
    CASE
        WHEN d::date <= '2026-05-22' THEN 6
        ELSE NULL
    END,
    CASE
        WHEN d::date <= '2026-05-15' THEN 104
        ELSE NULL
    END,
    CASE
        WHEN d::date <= '2026-05-22' THEN 'Submitted'
        ELSE NULL
    END,
    CASE
        WHEN d::date <= '2026-05-15' THEN 'Approved'
        ELSE NULL
    END
FROM generate_series(
        '2026-05-01'::date,
        '2026-05-31'::date,
        '1 day'::interval
    ) d
WHERE EXTRACT(
        DOW
        FROM d
    ) BETWEEN 1 AND 5;
-- -------------------------------------------------------------------------
-- Surgery — Allan Ssembatya (emp 1007, dept 1, user 7)
-- -------------------------------------------------------------------------
INSERT INTO clinician_app.weeklyreport (
        id,
        hospital,
        department,
        employee,
        start,
        stop,
        attendance,
        days_worked,
        ward_rounds,
        patients_reviewed,
        theatre_days,
        elective,
        emergency,
        opd_clinics,
        opd_patients,
        teaching_rounds,
        students_taught,
        mortality_reviews,
        maternal,
        perinatal,
        surgical,
        medical,
        paed,
        labs_requests,
        imaging_requests,
        created_on,
        last_updated_on,
        entered_by,
        submitted_by,
        approved_by,
        submit_status,
        report_status
    )
SELECT nextval('clinician_app._seed_id_seq'),
    2,
    1,
    1007,
    d::date,
    d::date,
    1,
    to_char(d, 'Dy'),
    2,
    8 + (
        EXTRACT(
            DOW
            FROM d
        )::int % 5
    ),
    CASE
        WHEN EXTRACT(
            DOW
            FROM d
        ) IN (1, 3) THEN 1
        ELSE 0
    END,
    CASE
        WHEN EXTRACT(
            DOW
            FROM d
        ) IN (1, 3) THEN 2
        ELSE 0
    END,
    CASE
        WHEN EXTRACT(
            DOW
            FROM d
        ) IN (1, 3) THEN 1
        ELSE 0
    END,
    1,
    5 + (
        EXTRACT(
            DOW
            FROM d
        )::int % 5
    ),
    CASE
        WHEN EXTRACT(
            DOW
            FROM d
        ) IN (2, 4) THEN 1
        ELSE 0
    END,
    CASE
        WHEN EXTRACT(
            DOW
            FROM d
        ) IN (2, 4) THEN 3
        ELSE 0
    END,
    CASE
        WHEN EXTRACT(
            DOW
            FROM d
        ) = 5 THEN 1
        ELSE 0
    END,
    0,
    0,
    CASE
        WHEN EXTRACT(
            DOW
            FROM d
        ) = 5 THEN 1
        ELSE 0
    END,
    0,
    0,
    3 + (
        EXTRACT(
            DOW
            FROM d
        )::int % 3
    ),
    2 + (
        EXTRACT(
            DOW
            FROM d
        )::int % 2
    ),
    d::date::timestamp,
    d::date::timestamp,
    7,
    CASE
        WHEN d::date <= '2026-05-22' THEN 7
        ELSE NULL
    END,
    CASE
        WHEN d::date <= '2026-05-15' THEN 104
        ELSE NULL
    END,
    CASE
        WHEN d::date <= '2026-05-22' THEN 'Submitted'
        ELSE NULL
    END,
    CASE
        WHEN d::date <= '2026-05-15' THEN 'Approved'
        ELSE NULL
    END
FROM generate_series(
        '2026-05-01'::date,
        '2026-05-31'::date,
        '1 day'::interval
    ) d
WHERE EXTRACT(
        DOW
        FROM d
    ) BETWEEN 1 AND 5;
-- -------------------------------------------------------------------------
-- Internal Medicine — Juliet Nalwadda (emp 1008, dept 2, user 8)
-- -------------------------------------------------------------------------
INSERT INTO clinician_app.weeklyreport (
        id,
        hospital,
        department,
        employee,
        start,
        stop,
        attendance,
        days_worked,
        ward_rounds,
        patients_reviewed,
        opd_clinics,
        opd_patients,
        teaching_rounds,
        students_taught,
        mortality_reviews,
        maternal,
        perinatal,
        surgical,
        medical,
        paed,
        labs_requests,
        imaging_requests,
        created_on,
        last_updated_on,
        entered_by,
        submitted_by,
        approved_by,
        submit_status,
        report_status
    )
SELECT nextval('clinician_app._seed_id_seq'),
    2,
    2,
    1008,
    d::date,
    d::date,
    1,
    to_char(d, 'Dy'),
    2,
    10 + (
        EXTRACT(
            DOW
            FROM d
        )::int % 6
    ),
    1,
    6 + (
        EXTRACT(
            DOW
            FROM d
        )::int % 5
    ),
    CASE
        WHEN EXTRACT(
            DOW
            FROM d
        ) IN (3, 5) THEN 1
        ELSE 0
    END,
    CASE
        WHEN EXTRACT(
            DOW
            FROM d
        ) IN (3, 5) THEN 4
        ELSE 0
    END,
    CASE
        WHEN EXTRACT(
            DOW
            FROM d
        ) = 5 THEN 1
        ELSE 0
    END,
    0,
    CASE
        WHEN EXTRACT(
            DOW
            FROM d
        ) = 5 THEN 1
        ELSE 0
    END,
    0,
    CASE
        WHEN EXTRACT(
            DOW
            FROM d
        ) = 5 THEN 1
        ELSE 0
    END,
    0,
    4 + (
        EXTRACT(
            DOW
            FROM d
        )::int % 3
    ),
    1 + (
        EXTRACT(
            DOW
            FROM d
        )::int % 2
    ),
    d::date::timestamp,
    d::date::timestamp,
    8,
    CASE
        WHEN d::date <= '2026-05-22' THEN 8
        ELSE NULL
    END,
    CASE
        WHEN d::date <= '2026-05-15' THEN 104
        ELSE NULL
    END,
    CASE
        WHEN d::date <= '2026-05-22' THEN 'Submitted'
        ELSE NULL
    END,
    CASE
        WHEN d::date <= '2026-05-15' THEN 'Approved'
        ELSE NULL
    END
FROM generate_series(
        '2026-05-01'::date,
        '2026-05-31'::date,
        '1 day'::interval
    ) d
WHERE EXTRACT(
        DOW
        FROM d
    ) BETWEEN 1 AND 5;
-- -------------------------------------------------------------------------
-- Paediatrics — Mark Nsubuga (emp 1009, dept 3, user 9)
-- -------------------------------------------------------------------------
INSERT INTO clinician_app.weeklyreport (
        id,
        hospital,
        department,
        employee,
        start,
        stop,
        attendance,
        days_worked,
        ward_rounds,
        patients_reviewed,
        theatre_days,
        elective,
        emergency,
        opd_clinics,
        opd_patients,
        teaching_rounds,
        students_taught,
        mortality_reviews,
        maternal,
        perinatal,
        surgical,
        medical,
        paed,
        labs_requests,
        imaging_requests,
        created_on,
        last_updated_on,
        entered_by,
        submitted_by,
        approved_by,
        submit_status,
        report_status
    )
SELECT nextval('clinician_app._seed_id_seq'),
    2,
    3,
    1009,
    d::date,
    d::date,
    1,
    to_char(d, 'Dy'),
    3,
    12 + (
        EXTRACT(
            DOW
            FROM d
        )::int % 7
    ),
    CASE
        WHEN EXTRACT(
            DOW
            FROM d
        ) = 4 THEN 1
        ELSE 0
    END,
    CASE
        WHEN EXTRACT(
            DOW
            FROM d
        ) = 4 THEN 1
        ELSE 0
    END,
    CASE
        WHEN EXTRACT(
            DOW
            FROM d
        ) = 4 THEN 1
        ELSE 0
    END,
    1,
    8 + (
        EXTRACT(
            DOW
            FROM d
        )::int % 5
    ),
    CASE
        WHEN EXTRACT(
            DOW
            FROM d
        ) IN (1, 4) THEN 1
        ELSE 0
    END,
    CASE
        WHEN EXTRACT(
            DOW
            FROM d
        ) IN (1, 4) THEN 5
        ELSE 0
    END,
    CASE
        WHEN EXTRACT(
            DOW
            FROM d
        ) = 5 THEN 1
        ELSE 0
    END,
    0,
    0,
    0,
    0,
    CASE
        WHEN EXTRACT(
            DOW
            FROM d
        ) = 5 THEN 1
        ELSE 0
    END,
    5 + (
        EXTRACT(
            DOW
            FROM d
        )::int % 4
    ),
    2 + (
        EXTRACT(
            DOW
            FROM d
        )::int % 3
    ),
    d::date::timestamp,
    d::date::timestamp,
    9,
    CASE
        WHEN d::date <= '2026-05-22' THEN 9
        ELSE NULL
    END,
    CASE
        WHEN d::date <= '2026-05-15' THEN 104
        ELSE NULL
    END,
    CASE
        WHEN d::date <= '2026-05-22' THEN 'Submitted'
        ELSE NULL
    END,
    CASE
        WHEN d::date <= '2026-05-15' THEN 'Approved'
        ELSE NULL
    END
FROM generate_series(
        '2026-05-01'::date,
        '2026-05-31'::date,
        '1 day'::interval
    ) d
WHERE EXTRACT(
        DOW
        FROM d
    ) BETWEEN 1 AND 5;
-- -------------------------------------------------------------------------
-- Obstetrics & Gynaecology — Allen Nakato (emp 1010, dept 4, user 10)
-- -------------------------------------------------------------------------
INSERT INTO clinician_app.weeklyreport (
        id,
        hospital,
        department,
        employee,
        start,
        stop,
        attendance,
        days_worked,
        ward_rounds,
        patients_reviewed,
        theatre_days,
        elective,
        emergency,
        anc_patients,
        opd_clinics,
        opd_patients,
        teaching_rounds,
        students_taught,
        mortality_reviews,
        maternal,
        perinatal,
        surgical,
        medical,
        paed,
        labs_requests,
        imaging_requests,
        created_on,
        last_updated_on,
        entered_by,
        submitted_by,
        approved_by,
        submit_status,
        report_status
    )
SELECT nextval('clinician_app._seed_id_seq'),
    2,
    4,
    1010,
    d::date,
    d::date,
    1,
    to_char(d, 'Dy'),
    2,
    9 + (
        EXTRACT(
            DOW
            FROM d
        )::int % 5
    ),
    CASE
        WHEN EXTRACT(
            DOW
            FROM d
        ) IN (2, 5) THEN 1
        ELSE 0
    END,
    CASE
        WHEN EXTRACT(
            DOW
            FROM d
        ) IN (2, 5) THEN 2
        ELSE 0
    END,
    CASE
        WHEN EXTRACT(
            DOW
            FROM d
        ) IN (2, 5) THEN 1
        ELSE 0
    END,
    4 + (
        EXTRACT(
            DOW
            FROM d
        )::int % 4
    ),
    1,
    5 + (
        EXTRACT(
            DOW
            FROM d
        )::int % 4
    ),
    CASE
        WHEN EXTRACT(
            DOW
            FROM d
        ) = 3 THEN 1
        ELSE 0
    END,
    CASE
        WHEN EXTRACT(
            DOW
            FROM d
        ) = 3 THEN 4
        ELSE 0
    END,
    CASE
        WHEN EXTRACT(
            DOW
            FROM d
        ) = 5 THEN 1
        ELSE 0
    END,
    CASE
        WHEN EXTRACT(
            DOW
            FROM d
        ) = 5 THEN 1
        ELSE 0
    END,
    CASE
        WHEN EXTRACT(
            DOW
            FROM d
        ) = 5 THEN 1
        ELSE 0
    END,
    0,
    0,
    0,
    3 + (
        EXTRACT(
            DOW
            FROM d
        )::int % 3
    ),
    2 + (
        EXTRACT(
            DOW
            FROM d
        )::int % 2
    ),
    d::date::timestamp,
    d::date::timestamp,
    10,
    CASE
        WHEN d::date <= '2026-05-22' THEN 10
        ELSE NULL
    END,
    CASE
        WHEN d::date <= '2026-05-15' THEN 104
        ELSE NULL
    END,
    CASE
        WHEN d::date <= '2026-05-22' THEN 'Submitted'
        ELSE NULL
    END,
    CASE
        WHEN d::date <= '2026-05-15' THEN 'Approved'
        ELSE NULL
    END
FROM generate_series(
        '2026-05-01'::date,
        '2026-05-31'::date,
        '1 day'::interval
    ) d
WHERE EXTRACT(
        DOW
        FROM d
    ) BETWEEN 1 AND 5;
-- Step 5: Drop temporary sequence
DROP SEQUENCE clinician_app._seed_id_seq;
-- =============================================================================
-- Step 6: Verify summary
-- =============================================================================
SELECT e.id AS emp_id,
    e.fname || ' ' || e.lname AS name,
    d.d_name AS department,
    COUNT(*) AS total_records,
    SUM(
        CASE
            WHEN wr.submit_status = 'Submitted' THEN 1
            ELSE 0
        END
    ) AS submitted,
    SUM(
        CASE
            WHEN wr.report_status = 'Approved' THEN 1
            ELSE 0
        END
    ) AS approved
FROM clinician_app.weeklyreport wr
    JOIN clinician_app.employees e ON e.id = wr.employee
    JOIN clinician_app.departments d ON d.id = wr.department
GROUP BY e.id,
    e.fname,
    e.lname,
    d.d_name
ORDER BY e.id;
COMMIT;