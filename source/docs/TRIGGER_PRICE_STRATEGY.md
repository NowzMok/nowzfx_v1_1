# 触发价格策略配置文档

## 问题背景

当前系统的触发价格计算是硬编码的，没有考虑不同交易员的风格差异，导致：
- 长线交易员可能觉得触发太敏感
- 短线交易员可能觉得触发太迟钝

## 解决方案

将触发价格计算逻辑移至**策略制定部分**，支持针对不同交易风格的个性化配置。

## 策略配置结构

### 1. 触发价格策略类型

```go
type TriggerPriceStrategy struct {
    // 交易风格
    TradingStyle string `json:"trading_style"` // "long_term" | "short_term" | "swing" | "scalp"
    
    // 开多触发策略
    OpenLongTrigger struct {
        Mode          string  `json:"mode"`           // "current_price" | "pullback" | "breakout"
        PullbackRatio float64 `json:"pullback_ratio"` // 回调比例 (0.01-0.10)
        BreakoutRatio float64 `json:"breakout_ratio"` // 突破比例 (0.01-0.05)
    } `json:"open_long_trigger"`
    
    // 开空触发策略
    OpenShortTrigger struct {
        Mode          string  `json:"mode"`           // "current_price" | "pullback" | "breakout"
        PullbackRatio float64 `json:"pullback_ratio"` // 回调比例 (0.01-0.10)
        BreakoutRatio float64 `json:"breakout_ratio"` // 突破比例 (0.01-0.05)
    } `json:"open_short_trigger"`
    
    // 触发价格计算参数
    Parameters struct {
        // 当前价格模式参数
        UseCurrentPrice bool `json:"use_current_price"` // 是否使用当前价格
        
        // 回调模式参数
        UseStopLossAsTrigger bool    `json:"use_stop_loss_as_trigger"` // 使用止损价作为触发价
        AdditionalBuffer     float64 `json:"additional_buffer"`         // 额外缓冲比例
        
        // 突破模式参数
        BreakoutThreshold float64 `json:"breakout_threshold"` // 突破阈值
        WaitForConfirmation bool  `json:"wait_for_confirmation"` // 是否等待确认
    } `json:"parameters"`
}
```

### 2. 预设配置模板

#### 长线交易员 (Long-term Trader)
```json
{
    "trading_style": "long_term",
    "open_long_trigger": {
        "mode": "pullback",
        "pullback_ratio": 0.05,
        "breakout_ratio": 0.02
    },
    "open_short_trigger": {
        "mode": "pullback",
        "pullback_ratio": 0.05,
        "breakout_ratio": 0.02
    },
    "parameters": {
        "use_current_price": false,
        "use_stop_loss_as_trigger": true,
        "additional_buffer": 0.01,
        "breakout_threshold": 0.02,
        "wait_for_confirmation": true
    }
}
```

**特点**：
- 等待更好的回调机会
- 使用止损价作为触发价
- 需要更多确认信号
- 适合波段操作

#### 短线交易员 (Short-term Trader)
```json
{
    "trading_style": "short_term",
    "open_long_trigger": {
        "mode": "current_price",
        "pullback_ratio": 0.01,
        "breakout_ratio": 0.01
    },
    "open_short_trigger": {
        "mode": "current_price",
        "pullback_ratio": 0.01,
        "breakout_ratio": 0.01
    },
    "parameters": {
        "use_current_price": true,
        "use_stop_loss_as_trigger": false,
        "additional_buffer": 0.005,
        "breakout_threshold": 0.01,
        "wait_for_confirmation": false
    }
}
```

**特点**：
- 快速响应，使用当前价格
- 小回调即入场
- 最小化确认要求
- 适合日内交易

#### 摆动交易员 (Swing Trader)
```json
{
    "trading_style": "swing",
    "open_long_trigger": {
        "mode": "pullback",
        "pullback_ratio": 0.03,
        "breakout_ratio": 0.015
    },
    "open_short_trigger": {
        "mode": "pullback",
        "pullback_ratio": 0.03,
        "breakout_ratio": 0.015
    },
    "parameters": {
        "use_current_price": false,
        "use_stop_loss_as_trigger": true,
        "additional_buffer": 0.008,
        "breakout_threshold": 0.015,
        "wait_for_confirmation": true
    }
}
```

**特点**：
- 中等回调等待
- 平衡风险与机会
- 适度确认要求

