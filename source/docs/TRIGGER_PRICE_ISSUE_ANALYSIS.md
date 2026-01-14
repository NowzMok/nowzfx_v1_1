# 触发价格配置丢失问题 - 系统性排查报告

## 问题描述

用户反馈：选择的是scalp风格，但触发价格18.8575被判断为Long Term风格，说明配置没有正确应用。

## 配置传递链路分析

### 1. 前端 → 后端 API
**文件**: `nofx/web/src/components/strategy/TriggerPriceEditor.tsx`
**问题**: ✅ 已修复状态管理bug

```typescript
// 关键代码：预设选择时的配置更新
const handlePresetChange = (presetName: string) => {
  setPreset(presetName)
  const presetConfig = presets[presetName as keyof typeof presets]
  onChange({ ...presetConfig })  // ✅ 正确传递完整配置
}
```

### 2. 后端 API → 数据库
**文件**: `nofx/api/strategy.go` - `handleUpdateStrategy`
**调试日志**: ✅ 已添加

```go
// 关键代码：接收并验证配置
logger.Infof("🔧 [API] Strategy update request received")
if req.Config.TriggerPriceConfig != nil {
    logger.Infof("  TriggerPriceConfig: mode=%s, style=%s, pullback=%.3f, breakout=%.3f, buffer=%.3f",
        req.Config.TriggerPriceConfig.Mode,
        req.Config.TriggerPriceConfig.Style,
        req.Config.TriggerPriceConfig.PullbackRatio,
        req.Config.TriggerPriceConfig.BreakoutRatio,
        req.Config.TriggerPriceConfig.ExtraBuffer)
} else {
    logger.Warnf("⚠️ TriggerPriceConfig is nil in request!")
}
```

### 3. 数据库 → TraderManager
**文件**: `nofx/manager/trader_manager.go` - `addTraderFromStore`
**调试日志**: ✅ 已添加

```go
// 关键代码：从数据库加载策略配置
strategy, err := st.Strategy().Get(traderCfg.UserID, traderCfg.StrategyID)
if err != nil {
    return fmt.Errorf("failed to load strategy %s for trader %s: %w", traderCfg.StrategyID, traderCfg.Name, err)
}
strategyConfig, err = strategy.ParseConfig()
if err != nil {
    return fmt.Errorf("failed to parse strategy config for trader %s: %w", traderCfg.Name, err)
}

// 🔍 调试：检查TriggerPriceConfig
if strategyConfig.TriggerPriceConfig != nil {
    logger.Infof("🔧 [TRADER_MANAGER] TriggerPriceConfig loaded: mode=%s, style=%s, pullback=%.3f, breakout=%.3f, buffer=%.3f",
        strategyConfig.TriggerPriceConfig.Mode,
        strategyConfig.TriggerPriceConfig.Style,
        strategyConfig.TriggerPriceConfig.PullbackRatio,
        strategyConfig.TriggerPriceConfig.BreakoutRatio,
        strategyConfig.TriggerPriceConfig.ExtraBuffer)
} else {
    logger.Warnf("⚠️ [TRADER_MANAGER] TriggerPriceConfig is nil for strategy %s", strategy.Name)
}
```

### 4. TraderManager → AutoTrader
**文件**: `nofx/trader/auto_trader_analysis.go` - `SaveAnalysisAndCreatePendingOrders`
**调试日志**: ✅ 已添加

```go
// 关键代码：获取并验证配置
triggerConfig := at.config.StrategyConfig.TriggerPriceConfig
if triggerConfig == nil {
    style := "swing"
    if at.config.StrategyConfig.TriggerPriceConfig != nil {
        style = at.config.StrategyConfig.TriggerPriceConfig.Style
    }
    triggerConfig = store.GetDefaultTriggerPriceConfig(style)
    logger.Warnf("⚠️ TriggerPriceConfig is nil, using default style '%s'", style)
}

// 🚨 调试：打印配置信息
logger.Infof("🔧 [TRIGGER_PRICE_DEBUG] Strategy Config Check:")
logger.Infof("  TriggerPriceConfig is nil: %v", triggerConfig == nil)
if triggerConfig != nil {
    logger.Infof("  Config Mode: %s", triggerConfig.Mode)
    logger.Infof("  Config Style: %s", triggerConfig.Style)
    logger.Infof("  Pullback Ratio: %.4f", triggerConfig.PullbackRatio)
    logger.Infof("  Breakout Ratio: %.4f", triggerConfig.BreakoutRatio)
    logger.Infof("  Extra Buffer: %.4f", triggerConfig.ExtraBuffer)
} else {
    logger.Errorf("❌ TriggerPriceConfig is nil! This indicates configuration was not properly saved or loaded")
}
```

