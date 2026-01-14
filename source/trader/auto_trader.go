package trader

import (
	"encoding/json"
	"fmt"
	"math"
	"nofx/experience"
	"nofx/kernel"
	"nofx/logger"
	"nofx/market"
	"nofx/mcp"
	"nofx/store"
	"strings"
	"sync"
	"time"
)

// AutoTraderConfig auto trading configuration (simplified version - AI makes all decisions)
type AutoTraderConfig struct {
	// Trader identification
	ID      string // Trader unique identifier (for log directory, etc.)
	Name    string // Trader display name
	AIModel string // AI model: "qwen" or "deepseek"

	// Trading platform selection
	Exchange   string // Exchange type: "binance", "bybit", "okx", "bitget", "hyperliquid", "aster" or "lighter"
	ExchangeID string // Exchange account UUID (for multi-account support)

	// Binance API configuration
	BinanceAPIKey    string
	BinanceSecretKey string

	// Bybit API configuration
	BybitAPIKey    string
	BybitSecretKey string

	// OKX API configuration
	OKXAPIKey     string
	OKXSecretKey  string
	OKXPassphrase string

	// Bitget API configuration
	BitgetAPIKey     string
	BitgetSecretKey  string
	BitgetPassphrase string

	// Hyperliquid configuration
	HyperliquidPrivateKey string
	HyperliquidWalletAddr string
	HyperliquidTestnet    bool

	// Aster configuration
	AsterUser       string // Aster main wallet address
	AsterSigner     string // Aster API wallet address
	AsterPrivateKey string // Aster API wallet private key

	// LIGHTER configuration
	LighterWalletAddr       string // LIGHTER wallet address (L1 wallet)
	LighterPrivateKey       string // LIGHTER L1 private key (for account identification)
	LighterAPIKeyPrivateKey string // LIGHTER API Key private key (40 bytes, for transaction signing)
	LighterAPIKeyIndex      int    // LIGHTER API Key index (0-255)
	LighterTestnet          bool   // Whether to use testnet

	// AI configuration
	UseQwen     bool
	DeepSeekKey string
	QwenKey     string

	// Custom AI API configuration
	CustomAPIURL    string
	CustomAPIKey    string
	CustomModelName string

	// Scan configuration
	ScanInterval time.Duration // Scan interval (recommended 3 minutes)

	// Account configuration
	InitialBalance float64 // Initial balance (for P&L calculation, must be set manually)

	// Risk control (only as hints, AI can make autonomous decisions)
	MaxDailyLoss    float64       // Maximum daily loss percentage (hint)
	MaxDrawdown     float64       // Maximum drawdown percentage (hint)
	StopTradingTime time.Duration // Pause duration after risk control triggers

	// Position mode
	IsCrossMargin bool // true=cross margin mode, false=isolated margin mode

	// Competition visibility
	ShowInCompetition bool // Whether to show in competition page

	// Strategy configuration (use complete strategy config)
	StrategyConfig *store.StrategyConfig // Strategy configuration (includes coin sources, indicators, risk control, prompts, etc.)
}

// AutoTrader automatic trader
type AutoTrader struct {
	id                         string // Trader unique identifier
	name                       string // Trader display name
	aiModel                    string // AI model name
	exchange                   string // Trading platform type (binance/bybit/etc)
	exchangeID                 string // Exchange account UUID
	showInCompetition          bool   // Whether to show in competition page
	config                     AutoTraderConfig
	trader                     Trader // Use Trader interface (supports multiple platforms)
	mcpClient                  mcp.AIClient
	store                      *store.Store           // Data storage (decision records, etc.)
	strategyEngine             *kernel.StrategyEngine // Strategy engine (uses strategy configuration)
	cycleNumber                int                    // Current cycle number
	initialBalance             float64
	dailyPnL                   float64
	customPrompt               string // Custom trading strategy prompt
	overrideBasePrompt         bool   // Whether to override base prompt
	lastResetTime              time.Time
	stopUntil                  time.Time
	isRunning                  bool
	isRunningMutex             sync.RWMutex       // Mutex to protect isRunning flag
	startTime                  time.Time          // System start time
	callCount                  int                // AI call count
	positionFirstSeenTime      map[string]int64   // Position first seen time (symbol_side -> timestamp in milliseconds)
	positionFirstSeenTimeMutex sync.RWMutex       // Mutex to protect positionFirstSeenTime map
	stopMonitorCh              chan struct{}      // Used to stop monitoring goroutine
	monitorWg                  sync.WaitGroup     // Used to wait for monitoring goroutine to finish
	peakPnLCache               map[string]float64 // Peak profit cache (symbol -> peak P&L percentage)
	peakPnLCacheMutex          sync.RWMutex       // Cache read-write lock
	lastBalanceSyncTime        time.Time          // Last balance sync time
	userID                     string             // User ID

	// Option B: Enhanced trading modules
	enhancedSetup *EnhancedAutoTraderSetup // All advanced modules (parameter optimizer, risk manager, etc.)

	// Order deduplication manager
	orderDedupManager *OrderDeduplicationManager // Order deduplication and cleanup

	// Pending order retry tracking
	pendingOrderRetries map[string]int // Order ID -> retry count
	cycleCount          int            // Cycle counter for periodic tasks
	mu                  sync.Mutex     // Mutex for pendingOrderRetries

	// Error tracking
	errorTracker *ErrorTracker // Error monitoring and statistics
}

// NewAutoTrader creates an automatic trader
// st parameter is used to store decision records to database
func NewAutoTrader(config AutoTraderConfig, st *store.Store, userID string) (*AutoTrader, error) {
	// Set default values
	if config.ID == "" {
		config.ID = "default_trader"
	}
	if config.Name == "" {
		config.Name = "Default Trader"
	}
	if config.AIModel == "" {
		if config.UseQwen {
			config.AIModel = "qwen"
		} else {
			config.AIModel = "deepseek"
		}
	}

	// Initialize AI client based on provider
	var mcpClient mcp.AIClient
	aiModel := config.AIModel
	if config.UseQwen && aiModel == "" {
		aiModel = "qwen"
	}

	switch aiModel {
	case "claude":
		mcpClient = mcp.NewClaudeClient()
		mcpClient.SetAPIKey(config.CustomAPIKey, config.CustomAPIURL, config.CustomModelName)
		logger.Infof("🤖 [%s] Using Claude AI", config.Name)

	case "kimi":
		mcpClient = mcp.NewKimiClient()
		mcpClient.SetAPIKey(config.CustomAPIKey, config.CustomAPIURL, config.CustomModelName)
		logger.Infof("🤖 [%s] Using Kimi (Moonshot) AI", config.Name)

	case "gemini":
		mcpClient = mcp.NewGeminiClient()
		mcpClient.SetAPIKey(config.CustomAPIKey, config.CustomAPIURL, config.CustomModelName)
		logger.Infof("🤖 [%s] Using Google Gemini AI", config.Name)

	case "grok":
		mcpClient = mcp.NewGrokClient()
		mcpClient.SetAPIKey(config.CustomAPIKey, config.CustomAPIURL, config.CustomModelName)
		logger.Infof("🤖 [%s] Using xAI Grok AI", config.Name)

	case "openai":
		mcpClient = mcp.NewOpenAIClient()
		mcpClient.SetAPIKey(config.CustomAPIKey, config.CustomAPIURL, config.CustomModelName)
		logger.Infof("🤖 [%s] Using OpenAI", config.Name)

	case "qwen":
		mcpClient = mcp.NewQwenClient()
		apiKey := config.QwenKey
		if apiKey == "" {
			apiKey = config.CustomAPIKey
		}
		mcpClient.SetAPIKey(apiKey, config.CustomAPIURL, config.CustomModelName)
		logger.Infof("🤖 [%s] Using Alibaba Cloud Qwen AI", config.Name)

	case "custom":
		mcpClient = mcp.New()
		mcpClient.SetAPIKey(config.CustomAPIKey, config.CustomAPIURL, config.CustomModelName)
		logger.Infof("🤖 [%s] Using custom AI API: %s (model: %s)", config.Name, config.CustomAPIURL, config.CustomModelName)

	default: // deepseek or empty
		mcpClient = mcp.NewDeepSeekClient()
		apiKey := config.DeepSeekKey
		if apiKey == "" {
			apiKey = config.CustomAPIKey
		}
		mcpClient.SetAPIKey(apiKey, config.CustomAPIURL, config.CustomModelName)
		logger.Infof("🤖 [%s] Using DeepSeek AI", config.Name)
	}

	if config.CustomAPIURL != "" || config.CustomModelName != "" {
		logger.Infof("🔧 [%s] Custom config - URL: %s, Model: %s", config.Name, config.CustomAPIURL, config.CustomModelName)
	}

	// Set default trading platform
	if config.Exchange == "" {
		config.Exchange = "binance"
	}

	// Create corresponding trader based on configuration
	var trader Trader
	var err error

	// Record position mode (general)
	marginModeStr := "Cross Margin"
	if !config.IsCrossMargin {
		marginModeStr = "Isolated Margin"
	}
	logger.Infof("📊 [%s] Position mode: %s", config.Name, marginModeStr)

	switch config.Exchange {
	case "binance":
		logger.Infof("🏦 [%s] Using Binance Futures trading", config.Name)
		trader = NewFuturesTrader(config.BinanceAPIKey, config.BinanceSecretKey, userID)
	case "bybit":
		logger.Infof("🏦 [%s] Using Bybit Futures trading", config.Name)
		trader = NewBybitTrader(config.BybitAPIKey, config.BybitSecretKey)
	case "okx":
		logger.Infof("🏦 [%s] Using OKX Futures trading", config.Name)
		trader = NewOKXTrader(config.OKXAPIKey, config.OKXSecretKey, config.OKXPassphrase)
	case "bitget":
		logger.Infof("🏦 [%s] Using Bitget Futures trading", config.Name)
		trader = NewBitgetTrader(config.BitgetAPIKey, config.BitgetSecretKey, config.BitgetPassphrase)
	case "hyperliquid":
		logger.Infof("🏦 [%s] Using Hyperliquid trading", config.Name)
		trader, err = NewHyperliquidTrader(config.HyperliquidPrivateKey, config.HyperliquidWalletAddr, config.HyperliquidTestnet)
		if err != nil {
			return nil, fmt.Errorf("failed to initialize Hyperliquid trader: %w", err)
		}
	case "aster":
		logger.Infof("🏦 [%s] Using Aster trading", config.Name)
		trader, err = NewAsterTrader(config.AsterUser, config.AsterSigner, config.AsterPrivateKey)
		if err != nil {
			return nil, fmt.Errorf("failed to initialize Aster trader: %w", err)
		}
	case "lighter":
		logger.Infof("🏦 [%s] Using LIGHTER trading", config.Name)

		if config.LighterWalletAddr == "" || config.LighterAPIKeyPrivateKey == "" {
			return nil, fmt.Errorf("Lighter requires wallet address and API Key private key")
		}

		// Lighter only supports mainnet (testnet disabled)
		trader, err = NewLighterTraderV2(
			config.LighterWalletAddr,
			config.LighterAPIKeyPrivateKey,
			config.LighterAPIKeyIndex,
			false, // Always use mainnet for Lighter
		)
		if err != nil {
			return nil, fmt.Errorf("failed to initialize LIGHTER trader: %w", err)
		}
		logger.Infof("✓ LIGHTER trader initialized successfully")
	default:
		return nil, fmt.Errorf("unsupported trading platform: %s", config.Exchange)
	}

	// Validate initial balance configuration, auto-fetch from exchange if 0
	if config.InitialBalance <= 0 {
		logger.Infof("📊 [%s] Initial balance not set, attempting to fetch current balance from exchange...", config.Name)
		account, err := trader.GetBalance()
		if err != nil {
			return nil, fmt.Errorf("initial balance not set and unable to fetch balance from exchange: %w", err)
		}
		// Try multiple balance field names (different exchanges return different formats)
		balanceKeys := []string{"total_equity", "totalWalletBalance", "wallet_balance", "totalEq", "balance"}
		var foundBalance float64
		for _, key := range balanceKeys {
			if balance, ok := account[key].(float64); ok && balance > 0 {
				foundBalance = balance
				break
			}
		}
		if foundBalance > 0 {
			config.InitialBalance = foundBalance
			logger.Infof("✓ [%s] Auto-fetched initial balance: %.2f USDT", config.Name, foundBalance)
			// Save to database so it persists across restarts
			if st != nil {
				if err := st.Trader().UpdateInitialBalance(userID, config.ID, foundBalance); err != nil {
					logger.Infof("⚠️  [%s] Failed to save initial balance to database: %v", config.Name, err)
				} else {
					logger.Infof("✓ [%s] Initial balance saved to database", config.Name)
				}
			}
		} else {
			return nil, fmt.Errorf("initial balance must be greater than 0, please set InitialBalance in config or ensure exchange account has balance")
		}
	}

	// Get last cycle number (for recovery)
	var cycleNumber int
	if st != nil {
		cycleNumber, _ = st.Decision().GetLastCycleNumber(config.ID)
		logger.Infof("📊 [%s] Decision records will be stored to database", config.Name)
	}

	// Create strategy engine (must have strategy config)
	if config.StrategyConfig == nil {
		return nil, fmt.Errorf("[%s] strategy not configured", config.Name)
	}
	strategyEngine := kernel.NewStrategyEngine(config.StrategyConfig)
	logger.Infof("✓ [%s] Using strategy engine (strategy configuration loaded)", config.Name)

	// Option B: Initialize enhanced trading modules
	var enhancedSetup *EnhancedAutoTraderSetup
	if st != nil {
		enhancedSetup = InitializeEnhancedModules(config.ID, config.InitialBalance, *st)
		logger.Infof("✓ [%s] Enhanced trading modules initialized (Option B)", config.Name)
	}

	// Initialize order deduplication manager
	var orderDedupManager *OrderDeduplicationManager
	if st != nil {
		orderDedupManager = NewOrderDeduplicationManager(config.ID, st)
		logger.Infof("✓ [%s] Order deduplication manager initialized", config.Name)
	}

	// Initialize error tracker
	errorTracker := NewErrorTracker(100) // Keep last 100 errors
	logger.Infof("✓ [%s] Error tracker initialized", config.Name)

	return &AutoTrader{
		id:                         config.ID,
		name:                       config.Name,
		aiModel:                    config.AIModel,
		exchange:                   config.Exchange,
		exchangeID:                 config.ExchangeID,
		showInCompetition:          config.ShowInCompetition,
		config:                     config,
		trader:                     trader,
		mcpClient:                  mcpClient,
		store:                      st,
		strategyEngine:             strategyEngine,
		enhancedSetup:              enhancedSetup,
		orderDedupManager:          orderDedupManager,
		errorTracker:               errorTracker,
		cycleNumber:                cycleNumber,
		initialBalance:             config.InitialBalance,
		lastResetTime:              time.Now(),
		startTime:                  time.Now(),
		callCount:                  0,
		isRunning:                  false,
		positionFirstSeenTime:      make(map[string]int64),
		positionFirstSeenTimeMutex: sync.RWMutex{},
		stopMonitorCh:              make(chan struct{}),
		monitorWg:                  sync.WaitGroup{},
		peakPnLCache:               make(map[string]float64),
		peakPnLCacheMutex:          sync.RWMutex{},
		lastBalanceSyncTime:        time.Now(),
		userID:                     userID,
		pendingOrderRetries:        make(map[string]int),
	}, nil
}

