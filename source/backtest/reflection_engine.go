package backtest

import (
	"encoding/json"
	"fmt"
	"math"
	"nofx/logger"
	"nofx/mcp"
	"nofx/store"
	"time"
)

// ReflectionEngine AI 反思分析引擎
type ReflectionEngine struct {
	mcpClient mcp.AIClient // AI 客户端
	store     *store.Store
}

// NewReflectionEngine creates reflection engine
func NewReflectionEngine(client mcp.AIClient, store *store.Store) *ReflectionEngine {
	return &ReflectionEngine{
		mcpClient: client,
		store:     store,
	}
}

// AnalyzePeriod analyzes a trading period
func (re *ReflectionEngine) AnalyzePeriod(traderID string, startTime, endTime time.Time) (*store.ReflectionRecord, error) {
	logger.Infof("🔍 Analyzing trading period: %s to %s", startTime.Format("2006-01-02"), endTime.Format("2006-01-02"))

	// 1. 获取交易历史数据
	tradeHistory, err := re.store.Analysis().GetTradeHistoryInPeriod(traderID, startTime, endTime)
	if err != nil {
		return nil, fmt.Errorf("failed to get trade history: %w", err)
	}

	if len(tradeHistory) == 0 {
		logger.Infof("⚠️  No trades in this period, skipping reflection")
		return nil, nil
	}

	// 2. 计算统计指标
	stats := re.calculateStats(tradeHistory)
	logger.Infof("📊 Period stats: %d trades, %.2f%% success rate, PnL: %.2f USDT",
		stats.TotalTrades, stats.SuccessRate*100, stats.TotalPnL)

	// 3. 调用 AI 进行反思分析
	recommendations, err := re.getAIReflection(traderID, tradeHistory, stats)
	if err != nil {
		logger.Warnf("⚠️  AI reflection failed: %v", err)
		// 继续进行，不让 AI 失败阻止整个反思过程
	}

	// 4. 分离建议
	tradeSystemAdvice, aiLearningAdvice := re.separateAdvice(recommendations)

	// 5. 创建反思记录
	reflection := &store.ReflectionRecord{
		TraderID:           traderID,
		ReflectionTime:     time.Now().UTC(),
		PeriodStartTime:    startTime,
		PeriodEndTime:      endTime,
		TotalTrades:        stats.TotalTrades,
		SuccessfulTrades:   stats.SuccessfulTrades,
		FailedTrades:       stats.FailedTrades,
		SuccessRate:        stats.SuccessRate,
		AveragePnL:         stats.AveragePnL,
		MaxProfit:          stats.MaxProfit,
		MaxLoss:            stats.MaxLoss,
		TotalPnL:           stats.TotalPnL,
		PnLPercentage:      stats.PnLPercentage,
		SharpeRatio:        stats.SharpeRatio,
		MaxDrawdown:        stats.MaxDrawdown,
		WinLossRatio:       stats.WinLossRatio,
		ConfidenceAccuracy: stats.ConfidenceAccuracy,
		SymbolPerformance:  stats.SymbolPerformance,
		AIReflection:       recommendations,
		TradeSystemAdvice:  tradeSystemAdvice,
		AILearningAdvice:   aiLearningAdvice,
	}

	// 6. 保存反思记录
	if err := re.store.Reflection().SaveReflection(reflection); err != nil {
		return nil, fmt.Errorf("failed to save reflection: %w", err)
	}

	logger.Infof("✅ Reflection saved: %d trades analyzed, %d recommendations",
		stats.TotalTrades, len(recommendations))

	return reflection, nil
}

