package middleware

import (
	"context"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"vmmanager/internal/i18n"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

const (
	RequestIDKey     = "request_id"
	LocaleContextKey = "locale"
	DefaultLocale    = i18n.EnUS
)

func CORS() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Origin, Content-Type, Accept, Authorization, X-Requested-With, X-Request-ID")
		c.Header("Access-Control-Max-Age", "86400")
		c.Header("Access-Control-Expose-Headers", "Content-Length, X-Request-ID")
		c.Header("Access-Control-Allow-Credentials", "true")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
}

func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		requestID := c.GetHeader("X-Request-ID")
		if requestID == "" {
			requestID = uuid.New().String()
		}

		c.Set(RequestIDKey, requestID)
		c.Header("X-Request-ID", requestID)

		c.Next()
	}
}

func Logger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		query := c.Request.URL.RawQuery

		c.Next()

		latency := time.Since(start)
		status := c.Writer.Status()
		requestID, _ := c.Get(RequestIDKey)
		userID, _ := c.Get("user_id")

		if query != "" {
			path = path + "?" + query
		}

		log.Printf("[HTTP] %s | %3d | %13v | %15s | %-7s %s | user=%s",
			requestID,
			status,
			latency,
			c.ClientIP(),
			c.Request.Method,
			path,
			userID,
		)
	}
}

type RateLimiter struct {
	requests map[string]*clientInfo
	mu       sync.RWMutex
	rate     int
	burst    int
}

type clientInfo struct {
	tokens    float64
	lastCheck time.Time
}

func NewRateLimiter(requestsPerSecond int, burst int) *RateLimiter {
	return &RateLimiter{
		requests: make(map[string]*clientInfo),
		rate:     requestsPerSecond,
		burst:    burst,
	}
}

func (rl *RateLimiter) Allow(key string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	info, exists := rl.requests[key]
	if !exists {
		rl.requests[key] = &clientInfo{
			tokens:    float64(rl.burst - 1),
			lastCheck: now,
		}
		return true
	}

	elapsed := now.Sub(info.lastCheck).Seconds()
	info.tokens += elapsed * float64(rl.rate)
	if info.tokens > float64(rl.burst) {
		info.tokens = float64(rl.burst)
	}

	info.lastCheck = now

	if info.tokens >= 1 {
		info.tokens--
		return true
	}

	return false
}

func (rl *RateLimiter) Cleanup(maxAge time.Duration) {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	for key, info := range rl.requests {
		if now.Sub(info.lastCheck) > maxAge {
			delete(rl.requests, key)
		}
	}
}

func RateLimit(requestsPerSecond int, burst int) gin.HandlerFunc {
	limiter := NewRateLimiter(requestsPerSecond, burst)

	go func() {
		ticker := time.NewTicker(time.Minute)
		for range ticker.C {
			limiter.Cleanup(2 * time.Minute)
		}
	}()

	return func(c *gin.Context) {
		key := c.ClientIP()

		if !limiter.Allow(key) {
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"code":    429,
				"message": "rate limit exceeded",
			})
			return
		}

		c.Next()
	}
}

func Timeout(timeout time.Duration) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(c.Request.Context(), timeout)
		defer cancel()

		c.Request = c.Request.WithContext(ctx)

		done := make(chan struct{})
		go func() {
			c.Next()
			close(done)
		}()

		select {
		case <-done:
			return
		case <-ctx.Done():
			c.AbortWithStatusJSON(http.StatusRequestTimeout, gin.H{
				"code":    408,
				"message": "request timeout",
			})
			return
		}
	}
}

type Claims struct {
	UserID   string `json:"user_id"`
	Username string `json:"username"`
	Role     string `json:"role"`
	jwt.RegisteredClaims
}

func JWTRequired(secret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.AbortWithStatusJSON(401, gin.H{
				"code":    401,
				"message": "authorization required",
			})
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" {
			c.AbortWithStatusJSON(401, gin.H{
				"code":    401,
				"message": "invalid authorization format",
			})
			return
		}

		tokenString := parts[1]
		token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
			return []byte(secret), nil
		})

		if err != nil {
			c.AbortWithStatusJSON(401, gin.H{
				"code":    401,
				"message": "invalid token",
			})
			return
		}

		claims, ok := token.Claims.(*Claims)
		if !ok || !token.Valid {
			c.AbortWithStatusJSON(401, gin.H{
				"code":    401,
				"message": "invalid token claims",
			})
			return
		}

		c.Set("user_id", claims.UserID)
		c.Set("username", claims.Username)
		c.Set("role", claims.Role)

		c.Next()
	}
}

func AdminRequired() gin.HandlerFunc {
	return func(c *gin.Context) {
		role, exists := c.Get("role")
		if !exists || role != "admin" {
			c.AbortWithStatusJSON(403, gin.H{
				"code":    403,
				"message": "admin access required",
			})
			return
		}
		c.Next()
	}
}

func GenerateToken(claims *Claims, secret string, expiration time.Duration) (string, error) {
	now := time.Now()
	claims.RegisteredClaims = jwt.RegisteredClaims{
		ExpiresAt: jwt.NewNumericDate(now.Add(expiration)),
		IssuedAt:  jwt.NewNumericDate(now),
		NotBefore: jwt.NewNumericDate(now),
		Issuer:    "vmmanager",
		Subject:   claims.UserID,
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))
}

func RefreshToken(claims *Claims, secret string, expiration time.Duration) (string, error) {
	return GenerateToken(claims, secret, expiration)
}

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
