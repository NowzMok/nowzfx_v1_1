package main

import (
	"fmt"
	"nofx/store"
	"nofx/tools"
)

func main() {
	fmt.Println("=== riverusdt 触发价格计算分析 ===\n")

	// 测试不同场景
	scenarios := []struct {
		name         string
		currentPrice float64
		action       string
		config       *store.TriggerPriceStrategy
	}{
		{
			name:         "Swing风格 - 当前价格20.0",
			currentPrice: 20.0,
			action:       "open_long",
			config:       store.GetDefaultTriggerPriceConfig("swing"),
		},
		{
			name:         "Long Term风格 - 当前价格20.0",
			currentPrice: 20.0,
			action:       "open_long",
			config:       store.GetDefaultTriggerPriceConfig("long_term"),
		},
		{
			name:         "Short Term风格 - 当前价格20.0",
			currentPrice: 20.0,
			action:       "open_long",
			config:       store.GetDefaultTriggerPriceConfig("short_term"),
		},
		{
			name:         "Scalp风格 - 当前价格20.0",
			currentPrice: 20.0,
			action:       "open_long",
			config:       store.GetDefaultTriggerPriceConfig("scalp"),
		},
	}

	for _, scenario := range scenarios {
		fmt.Printf("场景: %s\n", scenario.name)
		tools.DebugTriggerPrice(scenario.currentPrice, scenario.action, scenario.config)
		fmt.Println()
	}

	// 反推计算：已知触发价格18.8575，反推当前价格
	fmt.Println("=== 反推计算：触发价格18.8575 ===\n")

	styles := []string{"swing", "long_term", "short_term", "scalp"}
	for _, style := range styles {
		config := store.GetDefaultTriggerPriceConfig(style)
		fmt.Printf("风格: %s\n", style)
		tools.ReverseCalculateTriggerPrice(18.8575, "open_long", config)
		fmt.Println()
	}
}
