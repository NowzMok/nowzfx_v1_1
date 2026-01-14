package trader

import (
	"encoding/json"
	"fmt"
	"nofx/logger"
	"nofx/market"
	"nofx/store"
	"sort"
	"strconv"
	"strings"
	"time"
)

// OKXTrade represents a trade record from OKX fills history
type OKXTrade struct {
	InstID      string
	Symbol      string
	TradeID     string
	OrderID     string
	Side        string // buy or sell
	PosSide     string // long or short
	FillPrice   float64
	FillQty     float64 // In contracts
	FillQtyBase float64 // In base asset (BTC, ETH, etc)
	Fee         float64
	FeeAsset    string
	ExecTime    time.Time
	IsMaker     bool
	OrderType   string
	OrderAction string // open_long, open_short, close_long, close_short
}

// GetTrades retrieves trade/fill records from OKX
func (t *OKXTrader) GetTrades(startTime time.Time, limit int) ([]OKXTrade, error) {
	if limit <= 0 {
		limit = 100
	}
	if limit > 100 {
		limit = 100 // OKX max limit is 100
	}

	// Build query path
	// OKX fills-history endpoint for historical fills
	path := fmt.Sprintf("/api/v5/trade/fills-history?instType=SWAP&limit=%d", limit)
	if !startTime.IsZero() {
		path += fmt.Sprintf("&begin=%d", startTime.UnixMilli())
	}

	data, err := t.doRequest("GET", path, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get fills history: %w", err)
	}

	var fills []struct {
		InstID   string `json:"instId"`   // e.g., "BTC-USDT-SWAP"
		TradeID  string `json:"tradeId"`  // Trade ID
		OrdID    string `json:"ordId"`    // Order ID
		BillID   string `json:"billId"`   // Bill ID
		Side     string `json:"side"`     // buy or sell
		PosSide  string `json:"posSide"`  // long, short, or net
		FillPx   string `json:"fillPx"`   // Fill price
		FillSz   string `json:"fillSz"`   // Fill size (contracts)
		Fee      string `json:"fee"`      // Fee (negative for cost)
		FeeCcy   string `json:"feeCcy"`   // Fee currency
		Ts       string `json:"ts"`       // Trade timestamp (ms)
		ExecType string `json:"execType"` // T: taker, M: maker
		Tag      string `json:"tag"`      // Order tag
	}

	if err := json.Unmarshal(data, &fills); err != nil {
		return nil, fmt.Errorf("failed to parse fills: %w", err)
	}

	trades := make([]OKXTrade, 0, len(fills))

	for _, fill := range fills {
		fillPrice, _ := strconv.ParseFloat(fill.FillPx, 64)
		fillSz, _ := strconv.ParseFloat(fill.FillSz, 64)
		fee, _ := strconv.ParseFloat(fill.Fee, 64)
		ts, _ := strconv.ParseInt(fill.Ts, 10, 64)

		// Convert symbol: BTC-USDT-SWAP -> BTCUSDT
		symbol := t.convertSymbolBack(fill.InstID)

		// Convert contract count to base asset quantity
		fillQtyBase := fillSz
		inst, err := t.getInstrument(symbol)
		if err == nil && inst.CtVal > 0 {
			fillQtyBase = fillSz * inst.CtVal
		}

		// Determine order action based on side and posSide
		// OKX uses dual position mode:
		// - buy + long = open long
		// - sell + long = close long
		// - sell + short = open short
		// - buy + short = close short
		orderAction := "open_long"
		posSide := strings.ToLower(fill.PosSide)
		side := strings.ToLower(fill.Side)

		if posSide == "long" {
			if side == "buy" {
				orderAction = "open_long"
			} else {
				orderAction = "close_long"
			}
		} else if posSide == "short" {
			if side == "sell" {
				orderAction = "open_short"
			} else {
				orderAction = "close_short"
			}
		} else {
			// One-way mode (net position)
			if side == "buy" {
				orderAction = "open_long"
			} else {
				orderAction = "open_short"
			}
		}

		trade := OKXTrade{
			InstID:      fill.InstID,
			Symbol:      symbol,
			TradeID:     fill.TradeID,
			OrderID:     fill.OrdID,
			Side:        fill.Side,
			PosSide:     fill.PosSide,
			FillPrice:   fillPrice,
			FillQty:     fillSz,
			FillQtyBase: fillQtyBase,
			Fee:         -fee, // OKX returns negative fee
			FeeAsset:    fill.FeeCcy,
			ExecTime:    time.UnixMilli(ts).UTC(),
			IsMaker:     fill.ExecType == "M",
			OrderType:   "MARKET",
			OrderAction: orderAction,
		}

		trades = append(trades, trade)
	}

	return trades, nil
}

