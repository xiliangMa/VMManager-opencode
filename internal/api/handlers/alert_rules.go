package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"

	"vmmanager/internal/models"
	"vmmanager/internal/repository"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type AlertRuleHandler struct {
	repo *repository.AlertRuleRepository
}

func NewAlertRuleHandler(repo *repository.AlertRuleRepository) *AlertRuleHandler {
	return &AlertRuleHandler{repo: repo}
}

type CreateAlertRuleRequest struct {
	Name           string   `json:"name" binding:"required"`
	Description    string   `json:"description"`
	Metric         string   `json:"metric" binding:"required"`
	Condition      string   `json:"condition" binding:"required"`
	Threshold      float64  `json:"threshold" binding:"required"`
	Duration       int      `json:"duration"`
	Severity       string   `json:"severity" binding:"required"`
	NotifyChannels []string `json:"notifyChannels"`
	NotifyUsers    []string `json:"notifyUsers"`
	VMIDs          []string `json:"vmIds"`
	IsGlobal       bool     `json:"isGlobal"`
}

type UpdateAlertRuleRequest struct {
	Name           string   `json:"name"`
	Description    string   `json:"description"`
	Metric         string   `json:"metric"`
	Condition      string   `json:"condition"`
	Threshold      float64  `json:"threshold"`
	Duration       int      `json:"duration"`
	Severity       string   `json:"severity"`
	Enabled        *bool    `json:"enabled"`
	NotifyChannels []string `json:"notifyChannels"`
	NotifyUsers    []string `json:"notifyUsers"`
	VMIDs          []string `json:"vmIds"`
	IsGlobal       bool     `json:"isGlobal"`
}

func stoi(s string) (int, error) {
	n, err := strconv.Atoi(s)
	if err != nil {
		return 0, err
	}
	return n, nil
}

func arrayToJSON(arr []string) string {
	if arr == nil {
		return "[]"
	}
	data, _ := json.Marshal(arr)
	return string(data)
}

func arrayToPGArray(arr []string) string {
	if arr == nil || len(arr) == 0 {
		return "{}"
	}
	result := "{"
	for i, s := range arr {
		if i > 0 {
			result += ","
		}
		result += "\"" + s + "\""
	}
	result += "}"
	return result
}

func (h *AlertRuleHandler) ListAlertRules(c *gin.Context) {
	page := 1
	pageSize := 20

	var err error
	if p := c.Query("page"); p != "" {
		page, err = stoi(p)
		if err != nil {
			page = 1
		}
	}
	if ps := c.Query("page_size"); ps != "" {
		pageSize, err = stoi(ps)
		if err != nil {
			pageSize = 20
		}
	}

	offset := (page - 1) * pageSize
	rules, total, err := h.repo.List(c.Request.Context(), offset, pageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    50001,
			"message": "Failed to list alert rules",
			"data":    nil,
		})
		return
	}

	totalPages := int(total) / pageSize
	if int(total)%pageSize > 0 {
		totalPages++
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data":    rules,
		"meta": gin.H{
			"page":        page,
			"per_page":    pageSize,
			"total":       total,
			"total_pages": totalPages,
		},
	})
}

func (h *AlertRuleHandler) GetAlertRule(c *gin.Context) {
	id := c.Param("id")
	if _, err := uuid.Parse(id); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    40001,
			"message": "Invalid alert rule ID",
			"data":    nil,
		})
		return
	}

	rule, err := h.repo.FindByID(c.Request.Context(), id)
	if err != nil {
		if err == repository.ErrAlertRuleNotFound {
			c.JSON(http.StatusNotFound, gin.H{
				"code":    40401,
				"message": "Alert rule not found",
				"data":    nil,
			})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    50001,
			"message": "Failed to get alert rule",
			"data":    nil,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data":    rule,
	})
}

func (h *AlertRuleHandler) CreateAlertRule(c *gin.Context) {
	var req CreateAlertRuleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    40001,
			"message": err.Error(),
			"data":    nil,
		})
		return
	}

	rule := &models.AlertRule{
		Name:           req.Name,
		Description:    req.Description,
		Metric:         req.Metric,
		Condition:      req.Condition,
		Threshold:      req.Threshold,
		Duration:       req.Duration,
		Severity:       req.Severity,
		NotifyChannels: arrayToJSON(req.NotifyChannels),
		NotifyUsers:    arrayToJSON(req.NotifyUsers),
		VMIDs:          arrayToJSON(req.VMIDs),
		IsGlobal:       req.IsGlobal,
		Enabled:        true,
	}

	if err := h.repo.Create(c.Request.Context(), rule); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    50001,
			"message": "Failed to create alert rule",
			"details": err.Error(),
			"data":    nil,
		})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"code":    0,
		"message": "success",
		"data":    rule,
	})
}