#### 剥头皮交易员 (Scalp Trader)
```json
{
    "trading_style": "scalp",
    "open_long_trigger": {
        "mode": "current_price",
        "pullback_ratio": 0.005,
        "breakout_ratio": 0.005
    },
    "open_short_trigger": {
        "mode": "current_price",
        "pullback_ratio": 0.005,
        "breakout_ratio": 0.005
    },
    "parameters": {
        "use_current_price": true,
        "use_stop_loss_as_trigger": false,
        "additional_buffer": 0.002,
        "breakout_threshold": 0.005,
        "wait_for_confirmation": false
    }
}
```

**特点**：
- 极速响应
- 极小回调
- 无需确认
- 适合高频交易

## 触发价格计算逻辑

### 1. 当前价格模式 (Current Price Mode)
```go
func calculateCurrentPriceTrigger(
    currentPrice float64,
    action string,
    config TriggerPriceStrategy,
) float64 {
    if action == "open_long" {
        return currentPrice
    } else { // open_short
        return currentPrice
    }
}
```

### 2. 回调模式 (Pullback Mode)
```go
func calculatePullbackTrigger(
    currentPrice float64,
    stopLoss float64,
    action string,
    config TriggerPriceStrategy,
) float64 {
    if action == "open_long" {
        // 开多：等待价格回调到止损价附近
        if stopLoss > 0 && config.Parameters.UseStopLossAsTrigger {
            // 止损价 + 额外缓冲
            buffer := currentPrice * config.Parameters.AdditionalBuffer
            return stopLoss + buffer
        } else {
            // 或者使用当前价格减去回调比例
            pullback := currentPrice * config.OpenLongTrigger.PullbackRatio
            return currentPrice - pullback
        }
    } else { // open_short
        // 开空：等待价格反弹到止损价附近
        if stopLoss > 0 && config.Parameters.UseStopLossAsTrigger {
            buffer := currentPrice * config.Parameters.AdditionalBuffer
            return stopLoss - buffer
        } else {
            pullback := currentPrice * config.OpenShortTrigger.PullbackRatio
            return currentPrice + pullback
        }
    }
}
```

### 3. 突破模式 (Breakout Mode)
```go
func calculateBreakoutTrigger(
    currentPrice float64,
    action string,
    config TriggerPriceStrategy,
) float64 {
    if action == "open_long" {
        // 开多：等待突破当前价格
        threshold := currentPrice * config.OpenLongTrigger.BreakoutRatio
        return currentPrice + threshold
    } else { // open_short
        // 开空：等待跌破当前价格
        threshold := currentPrice * config.OpenShortTrigger.BreakoutRatio
        return currentPrice - threshold
    }
}
```

## 集成到现有系统

### 1. 策略配置扩展

在 `store/strategy_config.go` 中添加：

```go
type StrategyConfig struct {
    // ... 现有字段 ...
    
    // 触发价格策略
    TriggerPriceConfig *TriggerPriceStrategy `json:"trigger_price_config,omitempty"`
}

// GetDefaultTriggerPriceConfig 获取默认配置
func GetDefaultTriggerPriceConfig(style string) *TriggerPriceStrategy {
    switch style {
    case "long_term":
        return &longTermConfig
    case "short_term":
        return &shortTermConfig
    case "swing":
        return &swingConfig
    case "scalp":
        return &scalpConfig
    default:
        return &swingConfig // 默认使用摆动配置
    }
}
```

### 2. 修改待执行订单创建

在 `auto_trader_analysis.go` 中：

