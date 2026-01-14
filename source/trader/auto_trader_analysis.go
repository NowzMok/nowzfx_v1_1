package trader

import (
	"fmt"
	"nofx/kernel"
	"nofx/logger"
	"nofx/market"
	"nofx/store"
	"time"
)

// SaveAnalysisAndCreatePendingOrders 保存 AI 分析并创建待执行订单（支持同币种去重）
func (at *AutoTrader) SaveAnalysisAndCreatePendingOrders(aiDecision *kernel.FullDecision) error {
	if at.store == nil {
		return fmt.Errorf("store is not initialized")
	}

	if aiDecision == nil {
		return fmt.Errorf("AI decision is nil")
	}

	logger.Infof("💾 Saving AI analysis and creating pending orders...")

	// 获取所有PENDING状态的订单，用于同币种检查
	existingOrders, err := at.store.Analysis().GetPendingOrdersByTrader(at.id)
	if err != nil {
		logger.Errorf("❌ Failed to get existing pending orders: %v", err)
		// 继续执行，不影响保存分析
	}

	// 创建订单映射，便于快速查找
	// 🚨 增强版：检查所有非终态订单 + 最近成交订单（30分钟内），避免重复创建
	existingOrderMap := make(map[string]*store.PendingOrder)
	now := time.Now().UTC()
	for _, order := range existingOrders {
		// 检查 PENDING 和 TRIGGERED 状态的订单
		if order.Status == "PENDING" || order.Status == "TRIGGERED" {
			existingOrderMap[order.Symbol] = order
			continue
		}

		// 检查最近成交的订单（30分钟内），避免短时间内重复开仓
		if order.Status == "FILLED" && order.FilledAt != nil {
			timeSinceFilled := now.Sub(*order.FilledAt)
			if timeSinceFilled < 30*time.Minute {
				logger.Infof("⏰ %s has recently filled order (%.1f min ago), skipping duplicate",
					order.Symbol, timeSinceFilled.Minutes())
				existingOrderMap[order.Symbol] = order
			}
		}
	}

	for _, decision := range aiDecision.Decisions {
		// 1. 保存分析记录（总是保存）
		analysis := &store.AnalysisRecord{
			TraderID:       at.id,
			Symbol:         decision.Symbol,
			TargetPrice:    decision.TakeProfit,
			Confidence:     float64(decision.Confidence) / 100.0,
			AnalysisReason: decision.Reasoning,
			AnalysisPrompt: aiDecision.UserPrompt,
			AIResponse:     aiDecision.RawResponse,
			AnalysisTime:   time.Now().UTC(),
			Status:         "ACTIVE",
		}

		// 解析支撑位和压力位
		if decision.StopLoss > 0 {
			analysis.SupportLevels = store.SupportLevels{decision.StopLoss}
		}
		if decision.TakeProfit > 0 {
			analysis.ResistanceLevel = decision.TakeProfit
		}

		if err := at.store.Analysis().SaveAnalysis(analysis); err != nil {
			logger.Errorf("❌ Failed to save analysis for %s: %v", decision.Symbol, err)
			continue
		}

		logger.Infof("✅ Analysis saved: %s (confidence: %.2f%%)", decision.Symbol, float64(decision.Confidence))

		// 2. 为开仓决策创建待执行订单
		if decision.Action == "open_long" || decision.Action == "open_short" {
			// 获取当前价格来计算合理的触发价
			currentPrice := 0.0
			if marketData, err := market.Get(decision.Symbol); err == nil {
				currentPrice = marketData.CurrentPrice
			}

			// 计算触发价格（使用策略配置 + 止盈止损）
			triggerConfig := at.config.StrategyConfig.TriggerPriceConfig
			if triggerConfig == nil {
				// 如果未配置，使用策略中定义的风格，或默认为swing
				style := "swing"
				if at.config.StrategyConfig.TriggerPriceConfig != nil {
					style = at.config.StrategyConfig.TriggerPriceConfig.Style
				}
				triggerConfig = store.GetDefaultTriggerPriceConfig(style)
				logger.Warnf("⚠️ TriggerPriceConfig is nil, using default style '%s'", style)
			}

			// 🚨 调试：打印配置信息
			logger.Infof("🔧 [TRIGGER_PRICE_DEBUG] Strategy Config Check:")
			logger.Infof("  Trader ID: %s", at.id)
			logger.Infof("  Symbol: %s", decision.Symbol)
			logger.Infof("  Action: %s", decision.Action)
			logger.Infof("  Current Price: %.4f", currentPrice)
			logger.Infof("  Stop Loss: %.4f", decision.StopLoss)
			logger.Infof("  Take Profit: %.4f", decision.TakeProfit)
			logger.Infof("  TriggerPriceConfig is nil: %v", triggerConfig == nil)
			if triggerConfig != nil {
				logger.Infof("  Config Mode: %s", triggerConfig.Mode)
				logger.Infof("  Config Style: %s", triggerConfig.Style)
				logger.Infof("  Pullback Ratio: %.4f", triggerConfig.PullbackRatio)
				logger.Infof("  Breakout Ratio: %.4f", triggerConfig.BreakoutRatio)
				logger.Infof("  Extra Buffer: %.4f", triggerConfig.ExtraBuffer)
			} else {
				logger.Errorf("❌ TriggerPriceConfig is nil! This indicates configuration was not properly saved or loaded")
				logger.Infof("  Strategy Config exists: %v", at.config.StrategyConfig != nil)
				if at.config.StrategyConfig != nil {
					logger.Infof("  Full Strategy Config: %+v", at.config.StrategyConfig)
				}
			}

			triggerPriceCalculator := NewTriggerPriceCalculator(triggerConfig)

			// 使用新的基于止盈止损的计算方法
			triggerPrice := triggerPriceCalculator.CalculateWithStopLoss(
				currentPrice,
				decision.Action,
				decision.StopLoss,
				decision.TakeProfit,
			)

			logger.Infof("🔧 [TRIGGER_PRICE_DEBUG] Calculation Result:")
			logger.Infof("  Trigger Price: %.4f", triggerPrice)
			logger.Infof("  Stop Loss: %.4f", decision.StopLoss)
			logger.Infof("  Take Profit: %.4f", decision.TakeProfit)
			logger.Infof("  Trigger in range: %v", triggerPrice > decision.StopLoss && triggerPrice < decision.TakeProfit)
			logger.Infof("  Distance from current: %.4f (%.2f%%)",
				currentPrice-triggerPrice,
				((currentPrice - triggerPrice) / currentPrice * 100))

			// 检查同币种是否已存在PENDING订单
			if existingOrder, exists := existingOrderMap[decision.Symbol]; exists {
				// 使用智能替换策略
				newConfidence := float64(decision.Confidence) / 100.0
				shouldReplace := false
				replaceReason := ""

				// 计算订单年龄和价格偏离
				orderAge := time.Since(existingOrder.CreatedAt)
				priceDeviation := 0.0
				if currentPrice > 0 && existingOrder.TriggerPrice > 0 {
					priceDeviation = (currentPrice - existingOrder.TriggerPrice) / existingOrder.TriggerPrice
					if priceDeviation < 0 {
						priceDeviation = -priceDeviation
					}
				}

				// 智能替换条件：
				// 1. 新订单置信度更高
				// 2. 或者旧订单已存在超过 6 小时且新订单置信度 >= 0.7
				// 3. 或者旧订单价格偏离超过 10% 且新订单置信度 >= 0.75
				if newConfidence > existingOrder.Confidence {
					shouldReplace = true
					replaceReason = fmt.Sprintf("higher confidence (%.2f%% > %.2f%%)",
						newConfidence*100, existingOrder.Confidence*100)
				} else if orderAge > 6*time.Hour && newConfidence >= 0.7 {
					shouldReplace = true
					replaceReason = fmt.Sprintf("old order (%.1fh) with decent confidence (%.2f%%)",
						orderAge.Hours(), newConfidence*100)
				} else if priceDeviation > 0.10 && newConfidence >= 0.75 {
					shouldReplace = true
					replaceReason = fmt.Sprintf("price deviation %.2f%% with high confidence (%.2f%%)",
						priceDeviation*100, newConfidence*100)
				}

				if shouldReplace {
					logger.Infof("🔄 替换同币种订单: %s (原因: %s)", decision.Symbol, replaceReason)

					// 取消旧订单
					if err := at.store.Analysis().CancelPendingOrder(existingOrder.ID,
						fmt.Sprintf("Replaced: %s", replaceReason)); err != nil {
						logger.Warnf("⚠️ Failed to cancel old order: %v", err)
					}

					// 移除已替换的订单
					delete(existingOrderMap, decision.Symbol)
				} else {
					// 现有订单更优，跳过创建
					logger.Infof("⏭️ 跳过同币种订单: %s (保留现有订单: 置信度 %.2f%%, 年龄 %.1fh, 偏离 %.2f%%)",
						decision.Symbol, existingOrder.Confidence*100, orderAge.Hours(), priceDeviation*100)
					continue
				}
			}

			// 创建新订单
			pendingOrder := &store.PendingOrder{
				TraderID:     at.id,
				Symbol:       decision.Symbol,
				AnalysisID:   analysis.ID,
				TargetPrice:  decision.TakeProfit,
				TriggerPrice: triggerPrice,
				PositionSize: decision.PositionSizeUSD,
				Leverage:     decision.Leverage,
				StopLoss:     decision.StopLoss,
				TakeProfit:   decision.TakeProfit,
				Confidence:   float64(decision.Confidence) / 100.0,
				Status:       "PENDING",
			}

			if err := at.store.Analysis().SavePendingOrder(pendingOrder); err != nil {
				logger.Errorf("❌ Failed to save pending order for %s: %v", decision.Symbol, err)
				continue
			}

			// 更新映射
			existingOrderMap[decision.Symbol] = pendingOrder

			logger.Infof("⏳ Pending order created: %s (trigger: %.2f, target: %.2f, confidence: %.2f%%)",
				decision.Symbol, triggerPrice, decision.TakeProfit, float64(decision.Confidence))
		}
	}

	return nil
}

