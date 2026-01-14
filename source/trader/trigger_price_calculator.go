package trader

import (
	"fmt"
	"nofx/logger"
	"nofx/store"
)

// TriggerPriceCalculator 触发价格计算器
type TriggerPriceCalculator struct {
	config *store.TriggerPriceStrategy
}

// NewTriggerPriceCalculator 创建触发价格计算器
func NewTriggerPriceCalculator(config *store.TriggerPriceStrategy) *TriggerPriceCalculator {
	if config == nil {
		// 默认使用摆动交易配置
		config = store.GetDefaultTriggerPriceConfig("swing")
	}
	return &TriggerPriceCalculator{
		config: config,
	}
}

// Calculate 计算触发价格（基于当前价格和止损）
func (c *TriggerPriceCalculator) Calculate(
	currentPrice float64,
	action string,
	stopLoss float64,
) float64 {
	if currentPrice <= 0 {
		logger.Warnf("⚠️ Invalid current price: %.2f, using fallback", currentPrice)
		return c.fallbackTriggerPrice(action, stopLoss)
	}

	var triggerPrice float64

	switch action {
	case "open_long":
		triggerPrice = c.calculateOpenLong(currentPrice, stopLoss)
	case "open_short":
		triggerPrice = c.calculateOpenShort(currentPrice, stopLoss)
	default:
		logger.Warnf("⚠️ Unknown action: %s, using current price", action)
		triggerPrice = currentPrice
	}

	// 验证触发价格的合理性
	triggerPrice = c.validateTriggerPrice(triggerPrice, currentPrice, action)

	logger.Infof("🔧 Trigger Price: %s | Style: %s | Current: %.2f | Trigger: %.2f | Diff: %.2f%%",
		action,
		c.config.Style,
		currentPrice,
		triggerPrice,
		((triggerPrice-currentPrice)/currentPrice)*100,
	)

	return triggerPrice
}

// CalculateWithStopLoss 计算触发价格（基于止盈止损，确保在中间）
func (c *TriggerPriceCalculator) CalculateWithStopLoss(
	currentPrice float64,
	action string,
	stopLoss float64,
	takeProfit float64,
) float64 {
	if currentPrice <= 0 {
		logger.Warnf("⚠️ Invalid current price: %.2f, using fallback", currentPrice)
		return c.fallbackTriggerPrice(action, stopLoss)
	}

	if stopLoss <= 0 || takeProfit <= 0 {
		logger.Warnf("⚠️ Invalid stop loss or take profit, falling back to basic calculation")
		return c.Calculate(currentPrice, action, stopLoss)
	}

	var triggerPrice float64

	switch action {
	case "open_long":
		triggerPrice = c.calculateOpenLongWithTP(currentPrice, stopLoss, takeProfit)
	case "open_short":
		triggerPrice = c.calculateOpenShortWithTP(currentPrice, stopLoss, takeProfit)
	default:
		logger.Warnf("⚠️ Unknown action: %s, using current price", action)
		triggerPrice = currentPrice
	}

	// 验证触发价格必须在止盈止损之间
	triggerPrice = c.validateTriggerPriceInRange(triggerPrice, stopLoss, takeProfit, action)

	logger.Infof("🔧 Trigger Price (with TP): %s | Style: %s | Current: %.2f | SL: %.2f | TP: %.2f | Trigger: %.2f",
		action,
		c.config.Style,
		currentPrice,
		stopLoss,
		takeProfit,
		triggerPrice,
	)

	return triggerPrice
}

// calculateOpenLong 计算开多触发价格
func (c *TriggerPriceCalculator) calculateOpenLong(currentPrice, stopLoss float64) float64 {
	mode := c.config.Mode

	switch mode {
	case "current_price":
		return currentPrice

	case "pullback":
		return c.calculatePullback(currentPrice, stopLoss, "open_long")

	case "breakout":
		return c.calculateBreakout(currentPrice, "open_long")

	default:
		logger.Warnf("⚠️ Unknown mode: %s, using current price", mode)
		return currentPrice
	}
}

// calculateOpenShort 计算开空触发价格
func (c *TriggerPriceCalculator) calculateOpenShort(currentPrice, stopLoss float64) float64 {
	mode := c.config.Mode

	switch mode {
	case "current_price":
		return currentPrice

	case "pullback":
		return c.calculatePullback(currentPrice, stopLoss, "open_short")

	case "breakout":
		return c.calculateBreakout(currentPrice, "open_short")

	default:
		logger.Warnf("⚠️ Unknown mode: %s, using current price", mode)
		return currentPrice
	}
}

