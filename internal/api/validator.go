package api

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/shopspring/decimal"
)

var (
	maxQuantity   = decimal.NewFromInt(1_000_000)
	maxPrice      = decimal.NewFromInt(100_000_000)
	symbolPattern = regexp.MustCompile(`^[A-Z]{2,10}-[A-Z]{2,10}$`)
)

func validatePlaceOrderRequest(req *placeOrderRequest) error {
	// 验证symbol格式
	symbol := strings.ToUpper(req.Symbol)
	if !symbolPattern.MatchString(symbol) {
		return fmt.Errorf("交易对格式无效")
	}

	if req.Quantity.LessThanOrEqual(decimal.Zero) {
		return fmt.Errorf("下单数量必须大于 0")
	}
	if req.Quantity.GreaterThan(maxQuantity) {
		return fmt.Errorf("下单数量超出上限 (最大 %s)", maxQuantity.String())
	}

	if req.Quantity.Exponent() < -8 {
		return fmt.Errorf("下单数量精度过高")
	}

	if req.Price.LessThanOrEqual(decimal.Zero) {
		return fmt.Errorf("下单价格必须大于0")
	}
	if req.Price.GreaterThan(maxPrice) {
		return fmt.Errorf("挂单价格超出上限(最大 %s)", maxPrice.String())
	}
	if req.Price.Exponent() < -8 {
		return fmt.Errorf("挂单价格精度过高")
	}

	return nil
}
