# 前端反思系统集成指南

## 🎯 概述

反思系统（Option A）已在后端完全实现，提供了 **12 个 REST API 端点**用于前端调用。本指南介绍如何在前端查看和使用反思系统。

---

## 📡 可用的 API 端点

### 1. 反思记录端点 (Reflection Endpoints)

#### 获取最近的反思记录
```
GET /api/reflection/{traderID}/recent?limit=10
```
**参数:**
- `traderID` - 交易者 ID
- `limit` - 返回记录数 (1-100, 默认 10)

**响应示例:**
```json
{
  "data": [
    {
      "id": "ref_123",
      "traderID": "trader_001",
      "analysisType": "performance",
      "findings": "Win rate improved by 5%",
      "timestamp": "2025-01-12T10:00:00Z",
      "severity": "info"
    }
  ],
  "count": 1
}
```

#### 触发分析反思
```
POST /api/reflection/{traderID}/analyze
```
**请求体:**
```json
{
  "analysisType": "performance|risk|strategy",
  "timeRange": {
    "startTime": "2025-01-01T00:00:00Z",
    "endTime": "2025-01-12T23:59:59Z"
  }
}
```

#### 获取反思统计信息
```
GET /api/reflection/{traderID}/stats
```
**响应:**
```json
{
  "totalReflections": 15,
  "averageSeverity": "medium",
  "lastReflectionTime": "2025-01-12T10:00:00Z",
  "findingsByType": {
    "performance": 8,
    "risk": 4,
    "strategy": 3
  }
}
```

#### 按 ID 获取反思详情
```
GET /api/reflection/id/{reflectionID}
```

---

### 2. 调整端点 (Adjustment Endpoints)

#### 获取待处理的调整建议
```
GET /api/adjustment/{traderID}/pending
```
**响应:**
```json
{
  "data": [
    {
      "id": "adj_456",
      "traderID": "trader_001",
      "suggestedAction": "increase_position_size",
      "reasoning": "Win rate trending up",
      "priority": "high",
      "status": "pending"
    }
  ],
  "count": 1
}
```

#### 获取调整历史
```
GET /api/adjustment/{traderID}/history
```

#### 应用调整建议
```
POST /api/adjustment/{adjustmentID}/apply
```

#### 拒绝调整建议
```
POST /api/adjustment/{adjustmentID}/reject
```

#### 恢复已应用的调整
```
POST /api/adjustment/{adjustmentID}/revert
```

---

### 3. 学习记忆端点 (Learning Memory Endpoints)

#### 获取学习记忆
```
GET /api/memory/{traderID}
```
**响应:**
```json
{
  "data": [
    {
      "id": "mem_789",
      "traderID": "trader_001",
      "lessonLearned": "High volatility reduces win rate",
      "timestamp": "2025-01-12T10:00:00Z"
    }
  ],
  "count": 1
}
```

#### 删除学习记忆
```
DELETE /api/memory/{memoryID}
```

---

## 🚀 前端集成方案

### 方案 1: 添加反思系统页面 (推荐)

#### 1. 创建反思系统组件

**文件位置:** `web/src/components/ReflectionDashboard.tsx`