// Run runs the automatic trading main loop
func (at *AutoTrader) Run() error {
	at.isRunningMutex.Lock()
	at.isRunning = true
	at.isRunningMutex.Unlock()

	at.stopMonitorCh = make(chan struct{})
	at.startTime = time.Now()

	logger.Info("🚀 AI-driven automatic trading system started")
	logger.Infof("💰 Initial balance: %.2f USDT", at.initialBalance)
	logger.Infof("⚙️  Scan interval: %v", at.config.ScanInterval)
	logger.Info("🤖 AI will make full decisions on leverage, position size, stop loss/take profit, etc.")
	at.monitorWg.Add(1)
	defer at.monitorWg.Done()

	// Start drawdown monitoring
	at.startDrawdownMonitor()

	// Start Lighter order sync if using Lighter exchange
	if at.exchange == "lighter" {
		if lighterTrader, ok := at.trader.(*LighterTraderV2); ok && at.store != nil {
			lighterTrader.StartOrderSync(at.id, at.exchangeID, at.exchange, at.store, 30*time.Second)
			logger.Infof("🔄 [%s] Lighter order+position sync enabled (every 30s)", at.name)
		}
	}

	// Start Hyperliquid order sync if using Hyperliquid exchange
	if at.exchange == "hyperliquid" {
		if hyperliquidTrader, ok := at.trader.(*HyperliquidTrader); ok && at.store != nil {
			hyperliquidTrader.StartOrderSync(at.id, at.exchangeID, at.exchange, at.store, 30*time.Second)
			logger.Infof("🔄 [%s] Hyperliquid order+position sync enabled (every 30s)", at.name)
		}
	}

	// Start Bybit order sync if using Bybit exchange
	if at.exchange == "bybit" {
		if bybitTrader, ok := at.trader.(*BybitTrader); ok && at.store != nil {
			bybitTrader.StartOrderSync(at.id, at.exchangeID, at.exchange, at.store, 30*time.Second)
			logger.Infof("🔄 [%s] Bybit order+position sync enabled (every 30s)", at.name)
		}
	}

	// Start OKX order sync if using OKX exchange
	if at.exchange == "okx" {
		if okxTrader, ok := at.trader.(*OKXTrader); ok && at.store != nil {
			okxTrader.StartOrderSync(at.id, at.exchangeID, at.exchange, at.store, 30*time.Second)
			logger.Infof("🔄 [%s] OKX order+position sync enabled (every 30s)", at.name)
		}
	}

	// Start Bitget order sync if using Bitget exchange
	if at.exchange == "bitget" {
		if bitgetTrader, ok := at.trader.(*BitgetTrader); ok && at.store != nil {
			bitgetTrader.StartOrderSync(at.id, at.exchangeID, at.exchange, at.store, 30*time.Second)
			logger.Infof("🔄 [%s] Bitget order+position sync enabled (every 30s)", at.name)
		}
	}

	// Start Aster order sync if using Aster exchange
	if at.exchange == "aster" {
		if asterTrader, ok := at.trader.(*AsterTrader); ok && at.store != nil {
			asterTrader.StartOrderSync(at.id, at.exchangeID, at.exchange, at.store, 30*time.Second)
			logger.Infof("🔄 [%s] Aster order+position sync enabled (every 30s)", at.name)
		}
	}

	// Start Binance order sync if using Binance exchange
	if at.exchange == "binance" {
		if binanceTrader, ok := at.trader.(*FuturesTrader); ok && at.store != nil {
			binanceTrader.StartOrderSync(at.id, at.exchangeID, at.exchange, at.store, 30*time.Second)
			logger.Infof("🔄 [%s] Binance order+position sync enabled (every 30s)", at.name)
		}
	}

	// NEW: Start WebSocket real-time monitoring (毫秒级触发)
	at.monitorWg.Add(1)
	go func() {
		defer at.monitorWg.Done()

		// 创建WebSocket监控器
		wsMonitor := market.NewWebSocketPriceMonitor(at.store.Analysis())

		// 启动WebSocket监控器
		if err := wsMonitor.Start(); err != nil {
			logger.Errorf("❌ Failed to start WebSocket monitor: %v", err)
			return
		}

		// 注册触发回调 - 当订单触发时执行交易
		wsMonitor.RegisterTriggerCallback("global", func(order *store.PendingOrder, currentPrice float64) {
			logger.Infof("🎯 WebSocket触发订单: %s @ %.2f (触发价: %.2f)",
				order.Symbol, currentPrice, order.TriggerPrice)

			// 原子地尝试标记为执行中（防止重复执行）
			if !at.store.Analysis().TryMarkAsExecuting(order.ID) {
				logger.Warnf("⚠️ Order already executing or completed: %s (ID: %d)", order.Symbol, order.ID)
				return
			}

			// 执行订单
			if err := at.executePendingOrder(order, currentPrice); err != nil {
				logger.Errorf("❌ WebSocket订单执行失败: %v", err)
				// 记录失败，增加重试计数
				at.recordPendingOrderFailure(order.ID, err)
				// 取消执行标记，允许重试
				if cancelErr := at.store.Analysis().CancelExecution(order.ID); cancelErr != nil {
					logger.Warnf("⚠️ Failed to cancel execution flag: %v", cancelErr)
				}
				return
			}

			// 标记为已执行
			if err := at.store.Analysis().MarkAsExecuted(order.ID); err != nil {
				logger.Warnf("⚠️ Failed to mark order as executed: %v", err)
			} else {
				logger.Infof("✅ WebSocket订单执行成功: %s", order.Symbol)
			}
		})

		// 定期更新WebSocket订阅（处理新订单）
		subscriptionTicker := time.NewTicker(30 * time.Second)
		defer subscriptionTicker.Stop()

		// 定期清理重复订单（每5分钟）
		cleanupTicker := time.NewTicker(5 * time.Minute)
		defer cleanupTicker.Stop()

		logger.Info("🔄 WebSocket real-time monitoring started (毫秒级触发)")

		for {
			select {
			case <-subscriptionTicker.C:
				// 更新WebSocket订阅以包含新订单
				if err := wsMonitor.UpdatePendingOrdersWithWebSocket(); err != nil {
					logger.Warnf("⚠️ Failed to update WebSocket subscriptions: %v", err)
				}

				// 显示订阅统计
				stats := wsMonitor.GetSubscriptionStats()
				logger.Infof("📊 WebSocket stats: %d subscriptions, %d callbacks",
					stats["total_subscriptions"], stats["total_callbacks"])

			case <-cleanupTicker.C:
				// 定期自动清理重复订单
				if at.orderDedupManager != nil {
					startTime := time.Now()
					results, err := at.orderDedupManager.AutoClean()
					duration := time.Since(startTime)

					if err != nil {
						logger.Warnf("⚠️ Auto cleanup failed: %v", err)
					} else {
						duplicatesCleaned := results["duplicates_cleaned"].(int)
						expiredCleaned := results["expired_cleaned"].(int)
						if duplicatesCleaned > 0 || expiredCleaned > 0 {
							logger.Infof("🧹 Auto cleanup: %d duplicates, %d expired orders cleaned (duration: %v)",
								duplicatesCleaned, expiredCleaned, duration)
						}
					}
				}
			case <-at.stopMonitorCh:
				wsMonitor.Stop()
				logger.Info("⏹ Stopped WebSocket real-time monitoring")
				return
			}
		}
	}()

	// 保留原有的30秒轮询作为备份（兼容性）
	at.monitorWg.Add(1)
	go func() {
		defer at.monitorWg.Done()
		monitorTicker := time.NewTicker(30 * time.Second)
		defer monitorTicker.Stop()

		logger.Info("🔄 Legacy polling monitoring started (backup, 30 seconds)")

		for {
			select {
			case <-monitorTicker.C:
				// 仅在WebSocket不可用时执行
				if err := at.MonitorAndExecutePendingOrders(); err != nil {
					logger.Warnf("⚠️ Error monitoring pending orders: %v", err)
				}
			case <-at.stopMonitorCh:
				logger.Info("⏹ Stopped legacy polling monitoring")
				return
			}
		}
	}()

	ticker := time.NewTicker(at.config.ScanInterval)
	defer ticker.Stop()

	// Execute immediately on first run
	if err := at.runCycle(); err != nil {
		logger.Infof("❌ Execution failed: %v", err)
	}

	for {
		at.isRunningMutex.RLock()
		running := at.isRunning
		at.isRunningMutex.RUnlock()

		if !running {
			break
		}

		select {
		case <-ticker.C:
			if err := at.runCycle(); err != nil {
				logger.Infof("❌ Execution failed: %v", err)
			}
		case <-at.stopMonitorCh:
			logger.Infof("[%s] ⏹ Stop signal received, exiting automatic trading main loop", at.name)
			return nil
		}
	}

	return nil
}

// Stop stops the automatic trading
func (at *AutoTrader) Stop() {
	at.isRunningMutex.Lock()
	if !at.isRunning {
		at.isRunningMutex.Unlock()
		return
	}
	at.isRunning = false
	at.isRunningMutex.Unlock()

	close(at.stopMonitorCh) // Notify monitoring goroutine to stop
	at.monitorWg.Wait()     // Wait for monitoring goroutine to finish
	logger.Info("⏹ Automatic trading system stopped")
}

