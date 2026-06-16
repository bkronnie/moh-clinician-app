-- Migration: add absence_reason column to weeklyreport
-- Run once against the clinician_app database.
ALTER TABLE clinician_app.weeklyreport
ADD COLUMN IF NOT EXISTS absence_reason TEXT;
COMMENT ON COLUMN clinician_app.weeklyreport.absence_reason IS 'Free-text reason provided by staff when they mark themselves absent (e.g. field duty, sick, etc.)';