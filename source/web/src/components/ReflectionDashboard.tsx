/**
 * ReflectionDashboard.tsx - 完整的反思系统前端组件（增强版）
 *
 * 支持可视化编辑功能
 * - 创建新反思
 * - 编辑现有反思
 * - 删除反思
 * - 编辑待处理调整
 *
 * 使用方式:
 * import { ReflectionDashboard } from '@/components/ReflectionDashboard'
 * <ReflectionDashboard traderID="trader_001" />
 */

import { useState, useEffect } from 'react'
import { ReflectionEditForm } from './ReflectionEditForm'

interface Reflection {
  id: string
  traderID: string
  analysisType: 'performance' | 'risk' | 'strategy'
  findings: string
  timestamp?: string
  createdAt?: string
  severity: 'info' | 'warning' | 'error'
}

interface Adjustment {
  id: string
  traderID: string
  suggestedAction: string
  reasoning: string
  priority: 'low' | 'medium' | 'high'
  status: 'pending' | 'applied' | 'rejected'
  createdAt: string
}

interface ReflectionStats {
  totalReflections: number
  averageSeverity: string
  lastReflectionTime: string
  findingsByType: {
    performance: number
    risk: number
    strategy: number
  }
}

interface ReflectionDashboardProps {
  traderID: string
  autoRefresh?: boolean
  refreshInterval?: number
}

