package notification

import (
	"context"
	"log"
	"strings"
	"time"

	"vmmanager/config"
	"vmmanager/internal/models"
	"vmmanager/internal/repository"

	"github.com/google/uuid"
)

type NotificationManager struct {
	cfg              *config.Config
	emailNotifier    *EmailNotifier
	dingtalkNotifier *DingtalkNotifier
	webhookNotifier  *WebhookNotifier
	alertRuleRepo    *repository.AlertRuleRepository
	alertHistoryRepo *repository.AlertHistoryRepository
}

func NewNotificationManager(
	cfg *config.Config,
	alertRuleRepo *repository.AlertRuleRepository,
	alertHistoryRepo *repository.AlertHistoryRepository,
) *NotificationManager {
	return &NotificationManager{
		cfg:              cfg,
		emailNotifier:    NewEmailNotifier(cfg),
		dingtalkNotifier: NewDingtalkNotifier(cfg),
		webhookNotifier:  NewWebhookNotifier(cfg),
		alertRuleRepo:    alertRuleRepo,
		alertHistoryRepo: alertHistoryRepo,
	}
}

func (m *NotificationManager) CheckAndSendAlert(
	vmID string,
	vmName string,
	metric string,
	currentValue float64,
) ([]models.AlertHistory, error) {
	var histories []models.AlertHistory
	ctx := context.Background()

	rules, err := m.alertRuleRepo.FindByVMAndMetric(ctx, vmID, metric)
	if err != nil {
		return nil, err
	}

	for _, rule := range rules {
		if !rule.Enabled {
			continue
		}

		triggered := m.evaluateCondition(currentValue, rule.Threshold, rule.Condition)
		if !triggered {
			continue
		}

		history, err := m.sendAlertNotification(ctx, &rule, vmID, vmName, metric, currentValue)
		if err != nil {
			log.Printf("[ALERT] Failed to send notification for rule %s: %v", rule.Name, err)
			continue
		}

		histories = append(histories, *history)
	}

	return histories, nil
}

func (m *NotificationManager) evaluateCondition(currentValue, threshold float64, condition string) bool {
	switch strings.ToLower(condition) {
	case ">":
		return currentValue > threshold
	case ">=":
		return currentValue >= threshold
	case "<":
		return currentValue < threshold
	case "<=":
		return currentValue <= threshold
	case "==":
		return currentValue == threshold
	case "!=":
		return currentValue != threshold
	default:
		return false
	}
}

func (m *NotificationManager) sendAlertNotification(
	ctx context.Context,
	rule *models.AlertRule,
	vmID string,
	vmName string,
	metric string,
	currentValue float64,
) (*models.AlertHistory, error) {
	vmUUID := uuid.MustParse(vmID)
	history := &models.AlertHistory{
		AlertRuleID:  rule.ID,
		VMID:         &vmUUID,
		Severity:     rule.Severity,
		Metric:       metric,
		CurrentValue: currentValue,
		Threshold:    rule.Threshold,
		Condition:    rule.Condition,
		Message:      rule.Description,
		Status:       "triggered",
	}

	data := &EmailAlertData{
		RuleName:     rule.Name,
		VMName:       vmName,
		Severity:     rule.Severity,
		Metric:       metric,
		CurrentValue: currentValue,
		Threshold:    rule.Threshold,
		Condition:    rule.Condition,
		Message:      rule.Description,
		Time:         time.Now().Format("2006-01-02 15:04:05"),
	}

	for _, channel := range rule.NotifyChannels {
		var notifErr error

		switch strings.ToLower(channel) {
		case "email":
			notifErr = m.sendEmailNotification(data, rule.NotifyUsers)
		case "dingtalk":
			notifErr = m.dingtalkNotifier.SendAlert(data, "")
		case "webhook":
			notifErr = m.webhookNotifier.SendAlert(data, "")
		}

		if notifErr != nil {
			log.Printf("[ALERT] Failed to send via %s: %v", channel, notifErr)
		}
	}

	now := time.Now()
	history.NotifiedAt = &now

	if err := m.alertHistoryRepo.Create(ctx, history); err != nil {
		log.Printf("[ALERT] Failed to save alert history: %v", err)
	}

	log.Printf("[ALERT] Triggered: %s - %s (value: %.2f%%, condition: %s %.2f%%)",
		rule.Name, vmName, currentValue, rule.Condition, rule.Threshold)

	return history, nil
}

func (m *NotificationManager) sendEmailNotification(data *EmailAlertData, recipients []string) error {
	if len(recipients) == 0 {
		recipients = []string{"admin@example.com"}
	}
	return m.emailNotifier.SendAlert(data, recipients)
}

func (m *NotificationManager) ResolveAlert(ctx context.Context, historyID string) error {
	return m.alertHistoryRepo.Resolve(ctx, historyID)
}

func (m *NotificationManager) GetAlertHistory(ctx context.Context, vmID string, limit int) ([]models.AlertHistory, error) {
	return m.alertHistoryRepo.GetByVM(ctx, vmID, limit)
}

func (m *NotificationManager) GetActiveAlerts(ctx context.Context) ([]models.AlertHistory, error) {
	return m.alertHistoryRepo.GetActive(ctx)
}