// MonitorAndExecutePendingOrders 监控待执行订单并在价格触发时自动执行
func (at *AutoTrader) MonitorAndExecutePendingOrders() error {
	if at.store == nil {
		return nil
	}

	// 获取所有 PENDING 状态的订单
	pendingOrders, err := at.store.Analysis().GetPendingOrdersByStatus(at.id, "PENDING")
	if err != nil {
		logger.Errorf("❌ Failed to get pending orders: %v", err)
		return err
	}

	if len(pendingOrders) == 0 {
		return nil
	}

	logger.Infof("📊 Checking %d pending orders...", len(pendingOrders))

	for _, order := range pendingOrders {
		// 获取当前价格
		currentPrice := 0.0
		if marketData, err := market.Get(order.Symbol); err == nil {
			currentPrice = marketData.CurrentPrice
		} else {
			logger.Warnf("⚠️ Failed to get current price for %s: %v", order.Symbol, err)
			continue
		}

		// 推断交易方向：通过止损和止盈的位置关系
		// 做多(long): stop_loss < trigger_price < take_profit (价格下跌到触发价买入)
		// 做空(short): stop_loss > trigger_price > take_profit (价格上涨到触发价卖出)
		isLong := order.StopLoss < order.TakeProfit

		// 计算价格偏离（统一为"到触发的距离"，正值表示还未触发）
		var deviation float64
		var deviationPct float64
		if isLong {
			// 做多：当前价格 > 触发价 = 正偏离（还未触发）
			deviation = currentPrice - order.TriggerPrice
			deviationPct = (deviation / order.TriggerPrice) * 100
		} else {
			// 做空：当前价格 < 触发价 = 正偏离（还未触发）
			deviation = order.TriggerPrice - currentPrice
			deviationPct = (deviation / order.TriggerPrice) * 100
		}

		direction := "LONG"
		if !isLong {
			direction = "SHORT"
		}

		logger.Infof("📈 %s [%s]: current=%.2f, trigger=%.2f (deviation: %.2f%%)",
			order.Symbol, direction, currentPrice, order.TriggerPrice, deviationPct)

		// 检查是否触发
		// 做多(LONG)：当前价格 >= 触发价（价格反弹到或穿过触发价）
		// 做空(SHORT)：当前价格 <= 触发价（价格下跌到或穿过触发价）
		triggered := false
		if isLong && currentPrice >= order.TriggerPrice {
			triggered = true
		} else if !isLong && currentPrice <= order.TriggerPrice {
			triggered = true
		}

		if triggered {
			logger.Infof("🎯 Pending order triggered: %s [%s] at %.2f", order.Symbol, direction, currentPrice)

			// � 改进：使用指数退避重试策略
			if err := at.executePendingOrderWithBackoff(order, currentPrice); err != nil {
				logger.Errorf("❌ Failed to execute pending order after backoff retries: %v", err)
				// 记录执行失败，增加重试计数
				at.recordPendingOrderFailure(order.ID, err)
				continue // 保持 PENDING 状态，允许重试
			}

			// ✅ 执行成功后才标记为 TRIGGERED
			if err := at.store.Analysis().UpdatePendingOrderStatus(
				order.ID, "TRIGGERED", currentPrice, time.Now().UTC(),
			); err != nil {
				logger.Warnf("⚠️ Failed to mark order as triggered: %v", err)
			} else {
				logger.Infof("✅ Pending order executed successfully: %s", order.Symbol)
			}
		} else {
			// 检查订单是否应该被取消（价格偏离过大或订单过旧）
			at.checkAndCleanupOrder(order, currentPrice)
		}
	}

	return nil
}

