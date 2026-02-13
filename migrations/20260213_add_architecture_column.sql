-- Add architecture column to virtual_machines table
ALTER TABLE virtual_machines ADD COLUMN IF NOT EXISTS architecture VARCHAR(20) DEFAULT 'x86_64';
