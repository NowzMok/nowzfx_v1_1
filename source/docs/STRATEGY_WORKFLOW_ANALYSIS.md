# NOFX 项目工作流分析与优化方案

## 📊 项目概览

**NOFX** 是一个 AI 驱动的多资产交易平台，支持加密货币、股票、外汇和贵金属交易。

### 核心技术栈
- **后端**: Go 1.21+（Gin 框架）
- **前端**: React 18+ TypeScript（Vite 构建）
- **数据库**: SQLite / PostgreSQL（GORM）
- **交易所**: Binance、Bybit、OKX、Bitget、Hyperliquid 等
- **AI 模型**: DeepSeek、GPT、Claude、Gemini、Qwen、Grok、Kimi

---

## 🔄 策略编辑工作流程

### 工作流架构图

```
前端 (StrategyStudioPage) 
    ↓
API 请求层 (api/strategy.go)
    ↓
业务逻辑层 (validateStrategyConfig)
    ↓
数据存储层 (store/strategy.go)
    ↓
数据库 (SQLite/PostgreSQL)
```

### 完整流程步骤

#### 1️⃣ **策略列表获取** (`GET /api/strategies`)

**前端文件**: [web/src/pages/StrategyStudioPage.tsx](web/src/pages/StrategyStudioPage.tsx#L192-L210)

```tsx
// 获取用户策略列表
const fetchStrategies = async () => {
  const response = await fetch(`${API_BASE}/api/strategies`, {
    headers: { Authorization: `Bearer ${token}` },
  })
  const data = await response.json()
  // setStrategies(data.strategies)
}
```

**后端处理**: [api/strategy.go](api/strategy.go#L64-L98)

```go
// 获取策略列表
func (s *Server) handleGetStrategies(c *gin.Context) {
  userID := c.GetString("user_id")
  strategies, err := s.store.Strategy().List(userID)
  // 转换为前端格式并返回
}
```

#### 2️⃣ **策略创建** (`POST /api/strategies`)

**前端调用**:
- 点击"新建"按钮 → 显示创建对话框
- 输入策略名称和描述 → 使用默认配置
- 发送 POST 请求

**后端处理**: [api/strategy.go](api/strategy.go#L140-L183)

```go
func (s *Server) handleCreateStrategy(c *gin.Context) {
  // 1. 解析请求参数
  // 2. 序列化配置为 JSON
  // 3. 创建 Strategy 记录
  // 4. 验证配置并返回警告信息
}
```

**数据流**:
```
请求体:
{
  "name": "my_strategy",
  "description": "...",
  "config": {
    "coin_source": { ... },
    "indicators": { ... },
    "risk_control": { ... },
    "prompt_sections": { ... }
  }
}

↓ 验证

响应:
{
  "id": "uuid",
  "message": "Strategy created successfully",
  "warnings": [...]
}
```

#### 3️⃣ **策略编辑** (`PUT /api/strategies/:id`)

**前端编辑器** ([web/src/pages/StrategyStudioPage.tsx](web/src/pages/StrategyStudioPage.tsx#L532-L635)):

编辑器包含 5 个主要部分：

1. **币种来源编辑** (`CoinSourceEditor`)
   - 来源类型: static / ai500 / oi_top / mixed
   - 静态币列表 / 排除列表
   - AI500 和 OI Top 限制数

2. **技术指标编辑** (`IndicatorEditor`)
   - K 线配置（周期、数量）
   - 技术指标开关 (EMA, MACD, RSI, ATR, BOLL, Volume)
   - NofxOS API 数据源配置
   - 量化数据开关

3. **风控参数编辑** (`RiskControlEditor`)
   - 单笔风险 / 日总风险
   - 仓位管理
   - 止盈止损设置

4. **Prompt 编辑** (`PromptSectionsEditor`)
   - 角色定义
   - 交易频率认知
   - 入场标准
   - 决策过程

5. **发布设置** (`PublishSettingsEditor`)
   - 是否公开
   - 配置可见性

**保存流程**: [api/strategy.go](api/strategy.go#L185-L241)

```go
func (s *Server) handleUpdateStrategy(c *gin.Context) {
  // 1. 验证用户权限
  // 2. 检查系统默认策略（不可修改）
  // 3. 序列化新配置
  // 4. 更新数据库记录
  // 5. 返回验证警告
}
```

#### 4️⃣ **策略激活** (`POST /api/strategies/:id/activate`)

```go
// 激活特定策略，使其成为当前交易使用的策略
func (s *Server) handleActivateStrategy(c *gin.Context) {
  // 设置 is_active = true
}
```

#### 5️⃣ **策略导入/导出**

**导出**: 下载为 JSON 文件
```tsx
const handleExportStrategy = () => {
  const exportData = {
    name: strategy.name,
    description: strategy.description,
    config: strategy.config,
    exported_at: new Date().toISOString(),
    version: '1.0',
  }
  // 创建 Blob 并触发下载
}
```

**导入**: 读取 JSON 文件并创建新策略
```tsx
const handleImportStrategy = async (file) => {
  // 1. 读取文件内容
  // 2. 解析 JSON
  // 3. 调用创建 API
  // 4. 刷新策略列表
}
```

---

## 📁 关键文件结构

### 后端结构
```
api/
├── strategy.go              # 策略 API 端点 (643 行)
│   ├── handleGetStrategies         # 列表
│   ├── handleCreateStrategy        # 创建
│   ├── handleUpdateStrategy        # 更新
│   ├── handleDeleteStrategy        # 删除
│   ├── handleActivateStrategy      # 激活
│   ├── handleDuplicateStrategy     # 复制
│   └── validateStrategyConfig()    # 配置验证
├── server.go                # API 服务器配置
└── ...

store/
├── strategy.go              # 数据库操作 (461 行)
│   ├── Strategy struct
│   ├── StrategyConfig struct
│   ├── CoinSourceConfig struct
│   ├── IndicatorConfig struct
│   ├── RiskControlConfig struct
│   └── PromptSectionsConfig struct
└── ...
```

### 前端结构
```
web/src/
├── pages/
│   └── StrategyStudioPage.tsx       # 主策略编辑页面 (1000+ 行)
│       ├── 策略列表管理
│       ├── 配置编辑器集成
│       ├── Prompt 预览
│       └── AI 测试运行

├── components/strategy/
│   ├── CoinSourceEditor.tsx         # 币种来源编辑器
│   ├── IndicatorEditor.tsx          # 技术指标编辑器
│   ├── RiskControlEditor.tsx        # 风控编辑器
│   ├── PromptSectionsEditor.tsx     # Prompt 编辑器
│   └── PublishSettingsEditor.tsx    # 发布设置编辑器

└── types.ts                         # TypeScript 类型定义
    └── Strategy, StrategyConfig 等
```

---

## 🐛 当前存在的问题

### 1. **StrategyStudioPage 代码臃肿**
- 单文件超过 1000 行
- 状态管理混乱（多个 useState）
- 编辑逻辑和 UI 混合

### 2. **配置验证不够完整**
- `validateStrategyConfig()` 只检查 NofxOS API key
- 缺少其他必要字段的验证
- 没有前端预验证

### 3. **错误处理不完善**
- 保存失败后没有回滚机制
- 网络错误时状态不一致
- 缺少详细的错误提示

### 4. **UI/UX 问题**
- 编辑器拥挤，缺少分组逻辑
- 没有草稿保存功能
- 编辑历史追踪缺失

### 5. **性能问题**
- 每次编辑都重新渲染整个配置
- 配置验证在保存时进行（应该实时）
- 没有缓存机制

### 6. **API 设计问题**
- 单个 PUT 请求包含太多数据
- 没有部分更新支持（PATCH）
- 缺少配置预览和验证端点

---

## ✨ 优化方案

### 方案 A: 重构前端架构

#### 1. 拆分 StrategyStudioPage（推荐）

**目标**: 将 1000+ 行的单文件拆分成清晰的组件树

**步骤**:
```
StrategyStudioPage.tsx (主容器)
├── StrategyHeader.tsx           # 头部
├── StrategyList.tsx             # 左侧列表
├── StrategyEditor.tsx           # 中央编辑器（新）
│   ├── BasicInfoEditor.tsx       # 名称 + 描述
│   ├── ConfigSections.tsx        # 折叠的编辑器
│   └── SaveBar.tsx               # 保存条形
├── RightPanel.tsx               # 右侧面板（新）
│   ├── PromptPreviewPanel.tsx    # Prompt 预览
│   └── AITestPanel.tsx           # AI 测试
└── ConfigValidation.tsx          # 验证错误显示
```

**预期代码量**: 每个文件 100-300 行

#### 2. 优化状态管理

**使用 useReducer 或 Zustand**:

```tsx
// 使用 useReducer 替代多个 useState
const [state, dispatch] = useReducer(strategyReducer, initialState)

const strategyReducer = (state, action) => {
  switch (action.type) {
    case 'LOAD_STRATEGY':
      return { ...state, selectedStrategy: action.payload, hasChanges: false }
    case 'UPDATE_CONFIG':
      return { ...state, editingConfig: action.payload, hasChanges: true }
    case 'SAVE_START':
      return { ...state, isSaving: true }
    case 'SAVE_ERROR':
      return { ...state, isSaving: false, error: action.payload }
    // ...
  }
}
```

#### 3. 前端配置验证

```tsx
// 新增实时验证
const validateConfig = useCallback((config: StrategyConfig) => {
  const errors: Record<string, string[]> = {}

  if (!config.coin_source?.source_type) {
    errors.coin_source = ['Source type is required']
  }

  if (config.coin_source?.source_type === 'static' && 
      (!config.coin_source?.static_coins || config.coin_source.static_coins.length === 0)) {
    errors.coin_source = [...(errors.coin_source || []), 'At least one coin required']
  }

  // 实时显示验证结果
  setValidationErrors(errors)
  return Object.keys(errors).length === 0
}, [])

// 在编辑时调用
useEffect(() => {
  const timer = setTimeout(() => validateConfig(editingConfig), 300) // 防抖
  return () => clearTimeout(timer)
}, [editingConfig, validateConfig])
```

#### 4. 草稿保存功能

```tsx
// localStorage 草稿保存
const saveDraft = useCallback(() => {
  localStorage.setItem(
    `strategy_draft_${selectedStrategy?.id}`,
    JSON.stringify(editingConfig),
    Date.now() + 86400000 // 24小时过期
  )
}, [selectedStrategy?.id, editingConfig])

// 定期保存
useEffect(() => {
  const timer = setInterval(() => {
    if (hasChanges && editingConfig) {
      saveDraft()
    }
  }, 30000) // 每 30 秒
  return () => clearInterval(timer)
}, [hasChanges, editingConfig, saveDraft])

// 恢复草稿
const loadDraft = useCallback((strategyId: string) => {
  const draft = localStorage.getItem(`strategy_draft_${strategyId}`)
  if (draft) {
    // 提示用户是否恢复
    confirmDialog('Found unsaved changes. Restore?', {
      onConfirm: () => setEditingConfig(JSON.parse(draft))
    })
  }
}, [])
```

---

### 方案 B: 增强 API 设计

#### 1. 增加分部分更新支持

```go
// 新增 PATCH 端点
router.PATCH("/strategies/:id", s.authMiddleware(), s.handlePatchStrategy)

func (s *Server) handlePatchStrategy(c *gin.Context) {
  strategyID := c.Param("id")
  userID := c.GetString("user_id")

  var patch map[string]interface{}
  if err := c.ShouldBindJSON(&patch); err != nil {
    SafeBadRequest(c, "Invalid request")
    return
  }

  // 只更新提供的字段
  strategy, _ := s.store.Strategy().Get(userID, strategyID)

  // 合并补丁
  if name, ok := patch["name"].(string); ok {
    strategy.Name = name
  }
  if config, ok := patch["config"].(map[string]interface{}); ok {
    // 深层合并配置
    mergeConfig(&strategy, config)
  }

  s.store.Strategy().Update(strategy)
}
```

#### 2. 增加配置验证端点

```go
// POST /api/strategies/validate-config
router.POST("/strategies/validate-config", s.authMiddleware(), s.handleValidateConfig)

func (s *Server) handleValidateConfig(c *gin.Context) {
  var config store.StrategyConfig
  c.ShouldBindJSON(&config)

  warnings := validateStrategyConfig(&config)
  errors := validateStrategyConfigFull(&config) // 新增完整验证

  c.JSON(http.StatusOK, gin.H{
    "valid": len(errors) == 0,
    "errors": errors,
    "warnings": warnings,
  })
}

func validateStrategyConfigFull(config *store.StrategyConfig) map[string][]string {
  errors := make(map[string][]string)

  // 验证币种来源
  if config.CoinSource.SourceType == "" {
    errors["coin_source"] = []string{"Source type is required"}
  }

  // 验证技术指标
  if !config.Indicators.EnableRawKlines && 
     !config.Indicators.EnableEMA && 
     !config.Indicators.EnableMACD {
    errors["indicators"] = []string{"At least one indicator must be enabled"}
  }

  // 验证风控
  if config.RiskControl.SingleTradeLoss <= 0 {
    errors["risk_control"] = []string{"Single trade loss must be greater than 0"}
  }

  return errors
}
```

#### 3. 增加配置预览模板端点

```go
// GET /api/strategies/config-schema
router.GET("/strategies/config-schema", s.handleGetConfigSchema)

func (s *Server) handleGetConfigSchema(c *gin.Context) {
  // 返回配置结构的 JSON Schema
  // 让前端能够动态生成表单
  schema := gin.H{
    "coin_source": gin.H{
      "source_type": gin.H{
        "type": "enum",
        "values": []string{"static", "ai500", "oi_top", "mixed"},
        "description": "Coin source type",
      },
      // ...
    },
  }
  c.JSON(http.StatusOK, schema)
}
```

---

### 方案 C: 改进错误处理和日志

#### 1. 详细的错误信息

```go
type ValidationError struct {
  Field   string `json:"field"`
  Code    string `json:"code"`
  Message string `json:"message"`
  Details string `json:"details,omitempty"`
}

func (s *Server) handleUpdateStrategy(c *gin.Context) {
  // ...

  if err := s.store.Strategy().Update(strategy); err != nil {
    errors := []ValidationError{
      {
        Field:   "config",
        Code:    "INVALID_CONFIG",
        Message: "Strategy configuration is invalid",
        Details: err.Error(),
      },
    }
    c.JSON(http.StatusBadRequest, gin.H{"errors": errors})
    return
  }
}
```

#### 2. 编辑历史和回滚

```go
// 新增字段到 Strategy
type Strategy struct {
  // ...
  EditHistory []StrategyEdit `gorm:"foreignKey:StrategyID" json:"-"`
}

type StrategyEdit struct {
  ID         string    `gorm:"primaryKey"`
  StrategyID string
  UserID     string
  ConfigBefore string // JSON
  ConfigAfter  string // JSON
  Timestamp    time.Time
  Reason       string
}

// 保存编辑历史
func (s *Server) handleUpdateStrategy(c *gin.Context) {
  oldConfig := strategy.Config
  // ... 更新逻辑 ...
  newConfig := strategy.Config

  // 记录历史
  edit := &StrategyEdit{
    StrategyID:   strategy.ID,
    UserID:       userID,
    ConfigBefore: oldConfig,
    ConfigAfter:  newConfig,
    Timestamp:    time.Now(),
  }
  s.store.Strategy().SaveEdit(edit)
}
```

---

### 方案 D: 编辑器增强

#### 1. 编辑器分标签页

```tsx
// StrategyEditor.tsx
<div className="flex border-b">
  <button 
    className={`px-4 py-2 ${activeTab === 'basic' ? 'border-b-2 border-gold' : ''}`}
    onClick={() => setActiveTab('basic')}
  >
    基本信息
  </button>
  <button className={...} onClick={() => setActiveTab('coin')}>
    币种来源
  </button>
  <button className={...} onClick={() => setActiveTab('indicators')}>
    技术指标
  </button>
  <button className={...} onClick={() => setActiveTab('risk')}>
    风控参数
  </button>
  <button className={...} onClick={() => setActiveTab('prompt')}>
    Prompt 设置
  </button>
  <button className={...} onClick={() => setActiveTab('publish')}>
    发布设置
  </button>
</div>

{activeTab === 'basic' && <BasicInfoEditor />}
{activeTab === 'coin' && <CoinSourceEditor />}
{/* ... */}
```

#### 2. 配置对比工具

```tsx
// 对比当前编辑和已保存版本的差异
const ConfigDiffView = ({ current, saved }) => {
  const diff = getDiff(saved, current)
  
  return (
    <div>
      {diff.changed.map(field => (
        <div key={field} className="p-2 border-l-2 border-yellow">
          <div className="text-xs text-gray-500">{field}</div>
          <div className="text-xs">
            <span className="line-through">{saved[field]}</span>
            {' → '}
            <span className="font-bold">{current[field]}</span>
          </div>
        </div>
      ))}
    </div>
  )
}
```

#### 3. 实时预览

```tsx
// 在编辑时实时预览生成的 Prompt
const generatePreviewPrompt = async (config: StrategyConfig) => {
  try {
    const response = await fetch(`${API_BASE}/api/strategies/preview-prompt`, {
      method: 'POST',
      body: JSON.stringify({ config }),
    })
    const data = await response.json()
    setPreviewPrompt(data.prompt)
  } catch (err) {
    setPreviewError(err.message)
  }
}

// 防抖调用
useEffect(() => {
  const timer = setTimeout(
    () => generatePreviewPrompt(editingConfig),
    800 // 延迟 800ms 防止过于频繁
  )
  return () => clearTimeout(timer)
}, [editingConfig])
```

---

## 📊 优化对比表

| 方面 | 当前状态 | 优化后 |
|------|--------|------|
| **代码行数** | 单文件 1000+ 行 | 每个文件 100-300 行 |
| **状态管理** | 多个 useState | 单个 useReducer 或 Zustand |
| **错误处理** | 简单的 toast | 详细的验证错误 + 回滚 |
| **性能** | 每次编辑全局重渲染 | 组件级优化 + 记忆化 |
| **用户体验** | 无草稿保存 | 自动草稿 + 编辑历史 |
| **API** | 只支持 PUT | 支持 PUT、PATCH、VALIDATE |
| **验证** | 保存时验证 | 实时验证 + 前端预验证 |

---

## 🚀 实施步骤（优先级）

### 第一阶段（高优先级 - 1-2 周）
- [ ] 拆分 StrategyStudioPage（组件模块化）
- [ ] 增加前端实时验证
- [ ] 改进错误提示信息

### 第二阶段（中优先级 - 2-3 周）
- [ ] 实现草稿保存功能
- [ ] 增加 PATCH 端点支持
- [ ] 完整配置验证函数

### 第三阶段（低优先级 - 3-4 周）
- [ ] 编辑历史和回滚
- [ ] 配置对比工具
- [ ] JSON Schema 端点

---

## 📝 总结

NOFX 的策略编辑系统是核心功能，当前实现基础但有优化空间。通过以上方案的实施，可以显著提升：

✅ **代码可维护性** - 从单个大文件拆分为模块化组件
✅ **用户体验** - 实时验证、草稿保存、详细错误提示
✅ **系统可靠性** - 完整的验证、错误处理和编辑历史
✅ **开发效率** - 清晰的架构便于新功能扩展

建议优先实施**第一阶段**，这将带来最直接的改进效果。