```typescript
import { useState, useEffect } from 'react'
import { useParams } from 'react-router-dom'

interface Reflection {
  id: string
  traderID: string
  analysisType: string
  findings: string
  timestamp: string
  severity: 'info' | 'warning' | 'error'
}

interface Adjustment {
  id: string
  traderID: string
  suggestedAction: string
  reasoning: string
  priority: 'low' | 'medium' | 'high'
  status: 'pending' | 'applied' | 'rejected'
}

export function ReflectionDashboard() {
  const { traderID } = useParams<{ traderID: string }>()
  const [reflections, setReflections] = useState<Reflection[]>([])
  const [adjustments, setAdjustments] = useState<Adjustment[]>([])
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    if (!traderID) return

    // 获取最近的反思
    Promise.all([
      fetch(`/api/reflection/${traderID}/recent?limit=20`)
        .then(r => r.json())
        .then(data => setReflections(data.data || [])),
      
      fetch(`/api/adjustment/${traderID}/pending`)
        .then(r => r.json())
        .then(data => setAdjustments(data.data || []))
    ]).finally(() => setLoading(false))
  }, [traderID])

  if (loading) return <div>加载中...</div>

  return (
    <div className="space-y-6 p-6">
      {/* 反思记录部分 */}
      <div>
        <h2 className="text-2xl font-bold mb-4">系统反思</h2>
        <div className="grid gap-4">
          {reflections.map(r => (
            <div key={r.id} className="p-4 border rounded-lg bg-slate-50">
              <div className="flex items-start justify-between">
                <div>
                  <h3 className="font-semibold capitalize">{r.analysisType}</h3>
                  <p className="text-gray-600 mt-2">{r.findings}</p>
                </div>
                <span className={`px-3 py-1 rounded text-sm font-medium ${
                  r.severity === 'error' ? 'bg-red-100 text-red-800' :
                  r.severity === 'warning' ? 'bg-yellow-100 text-yellow-800' :
                  'bg-blue-100 text-blue-800'
                }`}>
                  {r.severity}
                </span>
              </div>
              <p className="text-gray-400 text-sm mt-2">
                {new Date(r.timestamp).toLocaleString()}
              </p>
            </div>
          ))}
        </div>
      </div>

      {/* 待处理调整部分 */}
      <div>
        <h2 className="text-2xl font-bold mb-4">待处理的调整建议</h2>
        <div className="grid gap-4">
          {adjustments.filter(a => a.status === 'pending').map(a => (
            <div key={a.id} className="p-4 border rounded-lg bg-amber-50">
              <div className="flex items-start justify-between">
                <div>
                  <h3 className="font-semibold">{a.suggestedAction}</h3>
                  <p className="text-gray-600 mt-2">{a.reasoning}</p>
                </div>
                <span className={`px-3 py-1 rounded text-sm font-medium ${
                  a.priority === 'high' ? 'bg-red-100 text-red-800' :
                  a.priority === 'medium' ? 'bg-yellow-100 text-yellow-800' :
                  'bg-green-100 text-green-800'
                }`}>
                  {a.priority}
                </span>
              </div>
              <div className="flex gap-2 mt-4">
                <button
                  onClick={() => handleApply(a.id)}
                  className="px-4 py-2 bg-green-600 text-white rounded hover:bg-green-700"
                >
                  应用
                </button>
                <button
                  onClick={() => handleReject(a.id)}
                  className="px-4 py-2 bg-gray-400 text-white rounded hover:bg-gray-500"
                >
                  拒绝
                </button>
              </div>
            </div>
          ))}
        </div>
      </div>
    </div>
  )

  async function handleApply(adjustmentID: string) {
    await fetch(`/api/adjustment/${adjustmentID}/apply`, { method: 'POST' })
    // 刷新数据
  }

  async function handleReject(adjustmentID: string) {
    await fetch(`/api/adjustment/${adjustmentID}/reject`, { method: 'POST' })
    // 刷新数据
  }
}
```

#### 2. 在路由中添加反思系统页面

**文件位置:** `web/src/App.tsx`

```typescript
import { ReflectionDashboard } from './components/ReflectionDashboard'

function App() {
  return (
    <BrowserRouter>
      <Routes>
        {/* 其他路由 */}
        <Route path="/traders/:traderID/reflection" element={<ReflectionDashboard />} />
      </Routes>
    </BrowserRouter>
  )
}
```

#### 3. 在交易者仪表板中添加导航链接

**文件位置:** `web/src/components/TraderDashboardPage.tsx`

```typescript
<nav className="mb-4">
  <Link to={`/traders/${traderID}/dashboard`} className="tab active">
    交易仪表板
  </Link>
  <Link to={`/traders/${traderID}/reflection`} className="tab">
    📊 系统反思
  </Link>
</nav>
```

---

### 方案 2: 添加反思小组件到现有仪表板

如果不想创建新页面，可以在现有的 `TraderDashboardPage.tsx` 中添加反思小组件：

