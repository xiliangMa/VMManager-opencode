package handlers

import (
	"vmmanager/internal/api/middleware"
	"vmmanager/internal/i18n"

	"github.com/gin-gonic/gin"
)

func t(c *gin.Context, key string) string {
	locale := middleware.GetLocale(c)
	return i18n.GetInstance().T(key, locale)
}
