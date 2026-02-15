package database

import (
	"fmt"
	"time"

	"vmmanager/config"
	"vmmanager/internal/models"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"gorm.io/gorm/schema"
)

func hashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(bytes), err
}

func NewDatabase(cfg config.DatabaseConfig) (*gorm.DB, error) {
	var db *gorm.DB
	var err error

	if cfg.Driver == "sqlite" || cfg.Host == "" {
		db, err = newSQLite(cfg)
	} else {
		db, err = newPostgreSQL(cfg)
	}

	if err != nil {
		return nil, err
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get underlying sql.DB: %w", err)
	}

	sqlDB.SetMaxOpenConns(25)
	sqlDB.SetMaxIdleConns(5)
	sqlDB.SetConnMaxLifetime(time.Hour)

	if err := sqlDB.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return db, nil
}

func newSQLite(cfg config.DatabaseConfig) (*gorm.DB, error) {
	dbPath := cfg.Path
	if dbPath == "" {
		dbPath = "vmmanager.db"
	}

	logLevel := logger.Warn
	if cfg.Debug {
		logLevel = logger.Info
	}

	gormConfig := &gorm.Config{
		NamingStrategy: schema.NamingStrategy{
			TablePrefix:   "",
			SingularTable: true,
		},
		Logger: logger.Default.LogMode(logLevel),
	}

	db, err := gorm.Open(sqlite.Open(dbPath), gormConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to SQLite: %w", err)
	}

	return db, nil
}

func newPostgreSQL(cfg config.DatabaseConfig) (*gorm.DB, error) {
	dsn := fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		cfg.Host, cfg.Port, cfg.Username, cfg.Password, cfg.Name, cfg.SSLMode,
	)

	logLevel := logger.Info
	if !cfg.Debug {
		logLevel = logger.Warn
	}

	gormConfig := &gorm.Config{
		NamingStrategy: schema.NamingStrategy{
			TablePrefix:   "",
			SingularTable: false,
		},
		Logger: logger.Default.LogMode(logLevel),
	}

	db, err := gorm.Open(postgres.Open(dsn), gormConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to PostgreSQL: %w", err)
	}

	return db, nil
}

