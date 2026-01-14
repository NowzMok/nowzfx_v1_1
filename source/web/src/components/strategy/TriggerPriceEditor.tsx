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
  language,
}: TriggerPriceEditorProps) {
  // 从当前配置推断预设，或使用默认值
  const [preset, setPreset] = useState<string>(() => {
    // 根据配置的style字段确定预设
    if (config?.style) {
      return config.style
    }
    // 如果没有style，根据参数推断
    if (config?.pullback_ratio === 0.005 && config?.extra_buffer === 0.001) {
      return 'scalp'
    }
    if (config?.pullback_ratio === 0.01 && config?.extra_buffer === 0.002) {
      return 'short_term'
    }
    if (config?.pullback_ratio === 0.02 && config?.extra_buffer === 0.005) {
      return 'swing'
    }
    if (config?.pullback_ratio === 0.05 && config?.extra_buffer === 0.01) {
      return 'long_term'
    }
    return 'swing'
  })

  // 当配置变化时，更新preset状态
  useEffect(() => {
    if (config?.style) {
      setPreset(config.style)
    } else if (
      config?.pullback_ratio !== undefined &&
      config?.extra_buffer !== undefined
    ) {
      // 根据参数推断预设
      if (config.pullback_ratio === 0.005 && config.extra_buffer === 0.001) {
        setPreset('scalp')
      } else if (
        config.pullback_ratio === 0.01 &&
        config.extra_buffer === 0.002
      ) {
        setPreset('short_term')
      } else if (
        config.pullback_ratio === 0.02 &&
        config.extra_buffer === 0.005
      ) {
        setPreset('swing')
      } else if (
        config.pullback_ratio === 0.05 &&
        config.extra_buffer === 0.01
      ) {
        setPreset('long_term')
      } else {
        setPreset('swing') // 默认值
      }
    }
  }, [config])

  // 预设配置模板 - 针对不同交易员风格优化
  const presets: Record<string, TriggerPriceStrategy> = {
    long_term: {
      mode: 'pullback',
      style: 'long_term',
      pullback_ratio: 0.05, // 5% 大回调
      breakout_ratio: 0.03, // 3% 突破
      extra_buffer: 0.01, // 1% 额外缓冲
    },
    short_term: {
      mode: 'pullback',
      style: 'short_term',
      pullback_ratio: 0.01, // 1% 小回调
      breakout_ratio: 0.005, // 0.5% 突破
      extra_buffer: 0.002, // 0.2% 额外缓冲
    },
    swing: {
      mode: 'pullback',
      style: 'swing',
      pullback_ratio: 0.02, // 2% 标准回调
      breakout_ratio: 0.01, // 1% 突破
      extra_buffer: 0.005, // 0.5% 额外缓冲
    },
    scalp: {
      mode: 'current_price', // 剥头皮用当前价格模式
      style: 'scalp',
      pullback_ratio: 0.005, // 0.5% 微小回调
      breakout_ratio: 0.003, // 0.3% 微小突破
      extra_buffer: 0.001, // 0.1% 最小缓冲
    },
  }

  // 计算触发价格预览
  const calculateTriggerPreview = () => {
    const currentPrice = 100 // 假设当前价格为100用于预览
    let triggerPrice = currentPrice

    if (config.mode === 'pullback') {
      triggerPrice =
        currentPrice * (1 - config.pullback_ratio - config.extra_buffer)
    } else if (config.mode === 'breakout') {
      triggerPrice =
        currentPrice * (1 + config.breakout_ratio + config.extra_buffer)
    } else if (config.mode === 'current_price') {
      triggerPrice = currentPrice * (1 - config.extra_buffer) // 当前价格模式下，只应用缓冲
    }

    return {
      trigger_price: triggerPrice.toFixed(2),
      difference: (
        ((currentPrice - triggerPrice) / currentPrice) *
        100
      ).toFixed(2),
      current_price: currentPrice.toFixed(2),
    }
  }

  const handlePresetChange = (presetName: string) => {
    setPreset(presetName)
    const presetConfig = presets[presetName as keyof typeof presets]
    onChange({ ...presetConfig })
  }

  const handleManualChange = (field: string, value: number | string) => {
    onChange({ ...config, [field]: value })
  }

  const t = (key: string) => {
    const translations: Record<string, Record<string, string>> = {
      title: { zh: '触发价格策略', en: 'Trigger Price Strategy' },
      subtitle: {
        zh: '根据交易员风格自动调整触发价格',
        en: 'Auto-adjust trigger prices based on trader style',
      },
      preset: { zh: '交易员风格预设', en: 'Trader Style Preset' },
      mode: { zh: '触发模式', en: 'Trigger Mode' },
      pullback: { zh: '回调模式', en: 'Pullback' },
      breakout: { zh: '突破模式', en: 'Breakout' },
      current: { zh: '当前价格', en: 'Current Price' },
      pullbackRatio: { zh: '回调比例 (%)', en: 'Pullback Ratio (%)' },
      breakoutRatio: { zh: '突破比例 (%)', en: 'Breakout Ratio (%)' },
      extraBuffer: { zh: '额外缓冲 (%)', en: 'Extra Buffer (%)' },
      longTerm: { zh: '长线', en: 'Long Term' },
      shortTerm: { zh: '短线', en: 'Short Term' },
      swing: { zh: '摆动', en: 'Swing' },
      scalp: { zh: '剥头皮', en: 'Scalp' },
      description: {
        zh: '避免噪音交易，优化入场时机',
        en: 'Avoid noise trading, optimize entry timing',
      },
      preview: { zh: '触发价格预览', en: 'Trigger Price Preview' },
      currentPrice: { zh: '当前价格', en: 'Current Price' },
      calculatedTrigger: { zh: '计算触发价', en: 'Calculated Trigger' },
      priceDifference: { zh: '价格差异', en: 'Price Difference' },
      longTermDesc: {
        zh: '容忍大回调，适合长线持有',
        en: 'Tolerate large pullbacks, suitable for long-term holding',
      },
      shortTermDesc: {
        zh: '平衡敏感度，适合短线交易',
        en: 'Balanced sensitivity, suitable for short-term trading',
      },
      swingDesc: {
        zh: '标准配置，适合摆动交易',
        en: 'Standard config, suitable for swing trading',
      },
      scalpDesc: {
        zh: '高敏感度，快速响应，适合剥头皮',
        en: 'High sensitivity, fast response, suitable for scalping',
      },
      modeDesc: { zh: '触发模式说明', en: 'Mode Description' },
      modePullbackDesc: {
        zh: '价格回调时入场，适合大多数策略',
        en: 'Enter on price pullback, suitable for most strategies',
      },
      modeBreakoutDesc: {
        zh: '价格突破时入场，适合追涨杀跌',
        en: 'Enter on price breakout, suitable for momentum trading',
      },
      modeCurrentDesc: {
        zh: '当前价格附近入场，适合高频交易',
        en: 'Enter near current price, suitable for high-frequency trading',
      },
    }
    return translations[key]?.[language] || key
  }

  const preview = calculateTriggerPreview()

  return (
    <div className="space-y-4">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-2">
          <span className="text-sm font-medium text-nofx-text">
            {t('title')}
          </span>
          <span className="text-[10px] px-1.5 py-0.5 rounded bg-purple-500/20 text-purple-400">
            PRO
          </span>
        </div>
        <span className="text-[10px] text-nofx-text-muted">
          {t('subtitle')}
        </span>
      </div>

      {/* 预设选择 - 交易员风格 */}
      <div className="space-y-2">
        <div className="flex items-center gap-2">
          <span className="text-xs font-medium text-nofx-text">
            {t('preset')}
          </span>
          <span className="text-[10px] text-nofx-text-muted">
            ({t('description')})
          </span>
        </div>
        <div className="grid grid-cols-2 gap-2">
          {[
            { key: 'long_term', color: 'bg-blue-600', icon: '📈' },
            { key: 'short_term', color: 'bg-green-600', icon: '📊' },
            { key: 'swing', color: 'bg-yellow-600', icon: '⚡' },
            { key: 'scalp', color: 'bg-red-600', icon: '⚡' },
          ].map(({ key, color, icon }) => (
            <button
              key={key}
              onClick={() => handlePresetChange(key)}
              disabled={disabled}
              className={`relative px-3 py-2.5 rounded-lg text-left transition-all ${
                preset === key
                  ? `${color} text-white shadow-lg shadow-white/10 ring-2 ring-white/30`
                  : 'bg-nofx-bg border border-nofx-gold/20 text-nofx-text hover:border-nofx-gold/50 hover:bg-nofx-bg-lighter'
              } ${disabled ? 'opacity-50 cursor-not-allowed' : ''}`}
            >
              <div className="flex items-center justify-between">
                <div className="flex items-center gap-2">
                  <span className="text-lg">{icon}</span>
                  <div>
                    <div className="text-sm font-bold">{t(key)}</div>
                    <div
                      className={`text-[10px] ${preset === key ? 'text-white/90' : 'text-nofx-text-muted'}`}
                    >
                      {t(`${key}Desc`)}
                    </div>
                  </div>
                </div>
                {preset === key && (
                  <div className="w-2 h-2 rounded-full bg-white animate-pulse" />
                )}
              </div>
            </button>
          ))}
        </div>
      </div>

      {/* 手动配置区域 */}
      <div className="space-y-3 p-3 rounded-lg bg-nofx-bg border border-nofx-gold/20">
        <div className="flex items-center justify-between">
          <span className="text-xs font-medium text-nofx-text">
            {t('mode')}
          </span>
          <span className="text-[10px] text-nofx-text-muted">
            {t('modeDesc')}
          </span>
        </div>

        <div className="grid grid-cols-3 gap-2">
          <button
            onClick={() =>
              !disabled && handleManualChange('mode', 'current_price')
            }
            disabled={disabled}
            className={`px-2 py-1.5 rounded text-xs transition-all ${
              config.mode === 'current_price'
                ? 'bg-blue-600 text-white'
                : 'bg-nofx-bg-lighter border border-nofx-gold/20 text-nofx-text hover:border-nofx-gold/50'
            }`}
          >
            {t('current')}
          </button>
          <button
            onClick={() => !disabled && handleManualChange('mode', 'pullback')}
            disabled={disabled}
            className={`px-2 py-1.5 rounded text-xs transition-all ${
              config.mode === 'pullback'
                ? 'bg-green-600 text-white'
                : 'bg-nofx-bg-lighter border border-nofx-gold/20 text-nofx-text hover:border-nofx-gold/50'
            }`}
          >
            {t('pullback')}
          </button>
          <button
            onClick={() => !disabled && handleManualChange('mode', 'breakout')}
            disabled={disabled}
            className={`px-2 py-1.5 rounded text-xs transition-all ${
              config.mode === 'breakout'
                ? 'bg-yellow-600 text-white'
                : 'bg-nofx-bg-lighter border border-nofx-gold/20 text-nofx-text hover:border-nofx-gold/50'
            }`}
          >
            {t('breakout')}
          </button>
        </div>

        {/* 模式说明 */}
        <div className="text-[10px] text-nofx-text-muted italic">
          {config.mode === 'current_price' && t('modeCurrentDesc')}
          {config.mode === 'pullback' && t('modePullbackDesc')}
          {config.mode === 'breakout' && t('modeBreakoutDesc')}
        </div>

        {/* 参数滑块 */}
        <div className="space-y-2">
          <div>
            <div className="flex items-center justify-between mb-1">
              <label className="text-[10px] text-nofx-text">
                {t('pullbackRatio')}
              </label>
              <span className="text-[10px] font-mono text-nofx-gold">
                {(config.pullback_ratio * 100).toFixed(2)}%
              </span>
            </div>
            <input
              type="range"
              min="0"
              max="10"
              step="0.1"
              value={config.pullback_ratio * 100}
              onChange={(e) =>
                handleManualChange(
                  'pullback_ratio',
                  parseFloat(e.target.value) / 100
                )
              }
              disabled={disabled}
              className="w-full h-1.5 bg-nofx-bg-lighter rounded-lg appearance-none cursor-pointer accent-purple-500"
            />
          </div>

          <div>
            <div className="flex items-center justify-between mb-1">
              <label className="text-[10px] text-nofx-text">
                {t('breakoutRatio')}
              </label>
              <span className="text-[10px] font-mono text-nofx-gold">
                {(config.breakout_ratio * 100).toFixed(2)}%
              </span>
            </div>
            <input
              type="range"
              min="0"
              max="5"
              step="0.05"
              value={config.breakout_ratio * 100}
              onChange={(e) =>
                handleManualChange(
                  'breakout_ratio',
                  parseFloat(e.target.value) / 100
                )
              }
              disabled={disabled}
              className="w-full h-1.5 bg-nofx-bg-lighter rounded-lg appearance-none cursor-pointer accent-green-500"
            />
          </div>

          <div>
            <div className="flex items-center justify-between mb-1">
              <label className="text-[10px] text-nofx-text">
                {t('extraBuffer')}
              </label>
              <span className="text-[10px] font-mono text-nofx-gold">
                {(config.extra_buffer * 100).toFixed(2)}%
              </span>
            </div>
            <input
              type="range"
              min="0"
              max="2"
              step="0.05"
              value={config.extra_buffer * 100}
              onChange={(e) =>
                handleManualChange(
                  'extra_buffer',
                  parseFloat(e.target.value) / 100
                )
              }
              disabled={disabled}
              className="w-full h-1.5 bg-nofx-bg-lighter rounded-lg appearance-none cursor-pointer accent-yellow-500"
            />
          </div>
        </div>

        {/* 实时预览 */}
        <div className="mt-3 p-3 rounded-lg bg-gradient-to-r from-nofx-bg-lighter to-nofx-bg border border-nofx-gold/30">
          <div className="flex items-center gap-2 mb-2">
            <span className="text-xs font-medium text-nofx-text">
              {t('preview')}
            </span>
            <span className="text-[10px] px-1.5 py-0.5 rounded bg-purple-500/20 text-purple-400">
              LIVE
            </span>
          </div>
          <div className="grid grid-cols-3 gap-2 text-center">
            <div className="p-2 rounded bg-nofx-bg border border-nofx-gold/10">
              <div className="text-[9px] text-nofx-text-muted">
                {t('currentPrice')}
              </div>
              <div className="text-sm font-mono text-nofx-text font-bold">
                {preview.current_price}
              </div>
            </div>
            <div className="p-2 rounded bg-nofx-bg border border-nofx-gold/10">
              <div className="text-[9px] text-nofx-text-muted">
                {t('calculatedTrigger')}
              </div>
              <div className="text-sm font-mono text-purple-400 font-bold">
                {preview.trigger_price}
              </div>
            </div>
            <div className="p-2 rounded bg-nofx-bg border border-nofx-gold/10">
              <div className="text-[9px] text-nofx-text-muted">
                {t('priceDifference')}
              </div>
              <div className="text-sm font-mono text-nofx-gold font-bold">
                -{preview.difference}%
              </div>
            </div>
          </div>
        </div>

        {/* 配置摘要 */}
        <div className="mt-2 p-2 rounded bg-nofx-bg-lighter border border-nofx-gold/10">
          <div className="text-[10px] font-mono text-nofx-text-muted space-y-0.5">
            <div>
              Style: <span className="text-nofx-text">{config.style}</span>
            </div>
            <div>
              Mode: <span className="text-nofx-text">{config.mode}</span>
            </div>
            <div>
              Pullback:{' '}
              <span className="text-nofx-text">
                {(config.pullback_ratio * 100).toFixed(2)}%
              </span>
            </div>
            <div>
              Breakout:{' '}
              <span className="text-nofx-text">
                {(config.breakout_ratio * 100).toFixed(2)}%
              </span>
            </div>
            <div>
              Buffer:{' '}
              <span className="text-nofx-text">
                {(config.extra_buffer * 100).toFixed(2)}%
              </span>
            </div>
          </div>
        </div>
      </div>

      {/* 使用建议 */}
      <div className="text-[10px] text-nofx-text-muted italic bg-nofx-bg/50 p-2 rounded border border-nofx-gold/10">
        💡{' '}
        {language === 'zh'
          ? '提示：不同风格会自动调整参数。剥头皮风格使用当前价格模式，其他风格使用回调模式。长线容忍大回调，短线更敏感。'
          : 'Tip: Different styles auto-adjust parameters. Scalp uses current price mode, others use pullback. Long-term tolerates large pullbacks, short-term is more sensitive.'}
      </div>
    </div>
  )
}
