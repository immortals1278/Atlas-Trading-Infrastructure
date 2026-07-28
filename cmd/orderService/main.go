package main


// 初始化db
// 初始化handlers+执行handlers注册路由
import (
	"atlas-trading-infrastructure/internal/api"
)

handler := api.NewHandler(svc)