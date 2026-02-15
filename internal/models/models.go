package models

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
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
	ID                uuid.UUID  `gorm:"type:uuid;primaryKey" json:"id"`
	Name              string     `gorm:"size:100;not null" json:"name"`
	Description       string     `gorm:"type:text" json:"description"`
	OSType            string     `gorm:"size:50;not null" json:"osType"`
	OSVersion         string     `gorm:"size:50" json:"osVersion"`
	Architecture      string     `gorm:"size:20;default:'arm64'" json:"architecture"`
	Format            string     `gorm:"size:20;default:'qcow2'" json:"format"`
	CPUMin            int        `gorm:"default:1" json:"cpuMin"`
	CPUMax            int        `gorm:"default:4" json:"cpuMax"`
	MemoryMin         int        `gorm:"default:1024" json:"memoryMin"`
	MemoryMax         int        `gorm:"default:8192" json:"memoryMax"`
	DiskMin           int        `gorm:"default:20" json:"diskMin"`
	DiskMax           int        `gorm:"default:500" json:"diskMax"`
	TemplatePath      string     `gorm:"size:500;not null" json:"templatePath"`
	IconURL           string     `gorm:"size:500" json:"iconUrl"`
	ScreenshotURLs    []string   `gorm:"type:text[]" json:"screenshotUrls"`
	DiskSize          int64      `gorm:"not null" json:"diskSize"`
	MD5               string     `gorm:"size:32" json:"md5"`
	SHA256            string     `gorm:"size:64" json:"sha256"`
	IsPublic          bool       `gorm:"default:true" json:"isPublic"`
	IsActive          bool       `gorm:"default:true" json:"isActive"`
	Downloads         int        `gorm:"default:0" json:"downloads"`
	CreatedBy         *uuid.UUID `gorm:"type:uuid" json:"createdBy"`
	CreatedAt         time.Time  `json:"createdAt"`
	UpdatedAt         time.Time  `json:"updatedAt"`
	ISOPath           string     `gorm:"size:500" json:"isoPath"`
	InstallScript     string     `gorm:"type:text" json:"installScript"`
	PostInstallScript string     `gorm:"type:text" json:"postInstallScript"`
}

type VirtualMachine struct {
	ID                uuid.UUID   `gorm:"type:uuid;primaryKey" json:"id"`
	Name              string      `gorm:"size:100;not null" json:"name"`
	Description       string      `gorm:"type:text" json:"description"`
	TemplateID        *uuid.UUID  `gorm:"type:uuid" json:"templateId"`
	OwnerID           uuid.UUID   `gorm:"type:uuid;not null" json:"ownerId"`
	Status            string      `gorm:"size:20;default:'stopped'" json:"status"`
	Architecture      string      `gorm:"size:20;default:'x86_64'" json:"architecture"`
	VNCPort           int         `json:"vncPort"`
	VNCPassword       string      `gorm:"size:20" json:"-"`
	SPICEPort         int         `json:"spicePort"`
	MACAddress        string      `gorm:"size:17;uniqueIndex" json:"macAddress"`
	IPAddress         string      `json:"ipAddress"`
	Gateway           string      `json:"gateway"`
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
	IsInstalled       bool        `gorm:"default:false" json:"isInstalled"`
	InstallStatus     string      `gorm:"size:50;default:''" json:"installStatus"`
	InstallProgress   int         `gorm:"default:0" json:"installProgress"`
	AgentInstalled    bool        `gorm:"default:false" json:"agentInstalled"`
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
	IPAddress    string     `json:"ipAddress"`
	UserAgent    string     `gorm:"type:text" json:"userAgent"`
	Status       string     `gorm:"size:20;default:'success'" json:"status"`
	ErrorMessage string     `gorm:"type:text" json:"errorMessage"`
	CreatedAt    time.Time  `gorm:"index:idx_audit_created" json:"createdAt"`
}

