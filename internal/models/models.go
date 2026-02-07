package models

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type User struct {
	ID           uuid.UUID        `gorm:"type:uuid;primaryKey" json:"id"`
	Username     string           `gorm:"uniqueIndex:idx_users_username;size:50;not null" json:"username"`
	Email        string           `gorm:"uniqueIndex:idx_users_email;size:100;not null" json:"email"`
	PasswordHash string           `gorm:"size:255;not null" json:"-"`
	Role         string           `gorm:"size:20;default:'user'" json:"role"`
	IsActive     bool             `gorm:"default:true" json:"isActive"`
	AvatarURL    string           `gorm:"size:500" json:"avatarUrl"`
	Language     string           `gorm:"size:10;default:'zh-CN'" json:"language"`
	Timezone     string           `gorm:"size:50;default:'Asia/Shanghai'" json:"timezone"`
	QuotaCPU     int              `gorm:"default:4" json:"quotaCpu"`
	QuotaMemory  int              `gorm:"default:8192" json:"quotaMemory"`
	QuotaDisk    int              `gorm:"default:100" json:"quotaDisk"`
	QuotaVMCount int              `gorm:"default:5" json:"quotaVmCount"`
	LastLoginAt  *time.Time       `json:"lastLoginAt"`
	CreatedAt    time.Time        `json:"createdAt"`
	UpdatedAt    time.Time        `json:"updatedAt"`
	VMs          []VirtualMachine `gorm:"foreignKey:OwnerID" json:"-"`
	Templates    []VMTemplate     `gorm:"foreignKey:CreatedBy" json:"-"`
}

func (u *User) BeforeCreate(tx *gorm.DB) (err error) {
	if u.ID == uuid.Nil {
		u.ID = uuid.New()
	}
	return
}

type VMTemplate struct {
	ID             uuid.UUID  `gorm:"type:uuid;primaryKey" json:"id"`
	Name           string     `gorm:"size:100;not null" json:"name"`
	Description    string     `gorm:"type:text" json:"description"`
	OSType         string     `gorm:"size:50;not null" json:"osType"`
	OSVersion      string     `gorm:"size:50" json:"osVersion"`
	Architecture   string     `gorm:"size:20;default:'arm64'" json:"architecture"`
	Format         string     `gorm:"size:20;default:'qcow2'" json:"format"`
	CPUMin         int        `gorm:"default:1" json:"cpuMin"`
	CPUMax         int        `gorm:"default:4" json:"cpuMax"`
	MemoryMin      int        `gorm:"default:1024" json:"memoryMin"`
	MemoryMax      int        `gorm:"default:8192" json:"memoryMax"`
	DiskMin        int        `gorm:"default:20" json:"diskMin"`
	DiskMax        int        `gorm:"default:500" json:"diskMax"`
	TemplatePath   string     `gorm:"size:500;not null" json:"templatePath"`
	IconURL        string     `gorm:"size:500" json:"iconUrl"`
	ScreenshotURLs []string   `gorm:"type:text[]" json:"screenshotUrls"`
	DiskSize       int64      `gorm:"not null" json:"diskSize"`
	IsPublic       bool       `gorm:"default:true" json:"isPublic"`
	IsActive       bool       `gorm:"default:true" json:"isActive"`
	Downloads      int        `gorm:"default:0" json:"downloads"`
	CreatedBy      *uuid.UUID `gorm:"type:uuid" json:"createdBy"`
	CreatedAt      time.Time  `json:"createdAt"`
	UpdatedAt      time.Time  `json:"updatedAt"`
}

type VirtualMachine struct {
	ID                uuid.UUID   `gorm:"type:uuid;primaryKey" json:"id"`
	Name              string      `gorm:"size:100;not null" json:"name"`
	Description       string      `gorm:"type:text" json:"description"`
	TemplateID        *uuid.UUID  `gorm:"type:uuid" json:"templateId"`
	OwnerID           uuid.UUID   `gorm:"type:uuid;not null" json:"ownerId"`
	Status            string      `gorm:"size:20;default:'pending'" json:"status"`
	VNCPort           int         `json:"vncPort"`
	VNCPassword       string      `gorm:"size:20" json:"-"`
	SPICEPort         int         `json:"spicePort"`
	MACAddress        string      `gorm:"size:17;uniqueIndex" json:"macAddress"`
	IPAddress         net.IP      `json:"ipAddress"`
	Gateway           net.IP      `json:"gateway"`
	DNSServers        []string    `gorm:"type:inet[]" json:"dnsServers"`
	CPUAllocated      int         `gorm:"not null" json:"cpuAllocated"`
	MemoryAllocated   int         `gorm:"not null" json:"memoryAllocated"`
	DiskAllocated     int         `gorm:"not null" json:"diskAllocated"`
	DiskPath          string      `gorm:"size:500" json:"diskPath"`
	LibvirtDomainID   int         `json:"libvirtDomainId"`
	LibvirtDomainUUID string      `gorm:"size:50" json:"libvirtDomainUuid"`
	BootOrder         string      `gorm:"size:50;default:'hd,cdrom,network'" json:"bootOrder"`
	VCPUHotplug       bool        `gorm:"default:false" json:"vcpuHotplug"`
	MemoryHotplug     bool        `gorm:"default:false" json:"memoryHotplug"`
	Autostart         bool        `gorm:"default:false" json:"autostart"`
	Notes             string      `gorm:"type:text" json:"notes"`
	Tags              []string    `gorm:"type:text[]" json:"tags"`
	Owner             *User       `gorm:"foreignKey:OwnerID" json:"owner"`
	Template          *VMTemplate `gorm:"foreignKey:TemplateID" json:"template"`
	CreatedAt         time.Time   `json:"createdAt"`
	UpdatedAt         time.Time   `json:"updatedAt"`
	DeletedAt         *time.Time  `gorm:"index" json:"deletedAt"`
}

