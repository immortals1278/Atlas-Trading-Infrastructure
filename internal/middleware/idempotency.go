package middleware

import (
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

type idempotencyEntry struct {
	StatusCode int       `json:"status_code"`
	Body       []byte    `json:"body"` // 儲存原始 JSON bytes
	ExpiresAt  time.Time `json:"expires_at"`
}

type IdempotencyStore interface {
	Get(key string) *idempotencyEntry
	// ttl 后过期
	Set(key string, statusCode int, body []byte, ttl time.Duration)
}

type memoryIdempotencyStore struct {
	mu    sync.RWMutex
	store map[string]*idempotencyEntry
}

func NewMemoryIdempotencyStore() IdempotencyStore {
	s := &memoryIdempotencyStore{
		store: make(map[string]*idempotencyEntry),
	}
	// 定期清理过期条目
	go s.cleanupLoop()
	return s
}

func (s *memoryIdempotencyStore) Get(key string) *idempotencyEntry {
	s.mu.RLock()
	defer s.mu.RUnlock()
	entry, ok := s.store[key]
	if !ok || time.Now().After(entry.ExpiresAt) { // 已存在且未过期
		return nil
	}
	return entry
}

func (s *memoryIdempotencyStore) Set(key string, statusCode int, body []byte, ttl time.Duration) {
	s.mu.Lock()
	defer s.mu.Unlock()
	bodyCopy := make([]byte, len(body))
	copy(bodyCopy, body)
	s.store[key] = &idempotencyEntry{
		StatusCode: statusCode,
		Body:       bodyCopy,
		ExpiresAt:  time.Now().Add(ttl),
	}
}

func (s *memoryIdempotencyStore) cleanupLoop() {
	ticker := time.NewTicker(10 * time.Minute)
	defer ticker.Stop()
	for range ticker.C {
		s.mu.Lock()
		now := time.Now()
		for key, entry := range s.store {
			if now.After(entry.ExpiresAt) {
				delete(s.store, key)
			}
		}
		s.mu.Unlock()
	}
}

// IdempotencyMiddleware 构建幂等性 Gin 中间件
// 客户端必须在 Header 带 Idempotency-Key，若同一个 Key 已被处理，直接返回缓存结果。
func IdempotencyMiddleware(store IdempotencyStore, ttl time.Duration) gin.HandlerFunc {
	return func(c *gin.Context) {
		key := c.GetHeader("Idempotency-Key")
		if key == "" {
			// 若客户端没带 Key，视为不需要幂等性保护，直接放行
			c.Next()
			return
		}

		// 缓存命中：直接用 c.Data 返回存储好的原始 JSON bytes，不触发任何业务逻辑
		if entry := store.Get(key); entry != nil {
			c.Data(entry.StatusCode, "application/json", entry.Body)
			c.Abort()
			return
		}

		// 缓存未命中：首次请求，用 bodyLogWriter 拦截 Handler 返回的 bytes
		blw := &bodyLogWriter{ResponseWriter: c.Writer}
		c.Writer = blw
		c.Next()

		// Handler 完成后，若状态 < 400 才缓存（错误响应不应被重用）
		if blw.statusCode < 400 && len(blw.body) > 0 {
			store.Set(key, blw.statusCode, blw.body, ttl)
		}
	}
}

// bodyLogWriter 包装 gin.ResponseWriter，同时拦截并记录 Response bytes 以供缓存
type bodyLogWriter struct {
	gin.ResponseWriter
	statusCode int
	body       []byte // 存储 Handler 返回的原始 bytes
}

// WriteHeader 拦截并记录 HTTP Status Code
func (w *bodyLogWriter) WriteHeader(code int) {
	w.statusCode = code
	w.ResponseWriter.WriteHeader(code)
}

// Write 拦截 Response Body bytes：同时写给 Client 并 append 到本地缓存
func (w *bodyLogWriter) Write(b []byte) (int, error) {
	if w.statusCode == 0 {
		w.statusCode = http.StatusOK
	}

	w.body = append(w.body, b...)
	return w.ResponseWriter.Write(b)
}
