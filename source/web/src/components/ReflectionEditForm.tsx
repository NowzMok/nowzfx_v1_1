/**
 * ReflectionEditForm.tsx - 反思系统编辑表单组件
 *
 * 支持：
 * - 创建新反思
 * - 编辑现有反思
 * - 编辑待处理调整
 * - 可视化表单输入
 *
 * 使用方式:
 * <ReflectionEditForm
 *   traderID="trader_001"
 *   mode="create" | "edit"
 *   reflectionID={id}
 *   onSuccess={() => fetchData()}
 *   onCancel={() => setShowForm(false)}
 * />
 */

import { useState } from 'react'

interface Reflection {
  id: string
  traderID: string
  analysisType: 'performance' | 'risk' | 'strategy'
  findings: string
  severity: 'info' | 'warning' | 'error'
  timestamp?: string
  createdAt?: string
}

interface Adjustment {
  id: string
  suggestedAction: string
  reasoning: string
  priority: 'low' | 'medium' | 'high'
  status: string
}

interface ReflectionEditFormProps {
  traderID: string
  mode: 'create' | 'edit'
  reflectionID?: string
  adjustmentID?: string
  initialData?: Reflection | Adjustment
  onSuccess: (data: any) => void
  onCancel: () => void
}

export function ReflectionEditForm({
  traderID,
  mode,
  reflectionID,
  adjustmentID,
  initialData,
  onSuccess,
  onCancel,
}: ReflectionEditFormProps) {
  const isReflectionMode = !adjustmentID
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)

  // Reflection form state
  const [analysisType, setAnalysisType] = useState<
    'performance' | 'risk' | 'strategy'
  >(
    (isReflectionMode && (initialData as Reflection)?.analysisType) ||
      'performance'
  )
  const [findings, setFindings] = useState(
    (isReflectionMode && (initialData as Reflection)?.findings) || ''
  )
  const [severity, setSeverity] = useState<'info' | 'warning' | 'error'>(
    (isReflectionMode && (initialData as Reflection)?.severity) || 'info'
  )

  // Adjustment form state
  const [suggestedAction, setSuggestedAction] = useState(
    (!isReflectionMode && (initialData as Adjustment)?.suggestedAction) || ''
  )
  const [reasoning, setReasoning] = useState(
    (!isReflectionMode && (initialData as Adjustment)?.reasoning) || ''
  )
  const [priority, setPriority] = useState<'low' | 'medium' | 'high'>(
    (!isReflectionMode && (initialData as Adjustment)?.priority) || 'medium'
  )

  // 处理提交
  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    setLoading(true)
    setError(null)

    try {
      let url = ''
      let method = 'POST'
      let body = {}

      if (isReflectionMode) {
        // Reflection
        if (mode === 'create') {
          url = `/api/reflection/${traderID}/create`
          body = {
            analysisType,
            findings,
            severity,
          }
        } else if (reflectionID) {
          url = `/api/reflection/id/${reflectionID}`
          method = 'PUT'
          body = {
            findings,
            severity,
          }
        }
      } else {
        // Adjustment
        if (adjustmentID) {
          url = `/api/adjustment/${adjustmentID}`
          method = 'PUT'
          body = {
            suggestedAction,
            reasoning,
            priority,
          }
        }
      }

      const response = await fetch(url, {
        method,
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(body),
      })

      if (!response.ok) {
        const errorData = await response.json()
        throw new Error(errorData.error || 'Failed to save')
      }

      const result = await response.json()
      onSuccess(result.data)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Unknown error')
    } finally {
      setLoading(false)
    }
  }

  return (
    <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center p-4 z-50">
      <div className="bg-white rounded-lg shadow-lg max-w-2xl w-full">
        {/* Header */}
        <div className="border-b border-gray-200 p-6">
          <h2 className="text-2xl font-bold text-gray-900">
            {isReflectionMode
              ? mode === 'create'
                ? '📝 新建反思'
                : '✏️ 编辑反思'
              : '⚡ 编辑调整建议'}
          </h2>
        </div>

        {/* Form Content */}
        <form onSubmit={handleSubmit} className="p-6 space-y-6">
          {error && (
            <div className="p-4 bg-red-50 border border-red-200 rounded-lg text-red-800">
              {error}
            </div>
          )}

          {isReflectionMode ? (
            // Reflection Form
            <>
              {/* Analysis Type */}
              {mode === 'create' && (
                <div>
                  <label className="block text-sm font-medium text-gray-700 mb-2">
                    分析类型 *
                  </label>
                  <div className="grid grid-cols-3 gap-3">
                    {(['performance', 'risk', 'strategy'] as const).map(
                      (type) => (
                        <label
                          key={type}
                          className={`flex items-center p-3 border-2 rounded-lg cursor-pointer transition ${
                            analysisType === type
                              ? 'border-blue-500 bg-blue-50'
                              : 'border-gray-200 hover:border-gray-300'
                          }`}
                        >
                          <input
                            type="radio"
                            name="analysisType"
                            value={type}
                            checked={analysisType === type}
                            onChange={(e) =>
                              setAnalysisType(e.target.value as any)
                            }
                            className="h-4 w-4 text-blue-600"
                          />
                          <span className="ml-2 font-medium text-gray-900">
                            {type === 'performance' && '📊 性能'}
                            {type === 'risk' && '⚠️ 风险'}
                            {type === 'strategy' && '🎯 策略'}
                          </span>
                        </label>
                      )
                    )}
                  </div>
                </div>
              )}

              {/* Findings */}
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-2">
                  发现内容 *
                </label>
                <textarea
                  value={findings}
                  onChange={(e) => setFindings(e.target.value)}
                  placeholder="详细描述反思发现..."
                  rows={6}
                  className="w-full px-4 py-3 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent resize-none"
                  required
                />
              </div>

              {/* Severity */}
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-2">
                  严重程度 *
                </label>
                <div className="grid grid-cols-3 gap-3">
                  {(['info', 'warning', 'error'] as const).map((sev) => (
                    <label
                      key={sev}
                      className={`flex items-center p-3 border-2 rounded-lg cursor-pointer transition ${
                        severity === sev
                          ? 'border-blue-500 bg-blue-50'
                          : 'border-gray-200 hover:border-gray-300'
                      }`}
                    >
                      <input
                        type="radio"
                        name="severity"
                        value={sev}
                        checked={severity === sev}
                        onChange={(e) => setSeverity(e.target.value as any)}
                        className="h-4 w-4 text-blue-600"
                      />
                      <span className="ml-2 font-medium text-gray-900">
                        {sev === 'info' && 'ℹ️ 信息'}
                        {sev === 'warning' && '⚠️ 警告'}
                        {sev === 'error' && '❌ 错误'}
                      </span>
                    </label>
                  ))}
                </div>
              </div>
            </>
          ) : (
            // Adjustment Form
            <>
              {/* Suggested Action */}
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-2">
                  建议行动 *
                </label>
                <textarea
                  value={suggestedAction}
                  onChange={(e) => setSuggestedAction(e.target.value)}
                  placeholder="输入建议的具体行动..."
                  rows={4}
                  className="w-full px-4 py-3 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent resize-none"
                  required
                />
              </div>

              {/* Reasoning */}
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-2">
                  理由说明 *
                </label>
                <textarea
                  value={reasoning}
                  onChange={(e) => setReasoning(e.target.value)}
                  placeholder="解释为什么需要这个调整..."
                  rows={4}
                  className="w-full px-4 py-3 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent resize-none"
                  required
                />
              </div>

              {/* Priority */}
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-2">
                  优先级 *
                </label>
                <div className="grid grid-cols-3 gap-3">
                  {(['low', 'medium', 'high'] as const).map((pri) => (
                    <label
                      key={pri}
                      className={`flex items-center p-3 border-2 rounded-lg cursor-pointer transition ${
                        priority === pri
                          ? 'border-blue-500 bg-blue-50'
                          : 'border-gray-200 hover:border-gray-300'
                      }`}
                    >
                      <input
                        type="radio"
                        name="priority"
                        value={pri}
                        checked={priority === pri}
                        onChange={(e) => setPriority(e.target.value as any)}
                        className="h-4 w-4 text-blue-600"
                      />
                      <span className="ml-2 font-medium text-gray-900">
                        {pri === 'low' && '🟢 低'}
                        {pri === 'medium' && '🟡 中'}
                        {pri === 'high' && '🔴 高'}
                      </span>
                    </label>
                  ))}
                </div>
              </div>
            </>
          )}

          {/* Action Buttons */}
          <div className="flex justify-end gap-3 pt-4 border-t border-gray-200">
            <button
              type="button"
              onClick={onCancel}
              disabled={loading}
              className="px-6 py-2 border border-gray-300 rounded-lg text-gray-700 font-medium hover:bg-gray-50 transition disabled:opacity-50"
            >
              取消
            </button>
            <button
              type="submit"
              disabled={loading}
              className="px-6 py-2 bg-blue-600 text-white rounded-lg font-medium hover:bg-blue-700 transition disabled:opacity-50 flex items-center gap-2"
            >
              {loading && (
                <div className="animate-spin rounded-full h-4 w-4 border-b-2 border-white"></div>
              )}
              {loading ? '保存中...' : '保存'}
            </button>
          </div>
        </form>
      </div>
    </div>
  )
}

export default ReflectionEditForm