```typescript
import { ReflectionWidget } from './ReflectionWidget'

export function TraderDashboardPage() {
  return (
    <div className="grid grid-cols-3 gap-6">
      {/* 其他仪表板内容 */}
      
      {/* 反思小组件 */}
      <ReflectionWidget traderID={traderID} />
    </div>
  )
}
```

**反思小组件文件:** `web/src/components/ReflectionWidget.tsx`

```typescript
export function ReflectionWidget({ traderID }: { traderID: string }) {
  const [reflections, setReflections] = useState<Reflection[]>([])
  
  useEffect(() => {
    fetch(`/api/reflection/${traderID}/recent?limit=5`)
      .then(r => r.json())
      .then(data => setReflections(data.data || []))
  }, [traderID])

  return (
    <div className="bg-white p-4 rounded-lg shadow">
      <h3 className="text-lg font-bold mb-3">最近反思</h3>
      <div className="space-y-2 max-h-96 overflow-y-auto">
        {reflections.slice(0, 5).map(r => (
          <div key={r.id} className="text-sm p-2 bg-gray-50 rounded">
            <div className="font-medium text-gray-800">{r.findings}</div>
            <div className="text-gray-400 text-xs mt-1">
              {new Date(r.timestamp).toLocaleString()}
            </div>
          </div>
        ))}
      </div>
    </div>
  )
}
```

---

### 方案 3: 使用 HTTP 客户端库

#### 创建 API 服务文件

**文件位置:** `web/src/services/reflectionApi.ts`

```typescript
export const reflectionApi = {
  // 获取最近反思
  getRecentReflections: (traderID: string, limit: number = 10) =>
    fetch(`/api/reflection/${traderID}/recent?limit=${limit}`).then(r => r.json()),

  // 获取反思统计
  getReflectionStats: (traderID: string) =>
    fetch(`/api/reflection/${traderID}/stats`).then(r => r.json()),

  // 触发分析
  triggerAnalysis: (traderID: string, analysisType: string) =>
    fetch(`/api/reflection/${traderID}/analyze`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ analysisType })
    }).then(r => r.json()),

  // 获取待处理调整
  getPendingAdjustments: (traderID: string) =>
    fetch(`/api/adjustment/${traderID}/pending`).then(r => r.json()),

  // 应用调整
  applyAdjustment: (adjustmentID: string) =>
    fetch(`/api/adjustment/${adjustmentID}/apply`, { method: 'POST' })
      .then(r => r.json()),

  // 拒绝调整
  rejectAdjustment: (adjustmentID: string) =>
    fetch(`/api/adjustment/${adjustmentID}/reject`, { method: 'POST' })
      .then(r => r.json()),

  // 获取学习记忆
  getLearningMemories: (traderID: string) =>
    fetch(`/api/memory/${traderID}`).then(r => r.json()),
}
```

#### 在组件中使用

```typescript
import { reflectionApi } from '@/services/reflectionApi'

export function MyComponent() {
  const [reflections, setReflections] = useState([])

  useEffect(() => {
    reflectionApi.getRecentReflections(traderID)
      .then(data => setReflections(data.data || []))
  }, [traderID])

  return (
    // 使用 reflections 数据
  )
}
```

---

## 🔌 集成步骤

### 快速集成（5分钟）

1. **创建反思服务文件**
   ```bash
   touch web/src/services/reflectionApi.ts
   ```
   复制上面的 API 服务代码

2. **创建反思组件**
   ```bash
   touch web/src/components/ReflectionWidget.tsx
   ```
   复制上面的小组件代码

3. **在仪表板中导入和使用**
   ```typescript
   import { ReflectionWidget } from '@/components/ReflectionWidget'
   
   // 在你的仪表板中添加
   <ReflectionWidget traderID={traderID} />
   ```

4. **启动前端**
   ```bash
   cd web
   npm run dev
   ```

---

## 📊 数据展示示例

