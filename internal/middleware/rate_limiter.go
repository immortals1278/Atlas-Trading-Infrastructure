package middleware

import (
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"
)

type RateLimiter interface {
	Allow(key string) bool
}

// 限流器
type memoryRateLimiter struct {
	limiters map[string]*rate.Limiter
	mu       sync.Mutex
	r        rate.Limit // 每秒允许的请求速率
	b        int        // Bucket 最大容量
	ttl      time.Duration
	accessed map[string]time.Time // 记录最后的存取时间，用于清理过期的限流器
}

func NewMemoryRateLimiter(r rate.Limit, b int, ttl time.Duration) RateLimiter {
	rl := &memoryRateLimiter{
		limiters: make(map[string]*rate.Limiter),
		accessed: make(map[string]time.Time),
		r:        r,
		b:        b,
		ttl:      ttl,
	}
	// 定期清理长时间不活跃的限流器
	go rl.cleanupLoop()
	return rl
}

// 定期清理长时间没使用的限制器
func (m *memoryRateLimiter) cleanupLoop() {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()
	for range ticker.C {
		m.mu.Lock()
		now := time.Now()
		for key, lastSeen := range m.accessed {
			if now.Sub(lastSeen) > m.ttl {
				delete(m.limiters, key)
				delete(m.accessed, key)
			}
		}
		m.mu.Unlock()
	}
}

func (m *memoryRateLimiter) Allow(key string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.limiters[key]; !exists {
		m.limiters[key] = rate.NewLimiter(m.r, m.b)
	}
	m.accessed[key] = time.Now()
	return m.limiters[key].Allow() // 请求来一个消耗一个，桶空了就拒绝
}

func RateLimitMiddleware(limiter RateLimiter) gin.HandlerFunc {
	return func(c *gin.Context) {
		ip := c.ClientIP()
		if !limiter.Allow(ip) {
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error": "请求过于频繁",
			})
			c.Abort()
			return
		}
		c.Next() // 放行 → c.Next() 执行后续中间件和 handler
	}
}
