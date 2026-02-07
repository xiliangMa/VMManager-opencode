-- Users table
CREATE TABLE IF NOT EXISTS users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    username VARCHAR(50) UNIQUE NOT NULL,
    email VARCHAR(100) UNIQUE NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    role VARCHAR(20) DEFAULT 'user',
    is_active BOOLEAN DEFAULT true,
    avatar_url VARCHAR(500),
    language VARCHAR(10) DEFAULT 'zh-CN',
    timezone VARCHAR(50) DEFAULT 'Asia/Shanghai',
    quota_cpu INT DEFAULT 4,
    quota_memory INT DEFAULT 8192,
    quota_disk INT DEFAULT 100,
    quota_vm_count INT DEFAULT 5,
    last_login_at TIMESTAMP,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

-- VM Templates table
CREATE TABLE IF NOT EXISTS vm_templates (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(100) NOT NULL,
    description TEXT,
    os_type VARCHAR(50) NOT NULL,
    os_version VARCHAR(50),
    architecture VARCHAR(20) DEFAULT 'arm64',
    format VARCHAR(20) DEFAULT 'qcow2',
    cpu_min INT DEFAULT 1,
    cpu_max INT DEFAULT 4,
    memory_min INT DEFAULT 1024,
    memory_max INT DEFAULT 8192,
    disk_min INT DEFAULT 20,
    disk_max INT DEFAULT 500,
    template_path VARCHAR(500) NOT NULL,
    icon_url VARCHAR(500),
    screenshot_urls TEXT[],
    disk_size BIGINT NOT NULL,
    is_public BOOLEAN DEFAULT true,
    is_active BOOLEAN DEFAULT true,
    downloads INT DEFAULT 0,
    created_by UUID REFERENCES users(id),
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

-- Virtual Machines table
CREATE TABLE IF NOT EXISTS virtual_machines (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(100) NOT NULL,
    description TEXT,
    template_id UUID REFERENCES vm_templates(id),
    owner_id UUID REFERENCES users(id) NOT NULL,
    status VARCHAR(20) DEFAULT 'pending',
    vnc_port INT,
    vnc_password VARCHAR(20),
    spice_port INT,
    mac_address VARCHAR(17) UNIQUE,
    ip_address INET,
    gateway INET,
    dns_servers INET[],
    cpu_allocated INT NOT NULL,
    memory_allocated INT NOT NULL,
    disk_allocated INT NOT NULL,
    disk_path VARCHAR(500),
    libvirt_domain_id INT,
    libvirt_domain_uuid VARCHAR(50),
    boot_order VARCHAR(50) DEFAULT 'hd,cdrom,network',
    vcpu_hotplug BOOLEAN DEFAULT false,
    memory_hotplug BOOLEAN DEFAULT false,
    autostart BOOLEAN DEFAULT false,
    notes TEXT,
    tags TEXT[],
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),
    deleted_at TIMESTAMP
);

-- VM Stats table
CREATE TABLE IF NOT EXISTS vm_stats (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    vm_id UUID REFERENCES virtual_machines(id) NOT NULL,
    cpu_usage DECIMAL(5,2) DEFAULT 0,
    memory_usage BIGINT DEFAULT 0,
    memory_total BIGINT DEFAULT 0,
    disk_read BIGINT DEFAULT 0,
    disk_write BIGINT DEFAULT 0,
    network_rx BIGINT DEFAULT 0,
    network_tx BIGINT DEFAULT 0,
    collected_at TIMESTAMP DEFAULT NOW()
);

-- Audit Logs table
CREATE TABLE IF NOT EXISTS audit_logs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID REFERENCES users(id),
    action VARCHAR(50) NOT NULL,
    resource_type VARCHAR(50) NOT NULL,
    resource_id UUID,
    details JSONB,
    ip_address INET,
    user_agent TEXT,
    status VARCHAR(20) DEFAULT 'success',
    error_message TEXT,
    created_at TIMESTAMP DEFAULT NOW()
);

-- Template Uploads table
CREATE TABLE IF NOT EXISTS template_uploads (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(100) NOT NULL,
    description TEXT,
    file_name VARCHAR(255) NOT NULL,
    file_size BIGINT NOT NULL,
    format VARCHAR(20),
    architecture VARCHAR(20),
    upload_path VARCHAR(500),
    temp_path VARCHAR(500),
    status VARCHAR(20) DEFAULT 'uploading',
    progress INT DEFAULT 0,
    error_message TEXT,
    uploaded_by UUID REFERENCES users(id),
    created_at TIMESTAMP DEFAULT NOW(),
    completed_at TIMESTAMP
);

-- Indexes
CREATE INDEX IF NOT EXISTS idx_users_username ON users(username);
CREATE INDEX IF NOT EXISTS idx_users_email ON users(email);
CREATE INDEX IF NOT EXISTS idx_vms_owner ON virtual_machines(owner_id, status);
CREATE INDEX IF NOT EXISTS idx_vms_status ON virtual_machines(status);
CREATE INDEX IF NOT EXISTS idx_vm_stats_vm_id ON vm_stats(vm_id, collected_at DESC);
CREATE INDEX IF NOT EXISTS idx_audit_user ON audit_logs(user_id, created_at DESC);
