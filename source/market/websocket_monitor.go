package market

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"nofx/logger"
	"nofx/provider/coinank"
	"nofx/provider/coinank/coinank_api"
	"nofx/provider/coinank/coinank_enum"
	"nofx/store"
)

// WebSocketPriceMonitor WebSocket实时价格监控器
type WebSocketPriceMonitor struct {
	ws               *coinank_api.KlineWs
	connCtx          context.Context
	connCancel       context.CancelFunc
	subscriptions    map[string]*Subscription // symbol -> subscription
	subMu            sync.RWMutex
	priceCallbacks   map[string][]PriceCallback   // symbol -> callbacks
	triggerCallbacks map[string][]TriggerCallback // order_id -> callbacks
	marketData       *MarketData                  // 共享的市场数据
	store            store.AnalysisStore          // 用于获取待执行订单
	running          bool
	reconnectDelay   time.Duration
	maxReconnect     int
}

// Subscription 订阅信息
type Subscription struct {
	Symbol    string
	Exchange  coinank_enum.Exchange
	Interval  coinank_enum.Interval
	RefCount  int       // 引用计数
	LastPrice float64   // 最新价格
	UpdatedAt time.Time // 最后更新时间
}

// PriceCallback 价格回调函数
type PriceCallback func(symbol string, price float64, timestamp time.Time)

// TriggerCallback 触发回调函数
type TriggerCallback func(order *store.PendingOrder, currentPrice float64)

// MarketData 市场数据结构（复用现有结构）
type MarketData struct {
	Symbol       string  `json:"symbol"`
	CurrentPrice float64 `json:"current_price"`
	High24h      float64 `json:"high_24h"`
	Low24h       float64 `json:"low_24h"`
	Volume24h    float64 `json:"volume_24h"`
	UpdatedAt    int64   `json:"updated_at"`
}

// NewWebSocketPriceMonitor 创建WebSocket价格监控器
func NewWebSocketPriceMonitor(analysisStore store.AnalysisStore) *WebSocketPriceMonitor {
	monitor := &WebSocketPriceMonitor{
		subscriptions:    make(map[string]*Subscription),
		priceCallbacks:   make(map[string][]PriceCallback),
		triggerCallbacks: make(map[string][]TriggerCallback),
		marketData:       &MarketData{},
		store:            analysisStore,
		reconnectDelay:   5 * time.Second,
		maxReconnect:     5,
	}
	return monitor
}

// Start 启动监控器
func (m *WebSocketPriceMonitor) Start() error {
	if m.running {
		return fmt.Errorf("monitor already running")
	}

	m.connCtx, m.connCancel = context.WithCancel(context.Background())

	// 连接WebSocket
	if err := m.connect(); err != nil {
		return fmt.Errorf("failed to connect websocket: %w", err)
	}

	m.running = true
	go m.messageLoop()
	go m.triggerCheckLoop() // 启动触发检查循环

	logger.Info("✅ WebSocket价格监控器已启动")
	return nil
}

// Stop 停止监控器
func (m *WebSocketPriceMonitor) Stop() {
	if !m.running {
		return
	}

	m.running = false
	if m.connCancel != nil {
		m.connCancel()
	}

	if m.ws != nil {
		m.ws.Close()
	}

	logger.Info("🛑 WebSocket价格监控器已停止")
}

// connect 建立WebSocket连接
func (m *WebSocketPriceMonitor) connect() error {
	conn, err := coinank_api.WsConn(m.connCtx, true, false)
	if err != nil {
		return err
	}

	m.ws = conn
	logger.Info("🔗 WebSocket连接已建立")
	return nil
}

// reconnect 重新连接WebSocket
func (m *WebSocketPriceMonitor) reconnect() error {
	logger.Warnf("🔄 尝试重新连接WebSocket (延迟: %v)", m.reconnectDelay)
	time.Sleep(m.reconnectDelay)

	if m.ws != nil {
		m.ws.Close()
	}

	return m.connect()
}

