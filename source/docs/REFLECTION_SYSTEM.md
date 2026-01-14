## 🧠 反思系统架构文档

### 系统概述

**方案 A（双重反应）** 的完整实现，支持：
1. ✅ AI 反思分析 - 分析历史交易，生成改进建议
2. ✅ 交易系统调整 - 自动更新参数
3. ✅ AI 学习记忆 - 保存经验，指导未来决策
4. ✅ 可视化仪表板 - 展示交易质量评分

---

## 📊 数据模型

### 1. ReflectionRecord - 反思记录
```
id                  - 反思 ID
trader_id          - 交易员 ID
reflection_time    - 反思时间
period_start/end   - 分析周期
total_trades       - 总交易数
successful_trades  - 成功交易数
success_rate       - 成功率 (%)
average_pnl        - 平均盈亏
total_pnl          - 总盈亏
sharpe_ratio       - 风险调整收益率
max_drawdown       - 最大回撤
confidence_accuracy - 信心度准确率 (分组)
symbol_performance - 按交易对的表现
ai_reflection      - AI 反思内容（文本）
recommendations    - 具体建议（JSON 数组）
trade_system_advice - 交易系统建议
ai_learning_advice - AI 学习建议
```

### 2. SystemAdjustment - 系统参数调整
```
id                 - 调整 ID
trader_id          - 交易员 ID
reflection_id      - 关联的反思 ID
adjustment_time    - 调整时间
confidence_level   - 新的信心度阈值
btc_eth_leverage   - BTC/ETH 新杠杆
altcoin_leverage   - 山寨币新杠杆
max_position_size  - 最大仓位
max_daily_loss     - 日最大亏损
adjustment_reason  - 调整原因
status             - PENDING / APPLIED / REVERTED
applied_at         - 应用时间
```

### 3. AILearningMemory - AI 学习记忆
```
id                - 记忆 ID
trader_id         - 交易员 ID
reflection_id     - 来源反思 ID
memory_type       - 类型: "bias", "pattern", "lesson", "warning"
symbol            - 相关交易对（可选）
content           - 记忆内容（文本）
confidence        - 记忆可信度 (0-1)
usage_count       - 被使用次数
last_used_at      - 最后使用时间
prompt_injection  - AI prompt 注入内容
expires_at        - 过期时间（默认 1 个月）
```

---

## 🔄 工作流程

### 周期性反思流程

```
[每周末 22:00 触发]
  ↓
反思引擎.AnalyzePeriod()
  ├─ 获取交易历史（7 天）
  ├─ 计算统计指标
  │  ├─ 成功率、平均盈亏、夏普比率
  │  ├─ 按信心度分组计算准确率
  │  └─ 按交易对统计表现
  ├─ 调用 AI 进行反思分析
  │  └─ AI: "为什么失败了？如何改进？"
  ├─ 解析 AI 反思结果
  ├─ 分离建议
  │  ├─ 交易系统建议（参数调整）
  │  └─ AI 学习建议（经验记忆）
  └─ 保存 ReflectionRecord 到数据库
      ↓
反思引擎.ApplyRecommendations()
  ├─ 创建 SystemAdjustment 记录
  │  ├─ 解析建议
  │  ├─ 计算新参数值
  │  └─ 标记为 PENDING（等待用户确认）
  └─ 保存 AILearningMemory
     ├─ 记忆类型、内容、置信度
     ├─ 生成 prompt 注入内容
     └─ 设置 1 个月过期时间
```

### AI 学习流程

```
下次 AI 决策时：
  ↓
kernel.GetFullDecisionWithStrategy()
  ├─ 获取待决策交易对（如 BTCUSDT）
  ├─ 查询 AILearningMemory
  │  └─ SELECT * WHERE symbol='BTCUSDT' AND expires_at > NOW()
  ├─ 提取高可信度记忆（confidence >= 0.6）
  ├─ 注入 AI prompt
  │  └─ "基于过去的经验：[学习内容]..."
  └─ AI 结合记忆进行决策
     └─ "上次 BTC 分析偏乐观，这次保守一些"
```

### 交易系统调整流程

```
SystemAdjustment (PENDING)
  ↓
[用户在仪表板确认调整]
  ↓
Update status = APPLIED
  ├─ 信心度阈值: 75% → 65%
  ├─ BTC 杠杆: 5x → 3x
  ├─ 最大仓位: 10% → 7%
  └─ 其他参数...
      ↓
交易系统加载新参数
  ├─ 从 strategy_config 读取
  ├─ 对比 SystemAdjustment
  └─ 应用更新的参数
      ↓
[下个交易周期生效]
```

---

## 📈 统计指标计算

### 1. 成功率 (Success Rate)
```
成功率 = 盈利交易数 / 总交易数
```

