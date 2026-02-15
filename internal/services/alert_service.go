package services

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"vmmanager/internal/models"
	"vmmanager/internal/repository"

	"github.com/google/uuid"
)

type AlertService struct {
	ruleRepo     *repository.AlertRuleRepository
	historyRepo  *repository.AlertHistoryRepository
	vmRepo       *repository.VMRepository
	statsRepo    *repository.VMStatsRepository
	notifyChan   chan models.AlertNotification
	stopChan     chan struct{}
	runningAlert map[string]time.Time
	mu           sync.RWMutex
}

func NewAlertService(
	ruleRepo *repository.AlertRuleRepository,
	historyRepo *repository.AlertHistoryRepository,
	vmRepo *repository.VMRepository,
	statsRepo *repository.VMStatsRepository,
) *AlertService {
	return &AlertService{
		ruleRepo:     ruleRepo,
		historyRepo:  historyRepo,
		vmRepo:       vmRepo,
		statsRepo:    statsRepo,
		notifyChan:   make(chan models.AlertNotification, 100),
		stopChan:     make(chan struct{}),
		runningAlert: make(map[string]time.Time),
	}
}

func (s *AlertService) Start() {
	ticker := time.NewTicker(1 * time.Minute)

	go func() {
		for {
			select {
			case <-ticker.C:
				s.checkAlerts()
			case notification := <-s.notifyChan:
				go s.sendNotification(notification)
			case <-s.stopChan:
				ticker.Stop()
				return
			}
		}
	}()

	log.Println("[ALERT] Alert monitoring service started")
}

func (s *AlertService) Stop() {
	close(s.stopChan)
	log.Println("[ALERT] Alert monitoring service stopped")
}

func (s *AlertService) checkAlerts() {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	rules, err := s.ruleRepo.ListEnabled(ctx)
	if err != nil {
		log.Printf("[ALERT] Failed to get enabled rules: %v", err)
		return
	}

	for _, rule := range rules {
		if rule.IsGlobal {
			s.checkGlobalRule(ctx, rule)
		} else {
			s.checkVMRule(ctx, rule)
		}
	}
}

func (s *AlertService) checkGlobalRule(ctx context.Context, rule models.AlertRule) {
	vms, _, err := s.vmRepo.List(ctx, 0, 0)
	if err != nil {
		log.Printf("[ALERT] Failed to list VMs for global rule %s: %v", rule.Name, err)
		return
	}

	for _, vm := range vms {
		s.checkVMWithRule(ctx, rule, vm.ID.String(), vm.Name)
	}
}

func (s *AlertService) checkVMRule(ctx context.Context, rule models.AlertRule) {
	var vmIDs []string
	if rule.VMIDs != "" && rule.VMIDs != "{}" {
		if err := json.Unmarshal([]byte(rule.VMIDs), &vmIDs); err != nil {
			var rawIDs []string
			if err := json.Unmarshal([]byte(rule.VMIDs), &rawIDs); err == nil {
				vmIDs = rawIDs
			}
		}
	}

	for _, vmID := range vmIDs {
		vm, err := s.vmRepo.FindByID(ctx, vmID)
		if err != nil {
			continue
		}
		s.checkVMWithRule(ctx, rule, vmID, vm.Name)
	}
}

func (s *AlertService) checkVMWithRule(ctx context.Context, rule models.AlertRule, vmID, vmName string) {
	metricValue, err := s.getMetricValue(ctx, vmID, rule.Metric)
	if err != nil {
		return
	}

	alertKey := fmt.Sprintf("%s-%s", rule.ID.String(), vmID)

	if s.shouldTriggerAlert(rule, metricValue) {
		s.mu.RLock()
		firstTriggerTime, exists := s.runningAlert[alertKey]
		s.mu.RUnlock()

		if !exists {
			s.mu.Lock()
			s.runningAlert[alertKey] = time.Now()
			s.mu.Unlock()
			return
		}

		if time.Since(firstTriggerTime) >= time.Duration(rule.Duration)*time.Minute {
			s.triggerAlert(ctx, rule, vmID, vmName, metricValue)
			s.mu.Lock()
			delete(s.runningAlert, alertKey)
			s.mu.Unlock()
		}
	} else {
		s.mu.Lock()
		delete(s.runningAlert, alertKey)
		s.mu.Unlock()
	}
}