// runCycle runs one trading cycle (using AI full decision-making)
func (at *AutoTrader) runCycle() error {
	at.callCount++

	logger.Info("\n" + strings.Repeat("=", 70) + "\n")
	logger.Infof("⏰ %s - AI decision cycle #%d", time.Now().Format("2006-01-02 15:04:05"), at.callCount)
	logger.Info(strings.Repeat("=", 70))

	// 0. Check if trader is stopped (early exit to prevent trades after Stop() is called)
	at.isRunningMutex.RLock()
	running := at.isRunning
	at.isRunningMutex.RUnlock()
	if !running {
		logger.Infof("⏹ Trader is stopped, aborting cycle #%d", at.callCount)
		return nil
	}

	// Create decision record
	record := &store.DecisionRecord{
		ExecutionLog: []string{},
		Success:      true,
	}

	// 1. Check if trading needs to be stopped
	if time.Now().Before(at.stopUntil) {
		remaining := at.stopUntil.Sub(time.Now())
		logger.Infof("⏸ Risk control: Trading paused, remaining %.0f minutes", remaining.Minutes())
		record.Success = false
		record.ErrorMessage = fmt.Sprintf("Risk control paused, remaining %.0f minutes", remaining.Minutes())
		at.saveDecision(record)
		return nil
	}

	// 2. Reset daily P&L (reset every day)
	if time.Since(at.lastResetTime) > 24*time.Hour {
		at.dailyPnL = 0
		at.lastResetTime = time.Now()
		logger.Info("📅 Daily P&L reset")
	}

	// 4. Collect trading context
	ctx, err := at.buildTradingContext()
	if err != nil {
		record.Success = false
		record.ErrorMessage = fmt.Sprintf("Failed to build trading context: %v", err)
		at.saveDecision(record)
		return fmt.Errorf("failed to build trading context: %w", err)
	}

	// Save equity snapshot independently (decoupled from AI decision, used for drawing profit curve)
	at.saveEquitySnapshot(ctx)

	// ========================================
	// 3. 动态止盈止损更新（对现有持仓）
	// ========================================
	at.updateDynamicStopLoss(ctx)

	logger.Info(strings.Repeat("=", 70))
	for _, coin := range ctx.CandidateCoins {
		record.CandidateCoins = append(record.CandidateCoins, coin.Symbol)
	}

	logger.Infof("📊 Account equity: %.2f USDT | Available: %.2f USDT | Positions: %d",
		ctx.Account.TotalEquity, ctx.Account.AvailableBalance, ctx.Account.PositionCount)

	// 5. Use strategy engine to call AI for decision
	logger.Infof("🤖 Requesting AI analysis and decision... [Strategy Engine]")
	aiDecision, err := kernel.GetFullDecisionWithStrategy(ctx, at.mcpClient, at.strategyEngine, "balanced")

	if aiDecision != nil && aiDecision.AIRequestDurationMs > 0 {
		record.AIRequestDurationMs = aiDecision.AIRequestDurationMs
		logger.Infof("⏱️ AI call duration: %.2f seconds", float64(record.AIRequestDurationMs)/1000)
		record.ExecutionLog = append(record.ExecutionLog,
			fmt.Sprintf("AI call duration: %d ms", record.AIRequestDurationMs))
	}

	// Save chain of thought, decisions, and input prompt even if there's an error (for debugging)
	if aiDecision != nil {
		record.SystemPrompt = aiDecision.SystemPrompt // Save system prompt
		record.InputPrompt = aiDecision.UserPrompt
		record.CoTTrace = aiDecision.CoTTrace
		record.RawResponse = aiDecision.RawResponse // Save raw AI response for debugging
		if len(aiDecision.Decisions) > 0 {
			decisionJSON, _ := json.MarshalIndent(aiDecision.Decisions, "", "  ")
			record.DecisionJSON = string(decisionJSON)
		}
	}

	if err != nil {
		record.Success = false
		record.ErrorMessage = fmt.Sprintf("Failed to get AI decision: %v", err)

		// Print system prompt and AI chain of thought (output even with errors for debugging)
		if aiDecision != nil {
			logger.Info("\n" + strings.Repeat("=", 70) + "\n")
			logger.Infof("📋 System prompt (error case)")
			logger.Info(strings.Repeat("=", 70))
			logger.Info(aiDecision.SystemPrompt)
			logger.Info(strings.Repeat("=", 70))

			if aiDecision.CoTTrace != "" {
				logger.Info("\n" + strings.Repeat("-", 70) + "\n")
				logger.Info("💭 AI chain of thought analysis (error case):")
				logger.Info(strings.Repeat("-", 70))
				logger.Info(aiDecision.CoTTrace)
				logger.Info(strings.Repeat("-", 70))
			}
		}

		at.saveDecision(record)
		return fmt.Errorf("failed to get AI decision: %w", err)
	}

	// // 5. Print system prompt
	// logger.Infof("\n" + strings.Repeat("=", 70))
	// logger.Infof("📋 System prompt [template: %s]", at.systemPromptTemplate)
	// logger.Info(strings.Repeat("=", 70))
	// logger.Info(decision.SystemPrompt)
	// logger.Infof(strings.Repeat("=", 70) + "\n")

	// 6. Print AI chain of thought
	// logger.Infof("\n" + strings.Repeat("-", 70))
	// logger.Info("💭 AI chain of thought analysis:")
	// logger.Info(strings.Repeat("-", 70))
	// logger.Info(decision.CoTTrace)
	// logger.Infof(strings.Repeat("-", 70) + "\n")

	// 7. Print AI decisions
	// logger.Infof("📋 AI decision list (%d items):\n", len(kernel.Decisions))
	// for i, d := range kernel.Decisions {
	//     logger.Infof("  [%d] %s: %s - %s", i+1, d.Symbol, d.Action, d.Reasoning)
	//     if d.Action == "open_long" || d.Action == "open_short" {
	//        logger.Infof("      Leverage: %dx | Position: %.2f USDT | Stop loss: %.4f | Take profit: %.4f",
	//           d.Leverage, d.PositionSizeUSD, d.StopLoss, d.TakeProfit)
	//     }
	// }
	logger.Info()
	logger.Info(strings.Repeat("-", 70))
	// 8. Sort decisions: ensure close positions first, then open positions (prevent position stacking overflow)
	logger.Info(strings.Repeat("-", 70))

	// Option B: Apply parameter optimization and risk management
	if at.enhancedSetup != nil {
		// Validate risk limits first
		if allowed, reason := at.enhancedSetup.ValidateRiskLimits(); !allowed {
			logger.Warnf("⚠️ Risk control triggered: %s", reason)
			record.Success = false
			record.ErrorMessage = fmt.Sprintf("Risk control: %s", reason)
			at.saveDecision(record)
			return nil
		}

		// Apply parameter optimization to decisions based on default metrics
		// Note: For full metrics update, fetch recent trades from database as needed
		for i := range aiDecision.Decisions {
			d := &aiDecision.Decisions[i]

			// Adjust confidence based on current market state
			adjustedConfidence := float64(d.Confidence)
			if at.enhancedSetup.ParameterOptimizer.volatilityMultiplier > 1.2 {
				// High volatility: reduce confidence slightly
				adjustedConfidence = adjustedConfidence * 0.9
			} else if at.enhancedSetup.ParameterOptimizer.volatilityMultiplier < 0.8 {
				// Low volatility: increase confidence
				adjustedConfidence = adjustedConfidence * 1.1
			}

			d.Confidence = int(adjustedConfidence)

			logger.Infof("🔧 [%s] Parameters optimized: confidence %d → %d",
				d.Symbol, int(float64(d.Confidence)/0.9), d.Confidence)
		}
	}

	// 8. Sort decisions: ensure close positions first, then open positions (prevent position stacking overflow)
	sortedDecisions := sortDecisionsByPriority(aiDecision.Decisions)

	logger.Info("🔄 Execution order (optimized): Close positions first → Open positions later")
	for i, d := range sortedDecisions {
		logger.Infof("  [%d] %s %s", i+1, d.Symbol, d.Action)
	}
	logger.Info()

	// Check if trader is stopped before executing any decisions (prevent trades after Stop())
	at.isRunningMutex.RLock()
	running = at.isRunning
	at.isRunningMutex.RUnlock()
	if !running {
		logger.Infof("⏹ Trader stopped before decision execution, aborting cycle #%d", at.callCount)
		return nil
	}

	// NEW: Save AI analysis and create pending orders (延迟执行模式)
	logger.Info("🔄 NEW WORKFLOW: Saving AI analysis → Waiting for price triggers → Auto-executing")

	// 检查是否已存在同币种的PENDING订单，避免重复创建
	existingOrders, err := at.store.Analysis().GetPendingOrdersByTrader(at.id)
	if err != nil {
		logger.Warnf("⚠️ Failed to get existing pending orders: %v", err)
	}

	// 创建现有订单映射
	existingOrderMap := make(map[string]*store.PendingOrder)
	for _, order := range existingOrders {
		if order.Status == "PENDING" {
			existingOrderMap[order.Symbol] = order
		}
	}

	// 过滤掉已存在PENDING订单的决策
	filteredDecisions := make([]kernel.Decision, 0)
	for _, d := range sortedDecisions {
		if d.Action == "open_long" || d.Action == "open_short" {
			if existingOrder, exists := existingOrderMap[d.Symbol]; exists {
				// 检查置信度
				currentConfidence := float64(d.Confidence)
				existingConfidence := existingOrder.Confidence * 100

				if currentConfidence > existingConfidence {
					logger.Infof("🔄 将替换同币种订单: %s (置信度 %.2f%% → %.2f%%)",
						d.Symbol, existingConfidence, currentConfidence)
					filteredDecisions = append(filteredDecisions, d)
				} else {
					logger.Infof("⏭️ 跳过 %s: 已有更优订单 (%.2f%% > %.2f%%)",
						d.Symbol, existingConfidence, currentConfidence)
				}
			} else {
				// 无现有订单，添加
				filteredDecisions = append(filteredDecisions, d)
			}
		} else {
			// 非开仓决策，直接添加
			filteredDecisions = append(filteredDecisions, d)
		}
	}

	// 使用过滤后的决策创建订单
	if len(filteredDecisions) > 0 {
		// 临时创建只包含过滤后决策的FullDecision
		filteredAIDecision := &kernel.FullDecision{
			Decisions:           filteredDecisions,
			UserPrompt:          aiDecision.UserPrompt,
			RawResponse:         aiDecision.RawResponse,
			CoTTrace:            aiDecision.CoTTrace,
			SystemPrompt:        aiDecision.SystemPrompt,
			AIRequestDurationMs: aiDecision.AIRequestDurationMs,
		}

		if err := at.SaveAnalysisAndCreatePendingOrders(filteredAIDecision); err != nil {
			logger.Warnf("⚠️ Failed to save analysis or create pending orders: %v", err)
		} else {
			logger.Infof("✅ AI analysis saved and pending orders created (filtered: %d decisions)", len(filteredDecisions))
			// Record decision actions for audit
			for _, d := range filteredDecisions {
				actionRecord := store.DecisionAction{
					Action:     d.Action,
					Symbol:     d.Symbol,
					Leverage:   d.Leverage,
					StopLoss:   d.StopLoss,
					TakeProfit: d.TakeProfit,
					Confidence: d.Confidence,
					Reasoning:  d.Reasoning,
					Timestamp:  time.Now().UTC(),
					Success:    true,
				}
				record.Decisions = append(record.Decisions, actionRecord)
			}
		}
	} else {
		logger.Info("⏭️ No new pending orders to create (all filtered by existing orders)")
	}

	// 9. Save decision record
	if err := at.saveDecision(record); err != nil {
		logger.Infof("⚠ Failed to save decision record: %v", err)
	}

	return nil
}

// buildTradingContext builds trading context
func (at *AutoTrader) buildTradingContext() (*kernel.Context, error) {
	// 1. Get account information
	balance, err := at.trader.GetBalance()
	if err != nil {
		return nil, fmt.Errorf("failed to get account balance: %w", err)
	}

	// Get account fields
	totalWalletBalance := 0.0
	totalUnrealizedProfit := 0.0
	availableBalance := 0.0
	totalEquity := 0.0

	if wallet, ok := balance["totalWalletBalance"].(float64); ok {
		totalWalletBalance = wallet
	}
	if unrealized, ok := balance["totalUnrealizedProfit"].(float64); ok {
		totalUnrealizedProfit = unrealized
	}
	if avail, ok := balance["availableBalance"].(float64); ok {
		availableBalance = avail
	}

	// Use totalEquity directly if provided by trader (more accurate)
	if eq, ok := balance["totalEquity"].(float64); ok && eq > 0 {
		totalEquity = eq
	} else {
		// Fallback: Total Equity = Wallet balance + Unrealized profit
		totalEquity = totalWalletBalance + totalUnrealizedProfit
	}

	// 2. Get position information
	positions, err := at.trader.GetPositions()
	if err != nil {
		return nil, fmt.Errorf("failed to get positions: %w", err)
	}

	var positionInfos []kernel.PositionInfo
	totalMarginUsed := 0.0

	// Current position key set (for cleaning up closed position records)
	currentPositionKeys := make(map[string]bool)

	// Pre-load all open positions from database to avoid N+1 query problem
	var dbPositionMap map[string]*store.TraderPosition
	if at.store != nil {
		dbPositions, err := at.store.Position().GetOpenPositions(at.id)
		if err == nil && len(dbPositions) > 0 {
			dbPositionMap = make(map[string]*store.TraderPosition, len(dbPositions))
			for _, dbPos := range dbPositions {
				key := dbPos.Symbol + "_" + dbPos.Side
				dbPositionMap[key] = dbPos
			}
		}
	}

	for _, pos := range positions {
		symbol := pos["symbol"].(string)
		side := pos["side"].(string)
		entryPrice := pos["entryPrice"].(float64)
		markPrice := pos["markPrice"].(float64)
		quantity := pos["positionAmt"].(float64)
		if quantity < 0 {
			quantity = -quantity // Short position quantity is negative, convert to positive
		}

		// Skip closed positions (quantity = 0), prevent "ghost positions" from being passed to AI
		if quantity == 0 {
			continue
		}

		unrealizedPnl := pos["unRealizedProfit"].(float64)
		liquidationPrice := pos["liquidationPrice"].(float64)

		// Calculate margin used (estimated)
		leverage := 10 // Default value, should actually be fetched from position info
		if lev, ok := pos["leverage"].(float64); ok {
			leverage = int(lev)
		}
		marginUsed := (quantity * markPrice) / float64(leverage)
		totalMarginUsed += marginUsed

		// Calculate P&L percentage (based on margin, considering leverage)
		pnlPct := calculatePnLPercentage(unrealizedPnl, marginUsed)

		// Get position open time from exchange (preferred) or fallback to local tracking
		posKey := symbol + "_" + side
		currentPositionKeys[posKey] = true

		var updateTime int64
		// Priority 1: Get from pre-loaded database positions (optimized - no N+1 queries)
		if dbPositionMap != nil {
			if dbPos, ok := dbPositionMap[posKey]; ok && dbPos.EntryTime > 0 {
				updateTime = dbPos.EntryTime
			}
		}
		// Priority 2: Get from exchange API (Bybit: createdTime, OKX: createdTime)
		if updateTime == 0 {
			if createdTime, ok := pos["createdTime"].(int64); ok && createdTime > 0 {
				updateTime = createdTime
			}
		}
		// Priority 3: Fallback to local tracking
		if updateTime == 0 {
			at.positionFirstSeenTimeMutex.Lock()
			if _, exists := at.positionFirstSeenTime[posKey]; !exists {
				at.positionFirstSeenTime[posKey] = time.Now().UnixMilli()
			}
			updateTime = at.positionFirstSeenTime[posKey]
			at.positionFirstSeenTimeMutex.Unlock()
		}

		// Get peak profit rate for this position
		at.peakPnLCacheMutex.RLock()
		peakPnlPct := at.peakPnLCache[posKey]
		at.peakPnLCacheMutex.RUnlock()

		positionInfos = append(positionInfos, kernel.PositionInfo{
			Symbol:           symbol,
			Side:             side,
			EntryPrice:       entryPrice,
			MarkPrice:        markPrice,
			Quantity:         quantity,
			Leverage:         leverage,
			UnrealizedPnL:    unrealizedPnl,
			UnrealizedPnLPct: pnlPct,
			PeakPnLPct:       peakPnlPct,
			LiquidationPrice: liquidationPrice,
			MarginUsed:       marginUsed,
			UpdateTime:       updateTime,
		})
	}

	// Clean up closed position records
	at.positionFirstSeenTimeMutex.Lock()
	for key := range at.positionFirstSeenTime {
		if !currentPositionKeys[key] {
			delete(at.positionFirstSeenTime, key)
		}
	}
	at.positionFirstSeenTimeMutex.Unlock()

	// 3. Use strategy engine to get candidate coins (must have strategy engine)
	if at.strategyEngine == nil {
		return nil, fmt.Errorf("trader has no strategy engine configured")
	}
	candidateCoins, err := at.strategyEngine.GetCandidateCoins()
	if err != nil {
		return nil, fmt.Errorf("failed to get candidate coins: %w", err)
	}
	logger.Infof("📋 [%s] Strategy engine fetched candidate coins: %d", at.name, len(candidateCoins))

	// 4. Calculate total P&L
	totalPnL := totalEquity - at.initialBalance
	totalPnLPct := 0.0
	if at.initialBalance > 0 {
		totalPnLPct = (totalPnL / at.initialBalance) * 100
	}

	marginUsedPct := 0.0
	if totalEquity > 0 {
		marginUsedPct = (totalMarginUsed / totalEquity) * 100
	}

	// 5. Get leverage from strategy config
	strategyConfig := at.strategyEngine.GetConfig()
	btcEthLeverage := strategyConfig.RiskControl.BTCETHMaxLeverage
	altcoinLeverage := strategyConfig.RiskControl.AltcoinMaxLeverage
	logger.Infof("📋 [%s] Strategy leverage config: BTC/ETH=%dx, Altcoin=%dx", at.name, btcEthLeverage, altcoinLeverage)

	// 6. Build context
	ctx := &kernel.Context{
		CurrentTime:     time.Now().UTC().Format("2006-01-02 15:04:05 UTC"),
		RuntimeMinutes:  int(time.Since(at.startTime).Minutes()),
		CallCount:       at.callCount,
		BTCETHLeverage:  btcEthLeverage,
		AltcoinLeverage: altcoinLeverage,
		Account: kernel.AccountInfo{
			TotalEquity:      totalEquity,
			AvailableBalance: availableBalance,
			UnrealizedPnL:    totalUnrealizedProfit,
			TotalPnL:         totalPnL,
			TotalPnLPct:      totalPnLPct,
			MarginUsed:       totalMarginUsed,
			MarginUsedPct:    marginUsedPct,
			PositionCount:    len(positionInfos),
		},
		Positions:      positionInfos,
		CandidateCoins: candidateCoins,
	}

	// 7. Add recent closed trades (if store is available)
	if at.store != nil {
		// Get recent 10 closed trades for AI context
		recentTrades, err := at.store.Position().GetRecentTrades(at.id, 10)
		if err != nil {
			logger.Infof("⚠️ [%s] Failed to get recent trades: %v", at.name, err)
		} else {
			logger.Infof("📊 [%s] Found %d recent closed trades for AI context", at.name, len(recentTrades))
			for _, trade := range recentTrades {
				// Convert Unix timestamps to formatted strings for AI readability
				entryTimeStr := ""
				if trade.EntryTime > 0 {
					entryTimeStr = time.Unix(trade.EntryTime, 0).UTC().Format("01-02 15:04 UTC")
				}
				exitTimeStr := ""
				if trade.ExitTime > 0 {
					exitTimeStr = time.Unix(trade.ExitTime, 0).UTC().Format("01-02 15:04 UTC")
				}

				ctx.RecentOrders = append(ctx.RecentOrders, kernel.RecentOrder{
					Symbol:       trade.Symbol,
					Side:         trade.Side,
					EntryPrice:   trade.EntryPrice,
					ExitPrice:    trade.ExitPrice,
					RealizedPnL:  trade.RealizedPnL,
					PnLPct:       trade.PnLPct,
					EntryTime:    entryTimeStr,
					ExitTime:     exitTimeStr,
					HoldDuration: trade.HoldDuration,
				})
			}
		}
		// Get trading statistics for AI context
		stats, err := at.store.Position().GetFullStats(at.id)
		if err != nil {
			logger.Infof("⚠️ [%s] Failed to get trading stats: %v", at.name, err)
		} else if stats == nil {
			logger.Infof("⚠️ [%s] GetFullStats returned nil", at.name)
		} else if stats.TotalTrades == 0 {
			logger.Infof("⚠️ [%s] GetFullStats returned 0 trades (traderID=%s)", at.name, at.id)
		} else {
			ctx.TradingStats = &kernel.TradingStats{
				TotalTrades:    stats.TotalTrades,
				WinRate:        stats.WinRate,
				ProfitFactor:   stats.ProfitFactor,
				SharpeRatio:    stats.SharpeRatio,
				TotalPnL:       stats.TotalPnL,
				AvgWin:         stats.AvgWin,
				AvgLoss:        stats.AvgLoss,
				MaxDrawdownPct: stats.MaxDrawdownPct,
			}
			logger.Infof("📈 [%s] Trading stats: %d trades, %.1f%% win rate, PF=%.2f, Sharpe=%.2f, DD=%.1f%%",
				at.name, stats.TotalTrades, stats.WinRate, stats.ProfitFactor, stats.SharpeRatio, stats.MaxDrawdownPct)
		}
	} else {
		logger.Infof("⚠️ [%s] Store is nil, cannot get recent trades", at.name)
	}

	// 8. Get quantitative data (if enabled in strategy config)
	if strategyConfig.Indicators.EnableQuantData {
		// Collect symbols to query (candidate coins + position coins)
		symbolsToQuery := make(map[string]bool)
		for _, coin := range candidateCoins {
			symbolsToQuery[coin.Symbol] = true
		}
		for _, pos := range positionInfos {
			symbolsToQuery[pos.Symbol] = true
		}

		symbols := make([]string, 0, len(symbolsToQuery))
		for sym := range symbolsToQuery {
			symbols = append(symbols, sym)
		}

		logger.Infof("📊 [%s] Fetching quantitative data for %d symbols...", at.name, len(symbols))
		ctx.QuantDataMap = at.strategyEngine.FetchQuantDataBatch(symbols)
		logger.Infof("📊 [%s] Successfully fetched quantitative data for %d symbols", at.name, len(ctx.QuantDataMap))
	}

	// 9. Get OI ranking data (market-wide position changes)
	if strategyConfig.Indicators.EnableOIRanking {
		logger.Infof("📊 [%s] Fetching OI ranking data...", at.name)
		ctx.OIRankingData = at.strategyEngine.FetchOIRankingData()
		if ctx.OIRankingData != nil {
			logger.Infof("📊 [%s] OI ranking data ready: %d top, %d low positions",
				at.name, len(ctx.OIRankingData.TopPositions), len(ctx.OIRankingData.LowPositions))
		}
	}

	// 10. Get NetFlow ranking data (market-wide fund flow)
	if strategyConfig.Indicators.EnableNetFlowRanking {
		logger.Infof("💰 [%s] Fetching NetFlow ranking data...", at.name)
		ctx.NetFlowRankingData = at.strategyEngine.FetchNetFlowRankingData()
		if ctx.NetFlowRankingData != nil {
			logger.Infof("💰 [%s] NetFlow ranking data ready: inst_in=%d, inst_out=%d",
				at.name, len(ctx.NetFlowRankingData.InstitutionFutureTop), len(ctx.NetFlowRankingData.InstitutionFutureLow))
		}
	}

	// 11. Get Price ranking data (market-wide gainers/losers)
	if strategyConfig.Indicators.EnablePriceRanking {
		logger.Infof("📈 [%s] Fetching Price ranking data...", at.name)
		ctx.PriceRankingData = at.strategyEngine.FetchPriceRankingData()
		if ctx.PriceRankingData != nil {
			logger.Infof("📈 [%s] Price ranking data ready for %d durations",
				at.name, len(ctx.PriceRankingData.Durations))
		}
	}

	return ctx, nil
}

