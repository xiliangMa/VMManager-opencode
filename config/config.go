package config

import (
	"fmt"
	"time"

	"github.com/spf13/viper"
)

type Config struct {
	App          AppConfig          `mapstructure:"app"`
	Database     DatabaseConfig     `mapstructure:"database"`
	Redis        RedisConfig        `mapstructure:"redis"`
	Libvirt      LibvirtConfig      `mapstructure:"libvirt"`
	Storage      StorageConfig      `mapstructure:"storage"`
	JWT          JWTConfig          `mapstructure:"jwt"`
	VNC          VNCConfig          `mapstructure:"vnc"`
	SPICE        SPICEConfig        `mapstructure:"spice"`
	Logging      LoggingConfig      `mapstructure:"logging"`
	RateLimit    RateLimitConfig    `mapstructure:"rate_limit"`
	Quota        QuotaConfig        `mapstructure:"quota"`
	Email        EmailConfig        `mapstructure:"email"`
	Notification NotificationConfig  `mapstructure:"notification"`
}

type AppConfig struct {
	Name         string `mapstructure:"name"`
	Host         string `mapstructure:"host"`
	HTTPPort     int    `mapstructure:"http_port"`
	WSPort       int    `mapstructure:"ws_port"`
	URL          string `mapstructure:"url"`
	Debug        bool   `mapstructure:"debug"`
	UploadPath   string `mapstructure:"upload_path"`
	TemplatePath string `mapstructure:"template_path"`
	LogPath      string `mapstructure:"log_path"`
}

type DatabaseConfig struct {
	Host            string        `mapstructure:"host"`
	Port            int           `mapstructure:"port"`
	Username        string        `mapstructure:"username"`
	Password        string        `mapstructure:"password"`
	Name            string        `mapstructure:"name"`
	SSLMode         string        `mapstructure:"ssl_mode"`
	MaxOpenConns    int           `mapstructure:"max_open_conns"`
	MaxIdleConns    int           `mapstructure:"max_idle_conns"`
	ConnMaxLifetime time.Duration `mapstructure:"conn_max_lifetime"`
	Driver          string        `mapstructure:"driver"`
	Path            string        `mapstructure:"path"`
	Debug           bool          `mapstructure:"debug"`
}

type RedisConfig struct {
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	Password string `mapstructure:"password"`
	DB       int    `mapstructure:"db"`
	PoolSize int    `mapstructure:"pool_size"`
}

type LibvirtConfig struct {
	URI               string        `mapstructure:"uri"`
	ConnectionTimeout time.Duration `mapstructure:"connection_timeout"`
	ReadTimeout       time.Duration `mapstructure:"read_timeout"`
	WriteTimeout      time.Duration `mapstructure:"write_timeout"`
	StoragePath       string        `mapstructure:"storage_path"`
}

type StorageConfig struct {
	Path string `mapstructure:"path"`
}

type JWTConfig struct {
	Secret            string        `mapstructure:"secret"`
	Expiration        time.Duration `mapstructure:"expiration"`
	RefreshExpiration time.Duration `mapstructure:"refresh_expiration"`
}

type VNCConfig struct {
	PortRangeStart int           `mapstructure:"port_range_start"`
	PortRangeEnd   int           `mapstructure:"port_range_end"`
	PasswordLength int           `mapstructure:"password_length"`
	SessionTimeout time.Duration `mapstructure:"session_timeout"`
}

type SPICEConfig struct {
	PortRangeStart int `mapstructure:"port_range_start"`
	PortRangeEnd   int `mapstructure:"port_range_end"`
}

type LoggingConfig struct {
	Level  string `mapstructure:"level"`
	Format string `mapstructure:"format"`
	Output string `mapstructure:"output"`
}

type RateLimitConfig struct {
	RequestsPerSecond int `mapstructure:"requests_per_second"`
	Burst             int `mapstructure:"burst"`
}

type QuotaConfig struct {
	DefaultCPU     int `mapstructure:"default_cpu"`
	DefaultMemory  int `mapstructure:"default_memory"`
	DefaultDisk    int `mapstructure:"default_disk"`
	DefaultVMCount int `mapstructure:"default_vm_count"`
}

type EmailConfig struct {
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	Username string `mapstructure:"username"`
	Password string `mapstructure:"password"`
	From     string `mapstructure:"from"`
	FromName string `mapstructure:"from_name"`
	UseTLS   bool   `mapstructure:"use_tls"`
	UseSSL   bool   `mapstructure:"use_ssl"`
}

type NotificationConfig struct {
	DingtalkAccessToken string `mapstructure:"dingtalk_access_token"`
	DingtalkSecret      string `mapstructure:"dingtalk_secret"`
	WebhookURL          string `mapstructure:"webhook_url"`
	WebhookSecret       string `mapstructure:"webhook_secret"`
}

func Load() (*Config, error) {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")
	viper.AddConfigPath("./config")

	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("failed to read config file: %w", err)
		}
	}

	var cfg Config
	if err := viper.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	return &cfg, nil
}

func (c *DatabaseConfig) DSN() string {
	return fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		c.Host, c.Port, c.Username, c.Password, c.Name, c.SSLMode,
	)
}
