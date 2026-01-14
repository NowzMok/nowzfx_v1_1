# 待执行订单表格配置和常见问题

## 📋 配置设置

### 组件配置选项

PendingOrdersTable 组件支持以下配置参数：

```typescript
interface PendingOrdersTableProps {
  traderId: string           // 交易员ID（必需）
  autoRefresh?: boolean      // 是否自动刷新（默认：true）
  refreshInterval?: number   // 刷新间隔毫秒（默认：30000）
}
```

### 使用示例

```typescript
// 基础用法
<PendingOrdersTable traderId="your_trader_id" />

// 禁用自动刷新
<PendingOrdersTable 
  traderId="your_trader_id" 
  autoRefresh={false} 
/>

// 自定义刷新间隔（每10秒）
<PendingOrdersTable 
  traderId="your_trader_id"
  autoRefresh={true}
  refreshInterval={10000}
/>
```

### 后端配置

#### 1. 订单过期时间
文件：`nofx/store/analysis_impl.go`

```go
// 默认1天过期
order.ExpiresAt = time.Now().UTC().Add(24 * time.Hour)
```

#### 2. 重复订单检测
文件：`nofx/trader/order_deduplication.go`

```go
// 检测时间窗口（默认1小时）
timeWindow := time.Now().Add(-1 * time.Hour)

// 相似度阈值（默认0.95）
similarityThreshold := 0.95
```

#### 3. 清理策略
文件：`nofx/api/order_cleanup_handlers.go`

```go
// 自动清理配置
maxPendingOrders := 50      // 最大待执行订单数
cleanupInterval := 6 * time.Hour // 清理间隔
```

## 🔧 常见问题

### Q1: 分组折叠功能是如何工作的？

**A**: 系统自动按币种分组订单，只显示置信度最高的订单，支持展开/折叠查看所有重复订单。

**工作原理**：
1. **自动分组**：按交易对（如 BTCUSDT、ETHUSDT）分组
2. **最佳选择**：选择置信度最高的订单作为代表
3. **折叠显示**：默认只显示最佳订单
4. **展开查看**：点击币种行查看所有订单

**视觉标识**：
- **黄色高亮**：表示该币种有重复订单
- **"最佳"标签**：置信度最高或最新的订单
- **展开图标**：`>` 表示可展开，`v` 表示已展开
- **计数标记**：显示该币种的订单数量

**交互说明**：
```typescript
// 点击币种行切换展开状态
const toggleGroup = (symbol: string) => {
  setExpandedGroups(prev => {
    const newSet = new Set(prev)
    if (newSet.has(symbol)) {
      newSet.delete(symbol)
    } else {
      newSet.add(symbol)
    }
    return newSet
  })
}
```

**数据处理逻辑**：
```typescript
// 按币种分组并找出最佳订单
const groupOrdersBySymbol = (): GroupedOrders => {
  const groups: GroupedOrders = {}
  
  orders.forEach(order => {
    if (!groups[order.symbol]) {
      groups[order.symbol] = {
        best: order,
        all: [order],
        count: 1
      }
    } else {
      groups[order.symbol].all.push(order)
      groups[order.symbol].count++
      
      // 更新最佳订单
      const currentBest = groups[order.symbol].best
      if (
        order.confidence > currentBest.confidence ||
        (order.confidence === currentBest.confidence && 
         new Date(order.created_at) > new Date(currentBest.created_at))
      ) {
        groups[order.symbol].best = order
      }
    }
  })
  
  return groups
}
```

### Q2: 为什么有些订单被标记为"重复"？

### Q1: 为什么有些订单被标记为"重复"？

**A**: 当同一交易对（如 BTCUSDT）存在多个待执行订单时，系统会：
1. 按币种分组
2. 选择置信度最高的订单作为代表
3. 标记为重复订单，便于管理

**影响**：后端会自动清理重复订单，前端提供可视化管理界面。

