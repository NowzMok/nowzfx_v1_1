-- 插入测试交易数据用于反思系统演示
-- 这些数据将用于生成反思记录和调整建议

-- 插入测试交易历史数据（过去7天）
INSERT INTO trade_history (trader_id, symbol, entry_price, exit_price, position_size, pnl, success, created_at) VALUES 
('b98dff1e_5665133a-c749-421d-8ade-5fb2a0f960c3_deepseek_1768196860', 'BTCUSDT', 45000, 46000, 100, 1000, 1, datetime('now', '-6 days 2 hours')),
('b98dff1e_5665133a-c749-421d-8ade-5fb2a0f960c3_deepseek_1768196860', 'ETHUSDT', 3000, 2950, 50, -250, 0, datetime('now', '-5 days 23 hours')),
('b98dff1e_5665133a-c749-421d-8ade-5fb2a0f960c3_deepseek_1768196860', 'BTCUSDT', 46000, 45500, 100, -500, 0, datetime('now', '-4 days 20 hours')),
('b98dff1e_5665133a-c749-421d-8ade-5fb2a0f960c3_deepseek_1768196860', 'ETHUSDT', 2950, 3100, 50, 750, 1, datetime('now', '-3 days 18 hours')),
('b98dff1e_5665133a-c749-421d-8ade-5fb2a0f960c3_deepseek_1768196860', 'BTCUSDT', 45500, 47000, 100, 1500, 1, datetime('now', '-2 days 15 hours')),
('b98dff1e_5665133a-c749-421d-8ade-5fb2a0f960c3_deepseek_1768196860', 'ETHUSDT', 3100, 3050, 50, -250, 0, datetime('now', '-1 day 12 hours')),
('b98dff1e_5665133a-c749-421d-8ade-5fb2a0f960c3_deepseek_1768196860', 'BTCUSDT', 47000, 46500, 100, -500, 0, datetime('now', '-1 day 2 hours')),
('b98dff1e_5665133a-c749-421d-8ade-5fb2a0f960c3_deepseek_1768196860', 'ETHUSDT', 3050, 3200, 50, 750, 1, datetime('now', '-20 hours')),
('b98dff1e_5665133a-c749-421d-8ade-5fb2a0f960c3_deepseek_1768196860', 'BTCUSDT', 46500, 48000, 100, 1500, 1, datetime('now', '-15 hours')),
('b98dff1e_5665133a-c749-421d-8ade-5fb2a0f960c3_deepseek_1768196860', 'ETHUSDT', 3200, 3150, 50, -250, 0, datetime('now', '-10 hours'));

-- 插入反思记录（用于测试显示功能）
INSERT INTO reflections (id, trader_id, reflection_time, period_start_time, period_end_time, total_trades, successful_trades, failed_trades, success_rate, average_pn_l, max_profit, max_loss, total_pn_l, pn_l_percentage, sharpe_ratio, max_drawdown, win_loss_ratio, confidence_accuracy, symbol_performance, ai_reflection, recommendations, trade_system_advice, ai_learning_advice, created_at, updated_at) VALUES
('ref_001', 'b98dff1e_5665133a-c749-421d-8ade-5fb2a0f960c3_deepseek_1768196860', datetime('now'), datetime('now', '-7 days'), datetime('now'), 10, 5, 5, 0.5, 250, 1500, -500, 2500, 5.5, 1.2, 0.08, 1.0, '{"accuracy": 0.85, "confidence": 0.78}', '{"BTCUSDT": {"win_rate": 0.6, "avg_pnl": 400}, "ETHUSDT": {"win_rate": 0.4, "avg_pnl": 100}}', '过去7天交易表现中等，BTC交易优于ETH。建议关注趋势确认信号，减少逆势交易。', '[{"action": "INCREASE_POSITION_SIZE", "reason": "BTC趋势强劲，胜率60%", "priority": "high"}, {"action": "REDUCE_LEVERAGE", "reason": "ETH波动较大，建议降低风险", "priority": "medium"}]', '{"strategy": "趋势跟随", "adjustment": "增加BTC仓位，减少ETH仓位"}', '学习到BTC在45000-48000区间表现良好，ETH需要更严格的止损', datetime('now'), datetime('now'));

-- 插入系统调整建议
INSERT INTO system_adjustments (id, trader_id, reflection_id, adjustment_time, confidence_level, btceth_leverage, altcoin_leverage, max_position_size, max_daily_loss, stop_loss_pct, take_profit_pct, adjustment_reason, applied_at, status, created_at) VALUES
('adj_001', 'b98dff1e_5665133a-c749-421d-8ade-5fb2a0f960c3_deepseek_1768196860', 'ref_001', datetime('now'), 0.85, 5, 3, 150, 500, 2.0, 4.0, 'BTC趋势强劲，胜率60%，建议增加仓位规模', NULL, 'pending', datetime('now')),
('adj_002', 'b98dff1e_5665133a-c749-421d-8ade-5fb2a0f960c3_deepseek_1768196860', 'ref_001', datetime('now'), 0.72, 5, 2, 120, 500, 2.5, 3.5, 'ETH波动较大，建议降低杠杆和仓位', NULL, 'pending', datetime('now'));

-- 插入AI学习记忆
INSERT INTO ai_learning_memory (id, trader_id, reflection_id, pattern, insight, usage_count, active, created_at, ai_learning_advice, confidence_accuracy) VALUES
('mem_001', 'b98dff1e_5665133a-c749-421d-8ade-5fb2a0f960c3_deepseek_1768196860', 'ref_001', 'BTC_45000_48000_range', 'BTC在45000-48000区间内表现良好，适合趋势交易', 3, 1, datetime('now'), '保持在该区间内使用趋势策略', 0.85),
('mem_002', 'b98dff1e_5665133a-c749-421d-8ade-5fb2a0f960c3_deepseek_1768196860', 'ref_001', 'ETH_high_volatility', 'ETH波动性明显高于BTC，需要更严格的止损', 2, 1, datetime('now'), 'ETH交易使用2.5%止损而非2%', 0.78);

-- 插入调度器记录（如果不存在）
INSERT OR IGNORE INTO reflection_schedules (trader_id, next_run, schedule_hour, schedule_minute, schedule_day) VALUES 
('b98dff1e_5665133a-c749-421d-8ade-5fb2a0f960c3_deepseek_1768196860', datetime('now', '+6 days', 'start of day', '+22 hours'), 22, 0, 0);