// calculatePullback 计算回调触发价格
func (c *TriggerPriceCalculator) calculatePullback(
	currentPrice, stopLoss float64,
	action string,
) float64 {
	if action == "open_long" {
		// 开多：等待回调
		// 使用回调比例
		pullback := currentPrice * c.config.PullbackRatio
		triggerPrice := currentPrice - pullback

		// 添加额外缓冲
		if c.config.ExtraBuffer > 0 {
			buffer := currentPrice * c.config.ExtraBuffer
			triggerPrice -= buffer
		}

		return triggerPrice
	} else {
		// 开空：等待反弹
		// 使用回调比例
		pullback := currentPrice * c.config.PullbackRatio
		triggerPrice := currentPrice + pullback

		// 添加额外缓冲
		if c.config.ExtraBuffer > 0 {
			buffer := currentPrice * c.config.ExtraBuffer
			triggerPrice += buffer
		}

		return triggerPrice
	}
}

// calculateBreakout 计算突破触发价格
func (c *TriggerPriceCalculator) calculateBreakout(currentPrice float64, action string) float64 {
	if action == "open_long" {
		// 开多：等待突破
		threshold := currentPrice * c.config.BreakoutRatio
		return currentPrice + threshold
	} else {
		// 开空：等待跌破
		threshold := currentPrice * c.config.BreakoutRatio
		return currentPrice - threshold
	}
}

// calculateOpenLongWithTP 计算开多触发价格（基于止盈止损）
func (c *TriggerPriceCalculator) calculateOpenLongWithTP(
	currentPrice, stopLoss, takeProfit float64,
) float64 {
	// 对于开多单，触发价格应该在当前价格下方（等待回调）
	// 但必须在止损上方，且在止盈下方

	// 计算止盈止损距离
	slDistance := currentPrice - stopLoss

	// 根据交易风格调整触发价格位置
	var triggerPrice float64

	switch c.config.Style {
	case "scalp":
		// 剥头皮：非常接近当前价格，但略低于当前价格等待回调
		// 目标：在当前价格下方 1-2%
		triggerPrice = currentPrice * 0.985

	case "short_term":
		// 短线：在当前价格下方等待回调
		// 在止盈止损中间，但确保低于当前价格
		midpoint := (stopLoss + takeProfit) / 2
		if midpoint < currentPrice {
			triggerPrice = midpoint
		} else {
			triggerPrice = currentPrice * 0.98
		}

	case "swing":
		// 摆动：在当前价格下方等待回调
		// 在止盈止损中间，但确保低于当前价格
		midpoint := (stopLoss + takeProfit) / 2
		if midpoint < currentPrice {
			triggerPrice = midpoint
		} else {
			triggerPrice = currentPrice * 0.97
		}

	case "long_term":
		// 长线：在当前价格下方等待回调
		// 在止盈止损中间偏止盈（但仍在当前价格下方）
		midpoint := (stopLoss + takeProfit) / 2
		if midpoint < currentPrice {
			triggerPrice = midpoint
		} else {
			triggerPrice = currentPrice * 0.95
		}

	default:
		// 默认：使用当前价格下方2%
		triggerPrice = currentPrice * 0.98
	}

	// 确保触发价格在合理范围内
	// 1. 必须在止损上方
	if triggerPrice <= stopLoss {
		triggerPrice = stopLoss + (slDistance * 0.1) // 止损上方10%距离
	}

	// 2. 必须在止盈下方
	if triggerPrice >= takeProfit {
		triggerPrice = takeProfit * 0.95
	}

	// 3. 必须低于当前价格（开多单需要回调）
	if triggerPrice >= currentPrice {
		triggerPrice = currentPrice * 0.98 // 强制低于当前价格2%
	}

	return triggerPrice
}