```go
// SaveAnalysisAndCreatePendingOrders
func (at *AutoTrader) SaveAnalysisAndCreatePendingOrders(aiDecision *kernel.FullDecision) error {
    // ... 现有代码 ...
    
    // 获取触发价格策略配置
    triggerConfig := at.config.StrategyConfig.TriggerPriceConfig
    if triggerConfig == nil {
        // 如果未配置，使用默认值（保持向后兼容）
        triggerConfig = store.GetDefaultTriggerPriceConfig("swing")
    }
    
    for _, decision := range aiDecision.Decisions {
        // ... 现有代码 ...
        
        // 计算触发价格（使用策略配置）
        triggerPrice := at.calculateTriggerPrice(
            currentPrice,
            decision.Action,
            decision.StopLoss,
            triggerConfig,
        )
        
        // ... 创建订单 ...
    }
    
    return nil
}

// calculateTriggerPrice 根据策略配置计算触发价格
func (at *AutoTrader) calculateTriggerPrice(
    currentPrice float64,
    action string,
    stopLoss float64,
    config *store.TriggerPriceStrategy,
) float64 {
    var triggerPrice float64
    
    switch action {
    case "open_long":
        switch config.OpenLongTrigger.Mode {
        case "current_price":
            triggerPrice = currentPrice
        case "pullback":
            triggerPrice = calculatePullbackTrigger(currentPrice, stopLoss, "open_long", *config)
        case "breakout":
            triggerPrice = calculateBreakoutTrigger(currentPrice, "open_long", *config)
        }
        
    case "open_short":
        switch config.OpenShortTrigger.Mode {
        case "current_price":
            triggerPrice = currentPrice
        case "pullback":
            triggerPrice = calculatePullbackTrigger(currentPrice, stopLoss, "open_short", *config)
        case "breakout":
            triggerPrice = calculateBreakoutTrigger(currentPrice, "open_short", *config)
        }
    }
    
    logger.Infof("🔧 Trigger price calculated: %s %s | Current: %.2f | Trigger: %.2f | Mode: %s",
        action, config.TradingStyle, currentPrice, triggerPrice, 
        getTriggerMode(action, config))
    
    return triggerPrice
}
```

### 3. 前端配置界面

在策略配置页面添加：

```typescript
// Web界面配置结构
interface TriggerPriceConfigUI {
    tradingStyle: 'long_term' | 'short_term' | 'swing' | 'scalp';
    openLong: {
        mode: 'current_price' | 'pullback' | 'breakout';
        pullbackRatio: number; // 0-10%
        breakoutRatio: number; // 0-5%
    };
    openShort: {
        mode: 'current_price' | 'pullback' | 'breakout';
        pullbackRatio: number; // 0-10%
        breakoutRatio: number; // 0-5%
    };
    advanced: {
        useStopLossAsTrigger: boolean;
        additionalBuffer: number; // 0-2%
        waitForConfirmation: boolean;
    };
}
```

## 配置示例

### 配置长线交易员

```go
strategyConfig := &store.StrategyConfig{
    // ... 其他配置 ...
    TriggerPriceConfig: &store.TriggerPriceStrategy{
        TradingStyle: "long_term",
        OpenLongTrigger: struct {
            Mode          string  `json:"mode"`
            PullbackRatio float64 `json:"pullback_ratio"`
            BreakoutRatio float64 `json:"breakout_ratio"`
        }{
            Mode:          "pullback",
            PullbackRatio: 0.05,
            BreakoutRatio: 0.02,
        },
        OpenShortTrigger: struct {
            Mode          string  `json:"mode"`
            PullbackRatio float64 `json:"pullback_ratio"`
            BreakoutRatio float64 `json:"breakout_ratio"`
        }{
            Mode:          "pullback",
            PullbackRatio: 0.05,
            BreakoutRatio: 0.02,
        },
        Parameters: struct {
            UseCurrentPrice       bool    `json:"use_current_price"`
            UseStopLossAsTrigger  bool    `json:"use_stop_loss_as_trigger"`
            AdditionalBuffer      float64 `json:"additional_buffer"`
            BreakoutThreshold     float64 `json:"breakout_threshold"`
            WaitForConfirmation   bool    `json:"wait_for_confirmation"`
        }{
            UseCurrentPrice:       false,
            UseStopLossAsTrigger:  true,
            AdditionalBuffer:      0.01,
            BreakoutThreshold:     0.02,
            WaitForConfirmation:   true,
        },
    },
}
```

## 优势

1. **个性化**：每个交易员可以根据自己的风格配置
2. **灵活性**：支持多种触发模式组合
3. **可扩展**：易于添加新的触发策略
4. **向后兼容**：未配置时使用默认值
5. **透明度**：日志中清晰记录触发逻辑

## 测试建议

1. **回测验证**：使用历史数据测试不同配置的效果
2. **模拟交易**：在模拟环境中验证触发时机
3. **对比分析**：长线 vs 短线配置的触发差异
4. **性能监控**：跟踪触发成功率和执行效果

## 总结

这个方案完美解决了您提出的问题：
- ✅ 将触发价格计算移至策略制定部分
- ✅ 支持不同交易员风格的差异化配置
- ✅ 保持系统架构的清晰性和可维护性
- ✅ 向后兼容，不影响现有功能
