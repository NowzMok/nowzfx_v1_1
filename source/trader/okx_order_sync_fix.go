package trader

import (
	"fmt"
	"nofx/logger"
	"nofx/store"
	"sort"
	"strings"
	"time"
)

// SyncOrdersFromOKXWithTPSL 同步订单并设置TP/SL
func (t *OKXTrader) SyncOrdersFromOKXWithTPSL(traderID string, exchangeID string, exchangeType string, st *store.Store, autoTrader *AutoTrader) error {
	if st == nil {
		return fmt.Errorf("store is nil")
	}

	// Get recent trades (last 24 hours)
	startTime := time.Now().Add(-24 * time.Hour)

	logger.Infof("🔄 Syncing OKX trades with TPSL from: %s", startTime.Format(time.RFC3339))

	// Use GetTrades method to fetch trade records
	trades, err := t.GetTrades(startTime, 100)
	if err != nil {
		return fmt.Errorf("failed to get trades: %w", err)
	}

	logger.Infof("📥 Received %d trades from OKX", len(trades))

	// Sort trades by time ASC (oldest first) for proper position building
	sort.Slice(trades, func(i, j int) bool {
		return trades[i].ExecTime.UnixMilli() < trades[j].ExecTime.UnixMilli()
	})

	// Process trades one by one
	orderStore := st.Order()
	positionStore := st.Position()
	posBuilder := store.NewPositionBuilder(positionStore)
	syncedCount := 0

	for _, trade := range trades {
		// Check if trade already exists
		existing, err := orderStore.GetOrderByExchangeID(exchangeID, trade.TradeID)
		if err == nil && existing != nil {
			continue // Order already exists, skip
		}

		// Normalize symbol
		symbol := trade.Symbol

		// Determine position side from order action
		positionSide := "LONG"
		if strings.Contains(trade.OrderAction, "short") {
			positionSide = "SHORT"
		}

		// Normalize side for storage
		side := strings.ToUpper(trade.Side)

		// Create order record
		execTimeMs := trade.ExecTime.UTC().UnixMilli()
		orderRecord := &store.TraderOrder{
			TraderID:        traderID,
			ExchangeID:      exchangeID,
			ExchangeType:    exchangeType,
			ExchangeOrderID: trade.TradeID,
			Symbol:          symbol,
			Side:            side,
			PositionSide:    positionSide,
			Type:            trade.OrderType,
			OrderAction:     trade.OrderAction,
			Quantity:        trade.FillQtyBase,
			Price:           trade.FillPrice,
			Status:          "FILLED",
			FilledQuantity:  trade.FillQtyBase,
			AvgFillPrice:    trade.FillPrice,
			Commission:      trade.Fee,
			FilledAt:        execTimeMs,
			CreatedAt:       execTimeMs,
			UpdatedAt:       execTimeMs,
		}

		// Insert order record
		if err := orderStore.CreateOrder(orderRecord); err != nil {
			logger.Infof("  ⚠️ Failed to sync trade %s: %v", trade.TradeID, err)
			continue
		}

		// Create fill record
		fillRecord := &store.TraderFill{
			TraderID:        traderID,
			ExchangeID:      exchangeID,
			ExchangeType:    exchangeType,
			OrderID:         orderRecord.ID,
			ExchangeOrderID: trade.OrderID,
			ExchangeTradeID: trade.TradeID,
			Symbol:          symbol,
			Side:            side,
			Price:           trade.FillPrice,
			Quantity:        trade.FillQtyBase,
			QuoteQuantity:   trade.FillPrice * trade.FillQtyBase,
			Commission:      trade.Fee,
			CommissionAsset: trade.FeeAsset,
			RealizedPnL:     0,
			IsMaker:         trade.IsMaker,
			CreatedAt:       execTimeMs,
		}

		if err := orderStore.CreateFill(fillRecord); err != nil {
			logger.Infof("  ⚠️ Failed to sync fill for trade %s: %v", trade.TradeID, err)
		}

		// Create/update position record
		if err := posBuilder.ProcessTrade(
			traderID, exchangeID, exchangeType,
			symbol, positionSide, trade.OrderAction,
			trade.FillQtyBase, trade.FillPrice, trade.Fee, 0,
			execTimeMs, trade.TradeID,
		); err != nil {
			logger.Infof("  ⚠️ Failed to sync position for trade %s: %v", trade.TradeID, err)
		} else {
			logger.Infof("  📍 Position updated for trade: %s (action: %s, qty: %.6f)", trade.TradeID, trade.OrderAction, trade.FillQtyBase)
		}

		// 🚨 关键修复：在同步成交后设置TP/SL
		// 检查是否有对应的Pending订单，获取TP/SL信息
		if autoTrader != nil {
			// 从Pending订单中查找TP/SL配置
			pendingOrders, err := st.Analysis().GetPendingOrdersByTrader(traderID)
			if err == nil && len(pendingOrders) > 0 {
				var pendingOrder *store.PendingOrder
				for _, po := range pendingOrders {
					if po.Symbol == symbol && po.Status == "PENDING" {
						pendingOrder = po
						break
					}
				}

				if pendingOrder != nil {
					// 设置止损
					if pendingOrder.StopLoss > 0 {
						slErr := t.SetStopLoss(symbol, positionSide, trade.FillQtyBase, pendingOrder.StopLoss)
						if slErr != nil {
							logger.Warnf("  ⚠️ Failed to set stop loss for %s: %v", symbol, slErr)
						} else {
							logger.Infof("  ✅ Stop loss set for %s: %.4f", symbol, pendingOrder.StopLoss)
						}
					}

					// 设置止盈
					if pendingOrder.TakeProfit > 0 {
						tpErr := t.SetTakeProfit(symbol, positionSide, trade.FillQtyBase, pendingOrder.TakeProfit)
						if tpErr != nil {
							logger.Warnf("  ⚠️ Failed to set take profit for %s: %v", symbol, tpErr)
						} else {
							logger.Infof("  ✅ Take profit set for %s: %.4f", symbol, pendingOrder.TakeProfit)
						}
					}

					// 记录TP/SL到数据库
					if pendingOrder.StopLoss > 0 && pendingOrder.TakeProfit > 0 {
						// 获取刚创建的位置ID
						openPos, err := positionStore.GetOpenPositionBySymbol(traderID, symbol, positionSide)
						if err == nil && openPos != nil {
							if autoTrader != nil {
								autoTrader.recordTPSL(traderID, openPos, pendingOrder.TakeProfit, pendingOrder.StopLoss)
							}
						}
					}

					// 标记Pending订单为已执行
					if err := st.Analysis().UpdatePendingOrderStatus(pendingOrder.ID, "FILLED", trade.FillPrice, time.Now().UTC()); err != nil {
						logger.Warnf("  ⚠️ Failed to update pending order status: %v", err)
					}
				} else {
					logger.Warnf("  ⚠️ No pending order found for %s, cannot set TP/SL automatically", symbol)
				}
			} else {
				logger.Warnf("  ⚠️ No pending orders found for trader, cannot set TP/SL automatically")
			}
		}

		syncedCount++
		logger.Infof("  ✅ Synced trade: %s %s %s qty=%.6f price=%.6f fee=%.6f action=%s",
			trade.TradeID, trade.Symbol, side, trade.FillQtyBase, trade.FillPrice, trade.Fee, trade.OrderAction)
	}

	logger.Infof("✅ OKX order sync with TPSL completed: %d new trades synced", syncedCount)
	return nil
}