// executeDecisionWithRecord executes AI decision and records detailed information
func (at *AutoTrader) executeDecisionWithRecord(decision *kernel.Decision, actionRecord *store.DecisionAction) error {
	switch decision.Action {
	case "open_long":
		return at.executeOpenLongWithRecord(decision, actionRecord)
	case "open_short":
		return at.executeOpenShortWithRecord(decision, actionRecord)
	case "close_long":
		return at.executeCloseLongWithRecord(decision, actionRecord)
	case "close_short":
		return at.executeCloseShortWithRecord(decision, actionRecord)
	case "hold", "wait":
		// No execution needed, just record
		return nil
	default:
		return fmt.Errorf("unknown action: %s", decision.Action)
	}
}

// ExecuteDecision executes a trading decision from external sources (e.g., debate consensus)
// This is a public method that can be called by other modules
func (at *AutoTrader) ExecuteDecision(d *kernel.Decision) error {
	logger.Infof("[%s] Executing external decision: %s %s", at.name, d.Action, d.Symbol)

	// Create a minimal action record for tracking
	actionRecord := &store.DecisionAction{
		Symbol:     d.Symbol,
		Action:     d.Action,
		Leverage:   d.Leverage,
		StopLoss:   d.StopLoss,
		TakeProfit: d.TakeProfit,
		Confidence: d.Confidence,
		Reasoning:  d.Reasoning,
	}

	// Execute the decision
	err := at.executeDecisionWithRecord(d, actionRecord)
	if err != nil {
		logger.Errorf("[%s] External decision execution failed: %v", at.name, err)
		return err
	}

	logger.Infof("[%s] External decision executed successfully: %s %s", at.name, d.Action, d.Symbol)
	return nil
}

// executeOpenLongWithRecord executes open long position and records detailed information
func (at *AutoTrader) executeOpenLongWithRecord(decision *kernel.Decision, actionRecord *store.DecisionAction) error {
	logger.Infof("  📈 Open long: %s", decision.Symbol)

	// ⚠️ Get current positions for multiple checks
	positions, err := at.trader.GetPositions()
	if err != nil {
		return fmt.Errorf("failed to get positions: %w", err)
	}

	// [CODE ENFORCED] Check max positions limit
	if err := at.enforceMaxPositions(len(positions)); err != nil {
		return err
	}

	// Check if there's already a position in the same symbol and direction
	for _, pos := range positions {
		if pos["symbol"] == decision.Symbol && pos["side"] == "long" {
			return fmt.Errorf("❌ %s already has long position, close it first", decision.Symbol)
		}
	}

	// Get current price
	marketData, err := market.Get(decision.Symbol)
	if err != nil {
		return err
	}

	// Get balance (needed for multiple checks)
	balance, err := at.trader.GetBalance()
	if err != nil {
		return fmt.Errorf("failed to get account balance: %w", err)
	}
	availableBalance := 0.0
	if avail, ok := balance["availableBalance"].(float64); ok {
		availableBalance = avail
	}

	// Get equity for position value ratio check
	equity := 0.0
	if eq, ok := balance["totalEquity"].(float64); ok && eq > 0 {
		equity = eq
	} else if eq, ok := balance["totalWalletBalance"].(float64); ok && eq > 0 {
		equity = eq
	} else {
		equity = availableBalance // Fallback to available balance
	}

	// [CODE ENFORCED] Position Value Ratio Check: position_value <= equity × ratio
	adjustedPositionSize, wasCapped := at.enforcePositionValueRatio(decision.PositionSizeUSD, equity, decision.Symbol)
	if wasCapped {
		decision.PositionSizeUSD = adjustedPositionSize
	}

	// ⚠️ Auto-adjust position size if insufficient margin
	// Formula: totalRequired = positionSize/leverage + positionSize*0.001 + positionSize/leverage*0.01
	//        = positionSize * (1.01/leverage + 0.001)
	marginFactor := 1.01/float64(decision.Leverage) + 0.001
	maxAffordablePositionSize := availableBalance / marginFactor

	actualPositionSize := decision.PositionSizeUSD
	if actualPositionSize > maxAffordablePositionSize {
		// Use 98% of max to leave buffer for price fluctuation
		adjustedSize := maxAffordablePositionSize * 0.98
		logger.Infof("  ⚠️ Position size %.2f exceeds max affordable %.2f, auto-reducing to %.2f",
			actualPositionSize, maxAffordablePositionSize, adjustedSize)
		actualPositionSize = adjustedSize
		decision.PositionSizeUSD = actualPositionSize
	}

	// [CODE ENFORCED] Minimum position size check
	if err := at.enforceMinPositionSize(decision.PositionSizeUSD); err != nil {
		return err
	}

	// Calculate quantity with adjusted position size
	quantity := actualPositionSize / marketData.CurrentPrice
	actionRecord.Quantity = quantity
	actionRecord.Price = marketData.CurrentPrice

	// Set margin mode
	if err := at.trader.SetMarginMode(decision.Symbol, at.config.IsCrossMargin); err != nil {
		logger.Infof("  ⚠️ Failed to set margin mode: %v", err)
		// Continue execution, doesn't affect trading
	}

	// Open position
	order, err := at.trader.OpenLong(decision.Symbol, quantity, decision.Leverage)
	if err != nil {
		return err
	}

	// Record order ID
	if orderID, ok := order["orderId"].(int64); ok {
		actionRecord.OrderID = orderID
	}

	logger.Infof("  ✓ Position opened successfully, order ID: %v, quantity: %.4f", order["orderId"], quantity)

	// Record order to database and poll for confirmation
	// 在 recordAndConfirmOrder 中会创建持仓，之后我们需要记录 TP/SL
	at.recordAndConfirmOrder(order, decision.Symbol, "open_long", quantity, marketData.CurrentPrice, decision.Leverage, 0, decision.TakeProfit, decision.StopLoss)

	// Record position opening time
	posKey := decision.Symbol + "_long"
	at.positionFirstSeenTimeMutex.Lock()
	at.positionFirstSeenTime[posKey] = time.Now().UnixMilli()
	at.positionFirstSeenTimeMutex.Unlock()

	// Set stop loss and take profit with retry mechanism
	maxSLRetries := 3
	var slErr error
	for i := 0; i < maxSLRetries; i++ {
		slErr = at.trader.SetStopLoss(decision.Symbol, "LONG", quantity, decision.StopLoss)
		if slErr == nil {
			logger.Infof("  ✅ Stop loss set successfully for %s LONG at %.4f", decision.Symbol, decision.StopLoss)
			break
		}
		if i < maxSLRetries-1 {
			logger.Warnf("  ⚠️ Stop loss attempt %d/%d failed: %v, retrying...", i+1, maxSLRetries, slErr)
			time.Sleep(time.Duration(i+1) * time.Second) // Exponential backoff
		}
	}
	if slErr != nil {
		logger.Errorf("  🚨 Failed to set stop loss after %d retries: %v - Position without SL protection!", maxSLRetries, slErr)

		// ✅ 改进的错误恢复机制
		// 1. 对所有持仓执行紧急平仓（不仅限于低信心度）
		logger.Warnf("  🚨 Initiating emergency close for unprotected position %s", decision.Symbol)
		if closeErr := at.emergencyClosePosition(decision.Symbol, "long"); closeErr != nil {
			logger.Errorf("  ❌ Emergency close failed: %v", closeErr)
			// 2. 发送紧急警报
			at.sendEmergencyAlert(decision.Symbol, "LONG", "止损设置失败且紧急平仓失败")
		} else {
			logger.Infof("  ✅ Emergency close succeeded for %s LONG", decision.Symbol)
			// 3. 关闭关联的ASL记录
			if err := at.store.AdaptiveStopLoss().CloseRecordBySymbol(at.id, decision.Symbol); err != nil {
				logger.Warnf("  ⚠️ Failed to clean ASL record: %v", err)
			}
		}
		return fmt.Errorf("position closed due to SL setup failure and safety policy")
	}

	// Set take profit with retry mechanism (same as stop loss)
	maxTPRetries := 3
	var tpErr error
	for i := 0; i < maxTPRetries; i++ {
		tpErr = at.trader.SetTakeProfit(decision.Symbol, "LONG", quantity, decision.TakeProfit)
		if tpErr == nil {
			logger.Infof("  ✅ Take profit set successfully for %s LONG at %.4f", decision.Symbol, decision.TakeProfit)
			break
		}
		if i < maxTPRetries-1 {
			logger.Warnf("  ⚠️ Take profit attempt %d/%d failed: %v, retrying...", i+1, maxTPRetries, tpErr)
			time.Sleep(time.Duration(i+1) * time.Second) // Exponential backoff
		}
	}
	if tpErr != nil {
		logger.Warnf("  🚨 Failed to set take profit after %d retries: %v - Position opened without TP protection!", maxTPRetries, tpErr)
		// Emergency close for low confidence positions without TP
		if decision.Confidence < 60 {
			logger.Warnf("  🚨 Low confidence (<%d%%) position without TP - executing emergency close!", 60)
			if closeErr := at.emergencyClosePosition(decision.Symbol, "long"); closeErr != nil {
				logger.Errorf("  ❌ Emergency close failed: %v", closeErr)
			} else {
				return fmt.Errorf("position closed due to TP setup failure")
			}
		}
	}

	// Option B: Set adaptive stop loss
	if at.enhancedSetup != nil {
		// Use a default ATR value or fetch if available
		// SetStopLevelForPosition(symbol, entryPrice, stopLoss, takeProfit, atrValue)
		atrValue := math.Abs(marketData.CurrentPrice - decision.StopLoss) // Approximate ATR from stop distance
		at.enhancedSetup.AdaptiveStopLoss.SetStopLevelForPosition(
			decision.Symbol,
			marketData.CurrentPrice,
			decision.StopLoss,
			decision.TakeProfit,
			atrValue,
		)
		logger.Infof("  🛡️ Adaptive stop loss set for %s LONG (ATR: %.2f)", decision.Symbol, atrValue)
	}

	return nil
}

