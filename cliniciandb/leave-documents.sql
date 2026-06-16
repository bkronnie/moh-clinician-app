-- Migration: add leave_documents table for supporting document uploads
-- Run once against the clinician_app schema.
CREATE TABLE IF NOT EXISTS clinician_app.leave_documents (
    doc_id BIGSERIAL PRIMARY KEY,
    leave_id BIGINT NOT NULL REFERENCES clinician_app.staffleave(leave_id) ON DELETE CASCADE,
    original_name TEXT NOT NULL,
    stored_name TEXT NOT NULL,
    file_size BIGINT NOT NULL DEFAULT 0,
    mime_type TEXT NOT NULL DEFAULT '',
    uploaded_at TIMESTAMP NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_leave_documents_leave_id ON clinician_app.leave_documents(leave_id);