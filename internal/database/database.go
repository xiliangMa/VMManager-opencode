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

	gormConfig := &gorm.Config{
		NamingStrategy: schema.NamingStrategy{
			TablePrefix:   "",
			SingularTable: true,
		},
		Logger: logger.Default.LogMode(logger.Warn),
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

	gormConfig := &gorm.Config{
		NamingStrategy: schema.NamingStrategy{
			TablePrefix:   "",
			SingularTable: false,
		},
		Logger: logger.Default.LogMode(logger.Info),
	}

	db, err := gorm.Open(postgres.Open(dsn), gormConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to PostgreSQL: %w", err)
	}

	return db, nil
}

func Migrate(db *gorm.DB) error {
	return db.AutoMigrate(
		&models.User{},
		&models.VMTemplate{},
		&models.VirtualMachine{},
		&models.VMStats{},
		&models.AuditLog{},
		&models.TemplateUpload{},
		&models.AlertRule{},
	)
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
