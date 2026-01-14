# AI 交易 - 架构重设计文档

## 📋 概述

将原有的 **立即执行** 工作流（AI 分析 → 立即执行）重新设计为 **延迟执行** 工作流（AI 分析 → 保存 → 等待价格触发 → 自动执行）。

这种设计的优势：
- ✅ **分离关注点**：分析和执行解耦
- ✅ **持久化存储**：分析结果可供审计和重放
- ✅ **自动执行**：无需人工确认，价格触发时自动成交
- ✅ **失败恢复**：即使 AI 失败，已保存的分析仍可手动触发

## 🏗️ 新架构（3 阶段）

### 阶段 1：AI 分析（每 5 分钟）
```
┌─────────────────────────────┐
│  1. 收集账户和行情数据       │
│  2. 调用 AI 进行决策         │
│  3. 💾 保存分析到数据库      │
│  4. 创建待执行订单          │
└─────────────────────────────┘
             ↓
         分析完成
         (入库 ai_analysis 表)
```

**调用点**：`trader/auto_trader.go` → `runCycle()` → `SaveAnalysisAndCreatePendingOrders()`

### 阶段 2：价格监控（每 30 秒）
```
┌─────────────────────────────┐
│  监控线程运行（后台）        │
│  查询所有 PENDING 订单      │
│  获取当前市场价格            │
│  检查是否 ≤ 触发价格        │
└─────────────────────────────┘
             ↓
       触发条件满足？
         /        \
       是          否
       ↓          ↓
    执行      继续监控
```

**调用点**：`trader/auto_trader.go` → `Run()` → `MonitorAndExecutePendingOrders()`（后台协程）

### 阶段 3：自动执行（触发时）
```
┌─────────────────────────────┐
│  1. 标记订单为 TRIGGERED    │
│  2. 执行成交（调用交易所）  │
│  3. 记录到交易历史          │
│  4. 标记订单为 FILLED       │
└─────────────────────────────┘
```

**调用点**：`trader/auto_trader.go` → `executePendingOrder()`

---

## 📊 数据模型

### 1. `AnalysisRecord` - AI 分析记录
```go
type AnalysisRecord struct {
    ID                string        // UUID
    TraderID          string        // 交易者 ID
    Symbol            string        // 交易对 (e.g., "BTCUSDT")
    TargetPrice       float64       // 目标价格
    Confidence        float64       // 置信度 (0-1)
    AnalysisReason    string        // 分析理由
    AnalysisPrompt    string        // AI 输入的 Prompt
    AIResponse        string        // AI 输出的响应
    SupportLevels     []float64     // 支撑位数组
    ResistanceLevel   float64       // 压力位/目标价
    Status            string        // "ACTIVE" / "CLOSED"
    AnalysisTime      time.Time     // 分析时间
    ExpiresAt         time.Time     // 4 小时后自动过期
}

// 数据库表：ai_analysis
// 索引：(trader_id, symbol, analysis_time)
```

### 2. `PendingOrder` - 待执行订单
```go
type PendingOrder struct {
    ID                string        // UUID
    TraderID          string        // 交易者 ID
    AnalysisID        string        // 关联的分析记录 ID
    Symbol            string        // 交易对
    TargetPrice       float64       // 目标价格
    TriggerPrice      float64       // 触发价格 (通常 = TargetPrice * 0.95)
    PositionSize      float64       // 仓位大小 (USDT)
    Leverage          int           // 杠杆倍数
    StopLoss          float64       // 止损价格
    TakeProfit        float64       // 止盈价格
    Confidence        float64       // 置信度 (0-1)
    Status            string        // PENDING / TRIGGERED / FILLED / CANCELLED / EXPIRED
    TriggeredPrice    float64       // 触发时的实际价格
    TriggeredAt       time.Time     // 触发时间
    FilledAt          time.Time     // 成交时间
    OrderID           int64         // 交易所订单 ID
    ExpiresAt         time.Time     // 1 天后自动过期
}

// 数据库表：pending_orders
// 索引：(trader_id, status, symbol)
```