type TemplateUpload struct {
	ID             uuid.UUID  `gorm:"type:uuid;primaryKey" json:"id"`
	Name           string     `gorm:"size:100;not null" json:"name"`
	Description    string     `gorm:"type:text" json:"description"`
	FileName       string     `gorm:"size:255;not null" json:"fileName"`
	FileSize       int64      `gorm:"not null" json:"fileSize"`
	Format         string     `gorm:"size:20" json:"format"`
	Architecture   string     `gorm:"size:20" json:"architecture"`
	UploadPath     string     `gorm:"size:500" json:"uploadPath"`
	TempPath       string     `gorm:"size:500" json:"tempPath"`
	Status         string     `gorm:"size:20;default:'uploading'" json:"status"`
	Progress       int        `gorm:"default:0" json:"progress"`
	UploadedChunks string     `gorm:"type:text" json:"uploadedChunks"`
	TotalChunks    int        `gorm:"default:0" json:"totalChunks"`
	ChunkSize      int64      `gorm:"default:0" json:"chunkSize"`
	ErrorMessage   string     `gorm:"type:text" json:"errorMessage"`
	UploadedBy     *uuid.UUID `gorm:"type:uuid" json:"uploadedBy"`
	CreatedAt      time.Time  `json:"createdAt"`
	CompletedAt    *time.Time `json:"completedAt"`
}

type AlertRule struct {
	ID             uuid.UUID  `gorm:"type:uuid;primaryKey" json:"id"`
	Name           string     `gorm:"size:100;not null" json:"name"`
	Description    string     `gorm:"type:text" json:"description"`
	Metric         string     `gorm:"size:50;not null" json:"metric"`
	Condition      string     `gorm:"size:10;not null" json:"condition"`
	Threshold      float64    `gorm:"type:decimal(10,2)" json:"threshold"`
	Duration       int        `gorm:"default:5" json:"duration"`
	Severity       string     `gorm:"size:20;not null" json:"severity"`
	Enabled        bool       `gorm:"default:true" json:"enabled"`
	NotifyChannels string     `gorm:"type:text" json:"-"`
	NotifyUsers    string     `gorm:"type:text" json:"-"`
	VMIDs          string     `gorm:"type:text" json:"-"`
	IsGlobal       bool       `gorm:"default:false" json:"isGlobal"`
	CreatedBy      *uuid.UUID `gorm:"type:uuid" json:"createdBy"`
	CreatedAt      time.Time  `json:"createdAt"`
	UpdatedAt      time.Time  `json:"updatedAt"`
}