func Migrate(db *gorm.DB) error {
	// Create tables using raw SQL to avoid Gorm AutoMigrate issues
	sql := `
	CREATE TABLE IF NOT EXISTS users (
		id UUID PRIMARY KEY,
		username VARCHAR(50) NOT NULL UNIQUE,
		email VARCHAR(100) NOT NULL UNIQUE,
		password_hash VARCHAR(255) NOT NULL,
		role VARCHAR(20) DEFAULT 'user',
		is_active BOOLEAN DEFAULT true,
		avatar_url VARCHAR(500),
		language VARCHAR(10) DEFAULT 'zh-CN',
		timezone VARCHAR(50) DEFAULT 'Asia/Shanghai',
		quota_cpu BIGINT DEFAULT 4,
		quota_memory BIGINT DEFAULT 8192,
		quota_disk BIGINT DEFAULT 100,
		quota_vm_count BIGINT DEFAULT 5,
		last_login_at TIMESTAMPTZ,
		created_at TIMESTAMPTZ,
		updated_at TIMESTAMPTZ
	);
	
	CREATE TABLE IF NOT EXISTS vm_templates (
		id UUID PRIMARY KEY,
		name VARCHAR(100) NOT NULL,
		description TEXT,
		os_type VARCHAR(50) NOT NULL,
		os_version VARCHAR(50),
		architecture VARCHAR(20) DEFAULT 'arm64',
		format VARCHAR(20) DEFAULT 'qcow2',
		cpu_min BIGINT DEFAULT 1,
		cpu_max BIGINT DEFAULT 4,
		memory_min BIGINT DEFAULT 1024,
		memory_max BIGINT DEFAULT 8192,
		disk_min BIGINT DEFAULT 20,
		disk_max BIGINT DEFAULT 500,
		template_path VARCHAR(500) NOT NULL,
		icon_url VARCHAR(500),
		screenshot_urls TEXT[],
		disk_size BIGINT NOT NULL,
		is_public BOOLEAN DEFAULT true,
		is_active BOOLEAN DEFAULT true,
		downloads BIGINT DEFAULT 0,
		created_by UUID REFERENCES users(id),
		created_at TIMESTAMPTZ,
		updated_at TIMESTAMPTZ
	);
	
	CREATE TABLE IF NOT EXISTS virtual_machines (
		id UUID PRIMARY KEY,
		name VARCHAR(100) NOT NULL,
		description TEXT,
		template_id UUID REFERENCES vm_templates(id),
		owner_id UUID NOT NULL REFERENCES users(id),
		status VARCHAR(20) DEFAULT 'stopped',
		architecture VARCHAR(20) DEFAULT 'x86_64',
		vnc_port BIGINT,
		vnc_password VARCHAR(20),
		spice_port BIGINT,
		mac_address VARCHAR(17) UNIQUE,
		ip_address VARCHAR(45),
		gateway VARCHAR(45),
		dns_servers VARCHAR(255)[],
		cpu_allocated BIGINT NOT NULL,
		memory_allocated BIGINT NOT NULL,
		disk_allocated BIGINT NOT NULL,
		disk_path VARCHAR(500),
		libvirt_domain_id BIGINT,
		libvirt_domain_uuid VARCHAR(50),
		boot_order VARCHAR(50) DEFAULT 'hd,cdrom,network',
		v_cpu_hotplug BOOLEAN DEFAULT false,
		memory_hotplug BOOLEAN DEFAULT false,
		autostart BOOLEAN DEFAULT false,
		notes TEXT,
		tags TEXT[],
		created_at TIMESTAMPTZ,
		updated_at TIMESTAMPTZ,
		deleted_at TIMESTAMPTZ
	);
	
	CREATE TABLE IF NOT EXISTS vm_stats (
		id UUID PRIMARY KEY,
		vm_id UUID NOT NULL,
		cpu_usage DECIMAL(5,2) DEFAULT 0,
		memory_usage BIGINT DEFAULT 0,
		memory_total BIGINT DEFAULT 0,
		disk_read BIGINT DEFAULT 0,
		disk_write BIGINT DEFAULT 0,
		network_rx BIGINT DEFAULT 0,
		network_tx BIGINT DEFAULT 0,
		collected_at TIMESTAMPTZ NOT NULL
	);
	
	CREATE TABLE IF NOT EXISTS audit_logs (
		id UUID PRIMARY KEY,
		user_id UUID REFERENCES users(id),
		action VARCHAR(50) NOT NULL,
		resource_type VARCHAR(50) NOT NULL,
		resource_id UUID,
		details JSONB,
		ip_address VARCHAR(45),
		user_agent TEXT,
		status VARCHAR(20) DEFAULT 'success',
		error_message TEXT,
		created_at TIMESTAMPTZ NOT NULL
	);
	
	CREATE TABLE IF NOT EXISTS template_uploads (
		id UUID PRIMARY KEY,
		name VARCHAR(100) NOT NULL,
		description TEXT,
		file_name VARCHAR(255) NOT NULL,
		file_size BIGINT NOT NULL,
		format VARCHAR(20),
		architecture VARCHAR(20),
		upload_path VARCHAR(500),
		temp_path VARCHAR(500),
		status VARCHAR(20) DEFAULT 'uploading',
		progress BIGINT DEFAULT 0,
		error_message TEXT,
		uploaded_by UUID REFERENCES users(id),
		created_at TIMESTAMPTZ,
		completed_at TIMESTAMPTZ
	);
	
	CREATE TABLE IF NOT EXISTS alert_rules (
		id UUID PRIMARY KEY,
		name VARCHAR(100) NOT NULL,
		description TEXT,
		metric VARCHAR(50) NOT NULL,
		condition VARCHAR(10) NOT NULL,
		threshold DECIMAL(10,2) NOT NULL,
		duration BIGINT DEFAULT 5,
		severity VARCHAR(20) NOT NULL,
		enabled BOOLEAN DEFAULT true,
		notify_channels TEXT[],
		notify_users TEXT[],
		vm_ids TEXT[],
		is_global BOOLEAN DEFAULT false,
		created_by UUID REFERENCES users(id),
		created_at TIMESTAMPTZ,
		updated_at TIMESTAMPTZ
	);
	
	CREATE TABLE IF NOT EXISTS alert_histories (
		id UUID PRIMARY KEY,
		alert_rule_id UUID NOT NULL,
		vm_id UUID,
		severity VARCHAR(20) NOT NULL,
		metric VARCHAR(50) NOT NULL,
		current_value DECIMAL(10,2),
		threshold DECIMAL(10,2),
		condition VARCHAR(10),
		message TEXT,
		status VARCHAR(20) DEFAULT 'triggered',
		resolved_at TIMESTAMPTZ,
		notified_at TIMESTAMPTZ,
		created_at TIMESTAMPTZ NOT NULL
	);

	CREATE TABLE IF NOT EXISTS virtual_networks (
		id UUID PRIMARY KEY,
		name VARCHAR(100) NOT NULL UNIQUE,
		description TEXT,
		network_type VARCHAR(20) NOT NULL DEFAULT 'nat',
		bridge_name VARCHAR(100),
		subnet VARCHAR(20),
		gateway VARCHAR(20),
		dhcp_start VARCHAR(20),
		dhcp_end VARCHAR(20),
		dhcp_enabled BOOLEAN DEFAULT true,
		autostart BOOLEAN DEFAULT true,
		active BOOLEAN DEFAULT false,
		xml_def TEXT,
		created_by UUID REFERENCES users(id),
		created_at TIMESTAMPTZ,
		updated_at TIMESTAMPTZ
	);

	CREATE TABLE IF NOT EXISTS storage_pools (
		id UUID PRIMARY KEY,
		name VARCHAR(100) NOT NULL UNIQUE,
		description TEXT,
		pool_type VARCHAR(20) NOT NULL DEFAULT 'dir',
		target_path VARCHAR(500),
		source_path VARCHAR(500),
		capacity BIGINT DEFAULT 0,
		available BIGINT DEFAULT 0,
		used BIGINT DEFAULT 0,
		active BOOLEAN DEFAULT false,
		autostart BOOLEAN DEFAULT true,
		xml_def TEXT,
		created_by UUID REFERENCES users(id),
		created_at TIMESTAMPTZ,
		updated_at TIMESTAMPTZ
	);

	CREATE TABLE IF NOT EXISTS storage_volumes (
		id UUID PRIMARY KEY,
		pool_id UUID NOT NULL REFERENCES storage_pools(id),
		name VARCHAR(255) NOT NULL,
		volume_type VARCHAR(20),
		capacity BIGINT DEFAULT 0,
		allocation BIGINT DEFAULT 0,
		format VARCHAR(20),
		path VARCHAR(500),
		vm_id UUID REFERENCES virtual_machines(id),
		created_at TIMESTAMPTZ
	);

	CREATE INDEX IF NOT EXISTS idx_storage_volumes_pool ON storage_volumes(pool_id);
	CREATE INDEX IF NOT EXISTS idx_storage_volumes_vm ON storage_volumes(vm_id);

	CREATE TABLE IF NOT EXISTS vm_backups (
		id UUID PRIMARY KEY,
		vm_id UUID NOT NULL REFERENCES virtual_machines(id),
		name VARCHAR(255) NOT NULL,
		description TEXT,
		backup_type VARCHAR(20) NOT NULL DEFAULT 'full',
		status VARCHAR(20) NOT NULL DEFAULT 'pending',
		file_path VARCHAR(500),
		file_size BIGINT DEFAULT 0,
		progress INTEGER DEFAULT 0,
		scheduled_at TIMESTAMPTZ,
		started_at TIMESTAMPTZ,
		completed_at TIMESTAMPTZ,
		expires_at TIMESTAMPTZ,
		error_msg TEXT,
		created_by UUID REFERENCES users(id),
		created_at TIMESTAMPTZ,
		updated_at TIMESTAMPTZ
	);

	CREATE INDEX IF NOT EXISTS idx_vm_backups_vm ON vm_backups(vm_id);
	CREATE INDEX IF NOT EXISTS idx_vm_backups_status ON vm_backups(status);

	CREATE TABLE IF NOT EXISTS backup_schedules (
		id UUID PRIMARY KEY,
		vm_id UUID NOT NULL REFERENCES virtual_machines(id),
		name VARCHAR(255) NOT NULL,
		cron_expr VARCHAR(100) NOT NULL,
		backup_type VARCHAR(20) NOT NULL DEFAULT 'full',
		retention INTEGER DEFAULT 7,
		enabled BOOLEAN DEFAULT true,
		last_run_at TIMESTAMPTZ,
		next_run_at TIMESTAMPTZ,
		created_by UUID REFERENCES users(id),
		created_at TIMESTAMPTZ,
		updated_at TIMESTAMPTZ
	);

	CREATE INDEX IF NOT EXISTS idx_backup_schedules_vm ON backup_schedules(vm_id);
	CREATE INDEX IF NOT EXISTS idx_backup_schedules_enabled ON backup_schedules(enabled);

	CREATE TABLE IF NOT EXISTS vm_snapshots (
		id UUID PRIMARY KEY,
		vm_id UUID NOT NULL REFERENCES virtual_machines(id),
		name VARCHAR(255) NOT NULL,
		description TEXT,
		status VARCHAR(20) NOT NULL DEFAULT 'created',
		is_current BOOLEAN DEFAULT false,
		parent_id UUID REFERENCES vm_snapshots(id),
		created_by UUID REFERENCES users(id),
		created_at TIMESTAMPTZ
	);

	CREATE UNIQUE INDEX IF NOT EXISTS idx_vm_snapshots_vm_name ON vm_snapshots(vm_id, name);
	CREATE INDEX IF NOT EXISTS idx_vm_snapshots_vm ON vm_snapshots(vm_id);

	-- Migration: Add architecture column to existing virtual_machines table
	ALTER TABLE virtual_machines ADD COLUMN IF NOT EXISTS architecture VARCHAR(20) DEFAULT 'x86_64';

	-- Migration: Add installation columns
	ALTER TABLE virtual_machines ADD COLUMN IF NOT EXISTS is_installed BOOLEAN DEFAULT FALSE;
	ALTER TABLE virtual_machines ADD COLUMN IF NOT EXISTS install_status VARCHAR(50) DEFAULT '';
	ALTER TABLE virtual_machines ADD COLUMN IF NOT EXISTS install_progress INTEGER DEFAULT 0;
	ALTER TABLE virtual_machines ADD COLUMN IF NOT EXISTS agent_installed BOOLEAN DEFAULT FALSE;

	-- Migration: Add installation columns to templates
	ALTER TABLE vm_templates ADD COLUMN IF NOT EXISTS iso_path VARCHAR(500);
	ALTER TABLE vm_templates ADD COLUMN IF NOT EXISTS install_script TEXT;
	ALTER TABLE vm_templates ADD COLUMN IF NOT EXISTS post_install_script TEXT;

	-- Migration: Create ISO tables
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
		uploaded_by UUID REFERENCES users(id),
		created_at TIMESTAMPTZ,
		updated_at TIMESTAMPTZ
	);
	
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
		progress BIGINT DEFAULT 0,
		error_message TEXT,
		uploaded_by UUID REFERENCES users(id),
		created_at TIMESTAMPTZ,
		completed_at TIMESTAMPTZ
	);

	-- Migration: Add ISO reference to virtual_machines
	ALTER TABLE virtual_machines ADD COLUMN IF NOT EXISTS iso_id UUID;
	ALTER TABLE virtual_machines ADD COLUMN IF NOT EXISTS installation_mode VARCHAR(20) DEFAULT 'template';

	-- Create indexes for ISO tables
	CREATE INDEX IF NOT EXISTS idx_isos_status ON isos(status);
	CREATE INDEX IF NOT EXISTS idx_isos_architecture ON isos(architecture);
	CREATE INDEX IF NOT EXISTS idx_isos_os_type ON isos(os_type);
	CREATE INDEX IF NOT EXISTS idx_iso_uploads_status ON iso_uploads(status);

	-- Migration: Add resumable upload columns
	ALTER TABLE template_uploads ADD COLUMN IF NOT EXISTS uploaded_chunks TEXT DEFAULT '';
	ALTER TABLE template_uploads ADD COLUMN IF NOT EXISTS total_chunks INTEGER DEFAULT 0;
	ALTER TABLE template_uploads ADD COLUMN IF NOT EXISTS chunk_size BIGINT DEFAULT 0;
	ALTER TABLE iso_uploads ADD COLUMN IF NOT EXISTS uploaded_chunks TEXT DEFAULT '';
	ALTER TABLE iso_uploads ADD COLUMN IF NOT EXISTS total_chunks INTEGER DEFAULT 0;
	ALTER TABLE iso_uploads ADD COLUMN IF NOT EXISTS chunk_size BIGINT DEFAULT 0;

	-- Migration: Add checksum columns to templates
	ALTER TABLE vm_templates ADD COLUMN IF NOT EXISTS md5 VARCHAR(32);
	ALTER TABLE vm_templates ADD COLUMN IF NOT EXISTS sha256 VARCHAR(64);

	-- Migration: Fix IP address columns to use VARCHAR instead of INET
	ALTER TABLE virtual_machines DROP COLUMN IF EXISTS ip_address;
	ALTER TABLE virtual_machines DROP COLUMN IF EXISTS gateway;
	ALTER TABLE virtual_machines DROP COLUMN IF EXISTS dns_servers;
	ALTER TABLE virtual_machines ADD COLUMN ip_address VARCHAR(45);
	ALTER TABLE virtual_machines ADD COLUMN gateway VARCHAR(45);
	ALTER TABLE virtual_machines ADD COLUMN dns_servers VARCHAR(255)[];

	ALTER TABLE audit_logs DROP COLUMN IF EXISTS ip_address;
	ALTER TABLE audit_logs ADD COLUMN ip_address VARCHAR(45);

	-- Migration: Add ISO installation fields
	ALTER TABLE virtual_machines ADD COLUMN IF NOT EXISTS iso_id UUID;
	ALTER TABLE virtual_machines ADD COLUMN IF NOT EXISTS installation_mode VARCHAR(20) DEFAULT 'template';

	-- Migration: Add operation history tables
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
		logout_at TIMESTAMPTZ,
		session_duration INTEGER,
		created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
	);

	CREATE INDEX IF NOT EXISTS idx_login_histories_user_id ON login_histories(user_id);
	CREATE INDEX IF NOT EXISTS idx_login_histories_created_at ON login_histories(created_at);
	CREATE INDEX IF NOT EXISTS idx_login_histories_status ON login_histories(status);
	CREATE INDEX IF NOT EXISTS idx_login_histories_ip_address ON login_histories(ip_address);

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
		created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
	);

	CREATE INDEX IF NOT EXISTS idx_resource_change_histories_resource ON resource_change_histories(resource_type, resource_id);
	CREATE INDEX IF NOT EXISTS idx_resource_change_histories_changed_by ON resource_change_histories(changed_by);
	CREATE INDEX IF NOT EXISTS idx_resource_change_histories_created_at ON resource_change_histories(created_at);
	CREATE INDEX IF NOT EXISTS idx_resource_change_histories_action ON resource_change_histories(action);

	CREATE TABLE IF NOT EXISTS vm_operation_histories (
		id UUID PRIMARY KEY,
		vm_id UUID NOT NULL REFERENCES virtual_machines(id) ON DELETE CASCADE,
		operation VARCHAR(50) NOT NULL,
		status VARCHAR(20) NOT NULL DEFAULT 'pending',
		started_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
		completed_at TIMESTAMPTZ,
		duration INTEGER,
		triggered_by UUID REFERENCES users(id) ON DELETE SET NULL,
		ip_address VARCHAR(45),
		user_agent TEXT,
		request_params JSONB,
		response_data JSONB,
		error_message TEXT
	);

	CREATE INDEX IF NOT EXISTS idx_vm_operation_histories_vm_id ON vm_operation_histories(vm_id);
	CREATE INDEX IF NOT EXISTS idx_vm_operation_histories_triggered_by ON vm_operation_histories(triggered_by);
	CREATE INDEX IF NOT EXISTS idx_vm_operation_histories_status ON vm_operation_histories(status);
	CREATE INDEX IF NOT EXISTS idx_vm_operation_histories_started_at ON vm_operation_histories(started_at);
	`
	return db.Exec(sql).Error
}

func Seed(db *gorm.DB) error {
	adminPassword, _ := hashPassword("admin123")
	admin := models.User{
		Username:     "admin",
		Email:        "admin@vmmanager.local",
		PasswordHash: adminPassword,
		Role:         "admin",
		IsActive:     true,
		Language:     "zh-CN",
		Timezone:     "Asia/Shanghai",
		QuotaCPU:     64,
		QuotaMemory:  65536,
		QuotaDisk:    1000,
		QuotaVMCount: 50,
	}

	if err := db.Where("username = ?", "admin").FirstOrCreate(&admin).Error; err != nil {
		return fmt.Errorf("failed to seed admin user: %w", err)
	}

	userPassword, _ := hashPassword("user123")
	exampleUser := models.User{
		Username:     "example",
		Email:        "example@vmmanager.local",
		PasswordHash: userPassword,
		Role:         "user",
		IsActive:     true,
		Language:     "zh-CN",
		Timezone:     "Asia/Shanghai",
	}

	if err := db.Where("username = ?", "example").FirstOrCreate(&exampleUser).Error; err != nil {
		return fmt.Errorf("failed to seed example user: %w", err)
	}

	return nil
}