// messageLoop 消息处理循环
func (m *WebSocketPriceMonitor) messageLoop() {
	defer func() {
		if r := recover(); r != nil {
			logger.Errorf("💥 MessageLoop panic recovered: %v", r)
		}
	}()

	for m.running {
		select {
		case <-m.connCtx.Done():
			return
		case kline, ok := <-m.ws.KlineCh:
			if !ok {
				logger.Warn("⚠️ Kline channel closed, attempting reconnect...")
				if err := m.reconnect(); err != nil {
					logger.Errorf("❌ Reconnect failed: %v", err)
					return
				}
				continue
			}

			if kline != nil && kline.Success {
				m.processKline(kline)
			}
		}
	}
}

// processKline 处理K线数据
func (m *WebSocketPriceMonitor) processKline(kline *coinank_api.WsResult[coinank.KlineResult]) {
	// 解析symbol和价格
	symbol := parseSymbolFromArgs(kline.Args)
	price := kline.Data.Close
	timestamp := time.Unix(kline.Data.EndTime/1000, 0)

	// 更新订阅信息
	m.subMu.Lock()
	if sub, exists := m.subscriptions[symbol]; exists {
		sub.LastPrice = price
		sub.UpdatedAt = time.Now()
		m.subscriptions[symbol] = sub

		// 更新共享市场数据
		m.updateMarketData(symbol, price)
	}
	m.subMu.Unlock()

	// 触发价格回调
	m.triggerPriceCallbacks(symbol, price, timestamp)
}

// updateMarketData 更新市场数据
func (m *WebSocketPriceMonitor) updateMarketData(symbol string, price float64) {
	m.marketData.Symbol = symbol
	m.marketData.CurrentPrice = price
	m.marketData.UpdatedAt = time.Now().Unix()

	// 更新到全局市场数据缓存
	marketDataCache.Store(symbol, &MarketData{
		Symbol:       symbol,
		CurrentPrice: price,
		UpdatedAt:    time.Now().Unix(),
	})
}

// triggerPriceCallbacks 触发价格回调
func (m *WebSocketPriceMonitor) triggerPriceCallbacks(symbol string, price float64, timestamp time.Time) {
	m.subMu.RLock()
	defer m.subMu.RUnlock()

	// 触发该币种的所有价格回调
	if callbacks, exists := m.priceCallbacks[symbol]; exists {
		for _, cb := range callbacks {
			go cb(symbol, price, timestamp)
		}
	}
}

// triggerCheckLoop 触发条件检查循环（毫秒级）
func (m *WebSocketPriceMonitor) triggerCheckLoop() {
	ticker := time.NewTicker(100 * time.Millisecond) // 100ms检查一次
	defer ticker.Stop()

	for m.running {
		select {
		case <-m.connCtx.Done():
			return
		case <-ticker.C:
			m.checkTriggerConditions()
		}
	}
}

// checkTriggerConditions 检查所有待执行订单的触发条件
func (m *WebSocketPriceMonitor) checkTriggerConditions() {
	if m.store == nil {
		return
	}

	// 获取所有PENDING状态的订单
	pendingOrders, err := m.store.GetPendingOrdersByStatus("", "PENDING")
	if err != nil {
		logger.Errorf("❌ Failed to get pending orders: %v", err)
		return
	}

	if len(pendingOrders) == 0 {
		return
	}

	// 按币种分组
	ordersBySymbol := make(map[string][]*store.PendingOrder)
	for _, order := range pendingOrders {
		ordersBySymbol[order.Symbol] = append(ordersBySymbol[order.Symbol], order)
	}

	// 检查每个币种的订单
	for symbol, orders := range ordersBySymbol {
		// 获取最新价格
		m.subMu.RLock()
		sub, exists := m.subscriptions[symbol]
		m.subMu.RUnlock()

		if !exists || sub.LastPrice == 0 {
			continue
		}

		currentPrice := sub.LastPrice
		updatedAt := sub.UpdatedAt

		// 检查该币种所有订单的触发条件
		for _, order := range orders {
			if m.checkOrderTrigger(order, currentPrice, updatedAt) {
				// 触发订单执行
				m.triggerOrder(order, currentPrice)
			}
		}
	}
}

