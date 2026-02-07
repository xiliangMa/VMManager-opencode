package database

import (
	"fmt"

	"vmmanager/config"
	"vmmanager/internal/models"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"gorm.io/gorm/schema"
)

func hashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(bytes), err
}

func NewPostgreSQL(cfg config.DatabaseConfig) (*gorm.DB, error) {
	dsn := cfg.DSN()

	gormConfig := &gorm.Config{
		NamingStrategy: schema.NamingStrategy{
			TablePrefix:   "",
			SingularTable: false,
		},
		Logger: logger.Default.LogMode(logger.Info),
	}

	db, err := gorm.Open(postgres.Open(dsn), gormConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get underlying sql.DB: %w", err)
	}

	sqlDB.SetMaxOpenConns(cfg.MaxOpenConns)
	sqlDB.SetMaxIdleConns(cfg.MaxIdleConns)
	sqlDB.SetConnMaxLifetime(cfg.ConnMaxLifetime)

	if err := sqlDB.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
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
	)
}

func Seed(db *gorm.DB) error {
	// 创建管理员用户
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

	// 创建示例用户
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
