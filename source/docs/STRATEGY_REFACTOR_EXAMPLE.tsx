/**
 * 策略编辑器 - 模块化重构示例
 * 
 * 目标：将 StrategyStudioPage 拆分为清晰的组件
 * 
 * 新架构：
 * StrategyStudioPage (主容器)
 * ├── StrategyListPanel (左)
 * ├── StrategyEditorPanel (中)
 * └── PreviewPanel (右)
 */

// ============================================================================
// 1. 状态管理 (useStrategyStore.ts)
// ============================================================================

import { useReducer, useCallback } from 'react'

interface StrategyState {
  // 策略列表
  strategies: Strategy[]
  selectedStrategyId: string | null
  
  // 编辑状态
  editingConfig: StrategyConfig | null
  hasChanges: boolean
  hasErrors: boolean
  validationErrors: Record<string, string[]>
  
  // UI 状态
  isSaving: boolean
  isLoading: boolean
  error: string | null
  
  // 预览和测试
  promptPreview: string | null
  aiTestResult: any | null
}

const initialState: StrategyState = {
  strategies: [],
  selectedStrategyId: null,
  editingConfig: null,
  hasChanges: false,
  hasErrors: false,
  validationErrors: {},
  isSaving: false,
  isLoading: false,
  error: null,
  promptPreview: null,
  aiTestResult: null,
}

type StrategyAction = 
  | { type: 'LOAD_STRATEGIES'; payload: Strategy[] }
  | { type: 'SELECT_STRATEGY'; payload: Strategy }
  | { type: 'UPDATE_CONFIG'; payload: StrategyConfig }
  | { type: 'SET_VALIDATION_ERRORS'; payload: Record<string, string[]> }
  | { type: 'SAVE_START' }
  | { type: 'SAVE_SUCCESS' }
  | { type: 'SAVE_ERROR'; payload: string }
  | { type: 'SET_PROMPT_PREVIEW'; payload: string }
  | { type: 'CLEAR_CHANGES' }

const strategyReducer = (state: StrategyState, action: StrategyAction): StrategyState => {
  switch (action.type) {
    case 'LOAD_STRATEGIES':
      return { ...state, strategies: action.payload }
    
    case 'SELECT_STRATEGY':
      return {
        ...state,
        selectedStrategyId: action.payload.id,
        editingConfig: action.payload.config,
        hasChanges: false,
        hasErrors: false,
        validationErrors: {},
      }
    
    case 'UPDATE_CONFIG':
      return {
        ...state,
        editingConfig: action.payload,
        hasChanges: true,
      }
    
    case 'SET_VALIDATION_ERRORS':
      return {
        ...state,
        validationErrors: action.payload,
        hasErrors: Object.keys(action.payload).length > 0,
      }
    
    case 'SAVE_START':
      return { ...state, isSaving: true, error: null }
    
    case 'SAVE_SUCCESS':
      return { ...state, isSaving: false, hasChanges: false, error: null }
    
    case 'SAVE_ERROR':
      return { ...state, isSaving: false, error: action.payload }
    
    case 'SET_PROMPT_PREVIEW':
      return { ...state, promptPreview: action.payload }
    
    case 'CLEAR_CHANGES':
      return { ...state, hasChanges: false, validationErrors: {} }
    
    default:
      return state
  }
}

export const useStrategyStore = () => {
  const [state, dispatch] = useReducer(strategyReducer, initialState)
  
  const selectStrategy = useCallback((strategy: Strategy) => {
    dispatch({ type: 'SELECT_STRATEGY', payload: strategy })
  }, [])
  
  const updateConfig = useCallback(<K extends keyof StrategyConfig>(
    key: K,
    value: StrategyConfig[K]
  ) => {
    if (!state.editingConfig) return
    const newConfig = { ...state.editingConfig, [key]: value }
    dispatch({ type: 'UPDATE_CONFIG', payload: newConfig })
  }, [state.editingConfig])
  
  const setValidationErrors = useCallback((errors: Record<string, string[]>) => {
    dispatch({ type: 'SET_VALIDATION_ERRORS', payload: errors })
  }, [])
  
  return {
    // State
    ...state,
    
    // Actions
    selectStrategy,
    updateConfig,
    setValidationErrors,
    dispatch,
  }
}

// ============================================================================
// 2. 配置验证 Hook (useConfigValidator.ts)
// ============================================================================

import { useCallback, useEffect } from 'react'