### 2. 信心度准确率 (Confidence Accuracy)
```
按信心度分组（50%, 60%, ..., 100%）：
  50% 信心的交易中，60% 是盈利的
  75% 信心的交易中，82% 是盈利的
  
目标：信心度越高，准确率越高
```

### 3. 夏普比率 (Sharpe Ratio)
```
Sharpe = (平均收益 / 标准差) × √252

高夏普比率 = 稳定收益，低波动
```

### 4. 最大回撤 (Max Drawdown)
```
累计 PnL 从历史峰值下跌的最大幅度
示例：+100 → -50（回撤 150）
```

### 5. 胜负比 (Win/Loss Ratio)
```
胜负比 = 最大单笔盈利 / 最大单笔亏损
```

---

## 🎯 AI 反思提示词示例

```
您是一个交易系统分析专家。请根据以下数据进行反思：

交易统计：
- 总交易数：12 笔
- 成功率：75%（9 盈 3 亏）
- 平均收益：+2.1%
- 最大盈利：+8.5%（ETHUSDT）
- 最大亏损：-4.3%（ADAUSDT）
- 总收益：+25.2 USDT
- 夏普比率：1.8
- 最大回撤：-6.2%
- 胜负比：2.0

信心度准确率：
- 50-60% 信心：50% 准确（猜测太多）
- 70-80% 信心：89% 准确（表现好）
- 90%+ 信心：67% 准确（过度自信）

按交易对表现：
- BTCUSDT：4 笔，成功率 100%
- ETHUSDT：5 笔，成功率 80%
- ADAUSDT：3 笔，成功率 33%（需要改进）

请分析：
1. 信心度和实际表现的偏差在哪？
2. 哪个交易对最有问题？
3. 建议调整哪些参数？
4. 需要改进的具体方面？
```

### AI 反思响应示例

```
## 反思分析

### 关键发现：
1. **信心度偏差**：90%+ 信心的准确率低于预期（67% vs 80% 目标）
   → 问题：对强势币种判断过度自信
   → 建议：降低高信心度阈值 20%

2. **交易对问题**：ADAUSDT 成功率 33%，明显低于其他币
   → 问题：波动率高，支撑位判断不准
   → 建议：暂停交易该币，或降低杠杆

3. **BTC 表现优异**：100% 成功率
   → 原因：大币种波动平缓，支撑位清晰
   → 建议：增加 BTC 仓位配额

### 改进建议：

#### 交易系统调整：
1. 信心度阈值：75% → 70%（降低过度自信）
2. 山寨币杠杆：3x → 2x（降低风险）
3. 最大日亏损：5% → 3%（更严格的风险控制）

#### AI 学习建议：
- [LESSON] ADAUSDT 波动率高，需要宽松的支撑位判断
- [WARNING] 高信心度并不总是意味着高成功率，需要谨慎
- [PATTERN] BTC 稳定性强，可以增加杠杆和仓位
```

---

## 🛠️ 实现接口

### ReflectionStore 接口

```go
// 反思管理
SaveReflection(reflection *ReflectionRecord) error
GetReflectionByID(id string) (*ReflectionRecord, error)
GetRecentReflections(traderID string, limit int) ([]*ReflectionRecord, error)
GetReflectionByPeriod(traderID, startTime, endTime time.Time) (*ReflectionRecord, error)

// 系统调整管理
SaveSystemAdjustment(adjustment *SystemAdjustment) error
GetAdjustmentsByStatus(traderID string, status string) ([]*SystemAdjustment, error)
GetLatestAdjustment(traderID string) (*SystemAdjustment, error)
UpdateAdjustmentStatus(id string, status string, appliedAt *time.Time) error
GetAdjustmentHistory(traderID string, limit int) ([]*SystemAdjustment, error)

// AI 学习记忆管理
SaveLearningMemory(memory *AILearningMemory) error
GetActiveLearningMemory(traderID string) ([]*AILearningMemory, error)
GetLearningMemoryBySymbol(traderID, symbol string) ([]*AILearningMemory, error)
UpdateMemoryUsage(id string) error
DeleteExpiredMemory(traderID string) error
GetLearningMemoryForPrompt(traderID string, symbol string) ([]string, error)

// 统计
GetReflectionStats(traderID string, days int) (map[string]interface{}, error)
```

### ReflectionEngine 接口

```go
// 主要方法
AnalyzePeriod(traderID, startTime, endTime time.Time) (*ReflectionRecord, error)
ApplyRecommendations(reflection *ReflectionRecord) error

// 辅助方法
calculateStats(trades []*TradeHistoryRecord) *TradeStats
getAIReflection(traderID, trades, stats) (string, error)
separateAdvice(recommendations string) ([]json.RawMessage, []json.RawMessage)
```

---

## 📱 API 端点 (待实现)

### 反思 API

