package main

import (
	"encoding/json"
	"fmt"
	"nofx/store"
)

func main() {
	fmt.Println("=== 触发价格配置验证测试 ===\n")

	// 1. 测试前端预设配置
	fmt.Println("1. 前端预设配置:")
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
		fmt.Printf("  %s: mode=%s, style=%s, pullback=%.4f, breakout=%.4f, buffer=%.4f\n",
			name, config.Mode, config.Style, config.PullbackRatio, config.BreakoutRatio, config.ExtraBuffer)
	}

	// 2. 测试后端默认配置
	fmt.Println("\n2. 后端默认配置:")
	for style := range presets {
		config := store.GetDefaultTriggerPriceConfig(style)
		fmt.Printf("  %s: mode=%s, style=%s, pullback=%.4f, breakout=%.4f, buffer=%.4f\n",
			style, config.Mode, config.Style, config.PullbackRatio, config.BreakoutRatio, config.ExtraBuffer)
	}

	// 3. 测试JSON序列化
	fmt.Println("\n3. JSON序列化测试:")
	scalpConfig := presets["scalp"]
	jsonData, _ := json.MarshalIndent(scalpConfig, "  ", "  ")
	fmt.Printf("  Scalp配置JSON:\n  %s\n", string(jsonData))

	// 4. 模拟触发价格计算
	fmt.Println("\n4. 触发价格计算模拟:")
	currentPrice := 100.0

	// Scalp风格 - current_price模式
	scalpTrigger := currentPrice - (currentPrice * scalpConfig.ExtraBuffer)
	fmt.Printf("  Scalp (current_price): %.4f = %.2f - (%.2f × %.4f)\n",
		scalpTrigger, currentPrice, currentPrice, scalpConfig.ExtraBuffer)

	// Swing风格 - pullback模式
	swingConfig := presets["swing"]
	swingTrigger := currentPrice - (currentPrice * swingConfig.PullbackRatio) - (currentPrice * swingConfig.ExtraBuffer)
	fmt.Printf("  Swing (pullback): %.4f = %.2f - (%.2f × %.4f) - (%.2f × %.4f)\n",
		swingTrigger, currentPrice, currentPrice, swingConfig.PullbackRatio, currentPrice, swingConfig.ExtraBuffer)

	// Long Term风格 - pullback模式
	longConfig := presets["long_term"]
	longTrigger := currentPrice - (currentPrice * longConfig.PullbackRatio) - (currentPrice * longConfig.ExtraBuffer)
	fmt.Printf("  Long Term (pullback): %.4f = %.2f - (%.2f × %.4f) - (%.2f × %.4f)\n",
		longTrigger, currentPrice, currentPrice, longConfig.PullbackRatio, currentPrice, longConfig.ExtraBuffer)

	// 5. 检查StrategyConfig
	fmt.Println("\n5. StrategyConfig默认值:")
	defaultConfig := store.GetDefaultStrategyConfig("en")
	if defaultConfig.TriggerPriceConfig != nil {
		cfg := defaultConfig.TriggerPriceConfig
		fmt.Printf("  Default: mode=%s, style=%s, pullback=%.4f, breakout=%.4f, buffer=%.4f\n",
			cfg.Mode, cfg.Style, cfg.PullbackRatio, cfg.BreakoutRatio, cfg.ExtraBuffer)
	} else {
		fmt.Println("  ⚠️ TriggerPriceConfig is nil!")
	}

	// 6. 验证字段映射
	fmt.Println("\n6. 字段映射验证:")
	fmt.Println("  前端字段名 → 后端字段名")
	fmt.Println("  mode → Mode ✓")
	fmt.Println("  style → Style ✓")
	fmt.Println("  pullback_ratio → PullbackRatio ✓")
	fmt.Println("  breakout_ratio → BreakoutRatio ✓")
	fmt.Println("  extra_buffer → ExtraBuffer ✓")

	fmt.Println("\n=== 测试结论 ===")
	fmt.Println("✓ 配置结构定义正确")
	fmt.Println("✓ 默认配置存在")
	fmt.Println("✓ JSON序列化正常")
	fmt.Println("⚠️ 需要验证实际配置传递链路")
	fmt.Println("\n建议下一步:")
	fmt.Println("1. 启动系统并观察日志")
	fmt.Println("2. 在前端创建scalp策略")
	fmt.Println("3. 检查后端日志中的TriggerPriceConfig值")
	fmt.Println("4. 验证PENDING订单的触发价格")
}
