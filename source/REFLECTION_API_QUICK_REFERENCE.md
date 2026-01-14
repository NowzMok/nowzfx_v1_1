# 反思系统 API 快速参考

## 🔗 所有 API 端点一览表

| 功能 | HTTP 方法 | 端点 | 参数 | 用途 |
|------|----------|------|------|------|
| 获取最近反思 | GET | `/api/reflection/{traderID}/recent` | `limit=10` | 查看最近的系统分析 |
| 触发反思分析 | POST | `/api/reflection/{traderID}/analyze` | `analysisType` | 立即进行分析 |
| 获取反思统计 | GET | `/api/reflection/{traderID}/stats` | - | 查看反思指标 |
| 获取反思详情 | GET | `/api/reflection/id/{reflectionID}` | - | 查看单个反思详情 |
| 获取待调整 | GET | `/api/adjustment/{traderID}/pending` | - | 查看待处理建议 |
| 获取调整历史 | GET | `/api/adjustment/{traderID}/history` | - | 查看历史调整 |
| 应用调整 | POST | `/api/adjustment/{adjustmentID}/apply` | - | 接受建议 |
| 拒绝调整 | POST | `/api/adjustment/{adjustmentID}/reject` | - | 拒绝建议 |
| 恢复调整 | POST | `/api/adjustment/{adjustmentID}/revert` | - | 撤销已应用调整 |
| 获取学习记忆 | GET | `/api/memory/{traderID}` | - | 查看系统学习内容 |
| 删除记忆 | DELETE | `/api/memory/{memoryID}` | - | 删除单个记忆 |

---

## 📋 端点详情

### 1. 获取最近反思

**请求:**
```http
GET /api/reflection/{traderID}/recent?limit=20
```

**响应:**
```json
{
  "data": [
    {
      "id": "ref_abc123",
      "traderID": "trader_001",
      "analysisType": "performance",
      "findings": "Win rate has improved by 5%",
      "timestamp": "2025-01-12T10:30:00Z",
      "severity": "info"
    }
  ],
  "count": 1
}
```

**状态码:** `200` 成功 | `500` 服务器错误

---

### 2. 触发反思分析

**请求:**
```http
POST /api/reflection/{traderID}/analyze
Content-Type: application/json

{
  "analysisType": "performance|risk|strategy",
  "timeRange": {
    "startTime": "2025-01-01T00:00:00Z",
    "endTime": "2025-01-12T23:59:59Z"
  }
}
```

**响应:**
```json
{
  "status": "queued",
  "reflectionID": "ref_xyz789",
  "message": "Analysis queued successfully"
}
```

---

### 3. 获取反思统计

**请求:**
```http
GET /api/reflection/{traderID}/stats
```

**响应:**
```json
{
  "totalReflections": 45,
  "averageSeverity": "low",
  "lastReflectionTime": "2025-01-12T10:30:00Z",
  "findingsByType": {
    "performance": 25,
    "risk": 15,
    "strategy": 5
  }
}
```

---

### 4. 获取待处理调整

**请求:**
```http
GET /api/adjustment/{traderID}/pending
```

**响应:**
```json
{
  "data": [
    {
      "id": "adj_def456",
      "traderID": "trader_001",
      "suggestedAction": "increase_position_size",
      "reasoning": "Win rate trending up, safe to increase",
      "priority": "high",
      "status": "pending",
      "createdAt": "2025-01-12T09:00:00Z",
      "reflectionID": "ref_abc123"
    }
  ],
  "count": 1
}
```

---

### 5. 应用调整

**请求:**
```http
POST /api/adjustment/{adjustmentID}/apply
```

**响应:**
```json
{
  "status": "applied",
  "adjustmentID": "adj_def456",
  "appliedAt": "2025-01-12T10:30:00Z"
}
```

---

### 6. 获取学习记忆

**请求:**
```http
GET /api/memory/{traderID}
```

**响应:**
```json
{
  "data": [
    {
      "id": "mem_ghi789",
      "traderID": "trader_001",
      "lessonLearned": "High volatility periods reduce win rate",
      "timestamp": "2025-01-10T14:00:00Z",
      "confidence": 0.87
    }
  ],
  "count": 1
}
```

---

## 💻 前端集成代码片段

### React Hook 获取反思

```typescript
import { useEffect, useState } from 'react'

export function useReflections(traderID: string) {
  const [data, setData] = useState(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState(null)

  useEffect(() => {
    fetch(`/api/reflection/${traderID}/recent?limit=20`)
      .then(r => r.json())
      .then(d => {
        setData(d.data)
        setError(null)
      })
      .catch(e => setError(e))
      .finally(() => setLoading(false))
  }, [traderID])

  return { data, loading, error }
}

// 使用方式
function MyComponent() {
  const { data: reflections } = useReflections('trader_001')
  return (
    <div>
      {reflections?.map(r => (
        <div key={r.id}>{r.findings}</div>
      ))}
    </div>
  )
}
```

---

### 触发分析

```typescript
async function triggerAnalysis(traderID: string) {
  const response = await fetch(`/api/reflection/${traderID}/analyze`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({
      analysisType: 'performance'
    })
  })
  return response.json()
}

// 使用
<button onClick={() => triggerAnalysis('trader_001')}>
  分析现在
</button>
```

---

### 应用/拒绝调整

