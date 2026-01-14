# 反思系统 - 快速参考卡

## 🚀 5分钟快速开始

### 1. 在 main.go 中初始化
```go
import (
    "nofx/backtest"
    "nofx/api"
)

// 在应用启动时
reflectionEngine := backtest.NewReflectionEngine(aiClient, store)
scheduler := backtest.NewReflectionScheduler(reflectionEngine, store)
scheduler.RegisterTrader("trader_id_1")
scheduler.RegisterTrader("trader_id_2")
scheduler.Start()

// 注册 API 路由
handlers := api.NewReflectionHandlers(scheduler, store)
handlers.RegisterReflectionRoutes(router)

// 关闭时
defer scheduler.Stop()
```

### 2. 测试 API
```bash
# 获取最近反思
curl http://localhost:8080/api/reflection/trader_id_1/recent

# 手动触发反思
curl -X POST http://localhost:8080/api/reflection/trader_id_1/analyze

# 查看待审批调整
curl http://localhost:8080/api/adjustment/trader_id_1/pending

# 批准调整
curl -X POST http://localhost:8080/api/adjustment/adj_id/apply
```

## 📊 工作流概览

```
┌────────────────────────────┐
│ 每周日 22:00               │
│ 或手动 POST /analyze       │
└────────────┬───────────────┘
             │
             ▼
┌────────────────────────────┐
│ 收集 7 天交易历史          │
└────────────┬───────────────┘
             │
             ▼
┌────────────────────────────┐
│ 计算 8+ 指标               │
│ 成功率、Sharpe、最大回撤等 │
└────────────┬───────────────┘
             │
             ▼
┌────────────────────────────┐
│ AI 分析 (MCP)              │
│ 生成 JSON 格式建议         │
└────────────┬───────────────┘
             │
             ▼
┌────────────────────────────┐
│ 建议分类                   │
│ 交易系统 ↔ AI 学习         │
└────────────┬───────────────┘
             │
    ┌────────┴────────┐
    ▼                 ▼
┌─────────────┐  ┌─────────────┐
│ 参数调整    │  │ 学习内存    │
│ PENDING     │  │ 30天过期    │
└─────────────┘  └─────────────┘
    │                 │
    ▼                 ▼
用户批准           AI 注入提示
```

## 📋 API 快速参考

### 反思管理
```
GET  /api/reflection/{id}                     查看反思详情
GET  /api/reflection/{traderID}/recent        最近反思列表
POST /api/reflection/{traderID}/analyze       手动触发反思
GET  /api/reflection/{traderID}/stats         获取统计
```

### 参数调整（审批工作流）
```
GET  /api/adjustment/{traderID}/pending       待审批列表
POST /api/adjustment/{id}/apply               ✓ 批准
POST /api/adjustment/{id}/reject              ✗ 拒绝
POST /api/adjustment/{id}/revert              ⏮ 撤销已应用
GET  /api/adjustment/{traderID}/history       历史记录
```

### 学习内存
```
GET  /api/memory/{traderID}                   查看内存
DELETE /api/memory/{id}                       归档内存
```

## 🔑 关键概念

### ReflectionRecord (反思记录)
- **何时创建**: 每周日 22:00 或手动触发
- **包含内容**: 统计指标 + AI 分析 + 建议
- **保存时间**: 永久
- **作用**: 历史追踪，性能分析

### SystemAdjustment (参数调整)
- **何时创建**: 反思生成时（来自 AI 建议）
- **初始状态**: PENDING（待审批）
- **可能的转移**:
  - PENDING → APPLIED（用户批准）
  - PENDING → REJECTED（用户拒绝）
  - APPLIED → REVERTED（用户撤销）
- **作用**: 自动优化交易参数

### AILearningMemory (学习内存)
- **何时创建**: 反思生成时（来自 AI 学习建议）
- **生命周期**: 30 天（可自定义）
- **保存内容**:
  - `memory_type`: bias, pattern, lesson, warning
  - `prompt_injection`: 注入 AI 下次提示的内容
  - `confidence`: 0.0-1.0（只注入 ≥0.6 的内存）
- **作用**: AI 持续学习和改进

## 💡 常见场景

### 场景 1: 识别过度自信
```
问题: 90% 置信度的交易只有 60% 成功率
解决: AI 建议降低置信度阈值
步骤:
  1. GET /api/reflection/trader/stats
  2. 查看 confidence_accuracy 的 90% 行
  3. GET /api/adjustment/trader/pending
  4. 查看建议并 POST /apply
```

