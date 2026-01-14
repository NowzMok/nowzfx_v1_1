# 触发价格策略前端集成指南

## 问题确认

用户反馈：**"在策略制定部分为不同交易员配置个性化的触发价格计算方式我要在哪进行设置？前端我没找到"**

## 当前状态分析

### ✅ 已完成的后端工作
1. **数据结构定义** - `store/strategy.go`
   - `TriggerPriceStrategy` 结构体
   - 在 `StrategyConfig` 中添加 `TriggerPriceConfig` 字段

2. **核心计算逻辑** - `trader/trigger_price_calculator.go`
   - 四种触发模式：current_price, pullback, breakout
   - 四种交易风格预设：long_term, short_term, swing, scalp

3. **系统集成** - `trader/auto_trader_analysis.go`
   - 已集成触发价格计算器
   - 支持从策略配置读取参数

### ❌ 缺失的前端工作
**当前前端没有触发价格策略配置界面！**

## 需要添加的前端功能

### 1. 在 StrategyStudioPage 添加触发价格配置组件

#### 新增组件：`TriggerPriceEditor.tsx`
```typescript
// nofx/web/src/components/strategy/TriggerPriceEditor.tsx
import { useState, useEffect } from 'react'
import type { TriggerPriceStrategy } from '../../types'

interface TriggerPriceEditorProps {
  config: TriggerPriceStrategy
  onChange: (config: TriggerPriceStrategy) => void
  disabled?: boolean
  language: 'zh' | 'en'
}

export function TriggerPriceEditor({ 
  config, 
  onChange, 
  disabled = false, 
  language 
}: TriggerPriceEditorProps) {
  const [preset, setPreset] = useState<string>('swing')

  // 预设配置模板
  const presets = {
    long_term: {
      mode: 'pullback',
      style: 'long_term',
      pullback_ratio: 0.05,
      breakout_ratio: 0.03,
      extra_buffer: 0.01,
    },
    short_term: {
      mode: 'pullback',
      style: 'short_term',
      pullback_ratio: 0.01,
      breakout_ratio: 0.005,
      extra_buffer: 0.002,
    },
    swing: {
      mode: 'pullback',
      style: 'swing',
      pullback_ratio: 0.02,
      breakout_ratio: 0.01,
      extra_buffer: 0.005,
    },
    scalp: {
      mode: 'current_price',
      style: 'scalp',
      pullback_ratio: 0.005,
      breakout_ratio: 0.003,
      extra_buffer: 0.001,
    },
  }

  const handlePresetChange = (presetName: string) => {
    setPreset(presetName)
    const presetConfig = presets[presetName as keyof typeof presets]
    onChange({ ...presetConfig })
  }

  const handleManualChange = (field: string, value: number) => {
    onChange({ ...config, [field]: value })
  }

  const t = (key: string) => {
    const translations: Record<string, Record<string, string>> = {
      title: { zh: '触发价格策略', en: 'Trigger Price Strategy' },
      preset: { zh: '预设风格', en: 'Preset Style' },
      mode: { zh: '触发模式', en: 'Trigger Mode' },
      pullback: { zh: '回调模式', en: 'Pullback' },
      breakout: { zh: '突破模式', en: 'Breakout' },
      current: { zh: '当前价格', en: 'Current Price' },
      pullbackRatio: { zh: '回调比例 (%)', en: 'Pullback Ratio (%)' },
      breakoutRatio: { zh: '突破比例 (%)', en: 'Breakout Ratio (%)' },
      extraBuffer: { zh: '额外缓冲 (%)', en: 'Extra Buffer (%)' },
      longTerm: { zh: '长线 (大回调)', en: 'Long Term (Large Pullback)' },
      shortTerm: { zh: '短线 (小回调)', en: 'Short Term (Small Pullback)' },
      swing: { zh: '摆动 (标准)', en: 'Swing (Standard)' },
      scalp: { zh: '剥头皮 (高敏感)', en: 'Scalp (High Sensitivity)' },
      description: { zh: '根据交易员风格自动调整触发价格，避免噪音交易', en: 'Auto-adjust trigger prices based on trader style to avoid noise' },
    }
    return translations[key]?.[language] || key
  }

  return (
    <div className="space-y-3">
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-2">
          <span className="text-sm font-medium text-nofx-text">{t('title')}</span>
          <span className="text-[10px] px-1.5 py-0.5 rounded bg-purple-500/20 text-purple-400">
            NEW
          </span>
        </div>
        <span className="text-[10px] text-nofx-text-muted">{t('description')}</span>
      </div>

      {/* 预设选择 */}
      <div className="grid grid-cols-4 gap-2">
        {['long_term', 'short_term', 'swing', 'scalp'].map((name) => (
          <button
            key={name}
            onClick={() => handlePresetChange(name)}
            disabled={disabled}
            className={`px-2 py-1.5 rounded text-[10px] font-medium transition-all ${
              preset === name
                ? 'bg-purple-600 text-white shadow-lg shadow-purple-500/20'
                : 'bg-nofx-bg border border-nofx-gold/20 text-nofx-text hover:border-nofx-gold/50'
            } ${disabled ? 'opacity-50 cursor-not-allowed' : ''}`}
          >
            {t(name)}
          </button>
        ))}
      </div>

      {/* 手动配置 */}
      <div className="space-y-2 p-3 rounded-lg bg-nofx-bg border border-nofx-gold/20">
        <div className="grid grid-cols-2 gap-2">
          <div>
            <label className="text-[10px] text-nofx-text-muted block mb-1">{t('mode')}</label>
            <select
              value={config.mode}
              onChange={(e) => handleManualChange('mode', e.target.value as any)}
              disabled={disabled}
              className="w-full px-2 py-1 rounded text-xs bg-nofx-bg-lighter border border-nofx-gold/20 text-nofx-text"
            >
              <option value="current_price">{t('current')}</option>
              <option value="pullback">{t('pullback')}</option>
              <option value="breakout">{t('breakout')}</option>
            </select>
          </div>
          <div>
            <label className="text-[10px] text-nofx-text-muted block mb-1">{t('pullbackRatio')}</label>
            <input
              type="number"
              step="0.1"
              value={config.pullback_ratio * 100}
              onChange={(e) => handleManualChange('pullback_ratio', parseFloat(e.target.value) / 100)}
              disabled={disabled}
              className="w-full px-2 py-1 rounded text-xs bg-nofx-bg-lighter border border-nofx-gold/20 text-nofx-text"
            />
          </div>
          <div>
            <label className="text-[10px] text-nofx-text-muted block mb-1">{t('breakoutRatio')}</label>
            <input
              type="number"
              step="0.1"
              value={config.breakout_ratio * 100}
              onChange={(e) => handleManualChange('breakout_ratio', parseFloat(e.target.value) / 100)}
              disabled={disabled}
              className="w-full px-2 py-1 rounded text-xs bg-nofx-bg-lighter border border-nofx-gold/20 text-nofx-text"
            />
          </div>
          <div>
            <label className="text-[10px] text-nofx-text-muted block mb-1">{t('extraBuffer')}</label>
            <input
              type="number"
              step="0.1"
              value={config.extra_buffer * 100}
              onChange={(e) => handleManualChange('extra_buffer', parseFloat(e.target.value) / 100)}
              disabled={disabled}
              className="w-full px-2 py-1 rounded text-xs bg-nofx-bg-lighter border border-nofx-gold/20 text-nofx-text"
            />
          </div>
        </div>

        {/* 配置预览 */}
        <div className="mt-2 p-2 rounded bg-nofx-bg-lighter border border-nofx-gold/10">
          <div className="text-[10px] font-mono text-nofx-text-muted">
            <div>Mode: <span className="text-nofx-text">{config.mode}</span></div>
            <div>Pullback: <span className="text-nofx-text">{(config.pullback_ratio * 100).toFixed(1)}%</span></div>
            <div>Breakout: <span className="text-nofx-text">{(config.breakout_ratio * 100).toFixed(1)}%</span></div>
            <div>Buffer: <span className="text-nofx-text">{(config.extra_buffer * 100).toFixed(1)}%</span></div>
          </div>
        </div>
      </div>
    </div>
  )
}
```

