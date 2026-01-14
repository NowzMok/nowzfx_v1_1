import { useState } from 'react'
import {
  LineChart,
  Line,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  ResponsiveContainer,
  ReferenceLine,
} from 'recharts'
import useSWR from 'swr'
import { api } from '../lib/api'
import { useLanguage } from '../contexts/LanguageContext'
import { useAuth } from '../contexts/AuthContext'
import { t } from '../i18n/translations'
import {
  AlertTriangle,
  BarChart3,
  DollarSign,
  Percent,
  TrendingUp as ArrowUp,
  TrendingDown as ArrowDown,
} from 'lucide-react'

interface EquityPoint {
  timestamp: string
  total_equity: number
  pnl: number
  pnl_pct: number
  cycle_number: number
}

interface EquityChartProps {
  traderId?: string
  embedded?: boolean // 嵌入模式（不显示外层卡片）
}

export function EquityChart({ traderId, embedded = false }: EquityChartProps) {
  const { language } = useLanguage()
  const { user, token } = useAuth()
  const [displayMode, setDisplayMode] = useState<'dollar' | 'percent'>('dollar')

  const {
    data: history,
    error,
    isLoading,
  } = useSWR<EquityPoint[]>(
    user && token && traderId ? `equity-history-${traderId}` : null,
    () => api.getEquityHistory(traderId),
    {
      refreshInterval: 30000, // 30秒刷新（历史数据更新频率较低）
      revalidateOnFocus: false,
      dedupingInterval: 20000,
    }
  )

  const { data: account } = useSWR(
    user && token && traderId ? `account-${traderId}` : null,
    () => api.getAccount(traderId),
    {
      refreshInterval: 15000, // 15秒刷新（配合后端缓存）
      revalidateOnFocus: false,
      dedupingInterval: 10000,
    }
  )

  // Loading state - show skeleton
  if (isLoading) {
    return (
      <div className={embedded ? 'p-6' : 'binance-card p-6'}>
        {!embedded && (
          <h3
            className="text-lg font-semibold mb-6"
            style={{ color: '#EAECEF' }}
          >
            {t('accountEquityCurve', language)}
          </h3>
        )}
        <div className="animate-pulse">
          <div className="skeleton h-64 w-full rounded"></div>
        </div>
      </div>
    )
  }

  if (error) {
    return (
      <div className={embedded ? 'p-6' : 'binance-card p-6'}>
        <div
          className="flex items-center gap-3 p-4 rounded"
          style={{
            background: 'rgba(246, 70, 93, 0.1)',
            border: '1px solid rgba(246, 70, 93, 0.2)',
          }}
        >
          <AlertTriangle className="w-6 h-6" style={{ color: '#F6465D' }} />
          <div>
            <div className="font-semibold" style={{ color: '#F6465D' }}>
              {t('loadingError', language)}
            </div>
            <div className="text-sm" style={{ color: '#848E9C' }}>
              {error.message}
            </div>
          </div>
        </div>
      </div>
    )
  }

  // 过滤掉无效数据：total_equity为0或小于1的数据点（API失败导致）
  const validHistory = history?.filter((point) => point.total_equity > 1) || []

  if (!validHistory || validHistory.length === 0) {
    return (
      <div className={embedded ? 'p-6' : 'binance-card p-6'}>
        {!embedded && (
          <h3
            className="text-lg font-semibold mb-6"
            style={{ color: '#EAECEF' }}
          >
            {t('accountEquityCurve', language)}
          </h3>
        )}
        <div className="text-center py-16" style={{ color: '#848E9C' }}>
          <div className="mb-4 flex justify-center opacity-50">
            <BarChart3 className="w-16 h-16" />
          </div>
          <div className="text-lg font-semibold mb-2">
            {t('noHistoricalData', language)}
          </div>
          <div className="text-sm">{t('dataWillAppear', language)}</div>
        </div>
      </div>
    )
  }

  // 限制显示最近的数据点（性能优化）
  // 如果数据超过2000个点，只显示最近2000个
  const MAX_DISPLAY_POINTS = 2000
  const displayHistory =
    validHistory.length > MAX_DISPLAY_POINTS
      ? validHistory.slice(-MAX_DISPLAY_POINTS)
      : validHistory

  // 计算初始余额（优先从 account 获取配置的初始余额，备选从历史数据反推）
  const initialBalance =
    account?.initial_balance || // 从交易员配置读取真实初始余额
    (validHistory[0]
      ? validHistory[0].total_equity - validHistory[0].pnl
      : undefined) || // 备选：淨值 - 盈亏
    1000 // 默认值（与创建交易员时的默认配置一致）

  // 转换数据格式
  const chartData = displayHistory.map((point) => {
    const pnl = point.total_equity - initialBalance
    const pnlPct = ((pnl / initialBalance) * 100).toFixed(2)
    return {
      time: new Date(point.timestamp).toLocaleTimeString('zh-CN', {
        hour: '2-digit',
        minute: '2-digit',
      }),
      value: displayMode === 'dollar' ? point.total_equity : parseFloat(pnlPct),
      cycle: point.cycle_number,
      raw_equity: point.total_equity,
      raw_pnl: pnl,
      raw_pnl_pct: parseFloat(pnlPct),
    }
  })

  const currentValue = chartData[chartData.length - 1]
  const isProfit = currentValue.raw_pnl >= 0

  // 计算Y轴范围
  const calculateYDomain = () => {
    if (displayMode === 'percent') {
      // 百分比模式：找到最大最小值，留20%余量
      const values = chartData.map((d) => d.value)
      const minVal = Math.min(...values)
      const maxVal = Math.max(...values)
      const range = Math.max(Math.abs(maxVal), Math.abs(minVal))
      const padding = Math.max(range * 0.2, 1) // 至少留1%余量
      return [Math.floor(minVal - padding), Math.ceil(maxVal + padding)]
    } else {
      // 美元模式：以初始余额为基准，上下留10%余量
      const values = chartData.map((d) => d.value)
      const minVal = Math.min(...values, initialBalance)
      const maxVal = Math.max(...values, initialBalance)
      const range = maxVal - minVal
      const padding = Math.max(range * 0.15, initialBalance * 0.01) // 至少留1%余量
      return [Math.floor(minVal - padding), Math.ceil(maxVal + padding)]
    }
  }

  // 自定义Tooltip - Premium Style
  const CustomTooltip = ({ active, payload }: any) => {
    if (active && payload && payload.length) {
      const data = payload[0].payload
      const isPositive = data.raw_pnl >= 0
      return (
        <div
          className="rounded-xl p-4 shadow-2xl backdrop-blur-sm"
          style={{ 
            background: 'linear-gradient(135deg, rgba(30, 35, 41, 0.95) 0%, rgba(20, 24, 30, 0.98) 100%)', 
            border: '1px solid rgba(255, 255, 255, 0.1)',
            boxShadow: '0 10px 40px rgba(0, 0, 0, 0.5)',
          }}
        >
          <div className="text-[10px] mb-2 uppercase tracking-wider font-medium" style={{ color: '#5E6673' }}>
            Cycle #{data.cycle}
          </div>
          <div className="text-lg font-bold mono mb-1" style={{ color: '#EAECEF' }}>
            {data.raw_equity.toFixed(2)} <span className="text-xs text-gray-500">USDT</span>
          </div>
          <div
            className="text-sm mono font-bold flex items-center gap-1"
            style={{ color: isPositive ? '#0ECB81' : '#F6465D' }}
          >
            <span className={`w-1.5 h-1.5 rounded-full ${isPositive ? 'bg-green-400' : 'bg-red-400'}`}></span>
            {isPositive ? '+' : ''}
            {data.raw_pnl.toFixed(2)} ({isPositive ? '+' : ''}
            {data.raw_pnl_pct}%)
          </div>
        </div>
      )
    }
    return null
  }

  return (
    <div
      className={
        embedded ? 'p-4 sm:p-5 h-full flex flex-col' : 'binance-card p-4 sm:p-5 animate-fade-in'
      }
    >
      {/* Header */}
      <div className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between mb-3 shrink-0">
        <div className="flex-1">
          {!embedded && (
            <h3
              className="text-base sm:text-lg font-bold mb-2 flex items-center gap-2"
              style={{ color: '#EAECEF' }}
            >
              <span className="w-2 h-2 rounded-full bg-nofx-gold animate-pulse"></span>
              {t('accountEquityCurve', language)}
            </h3>
          )}
          <div className="flex flex-col sm:flex-row sm:items-baseline gap-2 sm:gap-4">
            <div className="flex items-baseline gap-2">
              <span
                className="text-2xl sm:text-3xl font-bold mono tracking-tight"
                style={{ color: '#EAECEF' }}
              >
                {account?.total_equity.toFixed(2) || '0.00'}
              </span>
              <span
                className="text-sm font-medium"
                style={{ color: '#848E9C' }}
              >
                USDT
              </span>
            </div>
            <div className="flex items-center gap-2 flex-wrap">
              <span
                className="text-sm sm:text-base font-bold mono px-3 py-1.5 rounded-lg flex items-center gap-1.5 transition-all"
                style={{
                  color: isProfit ? '#0ECB81' : '#F6465D',
                  background: isProfit
                    ? 'rgba(14, 203, 129, 0.15)'
                    : 'rgba(246, 70, 93, 0.15)',
                  border: `1px solid ${isProfit ? 'rgba(14, 203, 129, 0.3)' : 'rgba(246, 70, 93, 0.3)'}`,
                  boxShadow: isProfit 
                    ? '0 0 20px rgba(14, 203, 129, 0.2)' 
                    : '0 0 20px rgba(246, 70, 93, 0.2)',
                }}
              >
                {isProfit ? (
                  <ArrowUp className="w-4 h-4" />
                ) : (
                  <ArrowDown className="w-4 h-4" />
                )}
                {isProfit ? '+' : ''}
                {currentValue.raw_pnl_pct}%
              </span>
              <span
                className="text-xs sm:text-sm mono font-medium"
                style={{ color: '#5E6673' }}
              >
                ({isProfit ? '+' : ''}
                {currentValue.raw_pnl.toFixed(2)} USDT)
              </span>
            </div>
          </div>
        </div>

        {/* Display Mode Toggle */}
        <div
          className="flex gap-1 rounded-lg p-1 self-start sm:self-auto"
          style={{ background: 'rgba(0, 0, 0, 0.4)', border: '1px solid rgba(255, 255, 255, 0.08)' }}
        >
          <button
            onClick={() => setDisplayMode('dollar')}
            className="px-3 py-1.5 rounded-md text-xs font-bold transition-all flex items-center gap-1"
            style={
              displayMode === 'dollar'
                ? {
                    background: 'linear-gradient(135deg, #F0B90B 0%, #FCD535 100%)',
                    color: '#000',
                    boxShadow: '0 4px 12px rgba(240, 185, 11, 0.35)',
                  }
                : { background: 'transparent', color: '#5E6673' }
            }
          >
            <DollarSign className="w-3.5 h-3.5" />
            <span className="hidden sm:inline">USDT</span>
          </button>
          <button
            onClick={() => setDisplayMode('percent')}
            className="px-3 py-1.5 rounded-md text-xs font-bold transition-all flex items-center gap-1"
            style={
              displayMode === 'percent'
                ? {
                    background: 'linear-gradient(135deg, #F0B90B 0%, #FCD535 100%)',
                    color: '#000',
                    boxShadow: '0 4px 12px rgba(240, 185, 11, 0.35)',
                  }
                : { background: 'transparent', color: '#5E6673' }
            }
          >
            <Percent className="w-3.5 h-3.5" />
          </button>
        </div>
      </div>

      {/* Chart */}
      <div
        className="flex-1 min-h-0 my-2"
        style={{
          borderRadius: '12px',
          overflow: 'hidden',
          position: 'relative',
          background: 'linear-gradient(180deg, rgba(0, 0, 0, 0.2) 0%, rgba(0, 0, 0, 0.4) 100%)',
        }}
      >
        {/* NOFX Watermark */}
        <div
          style={{
            position: 'absolute',
            top: '50%',
            left: '50%',
            transform: 'translate(-50%, -50%)',
            fontSize: '48px',
            fontWeight: 'bold',
            color: 'rgba(240, 185, 11, 0.04)',
            zIndex: 5,
            pointerEvents: 'none',
            fontFamily: 'monospace',
            letterSpacing: '0.2em',
          }}
        >
          NOFX
        </div>
        <ResponsiveContainer width="100%" height="100%">
          <LineChart
            data={chartData}
            margin={{ top: 15, right: 25, left: 10, bottom: 25 }}
          >
            <defs>
              <linearGradient id="lineGradient" x1="0" y1="0" x2="1" y2="0">
                <stop offset="0%" stopColor="#F0B90B" stopOpacity={1} />
                <stop offset="100%" stopColor="#FCD535" stopOpacity={1} />
              </linearGradient>
              <linearGradient id="areaGradient" x1="0" y1="0" x2="0" y2="1">
                <stop offset="0%" stopColor="#F0B90B" stopOpacity={0.25} />
                <stop offset="100%" stopColor="#F0B90B" stopOpacity={0} />
              </linearGradient>
              <filter id="glow">
                <feGaussianBlur stdDeviation="2" result="coloredBlur"/>
                <feMerge>
                  <feMergeNode in="coloredBlur"/>
                  <feMergeNode in="SourceGraphic"/>
                </feMerge>
              </filter>
            </defs>
            <CartesianGrid 
              strokeDasharray="3 3" 
              stroke="rgba(255, 255, 255, 0.04)" 
              vertical={false}
            />
            <XAxis
              dataKey="time"
              stroke="transparent"
              tick={{ fill: '#5E6673', fontSize: 10 }}
              tickLine={false}
              axisLine={false}
              interval={Math.floor(chartData.length / 8)}
              dy={8}
            />
            <YAxis
              stroke="transparent"
              tick={{ fill: '#5E6673', fontSize: 10 }}
              tickLine={false}
              axisLine={false}
              domain={calculateYDomain()}
              tickFormatter={(value) =>
                displayMode === 'dollar' ? `$${value.toFixed(0)}` : `${value}%`
              }
              dx={-5}
            />
            <Tooltip content={<CustomTooltip />} />
            <ReferenceLine
              y={displayMode === 'dollar' ? initialBalance : 0}
              stroke="rgba(255, 255, 255, 0.15)"
              strokeDasharray="4 4"
              label={{
                value:
                  displayMode === 'dollar'
                    ? t('initialBalance', language).split(' ')[0]
                    : '0%',
                fill: '#5E6673',
                fontSize: 10,
                position: 'right',
              }}
            />
            <Line
              type="monotone"
              dataKey="value"
              stroke="url(#lineGradient)"
              strokeWidth={2.5}
              dot={false}
              activeDot={{
                r: 5,
                fill: '#FCD535',
                stroke: '#F0B90B',
                strokeWidth: 2,
                filter: 'url(#glow)',
              }}
              connectNulls={true}
            />
          </LineChart>
        </ResponsiveContainer>
      </div>

      {/* Footer Stats */}
      <div
        className="mt-3 grid grid-cols-2 sm:grid-cols-4 gap-2 pt-3 shrink-0"
        style={{ borderTop: '1px solid rgba(255, 255, 255, 0.06)' }}
      >
        <div
          className="p-2.5 rounded-lg transition-all hover:scale-[1.02] cursor-default"
          style={{ 
            background: 'linear-gradient(135deg, rgba(240, 185, 11, 0.08) 0%, rgba(240, 185, 11, 0.03) 100%)',
            border: '1px solid rgba(240, 185, 11, 0.1)',
          }}
        >
          <div
            className="text-[10px] mb-1 uppercase tracking-wider font-medium"
            style={{ color: '#5E6673' }}
          >
            {t('initialBalance', language)}
          </div>
          <div
            className="text-sm font-bold mono"
            style={{ color: '#EAECEF' }}
          >
            {initialBalance.toFixed(2)}
          </div>
        </div>
        <div
          className="p-2.5 rounded-lg transition-all hover:scale-[1.02] cursor-default"
          style={{ 
            background: 'linear-gradient(135deg, rgba(240, 185, 11, 0.08) 0%, rgba(240, 185, 11, 0.03) 100%)',
            border: '1px solid rgba(240, 185, 11, 0.1)',
          }}
        >
          <div
            className="text-[10px] mb-1 uppercase tracking-wider font-medium"
            style={{ color: '#5E6673' }}
          >
            {t('currentEquity', language)}
          </div>
          <div
            className="text-sm font-bold mono"
            style={{ color: '#EAECEF' }}
          >
            {currentValue.raw_equity.toFixed(2)}
          </div>
        </div>
        <div
          className="p-2.5 rounded-lg transition-all hover:scale-[1.02] cursor-default"
          style={{ 
            background: 'linear-gradient(135deg, rgba(240, 185, 11, 0.08) 0%, rgba(240, 185, 11, 0.03) 100%)',
            border: '1px solid rgba(240, 185, 11, 0.1)',
          }}
        >
          <div
            className="text-[10px] mb-1 uppercase tracking-wider font-medium"
            style={{ color: '#5E6673' }}
          >
            {t('historicalCycles', language)}
          </div>
          <div
            className="text-sm font-bold mono"
            style={{ color: '#EAECEF' }}
          >
            {validHistory.length}
          </div>
        </div>
        <div
          className="p-2.5 rounded-lg transition-all hover:scale-[1.02] cursor-default"
          style={{ 
            background: 'linear-gradient(135deg, rgba(240, 185, 11, 0.08) 0%, rgba(240, 185, 11, 0.03) 100%)',
            border: '1px solid rgba(240, 185, 11, 0.1)',
          }}
        >
          <div
            className="text-[10px] mb-1 uppercase tracking-wider font-medium"
            style={{ color: '#5E6673' }}
          >
            {t('displayRange', language)}
          </div>
          <div
            className="text-sm font-bold mono"
            style={{ color: '#EAECEF' }}
          >
            {validHistory.length > MAX_DISPLAY_POINTS
              ? `${t('recent', language)} ${MAX_DISPLAY_POINTS}`
              : t('allData', language)}
          </div>
        </div>
      </div>
    </div>
  )
}
