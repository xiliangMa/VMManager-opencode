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
		status VARCHAR(20) DEFAULT 'pending',
		architecture VARCHAR(20) DEFAULT 'x86_64',
		vnc_port BIGINT,
		vnc_password VARCHAR(20),
		spice_port BIGINT,
		mac_address VARCHAR(17) UNIQUE,
		ip_address INET,
		gateway INET,
		dns_servers INET[],
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
		ip_address INET,
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