### 3. `TradeHistoryRecord` - 交易历史
```go
type TradeHistoryRecord struct {
    ID                string        // UUID
    TraderID          string        // 交易者 ID
    AnalysisID        string        // 源分析记录 ID
    PendingOrderID    string        // 源待执行订单 ID
    Symbol            string        // 交易对
    EntryPrice        float64       // 成交价格
    Quantity          float64       // 成交数量
    Leverage          int           // 杠杆
    EntryTime         time.Time     // 成交时间
}

// 数据库表：trade_history
// 索引：(trader_id, symbol, entry_time)
```

---

## 🔄 工作流程

### 流程图

```
┌─────────────────────────────────────────────┐
│  Main Loop: runCycle() - 每 5 分钟执行       │
├─────────────────────────────────────────────┤
│                                             │
│  1️⃣  buildTradingContext()                 │
│     • 获取账户余额、持仓、候选币列表       │
│                                             │
│  2️⃣  kernel.GetFullDecisionWithStrategy()  │
│     • 调用 AI（Claude/Deepseek）           │
│     • 获取决策列表                        │
│                                             │
│  3️⃣  SaveAnalysisAndCreatePendingOrders()  │
│     • 💾 遍历每个决策                      │
│     • 保存为 AnalysisRecord                │
│     • 创建 PendingOrder (只限 open_* 动作) │
│     • 计算触发价格 = 目标价 * 95%         │
│                                             │
│  4️⃣  saveDecision() - 记录决策日志         │
│                                             │
└─────────────────────────────────────────────┘
                    ↓↓↓
┌─────────────────────────────────────────────┐
│  Background: MonitorAndExecutePendingOrders()│
│  每 30 秒执行一次（后台协程）             │
├─────────────────────────────────────────────┤
│                                             │
│  FOR 每个 PENDING 订单:                    │
│    • 获取当前市场价格                     │
│    • 检查 currentPrice ≤ triggerPrice?   │
│    • 是 → 执行 executePendingOrder()     │
│         • 调用交易所 API 成交             │
│         • UpdatePendingOrderStatus()      │
│         • SaveTradeHistory()              │
│    • 否 → 继续监控                        │
│                                             │
└─────────────────────────────────────────────┘
```

---

## 📁 新增文件

### 1. `store/analysis.go` (400+ 行)
**功能**：定义数据模型和接口

**主要内容**：
- `AnalysisRecord` struct - 分析记录
- `PendingOrder` struct - 待执行订单
- `TradeHistoryRecord` struct - 交易历史
- `SupportLevels` custom type - 支撑位数组（JSON 序列化）
- `AnalysisStore` interface - 15 个方法

**主要方法**：
```go
SaveAnalysis(analysis *AnalysisRecord) error
GetActiveAnalyses(traderID string) ([]*AnalysisRecord, error)
SavePendingOrder(order *PendingOrder) error
GetPendingOrdersByStatus(traderID, status string) ([]*PendingOrder, error)
UpdatePendingOrderStatus(id, status string, triggeredPrice float64, triggeredAt time.Time) error
UpdatePendingOrderFilled(id string, filledAt time.Time, orderID int64) error
SaveTradeHistory(trade *TradeHistoryRecord) error
DeleteExpiredAnalyses(traderID string) error
DeleteExpiredPendingOrders(traderID string) error
...
```

### 2. `store/analysis_impl.go` (380+ 行)
**功能**：使用 GORM ORM 实现持久化层

**关键特性**：
- AutoMigrate 3 张表（ai_analysis, pending_orders, trade_history）
- 自动过期清理：
  - 分析记录：4 小时后过期
  - 待执行订单：1 天后过期
- 数据库索引优化查询性能
- UUID 生成器

### 3. `trader/auto_trader_analysis.go` (220+ 行)
**功能**：实现新工作流的核心逻辑

**主要方法**：