export const useConfigValidator = (
  config: StrategyConfig | null,
  onErrorsChange: (errors: Record<string, string[]>) => void
) => {
  const validateConfig = useCallback((cfg: StrategyConfig) => {
    const errors: Record<string, string[]> = {}

    // 验证币种来源
    if (!cfg.coin_source?.source_type) {
      errors.coin_source = ['Source type is required']
    } else if (
      cfg.coin_source.source_type === 'static' &&
      (!cfg.coin_source?.static_coins || cfg.coin_source.static_coins.length === 0)
    ) {
      errors.coin_source = ['At least one coin is required']
    }

    // 验证技术指标
    const hasIndicator = 
      cfg.indicators?.enable_ema ||
      cfg.indicators?.enable_macd ||
      cfg.indicators?.enable_rsi ||
      cfg.indicators?.enable_atr ||
      cfg.indicators?.enable_boll ||
      cfg.indicators?.enable_volume ||
      cfg.indicators?.enable_oi ||
      cfg.indicators?.enable_funding_rate

    if (!hasIndicator) {
      errors.indicators = ['At least one indicator must be enabled']
    }

    if (!cfg.indicators?.klines?.primary_timeframe) {
      errors.indicators = [
        ...(errors.indicators || []),
        'Primary timeframe is required',
      ]
    }

    // 验证风控
    if (!cfg.risk_control?.single_trade_loss || cfg.risk_control.single_trade_loss <= 0) {
      errors.risk_control = ['Single trade loss must be greater than 0']
    }

    // NofxOS 检查
    const needsNofxOS =
      cfg.indicators?.enable_quant_data ||
      cfg.indicators?.enable_oi_ranking ||
      cfg.indicators?.enable_netflow_ranking ||
      cfg.indicators?.enable_price_ranking

    if (needsNofxOS && !cfg.indicators?.nofxos_api_key) {
      errors.indicators = [
        ...(errors.indicators || []),
        'NofxOS API key is required for selected features',
      ]
    }

    onErrorsChange(errors)
    return Object.keys(errors).length === 0
  }, [onErrorsChange])

  // 防抖验证
  useEffect(() => {
    if (!config) return

    const timer = setTimeout(() => {
      validateConfig(config)
    }, 300)

    return () => clearTimeout(timer)
  }, [config, validateConfig])

  return { validateConfig }
}

// ============================================================================
// 3. 草稿保存 Hook (useDraftSave.ts)
// ============================================================================

export const useDraftSave = (strategyId: string | null, config: StrategyConfig | null) => {
  const saveDraft = useCallback(() => {
    if (!strategyId || !config) return
    
    const draft = {
      config,
      timestamp: Date.now(),
    }
    
    localStorage.setItem(
      `strategy_draft_${strategyId}`,
      JSON.stringify(draft)
    )
  }, [strategyId, config])

  const loadDraft = useCallback((id: string) => {
    const draft = localStorage.getItem(`strategy_draft_${id}`)
    if (!draft) return null
    
    try {
      return JSON.parse(draft)
    } catch {
      return null
    }
  }, [])

  const clearDraft = useCallback((id: string) => {
    localStorage.removeItem(`strategy_draft_${id}`)
  }, [])

  // 定期保存草稿（每 30 秒）
  useEffect(() => {
    const timer = setInterval(() => {
      if (config) {
        saveDraft()
      }
    }, 30000)

    return () => clearInterval(timer)
  }, [config, saveDraft])

  return { saveDraft, loadDraft, clearDraft }
}

// ============================================================================
// 4. 左侧面板 - 策略列表 (StrategyListPanel.tsx)
// ============================================================================

interface StrategyListPanelProps {
  strategies: Strategy[]
  selectedId: string | null
  isLoading: boolean
  onSelect: (strategy: Strategy) => void
  onNew: () => void
  onDelete: (id: string) => void
  onDuplicate: (id: string) => void
  language: string
}