// ApplyRecommendations applies recommendations
func (re *ReflectionEngine) ApplyRecommendations(reflection *store.ReflectionRecord) error {
	logger.Infof("🔧 Applying recommendations from reflection %s", reflection.ID)

	// 1. 创建系统调整记录
	adjustment := &store.SystemAdjustment{
		TraderID:       reflection.TraderID,
		ReflectionID:   reflection.ID,
		AdjustmentTime: time.Now().UTC(),
		AdjustmentReason: fmt.Sprintf("Based on period analysis: %s to %s",
			reflection.PeriodStartTime.Format("2006-01-02"),
			reflection.PeriodEndTime.Format("2006-01-02")),
		Status: "PENDING",
	}

	// 2. 解析建议并应用参数调整
	for _, adviceJSON := range reflection.TradeSystemAdvice {
		var advice store.ReflectionRecommendation
		if err := unmarshalJSON(string(adviceJSON), &advice); err != nil {
			logger.Warnf("⚠️  Failed to parse advice: %v", err)
			continue
		}

		switch advice.Category {
		case "confidence":
			adjustment.ConfidenceLevel = advice.Recommended
		case "leverage":
			// 这里需要根据交易对类型判断
			if advice.Symbol == "BTCUSDT" || advice.Symbol == "ETHUSDT" {
				adjustment.BTCETHLeverage = int(advice.Recommended)
			} else {
				adjustment.AltcoinLeverage = int(advice.Recommended)
			}
		case "position_size":
			adjustment.MaxPositionSize = advice.Recommended
		case "risk_control":
			adjustment.MaxDailyLoss = advice.Recommended
		}
	}

	if err := re.store.Reflection().SaveSystemAdjustment(adjustment); err != nil {
		return fmt.Errorf("failed to save adjustment: %w", err)
	}

	// 3. 保存 AI 学习记忆
	for _, adviceJSON := range reflection.AILearningAdvice {
		var advice store.ReflectionRecommendation
		if err := unmarshalJSON(string(adviceJSON), &advice); err != nil {
			logger.Warnf("⚠️  Failed to parse AI learning advice: %v", err)
			continue
		}

		memory := &store.AILearningMemory{
			TraderID:     reflection.TraderID,
			ReflectionID: reflection.ID,
			MemoryType:   advice.Type,
			Symbol:       advice.Symbol,
			Content:      advice.Reason,
			Confidence:   float64(advice.Priority) / 5.0, // 优先级转换为信心度
			PromptInjection: fmt.Sprintf(
				"Based on past analysis: %s. Previous recommendation: %v → %v",
				advice.Reason, advice.Current, advice.Recommended,
			),
			ExpiresAt: time.Now().UTC().AddDate(0, 1, 0), // 1 个月过期
		}

		if err := re.store.Reflection().SaveLearningMemory(memory); err != nil {
			logger.Warnf("⚠️  Failed to save learning memory: %v", err)
		}
	}

	logger.Infof("✅ Recommendations applied and learning memory saved")
	return nil
}

// ============================================================================
// Helper Methods
// ============================================================================

// TradeStats holds trade statistics
type TradeStats struct {
	TotalTrades        int
	SuccessfulTrades   int
	FailedTrades       int
	SuccessRate        float64
	AveragePnL         float64
	MaxProfit          float64
	MaxLoss            float64
	TotalPnL           float64
	PnLPercentage      float64
	SharpeRatio        float64
	MaxDrawdown        float64
	WinLossRatio       float64
	ConfidenceAccuracy map[string]float64
	SymbolPerformance  map[string]interface{}
}

// calculateStats calculates statistics
func (re *ReflectionEngine) calculateStats(trades []*store.TradeHistoryRecord) *TradeStats {
	stats := &TradeStats{
		TotalTrades:        len(trades),
		ConfidenceAccuracy: make(map[string]float64),
		SymbolPerformance:  make(map[string]interface{}),
	}

	if len(trades) == 0 {
		return stats
	}

	totalPnL := 0.0
	totalPnLAbs := 0.0
	maxProfit := 0.0
	maxLoss := 0.0
	symbolPnL := make(map[string]float64)
	symbolCount := make(map[string]int)
	confidenceBuckets := make(map[string][]float64)

	for _, trade := range trades {
		pnl := trade.ExitPrice*trade.Quantity - trade.EntryPrice*trade.Quantity
		totalPnL += pnl
		totalPnLAbs += math.Abs(pnl)

		if pnl > 0 {
			stats.SuccessfulTrades++
			if pnl > maxProfit {
				maxProfit = pnl
			}
		} else if pnl < 0 {
			stats.FailedTrades++
			if pnl < maxLoss {
				maxLoss = pnl
			}
		}

		// Symbol performance
		symbolPnL[trade.Symbol] += pnl
		symbolCount[trade.Symbol]++

		// Confidence accuracy grouping
		confBucket := fmt.Sprintf("%.0f%%", trade.Confidence*100)
		confidenceBuckets[confBucket] = append(confidenceBuckets[confBucket], float64(pnl))
	}

	stats.TotalPnL = totalPnL
	stats.MaxProfit = maxProfit
	stats.MaxLoss = maxLoss
	stats.SuccessRate = float64(stats.SuccessfulTrades) / float64(stats.TotalTrades)

	if stats.TotalTrades > 0 {
		stats.AveragePnL = totalPnL / float64(stats.TotalTrades)
	}

	if stats.SuccessfulTrades > 0 {
		stats.WinLossRatio = maxProfit / math.Abs(maxLoss)
	}

	// Calculate Sharpe Ratio (simplified)
	if len(trades) > 1 {
		variance := 0.0
		for _, pnl := range trades {
			diff := float64(pnl.PnL) - stats.AveragePnL
			variance += diff * diff
		}
		stdDev := math.Sqrt(variance / float64(len(trades)))
		if stdDev > 0 {
			stats.SharpeRatio = stats.AveragePnL / stdDev * math.Sqrt(252) // 年化
		}
	}

	// Calculate max drawdown
	cumPnL := 0.0
	peak := 0.0
	for _, trade := range trades {
		cumPnL += trade.PnL
		if cumPnL > peak {
			peak = cumPnL
		}
		drawdown := peak - cumPnL
		if drawdown > stats.MaxDrawdown {
			stats.MaxDrawdown = drawdown
		}
	}

	// Symbol performance
	for symbol, pnl := range symbolPnL {
		count := symbolCount[symbol]
		stats.SymbolPerformance[symbol] = map[string]interface{}{
			"total_pnl": pnl,
			"count":     count,
			"avg_pnl":   pnl / float64(count),
		}
	}

	// Confidence accuracy
	for bucket, pnls := range confidenceBuckets {
		successCount := 0
		for _, pnl := range pnls {
			if pnl > 0 {
				successCount++
			}
		}
		accuracy := float64(successCount) / float64(len(pnls))
		stats.ConfidenceAccuracy[bucket] = accuracy
	}

	return stats
}

