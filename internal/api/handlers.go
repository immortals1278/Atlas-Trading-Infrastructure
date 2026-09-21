package api

import (
	"atlas-trading-infrastructure/internal/order"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	orderSvc order.OrderService
}

func NewHandler(orderSvc *order.Service) *Handler {
	return &Handler{
		orderSvc: orderSvc,
	}
}

// 根据功能注册路由函数
func (h *Handler) RegisterRoutes(router gin.IRouter) {

	private := router.Group("/")
	{
		private.GET("/orders/:id", h.GetOrder)
		private.DELETE("/orders/:id", h.CancelOrder)
		private.GET("/accounts", h.GetBalances)
	}

	orders := router.Group("/")
	{
		orders.POST("/orders", h.PlaceOrder)
		orders.POST("/orders/batch", h.BatchPlaceOrders)
	}
}