func (a *AlertRule) BeforeCreate(tx *gorm.DB) (err error) {
	if a.ID == uuid.Nil {
		a.ID = uuid.New()
	}
	return
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

type ISO struct {
	ID           uuid.UUID  `gorm:"type:uuid;primaryKey" json:"id"`
	Name         string     `gorm:"size:255;not null" json:"name"`
	Description  string     `gorm:"type:text" json:"description"`
	FileName     string     `gorm:"size:255;not null" json:"fileName"`
	FileSize     int64      `gorm:"not null" json:"fileSize"`
	ISOPath      string     `gorm:"size:500;not null" json:"isoPath"`
	MD5          string     `gorm:"size:32" json:"md5"`
	SHA256       string     `gorm:"size:64" json:"sha256"`
	OSType       string     `gorm:"size:50" json:"osType"`
	OSVersion    string     `gorm:"size:50" json:"osVersion"`
	Architecture string     `gorm:"size:20;default:'x86_64'" json:"architecture"`
	Status       string     `gorm:"size:20;default:'active'" json:"status"`
	UploadedBy   *uuid.UUID `gorm:"type:uuid" json:"uploadedBy"`
	CreatedAt    time.Time  `json:"createdAt"`
	UpdatedAt    time.Time  `json:"updatedAt"`
}

func (i *ISO) BeforeCreate(tx *gorm.DB) (err error) {
	if i.ID == uuid.Nil {
		i.ID = uuid.New()
	}
	return
}

type ISOUpload struct {
	ID             uuid.UUID  `gorm:"type:uuid;primaryKey" json:"id"`
	Name           string     `gorm:"size:255;not null" json:"name"`
	Description    string     `gorm:"type:text" json:"description"`
	FileName       string     `gorm:"size:255;not null" json:"fileName"`
	FileSize       int64      `gorm:"not null" json:"fileSize"`
	Architecture   string     `gorm:"size:20" json:"architecture"`
	OSType         string     `gorm:"size:50" json:"osType"`
	OSVersion      string     `gorm:"size:50" json:"osVersion"`
	UploadPath     string     `gorm:"size:500" json:"uploadPath"`
	TempPath       string     `gorm:"size:500" json:"tempPath"`
	Status         string     `gorm:"size:20;default:'uploading'" json:"status"`
	Progress       int        `gorm:"default:0" json:"progress"`
	UploadedChunks string     `gorm:"type:text" json:"uploadedChunks"`
	TotalChunks    int        `gorm:"default:0" json:"totalChunks"`
	ChunkSize      int64      `gorm:"default:0" json:"chunkSize"`
	ErrorMessage   string     `gorm:"type:text" json:"errorMessage"`
	UploadedBy     *uuid.UUID `gorm:"type:uuid" json:"uploadedBy"`
	CreatedAt      time.Time  `json:"createdAt"`
	CompletedAt    *time.Time `json:"completedAt"`
}

func (u *ISOUpload) BeforeCreate(tx *gorm.DB) (err error) {
	if u.ID == uuid.Nil {
		u.ID = uuid.New()
	}
	return
}

type VirtualNetwork struct {
	ID          uuid.UUID  `gorm:"type:uuid;primaryKey" json:"id"`
	Name        string     `gorm:"size:100;not null;uniqueIndex" json:"name"`
	Description string     `gorm:"type:text" json:"description"`
	NetworkType string     `gorm:"size:20;not null;default:'nat'" json:"networkType"`
	BridgeName  string     `gorm:"size:100" json:"bridgeName"`
	Subnet      string     `gorm:"size:20" json:"subnet"`
	Gateway     string     `gorm:"size:20" json:"gateway"`
	DHCPStart   string     `gorm:"size:20" json:"dhcpStart"`
	DHCPEnd     string     `gorm:"size:20" json:"dhcpEnd"`
	DHCPEnabled bool       `gorm:"default:true" json:"dhcpEnabled"`
	Autostart   bool       `gorm:"default:true" json:"autostart"`
	Active      bool       `gorm:"default:false" json:"active"`
	XMLDef      string     `gorm:"type:text" json:"xmlDef"`
	CreatedBy   *uuid.UUID `gorm:"type:uuid" json:"createdBy"`
	CreatedAt   time.Time  `json:"createdAt"`
	UpdatedAt   time.Time  `json:"updatedAt"`
}

func (n *VirtualNetwork) BeforeCreate(tx *gorm.DB) (err error) {
	if n.ID == uuid.Nil {
		n.ID = uuid.New()
	}
	return
}

type StoragePool struct {
	ID          uuid.UUID  `gorm:"type:uuid;primaryKey" json:"id"`
	Name        string     `gorm:"size:100;not null;uniqueIndex" json:"name"`
	Description string     `gorm:"type:text" json:"description"`
	PoolType    string     `gorm:"size:20;not null;default:'dir'" json:"poolType"`
	TargetPath  string     `gorm:"size:500" json:"targetPath"`
	SourcePath  string     `gorm:"size:500" json:"sourcePath"`
	Capacity    int64      `gorm:"default:0" json:"capacity"`
	Available   int64      `gorm:"default:0" json:"available"`
	Used        int64      `gorm:"default:0" json:"used"`
	Active      bool       `gorm:"default:false" json:"active"`
	Autostart   bool       `gorm:"default:true" json:"autostart"`
	XMLDef      string     `gorm:"type:text" json:"xmlDef"`
	CreatedBy   *uuid.UUID `gorm:"type:uuid" json:"createdBy"`
	CreatedAt   time.Time  `json:"createdAt"`
	UpdatedAt   time.Time  `json:"updatedAt"`
}

func (s *StoragePool) BeforeCreate(tx *gorm.DB) (err error) {
	if s.ID == uuid.Nil {
		s.ID = uuid.New()
	}
	return
}

type StorageVolume struct {
	ID         uuid.UUID  `gorm:"type:uuid;primaryKey" json:"id"`
	PoolID     uuid.UUID  `gorm:"type:uuid;not null;index" json:"poolId"`
	Name       string     `gorm:"size:255;not null" json:"name"`
	VolumeType string     `gorm:"size:20" json:"volumeType"`
	Capacity   int64      `gorm:"default:0" json:"capacity"`
	Allocation int64      `gorm:"default:0" json:"allocation"`
	Format     string     `gorm:"size:20" json:"format"`
	Path       string     `gorm:"size:500" json:"path"`
	VMID       *uuid.UUID `gorm:"type:uuid;index" json:"vmId"`
	CreatedAt  time.Time  `json:"createdAt"`
}

func (v *StorageVolume) BeforeCreate(tx *gorm.DB) (err error) {
	if v.ID == uuid.Nil {
		v.ID = uuid.New()
	}
	return
}

type VMBackup struct {
	ID          uuid.UUID  `gorm:"type:uuid;primaryKey" json:"id"`
	VMID        uuid.UUID  `gorm:"type:uuid;not null;index" json:"vmId"`
	Name        string     `gorm:"size:255;not null" json:"name"`
	Description string     `gorm:"type:text" json:"description"`
	BackupType  string     `gorm:"size:20;not null;default:'full'" json:"backupType"`
	Status      string     `gorm:"size:20;not null;default:'pending'" json:"status"`
	FilePath    string     `gorm:"size:500" json:"filePath"`
	FileSize    int64      `gorm:"default:0" json:"fileSize"`
	Progress    int        `gorm:"default:0" json:"progress"`
	ScheduledAt *time.Time `json:"scheduledAt"`
	StartedAt   *time.Time `json:"startedAt"`
	CompletedAt *time.Time `json:"completedAt"`
	ExpiresAt   *time.Time `json:"expiresAt"`
	ErrorMsg    string     `gorm:"type:text" json:"errorMsg"`
	CreatedBy   *uuid.UUID `gorm:"type:uuid" json:"createdBy"`
	CreatedAt   time.Time  `json:"createdAt"`
	UpdatedAt   time.Time  `json:"updatedAt"`
}

func (b *VMBackup) BeforeCreate(tx *gorm.DB) (err error) {
	if b.ID == uuid.Nil {
		b.ID = uuid.New()
	}
	return
}

type BackupSchedule struct {
	ID         uuid.UUID  `gorm:"type:uuid;primaryKey" json:"id"`
	VMID       uuid.UUID  `gorm:"type:uuid;not null;index" json:"vmId"`
	Name       string     `gorm:"size:255;not null" json:"name"`
	CronExpr   string     `gorm:"size:100;not null" json:"cronExpr"`
	BackupType string     `gorm:"size:20;not null;default:'full'" json:"backupType"`
	Retention  int        `gorm:"default:7" json:"retention"`
	Enabled    bool       `gorm:"default:true" json:"enabled"`
	LastRunAt  *time.Time `json:"lastRunAt"`
	NextRunAt  *time.Time `json:"nextRunAt"`
	CreatedBy  *uuid.UUID `gorm:"type:uuid" json:"createdBy"`
	CreatedAt  time.Time  `json:"createdAt"`
	UpdatedAt  time.Time  `json:"updatedAt"`
}

func (s *BackupSchedule) BeforeCreate(tx *gorm.DB) (err error) {
	if s.ID == uuid.Nil {
		s.ID = uuid.New()
	}
	return
}

type VMSnapshot struct {
	ID          uuid.UUID  `gorm:"type:uuid;primaryKey" json:"id"`
	VMID        uuid.UUID  `gorm:"type:uuid;not null;index" json:"vmId"`
	Name        string     `gorm:"size:255;not null;uniqueIndex:idx_vm_snapshot" json:"name"`
	Description string     `gorm:"type:text" json:"description"`
	Status      string     `gorm:"size:20;not null;default:'created'" json:"status"`
	IsCurrent   bool       `gorm:"default:false" json:"isCurrent"`
	ParentID    *uuid.UUID `gorm:"type:uuid" json:"parentId"`
	CreatedBy   *uuid.UUID `gorm:"type:uuid" json:"createdBy"`
	CreatedAt   time.Time  `json:"createdAt"`
}

func (s *VMSnapshot) BeforeCreate(tx *gorm.DB) (err error) {
	if s.ID == uuid.Nil {
		s.ID = uuid.New()
	}
	return
}