### Q2: 如何查看所有重复订单？

**A**: 点击币种行（黄色高亮区域）：
- 展开：显示该币种所有订单
- 折叠：只显示最佳订单
- 最佳订单会标记"最佳"标签

### Q3: 重复订单的判定标准是什么？

**A**: 基于以下条件判断重复：
- 相同交易对（symbol）
- 相似的价格区间（触发价、目标价）
- 相同的方向（做多/做空）
- 时间窗口内创建（默认1小时）

### Q4: 重复订单会被自动清理吗？

**A**: 是的，后端有自动清理机制：
- 保留置信度最高的订单
- 自动取消其他重复订单
- 清理间隔：6小时
- 可通过 API 手动触发清理

### Q5: 如何调整分组折叠的行为？

**A**: 修改组件代码中的相关逻辑：

```typescript
// 修改最佳订单选择逻辑
if (
  order.confidence > currentBest.confidence ||
  (order.confidence === currentBest.confidence && 
   new Date(order.created_at) > new Date(currentBest.created_at))
) {
  groups[order.symbol].best = order
}

// 修改重复订单阈值（需要后端配合）
// 修改 nofx/trader/order_deduplication.go 中的相似度阈值
```

### Q6: 为什么看不到分组折叠效果？

**A**: 可能原因：
1. 没有重复订单（正常现象）
2. 订单数据为空
3. 后端未正确标记重复订单

**检查方法**：
```bash
# 查看是否有重复订单
sqlite3 data/data.db "SELECT symbol, COUNT(*) as count FROM pending_orders WHERE status='PENDING' GROUP BY symbol HAVING count > 1;"
```

### Q7: 如何自定义统计卡片？

**A**: 修改 StatCard 组件或统计逻辑：

```typescript
// 在组件中添加新的统计项
<StatCard 
  label="自定义统计" 
  value={yourCustomValue} 
  color="#FFFFFF" 
  icon="🎯"
/>
```

### Q8: 自动刷新不工作怎么办？

**A**: 检查以下配置：

```typescript
// 1. 确保 autoRefresh 为 true
autoRefresh={true}

// 2. 检查刷新间隔（毫秒）
refreshInterval={30000} // 30秒

// 3. 确保 API 端点可用
// GET /api/orders/pending/{traderId}
```

### Q9: 如何修改表格样式？

**A**: 组件使用 Tailwind CSS 和内联样式：

```typescript
// 修改颜色主题
style={{ 
  background: 'linear-gradient(135deg, #1E2329 0%, #181C21 100%)',
  border: '1px solid #2B3139',
  color: '#EAECEF'
}}

// 修改悬停效果
className="transition-all duration-200 hover:bg-white/5"
```

### Q10: 如何添加新的订单状态？

**A**: 修改状态映射：

```typescript
const statusMap = {
  PENDING: { /* ... */ },
  TRIGGERED: { /* ... */ },
  FILLED: { /* ... */ },
  CANCELLED: { /* ... */ },
  EXPIRED: { /* ... */ },
  // 添加新状态
  YOUR_STATUS: { 
    icon: <YourIcon className="w-3.5 h-3.5" />, 
    color: '#FFFFFF', 
    bg: 'rgba(255, 255, 255, 0.15)',
    text: '你的状态' 
  }
}
```

### Q11: API 返回数据格式是什么？

**A**: 待执行订单数据格式：

```typescript
interface PendingOrder {
  id: string
  trader_id: string
  symbol: string
  analysis_id: string
  target_price: number
  trigger_price: number
  position_size: number
  leverage: number
  stop_loss: number
  take_profit: number
  confidence: number
  status: 'PENDING' | 'TRIGGERED' | 'FILLED' | 'CANCELLED' | 'EXPIRED'
  created_at: string
  expires_at: string
  triggered_price?: number
  triggered_at?: string
  filled_at?: string
  cancel_reason?: string
  order_id?: number
}
```

