package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type AlertHistory struct {
	ID           uuid.UUID  `gorm:"type:uuid;primaryKey" json:"id"`
	AlertRuleID  uuid.UUID  `gorm:"type:uuid;not null;index" json:"alertRuleId"`
	VMID         *uuid.UUID `gorm:"type:uuid;index" json:"vmId"`
	Severity     string     `gorm:"size:20;not null" json:"severity"`
	Metric       string     `gorm:"size:50;not null" json:"metric"`
	CurrentValue float64    `json:"currentValue"`
	Threshold    float64    `json:"threshold"`
	Condition    string     `gorm:"size:10" json:"condition"`
	Message      string     `gorm:"type:text" json:"message"`
	Status       string     `gorm:"size:20;default:'triggered'" json:"status"`
	ResolvedAt   *time.Time `json:"resolvedAt"`
	NotifiedAt   *time.Time `json:"notifiedAt"`
	CreatedAt    time.Time  `json:"createdAt"`
}

func (a *AlertHistory) BeforeCreate(tx *gorm.DB) (err error) {
	if a.ID == uuid.Nil {
		a.ID = uuid.New()
	}
	return
}

type NotificationChannel string

const (
	ChannelEmail    NotificationChannel = "email"
	ChannelDingtalk NotificationChannel = "dingtalk"
	ChannelWebhook  NotificationChannel = "webhook"
)

type AlertNotification struct {
	AlertRuleID  uuid.UUID
	RuleName     string
	VMID         *uuid.UUID
	VMName       string
	Severity     string
	Metric       string
	CurrentValue float64
	Threshold    float64
	Condition    string
	Message      string
	Channels     []string
}
