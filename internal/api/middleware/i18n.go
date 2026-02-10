package middleware

import (
	"strings"

	"vmmanager/internal/i18n"

	"github.com/gin-gonic/gin"
)

const (
	LocaleContextKey = "locale"
	DefaultLocale    = i18n.EnUS
)

func I18n() gin.HandlerFunc {
	return func(c *gin.Context) {
		acceptLanguage := c.GetHeader("Accept-Language")
		locale := parseLocale(acceptLanguage)

		c.Set(LocaleContextKey, locale)
		c.Next()
	}
}

func GetLocale(c *gin.Context) i18n.Locale {
	if locale, exists := c.Get(LocaleContextKey); exists {
		if l, ok := locale.(i18n.Locale); ok {
			return l
		}
	}
	return DefaultLocale
}

func parseLocale(acceptLanguage string) i18n.Locale {
	if acceptLanguage == "" {
		return DefaultLocale
	}

	parts := strings.Split(acceptLanguage, ",")
	if len(parts) == 0 {
		return DefaultLocale
	}

	firstLocale := strings.TrimSpace(parts[0])
	if strings.Contains(firstLocale, ";") {
		firstLocale = strings.Split(firstLocale, ";")[0]
	}

	firstLocale = strings.ToLower(firstLocale)

	if strings.HasPrefix(firstLocale, "zh") {
		return i18n.ZhCN
	} else if strings.HasPrefix(firstLocale, "en") {
		return i18n.EnUS
	}

	return DefaultLocale
}