// checkOrderTrigger 检查单个订单是否触发
func (m *WebSocketPriceMonitor) checkOrderTrigger(order *store.PendingOrder, currentPrice float64, updatedAt time.Time) bool {
	// 订单年龄检查（超过12小时自动取消）
	orderAge := time.Since(order.CreatedAt)
	if orderAge > 12*time.Hour {
		logger.Infof("🗑️ 订单过期自动取消: %s (%.1fh old)", order.Symbol, orderAge.Hours())
		m.store.CancelPendingOrder(order.ID, fmt.Sprintf("Expired: %.1fh", orderAge.Hours()))
		return false
	}

	// 价格偏离检查（超过15%自动取消）
	isLong := order.StopLoss < order.TakeProfit
	var deviation float64
	if isLong {
		deviation = (currentPrice - order.TriggerPrice) / order.TriggerPrice
	} else {
		deviation = (order.TriggerPrice - currentPrice) / order.TriggerPrice
	}
	if deviation < 0 {
		deviation = -deviation
	}

	if deviation > 0.15 {
		logger.Infof("🗑️ 订单偏离过大取消: %s (%.2f%%)", order.Symbol, deviation*100)
		m.store.CancelPendingOrder(order.ID, fmt.Sprintf("Deviation: %.2f%%", deviation*100))
		return false
	}

	// 触发条件判断（毫秒级）
	// 做多(LONG)：当前价格 >= 触发价
	// 做空(SHORT)：当前价格 <= 触发价
	if isLong && currentPrice >= order.TriggerPrice {
		return true
	}
	if !isLong && currentPrice <= order.TriggerPrice {
		return true
	}

	return false
}

// triggerOrder 触发订单执行
func (m *WebSocketPriceMonitor) triggerOrder(order *store.PendingOrder, currentPrice float64) {
	// 触发触发回调
	m.subMu.RLock()
	callbacks, exists := m.triggerCallbacks[order.ID]
	m.subMu.RUnlock()

	if exists {
		for _, cb := range callbacks {
			go cb(order, currentPrice)
		}
	}

	// 更新订单状态为TRIGGERED
	if err := m.store.UpdatePendingOrderStatus(
		order.ID, "TRIGGERED", currentPrice, time.Now().UTC(),
	); err != nil {
		logger.Errorf("❌ Failed to update order status to TRIGGERED: %v", err)
	} else {
		logger.Infof("🎯 订单触发: %s [%.2f] at %.2f", order.Symbol, order.TriggerPrice, currentPrice)
	}
}

// Subscribe 订阅币种价格
func (m *WebSocketPriceMonitor) Subscribe(symbol string, exchange coinank_enum.Exchange, interval coinank_enum.Interval) error {
	m.subMu.Lock()
	defer m.subMu.Unlock()

	key := symbol

	// 检查是否已订阅
	if sub, exists := m.subscriptions[key]; exists {
		sub.RefCount++
		m.subscriptions[key] = sub
		logger.Infof("📈 增加引用计数: %s (当前: %d)", symbol, sub.RefCount)
		return nil
	}

	// 创建新订阅
	sub := &Subscription{
		Symbol:    symbol,
		Exchange:  exchange,
		Interval:  interval,
		RefCount:  1,
		UpdatedAt: time.Now(),
	}
	m.subscriptions[key] = sub

	// 发送订阅请求
	if m.ws != nil {
		err := m.ws.Subscribe(symbol, exchange, interval)
		if err != nil {
			delete(m.subscriptions, key)
			return fmt.Errorf("failed to subscribe: %w", err)
		}
		logger.Infof("✅ 订阅成功: %s@%s", symbol, exchange)
	}

	return nil
}

