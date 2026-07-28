package main

import (
	"net/url"

	"atlas-trading-infrastructure/internal/infrastructure/logger"

	"net/http/httputil"

	"go.uber.org/zap"

	"github.com/gin-gonic/gin"
)

func main() {

	defer logger.Sync()

	port := 8100

	orderServiceURL := "http://localhost:8103"

	orderURL, err := url.Parse(orderServiceURL)
	if err != nil {
		logger.Log.Fatal("order service url 格式错误", zap.Error(err))
	}

	orderProxy := newReverseProxy(orderURL)

	r := gin.New()
	// 注册要转发的路由

}

func newReverseProxy(url *url.URL) *httputil.ReverseProxy {
	proxy := httputil.NewSingleHostReverseProxy(url)
	return proxy
}
