package services

import (
	"context"
	"encoding/json"
	"net"
	"time"

	"vmmanager/internal/models"
	"vmmanager/internal/repository"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type AuditService struct {
	auditRepo *repository.AuditLogRepository
}

func NewAuditService(auditRepo *repository.AuditLogRepository) *AuditService {
	return &AuditService{auditRepo: auditRepo}
}

type AuditLogInput struct {
	Action       string
	ResourceType string
	ResourceID   *uuid.UUID
	Details      map[string]interface{}
	Status       string
	ErrorMessage string
}

func (s *AuditService) Log(c *gin.Context, input AuditLogInput) {
	var userID *uuid.UUID
	if id, exists := c.Get("user_id"); exists {
		if uid, err := uuid.Parse(id.(string)); err == nil {
			userID = &uid
		}
	}

	var detailsJSON string
	if input.Details != nil {
		if bytes, err := json.Marshal(input.Details); err == nil {
			detailsJSON = string(bytes)
		}
	}

	var ip net.IP
	if c.ClientIP() != "" {
		ip = net.ParseIP(c.ClientIP())
	}

	userAgent := c.GetHeader("User-Agent")
	if len(userAgent) > 500 {
		userAgent = userAgent[:500]
	}

	if input.Status == "" {
		input.Status = "success"
	}

	log := &models.AuditLog{
		ID:           uuid.New(),
		UserID:       userID,
		Action:       input.Action,
		ResourceType: input.ResourceType,
		ResourceID:   input.ResourceID,
		Details:      detailsJSON,
		IPAddress:    ip,
		UserAgent:    userAgent,
		Status:       input.Status,
		ErrorMessage: input.ErrorMessage,
		CreatedAt:    time.Now(),
	}

	go func() {
		_ = s.auditRepo.Create(context.Background(), log)
	}()
}

func (s *AuditService) LogSuccess(c *gin.Context, action, resourceType string, resourceID *uuid.UUID, details map[string]interface{}) {
	s.Log(c, AuditLogInput{
		Action:       action,
		ResourceType: resourceType,
		ResourceID:   resourceID,
		Details:      details,
		Status:       "success",
	})
}

func (s *AuditService) LogError(c *gin.Context, action, resourceType string, resourceID *uuid.UUID, errMsg string) {
	s.Log(c, AuditLogInput{
		Action:       action,
		ResourceType: resourceType,
		ResourceID:   resourceID,
		Status:       "failed",
		ErrorMessage: errMsg,
	})
}