```typescript
async function applyAdjustment(adjustmentID: string) {
  const response = await fetch(`/api/adjustment/${adjustmentID}/apply`, {
    method: 'POST'
  })
  return response.json()
}

async function rejectAdjustment(adjustmentID: string) {
  const response = await fetch(`/api/adjustment/${adjustmentID}/reject`, {
    method: 'POST'
  })
  return response.json()
}

// 使用
<div>
  <button onClick={() => applyAdjustment(adj.id)} className="btn-success">
    应用
  </button>
  <button onClick={() => rejectAdjustment(adj.id)} className="btn-danger">
    拒绝
  </button>
</div>
```

---

## 🎨 UI 组件集合

### 反思卡片组件

```typescript
interface ReflectionCardProps {
  finding: string
  type: 'performance' | 'risk' | 'strategy'
  severity: 'info' | 'warning' | 'error'
  timestamp: string
}

export function ReflectionCard({
  finding,
  type,
  severity,
  timestamp
}: ReflectionCardProps) {
  const icon = type === 'performance' ? '📊' : type === 'risk' ? '⚠️' : '🎯'
  const bgColor = severity === 'error' ? 'bg-red-50' : severity === 'warning' ? 'bg-yellow-50' : 'bg-blue-50'
  
  return (
    <div className={`${bgColor} p-4 rounded-lg border-l-4`}>
      <div className="flex items-start gap-3">
        <span className="text-2xl">{icon}</span>
        <div className="flex-1">
          <p className="font-semibold text-gray-900">{finding}</p>
          <time className="text-xs text-gray-500">
            {new Date(timestamp).toLocaleString()}
          </time>
        </div>
      </div>
    </div>
  )
}
```

### 调整建议卡片组件

```typescript
interface AdjustmentCardProps {
  action: string
  reasoning: string
  priority: 'low' | 'medium' | 'high'
  onApply: () => void
  onReject: () => void
}

export function AdjustmentCard({
  action,
  reasoning,
  priority,
  onApply,
  onReject
}: AdjustmentCardProps) {
  const priorityColor = {
    low: 'bg-green-100 text-green-800',
    medium: 'bg-yellow-100 text-yellow-800',
    high: 'bg-red-100 text-red-800'
  }[priority]

  return (
    <div className="bg-white p-4 rounded-lg border border-gray-200">
      <div className="flex items-start justify-between mb-3">
        <h3 className="font-semibold text-gray-900">
          {action.replace(/_/g, ' ')}
        </h3>
        <span className={`px-2 py-1 rounded text-xs font-semibold ${priorityColor}`}>
          {priority.toUpperCase()}
        </span>
      </div>
      <p className="text-gray-600 text-sm mb-4">{reasoning}</p>
      <div className="flex gap-2">
        <button
          onClick={onApply}
          className="flex-1 bg-green-600 text-white py-2 rounded hover:bg-green-700"
        >
          ✓ 应用
        </button>
        <button
          onClick={onReject}
          className="flex-1 bg-gray-400 text-white py-2 rounded hover:bg-gray-500"
        >
          ✗ 拒绝
        </button>
      </div>
    </div>
  )
}
```

---

## 🔒 错误处理

### 常见错误响应

```typescript
// 400 - 无效请求
{
  "error": "Invalid trader ID"
}

// 401 - 未授权
{
  "error": "Authentication required"
}

// 404 - 不存在
{
  "error": "Reflection not found"
}

// 500 - 服务器错误
{
  "error": "Internal server error"
}
```

### 错误处理示例

```typescript
const fetchReflections = async (traderID: string) => {
  try {
    const response = await fetch(`/api/reflection/${traderID}/recent`)
    
    if (!response.ok) {
      throw new Error(`HTTP error! status: ${response.status}`)
    }
    
    const data = await response.json()
    return data.data
  } catch (error) {
    console.error('Failed to fetch reflections:', error)
    return []
  }
}
```

---

## 🧪 测试检查清单

- [ ] 可以获取反思列表
- [ ] 可以查看反思详情
- [ ] 可以查看反思统计
- [ ] 可以触发新的分析
- [ ] 可以查看待处理调整
- [ ] 可以应用调整
- [ ] 可以拒绝调整
- [ ] 可以查看学习记忆
- [ ] 可以删除学习记忆
- [ ] 错误处理正确

---

## 📊 数据字段说明

| 字段 | 类型 | 说明 |
|------|------|------|
| `id` | string | 唯一标识符 |
| `traderID` | string | 交易者 ID |
| `analysisType` | string | 分析类型 (performance/risk/strategy) |
| `findings` | string | 发现的内容 |
| `timestamp` | ISO 8601 | 时间戳 |
| `severity` | string | 严重程度 (info/warning/error) |
| `suggestedAction` | string | 建议的行动 |
| `priority` | string | 优先级 (low/medium/high) |
| `status` | string | 状态 (pending/applied/rejected) |
| `reasoning` | string | 理由说明 |
| `lessonLearned` | string | 学到的知识 |
| `confidence` | float | 置信度 (0-1) |

---

## 🚀 快速开始

### 1. 最小化示例

```typescript
// 获取和显示反思
const [reflections, setReflections] = useState([])

useEffect(() => {
  fetch('/api/reflection/trader_001/recent')
    .then(r => r.json())
    .then(d => setReflections(d.data || []))
}, [])

return (
  <div>
    {reflections.map(r => (
      <div key={r.id}>{r.findings}</div>
    ))}
  </div>
)
```

### 2. 完整示例

参考本文件开头的完整集成指南

### 3. 高级示例

- 自定义 Hook 管理状态
- Redux/Zustand 存储管理
- WebSocket 实时更新
- 缓存策略优化

---

**最后更新:** 2025-01-12
**API 版本:** v1.0.0
**状态:** ✅ 就绪

