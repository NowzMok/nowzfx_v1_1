package tools

import (
	"fmt"
	"nofx/store"
	"nofx/trader"
)

// DebugTriggerPrice 计算并显示触发价格的详细计算过程
func DebugTriggerPrice(currentPrice float64, action string, config *store.TriggerPriceStrategy) {
	if config == nil {
		config = store.GetDefaultTriggerPriceConfig("swing")
	}

	calculator := trader.NewTriggerPriceCalculator(config)

	fmt.Printf("=== 触发价格计算详情 ===\n")
	fmt.Printf("当前价格: %.4f\n", currentPrice)
	fmt.Printf("交易动作: %s\n", action)
	fmt.Printf("交易风格: %s\n", config.Style)
	fmt.Printf("触发模式: %s\n", config.Mode)
	fmt.Printf("回调比例: %.4f (%.2f%%)\n", config.PullbackRatio, config.PullbackRatio*100)
	fmt.Printf("突破比例: %.4f (%.2f%%)\n", config.BreakoutRatio, config.BreakoutRatio*100)
	fmt.Printf("额外缓冲: %.4f (%.2f%%)\n", config.ExtraBuffer, config.ExtraBuffer*100)
	fmt.Printf("\n")

	// 计算触发价格
	triggerPrice := calculator.Calculate(currentPrice, action, 0)

	fmt.Printf("计算结果:\n")
	fmt.Printf("触发价格: %.4f\n", triggerPrice)
	fmt.Printf("回调幅度: %.4f%%\n", ((currentPrice-triggerPrice)/currentPrice)*100)
	fmt.Printf("\n")

	// 详细计算过程
	if config.Mode == "pullback" {
		if action == "open_long" {
			pullbackAmount := currentPrice * config.PullbackRatio
			bufferAmount := currentPrice * config.ExtraBuffer
			fmt.Printf("详细计算:\n")
			fmt.Printf("  回调金额: %.4f = %.4f × %.4f\n", pullbackAmount, currentPrice, config.PullbackRatio)
			fmt.Printf("  缓冲金额: %.4f = %.4f × %.4f\n", bufferAmount, currentPrice, config.ExtraBuffer)
			fmt.Printf("  触发价格: %.4f = %.4f - %.4f - %.4f\n",
				triggerPrice, currentPrice, pullbackAmount, bufferAmount)
		} else {
			pullbackAmount := currentPrice * config.PullbackRatio
			bufferAmount := currentPrice * config.ExtraBuffer
			fmt.Printf("详细计算:\n")
			fmt.Printf("  反弹金额: %.4f = %.4f × %.4f\n", pullbackAmount, currentPrice, config.PullbackRatio)
			fmt.Printf("  缓冲金额: %.4f = %.4f × %.4f\n", bufferAmount, currentPrice, config.ExtraBuffer)
			fmt.Printf("  触发价格: %.4f = %.4f + %.4f + %.4f\n",
				triggerPrice, currentPrice, pullbackAmount, bufferAmount)
		}
	} else if config.Mode == "breakout" {
		if action == "open_long" {
			threshold := currentPrice * config.BreakoutRatio
			fmt.Printf("详细计算:\n")
			fmt.Printf("  突破阈值: %.4f = %.4f × %.4f\n", threshold, currentPrice, config.BreakoutRatio)
			fmt.Printf("  触发价格: %.4f = %.4f + %.4f\n", triggerPrice, currentPrice, threshold)
		} else {
			threshold := currentPrice * config.BreakoutRatio
			fmt.Printf("详细计算:\n")
			fmt.Printf("  跌破阈值: %.4f = %.4f × %.4f\n", threshold, currentPrice, config.BreakoutRatio)
			fmt.Printf("  触发价格: %.4f = %.4f - %.4f\n", triggerPrice, currentPrice, threshold)
		}
	}
}

// ReverseCalculateTriggerPrice 根据目标触发价格反推当前价格和参数
func ReverseCalculateTriggerPrice(targetTriggerPrice float64, action string, config *store.TriggerPriceStrategy) {
	if config == nil {
		config = store.GetDefaultTriggerPriceConfig("swing")
	}

	fmt.Printf("=== 反推计算分析 ===\n")
	fmt.Printf("目标触发价格: %.4f\n", targetTriggerPrice)
	fmt.Printf("交易动作: %s\n", action)
	fmt.Printf("交易风格: %s\n", config.Style)
	fmt.Printf("\n")

	if config.Mode == "pullback" && action == "open_long" {
		// 触发价格 = 当前价格 × (1 - 回调比例 - 额外缓冲)
		// 当前价格 = 触发价格 / (1 - 回调比例 - 额外缓冲)
		totalRatio := config.PullbackRatio + config.ExtraBuffer
		currentPrice := targetTriggerPrice / (1 - totalRatio)

		fmt.Printf("反推结果:\n")
		fmt.Printf("当前价格 ≈ %.4f\n", currentPrice)
		fmt.Printf("总回调比例: %.4f (%.2f%%)\n", totalRatio, totalRatio*100)
		fmt.Printf("验证: %.4f × (1 - %.4f) = %.4f\n",
			currentPrice, totalRatio, currentPrice*(1-totalRatio))

		// 计算实际回调幅度
		actualPullback := ((currentPrice - targetTriggerPrice) / currentPrice) * 100
		fmt.Printf("实际回调幅度: %.2f%%\n", actualPullback)
	}
}