// executePendingOrderWithBackoff 使用指数退避策略执行订单
func (at *AutoTrader) executePendingOrderWithBackoff(order *store.PendingOrder, currentPrice float64) error {
	const maxRetries = 5
	baseDelay := 2 * time.Second

	var lastErr error
	for attempt := 0; attempt < maxRetries; attempt++ {
		if attempt > 0 {
			// 指数退避：2s, 4s, 8s, 16s, 32s
			delay := baseDelay * time.Duration(1<<uint(attempt-1))
			logger.Infof("  ⏳ Retry %d/%d after %v delay...", attempt+1, maxRetries, delay)
			time.Sleep(delay)
		}

		err := at.executePendingOrder(order, currentPrice)
		if err == nil {
			if attempt > 0 {
				logger.Infof("  ✅ Order executed successfully on retry %d", attempt+1)
				// 记录重试成功
				if at.errorTracker != nil {
					at.errorTracker.RecordError(
						"RETRY_SUCCESS",
						order.Symbol,
						fmt.Sprintf("Order executed after %d retries", attempt+1),
						"INFO",
					)
				}
			}
			return nil
		}

		lastErr = err
		logger.Warnf("  ⚠️ Attempt %d/%d failed: %v", attempt+1, maxRetries, err)

		// 记录重试失败
		if at.errorTracker != nil {
			severity := "WARN"
			if attempt == maxRetries-1 {
				severity = "ERROR"
			}
			at.errorTracker.RecordError(
				"RETRY_FAILED",
				order.Symbol,
				fmt.Sprintf("Attempt %d/%d: %v", attempt+1, maxRetries, err),
				severity,
			)
		}

		// 检查是否是不可重试的错误（如余额不足）
		if isNonRetryableError(err) {
			logger.Errorf("  ❌ Non-retryable error detected, stopping retries")
			// 记录不可重试错误
			if at.errorTracker != nil {
				at.errorTracker.RecordError(
					"NON_RETRYABLE_ERROR",
					order.Symbol,
					fmt.Sprintf("Error type prevents retry: %v", err),
					"CRITICAL",
				)
			}
			return err
		}
	}

	// 记录最终失败
	if at.errorTracker != nil {
		at.errorTracker.RecordError(
			"EXECUTION_FAILED",
			order.Symbol,
			fmt.Sprintf("Failed after %d attempts: %v", maxRetries, lastErr),
			"CRITICAL",
		)
	}

	return fmt.Errorf("failed after %d attempts with exponential backoff: %w", maxRetries, lastErr)
}