// calculateOpenShortWithTP 计算开空触发价格（基于止盈止损）
func (c *TriggerPriceCalculator) calculateOpenShortWithTP(
	currentPrice, stopLoss, takeProfit float64,
) float64 {
	// 对于开空单，触发价格应该在当前价格上方（等待反弹）
	// 但必须在止损下方，且在止盈上方

	// 计算止盈止损距离
	slDistance := stopLoss - currentPrice

	// 根据交易风格调整触发价格位置
	var triggerPrice float64

	switch c.config.Style {
	case "scalp":
		// 剥头皮：非常接近当前价格，但略高于当前价格等待反弹
		// 目标：在当前价格上方 1-2%
		triggerPrice = currentPrice * 1.015

	case "short_term":
		// 短线：在当前价格上方等待反弹
		// 在止盈止损中间，但确保高于当前价格
		midpoint := (stopLoss + takeProfit) / 2
		if midpoint > currentPrice {
			triggerPrice = midpoint
		} else {
			triggerPrice = currentPrice * 1.02
		}

	case "swing":
		// 摆动：在当前价格上方等待反弹
		// 在止盈止损中间，但确保高于当前价格
		midpoint := (stopLoss + takeProfit) / 2
		if midpoint > currentPrice {
			triggerPrice = midpoint
		} else {
			triggerPrice = currentPrice * 1.03
		}

	case "long_term":
		// 长线：在当前价格上方等待反弹
		// 在止盈止损中间偏止盈（但仍在当前价格上方）
		midpoint := (stopLoss + takeProfit) / 2
		if midpoint > currentPrice {
			triggerPrice = midpoint
		} else {
			triggerPrice = currentPrice * 1.05
		}

	default:
		// 默认：使用当前价格上方2%
		triggerPrice = currentPrice * 1.02
	}

	// 确保触发价格在合理范围内
	// 1. 必须在止损下方
	if triggerPrice >= stopLoss {
		triggerPrice = stopLoss - (slDistance * 0.1) // 止损下方10%距离
	}

	// 2. 必须在止盈上方
	if triggerPrice <= takeProfit {
		triggerPrice = takeProfit * 1.05
	}

	// 3. 必须高于当前价格（开空单需要反弹）
	if triggerPrice <= currentPrice {
		triggerPrice = currentPrice * 1.02 // 强制高于当前价格2%
	}

	return triggerPrice
}

// validateTriggerPriceInRange 验证触发价格必须在止盈止损之间
func (c *TriggerPriceCalculator) validateTriggerPriceInRange(
	triggerPrice, stopLoss, takeProfit float64,
	action string,
) float64 {
	// 开多：stopLoss < triggerPrice < takeProfit
	// 开空：takeProfit < triggerPrice < stopLoss
	if action == "open_long" {
		if triggerPrice <= stopLoss {
			logger.Warnf("⚠️ Trigger price %.2f <= SL %.2f, adjusting to midpoint", triggerPrice, stopLoss)
			return (stopLoss + takeProfit) / 2
		}
		if triggerPrice >= takeProfit {
			logger.Warnf("⚠️ Trigger price %.2f >= TP %.2f, adjusting to midpoint", triggerPrice, takeProfit)
			return (stopLoss + takeProfit) / 2
		}
	} else {
		if triggerPrice >= stopLoss {
			logger.Warnf("⚠️ Trigger price %.2f >= SL %.2f, adjusting to midpoint", triggerPrice, stopLoss)
			return (stopLoss + takeProfit) / 2
		}
		if triggerPrice <= takeProfit {
			logger.Warnf("⚠️ Trigger price %.2f <= TP %.2f, adjusting to midpoint", triggerPrice, takeProfit)
			return (stopLoss + takeProfit) / 2
		}
	}

	return triggerPrice
}

// validateTriggerPrice 验证触发价格的合理性
func (c *TriggerPriceCalculator) validateTriggerPrice(
	triggerPrice, currentPrice float64,
	action string,
) float64 {
	// 防止触发价格过于离谱
	maxDiff := 0.5 // 最大50%差异
	diff := (triggerPrice - currentPrice) / currentPrice

	if diff > maxDiff {
		logger.Warnf("⚠️ Trigger price too high (%.2f%%), using current price", diff*100)
		return currentPrice
	}

	if diff < -maxDiff {
		logger.Warnf("⚠️ Trigger price too low (%.2f%%), using current price", diff*100)
		return currentPrice
	}

	return triggerPrice
}

// fallbackTriggerPrice 降级方案
func (c *TriggerPriceCalculator) fallbackTriggerPrice(action string, stopLoss float64) float64 {
	// 如果无法获取当前价格，使用止损价作为触发价
	if stopLoss > 0 {
		return stopLoss
	}

	// 如果连止损价都没有，使用一个保守的默认值
	logger.Warnf("⚠️ No valid price data available, using conservative default")
	return 100.0
}

// GetTriggerMode 获取触发模式描述
func (c *TriggerPriceCalculator) GetTriggerMode(action string) string {
	return fmt.Sprintf("%s/%s", c.config.Style, c.config.Mode)
}

// ========== 预设配置 ==========

// GetDefaultTriggerPriceConfig 获取默认触发价格配置 (移动到 store 包，这里保留兼容性)
func GetDefaultTriggerPriceConfig(style string) *store.TriggerPriceStrategy {
	return store.GetDefaultTriggerPriceConfig(style)
}