// executeOpenShortWithRecord executes open short position and records detailed information
func (at *AutoTrader) executeOpenShortWithRecord(decision *kernel.Decision, actionRecord *store.DecisionAction) error {
	logger.Infof("  📉 Open short: %s", decision.Symbol)

	// ⚠️ Get current positions for multiple checks
	positions, err := at.trader.GetPositions()
	if err != nil {
		return fmt.Errorf("failed to get positions: %w", err)
	}

	// [CODE ENFORCED] Check max positions limit
	if err := at.enforceMaxPositions(len(positions)); err != nil {
		return err
	}

	// Check if there's already a position in the same symbol and direction
	for _, pos := range positions {
		if pos["symbol"] == decision.Symbol && pos["side"] == "short" {
			return fmt.Errorf("❌ %s already has short position, close it first", decision.Symbol)
		}
	}

	// Get current price
	marketData, err := market.Get(decision.Symbol)
	if err != nil {
		return err
	}

	// Get balance (needed for multiple checks)
	balance, err := at.trader.GetBalance()
	if err != nil {
		return fmt.Errorf("failed to get account balance: %w", err)
	}
	availableBalance := 0.0
	if avail, ok := balance["availableBalance"].(float64); ok {
		availableBalance = avail
	}

	// Get equity for position value ratio check
	equity := 0.0
	if eq, ok := balance["totalEquity"].(float64); ok && eq > 0 {
		equity = eq
	} else if eq, ok := balance["totalWalletBalance"].(float64); ok && eq > 0 {
		equity = eq
	} else {
		equity = availableBalance // Fallback to available balance
	}

	// [CODE ENFORCED] Position Value Ratio Check: position_value <= equity × ratio
	adjustedPositionSize, wasCapped := at.enforcePositionValueRatio(decision.PositionSizeUSD, equity, decision.Symbol)
	if wasCapped {
		decision.PositionSizeUSD = adjustedPositionSize
	}

	// ⚠️ Auto-adjust position size if insufficient margin
	// Formula: totalRequired = positionSize/leverage + positionSize*0.001 + positionSize/leverage*0.01
	//        = positionSize * (1.01/leverage + 0.001)
	marginFactor := 1.01/float64(decision.Leverage) + 0.001
	maxAffordablePositionSize := availableBalance / marginFactor

	actualPositionSize := decision.PositionSizeUSD
	if actualPositionSize > maxAffordablePositionSize {
		// Use 98% of max to leave buffer for price fluctuation
		adjustedSize := maxAffordablePositionSize * 0.98
		logger.Infof("  ⚠️ Position size %.2f exceeds max affordable %.2f, auto-reducing to %.2f",
			actualPositionSize, maxAffordablePositionSize, adjustedSize)
		actualPositionSize = adjustedSize
		decision.PositionSizeUSD = actualPositionSize
	}

	// [CODE ENFORCED] Minimum position size check
	if err := at.enforceMinPositionSize(decision.PositionSizeUSD); err != nil {
		return err
	}

	// Calculate quantity with adjusted position size
	quantity := actualPositionSize / marketData.CurrentPrice
	actionRecord.Quantity = quantity
	actionRecord.Price = marketData.CurrentPrice

	// Set margin mode
	if err := at.trader.SetMarginMode(decision.Symbol, at.config.IsCrossMargin); err != nil {
		logger.Infof("  ⚠️ Failed to set margin mode: %v", err)
		// Continue execution, doesn't affect trading
	}

	// Open position
	order, err := at.trader.OpenShort(decision.Symbol, quantity, decision.Leverage)
	if err != nil {
		return err
	}

	// Record order ID
	if orderID, ok := order["orderId"].(int64); ok {
		actionRecord.OrderID = orderID
	}

	logger.Infof("  ✓ Position opened successfully, order ID: %v, quantity: %.4f", order["orderId"], quantity)

	// Record order to database and poll for confirmation
	at.recordAndConfirmOrder(order, decision.Symbol, "open_short", quantity, marketData.CurrentPrice, decision.Leverage, 0, decision.TakeProfit, decision.StopLoss)

	// Record position opening time
	posKey := decision.Symbol + "_short"
	at.positionFirstSeenTimeMutex.Lock()
	at.positionFirstSeenTime[posKey] = time.Now().UnixMilli()
	at.positionFirstSeenTimeMutex.Unlock()

	// Set stop loss and take profit with retry mechanism
	maxSLRetries := 3
	var slErr error
	for i := 0; i < maxSLRetries; i++ {
		slErr = at.trader.SetStopLoss(decision.Symbol, "SHORT", quantity, decision.StopLoss)
		if slErr == nil {
			logger.Infof("  ✅ Stop loss set successfully for %s SHORT at %.4f", decision.Symbol, decision.StopLoss)
			break
		}
		if i < maxSLRetries-1 {
			logger.Warnf("  ⚠️ Stop loss attempt %d/%d failed: %v, retrying...", i+1, maxSLRetries, slErr)
			time.Sleep(time.Duration(i+1) * time.Second) // Exponential backoff
		}
	}
	if slErr != nil {
		logger.Errorf("  🚨 Failed to set stop loss after %d retries: %v - Position without SL protection!", maxSLRetries, slErr)

		// ✅ 改进的错误恢复机制
		// 1. 对所有持仓执行紧急平仓（不仅限于低信心度）
		logger.Warnf("  🚨 Initiating emergency close for unprotected position %s", decision.Symbol)
		if closeErr := at.emergencyClosePosition(decision.Symbol, "short"); closeErr != nil {
			logger.Errorf("  ❌ Emergency close failed: %v", closeErr)
			// 2. 发送紧急警报
			at.sendEmergencyAlert(decision.Symbol, "SHORT", "止损设置失败且紧急平仓失败")
		} else {
			logger.Infof("  ✅ Emergency close succeeded for %s SHORT", decision.Symbol)
			// 3. 关闭关联的ASL记录
			if err := at.store.AdaptiveStopLoss().CloseRecordBySymbol(at.id, decision.Symbol); err != nil {
				logger.Warnf("  ⚠️ Failed to clean ASL record: %v", err)
			}
		}
		return fmt.Errorf("position closed due to SL setup failure and safety policy")
	}

	// Set take profit with retry mechanism (same as stop loss)
	maxTPRetries := 3
	var tpErr error
	for i := 0; i < maxTPRetries; i++ {
		tpErr = at.trader.SetTakeProfit(decision.Symbol, "SHORT", quantity, decision.TakeProfit)
		if tpErr == nil {
			logger.Infof("  ✅ Take profit set successfully for %s SHORT at %.4f", decision.Symbol, decision.TakeProfit)
			break
		}
		if i < maxTPRetries-1 {
			logger.Warnf("  ⚠️ Take profit attempt %d/%d failed: %v, retrying...", i+1, maxTPRetries, tpErr)
			time.Sleep(time.Duration(i+1) * time.Second) // Exponential backoff
		}
	}
	if tpErr != nil {
		logger.Warnf("  🚨 Failed to set take profit after %d retries: %v - Position opened without TP protection!", maxTPRetries, tpErr)
		// Emergency close for low confidence positions without TP
		if decision.Confidence < 60 {
			logger.Warnf("  🚨 Low confidence (<%d%%) position without TP - executing emergency close!", 60)
			if closeErr := at.emergencyClosePosition(decision.Symbol, "short"); closeErr != nil {
				logger.Errorf("  ❌ Emergency close failed: %v", closeErr)
			} else {
				return fmt.Errorf("position closed due to TP setup failure")
			}
		}
	}

	// Option B: Set adaptive stop loss
	if at.enhancedSetup != nil {
		// Use a default ATR value or fetch if available
		// SetStopLevelForPosition(symbol, entryPrice, stopLoss, takeProfit, atrValue)
		atrValue := math.Abs(decision.StopLoss - marketData.CurrentPrice) // Approximate ATR from stop distance
		at.enhancedSetup.AdaptiveStopLoss.SetStopLevelForPosition(
			decision.Symbol,
			marketData.CurrentPrice,
			decision.StopLoss,
			decision.TakeProfit,
			atrValue,
		)
		logger.Infof("  🛡️ Adaptive stop loss set for %s SHORT (ATR: %.2f)", decision.Symbol, atrValue)
	}

	return nil
}

// executeCloseLongWithRecord executes close long position and records detailed information
func (at *AutoTrader) executeCloseLongWithRecord(decision *kernel.Decision, actionRecord *store.DecisionAction) error {
	logger.Infof("  🔄 Close long: %s", decision.Symbol)

	// Get current price
	marketData, err := market.Get(decision.Symbol)
	if err != nil {
		return err
	}
	actionRecord.Price = marketData.CurrentPrice

	// Normalize symbol for database lookup
	normalizedSymbol := market.Normalize(decision.Symbol)

	// Get entry price and quantity - ALWAYS check exchange first for accurate quantity
	var entryPrice float64
	var quantity float64
	var exchangeQty float64

	// First get from exchange API (source of truth for current position)
	positions, posErr := at.trader.GetPositions()
	if posErr == nil {
		for _, pos := range positions {
			if pos["symbol"] == decision.Symbol && pos["side"] == "long" {
				if amt, ok := pos["positionAmt"].(float64); ok && amt > 0 {
					exchangeQty = amt
				}
				if ep, ok := pos["entryPrice"].(float64); ok {
					entryPrice = ep
				}
				break
			}
		}
	}

	// Then get from local database (for entry price if not available from exchange)
	if at.store != nil {
		if openPos, err := at.store.Position().GetOpenPositionBySymbol(at.id, normalizedSymbol, "LONG"); err == nil && openPos != nil {
			// Use exchange quantity as it's more accurate (may have been partially closed manually)
			if exchangeQty > 0 {
				quantity = exchangeQty
				// Update local DB if out of sync
				if openPos.Quantity != exchangeQty {
					logger.Warnf("  ⚠️ Local qty (%.8f) differs from exchange (%.8f), using exchange value", openPos.Quantity, exchangeQty)
				}
			} else {
				quantity = openPos.Quantity
			}
			// Use local entry price if exchange didn't provide it
			if entryPrice == 0 {
				entryPrice = openPos.EntryPrice
			}
			logger.Infof("  📊 Position data: qty=%.8f (exchange), entry=%.2f", quantity, entryPrice)
		}
	}

	// Fallback if local data not found
	if quantity == 0 && exchangeQty > 0 {
		quantity = exchangeQty
		logger.Infof("  📊 Using exchange position data only: qty=%.8f, entry=%.2f", quantity, entryPrice)
	}

	// Close position
	order, err := at.trader.CloseLong(decision.Symbol, 0) // 0 = close all
	if err != nil {
		return err
	}

	// Record order ID
	if orderID, ok := order["orderId"].(int64); ok {
		actionRecord.OrderID = orderID
	}

	// Record order to database and poll for confirmation
	at.recordAndConfirmOrder(order, decision.Symbol, "close_long", quantity, marketData.CurrentPrice, 0, entryPrice, 0, 0)

	logger.Infof("  ✓ Position closed successfully")
	return nil
}

// executeCloseShortWithRecord executes close short position and records detailed information
func (at *AutoTrader) executeCloseShortWithRecord(decision *kernel.Decision, actionRecord *store.DecisionAction) error {
	logger.Infof("  🔄 Close short: %s", decision.Symbol)

	// Get current price
	marketData, err := market.Get(decision.Symbol)
	if err != nil {
		return err
	}
	actionRecord.Price = marketData.CurrentPrice

	// Normalize symbol for database lookup
	normalizedSymbol := market.Normalize(decision.Symbol)

	// Get entry price and quantity - ALWAYS check exchange first for accurate quantity
	var entryPrice float64
	var quantity float64
	var exchangeQty float64

	// First get from exchange API (source of truth for current position)
	positions, posErr := at.trader.GetPositions()
	if posErr == nil {
		for _, pos := range positions {
			if pos["symbol"] == decision.Symbol && pos["side"] == "short" {
				if amt, ok := pos["positionAmt"].(float64); ok {
					exchangeQty = -amt // positionAmt is negative for short
				}
				if ep, ok := pos["entryPrice"].(float64); ok {
					entryPrice = ep
				}
				break
			}
		}
	}

	// Then get from local database (for entry price if not available from exchange)
	if at.store != nil {
		if openPos, err := at.store.Position().GetOpenPositionBySymbol(at.id, normalizedSymbol, "SHORT"); err == nil && openPos != nil {
			// Use exchange quantity as it's more accurate (may have been partially closed manually)
			if exchangeQty > 0 {
				quantity = exchangeQty
				// Update local DB if out of sync
				if openPos.Quantity != exchangeQty {
					logger.Warnf("  ⚠️ Local qty (%.8f) differs from exchange (%.8f), using exchange value", openPos.Quantity, exchangeQty)
				}
			} else {
				quantity = openPos.Quantity
			}
			// Use local entry price if exchange didn't provide it
			if entryPrice == 0 {
				entryPrice = openPos.EntryPrice
			}
			logger.Infof("  📊 Position data: qty=%.8f (exchange), entry=%.2f", quantity, entryPrice)
		}
	}

	// Fallback if local data not found
	if quantity == 0 && exchangeQty > 0 {
		quantity = exchangeQty
		logger.Infof("  📊 Using exchange position data only: qty=%.8f, entry=%.2f", quantity, entryPrice)
	}

	// Close position
	order, err := at.trader.CloseShort(decision.Symbol, 0) // 0 = close all
	if err != nil {
		return err
	}

	// Record order ID
	if orderID, ok := order["orderId"].(int64); ok {
		actionRecord.OrderID = orderID
	}

	// Record order to database and poll for confirmation
	at.recordAndConfirmOrder(order, decision.Symbol, "close_short", quantity, marketData.CurrentPrice, 0, entryPrice, 0, 0)

	logger.Infof("  ✓ Position closed successfully")
	return nil
}

// GetID gets trader ID
func (at *AutoTrader) GetID() string {
	return at.id
}

// GetName gets trader name
func (at *AutoTrader) GetName() string {
	return at.name
}

// GetAIModel gets AI model
func (at *AutoTrader) GetAIModel() string {
	return at.aiModel
}

// GetExchange gets exchange
func (at *AutoTrader) GetExchange() string {
	return at.exchange
}

// GetShowInCompetition returns whether trader should be shown in competition
func (at *AutoTrader) GetShowInCompetition() bool {
	return at.showInCompetition
}

// SetShowInCompetition sets whether trader should be shown in competition
func (at *AutoTrader) SetShowInCompetition(show bool) {
	at.showInCompetition = show
}

// SetCustomPrompt sets custom trading strategy prompt
func (at *AutoTrader) SetCustomPrompt(prompt string) {
	at.customPrompt = prompt
}

// SetOverrideBasePrompt sets whether to override base prompt
func (at *AutoTrader) SetOverrideBasePrompt(override bool) {
	at.overrideBasePrompt = override
}

// GetSystemPromptTemplate gets current system prompt template name (from strategy config)
func (at *AutoTrader) GetSystemPromptTemplate() string {
	if at.strategyEngine != nil {
		config := at.strategyEngine.GetConfig()
		if config.CustomPrompt != "" {
			return "custom"
		}
	}
	return ""
}

// GetErrorTracker returns the error tracker instance
func (at *AutoTrader) GetErrorTracker() *ErrorTracker {
	return at.errorTracker
}