// getAIReflection calls AI for reflection
func (re *ReflectionEngine) getAIReflection(traderID string, trades []*store.TradeHistoryRecord, stats *TradeStats) (string, error) {
	// Build AI prompt with JSON format request
	confidenceAccuracyStr := ""
	for bucket, accuracy := range stats.ConfidenceAccuracy {
		confidenceAccuracyStr += fmt.Sprintf("  %s: %.1f%%\n", bucket, accuracy*100)
	}

	symbolPerformanceStr := ""
	for symbol, perf := range stats.SymbolPerformance {
		if perfMap, ok := perf.(map[string]interface{}); ok {
			symbolPerformanceStr += fmt.Sprintf("  %s: 总盈亏=%.2f, 交易数=%v, 平均=%.2f\n",
				symbol, perfMap["total_pnl"], perfMap["count"], perfMap["avg_pnl"])
		}
	}

	prompt := fmt.Sprintf(`
您是一个交易系统分析专家。请基于以下交易数据进行深入反思分析，并生成结构化的改进建议。

【交易周期统计】
总交易数: %d
成功交易: %d | 失败交易: %d
成功率: %.2f%%
平均收益: %.2f USDT
最大盈利: %.2f USDT
最大亏损: %.2f USDT
总收益: %.2f USDT
夏普比率: %.2f
最大回撤: %.2f USDT
胜负比: %.2f

【信心度准确率（分组）】
%s

【交易对表现】
%s

【分析要求】
请从以下方面进行反思分析：

1. 📊 性能评估
   - 这个周期的总体表现如何？
   - 成功率是否达到预期？为什么？

2. 🎯 信心度偏差
   - 哪个信心度区间的准确率偏低？
   - 这意味着什么？(过度自信/保守？)

3. 💱 交易对分析
   - 表现最好的交易对是什么？为什么？
   - 表现最差的交易对是什么？为什么？
   - 是否应该调整某些交易对的参数或停止交易？

4. 🔧 参数改进建议
   - 信心度阈值是否需要调整？建议调整多少？
   - 杠杆倍数是否需要调整？
   - 最大仓位/日亏损是否需要调整？

5. 🧠 AI 学习点
   - 分析中存在的偏差或盲点是什么？
   - 需要学习哪些交易对特性？
   - 有哪些风险警告需要记住？

【输出格式】
请按以下 JSON 格式输出改进建议：

{
  "performance_summary": "总体表现评价（一句话）",
  "key_issues": ["问题1", "问题2", "问题3"],
  "recommendations": [
    {
      "type": "trade_system",
      "category": "confidence",
      "symbol": "BTC",
      "current": 75.0,
      "recommended": 70.0,
      "reason": "原因说明",
      "impact": "high",
      "priority": 1
    },
    {
      "type": "ai_learning",
      "category": "bias",
      "symbol": "ADAUSDT",
      "current": 0.0,
      "recommended": 0.0,
      "reason": "ADAUSDT 波动率高，需要更宽松的支撑位判断。过去高信心决策成功率反而较低。",
      "impact": "high",
      "priority": 2
    }
  ],
  "learning_memories": [
    {
      "type": "warning",
      "symbol": "ADAUSDT",
      "content": "ADAUSDT 波动率高，支撑位判断困难"
    },
    {
      "type": "pattern",
      "symbol": "BTCUSDT",
      "content": "BTCUSDT 稳定性强，100%% 成功率，可增加杠杆和仓位"
    }
  ]
}
`,
		stats.TotalTrades,
		stats.SuccessfulTrades,
		stats.FailedTrades,
		stats.SuccessRate*100,
		stats.AveragePnL,
		stats.MaxProfit,
		stats.MaxLoss,
		stats.TotalPnL,
		stats.SharpeRatio,
		stats.MaxDrawdown,
		stats.WinLossRatio,
		confidenceAccuracyStr,
		symbolPerformanceStr,
	)

	// Call MCP client to get AI reflection
	if re.mcpClient == nil {
		logger.Warnf("⚠️  MCP client not available, using mock reflection")
		return re.getMockReflection(stats), nil
	}

	// Call AI using CallWithMessages method
	response, err := re.mcpClient.CallWithMessages("", prompt)
	if err != nil {
		logger.Warnf("⚠️  AI call failed: %v, using mock reflection", err)
		return re.getMockReflection(stats), nil
	}

	return response, nil
}

