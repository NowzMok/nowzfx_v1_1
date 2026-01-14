/**
 * ReflectionPanel.tsx - 紧凑版反思面板
 * 
 * 适配 nofx-glass 暗色主题风格
 * 用于仪表板侧边栏或卡片展示
 */

import { useState, useEffect } from 'react'
import { RefreshCw, Brain, AlertTriangle, TrendingUp, Target } from 'lucide-react'

interface Reflection {
  id: string
  type: 'performance' | 'risk' | 'strategy'
  content: string
  severity: 'info' | 'warning' | 'error'
  timestamp: string
}

interface ReflectionPanelProps {
  traderID: string
  language?: 'zh' | 'en'
  maxItems?: number
}

export function ReflectionPanel({
  traderID,
  language = 'zh',
  maxItems = 5,
}: ReflectionPanelProps) {
  const [reflections, setReflections] = useState<Reflection[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  const t = {
    zh: {
      title: '系统反思',
      refresh: '刷新',
      loading: '加载中...',
      empty: '暂无反思记录',
      emptyDesc: 'AI 将在交易周期中自动生成反思',
      performance: '性能',
      risk: '风险',
      strategy: '策略',
      error: '加载失败',
    },
    en: {
      title: 'System Reflection',
      refresh: 'Refresh',
      loading: 'Loading...',
      empty: 'No reflections yet',
      emptyDesc: 'AI will generate reflections during trading cycles',
      performance: 'Performance',
      risk: 'Risk',
      strategy: 'Strategy',
      error: 'Failed to load',
    },
  }[language]

  const fetchReflections = async () => {
    try {
      setLoading(true)
      setError(null)

      const response = await fetch(`/api/reflection/${traderID}/recent?limit=${maxItems}`)
      if (!response.ok) throw new Error('Failed to fetch')

      const data = await response.json()
      const rawList = data.data || []

      const mapped: Reflection[] = rawList.map((r: any) => {
        const aiReflection: string = r.ai_reflection || r.AIReflection || r.findings || ''
        const typeMatch = aiReflection.match(/^\[(performance|risk|strategy)\]/i)
        const type = (typeMatch ? typeMatch[1].toLowerCase() : 'performance') as Reflection['type']

        return {
          id: r.id || r.ID,
          type,
          content: aiReflection.replace(/^\[[^\]]+\]\s*/i, ''),
          severity: (r.severity || 'info') as Reflection['severity'],
          timestamp: r.reflection_time || r.created_at || r.ReflectionTime || r.CreatedAt,
        }
      })

      setReflections(mapped)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Unknown error')
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    fetchReflections()
    const interval = setInterval(fetchReflections, 60000)
    return () => clearInterval(interval)
  }, [traderID, maxItems])

  const getTypeIcon = (type: Reflection['type']) => {
    switch (type) {
      case 'performance':
        return <TrendingUp className="w-3.5 h-3.5" />
      case 'risk':
        return <AlertTriangle className="w-3.5 h-3.5" />
      case 'strategy':
        return <Target className="w-3.5 h-3.5" />
    }
  }

  const getTypeColor = (type: Reflection['type']) => {
    switch (type) {
      case 'performance':
        return 'text-blue-400 bg-blue-500/10 border-blue-500/30'
      case 'risk':
        return 'text-yellow-400 bg-yellow-500/10 border-yellow-500/30'
      case 'strategy':
        return 'text-purple-400 bg-purple-500/10 border-purple-500/30'
    }
  }

  const getSeverityDot = (severity: Reflection['severity']) => {
    switch (severity) {
      case 'error':
        return 'bg-nofx-red'
      case 'warning':
        return 'bg-yellow-500'
      default:
        return 'bg-nofx-green'
    }
  }

  const formatTime = (timestamp: string) => {
    try {
      const date = new Date(timestamp)
      const now = new Date()
      const diffMs = now.getTime() - date.getTime()
      const diffMins = Math.floor(diffMs / 60000)
      const diffHours = Math.floor(diffMins / 60)
      const diffDays = Math.floor(diffHours / 24)

      if (diffMins < 1) return language === 'zh' ? '刚刚' : 'Just now'
      if (diffMins < 60) return `${diffMins}${language === 'zh' ? '分钟前' : 'm ago'}`
      if (diffHours < 24) return `${diffHours}${language === 'zh' ? '小时前' : 'h ago'}`
      return `${diffDays}${language === 'zh' ? '天前' : 'd ago'}`
    } catch {
      return ''
    }
  }

  return (
    <div className="h-full flex flex-col">
      {/* Header */}
      <div className="flex items-center justify-between mb-4 pb-3 border-b border-white/5">
        <h3 className="text-lg font-bold flex items-center gap-2 text-nofx-text-main">
          <Brain className="w-5 h-5 text-purple-400" />
          {t.title}
        </h3>
        <button
          onClick={fetchReflections}
          disabled={loading}
          className="p-1.5 rounded-lg hover:bg-white/10 transition-colors disabled:opacity-50"
          title={t.refresh}
        >
          <RefreshCw className={`w-4 h-4 text-nofx-text-muted ${loading ? 'animate-spin' : ''}`} />
        </button>
      </div>

      {/* Content */}
      <div className="flex-1 overflow-y-auto custom-scrollbar space-y-3">
        {loading && reflections.length === 0 ? (
          <div className="flex items-center justify-center py-8">
            <div className="text-center text-nofx-text-muted">
              <RefreshCw className="w-6 h-6 animate-spin mx-auto mb-2 opacity-50" />
              <span className="text-sm">{t.loading}</span>
            </div>
          </div>
        ) : error ? (
          <div className="text-center py-8 text-nofx-red/80">
            <AlertTriangle className="w-6 h-6 mx-auto mb-2 opacity-50" />
            <span className="text-sm">{t.error}</span>
          </div>
        ) : reflections.length === 0 ? (
          <div className="text-center py-8 text-nofx-text-muted">
            <div className="text-4xl mb-3 opacity-30 grayscale">🧠</div>
            <div className="text-sm font-medium mb-1">{t.empty}</div>
            <div className="text-xs opacity-60">{t.emptyDesc}</div>
          </div>
        ) : (
          reflections.map((reflection) => (
            <div
              key={reflection.id}
              className="p-3 rounded-lg bg-white/[0.02] border border-white/5 hover:border-white/10 transition-colors group"
            >
              {/* Type badge + time */}
              <div className="flex items-center justify-between mb-2">
                <span
                  className={`inline-flex items-center gap-1.5 px-2 py-0.5 rounded text-[10px] font-semibold uppercase tracking-wide border ${getTypeColor(reflection.type)}`}
                >
                  {getTypeIcon(reflection.type)}
                  {t[reflection.type]}
                </span>
                <div className="flex items-center gap-2">
                  <span className={`w-1.5 h-1.5 rounded-full ${getSeverityDot(reflection.severity)}`} />
                  <span className="text-[10px] text-nofx-text-muted font-mono">
                    {formatTime(reflection.timestamp)}
                  </span>
                </div>
              </div>

              {/* Content */}
              <p className="text-xs text-nofx-text-main leading-relaxed line-clamp-3 group-hover:line-clamp-none transition-all">
                {reflection.content}
              </p>
            </div>
          ))
        )}
      </div>

      {/* Footer - count */}
      {reflections.length > 0 && (
        <div className="mt-3 pt-3 border-t border-white/5 text-[10px] text-nofx-text-muted text-center font-mono">
          {language === 'zh' ? `显示最近 ${reflections.length} 条` : `Showing latest ${reflections.length}`}
        </div>
      )}
    </div>
  )
}
