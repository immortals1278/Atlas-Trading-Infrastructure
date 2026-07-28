package api

import (
	"atlas-trading-infrastructure/internal/order"
)

type Handler struct {
	orderService order.Service
}

func NewHandler(orderSvc order.Service) *Handler {
	return &Handler{
		orderService: orderSvc,
	}
}

//根据功能注册路由函数