export const StrategyListPanel = ({
  strategies,
  selectedId,
  isLoading,
  onSelect,
  onNew,
  onDelete,
  onDuplicate,
  language,
}: StrategyListPanelProps) => {
  const t = (key: string) => {
    const translations = {
      strategies: { zh: '策略', en: 'Strategies' },
      new: { zh: '新建', en: 'New' },
      deleteConfirm: { zh: '确定删除？', en: 'Confirm delete?' },
    }
    return translations[key]?.[language] || key
  }

  return (
    <div className="w-48 flex-shrink-0 border-r border-nofx-gold/20 overflow-y-auto">
      {/* Header */}
      <div className="p-3 border-b border-nofx-gold/10">
        <div className="flex items-center justify-between mb-2">
          <h3 className="text-xs font-semibold text-nofx-text-muted uppercase tracking-wider">
            {t('strategies')}
          </h3>
          <button
            onClick={onNew}
            className="p-1 rounded hover:bg-white/10 text-nofx-gold transition-colors"
            title={t('new')}
          >
            {/* Plus icon */}
          </button>
        </div>
      </div>

      {/* List */}
      <div className="space-y-1 p-2">
        {isLoading ? (
          <div className="text-xs text-nofx-text-muted text-center py-4">Loading...</div>
        ) : strategies.length === 0 ? (
          <div className="text-xs text-nofx-text-muted text-center py-4">
            No strategies
          </div>
        ) : (
          strategies.map((strategy) => (
            <div
              key={strategy.id}
              className={`p-2 rounded transition-colors cursor-pointer group ${
                selectedId === strategy.id
                  ? 'bg-nofx-gold/20 border-l-2 border-nofx-gold'
                  : 'hover:bg-white/5'
              }`}
              onClick={() => onSelect(strategy)}
            >
              <div className="flex items-start justify-between">
                <div className="flex-1 min-w-0">
                  <h4 className="text-sm font-medium text-nofx-text truncate">
                    {strategy.name}
                  </h4>
                  <p className="text-xs text-nofx-text-muted truncate">
                    {strategy.description || 'No description'}
                  </p>
                </div>
                
                {/* Action menu */}
                <div className="opacity-0 group-hover:opacity-100 transition-opacity flex gap-1 ml-2">
                  <button
                    onClick={(e) => {
                      e.stopPropagation()
                      onDuplicate(strategy.id)
                    }}
                    className="p-1 rounded hover:bg-white/20 text-xs"
                    title="Duplicate"
                  >
                    {/* Copy icon */}
                  </button>
                  <button
                    onClick={(e) => {
                      e.stopPropagation()
                      if (confirm(t('deleteConfirm'))) {
                        onDelete(strategy.id)
                      }
                    }}
                    className="p-1 rounded hover:bg-red-500/20 text-red-400 text-xs"
                    title="Delete"
                  >
                    {/* Trash icon */}
                  </button>
                </div>
              </div>

              {/* Status badges */}
              <div className="flex gap-1 mt-1">
                {strategy.is_active && (
                  <span className="px-1.5 py-0.5 text-xs bg-green-500/20 text-green-400 rounded">
                    Active
                  </span>
                )}
                {strategy.is_default && (
                  <span className="px-1.5 py-0.5 text-xs bg-blue-500/20 text-blue-400 rounded">
                    Default
                  </span>
                )}
                {strategy.is_public && (
                  <span className="px-1.5 py-0.5 text-xs bg-purple-500/20 text-purple-400 rounded">
                    Public
                  </span>
                )}
              </div>
            </div>
          ))
        )}
      </div>
    </div>
  )
}

// ============================================================================
// 5. 中央编辑面板 (StrategyEditorPanel.tsx)
// ============================================================================

interface StrategyEditorPanelProps {
  strategy: Strategy | null
  config: StrategyConfig | null
  validationErrors: Record<string, string[]>
  onConfigChange: <K extends keyof StrategyConfig>(key: K, value: StrategyConfig[K]) => void
  onNameChange: (name: string) => void
  onDescriptionChange: (desc: string) => void
  language: string
}

