package handlers

import (
	"net/http"
	"time"

	"vmmanager/config"
	"vmmanager/internal/models"
	"vmmanager/internal/repository"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

type AuthHandler struct {
	userRepo *repository.UserRepository
	jwtCfg   config.JWTConfig
}

func NewAuthHandler(userRepo *repository.UserRepository, jwtCfg config.JWTConfig) *AuthHandler {
	return &AuthHandler{
		userRepo: userRepo,
		jwtCfg:   jwtCfg,
	}
}

type TokenClaims struct {
	UserID string `json:"user_id"`
	Role   string `json:"role"`
	jwt.RegisteredClaims
}

func hashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(bytes), err
}

func verifyPassword(hash, password string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

func generateToken(userID, role string, cfg config.JWTConfig) (string, error) {
	claims := TokenClaims{
		UserID: userID,
		Role:   role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(cfg.Expiration)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(cfg.Secret))
}

func generateRefreshToken(userID string, cfg config.JWTConfig) (string, error) {
	claims := jwt.RegisteredClaims{
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(cfg.RefreshExpiration)),
		IssuedAt:  jwt.NewNumericDate(time.Now()),
		Subject:   userID,
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(cfg.Secret))
}

func (h *AuthHandler) Register(c *gin.Context) {
	var req struct {
		Username string `json:"username" binding:"required,min=3,max=50"`
		Email    string `json:"email" binding:"required,email"`
		Password string `json:"password" binding:"required,min=6"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 4001, "message": err.Error()})
		return
	}

	ctx := c.Request.Context()

	existingUser, _ := h.userRepo.FindByUsername(ctx, req.Username)
	if existingUser != nil {
		c.JSON(http.StatusConflict, gin.H{"code": 4009, "message": "username already exists"})
		return
	}

	existingEmail, _ := h.userRepo.FindByEmail(ctx, req.Email)
	if existingEmail != nil {
		c.JSON(http.StatusConflict, gin.H{"code": 4009, "message": "email already exists"})
		return
	}

	passwordHash, _ := hashPassword(req.Password)

	user := &models.User{
		Username:     req.Username,
		Email:        req.Email,
		PasswordHash: passwordHash,
		Role:         "user",
		IsActive:     true,
		Language:     "zh-CN",
		Timezone:     "Asia/Shanghai",
	}

	if err := h.userRepo.Create(ctx, user); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 5001, "message": "failed to create user"})
		return
	}

	token, _ := generateToken(user.ID.String(), user.Role, h.jwtCfg)
	refreshToken, _ := generateRefreshToken(user.ID.String(), h.jwtCfg)

	c.JSON(http.StatusCreated, gin.H{
		"code":    0,
		"message": "success",
		"data": gin.H{
			"user": gin.H{
				"id":       user.ID,
				"username": user.Username,
				"email":    user.Email,
				"role":     user.Role,
			},
			"token":         token,
			"refresh_token": refreshToken,
			"expires_in":    int(h.jwtCfg.Expiration.Seconds()),
		},
	})
}

func (h *AuthHandler) Login(c *gin.Context) {
	var req struct {
		Username string `json:"username" binding:"required"`
		Password string `json:"password" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 4001, "message": err.Error()})
		return
	}

	ctx := c.Request.Context()

	user, err := h.userRepo.FindByUsername(ctx, req.Username)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"code": 4002, "message": "invalid credentials"})
		return
	}

	if !user.IsActive {
		c.JSON(http.StatusUnauthorized, gin.H{"code": 4003, "message": "account is disabled"})
		return
	}

	if !verifyPassword(user.PasswordHash, req.Password) {
		c.JSON(http.StatusUnauthorized, gin.H{"code": 4002, "message": "invalid credentials"})
		return
	}

	h.userRepo.UpdateLastLogin(ctx, user.ID.String())

	token, _ := generateToken(user.ID.String(), user.Role, h.jwtCfg)
	refreshToken, _ := generateRefreshToken(user.ID.String(), h.jwtCfg)

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data": gin.H{
			"user": gin.H{
				"id":       user.ID,
				"username": user.Username,
				"email":    user.Email,
				"role":     user.Role,
				"avatar":   user.AvatarURL,
			},
			"token":         token,
			"refresh_token": refreshToken,
			"expires_in":    int(h.jwtCfg.Expiration.Seconds()),
		},
	})
}

func (h *AuthHandler) Logout(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
	})
}

func (h *AuthHandler) GetProfile(c *gin.Context) {
	userID, _ := c.Get("user_id")
	ctx := c.Request.Context()

	user, err := h.userRepo.FindByID(ctx, userID.(string))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 4004, "message": "user not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data": gin.H{
			"id":       user.ID,
			"username": user.Username,
			"email":    user.Email,
			"role":     user.Role,
			"avatar":   user.AvatarURL,
			"language": user.Language,
			"timezone": user.Timezone,
			"quota": gin.H{
				"cpu":      user.QuotaCPU,
				"memory":   user.QuotaMemory,
				"disk":     user.QuotaDisk,
				"vm_count": user.QuotaVMCount,
			},
			"last_login_at": user.LastLoginAt,
			"created_at":    user.CreatedAt,
		},
	})
}

func (h *AuthHandler) UpdateProfile(c *gin.Context) {
	userID, _ := c.Get("user_id")
	ctx := c.Request.Context()

	user, err := h.userRepo.FindByID(ctx, userID.(string))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 4004, "message": "user not found"})
		return
	}

	var req struct {
		Email    string `json:"email"`
		Avatar   string `json:"avatar"`
		Language string `json:"language"`
		Timezone string `json:"timezone"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 4001, "message": err.Error()})
		return
	}

	if req.Email != "" {
		user.Email = req.Email
	}
	if req.Avatar != "" {
		user.AvatarURL = req.Avatar
	}
	if req.Language != "" {
		user.Language = req.Language
	}
	if req.Timezone != "" {
		user.Timezone = req.Timezone
	}

	if err := h.userRepo.Update(ctx, user); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 5001, "message": "failed to update profile"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data": gin.H{
			"id":       user.ID,
			"username": user.Username,
			"email":    user.Email,
			"avatar":   user.AvatarURL,
			"language": user.Language,
			"timezone": user.Timezone,
		},
	})
}

func (h *AuthHandler) RefreshToken(c *gin.Context) {
	var req struct {
		RefreshToken string `json:"refresh_token" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 4001, "message": err.Error()})
		return
	}

	token, err := jwt.ParseWithClaims(req.RefreshToken, &jwt.RegisteredClaims{}, func(token *jwt.Token) (interface{}, error) {
		return []byte(h.jwtCfg.Secret), nil
	})

	if err != nil || !token.Valid {
		c.JSON(http.StatusUnauthorized, gin.H{"code": 4002, "message": "invalid refresh token"})
		return
	}

	claims := token.Claims.(*jwt.RegisteredClaims)
	userID := claims.Subject

	ctx := c.Request.Context()
	user, err := h.userRepo.FindByID(ctx, userID)
	if err != nil || !user.IsActive {
		c.JSON(http.StatusUnauthorized, gin.H{"code": 4003, "message": "user not found or disabled"})
		return
	}

	newToken, _ := generateToken(user.ID.String(), user.Role, h.jwtCfg)
	newRefreshToken, _ := generateRefreshToken(user.ID.String(), h.jwtCfg)

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data": gin.H{
			"token":         newToken,
			"refresh_token": newRefreshToken,
			"expires_in":    int(h.jwtCfg.Expiration.Seconds()),
		},
	})
}