// saveEquitySnapshot saves equity snapshot independently (for drawing profit curve, decoupled from AI decision)
func (at *AutoTrader) saveEquitySnapshot(ctx *kernel.Context) {
	if at.store == nil || ctx == nil {
		return
	}

	snapshot := &store.EquitySnapshot{
		TraderID:      at.id,
		Timestamp:     time.Now().UTC(),
		TotalEquity:   ctx.Account.TotalEquity,
		Balance:       ctx.Account.TotalEquity - ctx.Account.UnrealizedPnL,
		UnrealizedPnL: ctx.Account.UnrealizedPnL,
		PositionCount: ctx.Account.PositionCount,
		MarginUsedPct: ctx.Account.MarginUsedPct,
	}

	if err := at.store.Equity().Save(snapshot); err != nil {
		logger.Infof("⚠️ Failed to save equity snapshot: %v", err)
	}
}

// saveDecision saves AI decision log to database (only records AI input/output, for debugging)
func (at *AutoTrader) saveDecision(record *store.DecisionRecord) error {
	if at.store == nil {
		return nil
	}

	at.cycleNumber++
	record.CycleNumber = at.cycleNumber
	record.TraderID = at.id

	if record.Timestamp.IsZero() {
		record.Timestamp = time.Now().UTC()
	}

	if err := at.store.Decision().LogDecision(record); err != nil {
		logger.Infof("⚠️ Failed to save decision record: %v", err)
		return err
	}

	logger.Infof("📝 Decision record saved: trader=%s, cycle=%d", at.id, at.cycleNumber)
	return nil
}

// GetStore gets data store (for external access to decision records, etc.)
func (at *AutoTrader) GetStore() *store.Store {
	return at.store
}

// GetStatus gets system status (for API)
func (at *AutoTrader) GetStatus() map[string]interface{} {
	aiProvider := "DeepSeek"
	if at.config.UseQwen {
		aiProvider = "Qwen"
	}

	at.isRunningMutex.RLock()
	isRunning := at.isRunning
	at.isRunningMutex.RUnlock()

	return map[string]interface{}{
		"trader_id":       at.id,
		"trader_name":     at.name,
		"ai_model":        at.aiModel,
		"exchange":        at.exchange,
		"is_running":      isRunning,
		"start_time":      at.startTime.Format(time.RFC3339),
		"runtime_minutes": int(time.Since(at.startTime).Minutes()),
		"call_count":      at.callCount,
		"initial_balance": at.initialBalance,
		"scan_interval":   at.config.ScanInterval.String(),
		"stop_until":      at.stopUntil.Format(time.RFC3339),
		"last_reset_time": at.lastResetTime.Format(time.RFC3339),
		"ai_provider":     aiProvider,
	}
}

// GetAccountInfo gets account information (for API)
func (at *AutoTrader) GetAccountInfo() (map[string]interface{}, error) {
	balance, err := at.trader.GetBalance()
	if err != nil {
		return nil, fmt.Errorf("failed to get balance: %w", err)
	}

	// Get account fields
	totalWalletBalance := 0.0
	totalUnrealizedProfit := 0.0
	availableBalance := 0.0
	totalEquity := 0.0

	if wallet, ok := balance["totalWalletBalance"].(float64); ok {
		totalWalletBalance = wallet
	}
	if unrealized, ok := balance["totalUnrealizedProfit"].(float64); ok {
		totalUnrealizedProfit = unrealized
	}
	if avail, ok := balance["availableBalance"].(float64); ok {
		availableBalance = avail
	}

	// Use totalEquity directly if provided by trader (more accurate)
	if eq, ok := balance["totalEquity"].(float64); ok && eq > 0 {
		totalEquity = eq
	} else {
		// Fallback: Total Equity = Wallet balance + Unrealized profit
		totalEquity = totalWalletBalance + totalUnrealizedProfit
	}

	// Get positions to calculate total margin
	positions, err := at.trader.GetPositions()
	if err != nil {
		return nil, fmt.Errorf("failed to get positions: %w", err)
	}

	totalMarginUsed := 0.0
	totalUnrealizedPnLCalculated := 0.0
	for _, pos := range positions {
		markPrice := pos["markPrice"].(float64)
		quantity := pos["positionAmt"].(float64)
		if quantity < 0 {
			quantity = -quantity
		}
		unrealizedPnl := pos["unRealizedProfit"].(float64)
		totalUnrealizedPnLCalculated += unrealizedPnl

		leverage := 10
		if lev, ok := pos["leverage"].(float64); ok {
			leverage = int(lev)
		}
		marginUsed := (quantity * markPrice) / float64(leverage)
		totalMarginUsed += marginUsed
	}

	// Verify unrealized P&L consistency (API value vs calculated from positions)
	// Note: Lighter API may return 0 for unrealized PnL, this is a known limitation
	diff := math.Abs(totalUnrealizedProfit - totalUnrealizedPnLCalculated)
	if diff > 5.0 { // Only warn if difference is significant (> 5 USDT)
		logger.Infof("⚠️ Unrealized P&L inconsistency (Lighter API limitation): API=%.4f, Calculated=%.4f, Diff=%.4f",
			totalUnrealizedProfit, totalUnrealizedPnLCalculated, diff)
	}

	totalPnL := totalEquity - at.initialBalance
	totalPnLPct := 0.0
	if at.initialBalance > 0 {
		totalPnLPct = (totalPnL / at.initialBalance) * 100
	} else {
		logger.Infof("⚠️ Initial Balance abnormal: %.2f, cannot calculate P&L percentage", at.initialBalance)
	}

	marginUsedPct := 0.0
	if totalEquity > 0 {
		marginUsedPct = (totalMarginUsed / totalEquity) * 100
	}

	return map[string]interface{}{
		// Core fields
		"total_equity":      totalEquity,           // Account equity = wallet + unrealized
		"wallet_balance":    totalWalletBalance,    // Wallet balance (excluding unrealized P&L)
		"unrealized_profit": totalUnrealizedProfit, // Unrealized P&L (official value from exchange API)
		"available_balance": availableBalance,      // Available balance

		// P&L statistics
		"total_pnl":       totalPnL,          // Total P&L = equity - initial
		"total_pnl_pct":   totalPnLPct,       // Total P&L percentage
		"initial_balance": at.initialBalance, // Initial balance
		"daily_pnl":       at.dailyPnL,       // Daily P&L

		// Position information
		"position_count":  len(positions),  // Position count
		"margin_used":     totalMarginUsed, // Margin used
		"margin_used_pct": marginUsedPct,   // Margin usage rate
	}, nil
}

// GetPositions gets position list (for API)
func (at *AutoTrader) GetPositions() ([]map[string]interface{}, error) {
	positions, err := at.trader.GetPositions()
	if err != nil {
		return nil, fmt.Errorf("failed to get positions: %w", err)
	}

	var result []map[string]interface{}
	for _, pos := range positions {
		symbol := pos["symbol"].(string)
		side := pos["side"].(string)
		entryPrice := pos["entryPrice"].(float64)
		markPrice := pos["markPrice"].(float64)
		quantity := pos["positionAmt"].(float64)
		if quantity < 0 {
			quantity = -quantity
		}
		unrealizedPnl := pos["unRealizedProfit"].(float64)
		liquidationPrice := pos["liquidationPrice"].(float64)

		leverage := 10
		if lev, ok := pos["leverage"].(float64); ok {
			leverage = int(lev)
		}

		// Calculate margin used
		marginUsed := (quantity * markPrice) / float64(leverage)

		// Calculate P&L percentage (based on margin)
		pnlPct := calculatePnLPercentage(unrealizedPnl, marginUsed)

		result = append(result, map[string]interface{}{
			"symbol":             symbol,
			"side":               side,
			"entry_price":        entryPrice,
			"mark_price":         markPrice,
			"quantity":           quantity,
			"leverage":           leverage,
			"unrealized_pnl":     unrealizedPnl,
			"unrealized_pnl_pct": pnlPct,
			"liquidation_price":  liquidationPrice,
			"margin_used":        marginUsed,
		})
	}

	return result, nil
}

// calculatePnLPercentage calculates P&L percentage (based on margin, automatically considers leverage)
// Return rate = Unrealized P&L / Margin × 100%
func calculatePnLPercentage(unrealizedPnl, marginUsed float64) float64 {
	if marginUsed > 0 {
		return (unrealizedPnl / marginUsed) * 100
	}
	return 0.0
}

// sortDecisionsByPriority sorts decisions: close positions first, then open positions, finally hold/wait
// This avoids position stacking overflow when changing positions
func sortDecisionsByPriority(decisions []kernel.Decision) []kernel.Decision {
	if len(decisions) <= 1 {
		return decisions
	}

	// Define priority
	getActionPriority := func(action string) int {
		switch action {
		case "close_long", "close_short":
			return 1 // Highest priority: close positions first
		case "open_long", "open_short":
			return 2 // Second priority: open positions later
		case "hold", "wait":
			return 3 // Lowest priority: wait
		default:
			return 999 // Unknown actions at the end
		}
	}

	// Copy decision list
	sorted := make([]kernel.Decision, len(decisions))
	copy(sorted, decisions)

	// Sort by priority
	for i := 0; i < len(sorted)-1; i++ {
		for j := i + 1; j < len(sorted); j++ {
			if getActionPriority(sorted[i].Action) > getActionPriority(sorted[j].Action) {
				sorted[i], sorted[j] = sorted[j], sorted[i]
			}
		}
	}

	return sorted
}

// startDrawdownMonitor starts drawdown monitoring
func (at *AutoTrader) startDrawdownMonitor() {
	at.monitorWg.Add(1)
	go func() {
		defer at.monitorWg.Done()

		ticker := time.NewTicker(1 * time.Minute) // Check every minute
		defer ticker.Stop()

		logger.Info("📊 Started position drawdown monitoring (check every minute)")

		for {
			select {
			case <-ticker.C:
				at.checkPositionDrawdown()
			case <-at.stopMonitorCh:
				logger.Info("⏹ Stopped position drawdown monitoring")
				return
			}
		}
	}()
}

// checkPositionDrawdown checks position drawdown situation
func (at *AutoTrader) checkPositionDrawdown() {
	// Get current positions
	positions, err := at.trader.GetPositions()
	if err != nil {
		logger.Infof("❌ Drawdown monitoring: failed to get positions: %v", err)
		return
	}

	for _, pos := range positions {
		symbol := pos["symbol"].(string)
		side := pos["side"].(string)
		entryPrice := pos["entryPrice"].(float64)
		markPrice := pos["markPrice"].(float64)
		quantity := pos["positionAmt"].(float64)
		if quantity < 0 {
			quantity = -quantity // Short position quantity is negative, convert to positive
		}

		// Calculate current P&L percentage
		leverage := 10 // Default value
		if lev, ok := pos["leverage"].(float64); ok {
			leverage = int(lev)
		}

		var currentPnLPct float64
		if side == "long" {
			currentPnLPct = ((markPrice - entryPrice) / entryPrice) * float64(leverage) * 100
		} else {
			currentPnLPct = ((entryPrice - markPrice) / entryPrice) * float64(leverage) * 100
		}

		// Construct unique position identifier (distinguish long/short)
		posKey := symbol + "_" + side

		// Get historical peak profit for this position
		at.peakPnLCacheMutex.RLock()
		peakPnLPct, exists := at.peakPnLCache[posKey]
		at.peakPnLCacheMutex.RUnlock()

		if !exists {
			// If no historical peak record, use current P&L as initial value
			peakPnLPct = currentPnLPct
			at.UpdatePeakPnL(symbol, side, currentPnLPct)
		} else {
			// Update peak cache
			at.UpdatePeakPnL(symbol, side, currentPnLPct)
		}

		// Calculate drawdown (magnitude of decline from peak)
		var drawdownPct float64
		if peakPnLPct > 0 && currentPnLPct < peakPnLPct {
			drawdownPct = ((peakPnLPct - currentPnLPct) / peakPnLPct) * 100
		}

		// Check close position condition: profit > 5% and drawdown >= 40%
		if currentPnLPct > 5.0 && drawdownPct >= 40.0 {
			logger.Infof("🚨 Drawdown close position condition triggered: %s %s | Current profit: %.2f%% | Peak profit: %.2f%% | Drawdown: %.2f%%",
				symbol, side, currentPnLPct, peakPnLPct, drawdownPct)

			// Execute close position
			if err := at.emergencyClosePosition(symbol, side); err != nil {
				logger.Infof("❌ Drawdown close position failed (%s %s): %v", symbol, side, err)
			} else {
				logger.Infof("✅ Drawdown close position succeeded: %s %s", symbol, side)
				// Clear cache for this position after closing
				at.ClearPeakPnLCache(symbol, side)
			}
		} else if currentPnLPct > 5.0 {
			// Record situations close to close position condition (for debugging)
			logger.Infof("📊 Drawdown monitoring: %s %s | Profit: %.2f%% | Peak: %.2f%% | Drawdown: %.2f%%",
				symbol, side, currentPnLPct, peakPnLPct, drawdownPct)
		}
	}
}

// emergencyClosePosition emergency close position function
func (at *AutoTrader) emergencyClosePosition(symbol, side string) error {
	switch side {
	case "long":
		order, err := at.trader.CloseLong(symbol, 0) // 0 = close all
		if err != nil {
			return err
		}
		logger.Infof("✅ Emergency close long position succeeded, order ID: %v", order["orderId"])
	case "short":
		order, err := at.trader.CloseShort(symbol, 0) // 0 = close all
		if err != nil {
			return err
		}
		logger.Infof("✅ Emergency close short position succeeded, order ID: %v", order["orderId"])
	default:
		return fmt.Errorf("unknown position direction: %s", side)
	}

	return nil
}

// sendEmergencyAlert 发送紧急警报（集成日志、数据库记录、可扩展通知）
func (at *AutoTrader) sendEmergencyAlert(symbol, side, reason string) {
	// 1. 高级别日志警报
	logger.Errorf("🚨🚨🚨 EMERGENCY ALERT 🚨🚨🚨")
	logger.Errorf("Trader: %s | Exchange: %s", at.id, at.exchange)
	logger.Errorf("Symbol: %s %s | Reason: %s", symbol, side, reason)
	logger.Errorf("Time: %s", time.Now().UTC().Format(time.RFC3339))

	// 2. 写入数据库警报记录（用于后续审计）
	// 注意：这需要在store中添加AlertLog表，这里先预留接口
	// if err := at.store.Alerts().CreateAlert(at.id, symbol, side, reason); err != nil {

	// 3. 未来可扩展：发送到通知服务（Telegram、Email、企业微信等）
	// 例如: at.notificationService.Send(alertMessage)
}

// GetPeakPnLCache gets peak profit cache
func (at *AutoTrader) GetPeakPnLCache() map[string]float64 {
	at.peakPnLCacheMutex.RLock()
	defer at.peakPnLCacheMutex.RUnlock()

	// Return a copy of the cache
	cache := make(map[string]float64)
	for k, v := range at.peakPnLCache {
		cache[k] = v
	}
	return cache
}

