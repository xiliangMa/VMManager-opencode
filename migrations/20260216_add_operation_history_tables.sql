-- Login History Table
CREATE TABLE IF NOT EXISTS login_histories (
    id UUID PRIMARY KEY,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    login_type VARCHAR(20) NOT NULL DEFAULT 'password',
    ip_address VARCHAR(45),
    user_agent TEXT,
    location VARCHAR(200),
    device_info JSONB,
    status VARCHAR(20) NOT NULL DEFAULT 'success',
    failure_reason TEXT,
    logout_at TIMESTAMP,
    session_duration INTEGER,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_login_histories_user_id ON login_histories(user_id);
CREATE INDEX idx_login_histories_created_at ON login_histories(created_at);
CREATE INDEX idx_login_histories_status ON login_histories(status);
CREATE INDEX idx_login_histories_ip_address ON login_histories(ip_address);

-- Resource Change History Table
CREATE TABLE IF NOT EXISTS resource_change_histories (
    id UUID PRIMARY KEY,
    resource_type VARCHAR(50) NOT NULL,
    resource_id UUID NOT NULL,
    resource_name VARCHAR(200),
    action VARCHAR(50) NOT NULL,
    old_value JSONB,
    new_value JSONB,
    changed_by UUID REFERENCES users(id) ON DELETE SET NULL,
    change_reason TEXT,
    ip_address VARCHAR(45),
    user_agent TEXT,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_resource_change_histories_resource ON resource_change_histories(resource_type, resource_id);
CREATE INDEX idx_resource_change_histories_changed_by ON resource_change_histories(changed_by);
CREATE INDEX idx_resource_change_histories_created_at ON resource_change_histories(created_at);
CREATE INDEX idx_resource_change_histories_action ON resource_change_histories(action);

-- VM Operation History Table
CREATE TABLE IF NOT EXISTS vm_operation_histories (
    id UUID PRIMARY KEY,
    vm_id UUID NOT NULL REFERENCES virtual_machines(id) ON DELETE CASCADE,
    operation VARCHAR(50) NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'pending',
    started_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    completed_at TIMESTAMP,
    duration INTEGER,
    triggered_by UUID REFERENCES users(id) ON DELETE SET NULL,
    ip_address VARCHAR(45),
    user_agent TEXT,
    request_params JSONB,
    response_data JSONB,
    error_message TEXT
);

CREATE INDEX idx_vm_operation_histories_vm_id ON vm_operation_histories(vm_id);
CREATE INDEX idx_vm_operation_histories_triggered_by ON vm_operation_histories(triggered_by);
CREATE INDEX idx_vm_operation_histories_status ON vm_operation_histories(status);
CREATE INDEX idx_vm_operation_histories_started_at ON vm_operation_histories(started_at);
