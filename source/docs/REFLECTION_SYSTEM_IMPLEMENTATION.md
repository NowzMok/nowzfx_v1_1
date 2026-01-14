# 反思系统实现指南

## 概述

完整的周期性反思系统已实现，包括：

1. **定时调度器** (`backtest/reflection_scheduler.go`) - 每周日 22:00 自动触发反思
2. **API 端点** (`api/reflection_handlers.go`) - 提供完整的 REST API 接口
3. **数据存储** - ReflectionRecord, SystemAdjustment, AILearningMemory
4. **AI 集成** - 通过 MCP 客户端调用 AI 进行分析
5. **建议处理** - 自动分离为交易系统和 AI 学习建议

## 系统架构

```
┌─────────────────────────────────────────────────────────────┐
│ 周期性反思工作流                                           │
└─────────────────────────────────────────────────────────────┘
        │
        ▼
┌───────────────────────┐
│ ReflectionScheduler   │ ◄─── 每周日 22:00 触发
│ (backtest package)    │      或手动触发
└───────────────────────┘
        │
        ▼
┌───────────────────────┐
│ ReflectionEngine      │ ◄─── 获取交易历史
│ .AnalyzePeriod()      │      计算统计指标
└───────────────────────┘      调用 AI 分析
        │
        ▼
    ┌──────────────────────────────┐
    │ AI 反思 (MCP 客户端)          │
    │ 返回 JSON 格式分析            │
    └──────────────────────────────┘
        │
        ▼
┌──────────────────────────────────┐
│ 建议分离                         │
│ .separateAdvice()                │
│ ├─ 交易系统建议                  │
│ └─ AI 学习建议                   │
└──────────────────────────────────┘
        │
        ├──────────────────┬──────────────────┐
        ▼                  ▼                  ▼
    ┌────────────┐  ┌─────────────┐  ┌─────────────┐
    │ 数据库保存  │  │ 参数调整    │  │ 学习内存    │
    │ Reflection  │  │ Adjustment  │  │ LearningMem │
    └────────────┘  └─────────────┘  └─────────────┘
        │                  │                  │
        └──────────────────┼──────────────────┘
                           ▼
                   ┌───────────────────────┐
                   │ API 端点 + 前端仪表板 │
                   │ 用户审查和批准        │
                   └───────────────────────┘
```

## 部署步骤

### 1. 注册反思调度器到应用启动流程

在 `main.go` 或应用初始化代码中：

```go
// 初始化反思引擎
reflectionEngine := backtest.NewReflectionEngine(aiClient, store)

// 初始化调度器
reflectionScheduler := backtest.NewReflectionScheduler(reflectionEngine, store)

// 启动调度器
if err := reflectionScheduler.Start(); err != nil {
    logger.Errorf("Failed to start reflection scheduler: %v", err)
}

// 注册交易员进行反思
reflectionScheduler.RegisterTrader(traderID)

// 在应用关闭时
defer reflectionScheduler.Stop()
```

### 2. 注册 API 路由

在 Gin 路由配置中：

```go
import "nofx/api"

// 初始化反思处理器
reflectionHandlers := api.NewReflectionHandlers(reflectionScheduler, store)

// 注册路由
reflectionHandlers.RegisterReflectionRoutes(router)
```

### 3. 配置定时计划（可选 - 使用更高级的调度器）

如果要支持更复杂的调度，可以使用 `robfig/cron` 库：

```go
import "github.com/robfig/cron/v3"

c := cron.New()

// 每周日 22:00 运行反思
c.AddFunc("0 22 * * 0", func() {
    reflectionScheduler.runAllReflections()
})

c.Start()
```

## API 端点参考

### 反思管理

#### 获取最近反思列表
```
GET /api/reflection/{traderID}/recent?limit=10
Response: { data: [ReflectionRecord], count: number, trader: string }
```

#### 获取反思详情
```
GET /api/reflection/{reflectionID}
Response: { data: ReflectionRecord }
```

#### 手动触发反思
```
POST /api/reflection/{traderID}/analyze
Response: { message: "Reflection triggered successfully", trader: string }
```

#### 获取反思统计
```
GET /api/reflection/{traderID}/stats?days=30
Response: { 
  data: {
    total_reflections: number,
    total_trades: number,
    avg_success_rate: float,
    total_pnl: float,
    best_day: object,
    worst_day: object
  },
  trader: string,
  days: number
}
```

