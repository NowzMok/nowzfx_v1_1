package main

import (
	"fmt"
	"nofx/store"
	"nofx/trader"
)

func main() {
	fmt.Println("=== 触发价格修复验证测试 ===\n")

	// 测试场景：当前价格100，止损95，止盈115
	currentPrice := 100.0
	stopLoss := 95.0
	takeProfit := 115.0

	styles := []string{"scalp", "short_term", "swing", "long_term"}
	modes := []string{"pullback", "breakout"}

	for _, style := range styles {
		for _, mode := range modes {
			config := &store.TriggerPriceStrategy{
				Mode:          mode,
				Style:         style,
				PullbackRatio: 0.02,
				BreakoutRatio: 0.01,
				ExtraBuffer:   0.005,
			}

			calculator := trader.NewTriggerPriceCalculator(config)

			// 使用旧方法（仅基于当前价格）
			triggerOld := calculator.Calculate(currentPrice, "open_long", stopLoss)

			// 使用新方法（基于止盈止损）
			triggerNew := calculator.CalculateWithStopLoss(currentPrice, "open_long", stopLoss, takeProfit)

			fmt.Printf("风格: %-10s 模式: %-8s\n", style, mode)
			fmt.Printf("  当前价格: %.2f | 止损: %.2f | 止盈: %.2f\n", currentPrice, stopLoss, takeProfit)
			fmt.Printf("  旧触发价: %.2f | 距离当前: %.2f | 在范围内: %v\n",
				triggerOld, currentPrice-triggerOld, triggerOld > stopLoss && triggerOld < takeProfit)
			fmt.Printf("  新触发价: %.2f | 距离当前: %.2f | 在范围内: %v\n",
				triggerNew, currentPrice-triggerNew, triggerNew > stopLoss && triggerNew < takeProfit)

			// 验证风险回报比
			risk := currentPrice - stopLoss
			reward := takeProfit - currentPrice
			triggerDistance := currentPrice - triggerNew
			rrRatio := (reward / risk) * (triggerDistance / risk)

			fmt.Printf("  风险回报比: %.2f | 触发距离: %.2f\n", rrRatio, triggerDistance)
			fmt.Println()
		}
	}

	fmt.Println("=== 测试完成 ===")
}
