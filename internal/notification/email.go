package notification

import (
	"bytes"
	"fmt"
	"html/template"
	"log"
	"net/smtp"
	"strings"
	"time"

	"vmmanager/config"
)

type EmailNotifier struct {
	cfg    *config.Config
	client *smtp.Client
}

type EmailAlertData struct {
	RuleName      string
	VMName        string
	Severity      string
	SeverityColor string
	Metric        string
	CurrentValue  float64
	Threshold     float64
	Condition     string
	Message       string
	Time          string
	VMURL         string
	Year          int
}

var emailTemplate = `
<!DOCTYPE html>
<html>
<head>
    <meta charset="utf-8">
    <style>
        body { font-family: Arial, sans-serif; line-height: 1.6; color: #333; }
        .container { max-width: 600px; margin: 0 auto; padding: 20px; }
        .header { background: {{ .SeverityColor }}; color: white; padding: 20px; text-align: center; border-radius: 8px 8px 0 0; }
        .content { background: #f9f9f9; padding: 20px; border: 1px solid #ddd; }
        .severity-{ {.Severity} } { padding: 4px 12px; border-radius: 4px; font-weight: bold; }
        .critical { background: #ff4d4f; color: white; }
        .warning { background: #faad14; color: white; }
        .info { background: #1890ff; color: white; }
        .details { background: white; padding: 15px; border-radius: 4px; margin-top: 15px; }
        .footer { text-align: center; padding: 20px; color: #888; font-size: 12px; }
        table { width: 100%; border-collapse: collapse; margin: 15px 0; }
        td { padding: 8px; border-bottom: 1px solid #eee; }
        td:first-child { font-weight: bold; width: 120px; color: #666; }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>🚨 告警通知</h1>
            <p>Alert Notification</p>
        </div>
        <div class="content">
            <p>您好，</p>
            <p>系统检测到以下告警事件：</p>
            
            <div class="details">
                <h3>{{ .RuleName }}</h3>
                <span class="severity-{{ .Severity }}">{{ .Severity }}</span>
                
                <table>
                    <tr>
                        <td>虚拟机</td>
                        <td>{{ .VMName }}</td>
                    </tr>
                    <tr>
                        <td>监控指标</td>
                        <td>{{ .Metric }}</td>
                    </tr>
                    <tr>
                        <td>当前值</td>
                        <td>{{ .CurrentValue }}%</td>
                    </tr>
                    <tr>
                        <td>告警条件</td>
                        <td>{{ .Condition }} {{ .Threshold }}%</td>
                    </tr>
                    <tr>
                        <td>触发时间</td>
                        <td>{{ .Time }}</td>
                    </tr>
                </table>
                
                {{ if .Message }}
                <p><strong>详细信息：</strong>{{ .Message }}</p>
                {{ end }}
            </div>
            
            <p style="margin-top: 20px;">
                <a href="{{ .VMURL }}" style="display: inline-block; padding: 10px 20px; background: #1890ff; color: white; text-decoration: none; border-radius: 4px;">查看详情</a>
            </p>
        </div>
        <div class="footer">
            <p>此邮件由 VMManager 自动发送，请勿回复。</p>
            <p>© {{ .Year }} VMManager</p>
        </div>
    </div>
</body>
</html>
`

var textTemplate = `
告警通知 Alert Notification
====================

告警规则: {{ .RuleName }}
严重级别: {{ .Severity }}
虚拟机: {{ .VMName }}
监控指标: {{ .Metric }}
当前值: {{ .CurrentValue }}%
告警条件: {{ .Condition }} {{ .Threshold }}%
触发时间: {{ .Time }}
{{ if .Message }}
详细信息: {{ .Message }}
{{ end }}

---
此邮件由 VMManager 自动发送，请勿回复。
`

func NewEmailNotifier(cfg *config.Config) *EmailNotifier {
	return &EmailNotifier{cfg: cfg}
}

func (n *EmailNotifier) SendAlert(data *EmailAlertData, recipients []string) error {
	if len(recipients) == 0 {
		return nil
	}

	data.SeverityColor = n.getSeverityColor(data.Severity)
	data.VMURL = fmt.Sprintf("%s/#/vms/%s", n.cfg.App.URL, data.VMName)
	data.Year = time.Now().Year()

	subject := fmt.Sprintf("[%s] %s - %s", strings.ToUpper(data.Severity), data.RuleName, data.VMName)

	if n.cfg.Email.Host == "" {
		log.Printf("[EMAIL] Mock: Would send to %v, subject: %s", recipients, subject)
		return nil
	}

	auth := smtp.PlainAuth("", n.cfg.Email.Username, n.cfg.Email.Password, n.cfg.Email.Host)
	addr := fmt.Sprintf("%s:%d", n.cfg.Email.Host, n.cfg.Email.Port)

	htmlBody, err := n.parseHTMLTemplate(data)
	if err != nil {
		return fmt.Errorf("failed to parse email template: %w", err)
	}

	body := fmt.Sprintf(`From: %s <%s>
To: %s
Subject: %s
MIME-Version: 1.0
Content-Type: text/html; charset="utf-8"

%s
`, n.cfg.Email.FromName, n.cfg.Email.From, strings.Join(recipients, ","), subject, htmlBody)

	err = smtp.SendMail(addr, auth, n.cfg.Email.From, recipients, []byte(body))
	if err != nil {
		return fmt.Errorf("failed to send email: %w", err)
	}

	log.Printf("[EMAIL] Alert sent to %v: %s", recipients, subject)
	return nil
}

func (n *EmailNotifier) parseHTMLTemplate(data *EmailAlertData) (string, error) {
	tmpl, err := template.New("email").Parse(emailTemplate)
	if err != nil {
		return "", err
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", err
	}

	return buf.String(), nil
}

func (n *EmailNotifier) getSeverityColor(severity string) string {
	switch strings.ToLower(severity) {
	case "critical":
		return "#ff4d4f"
	case "warning":
		return "#faad14"
	case "info":
		return "#1890ff"
	default:
		return "#722ed1"
	}
}

func (n *EmailNotifier) SendTestEmail(recipient string) error {
	data := &EmailAlertData{
		RuleName:     "测试告警",
		VMName:       "test-vm",
		Severity:     "info",
		Metric:       "CPU Usage",
		CurrentValue: 50.0,
		Threshold:    80.0,
		Condition:    ">",
		Message:      "这是一封测试邮件，用于验证邮件配置是否正确。",
		Time:         time.Now().Format("2006-01-02 15:04:05"),
	}
	return n.SendAlert(data, []string{recipient})
}
