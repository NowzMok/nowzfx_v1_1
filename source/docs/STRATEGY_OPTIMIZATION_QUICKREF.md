# NOFX 策略编辑优化 - 快速参考

## 📋 核心问题速览

| 问题 | 当前 | 目标 | 优先级 |
|------|------|------|--------|
| 代码臃肿 | 1000+ 行单文件 | 100-300 行组件 | 🔴 高 |
| 状态混乱 | 10+ 个 useState | 1 个 useReducer | 🔴 高 |
| 验证不足 | 仅检查 API key | 完整字段验证 | 🟡 中 |
| 无草稿 | 关闭丢失编辑 | 自动保存到 localStorage | 🟡 中 |
| 错误处理 | 简单 toast | 详细错误 + 回滚 | 🟡 中 |
| API 设计 | 仅支持 PUT | PUT/PATCH/VALIDATE | 🟢 低 |

---

## 🎯 最高优先级实施项目（第一周）

### 1. 拆分 StrategyStudioPage

**文件**：`web/src/pages/StrategyStudioPage.tsx` (1000+ 行)

**目标**：拆分成 4 个独立文件

```tsx
// 拆分后的结构
web/src/
├── components/
│   ├── strategy/
│   │   ├── StrategyListPanel.tsx    (200 行)
│   │   ├── StrategyEditorPanel.tsx  (300 行)
│   │   └── PreviewPanel.tsx         (150 行)
│   └── hooks/
│       └── useStrategyStore.ts      (100 行)
└── pages/
    └── StrategyStudioPage.tsx       (200 行 - 主容器)
```

**预期效果**：代码清晰，易于维护和测试

### 2. 实现完整的配置验证

**文件**：`web/src/hooks/useConfigValidator.ts`（新增）

```tsx
// 关键功能
✅ 实时验证（防抖 300ms）
✅ 详细错误消息
✅ 多语言支持
✅ 字段级错误显示

// 验证规则
- CoinSource: 来源类型 ✓，静态币列表 ✓
- Indicators: 至少 1 个指标 ✓，K-line 周期有效 ✓
- RiskControl: 风险百分比范围 ✓，日限 > 笔限 ✓
- PromptSections: 长度限制 5000 字 ✓
```

**使用示例**：
```tsx
const { validateConfig } = useConfigValidator(
  editingConfig,
  (errors) => setValidationErrors(errors)
)

// 自动防抖验证
useEffect(() => {
  const timer = setTimeout(() => validateConfig(config), 300)
  return () => clearTimeout(timer)
}, [config, validateConfig])
```

### 3. 自动草稿保存

**文件**：`web/src/hooks/useDraftSave.ts`（新增）

```tsx
// 功能
✅ 每 30 秒自动保存
✅ localStorage 存储
✅ 24 小时自动过期
✅ 用户切换策略时恢复提示

// 实现
localStorage.setItem(
  `strategy_draft_${strategyId}`,
  JSON.stringify({ config, timestamp: Date.now() })
)
```

**用户体验**：
```
编辑策略 → 每 30 秒保存草稿 → 关闭浏览器
↓
重新打开 → 检测到草稿 → "发现未保存更改，是否恢复？"
```

---

## 🔧 技术实施细节

### 状态管理迁移

**旧方式**（10+ 个 useState）：
```tsx
const [strategies, setStrategies] = useState([])
const [selectedStrategy, setSelectedStrategy] = useState(null)
const [editingConfig, setEditingConfig] = useState(null)
const [isSaving, setIsSaving] = useState(false)
const [hasChanges, setHasChanges] = useState(false)
const [validationErrors, setValidationErrors] = useState({})
// ... 更多 ...
```

**新方式**（单个 useReducer）：
```tsx
type StrategyState = {
  strategies: Strategy[]
  selectedStrategyId: string | null
  editingConfig: StrategyConfig | null
  hasChanges: boolean
  validationErrors: Record<string, string[]>
  isSaving: boolean
  // ...
}

const [state, dispatch] = useReducer(strategyReducer, initialState)

// 操作
dispatch({ type: 'UPDATE_CONFIG', payload: newConfig })
dispatch({ type: 'SAVE_START' })
dispatch({ type: 'SET_VALIDATION_ERRORS', payload: errors })
```

**优势**：
- ✅ 状态一致性有保障
- ✅ 易于追踪状态变化
- ✅ 便于日志和调试

---

## 📱 前端代码示例