func (s *AlertService) getMetricValue(ctx context.Context, vmID, metric string) (float64, error) {
	stats, err := s.statsRepo.GetLatestByVMID(ctx, vmID)
	if err != nil {
		return 0, err
	}

	switch metric {
	case "cpu_usage":
		return stats.CPUUsage, nil
	case "memory_usage":
		if stats.MemoryTotal > 0 {
			return float64(stats.MemoryUsage) / float64(stats.MemoryTotal) * 100, nil
		}
		return 0, nil
	case "disk_usage":
		return 0, nil
	case "network_in":
		return float64(stats.NetworkRX), nil
	case "network_out":
		return float64(stats.NetworkTX), nil
	case "vm_status":
		vm, err := s.vmRepo.FindByID(ctx, vmID)
		if err != nil {
			return 0, err
		}
		if vm.Status == "running" {
			return 1, nil
		}
		return 0, nil
	default:
		return 0, fmt.Errorf("unknown metric: %s", metric)
	}
}

func (s *AlertService) shouldTriggerAlert(rule models.AlertRule, value float64) bool {
	switch rule.Condition {
	case ">":
		return value > rule.Threshold
	case "<":
		return value < rule.Threshold
	case "=":
		return value == rule.Threshold
	case "!=":
		return value != rule.Threshold
	case ">=":
		return value >= rule.Threshold
	case "<=":
		return value <= rule.Threshold
	default:
		return false
	}
}

func (s *AlertService) triggerAlert(ctx context.Context, rule models.AlertRule, vmID, vmName string, currentValue float64) {
	vmUUID, _ := uuid.Parse(vmID)

	history := &models.AlertHistory{
		AlertRuleID:  rule.ID,
		VMID:         &vmUUID,
		Severity:     rule.Severity,
		Metric:       rule.Metric,
		CurrentValue: currentValue,
		Threshold:    rule.Threshold,
		Condition:    rule.Condition,
		Message:      fmt.Sprintf("VM %s: %s %s %.2f (threshold: %.2f)", vmName, rule.Metric, rule.Condition, currentValue, rule.Threshold),
		Status:       "triggered",
	}

	if err := s.historyRepo.Create(ctx, history); err != nil {
		log.Printf("[ALERT] Failed to create alert history: %v", err)
		return
	}

	log.Printf("[ALERT] Alert triggered: %s - VM: %s, Metric: %s, Value: %.2f, Threshold: %.2f",
		rule.Name, vmName, rule.Metric, currentValue, rule.Threshold)

	var channels []string
	if rule.NotifyChannels != "" && rule.NotifyChannels != "{}" {
		json.Unmarshal([]byte(rule.NotifyChannels), &channels)
	}

	if len(channels) > 0 {
		notification := models.AlertNotification{
			AlertRuleID:  rule.ID,
			RuleName:     rule.Name,
			VMID:         &vmUUID,
			VMName:       vmName,
			Severity:     rule.Severity,
			Metric:       rule.Metric,
			CurrentValue: currentValue,
			Threshold:    rule.Threshold,
			Condition:    rule.Condition,
			Message:      history.Message,
			Channels:     channels,
		}
		s.notifyChan <- notification
	}
}

func (s *AlertService) sendNotification(notification models.AlertNotification) {
	for _, channel := range notification.Channels {
		switch channel {
		case "email":
			s.sendEmailNotification(notification)
		case "dingtalk":
			s.sendDingtalkNotification(notification)
		case "webhook":
			s.sendWebhookNotification(notification)
		default:
			log.Printf("[ALERT] Unknown notification channel: %s", channel)
		}
	}
}

func (s *AlertService) sendEmailNotification(notification models.AlertNotification) {
	log.Printf("[ALERT] Sending email notification for alert: %s", notification.RuleName)
}

func (s *AlertService) sendDingtalkNotification(notification models.AlertNotification) {
	log.Printf("[ALERT] Sending DingTalk notification for alert: %s", notification.RuleName)
}

func (s *AlertService) sendWebhookNotification(notification models.AlertNotification) {
	log.Printf("[ALERT] Sending webhook notification for alert: %s", notification.RuleName)
}