### 参数调整

#### 获取待审批调整
```
GET /api/adjustment/{traderID}/pending
Response: { data: [SystemAdjustment], count: number, trader: string, status: "PENDING" }
```

#### 批准调整
```
POST /api/adjustment/{adjustmentID}/apply
Response: { message: "Adjustment applied successfully", id: string, status: "APPLIED" }
```

#### 拒绝调整
```
POST /api/adjustment/{adjustmentID}/reject
Response: { message: "Adjustment rejected successfully", id: string, status: "REJECTED" }
```

#### 撤销调整
```
POST /api/adjustment/{adjustmentID}/revert
Response: { message: "Adjustment reverted successfully", id: string, status: "REVERTED" }
```

#### 获取调整历史
```
GET /api/adjustment/{traderID}/history?limit=50
Response: { data: [SystemAdjustment], count: number, trader: string }
```

### 学习内存

#### 获取学习内存
```
GET /api/memory/{traderID}?limit=50
Response: { data: [AILearningMemory], count: number, trader: string }
```

#### 归档学习内存
```
DELETE /api/memory/{memoryID}
Response: { message: "Memory archived successfully", id: string }
```

## 数据模型

### ReflectionRecord - 反思记录
```go
{
  "id": "uuid",
  "trader_id": "trader_id",
  "reflection_time": "2024-01-15T22:00:00Z",
  "period_start_time": "2024-01-08T00:00:00Z",
  "period_end_time": "2024-01-15T00:00:00Z",
  "total_trades": 25,
  "successful_trades": 18,
  "failed_trades": 7,
  "success_rate": 0.72,
  "average_pnl": 45.6,
  "total_pnl": 1140.0,
  "sharpe_ratio": 1.45,
  "max_drawdown": 0.125,
  "confidence_accuracy": {
    "50%": 0.55,
    "75%": 0.75,
    "90%": 0.67
  },
  "symbol_performance": {
    "BTCUSDT": { "total_pnl": 600.0, "count": 10, "avg_pnl": 60.0 },
    "ETHUSDT": { "total_pnl": 540.0, "count": 15, "avg_pnl": 36.0 }
  },
  "ai_reflection": "...",  // AI 生成的分析文本
  "trade_system_advice": [...],  // 交易系统建议（JSON）
  "ai_learning_advice": [...]    // AI 学习建议（JSON）
}
```

### SystemAdjustment - 参数调整
```go
{
  "id": "uuid",
  "trader_id": "trader_id",
  "reflection_id": "reflection_id",
  "adjustment_time": "2024-01-15T22:00:00Z",
  "confidence_level": 0.70,
  "btc_eth_leverage": 5,
  "altcoin_leverage": 3,
  "max_position_size": 5000.0,
  "max_daily_loss": 200.0,
  "adjustment_reason": "识别到高信心度准确率低于预期",
  "status": "PENDING",  // PENDING, APPLIED, REJECTED, REVERTED
  "applied_at": null,
  "created_at": "2024-01-15T22:00:00Z"
}
```

### AILearningMemory - AI 学习记忆
```go
{
  "id": "uuid",
  "trader_id": "trader_id",
  "reflection_id": "reflection_id",
  "created_at": "2024-01-15T22:00:00Z",
  "expires_at": "2024-02-15T22:00:00Z",  // 默认 30 天过期
  "memory_type": "lesson",  // bias, pattern, lesson, warning
  "symbol": "BTCUSDT",
  "content": "BTC 在大涨时期准确性更高，应该在波动性高时增加信心度阈值",
  "confidence": 0.85,
  "usage_count": 3,
  "last_used_at": "2024-01-14T15:30:00Z",
  "prompt_injection": "根据历史经验，BTC 交易对在大涨时期准确性更高，可以提高信心度阈值到 80%+",
  "updated_at": "2024-01-15T22:00:00Z"
}
```

## 配置示例

### 应用启动配置

```yaml
# config.yaml
reflection:
  enabled: true
  analysis_days: 7        # 分析周期（天数）
  schedule: "0 22 * * 0"  # Cron 表达式（每周日 22:00）
  max_concurrent: 3       # 最多并发运行数量
  notification:
    enabled: true
    email: true
    webhook: true
```

