package middleware

import (
	"net/http"
	"time"

	"vmmanager/internal/cache"

	"github.com/gin-gonic/gin"
)

type SessionData struct {
	UserID    string    `json:"user_id"`
	Username  string    `json:"username"`
	Role      string    `json:"role"`
	CreatedAt time.Time `json:"created_at"`
	ExpiresAt time.Time `json:"expires_at"`
}

func SessionMiddleware(cache cache.Cache) gin.HandlerFunc {
	return func(c *gin.Context) {
		sessionID, err := c.Cookie("session_id")
		if err != nil || sessionID == "" {
			c.Next()
			return
		}

		var sessionData SessionData
		err = cache.GetSession(c.Request.Context(), sessionID, &sessionData)
		if err != nil {
			c.SetCookie("session_id", "", -1, "/", "", false, true)
			c.Next()
			return
		}

		c.Set("user_id", sessionData.UserID)
		c.Set("username", sessionData.Username)
		c.Set("role", sessionData.Role)
		c.Set("session_id", sessionID)

		c.Next()
	}
}

func RateLimitMiddleware(cache cache.Cache, requestsPerSecond int, burst int) gin.HandlerFunc {
	return func(c *gin.Context) {
		key := c.ClientIP()

		count, _ := cache.GetRateLimit(c.Request.Context(), key)
		if count >= int64(burst) {
			c.JSON(http.StatusTooManyRequests, gin.H{
				"code":    429,
				"message": "rate limit exceeded",
			})
			c.Abort()
			return
		}

		cache.IncrRateLimit(c.Request.Context(), key, time.Second)
		c.Next()
	}
}

func CacheMiddleware(cache cache.Cache, expiration time.Duration) gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.Method != http.MethodGet {
			c.Next()
			return
		}

		key := c.Request.URL.Path + "?" + c.Request.URL.RawQuery
		var response interface{}
		err := cache.Get(c.Request.Context(), key, &response)
		if err == nil {
			c.JSON(http.StatusOK, response)
			c.Abort()
			return
		}

		c.Set("cache_key", key)
		c.Next()

		if c.Writer.Status() == http.StatusOK {
			cache.Set(c.Request.Context(), key, c.Value("cache_response"), expiration)
		}
	}
}