export const StrategyEditorPanel = ({
  strategy,
  config,
  validationErrors,
  onConfigChange,
  onNameChange,
  onDescriptionChange,
  language,
}: StrategyEditorPanelProps) => {
  const [activeTab, setActiveTab] = React.useState<'basic' | 'coin' | 'indicators' | 'risk' | 'prompt' | 'publish'>('basic')

  if (!strategy || !config) {
    return (
      <div className="flex-1 flex items-center justify-center text-nofx-text-muted">
        Select a strategy to edit
      </div>
    )
  }

  return (
    <div className="flex-1 flex flex-col overflow-hidden">
      {/* Basic info header */}
      <div className="flex-shrink-0 p-4 border-b border-nofx-gold/20">
        <input
          type="text"
          value={strategy.name}
          onChange={(e) => onNameChange(e.target.value)}
          disabled={strategy.is_default}
          className="text-lg font-bold bg-transparent border-none outline-none w-full text-nofx-text"
          placeholder="Strategy name"
        />
        <input
          type="text"
          value={strategy.description || ''}
          onChange={(e) => onDescriptionChange(e.target.value)}
          disabled={strategy.is_default}
          className="text-sm text-nofx-text-muted bg-transparent border-none outline-none w-full mt-2"
          placeholder="Description"
        />
      </div>

      {/* Tab navigation */}
      <div className="flex-shrink-0 border-b border-nofx-gold/20 flex bg-nofx-bg/50">
        <TabButton active={activeTab === 'basic'} onClick={() => setActiveTab('basic')}>
          Basic
        </TabButton>
        <TabButton active={activeTab === 'coin'} onClick={() => setActiveTab('coin')}>
          Coin Source
          {validationErrors.coin_source && <ErrorDot />}
        </TabButton>
        <TabButton active={activeTab === 'indicators'} onClick={() => setActiveTab('indicators')}>
          Indicators
          {validationErrors.indicators && <ErrorDot />}
        </TabButton>
        <TabButton active={activeTab === 'risk'} onClick={() => setActiveTab('risk')}>
          Risk Control
          {validationErrors.risk_control && <ErrorDot />}
        </TabButton>
        <TabButton active={activeTab === 'prompt'} onClick={() => setActiveTab('prompt')}>
          Prompt
        </TabButton>
        <TabButton active={activeTab === 'publish'} onClick={() => setActiveTab('publish')}>
          Publish
        </TabButton>
      </div>

      {/* Tab content */}
      <div className="flex-1 overflow-y-auto p-4">
        {activeTab === 'basic' && (
          <BasicInfoEditor strategy={strategy} disabled={strategy.is_default} />
        )}
        {activeTab === 'coin' && (
          <CoinSourceEditor
            config={config.coin_source}
            onChange={(value) => onConfigChange('coin_source', value)}
            disabled={strategy.is_default}
            errors={validationErrors.coin_source}
          />
        )}
        {activeTab === 'indicators' && (
          <IndicatorEditor
            config={config.indicators}
            onChange={(value) => onConfigChange('indicators', value)}
            disabled={strategy.is_default}
            errors={validationErrors.indicators}
          />
        )}
        {/* ... other tabs ... */}
      </div>
    </div>
  )
}

// ============================================================================
// 6. 预览面板 (PreviewPanel.tsx)
// ============================================================================

interface PreviewPanelProps {
  promptPreview: string | null
  aiTestResult: any | null
  isLoadingPrompt: boolean
  isLoadingTest: boolean
  onFetchPrompt: () => void
  onTestRun: () => void
}

export const PreviewPanel = ({
  promptPreview,
  aiTestResult,
  isLoadingPrompt,
  isLoadingTest,
  onFetchPrompt,
  onTestRun,
}: PreviewPanelProps) => {
  const [activeTab, setActiveTab] = React.useState<'prompt' | 'test'>('prompt')

  return (
    <div className="w-[420px] flex-shrink-0 flex flex-col border-l border-nofx-gold/20">
      {/* Tabs */}
      <div className="flex border-b border-nofx-gold/20">
        <button
          onClick={() => setActiveTab('prompt')}
          className={`flex-1 px-3 py-2 text-sm ${
            activeTab === 'prompt'
              ? 'border-b-2 border-nofx-gold text-nofx-gold'
              : 'text-nofx-text-muted'
          }`}
        >
          Prompt Preview
        </button>
        <button
          onClick={() => setActiveTab('test')}
          className={`flex-1 px-3 py-2 text-sm ${
            activeTab === 'test'
              ? 'border-b-2 border-nofx-gold text-nofx-gold'
              : 'text-nofx-text-muted'
          }`}
        >
          AI Test
        </button>
      </div>

      {/* Content */}
      <div className="flex-1 overflow-y-auto p-3">
        {activeTab === 'prompt' ? (
          <div className="space-y-3">
            <button
              onClick={onFetchPrompt}
              disabled={isLoadingPrompt}
              className="w-full px-3 py-2 bg-nofx-gold/20 hover:bg-nofx-gold/30 rounded text-sm font-medium"
            >
              {isLoadingPrompt ? 'Loading...' : 'Generate Prompt'}
            </button>
            {promptPreview && (
              <div className="bg-nofx-bg/50 rounded p-3 text-sm text-nofx-text/80 whitespace-pre-wrap font-mono">
                {promptPreview}
              </div>
            )}
          </div>
        ) : (
          <div className="space-y-3">
            <button
              onClick={onTestRun}
              disabled={isLoadingTest}
              className="w-full px-3 py-2 bg-nofx-gold/20 hover:bg-nofx-gold/30 rounded text-sm font-medium"
            >
              {isLoadingTest ? 'Testing...' : 'Run Test'}
            </button>
            {aiTestResult && (
              <div className="bg-nofx-bg/50 rounded p-3 text-sm">
                <pre className="whitespace-pre-wrap font-mono text-xs">
                  {JSON.stringify(aiTestResult, null, 2)}
                </pre>
              </div>
            )}
          </div>
        )}
      </div>
    </div>
  )
}

