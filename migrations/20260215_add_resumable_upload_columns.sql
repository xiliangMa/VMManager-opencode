-- Add resumable upload columns to template_uploads table
ALTER TABLE template_uploads ADD COLUMN IF NOT EXISTS uploaded_chunks TEXT DEFAULT '';
ALTER TABLE template_uploads ADD COLUMN IF NOT EXISTS total_chunks INTEGER DEFAULT 0;
ALTER TABLE template_uploads ADD COLUMN IF NOT EXISTS chunk_size BIGINT DEFAULT 0;

-- Add resumable upload columns to iso_uploads table
ALTER TABLE iso_uploads ADD COLUMN IF NOT EXISTS uploaded_chunks TEXT DEFAULT '';
ALTER TABLE iso_uploads ADD COLUMN IF NOT EXISTS total_chunks INTEGER DEFAULT 0;
ALTER TABLE iso_uploads ADD COLUMN IF NOT EXISTS chunk_size BIGINT DEFAULT 0;