export function ReflectionDashboard({
  traderID,
  autoRefresh = true,
  refreshInterval = 60000,
}: ReflectionDashboardProps) {
  const [reflections, setReflections] = useState<Reflection[]>([])
  const [adjustments, setAdjustments] = useState<Adjustment[]>([])
  const [stats, setStats] = useState<ReflectionStats | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [activeTab, setActiveTab] = useState<
    'reflections' | 'adjustments' | 'stats'
  >('reflections')

  // 编辑模态框状态
  const [showEditForm, setShowEditForm] = useState(false)
  const [editMode, setEditMode] = useState<'create' | 'edit'>('create')
  const [selectedReflection, setSelectedReflection] =
    useState<Reflection | null>(null)
  const [selectedAdjustment, setSelectedAdjustment] =
    useState<Adjustment | null>(null)

  // 获取数据
  const fetchData = async () => {
    try {
      setLoading(true)
      setError(null)

      const [reflectionsRes, adjustmentsRes, statsRes] = await Promise.all([
        fetch(`/api/reflection/${traderID}/recent?limit=30`),
        fetch(`/api/adjustment/${traderID}/pending`),
        fetch(`/api/reflection/${traderID}/stats`),
      ])

      if (!reflectionsRes.ok || !adjustmentsRes.ok || !statsRes.ok) {
        throw new Error('Failed to fetch data')
      }

      const reflectionsData = await reflectionsRes.json()
      const adjustmentsData = await adjustmentsRes.json()
      const statsData = await statsRes.json()

      const rawReflections = reflectionsData.data || []
      // 兼容后端 ReflectionRecord 字段到前端展示结构
      const mappedReflections: Reflection[] = rawReflections.map((r: any) => {
        const aiReflection: string = r.ai_reflection || r.AIReflection || r.findings || ''
        // 从 AIReflection 前缀解析分析类型，如 "[performance] ..."
        const typeMatch = aiReflection.match(/^\[(performance|risk|strategy)\]/i)
        const analysisType: 'performance' | 'risk' | 'strategy' =
          (typeMatch ? typeMatch[1].toLowerCase() : 'performance') as
            'performance' | 'risk' | 'strategy'
        return {
          id: r.id || r.ID,
          traderID: r.trader_id || r.TraderID || traderID,
          analysisType,
          findings: aiReflection.replace(/^\[[^\]]+\]\s*/i, ''),
          severity: (r.severity || 'info') as 'info' | 'warning' | 'error',
          createdAt: r.created_at || r.CreatedAt,
          timestamp: r.reflection_time || r.ReflectionTime,
        }
      })

      setReflections(mappedReflections)
      setAdjustments(adjustmentsData.data || [])
      // 兼容统计数据结构，缺失时提供兜底
      const statsRaw = statsData?.data || statsData || null
      const fallbackStats: ReflectionStats | null = statsRaw
        ? {
            totalReflections:
              statsRaw.total_reflections ?? mappedReflections.length ?? 0,
            averageSeverity: 'info',
            lastReflectionTime:
              (mappedReflections[0]?.timestamp as string) ||
              (mappedReflections[0]?.createdAt as string) ||
              new Date().toISOString(),
            findingsByType: {
              performance: 0,
              risk: 0,
              strategy: 0,
            },
          }
        : null
      setStats(fallbackStats)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Unknown error')
    } finally {
      setLoading(false)
    }
  }

  // 初始化和自动刷新
  useEffect(() => {
    fetchData()

    if (!autoRefresh) return

    const interval = setInterval(fetchData, refreshInterval)
    return () => clearInterval(interval)
  }, [traderID, autoRefresh, refreshInterval])

  // 应用调整
  const handleApplyAdjustment = async (adjustmentID: string) => {
    try {
      const response = await fetch(`/api/adjustment/${adjustmentID}/apply`, {
        method: 'POST',
      })
      if (response.ok) {
        await fetchData()
      }
    } catch (err) {
      console.error('Failed to apply adjustment:', err)
    }
  }

  // 拒绝调整
  const handleRejectAdjustment = async (adjustmentID: string) => {
    try {
      const response = await fetch(`/api/adjustment/${adjustmentID}/reject`, {
        method: 'POST',
      })
      if (response.ok) {
        await fetchData()
      }
    } catch (err) {
      console.error('Failed to reject adjustment:', err)
    }
  }

  // 删除反思
  const handleDeleteReflection = async (reflectionID: string) => {
    if (!confirm('确定要删除这条反思吗？')) return

    try {
      const response = await fetch(`/api/reflection/id/${reflectionID}`, {
        method: 'DELETE',
      })
      if (response.ok) {
        await fetchData()
      }
    } catch (err) {
      console.error('Failed to delete reflection:', err)
    }
  }

  // 打开编辑反思表单
  const openEditReflection = (reflection: Reflection) => {
    setSelectedReflection(reflection)
    setSelectedAdjustment(null)
    setEditMode('edit')
    setShowEditForm(true)
  }

  // 打开编辑调整表单
  const openEditAdjustment = (adjustment: Adjustment) => {
    setSelectedAdjustment(adjustment)
    setSelectedReflection(null)
    setEditMode('edit')
    setShowEditForm(true)
  }

  // 打开创建新反思表单
  const openCreateReflection = () => {
    setSelectedReflection(null)
    setSelectedAdjustment(null)
    setEditMode('create')
    setShowEditForm(true)
  }

  // 触发分析
  const handleTriggerAnalysis = async (
    type: 'performance' | 'risk' | 'strategy'
  ) => {
    try {
      const response = await fetch(`/api/reflection/${traderID}/analyze`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ analysisType: type }),
      })
      if (response.ok) {
        await fetchData()
      }
    } catch (err) {
      console.error('Failed to trigger analysis:', err)
    }
  }

  if (loading && reflections.length === 0) {
    return (
      <div className="flex items-center justify-center p-8">
        <div className="text-center">
          <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-blue-600 mx-auto mb-4"></div>
          <p className="text-gray-600">加载反思数据中...</p>
        </div>
      </div>
    )
  }

  return (
    <div className="space-y-6 p-6 max-w-6xl mx-auto">
      {/* 标题和刷新按钮 */}
      <div className="flex items-center justify-between">
        <h1 className="text-3xl font-bold text-gray-900">📊 系统反思</h1>
        <div className="flex gap-2">
          <button
            onClick={() => fetchData()}
            className="px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 transition"
          >
            🔄 刷新
          </button>
        </div>
      </div>

      {error && (
        <div className="p-4 bg-red-50 border border-red-200 rounded-lg text-red-800">
          错误: {error}
        </div>
      )}

      {/* 统计卡片 */}
      {stats && (
        <div className="grid grid-cols-1 md:grid-cols-4 gap-4">
          <div className="bg-white p-4 rounded-lg shadow border-l-4 border-blue-500">
            <p className="text-gray-600 text-sm">总反思次数</p>
            <p className="text-3xl font-bold text-gray-900">
              {stats.totalReflections}
            </p>
          </div>
          <div className="bg-white p-4 rounded-lg shadow border-l-4 border-green-500">
            <p className="text-gray-600 text-sm">性能分析</p>
            <p className="text-3xl font-bold text-gray-900">
              {stats.findingsByType.performance}
            </p>
          </div>
          <div className="bg-white p-4 rounded-lg shadow border-l-4 border-yellow-500">
            <p className="text-gray-600 text-sm">风险分析</p>
            <p className="text-3xl font-bold text-gray-900">
              {stats.findingsByType.risk}
            </p>
          </div>
          <div className="bg-white p-4 rounded-lg shadow border-l-4 border-purple-500">
            <p className="text-gray-600 text-sm">策略分析</p>
            <p className="text-3xl font-bold text-gray-900">
              {stats.findingsByType.strategy}
            </p>
          </div>
        </div>
      )}

      {/* 快速操作 */}
      <div className="bg-white p-4 rounded-lg shadow">
        <div className="flex items-center justify-between mb-3">
          <h3 className="font-semibold text-gray-900">快速操作</h3>
        </div>
        <div className="flex flex-wrap gap-2">
          <button
            onClick={() => handleTriggerAnalysis('performance')}
            className="px-4 py-2 bg-blue-100 text-blue-700 rounded-lg hover:bg-blue-200 transition font-medium"
          >
            📊 性能分析
          </button>
          <button
            onClick={() => handleTriggerAnalysis('risk')}
            className="px-4 py-2 bg-yellow-100 text-yellow-700 rounded-lg hover:bg-yellow-200 transition font-medium"
          >
            ⚠️ 风险分析
          </button>
          <button
            onClick={() => handleTriggerAnalysis('strategy')}
            className="px-4 py-2 bg-purple-100 text-purple-700 rounded-lg hover:bg-purple-200 transition font-medium"
          >
            🎯 策略分析
          </button>
          <button
            onClick={openCreateReflection}
            className="px-4 py-2 bg-green-100 text-green-700 rounded-lg hover:bg-green-200 transition font-medium ml-auto"
          >
            ✏️ 新建反思
          </button>
        </div>
      </div>

      {/* 标签页导航 */}
      <div className="flex border-b border-gray-200">
        <button
          onClick={() => setActiveTab('reflections')}
          className={`px-6 py-2 font-medium border-b-2 transition ${
            activeTab === 'reflections'
              ? 'border-blue-600 text-blue-600'
              : 'border-transparent text-gray-600 hover:text-gray-900'
          }`}
        >
          💭 反思记录 ({reflections.length})
        </button>
        <button
          onClick={() => setActiveTab('adjustments')}
          className={`px-6 py-2 font-medium border-b-2 transition ${
            activeTab === 'adjustments'
              ? 'border-blue-600 text-blue-600'
              : 'border-transparent text-gray-600 hover:text-gray-900'
          }`}
        >
          ⚡ 待处理调整 (
          {adjustments.filter((a) => a.status === 'pending').length})
        </button>
        <button
          onClick={() => setActiveTab('stats')}
          className={`px-6 py-2 font-medium border-b-2 transition ${
            activeTab === 'stats'
              ? 'border-blue-600 text-blue-600'
              : 'border-transparent text-gray-600 hover:text-gray-900'
          }`}
        >
          📈 详细统计
        </button>
      </div>

      {/* 反思记录标签页 */}
      {activeTab === 'reflections' && (
        <div className="space-y-4">
          {reflections.length === 0 ? (
            <div className="text-center py-8 text-gray-500">
              暂无反思记录，点击上面的按钮触发分析或新建反思
            </div>
          ) : (
            reflections.map((reflection) => (
              <ReflectionCard
                key={reflection.id}
                reflection={reflection}
                onEdit={() => openEditReflection(reflection)}
                onDelete={() => handleDeleteReflection(reflection.id)}
              />
            ))
          )}
        </div>
      )}

      {/* 待处理调整标签页 */}
      {activeTab === 'adjustments' && (
        <div className="space-y-4">
          {adjustments.filter((a) => a.status === 'pending').length === 0 ? (
            <div className="text-center py-8 text-gray-500">
              暂无待处理调整建议
            </div>
          ) : (
            adjustments
              .filter((a) => a.status === 'pending')
              .map((adjustment) => (
                <AdjustmentCard
                  key={adjustment.id}
                  adjustment={adjustment}
                  onApply={() => handleApplyAdjustment(adjustment.id)}
                  onReject={() => handleRejectAdjustment(adjustment.id)}
                  onEdit={() => openEditAdjustment(adjustment)}
                />
              ))
          )}
        </div>
      )}

      {/* 详细统计标签页 */}
      {activeTab === 'stats' && stats && (
        <div className="bg-white p-6 rounded-lg shadow space-y-6">
          <div>
            <h3 className="font-semibold text-gray-900 mb-3">分析类型分布</h3>
            <div className="grid grid-cols-3 gap-4">
              <div className="text-center p-4 bg-blue-50 rounded-lg">
                <p className="text-gray-600 text-sm">性能分析</p>
                <p className="text-2xl font-bold text-blue-600">
                  {stats.findingsByType.performance}
                </p>
              </div>
              <div className="text-center p-4 bg-yellow-50 rounded-lg">
                <p className="text-gray-600 text-sm">风险分析</p>
                <p className="text-2xl font-bold text-yellow-600">
                  {stats.findingsByType.risk}
                </p>
              </div>
              <div className="text-center p-4 bg-purple-50 rounded-lg">
                <p className="text-gray-600 text-sm">策略分析</p>
                <p className="text-2xl font-bold text-purple-600">
                  {stats.findingsByType.strategy}
                </p>
              </div>
            </div>
          </div>
          <div>
            <h3 className="font-semibold text-gray-900 mb-2">其他信息</h3>
            <div className="space-y-2 text-sm text-gray-600">
              <p>
                📊 总反思次数:{' '}
                <span className="font-semibold text-gray-900">
                  {stats.totalReflections}
                </span>
              </p>
              <p>
                ⏰ 最后反思:{' '}
                <span className="font-semibold text-gray-900">
                  {new Date(stats.lastReflectionTime).toLocaleString()}
                </span>
              </p>
            </div>
          </div>
        </div>
      )}

      {/* 编辑表单模态框 */}
      {showEditForm && (
          <ReflectionEditForm
          traderID={traderID}
          mode={editMode}
          reflectionID={selectedReflection?.id}
          adjustmentID={selectedAdjustment?.id}
            initialData={selectedReflection || selectedAdjustment || undefined}
          onSuccess={() => {
            setShowEditForm(false)
            fetchData()
          }}
          onCancel={() => setShowEditForm(false)}
        />
      )}
    </div>
  )
}

