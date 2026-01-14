/**
 * PendingOrdersTable.tsx - 待执行订单表格组件
 *
 * 显示延迟执行架构中的待执行订单列表
 * 支持按币种分组折叠，只显示置信度最高的订单
 * 新增：显示动态止损实时状态
 */

import { useState, useEffect, useCallback, useRef } from 'react'
import { api } from '../lib/api'
import { usePositions } from '../contexts/PositionContext'
import type { PendingOrder, AdaptiveStopLossRecord } from '../types'
import {
  Loader2,
  AlertCircle,
  CheckCircle2,
  XCircle,
  Clock,
  Zap,
  TrendingUp,
  TrendingDown,
  ChevronDown,
  ChevronRight,
} from 'lucide-react'

interface PendingOrdersTableProps {
  traderId: string
  autoRefresh?: boolean
  refreshInterval?: number
}

interface GroupedOrders {
  [symbol: string]: {
    best: PendingOrder
    all: PendingOrder[]
    count: number
  }
}

export function PendingOrdersTable({
  traderId,
  autoRefresh = true,
  refreshInterval = 30000, // 默认30秒刷新
}: PendingOrdersTableProps) {
  const [orders, setOrders] = useState<PendingOrder[]>([])
  const [adaptiveStopLoss, setAdaptiveStopLoss] = useState<AdaptiveStopLossRecord[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [expandedGroups, setExpandedGroups] = useState<Set<string>>(new Set())
  const { openSymbols } = usePositions() // 获取当前持仓的币种集合
  
  // 防抖定时器引用
  const aslDebounceRef = useRef<NodeJS.Timeout | null>(null)
  // 缓存的 openSymbols
  const cachedSymbolsRef = useRef<Set<string>>(new Set())

  // 获取待执行订单
  const fetchPendingOrders = useCallback(async () => {
    try {
      setLoading(true)
      setError(null)
      const data = await api.getPendingOrders(traderId)
      setOrders(data)
    } catch (err) {
      setError(err instanceof Error ? err.message : '获取订单失败')
    } finally {
      setLoading(false)
    }
  }, [traderId])

  // 获取动态止损状态（带防抖）
  const fetchAdaptiveStopLoss = useCallback(async () => {
    try {
      const response = await api.getAdaptiveStopLoss(traderId)
      if (response.exists && response.records) {
        console.log('[ASL] 原始 ASL 记录数:', response.records.length)
        console.log('[ASL] 当前持仓币种:', Array.from(openSymbols))
        console.log('[ASL] openSymbols.size:', openSymbols.size)
        
        // ✅ 双重过滤保护（后端已验证，前端再次确认）
        // 注意：如果没有持仓数据（openSymbols 为空），显示所有 ASL（不进行过滤）
        const filteredRecords = openSymbols.size > 0
          ? response.records.filter((record) => {
              const isOpen = openSymbols.has(record.symbol)
              console.log(`[ASL] ${record.symbol}: ${isOpen ? '保留' : '过滤'}`)
              return isOpen
            })
          : response.records
        
        console.log('[ASL] 过滤后 ASL 记录数:', filteredRecords.length)
        setAdaptiveStopLoss(filteredRecords)
        // 更新缓存
        cachedSymbolsRef.current = new Set(openSymbols)
      } else {
        setAdaptiveStopLoss([])
      }
    } catch (err) {
      console.warn('获取动态止损状态失败:', err)
      setAdaptiveStopLoss([])
    }
  }, [traderId, openSymbols])

  // 防抖包装的 ASL 获取函数
  const fetchAdaptiveStopLossDebounced = useCallback(() => {
    // 清除之前的定时器
    if (aslDebounceRef.current) {
      clearTimeout(aslDebounceRef.current)
    }

    // 检查 openSymbols 是否真的改变了
    const symbolsChanged = 
      cachedSymbolsRef.current.size !== openSymbols.size ||
      Array.from(openSymbols).some(s => !cachedSymbolsRef.current.has(s))

    if (!symbolsChanged) {
      // 如果符号没有改变，跳过此次更新
      return
    }

    // 延迟 500ms 后执行，防止频繁的过滤操作
    aslDebounceRef.current = setTimeout(() => {
      fetchAdaptiveStopLoss()
    }, 500)
  }, [openSymbols, fetchAdaptiveStopLoss])

  // 初始化和自动刷新
  useEffect(() => {
    const fetchData = async () => {
      await Promise.all([fetchPendingOrders(), fetchAdaptiveStopLoss()])
    }

    fetchData()

    if (!autoRefresh) return

    const interval = setInterval(fetchData, refreshInterval)
    return () => {
      clearInterval(interval)
      if (aslDebounceRef.current) {
        clearTimeout(aslDebounceRef.current)
      }
    }
  }, [traderId, autoRefresh, refreshInterval, fetchPendingOrders, fetchAdaptiveStopLoss])

  // 当 openSymbols 变化时，防抖更新 ASL
  useEffect(() => {
    fetchAdaptiveStopLossDebounced()
  }, [fetchAdaptiveStopLossDebounced])

  // 按币种分组订单
  const groupOrdersBySymbol = (): GroupedOrders => {
    const groups: GroupedOrders = {}

    orders.forEach((order) => {
      if (!groups[order.symbol]) {
        groups[order.symbol] = {
          best: order,
          all: [order],
          count: 1,
        }
      } else {
        groups[order.symbol].all.push(order)
        groups[order.symbol].count++

        // 更新最佳订单（置信度最高，如果相同则取最新）
        const currentBest = groups[order.symbol].best
        if (
          order.confidence > currentBest.confidence ||
          (order.confidence === currentBest.confidence &&
            new Date(order.created_at) > new Date(currentBest.created_at))
        ) {
          groups[order.symbol].best = order
        }
      }
    })

    return groups
  }

  // 切换分组展开/折叠
  const toggleGroup = (symbol: string) => {
    setExpandedGroups((prev) => {
      const newSet = new Set(prev)
      if (newSet.has(symbol)) {
        newSet.delete(symbol)
      } else {
        newSet.add(symbol)
      }
      return newSet
    })
  }

  // 格式化状态显示
  const getStatusBadge = (status: string) => {
    const statusMap = {
      PENDING: {
        icon: <Clock className="w-3.5 h-3.5" />,
        color: '#F0B90B',
        bg: 'rgba(240, 185, 11, 0.15)',
        text: '待执行',
      },
      TRIGGERED: {
        icon: <Zap className="w-3.5 h-3.5" />,
        color: '#F7931A',
        bg: 'rgba(247, 147, 26, 0.15)',
        text: '已触发',
      },
      FILLED: {
        icon: <CheckCircle2 className="w-3.5 h-3.5" />,
        color: '#0ECB81',
        bg: 'rgba(14, 203, 129, 0.15)',
        text: '已成交',
      },
      CANCELLED: {
        icon: <XCircle className="w-3.5 h-3.5" />,
        color: '#848E9C',
        bg: 'rgba(132, 142, 156, 0.15)',
        text: '已取消',
      },
      EXPIRED: {
        icon: <XCircle className="w-3.5 h-3.5" />,
        color: '#F6465D',
        bg: 'rgba(246, 70, 93, 0.15)',
        text: '已过期',
      },
    }

    const badge =
      statusMap[status as keyof typeof statusMap] || statusMap.PENDING
    return (
      <span
        className="inline-flex items-center gap-1 px-1.5 py-0.5 rounded text-[10px] font-medium border whitespace-nowrap"
        style={{
          color: badge.color,
          backgroundColor: badge.bg,
          borderColor: badge.color + '44',
        }}
      >
        {badge.icon}
        <span className="hidden xs:inline">{badge.text}</span>
      </span>
    )
  }

  // 格式化时间
  const formatTime = (timestamp: string) => {
    if (!timestamp) return '-'
    const date = new Date(timestamp)
    return date.toLocaleString('zh-CN', {
      month: '2-digit',
      day: '2-digit',
      hour: '2-digit',
      minute: '2-digit',
    })
  }

  // 计算距离过期时间
  const getExpiresIn = (expiresAt: string) => {
    const now = new Date()
    const expires = new Date(expiresAt)
    const diff = expires.getTime() - now.getTime()

    if (diff < 0) return '已过期'

    const hours = Math.floor(diff / (1000 * 60 * 60))
    const minutes = Math.floor((diff % (1000 * 60 * 60)) / (1000 * 60))

    if (hours > 0) return `${hours}小时${minutes}分钟`
    return `${minutes}分钟`
  }

  // 获取置信度颜色
  const getConfidenceColor = (confidence: number) => {
    if (confidence >= 0.8) return '#0ECB81'
    if (confidence >= 0.6) return '#F0B90B'
    return '#F6465D'
  }

  // 获取方向颜色和图标（基于止损和止盈判断）
  const getDirectionInfo = (stopLoss: number, takeProfit: number) => {
    if (takeProfit > stopLoss) {
      return {
        color: '#0ECB81',
        icon: <TrendingUp className="w-4 h-4" />,
        text: '做多',
      }
    } else {
      return {
        color: '#F6465D',
        icon: <TrendingDown className="w-4 h-4" />,
        text: '做空',
      }
    }
  }

  // 渲染单个订单行
  const renderOrderRow = (order: PendingOrder, isBest: boolean = false) => {
    const directionInfo = getDirectionInfo(order.stop_loss, order.take_profit)
    return (
      <tr
        key={order.id}
        className="transition-all duration-200 hover:bg-white/5"
        style={{
          borderBottom: '1px solid #2B3139',
          background: isBest ? 'rgba(14, 203, 129, 0.03)' : undefined,
        }}
      >
        {/* 状态 */}
        <td className="px-3 py-2" style={{ width: '8%' }}>
          <div className="flex items-center gap-1">
            {isBest && (
              <span
                className="text-[10px] px-1 py-0.5 rounded whitespace-nowrap"
                style={{
                  background: 'rgba(14, 203, 129, 0.2)',
                  color: '#0ECB81',
                  border: '1px solid #0ECB8144',
                }}
              >
                最佳
              </span>
            )}
            {getStatusBadge(order.status)}
          </div>
        </td>

        {/* 方向 */}
        <td className="px-3 py-2" style={{ width: '6%' }}>
          <span
            className="inline-flex items-center gap-1 px-1.5 py-1 rounded text-[10px] font-semibold uppercase whitespace-nowrap"
            style={{
              color: directionInfo.color,
              backgroundColor: directionInfo.color + '15',
              border: `1px solid ${directionInfo.color}44`,
            }}
          >
            {directionInfo.icon}
            {directionInfo.text}
          </span>
        </td>

        {/* 交易对 */}
        <td className="px-3 py-2" style={{ width: '6%' }}>
          <span
            className="font-mono font-semibold text-xs"
            style={{ color: '#EAECEF' }}
          >
            {order.symbol}
          </span>
        </td>

        {/* 目标价 */}
        <td
          className="px-3 py-2 text-center font-mono text-xs"
          style={{ width: '8%', color: '#EAECEF' }}
        >
          {order.target_price.toFixed(4)}
        </td>

        {/* 触发价 */}
        <td
          className="px-3 py-2 text-center font-mono text-xs"
          style={{ width: '8%', color: '#EAECEF' }}
        >
          {order.trigger_price.toFixed(4)}
        </td>

        {/* 仓位 */}
        <td
          className="px-3 py-2 text-center font-mono text-xs"
          style={{ width: '8%', color: '#EAECEF' }}
        >
          ${order.position_size.toFixed(2)}
        </td>

        {/* 杠杆 */}
        <td
          className="px-3 py-2 text-center font-mono text-xs"
          style={{ width: '5%', color: '#EAECEF' }}
        >
          {order.leverage}x
        </td>

        {/* 止盈 */}
        <td
          className="px-3 py-2 text-center font-mono text-xs"
          style={{ width: '8%', color: '#0ECB81', fontWeight: 600 }}
        >
          {order.take_profit.toFixed(4)}
        </td>

        {/* 止损 */}
        <td
          className="px-3 py-2 text-center font-mono text-xs"
          style={{ width: '8%', color: '#F6465D', fontWeight: 600 }}
        >
          {order.stop_loss.toFixed(4)}
        </td>

        {/* 置信度 */}
        <td className="px-3 py-2 text-center font-mono text-xs" style={{ width: '6%', color: '#EAECEF' }}>
          <span
            className="inline-block px-1.5 py-0.5 rounded text-[10px] font-semibold"
            style={{
              color: getConfidenceColor(order.confidence),
              backgroundColor: getConfidenceColor(order.confidence) + '15',
              border: `1px solid ${getConfidenceColor(order.confidence)}44`,
            }}
          >
            {(order.confidence * 100).toFixed(0)}%
          </span>
        </td>

        {/* 创建时间 */}
        <td
          className="px-3 py-2 text-center text-[10px] font-mono"
          style={{ width: '9%', color: '#848E9C' }}
        >
          {formatTime(order.created_at)}
        </td>

        {/* 过期时间 */}
        <td className="px-3 py-2 text-center" style={{ width: '10%' }}>
          <div className="flex flex-col gap-0.5 items-center">
            <span className="text-[10px] font-mono" style={{ color: '#848E9C' }}>
              {formatTime(order.expires_at)}
            </span>
            <span
              className="text-[10px] font-medium font-mono"
              style={{
                color:
                  getExpiresIn(order.expires_at) === '已过期'
                    ? '#F6465D'
                    : '#848E9C',
              }}
            >
              {getExpiresIn(order.expires_at)}
            </span>
          </div>
        </td>
      </tr>
    )
  }

  if (loading && orders.length === 0) {
    return (
      <div
        className="flex items-center justify-center p-12"
        style={{ color: '#848E9C' }}
      >
        <div className="text-center">
          <Loader2
            className="w-8 h-8 animate-spin mx-auto mb-3"
            style={{ color: '#F0B90B' }}
          />
          <p>加载待执行订单中...</p>
        </div>
      </div>
    )
  }

  if (error) {
    return (
      <div
        className="rounded-lg p-6 text-center"
        style={{
          background: 'rgba(246, 70, 93, 0.1)',
          border: '1px solid rgba(246, 70, 93, 0.3)',
          color: '#F6465D',
        }}
      >
        <div className="flex items-center justify-center gap-2 mb-2">
          <AlertCircle className="w-5 h-5" />
          <span className="font-semibold">错误</span>
        </div>
        <p className="text-sm mb-3">{error}</p>
        <button
          onClick={fetchPendingOrders}
          className="text-sm underline hover:opacity-80 transition-opacity"
        >
          重试
        </button>
      </div>
    )
  }

  if (orders.length === 0) {
    return (
      <div
        className="rounded-lg p-12 text-center"
        style={{
          background: 'linear-gradient(135deg, #1E2329 0%, #181C21 100%)',
          border: '1px solid #2B3139',
        }}
      >
        <div className="text-6xl mb-4 opacity-30">📋</div>
        <p className="text-lg font-semibold mb-1" style={{ color: '#EAECEF' }}>
          暂无待执行订单
        </p>
        <p className="text-sm" style={{ color: '#848E9C' }}>
          AI分析后创建的订单将显示在这里
        </p>
      </div>
    )
  }

  // 分组后的订单
  const groupedOrders = groupOrdersBySymbol()
  const hasDuplicates = Object.values(groupedOrders).some((g) => g.count > 1)

  // 🔧 修复：更准确的统计逻辑
  // 只统计真正"活跃"的订单（PENDING 未过期 + TRIGGERED）
  const activeOrders = orders.filter(o => {
    if (o.status === 'PENDING') {
      // PENDING 订单必须未过期
      return new Date(o.expires_at) > new Date()
    }
    // TRIGGERED 订单始终算作活跃（等待执行）
    return o.status === 'TRIGGERED'
  })

  // 已成交订单（FILLED）
  const filledOrders = orders.filter(o => o.status === 'FILLED')

  // 已取消/过期订单
  const cancelledOrders = orders.filter(o => 
    o.status === 'CANCELLED' || o.status === 'EXPIRED'
  )

  // 总订单数（所有状态）
  const totalOrders = orders.length

  // 获取动态止损状态信息
  const getAdaptiveStopLossInfo = (symbol: string) => {
    const record = adaptiveStopLoss.find(r => r.symbol === symbol)
    if (!record) return null

    const progress = (record.time_progression * 100).toFixed(1)
    const elapsedMinutes = (record.elapsed_seconds / 60).toFixed(1)
    
    return {
      ...record,
      progress,
      elapsedMinutes,
      isMoving: record.time_progression < 1.0 || record.is_in_profit,
      direction: record.is_in_profit ? '盈利追踪' : '回归入场价',
    }
  }

  return (
    <div className="space-y-4">
      {/* 动态止损实时状态 - 新增 */}
      {adaptiveStopLoss.length > 0 && (
        <div className="rounded-lg p-4 border border-[#2B3139]" style={{
          background: 'linear-gradient(135deg, #1E2329 0%, #181C21 100%)',
        }}>
          <div className="flex items-center gap-2 mb-3">
            <span className="text-lg">🛡️</span>
            <span className="font-semibold text-sm" style={{ color: '#EAECEF' }}>
              动态止损实时状态
            </span>
            <span className="text-xs px-2 py-0.5 rounded" style={{
              background: 'rgba(14, 203, 129, 0.2)',
              color: '#0ECB81',
              border: '1px solid #0ECB8144',
            }}>
              {adaptiveStopLoss.length} 个活跃
            </span>
          </div>
          
          <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-3">
            {adaptiveStopLoss.map(record => {
              const info = getAdaptiveStopLossInfo(record.symbol)
              if (!info) return null

              return (
                <div key={record.id} className="rounded p-3 border border-[#2B3139]" style={{
                  background: 'rgba(11, 14, 17, 0.5)',
                }}>
                  <div className="flex items-center justify-between mb-2">
                    <span className="font-mono font-bold text-sm" style={{ color: '#EAECEF' }}>
                      {record.symbol}
                    </span>
                    <span className="text-xs px-1.5 py-0.5 rounded" style={{
                      background: info.isMoving ? 'rgba(14, 203, 129, 0.2)' : 'rgba(240, 185, 11, 0.2)',
                      color: info.isMoving ? '#0ECB81' : '#F0B90B',
                      border: `1px solid ${info.isMoving ? '#0ECB8144' : '#F0B90B44'}`,
                    }}>
                      {info.direction}
                    </span>
                  </div>

                  <div className="space-y-1 text-xs">
                    <div className="flex justify-between items-center">
                      <span style={{ color: '#848E9C' }}>当前止损:</span>
                      <span className="font-mono" style={{ color: '#F6465D', fontWeight: 600 }}>
                        {record.current_stop_loss.toFixed(4)}
                      </span>
                    </div>
                    <div className="flex justify-between items-center">
                      <span style={{ color: '#848E9C' }}>入场价:</span>
                      <span className="font-mono" style={{ color: '#EAECEF' }}>
                        {record.entry_price.toFixed(4)}
                      </span>
                    </div>
                    <div className="flex justify-between items-center">
                      <span style={{ color: '#848E9C' }}>当前价格:</span>
                      <span className="font-mono" style={{ color: '#EAECEF' }}>
                        {record.current_price.toFixed(4)}
                      </span>
                    </div>
                    <div className="flex justify-between items-center">
                      <span style={{ color: '#848E9C' }}>止盈:</span>
                      <span className="font-mono" style={{ color: '#0ECB81', fontWeight: 600 }}>
                        {record.take_profit.toFixed(4)}
                      </span>
                    </div>
                    
                    {/* 进度条 */}
                    <div className="mt-2">
                      <div className="flex justify-between text-[10px] mb-1 items-center">
                        <span style={{ color: '#848E9C' }}>时间进度</span>
                        <span style={{ color: '#EAECEF' }}>{info.progress}%</span>
                      </div>
                      <div className="w-full h-1.5 rounded-full overflow-hidden" style={{
                        background: 'rgba(132, 142, 156, 0.3)',
                      }}>
                        <div className="h-full rounded-full transition-all duration-300" style={{
                          width: `${info.progress}%`,
                          background: info.isMoving 
                            ? 'linear-gradient(90deg, #0ECB81, #F0B90B)' 
                            : 'linear-gradient(90deg, #F0B90B, #F6465D)',
                        }} />
                      </div>
                    </div>

                    {/* 状态详情 */}
                    <div className="flex justify-between items-center pt-1 mt-1 border-t border-[#2B3139]">
                      <span style={{ color: '#848E9C' }}>已运行</span>
                      <span className="font-mono" style={{ color: '#EAECEF' }}>
                        {info.elapsedMinutes} 分钟
                      </span>
                    </div>
                    {record.is_in_profit && (
                      <div className="flex justify-between items-center">
                        <span style={{ color: '#848E9C' }}>盈利距离</span>
                        <span className="font-mono" style={{ color: '#0ECB81' }}>
                          {record.profit_distance.toFixed(4)}
                        </span>
                      </div>
                    )}
                  </div>
                </div>
              )
            })}
          </div>
        </div>
      )}

      {/* 统计信息 */}
      <div className="grid grid-cols-2 sm:grid-cols-3 lg:grid-cols-6 gap-2 md:gap-3">
        <StatCard
          label="总订单数"
          value={totalOrders}
          color="#EAECEF"
          icon="📊"
          tooltip="所有状态订单总数"
        />
        <StatCard
          label="币种组数"
          value={Object.keys(groupedOrders).length}
          color="#EAECEF"
          icon="🏷️"
          tooltip="不同交易对数量"
        />
        <StatCard
          label="重复币种"
          value={Object.values(groupedOrders).filter((g) => g.count > 1).length}
          color="#F0B90B"
          icon="🔄"
          tooltip="有重复订单的交易对"
        />
        <StatCard
          label="活跃订单"
          value={activeOrders.length}
          color="#F0B90B"
          icon="⏰"
          tooltip="PENDING(未过期) + TRIGGERED"
        />
        <StatCard
          label="已成交"
          value={filledOrders.length}
          color="#0ECB81"
          icon="✅"
          tooltip="FILLED 状态订单"
        />
        <StatCard
          label="已取消/过期"
          value={cancelledOrders.length}
          color="#848E9C"
          icon="❌"
          tooltip="CANCELLED + EXPIRED"
        />
      </div>

      {/* 分组说明 - 只有在有重复时显示 */}
      {hasDuplicates && (
        <div
          className="rounded-lg p-3"
          style={{
            background: 'rgba(240, 185, 11, 0.1)',
            border: '1px solid rgba(240, 185, 11, 0.3)',
          }}
        >
          <div
            className="flex items-center gap-2 text-sm"
            style={{ color: '#F0B90B' }}
          >
            <AlertCircle className="w-4 h-4" />
            <span className="font-semibold">发现重复订单</span>
            <span className="opacity-70">
              系统已按币种分组，只显示置信度最高的订单，点击展开可查看所有
            </span>
          </div>
        </div>
      )}

      {/* 分组订单表格 - 深色主题 */}
      <div
        className="rounded-lg overflow-hidden border border-[#2B3139]"
        style={{
          background: 'linear-gradient(135deg, #1E2329 0%, #181C21 100%)',
        }}
      >
        <div className="overflow-x-auto">
          <table className="w-full text-sm border-collapse">
            {/* 表头 - 固定样式 */}
            <thead style={{ background: '#0B0E11' }}>
              <tr className="border-b border-[#2B3139]">
                <th className="px-3 py-3 text-left text-[10px] font-semibold uppercase tracking-wider" style={{ color: '#848E9C', width: '8%' }}>状态</th>
                <th className="px-3 py-3 text-left text-[10px] font-semibold uppercase tracking-wider" style={{ color: '#848E9C', width: '6%' }}>方向</th>
                <th className="px-3 py-3 text-left text-[10px] font-semibold uppercase tracking-wider" style={{ color: '#848E9C', width: '6%' }}>交易对</th>
                <th className="px-3 py-3 text-center text-[10px] font-semibold uppercase tracking-wider" style={{ color: '#848E9C', width: '8%' }}>目标价</th>
                <th className="px-3 py-3 text-center text-[10px] font-semibold uppercase tracking-wider" style={{ color: '#848E9C', width: '8%' }}>触发价</th>
                <th className="px-3 py-3 text-center text-[10px] font-semibold uppercase tracking-wider" style={{ color: '#848E9C', width: '8%' }}>仓位</th>
                <th className="px-3 py-3 text-center text-[10px] font-semibold uppercase tracking-wider" style={{ color: '#848E9C', width: '5%' }}>杠杆</th>
                <th className="px-3 py-3 text-center text-[10px] font-semibold uppercase tracking-wider" style={{ color: '#848E9C', width: '8%' }}>止盈</th>
                <th className="px-3 py-3 text-center text-[10px] font-semibold uppercase tracking-wider" style={{ color: '#848E9C', width: '8%' }}>止损</th>
                <th className="px-3 py-3 text-center text-[10px] font-semibold uppercase tracking-wider" style={{ color: '#848E9C', width: '6%' }}>置信度</th>
                <th className="px-3 py-3 text-center text-[10px] font-semibold uppercase tracking-wider" style={{ color: '#848E9C', width: '9%' }}>创建时间</th>
                <th className="px-3 py-3 text-center text-[10px] font-semibold uppercase tracking-wider" style={{ color: '#848E9C', width: '10%' }}>过期时间</th>
              </tr>
            </thead>
            <tbody>
              {Object.entries(groupedOrders).map(([symbol, group]) => {
                const isExpanded = expandedGroups.has(symbol)
                const hasDuplicates = group.count > 1
                const showBestOnly = hasDuplicates && !isExpanded

                return (
                  <tr key={symbol} className="border-b border-[#2B3139] last:border-b-0">
                    <td colSpan={12} className="p-0">
                      {/* 分组标题行 */}
                      <div
                        className="flex items-center gap-2 px-3 py-3 cursor-pointer hover:bg-white/5 transition-colors"
                        onClick={() => toggleGroup(symbol)}
                        style={{
                          background: hasDuplicates
                            ? 'rgba(240, 185, 11, 0.05)'
                            : 'transparent',
                        }}
                      >
                        {/* 展开图标 */}
                        <div className="flex-shrink-0 w-5 flex items-center justify-center">
                          {hasDuplicates ? (
                            isExpanded ? (
                              <ChevronDown className="w-4 h-4" style={{ color: '#F0B90B' }} />
                            ) : (
                              <ChevronRight className="w-4 h-4" style={{ color: '#F0B90B' }} />
                            )
                          ) : (
                            <div className="w-4 h-4" />
                          )}
                        </div>

                        {/* 币种信息 */}
                        <div className="flex items-center gap-2 flex-1 min-w-0">
                          <span className="font-mono font-bold text-sm truncate" style={{ color: '#EAECEF' }}>
                            {symbol}
                          </span>
                          {hasDuplicates && (
                            <span className="text-[10px] px-1.5 py-0.5 rounded whitespace-nowrap" style={{
                              background: 'rgba(240, 185, 11, 0.2)',
                              color: '#F0B90B',
                              border: '1px solid #F0B90B44',
                            }}>
                              {group.count} 个订单
                            </span>
                          )}
                          {showBestOnly && (
                            <span className="text-[10px] hidden sm:inline" style={{ color: '#848E9C' }}>
                              只显示置信度最高的订单
                            </span>
                          )}
                        </div>

                        {/* 最佳订单快速信息 */}
                        <div className="flex items-center gap-3 text-[10px] hidden lg:flex">
                          <span style={{ color: '#848E9C' }}>
                            置信度:
                            <span className="ml-1 font-semibold" style={{ color: getConfidenceColor(group.best.confidence) }}>
                              {(group.best.confidence * 100).toFixed(0)}%
                            </span>
                          </span>
                          <span style={{ color: '#848E9C' }}>
                            仓位:
                            <span className="ml-1 font-mono" style={{ color: '#EAECEF' }}>
                              ${group.best.position_size.toFixed(2)}
                            </span>
                          </span>
                          <span style={{ color: '#848E9C' }}>
                            杠杆:
                            <span className="ml-1 font-mono" style={{ color: '#EAECEF' }}>
                              {group.best.leverage}x
                            </span>
                          </span>
                        </div>
                      </div>

                      {/* 订单详情行 */}
                      {isExpanded && (
                        <div className="bg-[#0B0E11]/50">
                          {/* 最佳订单 */}
                          {renderOrderRow(group.best, true)}

                          {/* 其他订单 */}
                          {group.all
                            .filter((o) => o.id !== group.best.id)
                            .map((order) => renderOrderRow(order, false))}
                        </div>
                      )}
                    </td>
                  </tr>
                )
              })}
            </tbody>
          </table>
        </div>
      </div>
    </div>
  )
}

// 统计卡片组件 - 深色主题
function StatCard({
  label,
  value,
  color,
  icon,
  tooltip,
}: {
  label: string
  value: number
  color: string
  icon: string
  tooltip?: string
}) {
  return (
    <div
      className="rounded-lg p-4 transition-all duration-200 hover:scale-[1.02]"
      style={{
        background: 'linear-gradient(135deg, #1E2329 0%, #181C21 100%)',
        border: '1px solid #2B3139',
        boxShadow: '0 4px 12px rgba(0, 0, 0, 0.2)',
      }}
      title={tooltip}
    >
      <div className="flex items-center gap-2 mb-2">
        <span className="text-lg">{icon}</span>
        <span className="text-xs" style={{ color: '#848E9C' }}>
          {label}
        </span>
      </div>
      <div className="text-2xl font-bold font-mono" style={{ color }}>
        {value}
      </div>
    </div>
  )
}

export default PendingOrdersTable