#### 更新类型定义
```typescript
// nofx/web/src/types.ts
export interface TriggerPriceStrategy {
  mode: 'current_price' | 'pullback' | 'breakout'
  style: 'long_term' | 'short_term' | 'swing' | 'scalp'
  pullback_ratio: number
  breakout_ratio: number
  extra_buffer: number
}

export interface StrategyConfig {
  language?: 'zh' | 'en'
  coin_source: CoinSourceConfig
  indicators: IndicatorConfig
  custom_prompt?: string
  risk_control: RiskControlConfig
  prompt_sections?: PromptSectionsConfig
  trigger_price_config?: TriggerPriceStrategy  // 新增字段
}
```

### 2. 在 StrategyStudioPage 集成 TriggerPriceEditor

#### 修改 StrategyStudioPage.tsx
```typescript
// 在 configSections 数组中添加新配置项
const configSections = [
  // ... 现有配置项
  {
    key: 'triggerPrice' as const,
    icon: Target,
    color: '#a855f7',
    title: t('triggerPrice'), // 需要添加翻译
    content: editingConfig && (
      <TriggerPriceEditor
        config={editingConfig.trigger_price_config || {
          mode: 'pullback',
          style: 'swing',
          pullback_ratio: 0.02,
          breakout_ratio: 0.01,
          extra_buffer: 0.005,
        }}
        onChange={(triggerPriceConfig) => updateConfig('trigger_price_config', triggerPriceConfig)}
        disabled={selectedStrategy?.is_default}
        language={language}
      />
    ),
  },
]
```