// 反思卡片组件
function ReflectionCard({
  reflection,
  onEdit,
  onDelete,
}: {
  reflection: Reflection
  onEdit: () => void
  onDelete: () => void
}) {
  const iconMap = {
    performance: '📊',
    risk: '⚠️',
    strategy: '🎯',
  }

  const bgColorMap = {
    info: 'bg-blue-50 border-blue-200',
    warning: 'bg-yellow-50 border-yellow-200',
    error: 'bg-red-50 border-red-200',
  }

  const textColorMap = {
    info: 'text-blue-800',
    warning: 'text-yellow-800',
    error: 'text-red-800',
  }

  const severityBadgeMap = {
    info: 'bg-blue-100 text-blue-800',
    warning: 'bg-yellow-100 text-yellow-800',
    error: 'bg-red-100 text-red-800',
  }

  return (
    <div className={`p-4 rounded-lg border ${bgColorMap[reflection.severity]}`}>
      <div className="flex items-start justify-between gap-4">
        <div className="flex items-start gap-3 flex-1">
          <span className="text-2xl">{iconMap[reflection.analysisType]}</span>
          <div className="flex-1">
            <h3
              className={`font-semibold ${textColorMap[reflection.severity]}`}
            >
              {reflection.analysisType.charAt(0).toUpperCase() +
                reflection.analysisType.slice(1)}{' '}
              分析
            </h3>
            <p className="text-gray-700 mt-2">{reflection.findings}</p>
            <time className="text-xs text-gray-500 mt-2 block">
              {new Date(
                reflection.createdAt || reflection.timestamp || ''
              ).toLocaleString()}
            </time>
          </div>
        </div>
        <div className="flex items-center gap-2">
          <span
            className={`px-3 py-1 rounded text-sm font-medium whitespace-nowrap ${severityBadgeMap[reflection.severity]}`}
          >
            {reflection.severity.toUpperCase()}
          </span>
          <button
            onClick={onEdit}
            className="px-3 py-1 bg-blue-100 text-blue-700 rounded hover:bg-blue-200 transition text-sm font-medium"
            title="编辑反思"
          >
            ✏️
          </button>
          <button
            onClick={onDelete}
            className="px-3 py-1 bg-red-100 text-red-700 rounded hover:bg-red-200 transition text-sm font-medium"
            title="删除反思"
          >
            🗑️
          </button>
        </div>
      </div>
    </div>
  )
}

