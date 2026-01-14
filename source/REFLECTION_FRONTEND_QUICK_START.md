# 前端查看反思系统 - 快速使用指南

## 🚀 3 分钟快速上手

### 1️⃣ 复制前端组件

我已经为你创建了完整的反思系统前端组件：

**文件位置**: `web/src/components/ReflectionDashboard.tsx`

这个组件包含：
- ✅ 反思记录列表
- ✅ 待处理调整建议
- ✅ 统计信息仪表板
- ✅ 一键触发分析
- ✅ 应用/拒绝建议的功能

### 2️⃣ 在你的页面中使用它

```typescript
// 在你的交易者页面或仪表板中导入
import ReflectionDashboard from '@/components/ReflectionDashboard'

export function TraderPage({ traderID }: { traderID: string }) {
  return (
    <div>
      {/* 其他内容... */}
      
      {/* 添加反思系统仪表板 */}
      <ReflectionDashboard 
        traderID={traderID}
        autoRefresh={true}
        refreshInterval={60000}
      />
    </div>
  )
}
```

### 3️⃣ 完成！

应用现在可以显示：
- 📊 **反思记录** - 系统自动分析的结果
- ⚡ **调整建议** - AI 建议的改进
- 📈 **统计数据** - 分析汇总信息

---

## 📡 API 端点速查

| 功能 | API | 说明 |
|------|-----|------|
| 获取最近反思 | `GET /api/reflection/{traderID}/recent` | 查看最近分析结果 |
| 获取统计信息 | `GET /api/reflection/{traderID}/stats` | 查看反思汇总 |
| 获取调整建议 | `GET /api/adjustment/{traderID}/pending` | 查看待处理建议 |
| 应用建议 | `POST /api/adjustment/{id}/apply` | 接受建议 |
| 拒绝建议 | `POST /api/adjustment/{id}/reject` | 拒绝建议 |
| 触发分析 | `POST /api/reflection/{traderID}/analyze` | 立即分析 |

---

## 🎨 组件属性

```typescript
interface ReflectionDashboardProps {
  traderID: string           // 交易者 ID（必需）
  autoRefresh?: boolean      // 自动刷新（默认：true）
  refreshInterval?: number   // 刷新间隔（毫秒，默认：60000）
}
```

### 使用示例

```typescript
// 基本用法
<ReflectionDashboard traderID="trader_001" />

// 禁用自动刷新
<ReflectionDashboard 
  traderID="trader_001" 
  autoRefresh={false} 
/>

// 自定义刷新间隔（每 30 秒）
<ReflectionDashboard 
  traderID="trader_001" 
  autoRefresh={true}
  refreshInterval={30000}
/>
```

---

## 📊 显示效果

### 统计卡片
```
┌──────────────────────────────────────────────────┐
│ 📊 系统反思                            🔄 刷新   │
├──────────────────────────────────────────────────┤
│
│ ┌────────────┐ ┌────────────┐ ┌────────────┐ ┌────────────┐
│ │ 总反思次数 │ │ 性能分析   │ │ 风险分析   │ │ 策略分析   │
│ │    45      │ │    25      │ │    15      │ │    5       │
│ └────────────┘ └────────────┘ └────────────┘ └────────────┘
│
├──────────────────────────────────────────────────┤
│ 触发分析                                          │
│ [📊 性能分析] [⚠️ 风险分析] [🎯 策略分析]      │
└──────────────────────────────────────────────────┘
```

### 反思记录
```
┌──────────────────────────────────────────────────┐
│ 📊 性能分析                              [INFO]   │
│                                                  │
│ Win rate improved by 5% in last 7 days.         │
│ Current rate: 52%                               │
│                                                  │
│ 2025-01-12 10:30 AM                           │
└──────────────────────────────────────────────────┘
```

### 调整建议
```
┌──────────────────────────────────────────────────┐
│ INCREASE_POSITION_SIZE                  [HIGH]  │
│                                                  │
│ Win rate trending up, safe to increase position  │
│ size for higher potential returns.              │
│                                                  │
│ 建议时间: 2025-01-12 09:00 AM                 │
│                                                  │
│ [✓ 应用建议]              [✗ 拒绝]           │
└──────────────────────────────────────────────────┘
```

---

## 🔌 集成到现有项目

### 选项 1: 作为独立页面

```typescript
// web/src/pages/ReflectionPage.tsx
import ReflectionDashboard from '@/components/ReflectionDashboard'

export function ReflectionPage() {
  const { traderID } = useParams()
  return <ReflectionDashboard traderID={traderID || ''} />
}

// 在 App.tsx 中添加路由
<Route path="/traders/:traderID/reflection" element={<ReflectionPage />} />
```

### 选项 2: 作为仪表板小部件

```typescript
// 在 TraderDashboardPage.tsx 中
import ReflectionDashboard from '@/components/ReflectionDashboard'

export function TraderDashboardPage() {
  const { traderID } = useParams()
  
  return (
    <div className="grid grid-cols-3 gap-6">
      {/* 其他小部件 */}
      
      {/* 反思仪表板占据第 3 列 */}
      <div className="col-span-1">
        <ReflectionDashboard traderID={traderID || ''} />
      </div>
    </div>
  )
}
```

