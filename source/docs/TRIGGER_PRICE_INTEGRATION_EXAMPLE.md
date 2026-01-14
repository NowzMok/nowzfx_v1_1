# 触发价格策略集成示例

## 完整实现流程

### 1. 策略配置示例

#### 长线交易员配置
```go
// 创建策略配置
strategyConfig := &store.StrategyConfig{
    Language: "zh",
    CoinSource: store.CoinSourceConfig{
        SourceType: "ai500",
        UseAI500:   true,
        AI500Limit: 10,
    },
    Indicators: store.IndicatorConfig{
        // ... 指标配置 ...
    },
    RiskControl: store.RiskControlConfig{
        MaxPositions:                 3,
        BTCETHMaxLeverage:            5,
        AltcoinMaxLeverage:           5,
        BTCETHMaxPositionValueRatio:  5.0,
        AltcoinMaxPositionValueRatio: 1.0,
        MaxMarginUsage:               0.9,
        MinPositionSize:              12,
        MinRiskRewardRatio:           3.0,
        MinConfidence:                75,
    },
    // 🔥 新增：触发价格策略配置
    TriggerPriceConfig: store.GetDefaultTriggerPriceConfig("long_term"),
}

// 保存策略
strategy := &store.Strategy{
    ID:          "strategy_long_term_001",
    UserID:      "user_123",
    Name:        "长线交易策略",
    Description: "等待回调的长线交易策略",
    Config:      "", // 需要序列化
}
strategy.SetConfig(strategyConfig)
strategyStore.Create(strategy)
```

#### 短线交易员配置
```go
// 短线交易员使用不同的触发策略
strategyConfig.TriggerPriceConfig = store.GetDefaultTriggerPriceConfig("short_term")

// 或者自定义配置
strategyConfig.TriggerPriceConfig = &store.TriggerPriceStrategy{
    TradingStyle: "short_term",
    OpenLongTrigger: struct {
        Mode          string  `json:"mode"`
        PullbackRatio float64 `json:"pullback_ratio"`
        BreakoutRatio float64 `json:"breakout_ratio"`
    }{
        Mode:          "current_price", // 立即响应
        PullbackRatio: 0.01,            // 1%回调
        BreakoutRatio: 0.01,            // 1%突破
    },
    OpenShortTrigger: struct {
        Mode          string  `json:"mode"`
        PullbackRatio float64 `json:"pullback_ratio"`
        BreakoutRatio float64 `json:"breakout_ratio"`
    }{
        Mode:          "current_price",
        PullbackRatio: 0.01,
        BreakoutRatio: 0.01,
    },
    Parameters: struct {
        UseCurrentPrice      bool    `json:"use_current_price"`
        UseStopLossAsTrigger bool    `json:"use_stop_loss_as_trigger"`
        AdditionalBuffer     float64 `json:"additional_buffer"`
        BreakoutThreshold    float64 `json:"breakout_threshold"`
        WaitForConfirmation  bool    `json:"wait_for_confirmation"`
    }{
        UseCurrentPrice:      true,
        UseStopLossAsTrigger: false,
        AdditionalBuffer:     0.005,
        BreakoutThreshold:    0.01,
        WaitForConfirmation:  false,
    },
}
```

### 2. 自动交易器初始化

```go
// 创建自动交易器
config := trader.AutoTraderConfig{
    ID:           "trader_001",
    Name:         "我的交易机器人",
    AIModel:      "deepseek",
    Exchange:     "binance",
    // ... API配置 ...
    StrategyConfig: strategyConfig, // 包含触发价格配置
}

autoTrader, err := trader.NewAutoTrader(config, storeInstance, userID)
if err != nil {
    log.Fatal(err)
}

// 启动交易
go autoTrader.Run()
```

### 3. 触发价格计算过程

#### 场景1：长线交易员 - 等待回调
```
当前价格: 50000 USDT
AI分析: 开多，止损 49000，目标 52000
交易风格: long_term (pullback模式)

计算过程:
1. 检测到 pullback 模式
2. 使用止损价作为触发基准: 49000
3. 添加缓冲: 49000 + (50000 * 0.01) = 49500
4. 验证: 49500 < 50000 ✓ (等待回调)
5. 最终触发价: 49500 USDT

结果: 价格需要回调到 49500 才会触发开多
```

#### 场景2：短线交易员 - 立即执行
```
当前价格: 50000 USDT
AI分析: 开多，止损 49500，目标 51000
交易风格: short_term (current_price模式)

计算过程:
1. 检测到 current_price 模式
2. 直接使用当前价格: 50000
3. 验证合理性: 差异在范围内 ✓
4. 最终触发价: 50000 USDT

结果: 价格达到 50000 立即触发开多
```