#### `SaveAnalysisAndCreatePendingOrders(aiDecision)`
- 遍历 AI 决策
- 为每个决策创建 `AnalysisRecord`
- 为 open_* 动作创建 `PendingOrder`
- 计算触发价格 = TakeProfit * 0.95

#### `MonitorAndExecutePendingOrders()`
- 查询所有 PENDING 订单
- 获取当前市场价格（通过 market.Get()）
- 检查价格 ≤ 触发价格
- 触发时调用 `executePendingOrder()`

#### `executePendingOrder(order, currentPrice)`
- 检查账户余额
- 构造 Decision 对象
- 调用 `executeDecisionWithRecord()` 执行
- 记录交易历史
- 更新订单状态为 FILLED

---

## 🔧 文件修改

### `store/store.go`
```go
// 新增字段
type Store struct {
    ...
    analysis AnalysisStore  // 分析存储
}

// 新增方法
func (s *Store) Analysis() AnalysisStore {
    if s.analysis == nil {
        s.analysis = NewAnalysisImpl(s.gdb)
    }
    return s.analysis
}

// 修改 initTables()
func (s *Store) initTables() error {
    ...
    // Initialize analysis tables
    analysisStore := NewAnalysisImpl(s.gdb)
    if analysisImpl, ok := analysisStore.(*AnalysisImpl); ok {
        if err := analysisImpl.InitSchema(); err != nil {
            return fmt.Errorf("failed to initialize analysis tables: %w", err)
        }
    }
    ...
}
```

### `trader/auto_trader.go`

#### 修改 `Run()` 方法
```go
// 新增后台监控协程（每 30 秒）
at.monitorWg.Add(1)
go func() {
    defer at.monitorWg.Done()
    monitorTicker := time.NewTicker(30 * time.Second)
    defer monitorTicker.Stop()
    
    for {
        ...
        case <-monitorTicker.C:
            if err := at.MonitorAndExecutePendingOrders(); err != nil {
                logger.Warnf("⚠️ Error monitoring pending orders: %v", err)
            }
        ...
    }
}()
```

#### 修改 `runCycle()` 方法
```go
// 原来的执行逻辑被替换为：
// 8. NEW WORKFLOW: Save AI analysis and create pending orders
logger.Info("🔄 NEW WORKFLOW: Saving AI analysis → Waiting for price triggers → Auto-executing")

if err := at.SaveAnalysisAndCreatePendingOrders(aiDecision); err != nil {
    logger.Warnf("⚠️ Failed to save analysis or create pending orders: %v", err)
} else {
    logger.Infof("✅ AI analysis saved and pending orders created")
    // 创建决策日志记录
    for _, d := range aiDecision.Decisions {
        actionRecord := store.DecisionAction{...}
        record.Decisions = append(record.Decisions, actionRecord)
    }
}
```

---

## 🎯 执行流程示例

### 场景：用户启动交易机器人

```
时刻 00:00 - runCycle() 第 1 次执行
├─ AI 分析 BTC/ETH/SOL
├─ 决策：
│  ├─ BTCUSDT: open_long @ 45000 (targetPrice)
│  ├─ ETHUSDT: open_long @ 2800 (targetPrice)
│  └─ SOL: wait
├─ 保存分析到 ai_analysis 表
├─ 创建待执行订单：
│  ├─ BTCUSDT: triggerPrice = 42750 (45000 * 95%)
│  └─ ETHUSDT: triggerPrice = 2660 (2800 * 95%)
└─ 订单状态：PENDING

时刻 00:00 ~ 00:05 - MonitorAndExecutePendingOrders() 每 30 秒检查
├─ 00:00:30 - 检查：BTC 当前 44000 > 42750 ❌ 未触发
├─ 00:01:00 - 检查：BTC 当前 43000 > 42750 ❌ 未触发
├─ 00:01:30 - 检查：BTC 当前 42500 ≤ 42750 ✅ 触发！
│  ├─ UpdatePendingOrderStatus(TRIGGERED, 42500)
│  ├─ executeDecisionWithRecord()
│  │  └─ 调用交易所 API 成交
│  ├─ SaveTradeHistory()
│  ├─ UpdatePendingOrderFilled()
│  └─ 订单状态：FILLED
├─ 00:02:00 - 检查：ETH 当前 2750 > 2660 ❌ 未触发
├─ 00:02:30 - 检查：ETH 当前 2655 ≤ 2660 ✅ 触发！
│  └─ (同上执行流程)
└─ ...

时刻 05:00 - runCycle() 第 2 次执行
├─ AI 重新分析市场
├─ 之前的待执行订单仍在 pending_orders 表中
├─ 新分析结果保存为新记录
└─ ...

[后台清理]
每天凌晨（或 cron）：
├─ DeleteExpiredAnalyses() - 删除 4 小时前的分析
├─ DeleteExpiredPendingOrders() - 删除 1 天前的待执行订单
└─ ...
```

