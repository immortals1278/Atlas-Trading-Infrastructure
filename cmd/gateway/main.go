package main

import (
	"net/url"

	"atlas-trading-infrastructure/internal/infrastructure/logger"

	"go.uber.org/zap"
)

func main() {

	defer logger.Sync()

	port := 8100

	orderServiceURL := "http://localhost:8103"

	orderURL, err := url.Parse(orderServiceURL)
	if err != nil {
		logger.Log.Fatal("order service url 格式错误", zap.Error(err))
	}

}