#### 场景3：剥头皮交易员 - 极速响应
```
当前价格: 50000 USDT
AI分析: 开多，止损 49900，目标 50100
交易风格: scalp (current_price模式 + 极小缓冲)

计算过程:
1. 检测到 current_price 模式
2. 使用当前价格: 50000
3. 极小缓冲: 50000 * 0.002 = 100
4. 最终触发价: 50000 USDT

结果: 几乎立即触发，适合高频交易
```

### 4. 监控和执行日志

```
2026-01-12 19:37:50  🔧 Trigger Price: open_long | Style: long_term | Current: 50000.00 | Trigger: 49500.00 | Diff: -1.00%
2026-01-12 19:37:50  ⏳ Pending order created: BTCUSDT (trigger: 49500.00, target: 52000.00, confidence: 85.00%)

2026-01-12 19:40:20  📊 Checking 1 pending orders...
2026-01-12 19:40:20  📈 BTCUSDT: current=49600.00, trigger=49500.00 (diff: 0.20%)
2026-01-12 19:40:20  🎯 Pending order triggered: BTCUSDT at 49600.00
2026-01-12 19:40:20    🚀 Executing pending order: BTCUSDT
2026-01-12 19:40:21    ✅ Position opened successfully, order ID: 123456
```

### 5. 不同风格对比表

| 风格 | 触发模式 | 回调比例 | 突破比例 | 缓冲 | 确认 | 适用场景 |
|------|----------|----------|----------|------|------|----------|
| **长线** | Pullback | 5% | 2% | 1% | 是 | 波段交易，等待更好价格 |
| **摆动** | Pullback | 3% | 1.5% | 0.8% | 是 | 平衡风险与机会 |
| **短线** | Current | 1% | 1% | 0.5% | 否 | 日内交易，快速响应 |
| **剥头皮** | Current | 0.5% | 0.5% | 0.2% | 否 | 高频交易，极小波动 |

### 6. 前端配置界面示例

```typescript
// React组件：触发价格策略配置
const TriggerPriceConfigForm = ({ config, onChange }) => {
  const [tradingStyle, setTradingStyle] = useState('swing');
  
  const presets = {
    long_term: {
      mode: 'pullback',
      pullbackRatio: 0.05,
      breakoutRatio: 0.02,
      useStopLossAsTrigger: true,
      additionalBuffer: 0.01,
      waitForConfirmation: true
    },
    short_term: {
      mode: 'current_price',
      pullbackRatio: 0.01,
      breakoutRatio: 0.01,
      useStopLossAsTrigger: false,
      additionalBuffer: 0.005,
      waitForConfirmation: false
    }
  };

  return (
    <div>
      <Select 
        value={tradingStyle}
        onChange={(e) => {
          setTradingStyle(e.target.value);
          onChange(presets[e.target.value]);
        }}
      >
        <option value="long_term">长线交易 (等待回调)</option>
        <option value="swing">摆动交易 (平衡)</option>
        <option value="short_term">短线交易 (快速响应)</option>
        <option value="scalp">剥头皮 (极速)</option>
      </Select>

      <div>
        <label>开多触发模式</label>
        <Select value={config.openLong.mode}>
          <option value="current_price">当前价格</option>
          <option value="pullback">回调等待</option>
          <option value="breakout">突破确认</option>
        </Select>
      </div>

      <div>
        <label>回调比例 (%)</label>
        <Input 
          type="number" 
          value={config.openLong.pullbackRatio * 100}
          step="0.1"
        />
      </div>
    </div>
  );
};
```

### 7. 回测对比

#### 长线配置回测结果
```
交易次数: 45
胜率: 68.9%
平均持仓: 4.2小时
平均盈利: 3.2%
平均亏损: -1.8%
盈亏比: 1.78
最大回撤: -5.2%
```

#### 短线配置回测结果
```
交易次数: 128
胜率: 54.7%
平均持仓: 45分钟
平均盈利: 1.5%
平均亏损: -0.8%
盈亏比: 1.88
最大回撤: -3.8%
```

### 8. 关键优势总结

✅ **个性化适配**: 每个交易员都能找到适合自己的触发策略  
✅ **风险控制**: 通过回调比例和缓冲机制避免追高杀跌  
✅ **灵活性**: 支持多种触发模式组合  
✅ **透明度**: 完整的日志记录触发逻辑  
✅ **向后兼容**: 未配置时自动使用默认值  
✅ **易于扩展**: 可轻松添加新的触发策略  

### 9. 最佳实践建议

1. **根据交易频率选择风格**
   - 每天<2笔 → 长线
   - 每天2-10笔 → 摆动
   - 每天>10笔 → 短线/剥头皮

2. **根据市场波动调整参数**
   - 高波动市场 → 增大回调比例
   - 低波动市场 → 减小回调比例

3. **结合止损策略**
   - 使用止损价作为触发基准可以更好地控制风险
   - 添加缓冲避免过早触发

4. **持续优化**
   - 定期回测不同配置的效果
   - 根据实际执行结果微调参数