// isNonRetryableError 判断错误是否不可重试
func isNonRetryableError(err error) bool {
	if err == nil {
		return false
	}
	errMsg := err.Error()
	// 不可重试的错误类型
	nonRetryable := []string{
		"insufficient",
		"balance",
		"margin",
		"invalid symbol",
		"position limit",
		"order would trigger immediately",
	}

	for _, pattern := range nonRetryable {
		if containsAnyPattern(errMsg, pattern) {
			return true
		}
	}
	return false
}

// containsAnyPattern 检查字符串是否包含任何子串（不区分大小写）
func containsAnyPattern(s, substr string) bool {
	return len(s) >= len(substr) &&
		(s == substr || len(s) > len(substr) &&
			(s[:len(substr)] == substr || s[len(s)-len(substr):] == substr ||
				indexOfPattern(s, substr) >= 0))
}

func indexOfPattern(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}

// recordPendingOrderFailure 记录待决策订单执行失败
func (at *AutoTrader) recordPendingOrderFailure(orderID string, execErr error) {
	at.mu.Lock()
	if at.pendingOrderRetries == nil {
		at.pendingOrderRetries = make(map[string]int)
	}
	at.pendingOrderRetries[orderID]++
	retries := at.pendingOrderRetries[orderID]
	at.mu.Unlock()

	const maxCycleRetries = 3 // 最多在3个周期内重试
	if retries >= maxCycleRetries {
		reason := fmt.Sprintf("Execution failed %d times across cycles: %v", retries, execErr)
		if err := at.store.Analysis().CancelPendingOrder(orderID, reason); err != nil {
			logger.Warnf("⚠️ Failed to cancel failed order %s: %v", orderID, err)
		} else {
			logger.Infof("🗑️ Cancelled order %s after %d cycle failures", orderID[:8], retries)
		}
		at.mu.Lock()
		delete(at.pendingOrderRetries, orderID)
		at.mu.Unlock()
	} else {
		logger.Warnf("⚠️ Order %s failed in cycle (%d/%d cycle retries remaining)",
			orderID[:8], retries, maxCycleRetries)
	}
}