// ============================================================================
// 7. 主容器重构 (StrategyStudioPage.tsx - 简化版)
// ============================================================================

export const StrategyStudioPageRefactored = () => {
  const { token } = useAuth()
  const { language } = useLanguage()

  // 使用简化的状态管理
  const store = useStrategyStore()
  const { validateConfig } = useConfigValidator(
    store.editingConfig,
    (errors) => store.setValidationErrors(errors)
  )
  const { saveDraft, loadDraft } = useDraftSave(store.selectedStrategyId, store.editingConfig)

  // API 调用
  const fetchStrategies = async () => {
    // 实现省略...
  }

  const handleSaveStrategy = async () => {
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

      if (!response.ok) throw new Error('Failed to save')
      
      store.dispatch({ type: 'SAVE_SUCCESS' })
      await fetchStrategies()
      saveDraft()
    } catch (err) {
      store.dispatch({ type: 'SAVE_ERROR', payload: err.message })
    }
  }

  return (
    <div className="h-[calc(100vh-64px)] flex flex-col overflow-hidden bg-nofx-bg">
      {/* Header */}
      <div className="flex-shrink-0 p-4 border-b border-nofx-gold/20">
        <h1 className="text-2xl font-bold text-nofx-text">Strategy Studio</h1>
        <p className="text-sm text-nofx-text-muted">
          Create and manage your trading strategies
        </p>
      </div>

      {/* Error display */}
      {store.error && (
        <div className="flex-shrink-0 mx-4 mt-2 p-3 bg-red-500/20 border border-red-500/50 rounded text-sm text-red-400">
          {store.error}
        </div>
      )}

      {/* Main content - Three columns */}
      <div className="flex-1 flex overflow-hidden gap-0">
        <StrategyListPanel
          strategies={store.strategies}
          selectedId={store.selectedStrategyId}
          isLoading={store.isLoading}
          onSelect={store.selectStrategy}
          onNew={() => { /* 实现 */ }}
          onDelete={() => { /* 实现 */ }}
          onDuplicate={() => { /* 实现 */ }}
          language={language}
        />

        <StrategyEditorPanel
          strategy={
            store.strategies.find((s) => s.id === store.selectedStrategyId) || null
          }
          config={store.editingConfig}
          validationErrors={store.validationErrors}
          onConfigChange={store.updateConfig}
          onNameChange={(name) => {
            // 实现
          }}
          onDescriptionChange={(desc) => {
            // 实现
          }}
          language={language}
        />

        <PreviewPanel
          promptPreview={store.promptPreview}
          aiTestResult={store.aiTestResult}
          isLoadingPrompt={false}
          isLoadingTest={false}
          onFetchPrompt={() => { /* 实现 */ }}
          onTestRun={() => { /* 实现 */ }}
        />
      </div>

      {/* Footer - Save bar */}
      <div className="flex-shrink-0 flex items-center justify-between p-4 border-t border-nofx-gold/20 bg-nofx-bg/50">
        <div className="flex items-center gap-2">
          {store.hasChanges && (
            <span className="text-sm text-yellow-400 flex items-center gap-1">
              ⚠️ Unsaved changes
            </span>
          )}
          {Object.keys(store.validationErrors).length > 0 && (
            <span className="text-sm text-red-400">
              {Object.keys(store.validationErrors).length} validation error(s)
            </span>
          )}
        </div>
        
        <button
          onClick={handleSaveStrategy}
          disabled={!store.hasChanges || store.isSaving || store.hasErrors}
          className="px-4 py-2 bg-nofx-gold/20 hover:bg-nofx-gold/30 disabled:opacity-50 rounded font-medium"
        >
          {store.isSaving ? 'Saving...' : 'Save'}
        </button>
      </div>
    </div>
  )
}

export default StrategyStudioPageRefactored