### 3. 在 TraderConfigModal 添加触发价格配置

#### 修改 TraderConfigModal.tsx
```typescript
// 在策略预览部分添加触发价格信息
{selectedStrategy && (
  <div className="mt-3 p-4 bg-[#1E2329] border border-[#2B3139] rounded-lg">
    {/* 现有策略信息 */}
    <div>
      触发价格策略: {selectedStrategy.config.trigger_price_config?.style || 'swing'} 
      ({selectedStrategy.config.trigger_price_config?.mode || 'pullback'})
    </div>
    <div>
      回调比例: {((selectedStrategy.config.trigger_price_config?.pullback_ratio || 0.02) * 100).toFixed(1)}%
    </div>
  </div>
)}
```

## 使用流程

### 1. 创建策略时配置触发价格
```
策略工作室 → 新建策略 → 触发价格策略 → 选择预设或手动配置
```

### 2. 为不同交易员选择不同策略
```
交易员配置 → 选择策略 → 策略包含触发价格配置 → 自动应用
```

### 3. 实际效果
- **长线交易员**：使用长线策略 → 大回调比例(5%) → 避免噪音
- **短线交易员**：使用短线策略 → 小回调比例(1%) → 快速响应

## 配置示例

### 长线交易员策略
```json
{
  "trigger_price_config": {
    "mode": "pullback",
    "style": "long_term",
    "pullback_ratio": 0.05,
    "breakout_ratio": 0.03,
    "extra_buffer": 0.01
  }
}
```

### 短线交易员策略
```json
{
  "trigger_price_config": {
    "mode": "pullback",
    "style": "short_term",
    "pullback_ratio": 0.01,
    "breakout_ratio": 0.005,
    "extra_buffer": 0.002
  }
}
```

## 总结

**问题**：前端找不到触发价格配置界面
**原因**：后端已实现，但前端尚未集成
**解决方案**：
1. 创建 `TriggerPriceEditor` 组件
2. 在 `StrategyStudioPage` 添加配置入口
3. 在 `TraderConfigModal` 显示配置预览
4. 通过策略系统自动应用到不同交易员

这样用户就可以在策略制定部分为不同交易风格配置个性化的触发价格计算方式了！