func (h *AlertRuleHandler) UpdateAlertRule(c *gin.Context) {
	id := c.Param("id")
	if _, err := uuid.Parse(id); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    40001,
			"message": "Invalid alert rule ID",
			"data":    nil,
		})
		return
	}

	rule, err := h.repo.FindByID(c.Request.Context(), id)
	if err != nil {
		if err == repository.ErrAlertRuleNotFound {
			c.JSON(http.StatusNotFound, gin.H{
				"code":    40401,
				"message": "Alert rule not found",
				"data":    nil,
			})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    50001,
			"message": "Failed to get alert rule",
			"data":    nil,
		})
		return
	}

	var req UpdateAlertRuleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    40001,
			"message": err.Error(),
			"data":    nil,
		})
		return
	}

	if req.Name != "" {
		rule.Name = req.Name
	}
	if req.Description != "" {
		rule.Description = req.Description
	}
	if req.Metric != "" {
		rule.Metric = req.Metric
	}
	if req.Condition != "" {
		rule.Condition = req.Condition
	}
	if req.Threshold > 0 {
		rule.Threshold = req.Threshold
	}
	if req.Duration > 0 {
		rule.Duration = req.Duration
	}
	if req.Severity != "" {
		rule.Severity = req.Severity
	}
	if req.Enabled != nil {
		rule.Enabled = *req.Enabled
	}
	rule.NotifyChannels = arrayToJSON(req.NotifyChannels)
	rule.NotifyUsers = arrayToJSON(req.NotifyUsers)
	rule.VMIDs = arrayToJSON(req.VMIDs)
	rule.IsGlobal = req.IsGlobal

	if err := h.repo.Update(c.Request.Context(), rule); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    50001,
			"message": "Failed to update alert rule",
			"data":    nil,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data":    rule,
	})
}

func (h *AlertRuleHandler) DeleteAlertRule(c *gin.Context) {
	id := c.Param("id")
	if _, err := uuid.Parse(id); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    40001,
			"message": "Invalid alert rule ID",
			"data":    nil,
		})
		return
	}

	if err := h.repo.Delete(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    50001,
			"message": "Failed to delete alert rule",
			"data":    nil,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data":    nil,
	})
}

func (h *AlertRuleHandler) ToggleAlertRule(c *gin.Context) {
	id := c.Param("id")
	if _, err := uuid.Parse(id); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    40001,
			"message": "Invalid alert rule ID",
			"data":    nil,
		})
		return
	}

	rule, err := h.repo.FindByID(c.Request.Context(), id)
	if err != nil {
		if err == repository.ErrAlertRuleNotFound {
			c.JSON(http.StatusNotFound, gin.H{
				"code":    40401,
				"message": "Alert rule not found",
				"data":    nil,
			})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    50001,
			"message": "Failed to get alert rule",
			"data":    nil,
		})
		return
	}

	rule.Enabled = !rule.Enabled
	if err := h.repo.Update(c.Request.Context(), rule); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    50001,
			"message": "Failed to toggle alert rule",
			"data":    nil,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data":    rule,
	})
}

func (h *AlertRuleHandler) GetAlertStats(c *gin.Context) {
	criticalCount, _ := h.repo.CountBySeverity(c.Request.Context(), "critical")
	warningCount, _ := h.repo.CountBySeverity(c.Request.Context(), "warning")
	infoCount, _ := h.repo.CountBySeverity(c.Request.Context(), "info")

	rules, _, _ := h.repo.List(c.Request.Context(), 0, 0)
	enabledCount := 0
	for _, r := range rules {
		if r.Enabled {
			enabledCount++
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data": gin.H{
			"total_rules":    len(rules),
			"enabled_rules":  enabledCount,
			"critical_rules": criticalCount,
			"warning_rules":  warningCount,
			"info_rules":     infoCount,
		},
	})
}
