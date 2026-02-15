package handlers_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"vmmanager/config"
	"vmmanager/internal/api/handlers"
	"vmmanager/internal/middleware"
	"vmmanager/internal/models"
	"vmmanager/internal/repository"

	"github.com/gin-gonic/gin"
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

	err = db.AutoMigrate(
		&models.User{},
		&models.VirtualMachine{},
	)
	if err != nil {
		t.Fatalf("failed to migrate test database: %v", err)
	}

	return db
}

func setupRouter(db *gorm.DB) (*gin.Engine, *repository.Repositories) {
	gin.SetMode(gin.TestMode)

	repos := repository.NewRepositories(db)

	router := gin.New()
	router.Use(gin.Recovery())

	jwtMiddleware := middleware.JWTRequired("test-secret")

	jwtCfg := config.JWTConfig{
		Secret:            "test-secret",
		Expiration:        24 * time.Hour,
		RefreshExpiration: 168 * time.Hour,
	}

	authHandler := handlers.NewAuthHandler(repos.User, jwtCfg)

	router.POST("/auth/register", authHandler.Register)
	router.POST("/auth/login", authHandler.Login)
	router.GET("/auth/profile", jwtMiddleware, authHandler.GetProfile)

	return router, repos
}

func TestRegister_Success(t *testing.T) {
	db := setupTestDB(t)
	router, _ := setupRouter(db)

	body := `{"username":"testuser","email":"test@example.com","password":"password123"}`
	req, _ := http.NewRequest("POST", "/auth/register", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("expected status %d, got %d", http.StatusCreated, w.Code)
	}

	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)

	if response["code"] != float64(0) {
		t.Errorf("expected code 0, got %v", response["code"])
	}

	if response["data"] == nil {
		t.Error("expected data in response")
	}
}

func TestRegister_ValidationError(t *testing.T) {
	db := setupTestDB(t)
	router, _ := setupRouter(db)

	body := `{"username":"","email":"invalid","password":"123"}`
	req, _ := http.NewRequest("POST", "/auth/register", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestRegister_UsernameExists(t *testing.T) {
	db := setupTestDB(t)
	router, _ := setupRouter(db)

	body := `{"username":"testuser","email":"test@example.com","password":"password123"}`
	req, _ := http.NewRequest("POST", "/auth/register", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	req2, _ := http.NewRequest("POST", "/auth/register", bytes.NewBufferString(body))
	req2.Header.Set("Content-Type", "application/json")

	w2 := httptest.NewRecorder()
	router.ServeHTTP(w2, req2)

	if w2.Code != http.StatusConflict {
		t.Errorf("expected status %d, got %d", http.StatusConflict, w2.Code)
	}
}

func TestLogin_Success(t *testing.T) {
	db := setupTestDB(t)
	router, _ := setupRouter(db)

	registerBody := `{"username":"loginuser","email":"login@example.com","password":"password123"}`
	req, _ := http.NewRequest("POST", "/auth/register", bytes.NewBufferString(registerBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	loginBody := `{"username":"loginuser","password":"password123"}`
	req, _ = http.NewRequest("POST", "/auth/login", bytes.NewBufferString(loginBody))
	req.Header.Set("Content-Type", "application/json")

	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)

	if response["code"] != float64(0) {
		t.Errorf("expected code 0, got %v", response["code"])
	}

	data := response["data"].(map[string]interface{})
	if data["token"] == nil {
		t.Error("expected token in response")
	}
}

func TestLogin_InvalidCredentials(t *testing.T) {
	db := setupTestDB(t)
	router, _ := setupRouter(db)

	body := `{"username":"nonexistent","password":"wrongpassword"}`
	req, _ := http.NewRequest("POST", "/auth/login", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected status %d, got %d", http.StatusUnauthorized, w.Code)
	}
}

func TestProfile_WithoutAuth(t *testing.T) {
	db := setupTestDB(t)
	router, _ := setupRouter(db)

	req, _ := http.NewRequest("GET", "/auth/profile", nil)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected status %d, got %d", http.StatusUnauthorized, w.Code)
	}
}
