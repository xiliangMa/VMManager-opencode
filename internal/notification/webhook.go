package notification

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"vmmanager/config"
)

type WebhookNotifier struct {
	cfg *config.Config
}

type WebhookPayload struct {
	RuleName     string  `json:"rule_name"`
	RuleID       string  `json:"rule_id"`
	VMID         string  `json:"vm_id,omitempty"`
	VMName       string  `json:"vm_name"`
	Severity     string  `json:"severity"`
	Metric       string  `json:"metric"`
	CurrentValue float64 `json:"current_value"`
	Threshold    float64 `json:"threshold"`
	Condition    string  `json:"condition"`
	Message      string  `json:"message"`
	Timestamp    string  `json:"timestamp"`
	Status       string  `json:"status"`
}

type WebhookPayloadV2 struct {
	Version     string                 `json:"version"`
	EventType   string                 `json:"event_type"`
	TriggeredAt string                 `json:"triggered_at"`
	Data        WebhookAlertData       `json:"data"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

type WebhookAlertData struct {
	AlertID      string            `json:"alert_id"`
	AlertName    string            `json:"alert_name"`
	Severity     string            `json:"severity"`
	Target       string            `json:"target"`
	TargetType   string            `json:"target_type"`
	Condition    string            `json:"condition"`
	CurrentValue float64           `json:"current_value"`
	Threshold    float64           `json:"threshold"`
	Labels       map[string]string `json:"labels,omitempty"`
	Annotations  map[string]string `json:"annotations,omitempty"`
}

func NewWebhookNotifier(cfg *config.Config) *WebhookNotifier {
	return &WebhookNotifier{cfg: cfg}
}

func (n *WebhookNotifier) SendAlert(data *EmailAlertData, webhookURL string) error {
	if webhookURL == "" {
		webhookURL = n.cfg.Notification.WebhookURL
	}

	if webhookURL == "" {
		log.Printf("[WEBHOOK] Mock: Would send alert to webhook, rule: %s, vm: %s", data.RuleName, data.VMName)
		return nil
	}

	payload := WebhookPayload{
		RuleName:     data.RuleName,
		VMName:       data.VMName,
		Severity:     data.Severity,
		Metric:       data.Metric,
		CurrentValue: data.CurrentValue,
		Threshold:    data.Threshold,
		Condition:    data.Condition,
		Message:      data.Message,
		Timestamp:    data.Time,
		Status:       "firing",
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal webhook payload: %w", err)
	}

	client := &http.Client{Timeout: 10 * time.Second}
	req, err := http.NewRequest("POST", webhookURL, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to create webhook request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "VMManager-AlertNotifier")
	req.Header.Set("X-Alert-Rule", data.RuleName)
	req.Header.Set("X-Alert-Severity", data.Severity)

	if n.cfg.Notification.WebhookSecret != "" {
		signature := n.signWebhook(body, n.cfg.Notification.WebhookSecret)
		req.Header.Set("X-Signature", signature)
	}

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send webhook: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)

	if resp.StatusCode >= 400 {
		return fmt.Errorf("webhook returned status %d: %s", resp.StatusCode, string(respBody))
	}

	log.Printf("[WEBHOOK] Alert sent: %s - %s (status: %d)", data.RuleName, data.VMName, resp.StatusCode)
	return nil
}

func (n *WebhookNotifier) SendPrometheusAlert(data *EmailAlertData, webhookURL string) error {
	if webhookURL == "" {
		webhookURL = n.cfg.Notification.WebhookURL
	}

	if webhookURL == "" {
		log.Printf("[WEBHOOK] Mock: Would send Prometheus alert, rule: %s", data.RuleName)
		return nil
	}

	now := time.Now()
	endsAt := now.Add(5 * time.Minute)

	payload := map[string]interface{}{
		"version":  "4",
		"groupKey": fmt.Sprintf("vmmanager:%s", data.Metric),
		"status":   "firing",
		"receiver": "vmmanager",
		"groupLabels": map[string]string{
			"alertname": data.RuleName,
			"severity":  data.Severity,
		},
		"commonLabels": map[string]string{
			"alertname": data.RuleName,
			"severity":  data.Severity,
			"vm":        data.VMName,
			"metric":    data.Metric,
		},
		"commonAnnotations": map[string]string{
			"summary":     fmt.Sprintf("[%s] %s", strings.ToUpper(data.Severity), data.RuleName),
			"description": fmt.Sprintf("VM %s %s is %.2f%%, condition: %s %.2f%%", data.VMName, data.Metric, data.CurrentValue, data.Condition, data.Threshold),
		},
		"externalURL": n.cfg.App.URL,
		"alerts": []map[string]interface{}{
			{
				"status": "firing",
				"labels": map[string]string{
					"alertname": data.RuleName,
					"severity":  data.Severity,
					"vm":        data.VMName,
					"metric":    data.Metric,
				},
				"annotations": map[string]string{
					"summary":     fmt.Sprintf("[%s] %s", strings.ToUpper(data.Severity), data.RuleName),
					"description": fmt.Sprintf("VM %s %s is %.2f%%", data.VMName, data.Metric, data.CurrentValue),
				},
				"startsAt":     now.Format(time.RFC3339),
				"endsAt":       endsAt.Format(time.RFC3339),
				"generatorURL": fmt.Sprintf("%s/#/vms/%s/monitor", n.cfg.App.URL, data.VMName),
			},
		},
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal Prometheus payload: %w", err)
	}

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Post(webhookURL, "application/json", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to send Prometheus webhook: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("Prometheus webhook returned status %d: %s", resp.StatusCode, string(respBody))
	}

	log.Printf("[WEBHOOK] Prometheus alert sent: %s", data.RuleName)
	return nil
}

func (n *WebhookNotifier) signWebhook(body []byte, secret string) string {
	h := hmac.New(sha256.New, []byte(secret))
	h.Write(body)
	return "sha256=" + hex.EncodeToString(h.Sum(nil))
}
