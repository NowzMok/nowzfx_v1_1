import React, { useState, useEffect } from 'react'
import axios from 'axios'
import {
  LineChart,
  Line,
  AreaChart,
  Area,
  BarChart,
  Bar,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  Legend,
  ResponsiveContainer,
} from 'recharts'
import {
  AlertCircle,
  CheckCircle,
  XCircle,
  TrendingUp,
  Activity,
  Zap,
  Heart,
  Info,
} from 'lucide-react'

interface PerformanceMetric {
  id: string
  trader_id: string
  timestamp: string
  win_rate: number
  profit_factor: number
  total_pnl: number
  max_drawdown: number
  current_drawdown: number
  sharpe_ratio: number
  total_trades: number
  open_positions: number
  total_equity: number
}

interface Alert {
  id: string
  alert_rule_id: string
  status: 'triggered' | 'acknowledged' | 'resolved'
  severity: 'info' | 'warning' | 'critical'
  message: string
  triggered_at: string
  acknowledged_at?: string
  resolved_at?: string
}

interface SystemHealth {
  id: string
  exchange_connected: boolean
  database_connected: boolean
  api_healthy: boolean
  api_latency: number
  database_latency: number
  memory_usage: number
  cpu_usage: number
  status: 'healthy' | 'degraded' | 'unhealthy'
}

interface MonitoringDashboardProps {
  traderID: string
  apiBaseURL?: string
}