// checkAndCleanupOrder 检查并清理不合理的订单
func (at *AutoTrader) checkAndCleanupOrder(order *store.PendingOrder, currentPrice float64) {
	// 1. 检查订单年龄（超过 12 小时自动取消）
	orderAge := time.Since(order.CreatedAt)
	if orderAge > 12*time.Hour {
		reason := fmt.Sprintf("Order too old: %.1f hours", orderAge.Hours())
		if err := at.store.Analysis().CancelPendingOrder(order.ID, reason); err != nil {
			logger.Warnf("⚠️ Failed to cancel old order: %v", err)
		} else {
			logger.Infof("🗑️ Cancelled old order %s: %s (%.1fh old)", order.Symbol, order.ID[:8], orderAge.Hours())
		}
		return
	}

	// 2. 检查价格偏离（偏离超过 15% 自动取消）
	// 使用方向感知的偏离计算
	if currentPrice > 0 && order.TriggerPrice > 0 {
		isLong := order.StopLoss < order.TakeProfit

		var deviation float64
		if isLong {
			// 做多：当前价格远高于触发价 = 大偏离
			deviation = (currentPrice - order.TriggerPrice) / order.TriggerPrice
		} else {
			// 做空：当前价格远低于触发价 = 大偏离
			deviation = (order.TriggerPrice - currentPrice) / order.TriggerPrice
		}

		// 偏离可能为负（已经穿过触发价但还没执行），取绝对值
		if deviation < 0 {
			deviation = -deviation
		}

		if deviation > 0.15 {
			direction := "LONG"
			if !isLong {
				direction = "SHORT"
			}
			reason := fmt.Sprintf("Price deviation too high [%s]: %.2f%% (current: %.4f, trigger: %.4f)",
				direction, deviation*100, currentPrice, order.TriggerPrice)
			if err := at.store.Analysis().CancelPendingOrder(order.ID, reason); err != nil {
				logger.Warnf("⚠️ Failed to cancel deviated order: %v", err)
			} else {
				logger.Infof("🗑️ Cancelled deviated order %s [%s]: %s (%.2f%% deviation)",
					order.Symbol, direction, order.ID[:8], deviation*100)
			}
		}
	}
}

