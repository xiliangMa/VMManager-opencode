package notification

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"

	"vmmanager/config"
)

type DingtalkNotifier struct {
	cfg *config.Config
}

type DingtalkMessage struct {
	MsgType  string            `json:"msgtype"`
	Markdown *DingtalkMarkdown `json:"markdown"`
}

type DingtalkMarkdown struct {
	Title string `json:"title"`
	Text  string `json:"text"`
}

type DingtalkLink struct {
	Title      string `json:"title"`
	Text       string `json:"text"`
	PicUrl     string `json:"picUrl"`
	MessageUrl string `json:"messageUrl"`
}

func NewDingtalkNotifier(cfg *config.Config) *DingtalkNotifier {
	return &DingtalkNotifier{cfg: cfg}
}

func (n *DingtalkNotifier) SendAlert(data *EmailAlertData, accessToken string) error {
	if accessToken == "" {
		accessToken = n.cfg.Notification.DingtalkAccessToken
	}

	if accessToken == "" {
		log.Printf("[DINGTALK] Mock: Would send alert to Dingtalk, rule: %s, vm: %s", data.RuleName, data.VMName)
		return nil
	}

	title := fmt.Sprintf("[%s] %s", strings.ToUpper(data.Severity), data.RuleName)

	color := n.getSeverityColor(data.Severity)
	text := fmt.Sprintf(`## %s

**告警规则**: %s
**严重级别**: <font color="%s">%s</font>
**虚拟机**: %s
**监控指标**: %s
**当前值**: %.2f%%
**告警条件**: %s %.2f%%
**触发时间**: %s

---
*VMManager 自动发送*`,
		title,
		data.RuleName,
		color,
		strings.ToUpper(data.Severity),
		data.VMName,
		data.Metric,
		data.CurrentValue,
		data.Condition,
		data.Threshold,
		data.Time,
	)

	msg := DingtalkMessage{
		MsgType: "markdown",
		Markdown: &DingtalkMarkdown{
			Title: title,
			Text:  text,
		},
	}

	body, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	apiURL := fmt.Sprintf("https://oapi.dingtalk.com/robot/send?access_token=%s", accessToken)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Post(apiURL, "application/json", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to send to dingtalk: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != 200 {
		return fmt.Errorf("dingtalk returned status %d: %s", resp.StatusCode, string(respBody))
	}

	var result map[string]interface{}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return fmt.Errorf("failed to parse dingtalk response: %w", err)
	}

	if errCode, ok := result["errcode"].(float64); ok && errCode != 0 {
		return fmt.Errorf("dingtalk error: %v", result["errmsg"])
	}

	log.Printf("[DINGTALK] Alert sent: %s - %s", title, data.VMName)
	return nil
}

func (n *DingtalkNotifier) SendSignAlert(data *EmailAlertData, accessToken, secret string) error {
	if accessToken == "" {
		accessToken = n.cfg.Notification.DingtalkAccessToken
	}
	if secret == "" {
		secret = n.cfg.Notification.DingtalkSecret
	}

	if accessToken == "" || secret == "" {
		log.Printf("[DINGTALK] Mock: Would send signed alert to Dingtalk, rule: %s", data.RuleName)
		return nil
	}

	timestamp := time.Now().UnixMilli()
	sign := n.sign(timestamp, secret)

	apiURL := fmt.Sprintf("https://oapi.dingtalk.com/robot/send?access_token=%s&timestamp=%d&sign=%s",
		accessToken, timestamp, url.QueryEscape(sign))

	color := n.getSeverityColor(data.Severity)
	text := fmt.Sprintf(`## [%s] %s

> **告警规则**: %s
> **严重级别**: <font color="%s">%s</font>
> **虚拟机**: %s
> **监控指标**: %s
> **当前值**: %.2f%%
> **告警条件**: %s %.2f%%
> **触发时间**: %s

---
*VMManager 自动发送*`,
		strings.ToUpper(data.Severity),
		data.RuleName,
		data.RuleName,
		color,
		strings.ToUpper(data.Severity),
		data.VMName,
		data.Metric,
		data.CurrentValue,
		data.Condition,
		data.Threshold,
		data.Time,
	)

	msg := DingtalkMessage{
		MsgType: "markdown",
		Markdown: &DingtalkMarkdown{
			Title: fmt.Sprintf("[%s] %s", strings.ToUpper(data.Severity), data.RuleName),
			Text:  text,
		},
	}

	body, _ := json.Marshal(msg)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Post(apiURL, "application/json", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to send signed message: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("dingtalk returned status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	log.Printf("[DINGTALK] Signed alert sent: %s", data.RuleName)
	return nil
}

func (n *DingtalkNotifier) sign(timestamp int64, secret string) string {
	h := hmac.New(sha256.New, []byte(secret))
	h.Write([]byte(fmt.Sprintf("%d\n%s", timestamp, secret)))
	return base64.StdEncoding.EncodeToString(h.Sum(nil))
}

func (n *DingtalkNotifier) getSeverityColor(severity string) string {
	switch strings.ToLower(severity) {
	case "critical":
		return "#FF0000"
	case "warning":
		return "#FFA500"
	case "info":
		return "#0000FF"
	default:
		return "#808080"
	}
}