### 环境变量

```bash
REFLECTION_ENABLED=true
REFLECTION_ANALYSIS_DAYS=7
REFLECTION_SCHEDULE="0 22 * * 0"
REFLECTION_NOTIFY_EMAIL=true
```

## 工作流示例

### 场景：完整的周期性反思

```
1. 星期日 22:00
   └─ ReflectionScheduler.runAllReflections()
      └─ 对每个注册的交易员
         └─ ReflectionEngine.AnalyzePeriod()
            ├─ 查询过去 7 天交易历史
            ├─ 计算 8+ 个指标
            │  ├─ 成功率
            │  ├─ Sharpe 比率
            │  ├─ 最大回撤
            │  └─ 信心度准确率（按区间分组）
            ├─ 调用 AI 分析（MCP）
            ├─ 解析 JSON 响应
            ├─ 分离建议为两类
            │  ├─ 交易系统 (parameter adjustment)
            │  └─ AI 学习 (memory + prompt injection)
            └─ 保存到数据库

2. 用户通过 API 查询
   GET /api/adjustment/{traderID}/pending
   └─ 显示 5 个待审批调整

3. 用户批准调整
   POST /api/adjustment/{id}/apply
   └─ 状态改为 APPLIED

4. 系统应用调整
   ├─ 加载已应用的 SystemAdjustment
   ├─ 更新交易参数
   └─ 下次 AI 决策使用新参数

5. 学习内存被注入
   AI 决策 → GetLearningMemoryForPrompt()
   └─ 在 system prompt 中注入历史经验
      "根据历史经验，BTC 交易对在大涨时期准确性更高..."
```

## 常见问题

### Q: 如何自定义反思时间？
A: 修改 `ReflectionScheduler.shouldRunReflection()` 方法，或使用 Cron 库：
```go
c.AddFunc("0 18 * * 1-5", func() {  // 工作日 18:00
    reflectionScheduler.runAllReflections()
})
```

### Q: 建议没有被应用怎么办？
A: 建议需要用户通过 API 明确批准才会应用。未批准的建议永远处于 PENDING 状态。

### Q: 学习内存如何影响 AI 决策？
A: 当 AI 进行决策时，系统会：
1. 调用 `GetLearningMemoryForPrompt(traderID, symbol)`
2. 获取置信度 ≥ 0.6 的活跃内存（包含 prompt_injection）
3. 将这些内存注入到 AI 的 system prompt 中
4. AI 基于这些信息做出更好的决策

### Q: 过期的学习内存会自动删除吗？
A: 不会自动删除，但可以通过 API 手动归档：
```bash
DELETE /api/memory/{memoryID}
```
或定期运行清理任务：
```go
reflectionScheduler.store.Reflection().DeleteExpiredMemory(traderID)
```

## 下一步

### 待实现功能

1. **前端仪表板**
   - 反思历史图表
   - 参数调整审批界面
   - 学习内存浏览器

2. **高级调度**
   - 支持 Cron 表达式
   - 自定义调度频率
   - 调度历史记录

3. **通知系统**
   - 反思完成通知
   - 待审批提醒
   - 异常告警

4. **性能优化**
   - 批量保存数据
   - 结果缓存
   - 增量分析（只分析新交易）

5. **审计日志**
   - 记录所有调整
   - 追踪谁批准了什么
   - 变更历史

## 文件清单

### 新增文件
- `backtest/reflection_scheduler.go` - 定时调度器
- `api/reflection_handlers.go` - API 处理器

### 修改文件
- `backtest/reflection_engine.go` - 更新 MCP 客户端集成
- `store/reflection.go` - 添加新接口方法
- `store/reflection_impl.go` - 实现新方法
- `main.go` - 需要初始化调度器并注册路由

## 测试检查清单

- [ ] 编译成功
- [ ] 定时调度器正常启动
- [ ] 可以手动触发反思分析
- [ ] API 端点返回正确数据
- [ ] 参数调整可以被保存和应用
- [ ] 学习内存被正确注入到 AI 提示中
- [ ] 过期的学习内存被清理

## 联系支持

如有问题或需要进一步的帮助，请联系开发团队。