type VMStats struct {
	ID          uuid.UUID `gorm:"type:uuid;primaryKey" json:"id"`
	VMID        uuid.UUID `gorm:"type:uuid;not null;index:idx_vm_stats_vm_id" json:"vmId"`
	CPUUsage    float64   `gorm:"type:decimal(5,2);default:0" json:"cpuUsage"`
	MemoryUsage int64     `gorm:"default:0" json:"memoryUsage"`
	MemoryTotal int64     `gorm:"default:0" json:"memoryTotal"`
	DiskRead    int64     `gorm:"default:0" json:"diskRead"`
	DiskWrite   int64     `gorm:"default:0" json:"diskWrite"`
	NetworkRX   int64     `gorm:"default:0" json:"networkRx"`
	NetworkTX   int64     `gorm:"default:0" json:"networkTx"`
	CollectedAt time.Time `gorm:"index:idx_vm_stats_collected" json:"collectedAt"`
}

type AuditLog struct {
	ID           uuid.UUID  `gorm:"type:uuid;primaryKey" json:"id"`
	UserID       *uuid.UUID `gorm:"type:uuid;index:idx_audit_user" json:"userId"`
	Action       string     `gorm:"size:50;not null;index:idx_audit_action" json:"action"`
	ResourceType string     `gorm:"size:50;not null;index:idx_audit_resource" json:"resourceType"`
	ResourceID   *uuid.UUID `gorm:"type:uuid" json:"resourceId"`
	Details      string     `gorm:"type:jsonb" json:"details"`
	IPAddress    net.IP     `json:"ipAddress"`
	UserAgent    string     `gorm:"type:text" json:"userAgent"`
	Status       string     `gorm:"size:20;default:'success'" json:"status"`
	ErrorMessage string     `gorm:"type:text" json:"errorMessage"`
	CreatedAt    time.Time  `gorm:"index:idx_audit_created" json:"createdAt"`
}

type TemplateUpload struct {
	ID           uuid.UUID  `gorm:"type:uuid;primaryKey" json:"id"`
	Name         string     `gorm:"size:100;not null" json:"name"`
	Description  string     `gorm:"type:text" json:"description"`
	FileName     string     `gorm:"size:255;not null" json:"fileName"`
	FileSize     int64      `gorm:"not null" json:"fileSize"`
	Format       string     `gorm:"size:20" json:"format"`
	Architecture string     `gorm:"size:20" json:"architecture"`
	UploadPath   string     `gorm:"size:500" json:"uploadPath"`
	TempPath     string     `gorm:"size:500" json:"tempPath"`
	Status       string     `gorm:"size:20;default:'uploading'" json:"status"`
	Progress     int        `gorm:"default:0" json:"progress"`
	ErrorMessage string     `gorm:"type:text" json:"errorMessage"`
	UploadedBy   *uuid.UUID `gorm:"type:uuid" json:"uploadedBy"`
	CreatedAt    time.Time  `json:"createdAt"`
	CompletedAt  *time.Time `json:"completedAt"`
}

func GenerateMACAddress() (string, error) {
	bytes := make([]byte, 3)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return fmt.Sprintf("52:54:00:%02x:%02x:%02x", bytes[0], bytes[1], bytes[2]), nil
}

func GenerateVNCPassword(length int) (string, error) {
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes)[:length], nil
}

func (t *VMTemplate) BeforeCreate(tx *gorm.DB) (err error) {
	if t.ID == uuid.Nil {
		t.ID = uuid.New()
	}
	return
}

func (vm *VirtualMachine) BeforeCreate(tx *gorm.DB) (err error) {
	if vm.ID == uuid.Nil {
		vm.ID = uuid.New()
	}
	return
}

func (s *VMStats) BeforeCreate(tx *gorm.DB) (err error) {
	if s.ID == uuid.Nil {
		s.ID = uuid.New()
	}
	return
}

func (a *AuditLog) BeforeCreate(tx *gorm.DB) (err error) {
	if a.ID == uuid.Nil {
		a.ID = uuid.New()
	}
	return
}

func (u *TemplateUpload) BeforeCreate(tx *gorm.DB) (err error) {
	if u.ID == uuid.Nil {
		u.ID = uuid.New()
	}
	return
}
