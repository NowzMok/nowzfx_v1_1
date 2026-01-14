package trader

import (
	"fmt"
	"math"
	"nofx/kernel"
	"nofx/logger"
)

// VerifyTPSLSync 验证止损止盈与交易所的同步状态
// 确保数据库记录与交易所实际订单一致
func (at *AutoTrader) VerifyTPSLSync(ctx *kernel.Context) error {
	if at.trader == nil || at.store == nil {
		return fmt.Errorf("trader or store not initialized")
	}

	// 获取所有开放仓位（使用ctx.Positions而不是ExchangeInstance）
	if len(ctx.Positions) == 0 {
		return nil
	}

	logger.Debugf("🔍 [TPSLSync] Checking %d positions for TP/SL sync...", len(ctx.Positions))

	syncErrors := 0
	syncSuccess := 0

	for _, pos := range ctx.Positions {
		// 从数据库获取TP/SL记录
		tpslRecord, err := at.store.TPSL().GetTPSLBySymbolAndTrader(at.id, pos.Symbol)
		if err != nil || len(tpslRecord) == 0 {
			logger.Debugf("  No TP/SL record for %s in database", pos.Symbol)
			continue
		}

		// 使用最新的记录（按创建时间排序，取第一个）
		record := tpslRecord[0]
		if record.Status != "ACTIVE" {
			continue
		}

		// 从交易所获取实际的TP/SL订单
		orders, err := at.trader.GetOpenOrders(pos.Symbol)
		if err != nil {
			logger.Warnf("  ⚠️ Failed to get orders for %s: %v", pos.Symbol, err)
			if at.errorTracker != nil {
				at.errorTracker.RecordError(
					"SYNC_GET_ORDERS_FAILED",
					pos.Symbol,
					fmt.Sprintf("Failed to retrieve orders: %v", err),
					"WARN",
				)
			}
			syncErrors++
			continue
		}

		// 检查止损订单
		slOrderFound := false
		tpOrderFound := false
		slNeedUpdate := false
		tpNeedUpdate := false

		for _, order := range orders {
			orderType := order.Type
			orderPrice := order.Price

			// 检查止损订单
			if orderType == "STOP_LOSS" || orderType == "STOP_MARKET" || orderType == "STOP" {
				slOrderFound = true
				// 比较价格（允许0.01%的误差）
				priceDiff := math.Abs(orderPrice-record.CurrentSL) / record.CurrentSL
				if priceDiff > 0.0001 { // 0.01%
					logger.Warnf("  ⚠️ SL price mismatch for %s: DB=%.6f, Exchange=%.6f (%.2f%% diff)",
						pos.Symbol, record.CurrentSL, orderPrice, priceDiff*100)
					slNeedUpdate = true
				}
			}

			// 检查止盈订单
			if orderType == "TAKE_PROFIT" || orderType == "TAKE_PROFIT_MARKET" || orderType == "LIMIT" {
				tpOrderFound = true
				// 比较价格
				priceDiff := math.Abs(orderPrice-record.CurrentTP) / record.CurrentTP
				if priceDiff > 0.0001 {
					logger.Warnf("  ⚠️ TP price mismatch for %s: DB=%.6f, Exchange=%.6f (%.2f%% diff)",
						pos.Symbol, record.CurrentTP, orderPrice, priceDiff*100)
					tpNeedUpdate = true
				}
			}
		}

		// 如果找不到订单，可能需要重新设置
		if !slOrderFound {
			logger.Warnf("  ⚠️ Stop Loss order not found on exchange for %s", pos.Symbol)
			if at.errorTracker != nil {
				at.errorTracker.RecordError(
					"SYNC_SL_MISSING",
					pos.Symbol,
					"Stop Loss order not found on exchange",
					"WARN",
				)
			}
			slNeedUpdate = true
		}

		if !tpOrderFound {
			logger.Warnf("  ⚠️ Take Profit order not found on exchange for %s", pos.Symbol)
			if at.errorTracker != nil {
				at.errorTracker.RecordError(
					"SYNC_TP_MISSING",
					pos.Symbol,
					"Take Profit order not found on exchange",
					"WARN",
				)
			}
			tpNeedUpdate = true
		}

		// 尝试同步（如果需要）
		if slNeedUpdate || tpNeedUpdate {
			logger.Infof("  🔄 Attempting to sync TP/SL for %s...", pos.Symbol)

			// 重新设置止损
			if slNeedUpdate && record.CurrentSL > 0 {
				side := "LONG"
				if pos.Side == "short" {
					side = "SHORT"
				}

				err := at.trader.SetStopLoss(pos.Symbol, side, pos.Quantity, record.CurrentSL)
				if err != nil {
					logger.Errorf("  ❌ Failed to sync SL for %s: %v", pos.Symbol, err)
					if at.errorTracker != nil {
						at.errorTracker.RecordError(
							"SYNC_SL_UPDATE_FAILED",
							pos.Symbol,
							fmt.Sprintf("Failed to update Stop Loss: %v", err),
							"ERROR",
						)
					}
					syncErrors++
				} else {
					logger.Infof("  ✅ SL synced successfully for %s: %.6f", pos.Symbol, record.CurrentSL)
					if at.errorTracker != nil {
						at.errorTracker.RecordError(
							"SYNC_SL_SUCCESS",
							pos.Symbol,
							fmt.Sprintf("Stop Loss synced: %.6f", record.CurrentSL),
							"INFO",
						)
					}
					syncSuccess++
				}
			}

			// 重新设置止盈
			if tpNeedUpdate && record.CurrentTP > 0 {
				side := "LONG"
				if pos.Side == "short" {
					side = "SHORT"
				}

				err := at.trader.SetTakeProfit(pos.Symbol, side, pos.Quantity, record.CurrentTP)
				if err != nil {
					logger.Errorf("  ❌ Failed to sync TP for %s: %v", pos.Symbol, err)
					if at.errorTracker != nil {
						at.errorTracker.RecordError(
							"SYNC_TP_UPDATE_FAILED",
							pos.Symbol,
							fmt.Sprintf("Failed to update Take Profit: %v", err),
							"ERROR",
						)
					}
					syncErrors++
				} else {
					logger.Infof("  ✅ TP synced successfully for %s: %.6f", pos.Symbol, record.CurrentTP)
					if at.errorTracker != nil {
						at.errorTracker.RecordError(
							"SYNC_TP_SUCCESS",
							pos.Symbol,
							fmt.Sprintf("Take Profit synced: %.6f", record.CurrentTP),
							"INFO",
						)
					}
					syncSuccess++
				}
			}
		}
	}

	if syncErrors > 0 {
		logger.Warnf("⚠️ [TPSLSync] Completed with %d errors, %d successful syncs", syncErrors, syncSuccess)
	} else if syncSuccess > 0 {
		logger.Infof("✅ [TPSLSync] All positions synced successfully (%d updates)", syncSuccess)
	} else {
		logger.Debugf("✅ [TPSLSync] All positions already in sync")
	}

	return nil
}