### 简化后的 StrategyStudioPage（200 行）

```tsx
export function StrategyStudioPage() {
  const { token } = useAuth()
  const { language } = useLanguage()

  // 集中式状态管理
  const store = useStrategyStore()
  
  // 验证和草稿
  const { validateConfig } = useConfigValidator(
    store.editingConfig,
    (errors) => store.setValidationErrors(errors)
  )
  const { saveDraft } = useDraftSave(store.selectedStrategyId, store.editingConfig)

  // API 调用
  const fetchStrategies = async () => {
    const response = await fetch(`${API_BASE}/api/strategies`, {
      headers: { Authorization: `Bearer ${token}` },
    })
    const { strategies } = await response.json()
    store.dispatch({ type: 'LOAD_STRATEGIES', payload: strategies })
  }

  const handleSave = async () => {
    if (!validateConfig(store.editingConfig!)) return
    
    store.dispatch({ type: 'SAVE_START' })
    try {
      const response = await fetch(`${API_BASE}/api/strategies/${store.selectedStrategyId}`, {
        method: 'PUT',
        headers: {
          'Content-Type': 'application/json',
          Authorization: `Bearer ${token}`,
        },
        body: JSON.stringify(store.editingConfig),
      })
      
      store.dispatch({ type: 'SAVE_SUCCESS' })
      saveDraft() // 清除草稿标记
      await fetchStrategies()
    } catch (err) {
      store.dispatch({ type: 'SAVE_ERROR', payload: err.message })
    }
  }

  // 加载
  useEffect(() => {
    fetchStrategies()
  }, [token])

  return (
    <div className="h-full flex flex-col overflow-hidden">
      {/* 三列布局 */}
      <div className="flex-1 flex">
        <StrategyListPanel
          strategies={store.strategies}
          selectedId={store.selectedStrategyId}
          onSelect={store.selectStrategy}
          // ...
        />
        <StrategyEditorPanel
          config={store.editingConfig}
          validationErrors={store.validationErrors}
          onConfigChange={store.updateConfig}
          // ...
        />
        <PreviewPanel
          // ...
        />
      </div>

      {/* 底部保存条 */}
      <div className="border-t p-4 flex justify-between">
        <div>
          {store.hasChanges && <span>⚠️ Unsaved changes</span>}
          {Object.keys(store.validationErrors).length > 0 && (
            <span className="text-red-400">
              {Object.keys(store.validationErrors).length} errors
            </span>
          )}
        </div>
        <button
          onClick={handleSave}
          disabled={!store.hasChanges || store.isSaving}
        >
          {store.isSaving ? 'Saving...' : 'Save'}
        </button>
      </div>
    </div>
  )
}
```

---

## 🔌 后端 API 增强

### 添加验证端点

```bash
# 请求
POST /api/strategies/validate-config
Content-Type: application/json

{
  "coin_source": { "source_type": "static", ... },
  "indicators": { ... },
  "risk_control": { ... },
  "prompt_sections": { ... }
}

# 响应
{
  "valid": true,
  "errors": [],
  "warnings": [
    {
      "field": "indicators",
      "code": "INDICATOR_WARNING",
      "message": "EMA enabled but no periods specified"
    }
  ]
}
```

### 支持 PATCH 请求（部分更新）

```bash
# 只更新名称
PATCH /api/strategies/:id
Content-Type: application/json

{ "name": "New name" }

# 只更新 coin_source
PATCH /api/strategies/:id
Content-Type: application/json

{ "config": { "coin_source": { ... } } }
```

---

## 📊 测试清单

### 前端测试

- [ ] **状态管理**
  - [ ] 加载策略列表
  - [ ] 选择和切换策略
  - [ ] 编辑各个字段
  - [ ] 验证错误显示

- [ ] **验证功能**
  - [ ] 实时验证工作
  - [ ] 错误消息准确
  - [ ] 防抖正常工作

- [ ] **草稿保存**
  - [ ] 自动保存草稿
  - [ ] 刷新页面恢复草稿
  - [ ] 成功保存后清除草稿

- [ ] **UI/UX**
  - [ ] 标签页切换流畅
  - [ ] 保存按钮状态正确
  - [ ] 错误提示清晰

### 后端测试

- [ ] **API 端点**
  - [ ] GET /strategies (列表)
  - [ ] GET /strategies/:id (详情)
  - [ ] POST /strategies (创建)
  - [ ] PUT /strategies/:id (完整更新)
  - [ ] PATCH /strategies/:id (部分更新)
  - [ ] DELETE /strategies/:id (删除)

