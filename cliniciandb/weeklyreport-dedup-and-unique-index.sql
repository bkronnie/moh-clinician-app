-- One-time maintenance script: remove duplicate weekly reports and enforce
-- uniqueness on (employee, hospital, start), enforce stop = start + 6 days,
-- and prevent overlapping date ranges for the same employee/facility.
--
-- Run with a role that has DELETE on public.weeklyreport and CREATE on schema public.
-- Example:
-- psql -h 127.0.0.1 -p 5432 -U <privileged_user> -d clinician -v ON_ERROR_STOP=1 -f cliniciandb/weeklyreport-dedup-and-unique-index.sql

BEGIN;

-- 1) Snapshot duplicate counts before cleanup.
WITH dupes AS (
    SELECT employee, hospital, start, COUNT(*) AS cnt
    FROM public.weeklyreport
    WHERE employee IS NOT NULL
      AND hospital IS NOT NULL
      AND start IS NOT NULL
    GROUP BY employee, hospital, start
    HAVING COUNT(*) > 1
)
SELECT COUNT(*) AS duplicate_groups, COALESCE(SUM(cnt - 1), 0) AS removable_rows
FROM dupes;

-- 1b) Normalize week end date to exactly 6 days after start (inclusive 7-day week window).
UPDATE public.weeklyreport
SET stop = start + INTERVAL '6 days'
WHERE start IS NOT NULL
  AND (stop IS NULL OR stop <> start + INTERVAL '6 days');

-- 2) Delete duplicates, keeping the best row per employee/week.
-- Priority kept: Approved > Submitted > others;
-- then records explicitly submitted; then latest update/create timestamp; then highest id.
WITH ranked AS (
    SELECT
        id,
        ROW_NUMBER() OVER (
      PARTITION BY employee, hospital, start
            ORDER BY
                CASE COALESCE(report_status, '')
                    WHEN 'Approved' THEN 4
                    WHEN 'Submitted' THEN 3
                    WHEN 'Declined' THEN 2
                    WHEN 'Rejected' THEN 2
                    ELSE 1
                END DESC,
                CASE COALESCE(submit_status, '')
                    WHEN 'Submitted' THEN 1
                    ELSE 0
                END DESC,
                COALESCE(last_updated_on, created_on, TIMESTAMP 'epoch') DESC,
                id DESC
        ) AS rn
    FROM public.weeklyreport
    WHERE employee IS NOT NULL
      AND hospital IS NOT NULL
      AND start IS NOT NULL
), deleted AS (
    DELETE FROM public.weeklyreport w
    USING ranked r
    WHERE w.id = r.id
      AND r.rn > 1
    RETURNING w.id
)
SELECT COUNT(*) AS deleted_rows FROM deleted;

-- 3) Enforce schema-level uniqueness for non-null employee/facility/week keys.
DROP INDEX IF EXISTS public.ux_weeklyreport_employee_start_stop;
CREATE UNIQUE INDEX IF NOT EXISTS ux_weeklyreport_employee_hospital_start
ON public.weeklyreport(employee, hospital, start)
WHERE employee IS NOT NULL AND hospital IS NOT NULL AND start IS NOT NULL;

ALTER TABLE public.weeklyreport
  DROP CONSTRAINT IF EXISTS ck_weeklyreport_start_stop_7_days;
ALTER TABLE public.weeklyreport
  DROP CONSTRAINT IF EXISTS ck_weeklyreport_start_stop_6_days;
ALTER TABLE public.weeklyreport
  ADD CONSTRAINT ck_weeklyreport_start_stop_6_days
  CHECK (start IS NULL OR stop IS NULL OR stop = start + INTERVAL '6 days');

CREATE EXTENSION IF NOT EXISTS btree_gist;
ALTER TABLE public.weeklyreport
  DROP CONSTRAINT IF EXISTS ex_weeklyreport_employee_hospital_no_overlap;
ALTER TABLE public.weeklyreport
  ADD CONSTRAINT ex_weeklyreport_employee_hospital_no_overlap
  EXCLUDE USING gist (
    employee WITH =,
    hospital WITH =,
    daterange(start, stop, '[]') WITH &&
  )
  WHERE (employee IS NOT NULL AND hospital IS NOT NULL AND start IS NOT NULL AND stop IS NOT NULL);

-- 4) Verify cleanup and index.
WITH dupes AS (
    SELECT employee, hospital, start, COUNT(*) AS cnt
    FROM public.weeklyreport
    WHERE employee IS NOT NULL
      AND hospital IS NOT NULL
      AND start IS NOT NULL
    GROUP BY employee, hospital, start
    HAVING COUNT(*) > 1
)
SELECT COUNT(*) AS duplicate_groups_after, COALESCE(SUM(cnt - 1), 0) AS removable_rows_after
FROM dupes;

SELECT COUNT(*) AS invalid_start_stop_span_rows
FROM public.weeklyreport
WHERE start IS NOT NULL
  AND stop IS NOT NULL
  AND stop <> start + INTERVAL '6 days';

SELECT COUNT(*) AS overlap_conflict_pairs
FROM public.weeklyreport a
JOIN public.weeklyreport b
  ON a.id < b.id
 AND a.employee = b.employee
 AND a.hospital = b.hospital
 AND a.start IS NOT NULL AND a.stop IS NOT NULL
 AND b.start IS NOT NULL AND b.stop IS NOT NULL
 AND daterange(a.start, a.stop, '[]') && daterange(b.start, b.stop, '[]');

SELECT indexname, indexdef
FROM pg_indexes
WHERE schemaname = 'public'
  AND tablename = 'weeklyreport'
  AND indexname = 'ux_weeklyreport_employee_hospital_start';

COMMIT;
