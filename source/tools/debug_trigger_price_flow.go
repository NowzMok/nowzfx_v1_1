package main

import (
	"encoding/json"
	"fmt"
	"nofx/store"
)

// DebugTriggerPriceFlow 调试触发价格配置的完整流程
func DebugTriggerPriceFlow() {
	fmt.Println("=== 触发价格配置流程调试 ===\n")

	// 1. 检查前端预设配置
	fmt.Println("1. 前端预设配置 (TriggerPriceEditor.tsx):")
	presets := map[string]store.TriggerPriceStrategy{
		"long_term": {
			Mode:          "pullback",
			Style:         "long_term",
			PullbackRatio: 0.05,
			BreakoutRatio: 0.03,
			ExtraBuffer:   0.01,
		},
		"short_term": {
			Mode:          "pullback",
			Style:         "short_term",
			PullbackRatio: 0.01,
			BreakoutRatio: 0.005,
			ExtraBuffer:   0.002,
		},
		"swing": {
			Mode:          "pullback",
			Style:         "swing",
			PullbackRatio: 0.02,
			BreakoutRatio: 0.01,
			ExtraBuffer:   0.005,
		},
		"scalp": {
			Mode:          "current_price",
			Style:         "scalp",
			PullbackRatio: 0.005,
			BreakoutRatio: 0.003,
			ExtraBuffer:   0.001,
		},
	}

	for name, config := range presets {
		fmt.Printf("  %s: mode=%s, style=%s, pullback=%.3f, breakout=%.3f, buffer=%.3f\n",
			name, config.Mode, config.Style, config.PullbackRatio, config.BreakoutRatio, config.ExtraBuffer)
	}

	// 2. 检查后端默认配置
	fmt.Println("\n2. 后端默认配置 (store/strategy.go):")
	defaultConfigs := []string{"long_term", "short_term", "swing", "scalp"}
	for _, style := range defaultConfigs {
		config := store.GetDefaultTriggerPriceConfig(style)
		fmt.Printf("  %s: mode=%s, style=%s, pullback=%.3f, breakout=%.3f, buffer=%.3f\n",
			style, config.Mode, config.Style, config.PullbackRatio, config.BreakoutRatio, config.ExtraBuffer)
	}

	// 3. 检查StrategyConfig结构
	fmt.Println("\n3. StrategyConfig结构:")
	strategyConfig := store.GetDefaultStrategyConfig("en")
	if strategyConfig.TriggerPriceConfig != nil {
		cfg := strategyConfig.TriggerPriceConfig
		fmt.Printf("  Default: mode=%s, style=%s, pullback=%.3f, breakout=%.3f, buffer=%.3f\n",
			cfg.Mode, cfg.Style, cfg.PullbackRatio, cfg.BreakoutRatio, cfg.ExtraBuffer)
	} else {
		fmt.Println("  TriggerPriceConfig is nil!")
	}

	// 4. 模拟JSON序列化/反序列化
	fmt.Println("\n4. JSON序列化测试:")
	jsonData, _ := json.Marshal(strategyConfig.TriggerPriceConfig)
	fmt.Printf("  JSON: %s\n", string(jsonData))

	var parsed store.TriggerPriceStrategy
	json.Unmarshal(jsonData, &parsed)
	fmt.Printf("  Parsed: mode=%s, style=%s, pullback=%.3f, breakout=%.3f, buffer=%.3f\n",
		parsed.Mode, parsed.Style, parsed.PullbackRatio, parsed.BreakoutRatio, parsed.ExtraBuffer)

	// 5. 检查字段名称一致性
	fmt.Println("\n5. 字段名称一致性检查:")
	fmt.Println("  前端: pullback_ratio, breakout_ratio, extra_buffer")
	fmt.Println("  后端: PullbackRatio, BreakoutRatio, ExtraBuffer")
	fmt.Println("  JSON标签: pullback_ratio, breakout_ratio, extra_buffer ✓")

	// 6. 检查配置读取路径
	fmt.Println("\n6. 配置读取路径:")
	fmt.Println("  TraderManager.addTraderFromStore() → Strategy.ParseConfig() → AutoTraderConfig.StrategyConfig")
	fmt.Println("  AutoTrader.SaveAnalysisAndCreatePendingOrders() → at.config.StrategyConfig.TriggerPriceConfig")
	fmt.Println("  如果为nil → store.GetDefaultTriggerPriceConfig(\"swing\")")

	// 7. 检查触发价格计算
	fmt.Println("\n7. 触发价格计算公式:")
	fmt.Println("  Pullback模式: trigger = current - (current × pullback) - (current × buffer)")
	fmt.Println("  Breakout模式: trigger = current + (current × breakout) + (current × buffer)")
	fmt.Println("  CurrentPrice模式: trigger = current - (current × buffer)")

	// 8. 检查风格判断逻辑
	fmt.Println("\n8. 风格判断逻辑:")
	fmt.Println("  问题: 18.8575被判断为Long Term，但用户选择的是scalp")
	fmt.Println("  可能原因:")
	fmt.Println("    a) 策略配置未正确保存到数据库")
	fmt.Println("    b) 配置读取时被覆盖为swing")
	fmt.Println("    c) TriggerPriceConfig为nil，回退到默认swing")
	fmt.Println("    d) 前端未正确发送TriggerPriceConfig到后端")

	fmt.Println("\n=== 调试建议 ===")
	fmt.Println("1. 在前端保存策略时，检查Network请求体是否包含trigger_price_config")
	fmt.Println("2. 在后端handleUpdateStrategy添加日志，打印接收到的TriggerPriceConfig")
	fmt.Println("3. 在addTraderFromStore添加日志，打印strategyConfig.TriggerPriceConfig")
	fmt.Println("4. 在auto_trader_analysis.go的SaveAnalysisAndCreatePendingOrders添加日志")
	fmt.Println("5. 运行系统，观察日志输出，定位配置丢失的位置")
}