// Unsubscribe 取消订阅
func (m *WebSocketPriceMonitor) Unsubscribe(symbol string, exchange coinank_enum.Exchange, interval coinank_enum.Interval) error {
	m.subMu.Lock()
	defer m.subMu.Unlock()

	key := symbol

	sub, exists := m.subscriptions[key]
	if !exists {
		return nil
	}

	sub.RefCount--
	if sub.RefCount <= 0 {
		// 引用计数为0，真正取消订阅
		if m.ws != nil {
			err := m.ws.UnSubscribe(symbol, exchange, interval)
			if err != nil {
				logger.Warnf("⚠️ Failed to unsubscribe: %v", err)
			}
		}
		delete(m.subscriptions, key)
		delete(m.priceCallbacks, key)
		logger.Infof("✅ 取消订阅: %s", symbol)
	} else {
		m.subscriptions[key] = sub
		logger.Infof("📉 减少引用计数: %s (剩余: %d)", symbol, sub.RefCount)
	}

	return nil
}

// RegisterPriceCallback 注册价格回调
func (m *WebSocketPriceMonitor) RegisterPriceCallback(symbol string, callback PriceCallback) {
	m.subMu.Lock()
	defer m.subMu.Unlock()

	m.priceCallbacks[symbol] = append(m.priceCallbacks[symbol], callback)
}

// RegisterTriggerCallback 注册触发回调
func (m *WebSocketPriceMonitor) RegisterTriggerCallback(orderID string, callback TriggerCallback) {
	m.subMu.Lock()
	defer m.subMu.Unlock()

	m.triggerCallbacks[orderID] = append(m.triggerCallbacks[orderID], callback)
}

// GetPrice 获取当前价格（非阻塞）
func (m *WebSocketPriceMonitor) GetPrice(symbol string) (float64, bool) {
	m.subMu.RLock()
	defer m.subMu.RUnlock()

	sub, exists := m.subscriptions[symbol]
	if !exists {
		return 0, false
	}

	// 检查数据新鲜度（超过30秒认为过期）
	if time.Since(sub.UpdatedAt) > 30*time.Second {
		return sub.LastPrice, false
	}

	return sub.LastPrice, true
}

// GetMarketData 获取市场数据
func (m *WebSocketPriceMonitor) GetMarketData(symbol string) *MarketData {
	if data, ok := marketDataCache.Load(symbol); ok {
		return data.(*MarketData)
	}
	return nil
}

// parseSymbolFromArgs 从订阅参数解析symbol
func parseSymbolFromArgs(args string) string {
	// 格式: "kline@BTC@coinank@1m"
	parts := strings.Split(args, "@")
	if len(parts) >= 2 {
		return parts[1]
	}
	return ""
}

// 全局市场数据缓存
var marketDataCache = &sync.Map{}

// AutoTrader集成函数
// UpdatePendingOrdersWithWebSocket 使用WebSocket更新待执行订单
func (m *WebSocketPriceMonitor) UpdatePendingOrdersWithWebSocket() error {
	if m.store == nil {
		return fmt.Errorf("store not initialized")
	}

	// 获取所有PENDING状态的订单
	pendingOrders, err := m.store.GetPendingOrdersByStatus("", "PENDING")
	if err != nil {
		return err
	}

	if len(pendingOrders) == 0 {
		return nil
	}

	// 订阅所有需要监控的币种
	symbolSet := make(map[string]bool)
	for _, order := range pendingOrders {
		symbolSet[order.Symbol] = true
	}

	// 批量订阅
	for symbol := range symbolSet {
		// 使用默认配置订阅
		if err := m.Subscribe(symbol, coinank_enum.Okex, coinank_enum.Minute1); err != nil {
			logger.Warnf("⚠️ Failed to subscribe %s: %v", symbol, err)
		}
	}

	logger.Infof("📊 WebSocket监控 %d 个币种的 %d 个待执行订单", len(symbolSet), len(pendingOrders))
	return nil
}

// GetSubscriptionStats 获取订阅统计
func (m *WebSocketPriceMonitor) GetSubscriptionStats() map[string]interface{} {
	m.subMu.RLock()
	defer m.subMu.RUnlock()

	stats := make(map[string]interface{})
	stats["total_subscriptions"] = len(m.subscriptions)
	stats["total_callbacks"] = len(m.priceCallbacks) + len(m.triggerCallbacks)

	symbols := make([]string, 0, len(m.subscriptions))
	for _, sub := range m.subscriptions {
		symbols = append(symbols, sub.Symbol)
	}
	stats["symbols"] = symbols

	return stats
}