## 问题根源分析

### 可能的断点位置

1. **前端配置未正确发送**
   - TriggerPriceEditor的onChange回调未被正确调用
   - 配置对象未被包含在完整的strategy config中

2. **API层配置丢失**
   - JSON序列化/反序列化问题
   - 字段名称不匹配（前端vs后端）

3. **数据库存储问题**
   - 配置未正确保存到数据库
   - 数据库字段类型不匹配

4. **配置加载问题**
   - TraderManager未正确解析JSON
   - TriggerPriceConfig字段为nil

## 验证步骤

### 步骤1：启动系统并观察日志
```bash
cd /Users/nowzmok/Desktop/圣灵/nonowz/nofx
go run main.go 2>&1 | grep -E "(TRIGGER_PRICE|TriggerPriceConfig|🔧|⚠️)"
```

### 步骤2：在前端创建scalp策略
1. 打开策略编辑器
2. 选择"剥头皮"预设
3. 保存策略
4. 观察后端日志输出

### 步骤3：检查日志输出
应该看到类似这样的日志：

```
🔧 [API] Strategy update request received
  TriggerPriceConfig: mode=current_price, style=scalp, pullback=0.005, breakout=0.003, buffer=0.001

✅ Strategy updated successfully in database

🔧 [TRADER_MANAGER] TriggerPriceConfig loaded: mode=current_price, style=scalp, pullback=0.005, breakout=0.003, buffer=0.001

🔧 [TRIGGER_PRICE_DEBUG] Strategy Config Check:
  Config Mode: current_price
  Config Style: scalp
  Pullback Ratio: 0.0050
  Breakout Ratio: 0.0030
  Extra Buffer: 0.0010

🔧 [TRIGGER_PRICE_DEBUG] Calculation Result:
  Trigger Price: 99.90
  Difference: 0.10
  Percentage: 0.10%
```

### 步骤4：验证数据库
```bash
sqlite3 nofx-data.db "SELECT config FROM strategies WHERE name = '你的策略名';"
```

检查返回的JSON中是否包含：
```json
"trigger_price_config": {
  "mode": "current_price",
  "style": "scalp",
  "pullback_ratio": 0.005,
  "breakout_ratio": 0.003,
  "extra_buffer": 0.001
}
```

## 预期结果

如果配置正确传递，不同风格的触发价格应该是：

| 风格 | 模式 | 当前价100时的触发价 | 差别 |
|------|------|-------------------|------|
| Scalp | current_price | 99.90 | 0.1% |
| Swing | pullback | 97.50 | 2.5% |
| Long Term | pullback | 94.00 | 6% |

## 如果问题仍然存在

### 检查前端配置发送
在浏览器开发者工具中检查Network请求，确保请求体包含：
```json
{
  "config": {
    "trigger_price_config": {
      "mode": "current_price",
      "style": "scalp",
      "pullback_ratio": 0.005,
      "breakout_ratio": 0.003,
      "extra_buffer": 0.001
    }
  }
}
```

### 检查数据库存储
```bash
# 查看所有策略的配置
sqlite3 nofx-data.db "SELECT name, config FROM strategies;"

# 查看是否有trigger_price_config字段
sqlite3 nofx-data.db ".schema strategies"
```

### 检查JSON解析
在`store/strategy.go`的`ParseConfig()`方法中添加调试：
```go
func (s *Strategy) ParseConfig() (*StrategyConfig, error) {
    var config StrategyConfig
    if err := json.Unmarshal([]byte(s.Config), &config); err != nil {
        return nil, fmt.Errorf("failed to parse strategy configuration: %w", err)
    }
    // 调试：打印解析结果
    if config.TriggerPriceConfig != nil {
        logger.Infof("✅ ParseConfig: TriggerPriceConfig loaded successfully")
    } else {
        logger.Warnf("❌ ParseConfig: TriggerPriceConfig is nil after parsing")
    }
    return &config, nil
}
```

## 总结

我们已经：
1. ✅ 修复了前端状态管理bug
2. ✅ 添加了全链路调试日志
3. ✅ 验证了配置结构定义正确
4. ✅ 验证了默认配置存在

现在需要通过实际运行系统来验证配置是否正确传递。如果日志显示配置在某个环节丢失，我们可以精确定位问题并修复。
