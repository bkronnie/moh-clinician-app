-- One-time maintenance script: remove duplicate weekly reports and enforce
-- uniqueness on (employee, start, stop).
--
-- Run with a role that has DELETE on public.weeklyreport and CREATE on schema public.
-- Example:
-- psql -h 127.0.0.1 -p 5432 -U <privileged_user> -d clinician -v ON_ERROR_STOP=1 -f cliniciandb/weeklyreport-dedup-and-unique-index.sql

BEGIN;

-- 1) Snapshot duplicate counts before cleanup.
WITH dupes AS (
    SELECT employee, start, stop, COUNT(*) AS cnt
    FROM public.weeklyreport
    WHERE employee IS NOT NULL
      AND start IS NOT NULL
      AND stop IS NOT NULL
    GROUP BY employee, start, stop
    HAVING COUNT(*) > 1
)
SELECT COUNT(*) AS duplicate_groups, COALESCE(SUM(cnt - 1), 0) AS removable_rows
FROM dupes;

-- 2) Delete duplicates, keeping the best row per employee/week.
-- Priority kept: Approved > Submitted > others;
-- then records explicitly submitted; then latest update/create timestamp; then highest id.
WITH ranked AS (
    SELECT
        id,
        ROW_NUMBER() OVER (
            PARTITION BY employee, start, stop
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
      AND start IS NOT NULL
      AND stop IS NOT NULL
), deleted AS (
    DELETE FROM public.weeklyreport w
    USING ranked r
    WHERE w.id = r.id
      AND r.rn > 1
    RETURNING w.id
)
SELECT COUNT(*) AS deleted_rows FROM deleted;

-- 3) Enforce schema-level uniqueness for non-null employee/week keys.
CREATE UNIQUE INDEX IF NOT EXISTS ux_weeklyreport_employee_start_stop
ON public.weeklyreport(employee, start, stop)
WHERE employee IS NOT NULL AND start IS NOT NULL AND stop IS NOT NULL;

-- 4) Verify cleanup and index.
WITH dupes AS (
    SELECT employee, start, stop, COUNT(*) AS cnt
    FROM public.weeklyreport
    WHERE employee IS NOT NULL
      AND start IS NOT NULL
      AND stop IS NOT NULL
    GROUP BY employee, start, stop
    HAVING COUNT(*) > 1
)
SELECT COUNT(*) AS duplicate_groups_after, COALESCE(SUM(cnt - 1), 0) AS removable_rows_after
FROM dupes;

SELECT indexname, indexdef
FROM pg_indexes
WHERE schemaname = 'public'
  AND tablename = 'weeklyreport'
  AND indexname = 'ux_weeklyreport_employee_start_stop';

COMMIT;