// SyncOrdersFromOKX syncs OKX exchange order history to local database
// Also creates/updates position records to ensure orders/fills/positions data consistency
// exchangeID: Exchange account UUID (from exchanges.id)
// exchangeType: Exchange type ("okx")
func (t *OKXTrader) SyncOrdersFromOKX(traderID string, exchangeID string, exchangeType string, st *store.Store) error {
	if st == nil {
		return fmt.Errorf("store is nil")
	}

	// Get recent trades (last 24 hours)
	startTime := time.Now().Add(-24 * time.Hour)

	logger.Infof("🔄 Syncing OKX trades from: %s", startTime.Format(time.RFC3339))

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

	// Process trades one by one (no transaction to avoid deadlock)
	orderStore := st.Order()
	positionStore := st.Position()
	posBuilder := store.NewPositionBuilder(positionStore)
	syncedCount := 0

	for _, trade := range trades {
		// Check if trade already exists (use exchangeID which is UUID, not exchange type)
		existing, err := orderStore.GetOrderByExchangeID(exchangeID, trade.TradeID)
		if err == nil && existing != nil {
			continue // Order already exists, skip
		}

		// Normalize symbol
		symbol := market.Normalize(trade.Symbol)

		// Determine position side from order action
		positionSide := "LONG"
		if strings.Contains(trade.OrderAction, "short") {
			positionSide = "SHORT"
		}

		// Normalize side for storage
		side := strings.ToUpper(trade.Side)

		// Create order record - use UTC time in milliseconds to avoid timezone issues
		execTimeMs := trade.ExecTime.UTC().UnixMilli()
		orderRecord := &store.TraderOrder{
			TraderID:        traderID,
			ExchangeID:      exchangeID,   // UUID
			ExchangeType:    exchangeType, // Exchange type
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

		// Create fill record - use UTC time in milliseconds
		fillRecord := &store.TraderFill{
			TraderID:        traderID,
			ExchangeID:      exchangeID,   // UUID
			ExchangeType:    exchangeType, // Exchange type
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
			RealizedPnL:     0, // OKX fills don't include PnL per trade
			IsMaker:         trade.IsMaker,
			CreatedAt:       execTimeMs,
		}

		if err := orderStore.CreateFill(fillRecord); err != nil {
			logger.Infof("  ⚠️ Failed to sync fill for trade %s: %v", trade.TradeID, err)
		}

		// Create/update position record using PositionBuilder
		logger.Infof("  🔄 Processing position for trade: %s | Symbol: %s | Side: %s | Action: %s | Qty: %.6f | Price: %.6f",
			trade.TradeID, symbol, positionSide, trade.OrderAction, trade.FillQtyBase, trade.FillPrice)

		// Debug: Check existing positions before processing
		existingBefore, _ := st.Position().GetOpenPositionBySymbol(traderID, symbol, positionSide)
		if existingBefore != nil {
			logger.Infof("  📊 Existing position before: ID=%d, Qty=%.6f, Entry=%.4f", existingBefore.ID, existingBefore.Quantity, existingBefore.EntryPrice)
		} else {
			logger.Infof("  📊 No existing position before processing")
		}

		// Process trade with correct parameter order
		// ProcessTrade(traderID, exchangeID, exchangeType, symbol, side, action, quantity, price, fee, realizedPnL, tradeTimeMs, orderID)
		processErr := posBuilder.ProcessTrade(
			traderID,
			exchangeID,
			exchangeType,
			symbol,
			positionSide,
			trade.OrderAction,
			trade.FillQtyBase,
			trade.FillPrice,
			trade.Fee,
			0, // No per-trade PnL from OKX
			execTimeMs,
			trade.TradeID,
		)

		if processErr != nil {
			logger.Errorf("  ❌ Failed to sync position for trade %s: %v", trade.TradeID, processErr)
		} else {
			logger.Infof("  ✅ PositionBuilder.ProcessTrade completed successfully")

			// Verify position was created/updated
			openPos, verifyErr := st.Position().GetOpenPositionBySymbol(traderID, symbol, positionSide)
			if verifyErr == nil && openPos != nil {
				logger.Infof("  ✅ Position verified: ID=%d, Qty=%.6f, Entry=%.4f, Status=%s", openPos.ID, openPos.Quantity, openPos.EntryPrice, openPos.Status)
			} else if verifyErr != nil {
				logger.Warnf("  ⚠️ Position verification failed: %v", verifyErr)
			} else {
				logger.Warnf("  ⚠️ Position not found after creation!")
				// Debug: Check all positions for this trader
				allPositions, _ := st.Position().GetOpenPositions(traderID)
				logger.Infof("  🔍 All open positions for trader: %d positions", len(allPositions))
				for _, pos := range allPositions {
					logger.Infof("     - Symbol: %s, Side: %s, Qty: %.6f, Entry: %.4f", pos.Symbol, pos.Side, pos.Quantity, pos.EntryPrice)
				}
			}
		}

		// 🚨 关键修复：在同步成交后设置TP/SL
		// 检查是否有对应的Pending订单，获取TP/SL信息
		if positionSide == "LONG" || positionSide == "SHORT" {
			// 从Pending订单中查找TP/SL配置
			// 🔍 调试：先检查所有Pending订单
			allPendingOrders, err := st.Analysis().GetPendingOrdersByTrader(traderID)
			logger.Infof("  🔍 Checking for pending orders: trader=%s, symbol=%s, side=%s", traderID, symbol, positionSide)
			logger.Infof("  🔍 Total pending orders found: %d", len(allPendingOrders))

			if err == nil && len(allPendingOrders) > 0 {
				// 打印所有Pending订单用于调试
				for _, po := range allPendingOrders {
					logger.Infof("     - Order: %s | Symbol: %s | Status: %s | Trigger: %.4f | SL: %.4f | TP: %.4f",
						po.ID, po.Symbol, po.Status, po.TriggerPrice, po.StopLoss, po.TakeProfit)
				}

				var pendingOrder *store.PendingOrder
				for _, po := range allPendingOrders {
					// 🔍 修复：只检查符号匹配，不检查状态（因为Order Sync可能已经更新了状态）
					if po.Symbol == symbol {
						// 检查是否有TP/SL配置
						if po.StopLoss > 0 || po.TakeProfit > 0 {
							pendingOrder = po
							logger.Infof("  ✅ Found matching pending order: %s (TP: %.4f, SL: %.4f)", po.ID, po.TakeProfit, po.StopLoss)
							break
						}
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
					} else {
						logger.Infof("  ⚠️ No stop loss configured in pending order")
					}

					// 设置止盈
					if pendingOrder.TakeProfit > 0 {
						tpErr := t.SetTakeProfit(symbol, positionSide, trade.FillQtyBase, pendingOrder.TakeProfit)
						if tpErr != nil {
							logger.Warnf("  ⚠️ Failed to set take profit for %s: %v", symbol, tpErr)
						} else {
							logger.Infof("  ✅ Take profit set for %s: %.4f", symbol, pendingOrder.TakeProfit)
						}
					} else {
						logger.Infof("  ⚠️ No take profit configured in pending order")
					}

					// 记录TP/SL到TPSLRecord表
					if pendingOrder.StopLoss > 0 || pendingOrder.TakeProfit > 0 {
						// 🔧 修复：添加重试机制，确保位置已创建并可查询
						var openPos *store.TraderPosition
						var err error

						// 重试3次，每次间隔100ms
						for retry := 0; retry < 3; retry++ {
							openPos, err = st.Position().GetOpenPositionBySymbol(traderID, symbol, positionSide)
							if err == nil && openPos != nil {
								break
							}
							if retry < 2 {
								time.Sleep(100 * time.Millisecond)
							}
						}

						if err == nil && openPos != nil {
							// 创建TP/SL记录
							tpslRecord := &store.TPSLRecord{
								TraderID:      traderID,
								PositionID:    openPos.ID,
								Symbol:        symbol,
								Side:          positionSide,
								CurrentTP:     pendingOrder.TakeProfit,
								CurrentSL:     pendingOrder.StopLoss,
								OriginalTP:    pendingOrder.TakeProfit,
								OriginalSL:    pendingOrder.StopLoss,
								EntryPrice:    trade.FillPrice,
								EntryQuantity: trade.FillQtyBase,
								Status:        "ACTIVE",
								CreatedAt:     time.Now().UTC(),
								UpdatedAt:     time.Now().UTC(),
							}
							if err := st.TPSL().SaveTPSLRecord(tpslRecord); err != nil {
								logger.Warnf("  ⚠️ Failed to save TPSL record: %v", err)
							} else {
								logger.Infof("  ✅ TPSL record saved for position %d", openPos.ID)
							}
						} else {
							logger.Warnf("  ⚠️ Cannot save TPSL record: position not found after retries (err: %v)", err)
							// 🔧 额外修复：尝试使用最近的OPEN位置
							// 由于ProcessTrade内部调用了CreateOpenPosition，我们尝试获取最近的OPEN位置
							logger.Infof("  🔧 Attempting alternative position lookup...")

							// 获取所有OPEN位置，找到最近创建的
							allOpenPositions, _ := st.Position().GetOpenPositions(traderID)
							var latestPos *store.TraderPosition
							var latestTime int64

							for _, pos := range allOpenPositions {
								if pos.Symbol == symbol && pos.Side == positionSide && pos.Status == "OPEN" {
									if pos.CreatedAt > latestTime {
										latestTime = pos.CreatedAt
										latestPos = pos
									}
								}
							}

							if latestPos != nil {
								logger.Infof("  🔧 Found latest position: ID=%d, CreatedAt=%d", latestPos.ID, latestPos.CreatedAt)
								// 创建TP/SL记录
								tpslRecord := &store.TPSLRecord{
									TraderID:      traderID,
									PositionID:    latestPos.ID,
									Symbol:        symbol,
									Side:          positionSide,
									CurrentTP:     pendingOrder.TakeProfit,
									CurrentSL:     pendingOrder.StopLoss,
									OriginalTP:    pendingOrder.TakeProfit,
									OriginalSL:    pendingOrder.StopLoss,
									EntryPrice:    trade.FillPrice,
									EntryQuantity: trade.FillQtyBase,
									Status:        "ACTIVE",
									CreatedAt:     time.Now().UTC(),
									UpdatedAt:     time.Now().UTC(),
								}
								if err := st.TPSL().SaveTPSLRecord(tpslRecord); err != nil {
									logger.Warnf("  ⚠️ Failed to save TPSL record via alternative method: %v", err)
								} else {
									logger.Infof("  ✅ TPSL record saved via alternative method for position %d", latestPos.ID)
								}
							} else {
								logger.Errorf("  ❌ Cannot find any OPEN position for %s %s, TPSL record NOT saved!", symbol, positionSide)
							}
						}
					}

					// 标记Pending订单为已执行（如果还是PENDING状态）
					if pendingOrder.Status == "PENDING" {
						if err := st.Analysis().UpdatePendingOrderStatus(pendingOrder.ID, "FILLED", trade.FillPrice, time.Now().UTC()); err != nil {
							logger.Warnf("  ⚠️ Failed to update pending order status: %v", err)
						} else {
							logger.Infof("  ✅ Pending order marked as FILLED: %s", pendingOrder.ID)
						}
					} else {
						logger.Infof("  ℹ️ Pending order already in status: %s", pendingOrder.Status)
					}

					logger.Infof("  🎯 TPSL configured for synced trade: %s (TP: %.4f, SL: %.4f)", symbol, pendingOrder.TakeProfit, pendingOrder.StopLoss)
				} else {
					logger.Warnf("  ⚠️ No pending order with TP/SL found for %s", symbol)
					logger.Warnf("  ℹ️ This might be a manual trade or order from different source")
				}
			} else {
				if err != nil {
					logger.Warnf("  ⚠️ Error fetching pending orders: %v", err)
				} else {
					logger.Warnf("  ⚠️ No pending orders found for trader, cannot set TP/SL automatically")
				}
			}
		}

		syncedCount++
		logger.Infof("  ✅ Synced trade: %s %s %s qty=%.6f price=%.6f fee=%.6f action=%s",
			trade.TradeID, trade.Symbol, side, trade.FillQtyBase, trade.FillPrice, trade.Fee, trade.OrderAction)
	}

	logger.Infof("✅ OKX order sync completed: %d new trades synced", syncedCount)
	return nil
}

// StartOrderSync starts background order sync task for OKX
func (t *OKXTrader) StartOrderSync(traderID string, exchangeID string, exchangeType string, st *store.Store, interval time.Duration) {
	ticker := time.NewTicker(interval)
	go func() {
		for range ticker.C {
			if err := t.SyncOrdersFromOKX(traderID, exchangeID, exchangeType, st); err != nil {
				logger.Infof("⚠️  OKX order sync failed: %v", err)
			}
		}
	}()
	logger.Infof("🔄 OKX order sync started (interval: %v)", interval)
}