### 反思卡片样式
```
┌─────────────────────────────────────┐
│ 📊 性能分析                          │
│                                     │
│ Win rate improved by 5% in last     │
│ 7 days. Current rate: 52%           │
│                                     │
│ [应用调整] [查看详情]              │
│ 2025-01-12 10:00 AM               │
└─────────────────────────────────────┘
```

### 调整建议样式
```
┌─────────────────────────────────────┐
│ ⚡ 增加头寸大小              [高优先] │
│                                     │
│ 原因：胜率趋势向上，考虑增加         │
│       头寸规模以获得更高回报        │
│                                     │
│ [✓ 应用]  [✗ 拒绝]                │
└─────────────────────────────────────┘
```

---

## 🎨 UI 配置

### Tailwind CSS 样式类

```typescript
// 反思严重程度样式
const severityStyles = {
  info: 'bg-blue-100 text-blue-800 border-blue-300',
  warning: 'bg-yellow-100 text-yellow-800 border-yellow-300',
  error: 'bg-red-100 text-red-800 border-red-300'
}

// 优先级样式
const priorityStyles = {
  low: 'bg-green-100 text-green-800',
  medium: 'bg-yellow-100 text-yellow-800',
  high: 'bg-red-100 text-red-800'
}

// 状态样式
const statusStyles = {
  pending: 'bg-gray-100 text-gray-800',
  applied: 'bg-green-100 text-green-800',
  rejected: 'bg-red-100 text-red-800'
}
```

---

## 🔄 实时更新

### 使用 WebSocket 获取实时反思更新

```typescript
export function useReflectionUpdates(traderID: string) {
  const [reflections, setReflections] = useState<Reflection[]>([])

  useEffect(() => {
    // 初始加载
    fetch(`/api/reflection/${traderID}/recent`)
      .then(r => r.json())
      .then(data => setReflections(data.data || []))

    // WebSocket 连接（如果后端支持）
    const ws = new WebSocket(`wss://api.example.com/ws/reflection/${traderID}`)
    
    ws.onmessage = (event) => {
      const newReflection = JSON.parse(event.data)
      setReflections(prev => [newReflection, ...prev])
    }

    return () => ws.close()
  }, [traderID])

  return reflections
}
```

---

## 🧪 测试端点

### 使用 curl 测试

```bash
# 获取最近反思
curl http://localhost:8080/api/reflection/trader_001/recent?limit=10

# 获取反思统计
curl http://localhost:8080/api/reflection/trader_001/stats

# 获取待处理调整
curl http://localhost:8080/api/adjustment/trader_001/pending

# 触发分析
curl -X POST http://localhost:8080/api/reflection/trader_001/analyze \
  -H "Content-Type: application/json" \
  -d '{"analysisType":"performance"}'

# 应用调整
curl -X POST http://localhost:8080/api/adjustment/adj_123/apply
```

---

## 📚 完整示例项目结构

```
web/src/
├── services/
│   └── reflectionApi.ts          (API 服务)
├── components/
│   ├── ReflectionDashboard.tsx   (完整页面)
│   ├── ReflectionWidget.tsx      (小组件)
│   └── ReflectionCard.tsx        (单个反思卡片)
├── pages/
│   └── ReflectionPage.tsx        (页面组件)
├── hooks/
│   └── useReflectionData.ts      (自定义 hook)
└── App.tsx                        (路由配置)
```

---

## 🚨 常见问题

### Q: API 返回 404
**A:** 确保后端已启动，反思系统已初始化。检查日志：
```
✅ Reflection routes registered
✅ Reflection system initialized successfully
```

### Q: 数据无法加载
**A:** 检查浏览器控制台的网络错误，确保：
- API 端点正确
- CORS 已配置
- traderID 有效

### Q: 调整无法应用
**A:** 确保：
- adjustmentID 正确
- 用户有权限
- 后端反思引擎正在运行

---

## 📞 支持

- API 文档: 查看 `api/reflection_handlers.go`
- 后端状态: 检查应用日志
- 数据库: `store/analysis.go`

---

**状态**: ✅ 反思系统已就绪
**版本**: v1.0.0
**更新**: 2025-01-12