```
GET  /api/reflection/{traderID}/recent      - 获取最近反思
GET  /api/reflection/{id}                   - 获取反思详情
GET  /api/reflection/{traderID}/period      - 获取周期反思（?start=&end=）
POST /api/reflection/{traderID}/analyze     - 手动触发反思
```

### 调整 API

```
GET  /api/adjustment/{traderID}/pending     - 获取待审批调整
GET  /api/adjustment/{traderID}/history     - 获取调整历史
POST /api/adjustment/{id}/apply             - 批准调整
POST /api/adjustment/{id}/revert            - 回滚调整
```

### 学习记忆 API

```
GET /api/memory/{traderID}                  - 获取记忆列表
GET /api/memory/{traderID}?symbol=BTCUSDT   - 获取特定币种记忆
DELETE /api/memory/{id}                     - 删除记忆（人工干预）
POST /api/memory/{id}/refresh               - 刷新记忆过期时间
```

### 仪表板 API

```
GET /api/dashboard/{traderID}               - 获取仪表板数据
  返回：最近反思、待审批调整、统计数据、学习记忆概览
```

---

## 🎨 前端仪表板 (待实现)

### 核心组件

1. **反思摘要卡片**
   - 周期统计（成功率、盈亏、夏普比率）
   - AI 反思简述
   - 主要建议列表

2. **调整审批面板**
   - 待审批调整列表
   - 参数对比（旧 vs 新）
   - 调整原因展示
   - 批准/回滚按钮

3. **学习记忆浏览**
   - 按交易对过滤
   - 记忆类型筛选
   - 可信度排序
   - 删除/刷新功能

4. **历史分析图表**
   - 成功率趋势
   - 信心度准确率变化
   - 参数调整历史时间线
   - 交易对表现对比

---

## 🔐 权限控制

- 只有**交易员自己**能看到自己的反思和调整
- **系统管理员**能查看所有反思（审计）
- 调整需要**交易员确认**才能生效（防止自动过度调整）
- 学习记忆**自动过期**，防止历史偏见积累

---

## ⚙️ 配置参数

```go
// reflection_config.yaml
reflection:
  enabled: true
  schedule: "0 22 * * 0"  // 每周日 22:00
  
  analysis_period_days: 7
  
  memory_config:
    default_expiry_days: 30
    min_confidence: 0.6      // prompt 注入的最低置信度
    max_memories_per_symbol: 10
  
  adjustment_limits:
    max_confidence_change: 0.2  // 单次最多改变 ±20%
    max_leverage_change: 2      // 单次最多改变 ±2x
    max_position_change: 0.05   // 单次最多改变 ±5%
  
  ai_reflection:
    enabled: true
    model: "deepseek"  // 或 "qwen"
    temperature: 0.7
```

---

## 📝 迁移指南

### 第 1 步：数据库迁移
```sql
-- 自动执行 (GORM AutoMigrate)
-- 创建 3 个新表：
-- - reflections
-- - system_adjustments
-- - ai_learning_memory
```

### 第 2 步：集成反思引擎
```go
reflectionEngine := backtest.NewReflectionEngine(mcpClient, store)

// 在定时任务中调用
reflection, err := reflectionEngine.AnalyzePeriod(
    traderID, 
    startTime, 
    endTime,
)
if err == nil {
    reflectionEngine.ApplyRecommendations(reflection)
}
```

### 第 3 步：更新 AI prompt
```go
// 在 kernel.GetFullDecisionWithStrategy() 中添加
memories, _ := store.Reflection().GetLearningMemoryForPrompt(
    traderID, 
    symbol,
)
// 将 memories 注入 system prompt
```

### 第 4 步：添加前端 API 和 UI
```
web/src/pages/Reflection.tsx
web/src/components/ReflectionSummary.tsx
web/src/components/AdjustmentApproval.tsx
```

---

## 📊 期望效果

### 短期（1-2 周）
- ✅ 识别信心度偏差
- ✅ 发现最差表现的交易对
- ✅ 生成初始参数建议

### 中期（1-2 个月）
- ✅ 累积学习记忆（10+ 个）
- ✅ 参数逐步优化
- ✅ AI 决策质量提升 5-10%
- ✅ 成功率稳步上升

### 长期（3+ 个月）
- ✅ 完整的经验知识库
- ✅ 自适应策略（参数自动调整）
- ✅ AI 质量明显改善（+15%+）
- ✅ 系统更稳定、收益更高

---

## 🚀 下一步

1. **实现 AI 反思** - 集成 MCP 客户端，生成真实反思文本
2. **前端仪表板** - 构建 React 组件展示反思和调整
3. **自动化调度** - 添加定时任务每周触发反思
4. **监控告警** - 当建议参数变化过大时预警
5. **回测验证** - 验证参数调整对历史交易的影响

