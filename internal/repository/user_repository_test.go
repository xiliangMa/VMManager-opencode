package repository_test

import (
	"context"
	"testing"

	"vmmanager/internal/models"
	"vmmanager/internal/repository"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/schema"
)

func setupTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		NamingStrategy: schema.NamingStrategy{
			TablePrefix:   "",
			SingularTable: true,
		},
	})
	if err != nil {
		t.Fatalf("failed to connect to test database: %v", err)
	}

	err = db.AutoMigrate(&models.User{})
	if err != nil {
		t.Fatalf("failed to migrate test database: %v", err)
	}

	return db
}

func TestUserRepository_Create(t *testing.T) {
	db := setupTestDB(t)
	userRepo := repository.NewUserRepository(db)

	user := &models.User{
		Username:     "testuser",
		Email:        "test@example.com",
		PasswordHash: "hash",
		Role:         "user",
		IsActive:     true,
	}

	err := userRepo.Create(context.Background(), user)
	if err != nil {
		t.Fatalf("failed to create user: %v", err)
	}

	if user.ID.String() == "" {
		t.Error("expected user ID to be set")
	}
}

func TestUserRepository_FindByUsername(t *testing.T) {
	db := setupTestDB(t)
	userRepo := repository.NewUserRepository(db)

	user := &models.User{
		Username:     "findme",
		Email:        "findme@example.com",
		PasswordHash: "hash",
		Role:         "user",
		IsActive:     true,
	}
	userRepo.Create(context.Background(), user)

	found, err := userRepo.FindByUsername(context.Background(), "findme")
	if err != nil {
		t.Fatalf("failed to find user: %v", err)
	}

	if found.Username != "findme" {
		t.Errorf("expected username 'findme', got '%s'", found.Username)
	}
}

func TestUserRepository_FindByUsername_NotFound(t *testing.T) {
	db := setupTestDB(t)
	userRepo := repository.NewUserRepository(db)

	_, err := userRepo.FindByUsername(context.Background(), "nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent user")
	}
}

func TestUserRepository_FindByEmail(t *testing.T) {
	db := setupTestDB(t)
	userRepo := repository.NewUserRepository(db)

	user := &models.User{
		Username:     "emailuser",
		Email:        "emailtest@example.com",
		PasswordHash: "hash",
		Role:         "user",
		IsActive:     true,
	}
	userRepo.Create(context.Background(), user)

	found, err := userRepo.FindByEmail(context.Background(), "emailtest@example.com")
	if err != nil {
		t.Fatalf("failed to find user by email: %v", err)
	}

	if found.Email != "emailtest@example.com" {
		t.Errorf("expected email 'emailtest@example.com', got '%s'", found.Email)
	}
}

func TestUserRepository_FindByID(t *testing.T) {
	db := setupTestDB(t)
	userRepo := repository.NewUserRepository(db)

	user := &models.User{
		Username:     "iduser",
		Email:        "idtest@example.com",
		PasswordHash: "hash",
		Role:         "user",
		IsActive:     true,
	}
	userRepo.Create(context.Background(), user)

	found, err := userRepo.FindByID(context.Background(), user.ID.String())
	if err != nil {
		t.Fatalf("failed to find user by ID: %v", err)
	}

	if found.ID != user.ID {
		t.Errorf("expected ID %s, got %s", user.ID, found.ID)
	}
}

func TestUserRepository_Update(t *testing.T) {
	db := setupTestDB(t)
	userRepo := repository.NewUserRepository(db)

	user := &models.User{
		Username:     "updateuser",
		Email:        "update@example.com",
		PasswordHash: "hash",
		Role:         "user",
		IsActive:     true,
	}
	userRepo.Create(context.Background(), user)

	user.Email = "updated@example.com"
	err := userRepo.Update(context.Background(), user)
	if err != nil {
		t.Fatalf("failed to update user: %v", err)
	}

	found, _ := userRepo.FindByID(context.Background(), user.ID.String())
	if found.Email != "updated@example.com" {
		t.Errorf("expected updated email, got '%s'", found.Email)
	}
}

func TestUserRepository_Delete(t *testing.T) {
	db := setupTestDB(t)
	userRepo := repository.NewUserRepository(db)

	user := &models.User{
		Username:     "deleteuser",
		Email:        "delete@example.com",
		PasswordHash: "hash",
		Role:         "user",
		IsActive:     true,
	}
	userRepo.Create(context.Background(), user)

	err := userRepo.Delete(context.Background(), user.ID.String())
	if err != nil {
		t.Fatalf("failed to delete user: %v", err)
	}

	_, err = userRepo.FindByID(context.Background(), user.ID.String())
	if err == nil {
		t.Error("expected error after deleting user")
	}
}

func TestUserRepository_List(t *testing.T) {
	db := setupTestDB(t)
	userRepo := repository.NewUserRepository(db)

	for i := 0; i < 5; i++ {
		user := &models.User{
			Username:     "listuser" + string(rune('0'+i)),
			Email:        "list" + string(rune('0'+i)) + "@example.com",
			PasswordHash: "hash",
			Role:         "user",
			IsActive:     true,
		}
		userRepo.Create(context.Background(), user)
	}

	users, total, err := userRepo.List(context.Background(), 0, 10)
	if err != nil {
		t.Fatalf("failed to list users: %v", err)
	}

	if len(users) != 5 {
		t.Errorf("expected 5 users, got %d", len(users))
	}

	if total != 5 {
		t.Errorf("expected total 5, got %d", total)
	}
}

