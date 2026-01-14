package main

import (
	"encoding/json"
	"fmt"
	"nofx/store"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func main() {
	// 连接数据库
	db, err := gorm.Open(sqlite.Open("nofx-data.db"), &gorm.Config{})
	if err != nil {
		fmt.Printf("❌ Failed to connect to database: %v\n", err)
		return
	}

	// 获取策略存储
	strategyStore := store.NewStrategyStore(db)

	// 获取所有策略
	var strategies []store.Strategy
	result := db.Find(&strategies)
	if result.Error != nil {
		fmt.Printf("❌ Failed to get strategies: %v\n", result.Error)
		return
	}

	fmt.Printf("📊 Found %d strategies in database\n\n", len(strategies))

	for _, strategy := range strategies {
		fmt.Printf("Strategy: %s (ID: %s)\n", strategy.Name, strategy.ID)
		fmt.Printf("Config JSON:\n%s\n", strategy.Config)

		// 解析配置
		config, err := strategy.ParseConfig()
		if err != nil {
			fmt.Printf("❌ Failed to parse config: %v\n\n", err)
			continue
		}

		// 检查TriggerPriceConfig
		if config.TriggerPriceConfig == nil {
			fmt.Printf("⚠️  TriggerPriceConfig is nil\n\n")
			continue
		}

		// 打印TriggerPriceConfig详情
		triggerJSON, _ := json.MarshalIndent(config.TriggerPriceConfig, "  ", "  ")
		fmt.Printf("  TriggerPriceConfig:\n  %s\n\n", string(triggerJSON))
	}
}