// UpdatePeakPnL updates peak profit cache
func (at *AutoTrader) UpdatePeakPnL(symbol, side string, currentPnLPct float64) {
	at.peakPnLCacheMutex.Lock()
	defer at.peakPnLCacheMutex.Unlock()

	posKey := symbol + "_" + side
	if peak, exists := at.peakPnLCache[posKey]; exists {
		// Update peak (if long, take larger value; if short, currentPnLPct is negative, also compare)
		if currentPnLPct > peak {
			at.peakPnLCache[posKey] = currentPnLPct
		}
	} else {
		// First time recording
		at.peakPnLCache[posKey] = currentPnLPct
	}
}

// ClearPeakPnLCache clears peak cache for specified position
func (at *AutoTrader) ClearPeakPnLCache(symbol, side string) {
	at.peakPnLCacheMutex.Lock()
	defer at.peakPnLCacheMutex.Unlock()

	posKey := symbol + "_" + side
	delete(at.peakPnLCache, posKey)
}

// recordAndConfirmOrder polls order status for actual fill data and records position
// action: open_long, open_short, close_long, close_short
// entryPrice: entry price when closing (0 when opening)
// tp, sl: take profit and stop loss (0 when closing or not applicable)
func (at *AutoTrader) recordAndConfirmOrder(orderResult map[string]interface{}, symbol, action string, quantity float64, price float64, leverage int, entryPrice float64, tp float64, sl float64) {
	if at.store == nil {
		return
	}

	// Get order ID (supports multiple types)
	var orderID string
	switch v := orderResult["orderId"].(type) {
	case int64:
		orderID = fmt.Sprintf("%d", v)
	case float64:
		orderID = fmt.Sprintf("%.0f", v)
	case string:
		orderID = v
	default:
		orderID = fmt.Sprintf("%v", v)
	}

	if orderID == "" || orderID == "0" {
		logger.Infof("  ⚠️ Order ID is empty, skipping record")
		return
	}

	// Determine positionSide
	var positionSide string
	switch action {
	case "open_long", "close_long":
		positionSide = "LONG"
	case "open_short", "close_short":
		positionSide = "SHORT"
	}

	var actualPrice = price
	var actualQty = quantity
	var fee float64

	// Exchanges with OrderSync: Skip immediate order recording, let OrderSync handle it
	// This ensures accurate data from GetTrades API and avoids duplicate records
	switch at.exchange {
	case "binance", "lighter", "hyperliquid", "bybit", "okx", "bitget", "aster":
		logger.Infof("  📝 Order submitted (id: %s), will be synced by OrderSync", orderID)
		return
	}

	// For exchanges without OrderSync (e.g., Binance): record immediately and poll for fill data
	orderRecord := at.createOrderRecord(orderID, symbol, action, positionSide, quantity, price, leverage)
	if err := at.store.Order().CreateOrder(orderRecord); err != nil {
		logger.Infof("  ⚠️ Failed to record order: %v", err)
	} else {
		logger.Infof("  📝 Order recorded: %s [%s] %s", orderID, action, symbol)
	}

	// Wait for order to be filled and get actual fill data
	time.Sleep(500 * time.Millisecond)
	var finalStatus string
	for i := 0; i < 5; i++ {
		status, err := at.trader.GetOrderStatus(symbol, orderID)
		if err == nil {
			statusStr, _ := status["status"].(string)
			finalStatus = statusStr
			if statusStr == "FILLED" {
				// Get actual fill price
				if avgPrice, ok := status["avgPrice"].(float64); ok && avgPrice > 0 {
					actualPrice = avgPrice
				}
				// Get actual executed quantity
				if execQty, ok := status["executedQty"].(float64); ok && execQty > 0 {
					actualQty = execQty
				}
				// Get commission/fee
				if commission, ok := status["commission"].(float64); ok {
					fee = commission
				}
				logger.Infof("  ✅ Order filled: avgPrice=%.6f, qty=%.6f, fee=%.6f", actualPrice, actualQty, fee)

				// Update order status to FILLED
				if err := at.store.Order().UpdateOrderStatus(orderRecord.ID, "FILLED", actualQty, actualPrice, fee); err != nil {
					logger.Infof("  ⚠️ Failed to update order status: %v", err)
				}

				// Record fill details
				at.recordOrderFill(orderRecord.ID, orderID, symbol, action, actualPrice, actualQty, fee)
				break
			} else if statusStr == "PARTIALLY_FILLED" {
				// Handle partial fill - record what's filled so far
				if execQty, ok := status["executedQty"].(float64); ok && execQty > 0 {
					actualQty = execQty
				}
				if avgPrice, ok := status["avgPrice"].(float64); ok && avgPrice > 0 {
					actualPrice = avgPrice
				}
				logger.Infof("  ⚠️ Order partially filled: qty=%.6f/%.6f, continuing to poll...", actualQty, quantity)
				// Continue polling for full fill
			} else if statusStr == "CANCELED" || statusStr == "EXPIRED" || statusStr == "REJECTED" {
				logger.Infof("  ⚠️ Order %s, skipping position record", statusStr)

				// Update order status
				if err := at.store.Order().UpdateOrderStatus(orderRecord.ID, statusStr, 0, 0, 0); err != nil {
					logger.Infof("  ⚠️ Failed to update order status: %v", err)
				}
				return
			}
		}
		time.Sleep(500 * time.Millisecond)
	}

	// Handle timeout - order still not fully filled after polling
	if finalStatus != "FILLED" && finalStatus != "" {
		logger.Warnf("  ⚠️ Order %s did not complete within timeout, final status: %s", orderID, finalStatus)
		if finalStatus == "PARTIALLY_FILLED" && actualQty > 0 {
			// Record partial fill
			logger.Infof("  📝 Recording partial fill: qty=%.6f", actualQty)
			if err := at.store.Order().UpdateOrderStatus(orderRecord.ID, "PARTIALLY_FILLED", actualQty, actualPrice, fee); err != nil {
				logger.Infof("  ⚠️ Failed to update order status: %v", err)
			}
		} else if finalStatus == "NEW" {
			// Order still pending - consider canceling
			logger.Warnf("  ⚠️ Order still pending after timeout, may need manual attention")
		}
	}

	// Normalize symbol for position record consistency
	normalizedSymbolForPosition := market.Normalize(symbol)

	logger.Infof("  📝 Recording position (ID: %s, action: %s, price: %.6f, qty: %.6f, fee: %.4f)",
		orderID, action, actualPrice, actualQty, fee)

	// Record position change with actual fill data (use normalized symbol)
	at.recordPositionChange(orderID, normalizedSymbolForPosition, positionSide, action, actualQty, actualPrice, leverage, entryPrice, fee, tp, sl)

	// Send anonymous trade statistics for experience improvement (async, non-blocking)
	// This helps us understand overall product usage across all deployments
	experience.TrackTrade(experience.TradeEvent{
		Exchange:  at.exchange,
		TradeType: action,
		Symbol:    symbol,
		AmountUSD: actualPrice * actualQty,
		Leverage:  leverage,
		UserID:    at.userID,
		TraderID:  at.id,
	})
}

// recordPositionChange records position change (create record on open, update record on close)
func (at *AutoTrader) recordPositionChange(orderID, symbol, side, action string, quantity, price float64, leverage int, entryPrice float64, fee float64, tp float64, sl float64) {
	if at.store == nil {
		return
	}

	switch action {
	case "open_long", "open_short":
		// Open position: create new position record
		nowMs := time.Now().UTC().UnixMilli()
		pos := &store.TraderPosition{
			TraderID:     at.id,
			ExchangeID:   at.exchangeID, // Exchange account UUID
			ExchangeType: at.exchange,   // Exchange type: binance/bybit/okx/etc
			Symbol:       symbol,
			Side:         side, // LONG or SHORT
			Quantity:     quantity,
			EntryPrice:   price,
			EntryOrderID: orderID,
			EntryTime:    nowMs,
			Leverage:     leverage,
			Status:       "OPEN",
			CreatedAt:    nowMs,
			UpdatedAt:    nowMs,
		}
		if err := at.store.Position().Create(pos); err != nil {
			logger.Infof("  ⚠️ Failed to record position: %v", err)
		} else {
			logger.Infof("  📊 Position recorded [%s] %s %s @ %.4f", at.id[:8], symbol, side, price)

			// Record TP/SL for the new position (only when tp and sl are provided)
			if tp > 0 && sl > 0 {
				if err := at.recordTPSL(at.id, pos, tp, sl); err != nil {
					logger.Warnf("  ⚠️ Failed to record TP/SL: %v", err)
				} else {
					logger.Infof("  ✅ TP/SL recorded: TP=%.2f, SL=%.2f", tp, sl)
				}
			}
		}

	case "close_long", "close_short":
		// Close position using PositionBuilder for consistent handling
		// PositionBuilder will handle both cases:
		// 1. If open position exists: close it properly
		// 2. If no open position (e.g., table cleared): create a closed position record
		posBuilder := store.NewPositionBuilder(at.store.Position())
		if err := posBuilder.ProcessTrade(
			at.id, at.exchangeID, at.exchange,
			symbol, side, action,
			quantity, price, fee, 0, // realizedPnL will be calculated
			time.Now().UTC().UnixMilli(), orderID,
		); err != nil {
			logger.Infof("  ⚠️ Failed to process close position: %v", err)
		} else {
			logger.Infof("  ✅ Position closed [%s] %s %s @ %.4f", at.id[:8], symbol, side, price)
		}

		// 关闭对应的ASL记录
		if err := at.store.AdaptiveStopLoss().CloseRecordBySymbol(at.id, symbol); err != nil {
			logger.Warnf("  ⚠️ Failed to close ASL record: %v", err)
		} else {
			logger.Infof("  ✅ ASL record closed for %s", symbol)
		}

		// Option B: Record trade outcome for metrics updates
		if at.enhancedSetup != nil {
			// For now, record 0 PnL (actual PnL will be available when position closes)
			// In the future, fetch actual PnL from database when position closes
			at.enhancedSetup.FundManagement.RecordTrade(0)
			logger.Infof("  📈 Trade outcome recorded for performance metrics")
		}
	}
}

// createOrderRecord creates an order record struct from order details
func (at *AutoTrader) createOrderRecord(orderID, symbol, action, positionSide string, quantity, price float64, leverage int) *store.TraderOrder {
	// Determine order type (market for auto trader)
	orderType := "MARKET"

	// Determine side (BUY/SELL)
	var side string
	switch action {
	case "open_long", "close_short":
		side = "BUY"
	case "open_short", "close_long":
		side = "SELL"
	}

	// Use action as orderAction directly (keep lowercase format)
	orderAction := action

	// Determine if it's a reduce only order
	reduceOnly := (action == "close_long" || action == "close_short")

	// Normalize symbol for consistency
	normalizedSymbol := market.Normalize(symbol)

	return &store.TraderOrder{
		TraderID:        at.id,
		ExchangeID:      at.exchangeID,
		ExchangeType:    at.exchange,
		ExchangeOrderID: orderID,
		Symbol:          normalizedSymbol,
		Side:            side,
		PositionSide:    positionSide,
		Type:            orderType,
		TimeInForce:     "GTC",
		Quantity:        quantity,
		Price:           price,
		Status:          "NEW",
		FilledQuantity:  0,
		AvgFillPrice:    0,
		Commission:      0,
		CommissionAsset: "USDT",
		Leverage:        leverage,
		ReduceOnly:      reduceOnly,
		ClosePosition:   reduceOnly,
		OrderAction:     orderAction,
		CreatedAt:       time.Now().UTC().UnixMilli(),
		UpdatedAt:       time.Now().UTC().UnixMilli(),
	}
}

// recordOrderFill records order fill/trade details
func (at *AutoTrader) recordOrderFill(orderRecordID int64, exchangeOrderID, symbol, action string, price, quantity, fee float64) {
	if at.store == nil {
		return
	}

	// Determine side (BUY/SELL)
	var side string
	switch action {
	case "open_long", "close_short":
		side = "BUY"
	case "open_short", "close_long":
		side = "SELL"
	}

	// Generate a simple trade ID (exchange doesn't always provide one)
	tradeID := fmt.Sprintf("%s-%d", exchangeOrderID, time.Now().UnixNano())

	// Normalize symbol for consistency
	normalizedSymbol := market.Normalize(symbol)

	fill := &store.TraderFill{
		TraderID:        at.id,
		ExchangeID:      at.exchangeID,
		ExchangeType:    at.exchange,
		OrderID:         orderRecordID,
		ExchangeOrderID: exchangeOrderID,
		ExchangeTradeID: tradeID,
		Symbol:          normalizedSymbol,
		Side:            side,
		Price:           price,
		Quantity:        quantity,
		QuoteQuantity:   price * quantity,
		Commission:      fee,
		CommissionAsset: "USDT",
		RealizedPnL:     0,     // Will be calculated for close orders
		IsMaker:         false, // Market orders are usually taker
		CreatedAt:       time.Now().UTC().UnixMilli(),
	}

	// Calculate realized PnL for close orders
	if action == "close_long" || action == "close_short" {
		// Try to get the entry price from the open position
		var positionSide string
		if action == "close_long" {
			positionSide = "LONG"
		} else {
			positionSide = "SHORT"
		}

		if openPos, err := at.store.Position().GetOpenPositionBySymbol(at.id, symbol, positionSide); err == nil && openPos != nil {
			if positionSide == "LONG" {
				fill.RealizedPnL = (price - openPos.EntryPrice) * quantity
			} else {
				fill.RealizedPnL = (openPos.EntryPrice - price) * quantity
			}
		}
	}

	if err := at.store.Order().CreateFill(fill); err != nil {
		logger.Infof("  ⚠️ Failed to record fill: %v", err)
	} else {
		logger.Infof("  📋 Fill recorded: %.4f @ %.6f, fee: %.4f", quantity, price, fee)
	}
}

// ============================================================================
// Risk Control Helpers
// ============================================================================

// isBTCETH checks if a symbol is BTC or ETH
func isBTCETH(symbol string) bool {
	symbol = strings.ToUpper(symbol)
	return strings.HasPrefix(symbol, "BTC") || strings.HasPrefix(symbol, "ETH")
}

// enforcePositionValueRatio checks and enforces position value ratio limits (CODE ENFORCED)
// Returns the adjusted position size (capped if necessary) and whether the position was capped
// positionSizeUSD: the original position size in USD
// equity: the account equity
// symbol: the trading symbol
func (at *AutoTrader) enforcePositionValueRatio(positionSizeUSD float64, equity float64, symbol string) (float64, bool) {
	if at.config.StrategyConfig == nil {
		return positionSizeUSD, false
	}

	riskControl := at.config.StrategyConfig.RiskControl

	// Get the appropriate position value ratio limit
	var maxPositionValueRatio float64
	if isBTCETH(symbol) {
		maxPositionValueRatio = riskControl.BTCETHMaxPositionValueRatio
		if maxPositionValueRatio <= 0 {
			maxPositionValueRatio = 5.0 // Default: 5x for BTC/ETH
		}
	} else {
		maxPositionValueRatio = riskControl.AltcoinMaxPositionValueRatio
		if maxPositionValueRatio <= 0 {
			maxPositionValueRatio = 1.0 // Default: 1x for altcoins
		}
	}

	// Calculate max allowed position value = equity × ratio
	maxPositionValue := equity * maxPositionValueRatio

	// Check if position size exceeds limit
	if positionSizeUSD > maxPositionValue {
		logger.Infof("  ⚠️ [RISK CONTROL] Position %.2f USDT exceeds limit (equity %.2f × %.1fx = %.2f USDT max for %s), capping",
			positionSizeUSD, equity, maxPositionValueRatio, maxPositionValue, symbol)
		return maxPositionValue, true
	}

	return positionSizeUSD, false
}

