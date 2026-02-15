-- Create isos table for ISO file management
CREATE TABLE IF NOT EXISTS isos (
    id UUID PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    file_name VARCHAR(255) NOT NULL,
    file_size BIGINT NOT NULL,
    iso_path VARCHAR(500) NOT NULL,
    md5 VARCHAR(32),
    sha256 VARCHAR(64),
    os_type VARCHAR(50),
    os_version VARCHAR(50),
    architecture VARCHAR(20) DEFAULT 'x86_64',
    status VARCHAR(20) DEFAULT 'active',
    uploaded_by UUID,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Create iso_uploads table for upload progress tracking
CREATE TABLE IF NOT EXISTS iso_uploads (
    id UUID PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    file_name VARCHAR(255) NOT NULL,
    file_size BIGINT NOT NULL,
    architecture VARCHAR(20),
    os_type VARCHAR(50),
    os_version VARCHAR(50),
    upload_path VARCHAR(500),
    temp_path VARCHAR(500),
    status VARCHAR(20) DEFAULT 'uploading',
    progress INTEGER DEFAULT 0,
    error_message TEXT,
    uploaded_by UUID,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    completed_at TIMESTAMP
);

-- Create indexes
CREATE INDEX IF NOT EXISTS idx_isos_status ON isos(status);
CREATE INDEX IF NOT EXISTS idx_isos_architecture ON isos(architecture);
CREATE INDEX IF NOT EXISTS idx_isos_os_type ON isos(os_type);
CREATE INDEX IF NOT EXISTS idx_isos_uploaded_by ON isos(uploaded_by);
CREATE INDEX IF NOT EXISTS idx_iso_uploads_status ON iso_uploads(status);
CREATE INDEX IF NOT EXISTS idx_iso_uploads_uploaded_by ON iso_uploads(uploaded_by);

-- Add iso_id column to virtual_machines table for ISO-based installation
ALTER TABLE virtual_machines ADD COLUMN IF NOT EXISTS iso_id UUID;
ALTER TABLE virtual_machines ADD COLUMN IF NOT EXISTS installation_mode VARCHAR(20) DEFAULT 'template';

-- Add comment
COMMENT ON TABLE isos IS 'ISO image files for VM installation';
COMMENT ON TABLE iso_uploads IS 'ISO upload progress tracking';
COMMENT ON COLUMN virtual_machines.installation_mode IS 'Installation mode: template or iso';
COMMENT ON COLUMN virtual_machines.iso_id IS 'Reference to ISO used for installation';