### 选项 3: 作为模态对话框

```typescript
import { useState } from 'react'
import ReflectionDashboard from '@/components/ReflectionDashboard'

export function TraderView() {
  const [showReflection, setShowReflection] = useState(false)
  
  return (
    <>
      <button onClick={() => setShowReflection(true)}>
        查看反思
      </button>
      
      {showReflection && (
        <div className="fixed inset-0 bg-black/50 flex items-center justify-center">
          <div className="bg-white rounded-lg max-w-4xl w-full max-h-96 overflow-y-auto">
            <ReflectionDashboard traderID={traderID} />
            <button 
              onClick={() => setShowReflection(false)}
              className="mt-4 px-4 py-2 bg-gray-400 text-white rounded"
            >
              关闭
            </button>
          </div>
        </div>
      )}
    </>
  )
}
```

---

## 🛠️ 自定义样式

### 使用 Tailwind CSS

组件已经使用 Tailwind CSS 样式。如果需要自定义：

```typescript
// 创建一个包装器组件来自定义样式
export function CustomReflectionDashboard(props: ReflectionDashboardProps) {
  return (
    <div className="your-custom-wrapper">
      <ReflectionDashboard {...props} />
    </div>
  )
}
```

### 深色模式支持

组件天然支持 Tailwind 深色模式。在父元素添加 `dark` 类即可：

```typescript
<div className="dark">
  <ReflectionDashboard traderID="trader_001" />
</div>
```

---

## 🔄 数据刷新

### 自动刷新

```typescript
// 每 60 秒自动刷新一次（默认）
<ReflectionDashboard traderID={traderID} autoRefresh={true} />

// 每 30 秒刷新一次
<ReflectionDashboard 
  traderID={traderID} 
  autoRefresh={true}
  refreshInterval={30000}
/>
```

### 手动刷新

组件有"刷新"按钮可以立即更新数据。

---

## 📱 响应式设计

组件完全响应式：
- 📱 手机: 单列布局
- 📊 平板: 两列布局
- 🖥️ 桌面: 四列统计卡片

---

## ⚡ 性能优化

### 分页加载

如果反思记录很多，可以修改组件添加分页：

```typescript
const [page, setPage] = useState(1)
const [limit, setLimit] = useState(30)

const url = `/api/reflection/${traderID}/recent?limit=${limit}&offset=${(page-1)*limit}`
```

### 虚拟滚动

对于非常长的列表，考虑使用虚拟滚动库：
- `react-window`
- `react-virtualized`
- `tanstack/react-virtual`

---

## 🐛 故障排除

### 问题：无法加载数据

**检查清单:**
- [ ] 后端已启动
- [ ] API 端点正确
- [ ] traderID 有效
- [ ] 网络连接正常

### 问题：样式不显示

**解决方法:**
- 确保 Tailwind CSS 已正确配置
- 检查 `tailwind.config.js` 包含 `src` 目录

### 问题：自动刷新不工作

**检查:**
```typescript
// 确保属性正确传递
<ReflectionDashboard 
  traderID={traderID}
  autoRefresh={true}
  refreshInterval={60000}
/>
```

---

## 📚 完整代码示例

### 最小化集成

```typescript
import ReflectionDashboard from '@/components/ReflectionDashboard'

function App() {
  return (
    <div>
      <h1>我的交易者</h1>
      <ReflectionDashboard traderID="trader_001" />
    </div>
  )
}

export default App
```

### 带导航的完整集成

```typescript
import { useState } from 'react'
import ReflectionDashboard from '@/components/ReflectionDashboard'

function App() {
  const [traderID] = useState('trader_001')
  const [activeTab, setActiveTab] = useState('dashboard')

  return (
    <div>
      <nav className="border-b">
        <button 
          onClick={() => setActiveTab('dashboard')}
          className={activeTab === 'dashboard' ? 'font-bold' : ''}
        >
          仪表板
        </button>
        <button 
          onClick={() => setActiveTab('reflection')}
          className={activeTab === 'reflection' ? 'font-bold' : ''}
        >
          反思系统
        </button>
      </nav>

      {activeTab === 'reflection' && (
        <ReflectionDashboard traderID={traderID} />
      )}
    </div>
  )
}

export default App
```

---

## 🎓 学习资源

### 相关文档
- [反思系统完整指南](./REFLECTION_FRONTEND_GUIDE.md)
- [API 快速参考](./REFLECTION_API_QUICK_REFERENCE.md)

### 后端信息
- API 处理器: `api/reflection_handlers.go`
- 数据库模型: `store/analysis.go`
- 调度器: `backtest/reflection_scheduler.go`

---

## ✅ 验证清单

- [ ] 复制 `ReflectionDashboard.tsx` 组件
- [ ] 在项目中导入组件
- [ ] 在需要的地方使用组件
- [ ] 运行前端应用
- [ ] 访问包含组件的页面
- [ ] 看到反思数据加载
- [ ] 测试应用/拒绝建议
- [ ] 测试触发分析
- [ ] 测试自动刷新

---

**状态**: ✅ 完全就绪
**版本**: 1.0.0
**最后更新**: 2025-01-12

