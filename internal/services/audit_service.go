package services

import (
	"context"
	"encoding/json"
	"net"
	"strings"
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

func getRealIP(c *gin.Context) string {
	extractForwardedIP := func(forwarded string) string {
		if forwarded == "" {
			return ""
		}
		ips := strings.Split(forwarded, ",")
		for i := len(ips) - 1; i >= 0; i-- {
			ip := strings.TrimSpace(ips[i])
			if ip != "" && ip != "unknown" && !strings.HasPrefix(ip, "10.") && !strings.HasPrefix(ip, "172.") && !strings.HasPrefix(ip, "192.168.") && ip != "127.0.0.1" && ip != "::1" {
				return ip
			}
		}
		for _, item := range ips {
			ip := strings.TrimSpace(item)
			if ip != "" && ip != "unknown" {
				return ip
			}
		}
		return ""
	}

	ipAddress := c.GetHeader("X-Real-IP")
	if ipAddress == "" || ipAddress == "127.0.0.1" || ipAddress == "::1" {
		ipAddress = extractForwardedIP(c.GetHeader("X-Forwarded-For"))
	}
	if ipAddress == "" || ipAddress == "127.0.0.1" || ipAddress == "::1" {
		ipAddress = extractForwardedIP(c.GetHeader("X-Original-Forwarded-For"))
	}
	if ipAddress == "" || ipAddress == "127.0.0.1" || ipAddress == "::1" {
		ipAddress = c.ClientIP()
	}
	if ipAddress == "" || ipAddress == "127.0.0.1" || ipAddress == "::1" {
		if host, _, err := net.SplitHostPort(c.Request.RemoteAddr); err == nil {
			ipAddress = host
		} else {
			ipAddress = c.Request.RemoteAddr
		}
	}
	if strings.Contains(ipAddress, ":") && !strings.HasPrefix(ipAddress, "[") {
		if host, _, err := net.SplitHostPort(ipAddress); err == nil {
			ipAddress = host
		}
	}
	if ipAddress == "" {
		ipAddress = "unknown"
	}
	return ipAddress
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

	ip := getRealIP(c)

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