const MonitoringDashboard: React.FC<MonitoringDashboardProps> = ({
  traderID,
  apiBaseURL = 'http://localhost:8080',
}) => {
  const [metrics, setMetrics] = useState<PerformanceMetric[]>([])
  const [alerts, setAlerts] = useState<Alert[]>([])
  const [health, setHealth] = useState<SystemHealth | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [activeTab, setActiveTab] = useState<'metrics' | 'alerts' | 'health'>(
    'metrics'
  )

  const fetchData = async () => {
    try {
      setLoading(true)
      setError(null)

      // 获取性能指标
      const metricsRes = await axios.get(
        `${apiBaseURL}/api/monitoring/${traderID}/metrics?limit=100`
      )
      if (metricsRes.data.data?.metrics) {
        setMetrics(metricsRes.data.data.metrics)
      }

      // 获取活跃告警
      const alertsRes = await axios.get(
        `${apiBaseURL}/api/monitoring/${traderID}/alerts/active`
      )
      if (alertsRes.data.data?.alerts) {
        setAlerts(alertsRes.data.data.alerts)
      }

      // 获取健康状态
      const healthRes = await axios.get(
        `${apiBaseURL}/api/monitoring/${traderID}/health`
      )
      if (healthRes.data.data) {
        setHealth(healthRes.data.data)
      }
    } catch (err) {
      setError(
        err instanceof Error ? err.message : 'Failed to fetch monitoring data'
      )
      console.error('Monitoring fetch error:', err)
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    fetchData()
    const interval = setInterval(fetchData, 30000) // 每 30 秒刷新一次
    return () => clearInterval(interval)
  }, [traderID])

  const handleAcknowledgeAlert = async (alertID: string) => {
    try {
      await axios.post(
        `${apiBaseURL}/api/monitoring/alerts/${alertID}/acknowledge`
      )
      fetchData() // 刷新数据
    } catch (err) {
      console.error('Failed to acknowledge alert:', err)
    }
  }

  const handleResolveAlert = async (alertID: string) => {
    try {
      await axios.post(`${apiBaseURL}/api/monitoring/alerts/${alertID}/resolve`)
      fetchData() // 刷新数据
    } catch (err) {
      console.error('Failed to resolve alert:', err)
    }
  }

  const getAlertIcon = (severity: string) => {
    switch (severity) {
      case 'critical':
        return <AlertCircle className="w-5 h-5 text-red-500" />
      case 'warning':
        return <AlertCircle className="w-5 h-5 text-yellow-500" />
      default:
        return <Info className="w-5 h-5 text-blue-500" />
    }
  }

  const getHealthStatusColor = (status: string) => {
    switch (status) {
      case 'healthy':
        return 'bg-green-100 text-green-800'
      case 'degraded':
        return 'bg-yellow-100 text-yellow-800'
      case 'unhealthy':
        return 'bg-red-100 text-red-800'
      default:
        return 'bg-gray-100 text-gray-800'
    }
  }

  const getAlertStatusColor = (status: string) => {
    switch (status) {
      case 'triggered':
        return 'text-red-600'
      case 'acknowledged':
        return 'text-yellow-600'
      case 'resolved':
        return 'text-green-600'
      default:
        return 'text-gray-600'
    }
  }

  const chartData = metrics.slice(-24).map((m) => ({
    time: new Date(m.timestamp).toLocaleTimeString(),
    winRate: Math.round(m.win_rate * 100),
    profitFactor: m.profit_factor,
    drawdown: Math.round(m.max_drawdown * 100),
    pnl: m.total_pnl,
  }))

  if (loading) {
    return (
      <div className="flex items-center justify-center h-96">
        <div className="text-center">
          <Activity className="w-12 h-12 text-blue-500 animate-spin mx-auto mb-4" />
          <p className="text-gray-600">加载监控数据...</p>
        </div>
      </div>
    )
  }

  return (
    <div className="w-full bg-gray-50">
      {/* 标题栏 */}
      <div className="bg-white border-b border-gray-200 p-6">
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-3">
            <Heart className="w-8 h-8 text-red-500" />
            <div>
              <h1 className="text-2xl font-bold text-gray-900">
                交易监控仪表板
              </h1>
              <p className="text-sm text-gray-500">交易员 ID: {traderID}</p>
            </div>
          </div>
          <button
            onClick={fetchData}
            className="px-4 py-2 bg-blue-500 text-white rounded-lg hover:bg-blue-600 transition"
          >
            刷新数据
          </button>
        </div>
      </div>

      {error && (
        <div className="bg-red-50 border-l-4 border-red-500 p-4 m-6">
          <p className="text-red-700">{error}</p>
        </div>
      )}

      {/* 关键指标卡片 */}
      {metrics.length > 0 && (
        <div className="grid grid-cols-1 md:grid-cols-4 gap-4 p-6">
          <div className="bg-white rounded-lg p-4 border border-gray-200">
            <div className="flex items-center justify-between">
              <div>
                <p className="text-gray-500 text-sm">胜率</p>
                <p className="text-2xl font-bold text-gray-900">
                  {(metrics[metrics.length - 1].win_rate * 100).toFixed(1)}%
                </p>
              </div>
              <TrendingUp className="w-8 h-8 text-green-500" />
            </div>
          </div>

          <div className="bg-white rounded-lg p-4 border border-gray-200">
            <div className="flex items-center justify-between">
              <div>
                <p className="text-gray-500 text-sm">盈利因子</p>
                <p className="text-2xl font-bold text-gray-900">
                  {metrics[metrics.length - 1].profit_factor.toFixed(2)}
                </p>
              </div>
              <Zap className="w-8 h-8 text-yellow-500" />
            </div>
          </div>

          <div className="bg-white rounded-lg p-4 border border-gray-200">
            <div className="flex items-center justify-between">
              <div>
                <p className="text-gray-500 text-sm">最大回撤</p>
                <p className="text-2xl font-bold text-gray-900">
                  {(metrics[metrics.length - 1].max_drawdown * 100).toFixed(1)}%
                </p>
              </div>
              <AlertCircle className="w-8 h-8 text-red-500" />
            </div>
          </div>

          <div className="bg-white rounded-lg p-4 border border-gray-200">
            <div className="flex items-center justify-between">
              <div>
                <p className="text-gray-500 text-sm">总损益</p>
                <p
                  className={`text-2xl font-bold ${
                    metrics[metrics.length - 1].total_pnl >= 0
                      ? 'text-green-600'
                      : 'text-red-600'
                  }`}
                >
                  ${metrics[metrics.length - 1].total_pnl.toFixed(0)}
                </p>
              </div>
              <Activity className="w-8 h-8 text-blue-500" />
            </div>
          </div>
        </div>
      )}

      {/* 标签页导航 */}
      <div className="bg-white border-b border-gray-200 px-6">
        <div className="flex gap-8">
          <button
            onClick={() => setActiveTab('metrics')}
            className={`py-4 px-2 font-medium border-b-2 transition ${
              activeTab === 'metrics'
                ? 'border-blue-500 text-blue-600'
                : 'border-transparent text-gray-600 hover:text-gray-900'
            }`}
          >
            📊 性能指标
          </button>
          <button
            onClick={() => setActiveTab('alerts')}
            className={`py-4 px-2 font-medium border-b-2 transition relative ${
              activeTab === 'alerts'
                ? 'border-blue-500 text-blue-600'
                : 'border-transparent text-gray-600 hover:text-gray-900'
            }`}
          >
            🚨 告警管理
            {alerts.length > 0 && (
              <span className="absolute top-2 right-0 inline-flex items-center justify-center px-2 py-1 text-xs font-bold leading-none text-white transform translate-x-1/2 -translate-y-1/2 bg-red-600 rounded-full">
                {alerts.length}
              </span>
            )}
          </button>
          <button
            onClick={() => setActiveTab('health')}
            className={`py-4 px-2 font-medium border-b-2 transition ${
              activeTab === 'health'
                ? 'border-blue-500 text-blue-600'
                : 'border-transparent text-gray-600 hover:text-gray-900'
            }`}
          >
            ❤️ 系统健康
          </button>
        </div>
      </div>

      {/* 性能指标面板 */}
      {activeTab === 'metrics' && (
        <div className="p-6 space-y-6">
          <div className="bg-white rounded-lg p-6 border border-gray-200">
            <h3 className="text-lg font-semibold text-gray-900 mb-4">
              胜率趋势
            </h3>
            <ResponsiveContainer width="100%" height={300}>
              <LineChart data={chartData}>
                <CartesianGrid strokeDasharray="3 3" />
                <XAxis dataKey="time" />
                <YAxis yAxisId="left" />
                <Tooltip />
                <Legend />
                <Line
                  yAxisId="left"
                  type="monotone"
                  dataKey="winRate"
                  stroke="#10b981"
                  name="胜率 (%)"
                />
              </LineChart>
            </ResponsiveContainer>
          </div>

          <div className="bg-white rounded-lg p-6 border border-gray-200">
            <h3 className="text-lg font-semibold text-gray-900 mb-4">
              损益和回撤
            </h3>
            <ResponsiveContainer width="100%" height={300}>
              <AreaChart data={chartData}>
                <CartesianGrid strokeDasharray="3 3" />
                <XAxis dataKey="time" />
                <YAxis />
                <Tooltip />
                <Legend />
                <Area
                  type="monotone"
                  dataKey="pnl"
                  fill="#3b82f6"
                  stroke="#3b82f6"
                  name="总损益"
                />
              </AreaChart>
            </ResponsiveContainer>
          </div>

          <div className="bg-white rounded-lg p-6 border border-gray-200">
            <h3 className="text-lg font-semibold text-gray-900 mb-4">
              回撤趋势
            </h3>
            <ResponsiveContainer width="100%" height={300}>
              <BarChart data={chartData}>
                <CartesianGrid strokeDasharray="3 3" />
                <XAxis dataKey="time" />
                <YAxis />
                <Tooltip />
                <Legend />
                <Bar dataKey="drawdown" fill="#ef4444" name="回撤 (%)" />
              </BarChart>
            </ResponsiveContainer>
          </div>
        </div>
      )}

      {/* 告警管理面板 */}
      {activeTab === 'alerts' && (
        <div className="p-6">
          {alerts.length === 0 ? (
            <div className="bg-white rounded-lg p-12 text-center border border-gray-200">
              <CheckCircle className="w-12 h-12 text-green-500 mx-auto mb-4" />
              <p className="text-gray-600">没有活跃告警</p>
            </div>
          ) : (
            <div className="space-y-4">
              {alerts.map((alert) => (
                <div
                  key={alert.id}
                  className="bg-white rounded-lg p-4 border-l-4 border-gray-200"
                  style={{
                    borderLeftColor:
                      alert.severity === 'critical'
                        ? '#ef4444'
                        : alert.severity === 'warning'
                          ? '#f59e0b'
                          : '#3b82f6',
                  }}
                >
                  <div className="flex items-start justify-between">
                    <div className="flex items-start gap-3 flex-1">
                      {getAlertIcon(alert.severity)}
                      <div className="flex-1">
                        <p className="font-semibold text-gray-900">
                          {alert.message}
                        </p>
                        <div className="mt-2 flex gap-4 text-sm text-gray-500">
                          <span>
                            触发时间:{' '}
                            {new Date(alert.triggered_at).toLocaleString()}
                          </span>
                          <span
                            className={`font-medium ${getAlertStatusColor(alert.status)}`}
                          >
                            状态: {alert.status}
                          </span>
                        </div>
                      </div>
                    </div>
                    <div className="flex gap-2">
                      {alert.status === 'triggered' && (
                        <>
                          <button
                            onClick={() => handleAcknowledgeAlert(alert.id)}
                            className="px-3 py-1 text-sm bg-yellow-100 text-yellow-700 rounded hover:bg-yellow-200 transition"
                          >
                            确认
                          </button>
                          <button
                            onClick={() => handleResolveAlert(alert.id)}
                            className="px-3 py-1 text-sm bg-green-100 text-green-700 rounded hover:bg-green-200 transition"
                          >
                            解决
                          </button>
                        </>
                      )}
                      {alert.status === 'acknowledged' && (
                        <button
                          onClick={() => handleResolveAlert(alert.id)}
                          className="px-3 py-1 text-sm bg-green-100 text-green-700 rounded hover:bg-green-200 transition"
                        >
                          解决
                        </button>
                      )}
                    </div>
                  </div>
                </div>
              ))}
            </div>
          )}
        </div>
      )}

      {/* 系统健康面板 */}
      {activeTab === 'health' && (
        <div className="p-6 space-y-6">
          {health && (
            <>
              <div className="bg-white rounded-lg p-6 border border-gray-200">
                <div className="flex items-center justify-between mb-6">
                  <h3 className="text-lg font-semibold text-gray-900">
                    系统状态
                  </h3>
                  <div
                    className={`px-4 py-2 rounded-full font-medium text-sm ${getHealthStatusColor(
                      health.status
                    )}`}
                  >
                    {health.status === 'healthy'
                      ? '✅ 健康'
                      : health.status === 'degraded'
                        ? '⚠️ 降级'
                        : '❌ 不健康'}
                  </div>
                </div>

                <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                  <div className="flex items-center justify-between p-4 bg-gray-50 rounded">
                    <span className="text-gray-600">交易所连接</span>
                    {health.exchange_connected ? (
                      <CheckCircle className="w-5 h-5 text-green-500" />
                    ) : (
                      <XCircle className="w-5 h-5 text-red-500" />
                    )}
                  </div>

                  <div className="flex items-center justify-between p-4 bg-gray-50 rounded">
                    <span className="text-gray-600">数据库连接</span>
                    {health.database_connected ? (
                      <CheckCircle className="w-5 h-5 text-green-500" />
                    ) : (
                      <XCircle className="w-5 h-5 text-red-500" />
                    )}
                  </div>

                  <div className="flex items-center justify-between p-4 bg-gray-50 rounded">
                    <span className="text-gray-600">API 服务</span>
                    {health.api_healthy ? (
                      <CheckCircle className="w-5 h-5 text-green-500" />
                    ) : (
                      <XCircle className="w-5 h-5 text-red-500" />
                    )}
                  </div>

                  <div className="flex items-center justify-between p-4 bg-gray-50 rounded">
                    <span className="text-gray-600">API 延迟</span>
                    <span className="font-medium">
                      {health.api_latency.toFixed(0)}ms
                    </span>
                  </div>

                  <div className="flex items-center justify-between p-4 bg-gray-50 rounded">
                    <span className="text-gray-600">内存使用</span>
                    <span className="font-medium">
                      {health.memory_usage.toFixed(1)}MB
                    </span>
                  </div>

                  <div className="flex items-center justify-between p-4 bg-gray-50 rounded">
                    <span className="text-gray-600">CPU 使用</span>
                    <span className="font-medium">
                      {health.cpu_usage.toFixed(1)}%
                    </span>
                  </div>
                </div>
              </div>
            </>
          )}
        </div>
      )}
    </div>
  )
}

export default MonitoringDashboard