### Q12: 如何处理大量订单的性能问题？

**A**: 优化建议：

```typescript
// 1. 调整刷新间隔
refreshInterval={60000} // 1分钟

// 2. 禁用自动刷新，手动刷新
autoRefresh={false}
// 提供刷新按钮
<button onClick={fetchPendingOrders}>刷新</button>

// 3. 分页加载（需要后端支持）
const fetchPendingOrders = async (page = 1, limit = 20) => {
  const data = await api.getPendingOrders(traderId, page, limit)
  setOrders(data)
}
```

### Q13: 如何添加订单详情弹窗？

**A**: 扩展组件：

```typescript
const [selectedOrder, setSelectedOrder] = useState<PendingOrder | null>(null)

// 在 renderOrderRow 中添加点击事件
<tr 
  key={order.id}
  onClick={() => setSelectedOrder(order)}
  className="cursor-pointer hover:bg-white/5"
>

// 添加详情弹窗
{selectedOrder && (
  <OrderDetailModal 
    order={selectedOrder}
    onClose={() => setSelectedOrder(null)}
  />
)}
```

### Q14: 如何导出订单数据？

**A**: 添加导出功能：

```typescript
const exportToCSV = () => {
  const headers = ['交易对', '方向', '目标价', '触发价', '仓位', '杠杆', '置信度', '状态']
  const rows = orders.map(order => [
    order.symbol,
    order.take_profit > order.stop_loss ? '做多' : '做空',
    order.target_price,
    order.trigger_price,
    order.position_size,
    order.leverage,
    (order.confidence * 100).toFixed(0) + '%',
    order.status
  ])
  
  const csv = [headers, ...rows].map(row => row.join(',')).join('\n')
  const blob = new Blob([csv], { type: 'text/csv' })
  const url = URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = url
  a.download = `pending_orders_${new Date().toISOString()}.csv`
  a.click()
}
```

### Q15: 如何添加订单操作按钮？

**A**: 在订单行中添加操作列：

```typescript
<td className="px-4 py-3">
  <div className="flex gap-2">
    <button 
      onClick={() => handleCancel(order.id)}
      className="text-xs px-2 py-1 rounded bg-red-500/20 text-red-500 hover:bg-red-500/30"
    >
      取消
    </button>
    <button 
      onClick={() => handleEdit(order)}
      className="text-xs px-2 py-1 rounded bg-blue-500/20 text-blue-500 hover:bg-blue-500/30"
    >
      编辑
    </button>
  </div>
</td>
```

## 🔍 故障排查

### 问题：订单数据不显示

**检查清单**：
1. ✅ 交易员ID是否正确
2. ✅ 后端API是否运行正常
3. ✅ 数据库中是否有待执行订单
4. ✅ 网络请求是否成功

**调试命令**：
```bash
# 检查数据库
sqlite3 data/data.db "SELECT COUNT(*) FROM pending_orders WHERE trader_id='your_trader_id' AND status='PENDING';"

# 测试API
curl http://localhost:8080/api/orders/pending/your_trader_id
```

### 问题：自动刷新失效

**解决方案**：
1. 检查 `autoRefresh` 参数
2. 确认 `refreshInterval` 大于0
3. 检查浏览器控制台是否有错误
4. 确认API端点可访问

### 问题：分组显示异常

**可能原因**：
1. 订单数据格式错误
2. 分组逻辑被修改
3. 浏览器缓存问题

**解决方法**：
```typescript
// 清空缓存并重新加载
localStorage.clear()
window.location.reload()
```

## 📞 获取帮助

如果遇到无法解决的问题：

1. **查看日志**：浏览器控制台和后端日志
2. **检查API**：使用 Postman 测试API端点
3. **验证数据**：直接查询数据库
4. **参考文档**：查看其他相关文档
5. **提交Issue**：在项目仓库提交详细Issue

---

**最后更新**: 2026-01-12
**版本**: 1.0.0
