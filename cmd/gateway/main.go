package main

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"syscall"
	"time"

	"atlas-trading-infrastructure/internal/infrastructure/logger"
	infraredis "atlas-trading-infrastructure/internal/infrastructure/redis"
	"atlas-trading-infrastructure/internal/middleware"

	"net/http/httputil"

	"go.uber.org/zap"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

func main() {

	defer logger.Sync()

	port := "8100"

	orderServiceURL := "http://localhost:8103"

	// 将orderServiceURL 字符串为 *url.URL
	orderURL, err := url.Parse(orderServiceURL)
	if err != nil {
		logger.Log.Fatal("order service url 格式错误", zap.Error(err))
	}

	// 初始化redis
	redisCfg := infraredis.DefaultConfig()
	redisClient, redisErr := infraredis.NewClient(redisCfg)
	if redisErr != nil {
		logger.Log.Warn("gateway redis 不可用", zap.Error(redisErr))
	}

	privateRateLimit := 100

	// TODO配置redis相关中间件
	var privateLimiter middleware.RateLimiter
	var idempStore middleware.IdempotencyStore

	if redisClient != nil {
		privateLimiter = middleware.NewRedisRateLimiter(redisClient, privateRateLimit, time.Second)
		idempStore = middleware.NewRedisIdempotencyStore(redisClient)
	} else {
		privateLimiter = middleware.NewMemoryRateLimiter(100, privateRateLimit, 10*time.Minute)
		idempStore = middleware.NewMemoryIdempotencyStore()
	}

	// 设置反向代理
	orderProxy := newReverseProxy(orderURL)

	// 设置gin
	r := gin.New()
	r.Use(gin.Recovery()) // 捕获panic
	// 网关是唯一入口，CORS 放这里统一处理，后面各服务不用各自配，避免重复和不一致。
	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"http://localhost:5173"},
		AllowMethods:     []string{"GET", "POST", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Idempotency-Key", "X-User-ID"},
		AllowCredentials: true, // // 允许携带 Cookie/Authorization 等凭证
		MaxAge:           12 * time.Hour,
	}))
	// 注册要转发的路由
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})
	r.Any("/docx/*path", gin.WrapH(orderProxy))
	r.Any("/swagger/*path", gin.WrapH(orderProxy))

	apiGroup := r.Group("/api/v1")
	{
		private := apiGroup.Group("/")
		private.Use(middleware.RateLimitMiddleware(privateLimiter))
		private.GET("/orders", gin.WrapH(orderProxy))
		private.GET("/orders/:id", gin.WrapH(orderProxy))
		private.DELETE("/orders/:id", gin.WrapH(orderProxy))
		private.GET("/accounts", gin.WrapH(orderProxy))

		orders := apiGroup.Group("/")
		orders.Use(middleware.RateLimitMiddleware(privateLimiter))
		orders.Use(middleware.IdempotencyMiddleware(idempStore, 24*time.Hour))
		orders.POST("/orders", gin.WrapH(orderProxy))
		orders.POST("/orders/batch", gin.WrapH(orderProxy))

	}

	r.NoRoute(gin.WrapH(orderProxy))

	srv := &http.Server{
		Addr:              fmt.Sprintf(":%s", port),
		Handler:           r,
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		logger.Log.Info("gateway启动完成",
			zap.String("port", port),
		)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Log.Fatal("gateway启动失败", zap.Error(err))
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	sig := <-quit
	logger.Log.Info("gateway 收到关闭信号", zap.String("signal", sig.String()))

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		logger.Log.Error("gateway 关闭失败", zap.Error(err))
		return
	}
	logger.Log.Info("gateway 关闭完成")

}

func newReverseProxy(url *url.URL) *httputil.ReverseProxy {
	proxy := httputil.NewSingleHostReverseProxy(url)
	return proxy
}
