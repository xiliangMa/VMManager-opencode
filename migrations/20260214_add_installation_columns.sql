-- Add installation-related columns to virtual_machines table
ALTER TABLE virtual_machines ADD COLUMN IF NOT EXISTS is_installed BOOLEAN DEFAULT FALSE;
ALTER TABLE virtual_machines ADD COLUMN IF NOT EXISTS install_status VARCHAR(50) DEFAULT '';
ALTER TABLE virtual_machines ADD COLUMN IF NOT EXISTS install_progress INTEGER DEFAULT 0;
ALTER TABLE virtual_machines ADD COLUMN IF NOT EXISTS agent_installed BOOLEAN DEFAULT FALSE;

-- Add installation-related columns to vm_templates table
ALTER TABLE vm_templates ADD COLUMN IF NOT EXISTS iso_path VARCHAR(500);
ALTER TABLE vm_templates ADD COLUMN IF NOT EXISTS install_script TEXT;
ALTER TABLE vm_templates ADD COLUMN IF NOT EXISTS post_install_script TEXT;
