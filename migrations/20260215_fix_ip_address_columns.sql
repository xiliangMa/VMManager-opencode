-- Fix IP address columns to use VARCHAR instead of INET
-- INET type in PostgreSQL doesn't accept empty strings

-- Drop existing columns and recreate as VARCHAR
ALTER TABLE virtual_machines DROP COLUMN IF EXISTS ip_address;
ALTER TABLE virtual_machines DROP COLUMN IF EXISTS gateway;
ALTER TABLE virtual_machines DROP COLUMN IF EXISTS dns_servers;

ALTER TABLE virtual_machines ADD COLUMN ip_address VARCHAR(45);
ALTER TABLE virtual_machines ADD COLUMN gateway VARCHAR(45);
ALTER TABLE virtual_machines ADD COLUMN dns_servers VARCHAR(255)[];

ALTER TABLE audit_logs DROP COLUMN IF EXISTS ip_address;
ALTER TABLE audit_logs ADD COLUMN ip_address VARCHAR(45);