// 调整建议卡片组件
function AdjustmentCard({
  adjustment,
  onApply,
  onReject,
  onEdit,
}: {
  adjustment: Adjustment
  onApply: () => void
  onReject: () => void
  onEdit: () => void
}) {
  const priorityColorMap = {
    low: 'bg-green-50 border-green-200',
    medium: 'bg-yellow-50 border-yellow-200',
    high: 'bg-red-50 border-red-200',
  }

  const priorityBadgeMap = {
    low: 'bg-green-100 text-green-800',
    medium: 'bg-yellow-100 text-yellow-800',
    high: 'bg-red-100 text-red-800',
  }

  return (
    <div
      className={`p-4 rounded-lg border ${priorityColorMap[adjustment.priority]}`}
    >
      <div className="flex items-start justify-between mb-3">
        <h3 className="font-semibold text-gray-900 flex-1">
          {adjustment.suggestedAction.replace(/_/g, ' ').toUpperCase()}
        </h3>
        <span
          className={`px-3 py-1 rounded text-sm font-medium whitespace-nowrap ${priorityBadgeMap[adjustment.priority]}`}
        >
          {adjustment.priority.toUpperCase()}
        </span>
      </div>
      <p className="text-gray-700 mb-4">{adjustment.reasoning}</p>
      <p className="text-xs text-gray-500 mb-4">
        建议时间: {new Date(adjustment.createdAt).toLocaleString()}
      </p>
      <div className="flex gap-2">
        <button
          onClick={onApply}
          className="flex-1 px-4 py-2 bg-green-600 text-white rounded-lg hover:bg-green-700 transition font-medium"
        >
          ✓ 应用建议
        </button>
        <button
          onClick={onReject}
          className="flex-1 px-4 py-2 bg-red-600 text-white rounded-lg hover:bg-red-700 transition font-medium"
        >
          ✕ 拒绝建议
        </button>
        <button
          onClick={onEdit}
          className="px-4 py-2 bg-blue-100 text-blue-700 rounded-lg hover:bg-blue-200 transition font-medium"
          title="编辑调整"
        >
          ✏️
        </button>
      </div>
    </div>
  )
}

// 在最后添加编辑表单和导出
  export function ReflectionDashboardWithEditor({
  traderID,
  autoRefresh = true,
  refreshInterval = 60000,
}: ReflectionDashboardProps) {
    const [showEditForm, setShowEditForm] = useState(false)
    const [editMode] = useState<'create' | 'edit'>('create')
    const [selectedReflection] = useState<Reflection | null>(null)
    const [selectedAdjustment] = useState<Adjustment | null>(null)

  const dashboard = (
    <ReflectionDashboard
      traderID={traderID}
      autoRefresh={autoRefresh}
      refreshInterval={refreshInterval}
    />
  )

    const handleEditForm = () => {
    setShowEditForm(false)
    // 表单会自动刷新数据
  }

  return (
    <>
      {dashboard}
      {showEditForm && (
          <ReflectionEditForm
          traderID={traderID}
          mode={editMode}
          reflectionID={selectedReflection?.id}
          adjustmentID={selectedAdjustment?.id}
            initialData={selectedReflection || selectedAdjustment || undefined}
          onSuccess={handleEditForm}
          onCancel={() => setShowEditForm(false)}
        />
      )}
    </>
  )
}

export default ReflectionDashboard