---

## 🔐 数据一致性

### 事务流程

1. **保存分析 → 创建订单**
   ```
   BEGIN TRANSACTION
     INSERT INTO ai_analysis VALUES (...)  // analysis.ID = "abc123"
     INSERT INTO pending_orders 
        VALUES (..., analysis_id="abc123")  // 外键关联
   COMMIT
   ```

2. **价格触发 → 更新状态 → 记录历史**
   ```
   BEGIN TRANSACTION
     UPDATE pending_orders SET status='TRIGGERED' WHERE id='order123'
     INSERT INTO trade_history VALUES (...)
     UPDATE pending_orders SET status='FILLED' WHERE id='order123'
   COMMIT
   ```

### 故障恢复

- **AI 决策失败**：分析记录丢失，但不影响已创建的待执行订单
- **执行失败**：订单保持 PENDING，下一个监控周期重试
- **网络断线**：订单保持在数据库，重启后继续监控

---

## 📈 性能影响

| 操作 | 周期 | 影响 |
|------|------|------|
| AI 分析 | 5 分钟 | 需要调用 AI API (延迟) |
| 价格监控 | 30 秒 | 轻量级数据库查询 + 市场数据获取 |
| 订单执行 | 随机 | 取决于市场波动 |
| 数据清理 | 每天 | 后台任务，不阻塞主流程 |

---

## 🚀 优势

1. **分离关注点**
   - 分析逻辑独立于执行逻辑
   - 易于测试、调试、审计

2. **故障恢复能力**
   - 即使 AI 或 API 失败，已保存的分析和订单仍然有效
   - 支持手动触发或重新分析

3. **自动化程度高**
   - 价格自动触发，无需人工确认
   - 后台协程自动监控，用户无感知

4. **可审计性**
   - 完整的分析记录、执行记录、交易历史
   - 支持回测和性能分析

5. **灵活性**
   - 可以手动创建待执行订单（未实现，但框架支持）
   - 可以调整触发价格、仓位等参数

---

## ⚠️ 限制和改进空间

1. **触发价格计算**
   - 当前固定为 TakeProfit * 95%
   - 可改进：根据波动率、支撑位动态调整

2. **执行方向推断**
   - 当前 `executePendingOrder()` 假设所有都是 open_long
   - 需改进：从 AI 决策中提取 action 字段（open_long / open_short）

3. **订单类型**
   - 当前只支持开仓（open_long/open_short）
   - 可扩展：平仓、加仓等操作

4. **风险管理**
   - 没有动态调整止损止盈的逻辑
   - 没有尾随止损等高级功能

---

## 📝 后续改进清单

- [ ] 实现尾随止损（Trailing Stop Loss）
- [ ] 支持手动创建待执行订单
- [ ] 实现订单的部分成交处理
- [ ] 添加价格预测模块优化触发价格
- [ ] 实现 WebSocket 实时价格监控（替代轮询）
- [ ] 支持多种订单类型（限价单、止损单等）
- [ ] 添加机器学习模型优化触发时机
- [ ] 实现账户风险实时监控和动态调整
