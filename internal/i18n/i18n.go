package i18n

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

type Locale string

const (
	ZhCN Locale = "zh-CN"
	EnUS Locale = "en-US"
)

type Translation map[string]string

type I18n struct {
	translations map[Locale]Translation
	current      Locale
	mu           sync.RWMutex
}

var instance *I18n
var once sync.Once

func GetInstance() *I18n {
	once.Do(func() {
		instance = &I18n{
			translations: make(map[Locale]Translation),
			current:      ZhCN,
		}
		instance.loadTranslations()
	})
	return instance
}

func (i *I18n) loadTranslations() {
	translationsDir := "./translations"

	entries, err := os.ReadDir(translationsDir)
	if err != nil {
		return
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		ext := filepath.Ext(entry.Name())
		if ext != ".json" {
			continue
		}

		localeName := strings.TrimSuffix(entry.Name(), ext)
		locale := Locale(localeName)

		data, err := os.ReadFile(filepath.Join(translationsDir, entry.Name()))
		if err != nil {
			continue
		}

		var translation Translation
		if err := json.Unmarshal(data, &translation); err != nil {
			continue
		}

		i.translations[locale] = translation
	}
}

func (i *I18n) SetLocale(locale Locale) {
	i.mu.Lock()
	i.current = locale
	i.mu.Unlock()
}

func (i *I18n) GetLocale() Locale {
	i.mu.RLock()
	defer i.mu.RUnlock()
	return i.current
}

func (i *I18n) T(key string, locale Locale) string {
	i.mu.RLock()
	defer i.mu.RUnlock()

	if translation, ok := i.translations[locale]; ok {
		if value, ok := translation[key]; ok {
			return value
		}
	}

	if translation, ok := i.translations[i.current]; ok {
		if value, ok := translation[key]; ok {
			return value
		}
	}

	return key
}

func (i *I18n) Tr(key string, args ...interface{}) string {
	template := i.T(key, i.current)
	if len(args) == 0 {
		return template
	}

	result := template
	for idx, arg := range args {
		placeholder := "{" + string(rune('0'+idx)) + "}"
		result = strings.ReplaceAll(result, placeholder, toString(arg))
	}
	return result
}

func toString(v interface{}) string {
	switch val := v.(type) {
	case string:
		return val
	default:
		return ""
	}
}

type APIError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Key     string `json:"key,omitempty"`
}

func (i *I18n) CreateError(code int, messageKey string, locale Locale) APIError {
	return APIError{
		Code:    code,
		Message: i.T(messageKey, locale),
		Key:     messageKey,
	}
}

type ErrorKey struct {
	Success          string
	ValidationError  string
	NotFound         string
	Unauthorized     string
	Forbidden        string
	InternalError    string
	RateLimited      string
	QuotaExceeded    string
	VMNotFound       string
	VMNotRunning     string
	TemplateNotFound string
	UserNotFound     string
	InvalidRequest   string
}

var ErrorMessages = ErrorKey{
	Success:          "success",
	ValidationError:  "validation_error",
	NotFound:         "not_found",
	Unauthorized:     "unauthorized",
	Forbidden:        "forbidden",
	InternalError:    "internal_error",
	RateLimited:      "rate_limited",
	QuotaExceeded:    "quota_exceeded",
	VMNotFound:       "vm_not_found",
	VMNotRunning:     "vm_not_running",
	TemplateNotFound: "template_not_found",
	UserNotFound:     "user_not_found",
	InvalidRequest:   "invalid_request",
}