- [ ] **验证**
  - [ ] POST /strategies/validate-config
  - [ ] 验证错误准确
  - [ ] 验证警告有用

- [ ] **错误处理**
  - [ ] 400 Bad Request
  - [ ] 401 Unauthorized
  - [ ] 403 Forbidden（默认策略）
  - [ ] 404 Not Found
  - [ ] 500 Server Error

---

## 🚀 实施时间表

### 第一周（优先级 🔴 高）

| 任务 | 工时 | 成员 |
|------|------|------|
| 1. 拆分 StrategyStudioPage | 1 天 | FE |
| 2. 实现 useStrategyStore | 0.5 天 | FE |
| 3. 实现 useConfigValidator | 0.5 天 | FE |
| 4. 完整配置验证函数 | 1 天 | BE |
| 测试 + 修复 | 1 天 | QA |
| **小计** | **4 天** | |

### 第二周（优先级 🟡 中）

| 任务 | 工时 | 成员 |
|------|------|------|
| 1. 实现草稿保存 | 0.5 天 | FE |
| 2. 添加 PATCH 端点 | 0.5 天 | BE |
| 3. 验证端点实现 | 0.5 天 | BE |
| 4. 错误处理改进 | 1 天 | FE+BE |
| 测试 + 修复 | 1 天 | QA |
| **小计** | **3.5 天** | |

### 第三周（优先级 🟢 低）

| 任务 | 工时 | 成员 |
|------|------|------|
| 1. 编辑历史快照 | 1.5 天 | BE |
| 2. 配置对比工具 | 1 天 | FE |
| 3. 性能优化（缓存） | 1 天 | BE |
| 4. 文档更新 | 0.5 天 | Tech Writer |
| **小计** | **4 天** | |

---

## 📚 相关文件参考

### 需要修改的文件

| 文件 | 行数 | 类型 | 优先级 |
|------|------|------|--------|
| `web/src/pages/StrategyStudioPage.tsx` | 1000+ | Refactor | 🔴 |
| `api/strategy.go` | 643 | Enhancement | 🟡 |
| `web/src/components/strategy/*.tsx` | 各200-300 | Update | 🟡 |
| `store/strategy.go` | 461 | Schema | 🟢 |

### 需要创建的文件

| 文件 | 用途 | 优先级 |
|------|------|--------|
| `web/src/hooks/useStrategyStore.ts` | 状态管理 | 🔴 |
| `web/src/hooks/useConfigValidator.ts` | 验证 Hook | 🔴 |
| `web/src/hooks/useDraftSave.ts` | 草稿保存 | 🟡 |
| `web/src/components/strategy/StrategyListPanel.tsx` | UI 组件 | 🔴 |
| `api/strategy_validation.go` | API 验证 | 🟡 |

---

## 🎓 学习资源

### React 最佳实践

- useReducer vs useState: [React Doc](https://react.dev/reference/react/useReducer)
- 自定义 Hook: [Custom Hooks](https://react.dev/learn/reusing-logic-with-custom-hooks)
- 性能优化: [Memoization](https://react.dev/reference/react/useMemo)

### Go API 最佳实践

- RESTful API 设计: [REST Best Practices](https://restfulapi.net/)
- 错误处理: [Gin Error Handling](https://gin-gonic.com/docs/examples/custom-http-config/)
- 验证框架: [Validator Package](https://github.com/go-playground/validator)

---

## 💡 快速查找

### 找不到某个概念？

- **状态管理问题** → 查看 `STRATEGY_REFACTOR_EXAMPLE.tsx` 的第 1 部分
- **验证实现** → 查看 `STRATEGY_API_OPTIMIZATION.go` 的第 1 部分
- **草稿保存** → 查看 `STRATEGY_REFACTOR_EXAMPLE.tsx` 的第 5 部分
- **完整项目流程** → 查看 `STRATEGY_WORKFLOW_ANALYSIS.md`

---

## ✅ 完成度检查表

- [ ] 阅读了 `STRATEGY_WORKFLOW_ANALYSIS.md`
- [ ] 理解了当前工作流程
- [ ] 理解了优化的必要性
- [ ] 查看了重构示例代码
- [ ] 查看了 API 优化方案
- [ ] 制定了实施计划
- [ ] 分配了开发人员

**下一步**：按照时间表开始实施第一周任务！🚀