### 场景 2: 某个交易对表现差
```
问题: ALTUSDT 成功率只有 40%
解决: 降低杠杆或增加止损
步骤:
  1. GET /api/reflection/{id}
  2. 检查 symbol_performance 中 ALTUSDT 数据
  3. 系统自动创建调整建议
  4. 用户审批应用
```

### 场景 3: 使用历史经验
```
流程:
  1. 上周学到: "BTC 在大涨时准确率高"
  2. 保存为 AILearningMemory
  3. 本周 AI 决策时
  4. 系统自动注入: "根据历史，BTC 大涨时准确率高"
  5. AI 基于此信息做出更好决策
```

## 🛠️ 配置和调优

### 修改分析周期
```go
scheduler.SetAnalysisDays(14)  // 分析 14 天而不是 7 天
```

### 修改调度时间
```go
scheduler.SetSchedule("0 18 * * 1-5")  // 周一至周五 18:00
```

### 手动触发反思
```go
scheduler.ManualTrigger("trader_id")  // 立即分析
```

### 注册/注销交易员
```go
scheduler.RegisterTrader("new_trader")     // 新增交易员
scheduler.UnregisterTrader("old_trader")   // 停止跟踪
```

## 📈 监控和调试

### 查看最近 10 次反思
```bash
curl "http://localhost:8080/api/reflection/trader_id/recent?limit=10"
```

### 查看统计信息（过去 30 天）
```bash
curl "http://localhost:8080/api/reflection/trader_id/stats?days=30"
```

### 检查待审批调整
```bash
curl "http://localhost:8080/api/adjustment/trader_id/pending"
```

### 查看学习内存（当前活跃的）
```bash
curl "http://localhost:8080/api/memory/trader_id?limit=50"
```

### 查看调整历史
```bash
curl "http://localhost:8080/api/adjustment/trader_id/history?limit=50"
```

## 🚨 常见问题排查

### Q: 反思没有生成怎么办？
**A**: 检查:
1. 是否有交易数据？ → 需要至少 1 笔交易
2. AI 客户端是否正常？ → 检查日志
3. 调度器是否启动？ → 检查 logger 输出
```bash
# 手动触发测试
curl -X POST http://localhost:8080/api/reflection/trader_id/analyze
```

### Q: 建议没有被创建？
**A**: 检查:
1. 反思是否成功生成？ → GET /api/reflection/{traderID}/recent
2. AI 是否返回了有效的 JSON？ → 查看日志
3. 建议格式是否正确？ → 应包含 recommendations 和 learning_memories

### Q: 学习内存没有被使用？
**A**: 检查:
1. 内存是否过期？ → expires_at > now()
2. 置信度是否足够？ → confidence >= 0.6
3. AI 代码是否注入内存？ → 需要在 kernel/engine.go 中实现

### Q: API 返回 404？
**A**: 检查:
1. 路由是否注册了？ → handlers.RegisterReflectionRoutes(router)
2. ID 是否正确？ → 使用 GET 端点确认存在
3. 数据库记录是否存在？ → 查看日志

## 📊 性能优化

### 对于大量交易员
```go
// 限制并发反思数
scheduler.maxConcurrent = 5  // 默认 3
```

### 对于长期历史
```go
// 定期清理过期的学习内存
store.Reflection().DeleteExpiredMemory(traderID)
```

### 对于 AI 调用
```go
// 设置 AI 客户端超时
aiClient.SetTimeout(30 * time.Second)
```

## 🔗 相关文件

| 文件 | 描述 |
|------|------|
| `backtest/reflection_scheduler.go` | 定时调度器 |
| `backtest/reflection_engine.go` | 分析引擎 |
| `api/reflection_handlers.go` | REST API |
| `store/reflection.go` | 数据模型 |
| `store/reflection_impl.go` | 数据库实现 |
| `docs/REFLECTION_SYSTEM_IMPLEMENTATION.md` | 详细指南 |
| `docs/reflection_integration_example.go` | 代码示例 |

## 📞 获取帮助

1. 查看详细文档: `docs/REFLECTION_SYSTEM_IMPLEMENTATION.md`
2. 参考代码示例: `docs/reflection_integration_example.go`
3. 查看日志输出: 搜索 🔍 符号的日志
4. 联系开发团队

---

**最后更新**: 2024-01-15  
**版本**: 1.0.0  
**状态**: ✅ 生产就绪