// getMockReflection returns mock reflection for testing
func (re *ReflectionEngine) getMockReflection(stats *TradeStats) string {
	return fmt.Sprintf(`{
  "performance_summary": "成功率%.0f%%，总盈亏%.2f USDT，需要优化信心度阈值",
  "key_issues": [
    "高信心度成功率偏低，存在过度自信",
    "某些交易对表现不佳，需要调整或暂停"
  ],
  "recommendations": [
    {
      "type": "trade_system",
      "category": "confidence",
      "symbol": "",
      "current": 75.0,
      "recommended": 70.0,
      "reason": "识别到高信心度准确率低于预期，建议降低阈值",
      "impact": "high",
      "priority": 1
    }
  ],
  "learning_memories": [
    {
      "type": "lesson",
      "symbol": "",
      "content": "信心度高不一定意味着成功率高，需要结合实际表现调整"
    }
  ]
}`, stats.SuccessRate*100, stats.TotalPnL)
}

// separateAdvice separates advice into trade system and AI learning
func (re *ReflectionEngine) separateAdvice(recommendations string) ([]json.RawMessage, []json.RawMessage) {
	var result map[string]interface{}
	if err := json.Unmarshal([]byte(recommendations), &result); err != nil {
		logger.Warnf("⚠️  Failed to parse AI recommendations JSON: %v", err)
		return []json.RawMessage{}, []json.RawMessage{}
	}

	var tradeSystemAdvice []json.RawMessage
	var aiLearningAdvice []json.RawMessage

	// Parse recommendations
	if recsAny, ok := result["recommendations"]; ok {
		if recsData, err := json.Marshal(recsAny); err == nil {
			var recs []map[string]interface{}
			if err := json.Unmarshal(recsData, &recs); err == nil {
				for _, rec := range recs {
					if recData, err := json.Marshal(rec); err == nil {
						if recType, ok := rec["type"].(string); ok {
							if recType == "trade_system" {
								tradeSystemAdvice = append(tradeSystemAdvice, json.RawMessage(recData))
							} else if recType == "ai_learning" {
								aiLearningAdvice = append(aiLearningAdvice, json.RawMessage(recData))
							}
						}
					}
				}
			}
		}
	}

	// Parse learning memories (add them to AI learning advice)
	if memoriesAny, ok := result["learning_memories"]; ok {
		if memoryData, err := json.Marshal(memoriesAny); err == nil {
			var memories []map[string]interface{}
			if err := json.Unmarshal(memoryData, &memories); err == nil {
				for _, mem := range memories {
					if memData, err := json.Marshal(mem); err == nil {
						aiLearningAdvice = append(aiLearningAdvice, json.RawMessage(memData))
					}
				}
			}
		}
	}

	logger.Infof("📊 Separated %d trade system advice and %d AI learning advice",
		len(tradeSystemAdvice), len(aiLearningAdvice))

	return tradeSystemAdvice, aiLearningAdvice
}

// unmarshalJSON unmarshals JSON
func unmarshalJSON(data string, v interface{}) error {
	return json.Unmarshal([]byte(data), v)
}
