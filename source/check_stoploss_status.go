package main

import (
	"database/sql"
	"fmt"
	"log"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

func main() {
	db, err := sql.Open("sqlite3", "data/nofx.db")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	// 查询活跃的动态止损记录
	rows, err := db.Query(`
		SELECT symbol, entry_price, current_stop_loss, take_profit, current_price, 
		       time_progression, elapsed_seconds, created_at, updated_at
		FROM adaptive_stoploss_records 
		WHERE status = 'ACTIVE' 
		ORDER BY created_at DESC
	`)
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()

	fmt.Println("📊 动态止损实时状态检查")
	fmt.Println("=" + string(make([]byte, 80)) + "=")

	hasRecords := false
	for rows.Next() {
		hasRecords = true
		var symbol string
		var entryPrice, currentStopLoss, takeProfit, currentPrice, timeProgression float64
		var elapsedSeconds int
		var createdAt, updatedAt time.Time

		err := rows.Scan(&symbol, &entryPrice, &currentStopLoss, &takeProfit, &currentPrice,
			&timeProgression, &elapsedSeconds, &createdAt, &updatedAt)
		if err != nil {
			log.Fatal(err)
		}

		// 计算当前状态
		isAtEntry := currentStopLoss == entryPrice
		profitPct := 0.0
		if entryPrice > 0 {
			profitPct = (currentPrice - entryPrice) / entryPrice * 100
		}

		fmt.Printf("\n币种: %s\n", symbol)
		fmt.Printf("  入场价: %.6f | 当前价: %.6f | 止损价: %.6f | 止盈价: %.6f\n",
			entryPrice, currentPrice, currentStopLoss, takeProfit)
		fmt.Printf("  盈利: %.2f%% | 时间进度: %.1f%% | 已过: %d秒\n",
			profitPct, timeProgression*100, elapsedSeconds)
		fmt.Printf("  止损在买入价: %v | 更新时间: %s\n", 
			isAtEntry, updatedAt.Format("15:04:05"))

		if isAtEntry {
			fmt.Println("  ✅ 正常：止损已移动到买入价，保护本金")
		} else {
			fmt.Println("  🔄 动态：止损正在向买入价移动")
		}
	}

	if !hasRecords {
		fmt.Println("\n暂无活跃的动态止损记录")
	}

	// 检查系统状态
	var count int
	db.QueryRow("SELECT COUNT(*) FROM adaptive_stoploss_records WHERE status = 'ACTIVE'").Scan(&count)
	fmt.Printf("\n📈 系统状态: %d 个活跃的动态止损\n", count)
}
