package api

import (
	"atlas-trading-infrastructure/internal/domain"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type placeOrderRequest struct {
	UserID   string          `json:"user_id" binding:"required"`
	Symbol   string          `json:"symbol" binding:"required"`
	Side     string          `json:"side" binding:"required,oneof=BUY SELL"`
	Price    decimal.Decimal `json:"price"`
	Quantity decimal.Decimal `json:"quantity" binding:"required"`
}

func (h *Handler) PlaceOrder(c *gin.Context) {
	var req placeOrderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 检验Symbol、Quantity、Price合法
	if err := validatePlaceOrderRequest(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID, err := uuid.Parse(req.UserID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user_id"})
		return
	}

	side, err := domain.SideFromString(req.Side)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	order := &domain.Order{
		UserID:   userID,
		Symbol:   req.Symbol,
		Side:     side,
		Price:    req.Price,
		Quantity: req.Quantity,
	}

	if err := h.orderSvc.PlaceOrder(c.Request.Context(), order); err != nil {
		if errors.Is(err, domain.ErrInsufficientFunds) {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}
		return
	}

	c.JSON(http.StatusAccepted, gin.H{
		"order_id": order.ID,
	})
}

func (h *Handler) BatchPlaceOrders(c *gin.Context) {
	var reqs []placeOrderRequest
	if err := c.ShouldBindJSON(&reqs); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if len(reqs) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "empty batch"})
		return
	}

	orders := make([]*domain.Order, 0, len(reqs))
	for _, req := range reqs {
		if err := validatePlaceOrderRequest(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		userID, err := uuid.Parse(req.UserID)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user_id"})
			return
		}
		side, err := domain.SideFromString(req.Side)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		orders = append(orders, &domain.Order{
			UserID:   userID,
			Symbol:   req.Symbol,
			Side:     side,
			Price:    req.Price,
			Quantity: req.Quantity,
		})
	}

	if err := h.orderSvc.BatchPlaceOrders(c.Request.Context(), orders); err != nil {
		if errors.Is(err, domain.ErrInsufficientFunds) {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}
		return
	}

	c.JSON(http.StatusAccepted, gin.H{
		"message": "batch accepted",
		"count":   len(orders),
	})
}

func (h *Handler) CancelOrder(c *gin.Context) {
	idStr := c.Param("id")
	orderID, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的订单 ID"})
		return
	}

	userIDStr := c.GetHeader("X-User-ID")
	if userIDStr == "" {
		userIDStr = c.Query("user_id") // 请求头没有就看查询参数
	}
	if userIDStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "缺少 user_id"})
		return
	}

	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的 user_id"})
		return
	}

	if err := h.orderSvc.CancelOrder(c.Request.Context(), orderID, userID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "订单已取消"})
}

func (h *Handler) GetOrder(c *gin.Context) {
	idStr := c.Param("id")
	orderID, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的订单 ID"})
		return
	}

	order, err := h.orderSvc.GetOrder(c.Request.Context(), orderID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "订单不存在"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"id":              order.ID,
		"user_id":         order.UserID,
		"symbol":          order.Symbol,
		"side":            domain.SideToString(order.Side),
		"price":           order.Price,
		"quantity":        order.Quantity,
		"filled_quantity": order.FilledQuantity,
		"status":          domain.StatusToString(order.Status),
	})
}