// enforceMinPositionSize checks minimum position size (CODE ENFORCED)
func (at *AutoTrader) enforceMinPositionSize(positionSizeUSD float64) error {
	if at.config.StrategyConfig == nil {
		return nil
	}

	minSize := at.config.StrategyConfig.RiskControl.MinPositionSize
	if minSize <= 0 {
		minSize = 12 // Default: 12 USDT
	}

	if positionSizeUSD < minSize {
		return fmt.Errorf("❌ [RISK CONTROL] Position %.2f USDT below minimum (%.2f USDT)", positionSizeUSD, minSize)
	}
	return nil
}

// enforceMaxPositions checks maximum positions count (CODE ENFORCED)
func (at *AutoTrader) enforceMaxPositions(currentPositionCount int) error {
	if at.config.StrategyConfig == nil {
		return nil
	}

	maxPositions := at.config.StrategyConfig.RiskControl.MaxPositions
	if maxPositions <= 0 {
		maxPositions = 3 // Default: 3 positions
	}

	if currentPositionCount >= maxPositions {
		return fmt.Errorf("❌ [RISK CONTROL] Already at max positions (%d/%d)", currentPositionCount, maxPositions)
	}
	return nil
}

// getSideFromAction converts order action to side (BUY/SELL)
func getSideFromAction(action string) string {
	switch action {
	case "open_long", "close_short":
		return "BUY"
	case "open_short", "close_long":
		return "SELL"
	default:
		return "BUY"
	}
}

// updateDynamicStopLoss 动态更新所有持仓的止损
// 每个交易周期调用，实现：
// 1. 每十秒一次接近买入点
// 2. 五分钟之内和买入点重合
// 3. 如果在盈利的话，那就以相同的速度和方向继续移动
func (at *AutoTrader) updateDynamicStopLoss(ctx *kernel.Context) {
	if at.enhancedSetup == nil || at.enhancedSetup.AdaptiveStopLoss == nil {
		return
	}

	if len(ctx.Positions) == 0 {
		return
	}

	logger.Info("📊 [Dynamic SL] Checking stop loss adjustments for open positions...")

	// 🔥 新增：每 5 个周期验证一次 TP/SL 同步
	at.cycleCount++
	if at.cycleCount%5 == 0 {
		if err := at.VerifyTPSLSync(ctx); err != nil {
			logger.Warnf("⚠️ [TPSLSync] Periodic check failed: %v", err)
		}
	}

	for _, pos := range ctx.Positions {
		// 获取市场数据
		marketData, err := market.Get(pos.Symbol)
		if err != nil {
			logger.Warnf("⚠️ [Dynamic SL] Failed to get market data for %s: %v", pos.Symbol, err)
			continue
		}

		// 计算简化的 ATR（用于初始设置）
		atrValue := at.calculateSimpleATR(pos.Symbol, marketData)

		// 调用动态止损更新（新的逻辑）
		newSL, _, updated := at.enhancedSetup.ApplyDynamicStopLoss(
			pos.Symbol,
			marketData.CurrentPrice,
			atrValue,
		)

		if !updated {
			// 该持仓未被追踪，先注册
			at.enhancedSetup.AdaptiveStopLoss.SetStopLevelForPosition(
				pos.Symbol,
				pos.EntryPrice,
				at.calculateInitialStopLoss(pos),
				at.calculateInitialTakeProfit(pos),
				atrValue,
			)
			logger.Infof("  📝 [%s] Registered for dynamic tracking | Entry: %.4f", pos.Symbol, pos.EntryPrice)

			// 保存初始状态到数据库
			if at.store != nil {
				at.saveDynamicStopLossRecord(pos, marketData.CurrentPrice, pos.EntryPrice, pos.EntryPrice, 0)
			}
			continue
		}

		// 获取当前交易所的止损价
		currentExchangeSL := at.getCurrentExchangeStopLoss(pos.Symbol, pos.Side)

		// 如果新止损价更优（做多时更高，做空时更低），则更新交易所订单
		shouldUpdate := false
		if pos.Side == "long" && newSL > currentExchangeSL && newSL < marketData.CurrentPrice {
			shouldUpdate = true
		} else if pos.Side == "short" && newSL < currentExchangeSL && newSL > marketData.CurrentPrice {
			shouldUpdate = true
		}

		if shouldUpdate {
			logger.Infof("  🔄 [%s] Updating stop loss: %.4f → %.4f (%.2f%% move)",
				pos.Symbol, currentExchangeSL, newSL,
				((newSL-currentExchangeSL)/currentExchangeSL)*100)

			// 更新交易所止损单
			if err := at.updateExchangeStopLoss(pos.Symbol, pos.Side, pos.Quantity, newSL); err != nil {
				logger.Warnf("  ⚠️ [%s] Failed to update exchange stop loss: %v", pos.Symbol, err)
			} else {
				logger.Infof("  ✅ [%s] Exchange stop loss updated to %.4f", pos.Symbol, newSL)

				// 保存动态止损更新记录到数据库
				if at.store != nil {
					// 计算移动距离和方向
					移动距离 := math.Abs(newSL - pos.EntryPrice)
					移动方向 := "toward_entry" // 向入场点移动
					if pos.Side == "long" && newSL > pos.EntryPrice {
						移动方向 = "away_from_entry" // 远离入场点（盈利追踪）
					} else if pos.Side == "short" && newSL < pos.EntryPrice {
						移动方向 = "away_from_entry" // 远离入场点（盈利追踪）
					}

					// 计算运行时间（秒）
					运行时间 := time.Since(time.UnixMilli(pos.UpdateTime)).Seconds()

					at.saveDynamicStopLossRecord(pos, marketData.CurrentPrice, newSL, pos.EntryPrice, 运行时间)
					logger.Infof("  💾 [%s] Dynamic SL state saved: SL=%.4f, Distance=%.4f, Direction=%s, Time=%.1fs",
						pos.Symbol, newSL, 移动距离, 移动方向, 运行时间)
				}
			}
		} else {
			logger.Debugf("  ⏭️ [%s] No stop loss update needed | Current: %.4f | Calculated: %.4f",
				pos.Symbol, currentExchangeSL, newSL)
		}

		// 注意：止盈保持不变，不更新 newTP
	}
}

// saveDynamicStopLossRecord 保存动态止损记录到数据库
func (at *AutoTrader) saveDynamicStopLossRecord(pos kernel.PositionInfo, currentPrice, newSL, entryPrice, elapsedTime float64) {
	if at.store == nil {
		return
	}

	// 计算进度（0-1，1表示已到达入场价）
	进度 := 0.0
	if pos.Side == "long" {
		初始距离 := math.Abs(pos.EntryPrice - at.calculateInitialStopLoss(pos))
		if 初始距离 > 0 {
			进度 = 1.0 - (math.Abs(newSL-entryPrice) / 初始距离)
		}
	} else {
		初始距离 := math.Abs(at.calculateInitialStopLoss(pos) - pos.EntryPrice)
		if 初始距离 > 0 {
			进度 = 1.0 - (math.Abs(newSL-entryPrice) / 初始距离)
		}
	}
	if 进度 < 0 {
		进度 = 0
	}
	if 进度 > 1 {
		进度 = 1
	}

	// 判断是否在盈利追踪模式
	盈利追踪 := false
	if pos.Side == "long" && newSL > entryPrice {
		盈利追踪 = true
	} else if pos.Side == "short" && newSL < entryPrice {
		盈利追踪 = true
	}

	record := &store.AdaptiveStopLossRecord{
		ID:              fmt.Sprintf("%s_%s", at.id, pos.Symbol),
		TraderID:        at.id,
		Symbol:          pos.Symbol,
		PositionID:      fmt.Sprintf("%s_%s", pos.Symbol, pos.Side),
		EntryPrice:      entryPrice,
		CurrentStopLoss: newSL,
		InitialStopLoss: at.calculateInitialStopLoss(pos),
		TakeProfit:      at.calculateInitialTakeProfit(pos),
		CurrentPrice:    currentPrice,
		IsInProfit:      盈利追踪,
		ProfitDistance:  math.Abs(newSL - entryPrice),
		TimeProgression: 进度,
		ElapsedSeconds:  int(elapsedTime),
		Status:          "ACTIVE",
		CreatedAt:       time.Now().UTC(),
		UpdatedAt:       time.Now().UTC(),
	}

	if err := at.store.AdaptiveStopLoss().SaveRecord(record); err != nil {
		logger.Warnf("  ⚠️ Failed to save dynamic SL record: %v", err)
	}
}

// calculateSimpleATR 计算简化的 ATR（基于 K 线数据）
func (at *AutoTrader) calculateSimpleATR(symbol string, marketData *market.Data) float64 {
	// 优先使用已计算的 ATR14
	if marketData != nil && marketData.TimeframeData != nil {
		// 尝试从 5 分钟时间框架获取 ATR
		if tf5m, ok := marketData.TimeframeData["5m"]; ok && tf5m.ATR14 > 0 {
			return tf5m.ATR14
		}
		// 尝试从 15 分钟时间框架获取 ATR
		if tf15m, ok := marketData.TimeframeData["15m"]; ok && tf15m.ATR14 > 0 {
			return tf15m.ATR14
		}
		// 使用任何可用的时间框架
		for _, tfData := range marketData.TimeframeData {
			if tfData.ATR14 > 0 {
				return tfData.ATR14
			}
			// 如果没有 ATR，从 K 线数据计算
			if len(tfData.Klines) >= 14 {
				var totalTR float64
				klines := tfData.Klines
				period := 14
				for i := len(klines) - period; i < len(klines); i++ {
					k := klines[i]
					tr := k.High - k.Low
					totalTR += tr
				}
				return totalTR / float64(period)
			}
		}
	}

	// 🔥 修复：使用动态波动率计算，而非固定百分比
	if marketData != nil {
		// 获取最近的价格变化范围作为ATR近似值
		// 优先使用最近14个K线的最高-最低价范围
		if marketData.TimeframeData != nil {
			for _, tfData := range marketData.TimeframeData {
				if len(tfData.Klines) >= 14 {
					// 计算最近14根K线的最高-最低价范围
					var maxHigh, minLow float64
					klines := tfData.Klines[len(tfData.Klines)-14:]
					for i, k := range klines {
						if i == 0 {
							maxHigh = k.High
							minLow = k.Low
						} else {
							if k.High > maxHigh {
								maxHigh = k.High
							}
							if k.Low < minLow {
								minLow = k.Low
							}
						}
					}
					// 使用价格范围作为ATR近似值
					priceRange := maxHigh - minLow
					if priceRange > 0 {
						return priceRange
					}
				}
			}
		}

		// 最终回退：使用当前价格的动态百分比（基于近期波动）
		// 这里使用1.5%作为保守估计，比原来的2%更合理
		return marketData.CurrentPrice * 0.015
	}
	return 0
}

// calculateInitialStopLoss 计算初始止损价（用于未追踪的持仓）
func (at *AutoTrader) calculateInitialStopLoss(pos kernel.PositionInfo) float64 {
	// 默认使用 3% 止损
	stopLossPct := 0.03
	if pos.Side == "long" {
		return pos.EntryPrice * (1 - stopLossPct)
	}
	return pos.EntryPrice * (1 + stopLossPct)
}

// calculateInitialTakeProfit 计算初始止盈价（用于未追踪的持仓）
func (at *AutoTrader) calculateInitialTakeProfit(pos kernel.PositionInfo) float64 {
	// 默认使用 6% 止盈（2:1 风险收益比）
	takeProfitPct := 0.06
	if pos.Side == "long" {
		return pos.EntryPrice * (1 + takeProfitPct)
	}
	return pos.EntryPrice * (1 - takeProfitPct)
}

// getCurrentExchangeStopLoss 获取当前交易所的止损价格
func (at *AutoTrader) getCurrentExchangeStopLoss(symbol, side string) float64 {
	orders, err := at.trader.GetOpenOrders(symbol)
	if err != nil {
		return 0
	}

	for _, order := range orders {
		// 查找止损单
		if order.Type == "STOP_MARKET" || order.Type == "STOP" || order.Type == "StopLoss" {
			// 对于做多，止损是卖单；对于做空，止损是买单
			if (side == "long" && order.Side == "SELL") || (side == "short" && order.Side == "BUY") {
				return order.StopPrice
			}
		}
	}

	return 0 // 没有找到止损单
}

// updateExchangeStopLoss 更新交易所的止损订单
func (at *AutoTrader) updateExchangeStopLoss(symbol, side string, quantity, newStopPrice float64) error {
	// 1. 先取消现有的止损单
	if err := at.trader.CancelStopLossOrders(symbol); err != nil {
		logger.Warnf("⚠️ Failed to cancel existing stop loss orders: %v", err)
		// 继续尝试设置新的止损
	}

	// 2. 设置新的止损单
	positionSide := "LONG"
	if side == "short" {
		positionSide = "SHORT"
	}

	return at.trader.SetStopLoss(symbol, positionSide, quantity, newStopPrice)
}

// recordTPSL 记录止盈止损
func (at *AutoTrader) recordTPSL(traderID string, position *store.TraderPosition, tp, sl float64) error {
	tpslRecord := &store.TPSLRecord{
		TraderID:      traderID,
		PositionID:    position.ID,
		Symbol:        position.Symbol,
		Side:          position.Side,
		CurrentTP:     tp,
		CurrentSL:     sl,
		OriginalTP:    tp,
		OriginalSL:    sl,
		EntryPrice:    position.EntryPrice,
		EntryQuantity: position.Quantity,
		Status:        "ACTIVE",
		CreatedAt:     time.Now().UTC(),
	}
	return at.store.TPSL().SaveTPSLRecord(tpslRecord)
}

// getLatestTPSL 获取最新的 TP/SL 值（来自当前决策）
func (at *AutoTrader) getLatestTPSL(symbol string) (float64, float64, bool) {
	// 这是一个占位符实现 - 实际上应该从当前的决策或执行上下文中获取
	// 为了简单起见，我们在这里返回默认值
	// 实际的 TP/SL 会由 executeOpenLongWithRecord/executeOpenShortWithRecord 调用 recordTPSL
	return 0, 0, false
}

// GetOpenOrders returns open orders (pending SL/TP) from exchange
func (at *AutoTrader) GetOpenOrders(symbol string) ([]OpenOrder, error) {
	return at.trader.GetOpenOrders(symbol)
}