// executePendingOrder 执行待执行的订单
func (at *AutoTrader) executePendingOrder(order *store.PendingOrder, currentPrice float64) error {
	logger.Infof("  🚀 Executing pending order: %s", order.Symbol)

	// 检查账户状态
	balance, err := at.trader.GetBalance()
	if err != nil {
		return fmt.Errorf("failed to get account balance: %w", err)
	}

	availableBalance := 0.0
	if avail, ok := balance["availableBalance"].(float64); ok {
		availableBalance = avail
	}

	// 检查余额是否充足
	marginFactor := 1.01/float64(order.Leverage) + 0.001
	requiredMargin := order.PositionSize * marginFactor
	if availableBalance < requiredMargin {
		return fmt.Errorf("insufficient margin: need %.2f, have %.2f", requiredMargin, availableBalance)
	}

	// 根据止损止盈推断交易方向
	// 做多: stop_loss < take_profit (止损在下，止盈在上)
	// 做空: stop_loss > take_profit (止损在上，止盈在下)
	action := "open_long"
	if order.StopLoss > order.TakeProfit {
		action = "open_short"
	}

	// 构造决策对象
	decision := &kernel.Decision{
		Symbol:          order.Symbol,
		Action:          action,
		Leverage:        order.Leverage,
		PositionSizeUSD: order.PositionSize,
		StopLoss:        order.StopLoss,
		TakeProfit:      order.TakeProfit,
		Confidence:      int(order.Confidence * 100), // 转换为 0-100 范围
		Reasoning:       fmt.Sprintf("Auto-executed from pending order [%s] at %.2f", action, currentPrice),
	}

	// 创建行动记录
	actionRecord := &store.DecisionAction{
		Action:     decision.Action,
		Symbol:     decision.Symbol,
		Leverage:   decision.Leverage,
		StopLoss:   decision.StopLoss,
		TakeProfit: decision.TakeProfit,
		Confidence: decision.Confidence,
		Reasoning:  decision.Reasoning,
		Timestamp:  time.Now().UTC(),
		Success:    false,
	}

	// 执行决策
	if err := at.executeDecisionWithRecord(decision, actionRecord); err != nil {
		return fmt.Errorf("failed to execute decision: %w", err)
	}

	// 执行成功后更新订单状态为 FILLED（无论是否有 OrderID）
	// 交易历史记录
	if actionRecord.Success {
		// 保存交易历史记录
		tradeHistory := &store.TradeHistoryRecord{
			TraderID:       at.id,
			Symbol:         order.Symbol,
			AnalysisID:     order.AnalysisID,
			PendingOrderID: order.ID,
			EntryPrice:     currentPrice,
			Quantity:       actionRecord.Quantity,
			Leverage:       order.Leverage,
			EntryTime:      time.Now().UTC(),
		}

		if err := at.store.Analysis().SaveTradeHistory(tradeHistory); err != nil {
			logger.Warnf("⚠️ Failed to save trade history: %v", err)
		}

		// 直接更新为 FILLED 状态（包含触发价格和成交信息）
		orderID := actionRecord.OrderID
		if orderID == 0 {
			orderID = -1 // 使用 -1 表示没有获取到 OrderID，但订单已执行
		}

		if err := at.store.Analysis().UpdatePendingOrderFilledWithPrice(
			order.ID, currentPrice, time.Now().UTC(), orderID,
		); err != nil {
			logger.Warnf("⚠️ Failed to mark order as filled: %v", err)
		} else {
			logger.Infof("✅ Order status updated to FILLED: %s (trigger price: %.2f)", order.Symbol, currentPrice)
		}
	} else {
		// 执行失败，订单保持 TRIGGERED 状态，下次重试
		logger.Warnf("⚠️ Order execution unsuccessful, will retry: %s", order.Symbol)
	}

	return nil
}
