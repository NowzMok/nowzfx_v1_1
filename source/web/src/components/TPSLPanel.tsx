import { useState, useEffect } from 'react'
import { api } from '../lib/api'
import { useLanguage } from '../contexts/LanguageContext'
import type { TPSLRecord } from '../types'

interface TPSLPanelProps {
  traderId?: string
}

const translations = {
  zh: {
    tpslTracking: '止盈止损跟踪',
    symbol: '交易对',
    side: '方向',
    currentTP: '当前TP',
    currentSL: '当前SL',
    originalTP: '原始TP',
    originalSL: '原始SL',
    entryPrice: '入场价',
    modifiedCount: '修改次数',
    status: '状态',
    active: '活跃',
    triggered: '已触发',
    closed: '已关闭',
    modifyBtn: '修改',
    noData: '暂无数据',
    newTP: '新TP',
    newSL: '新SL',
    save: '保存',
    cancel: '取消',
    modifyTPSL: '修改止盈止损',
    pnl: '收益率',
  },
  en: {
    tpslTracking: 'TP/SL Tracking',
    symbol: 'Symbol',
    side: 'Side',
    currentTP: 'Current TP',
    currentSL: 'Current SL',
    originalTP: 'Original TP',
    originalSL: 'Original SL',
    entryPrice: 'Entry Price',
    modifiedCount: 'Modified',
    status: 'Status',
    active: 'Active',
    triggered: 'Triggered',
    closed: 'Closed',
    modifyBtn: 'Modify',
    noData: 'No data',
    newTP: 'New TP',
    newSL: 'New SL',
    save: 'Save',
    cancel: 'Cancel',
    modifyTPSL: 'Modify TP/SL',
    pnl: 'PnL %',
  },
}

function formatPrice(price: number): string {
  if (!price || price === 0) return '-'
  if (price >= 1000) return price.toFixed(2)
  if (price >= 1) return price.toFixed(4)
  return price.toFixed(6)
}

function getPriceColor(currentPrice: number, entryPrice: number, side: string) {
  if (!currentPrice || !entryPrice) return '#EAECEF'
  const isLong = side === 'LONG'
  const change = isLong ? currentPrice - entryPrice : entryPrice - currentPrice
  return change > 0 ? '#22C55E' : change < 0 ? '#EF4444' : '#EAECEF'
}

export function TPSLPanel({ traderId }: TPSLPanelProps) {
  const { language } = useLanguage()
  const t_dict = translations[language as 'zh' | 'en'] || translations['en']
  const [tpslRecords, setTPSLRecords] = useState<TPSLRecord[]>([])
  const [loading, setLoading] = useState(false)
  const [editingId, setEditingId] = useState<number | null>(null)
  const [newTP, setNewTP] = useState<string>('')
  const [newSL, setNewSL] = useState<string>('')

  useEffect(() => {
    if (traderId) {
      fetchTPSLRecords()
      const interval = setInterval(fetchTPSLRecords, 30000) // Refresh every 30s
      return () => clearInterval(interval)
    }
  }, [traderId])

  async function fetchTPSLRecords() {
    if (!traderId) return
    setLoading(true)
    try {
      const response = await api.get(`/traders/${traderId}/tpsl-records`)
      if (response.data) {
        setTPSLRecords(response.data)
      }
    } catch (error) {
      console.error('Failed to fetch TP/SL records:', error)
    } finally {
      setLoading(false)
    }
  }

  async function handleModifyTPSL(recordId: number, tp: number, sl: number) {
    if (!traderId) return
    try {
      await api.post(`/traders/${traderId}/modify-tp-sl`, {
        tpsl_record_id: recordId,
        new_tp: tp,
        new_sl: sl,
      })
      setEditingId(null)
      setNewTP('')
      setNewSL('')
      await fetchTPSLRecords()
    } catch (error) {
      console.error('Failed to modify TP/SL:', error)
      alert('修改失败，请重试')
    }
  }

  if (loading && tpslRecords.length === 0) {
    return (
      <div className="p-4 text-center text-gray-400">
        {t_dict.noData}
      </div>
    )
  }

  if (tpslRecords.length === 0) {
    return (
      <div className="p-4 text-center text-gray-400">
        {t_dict.noData}
      </div>
    )
  }

  return (
    <div className="rounded-lg p-4" style={{ background: '#0F1117', border: '1px solid #2B3139' }}>
      <h3 className="text-lg font-bold mb-4 flex items-center gap-2">
        <span>📊</span>
        {t_dict.tpslTracking}
      </h3>

      <div className="overflow-x-auto">
        <table className="w-full text-sm">
          <thead>
            <tr style={{ borderBottom: '1px solid #2B3139' }}>
              <th className="text-left py-2 px-2">{t_dict.symbol}</th>
              <th className="text-left py-2 px-2">{t_dict.side}</th>
              <th className="text-right py-2 px-2">{t_dict.entryPrice}</th>
              <th className="text-right py-2 px-2">{t_dict.currentTP}</th>
              <th className="text-right py-2 px-2">{t_dict.currentSL}</th>
              <th className="text-center py-2 px-2">{t_dict.modifiedCount}</th>
              <th className="text-center py-2 px-2">{t_dict.status}</th>
              <th className="text-center py-2 px-2">操作</th>
            </tr>
          </thead>
          <tbody>
            {tpslRecords.map((record) => (
              <tr
                key={record.id}
                style={{
                  borderBottom: '1px solid #1E2329',
                  background: editingId === record.id ? '#1E2329' : 'transparent',
                }}
              >
                <td className="py-2 px-2 font-mono font-bold">{record.symbol}</td>
                <td className="py-2 px-2">
                  <span
                    style={{
                      color: record.side === 'LONG' ? '#22C55E' : '#EF4444',
                      fontWeight: 'bold',
                    }}
                  >
                    {record.side}
                  </span>
                </td>
                <td className="py-2 px-2 text-right">{formatPrice(record.entry_price)}</td>
                <td
                  className="py-2 px-2 text-right font-mono"
                  style={{ color: getPriceColor(record.current_tp, record.entry_price, record.side) }}
                >
                  {editingId === record.id ? (
                    <input
                      type="number"
                      value={newTP}
                      onChange={(e) => setNewTP(e.target.value)}
                      className="w-20 px-1 py-0 rounded bg-gray-700 text-white text-xs"
                      placeholder={formatPrice(record.current_tp)}
                      step="0.01"
                    />
                  ) : (
                    formatPrice(record.current_tp)
                  )}
                </td>
                <td
                  className="py-2 px-2 text-right font-mono"
                  style={{ color: getPriceColor(record.current_sl, record.entry_price, record.side) }}
                >
                  {editingId === record.id ? (
                    <input
                      type="number"
                      value={newSL}
                      onChange={(e) => setNewSL(e.target.value)}
                      className="w-20 px-1 py-0 rounded bg-gray-700 text-white text-xs"
                      placeholder={formatPrice(record.current_sl)}
                      step="0.01"
                    />
                  ) : (
                    formatPrice(record.current_sl)
                  )}
                </td>
                <td className="py-2 px-2 text-center">{record.modified_count}</td>
                <td className="py-2 px-2 text-center">
                  <span
                    style={{
                      color:
                        record.status === 'ACTIVE'
                          ? '#22C55E'
                          : record.status === 'TRIGGERED'
                          ? '#FDB022'
                          : '#848E9C',
                      fontSize: '0.75rem',
                      fontWeight: 'bold',
                    }}
                  >
                    {record.status}
                  </span>
                </td>
                <td className="py-2 px-2 text-center">
                  {editingId === record.id ? (
                    <div className="flex gap-1 justify-center">
                      <button
                        onClick={() => {
                          const tp = newTP ? parseFloat(newTP) : record.current_tp
                          const sl = newSL ? parseFloat(newSL) : record.current_sl
                          handleModifyTPSL(record.id, tp, sl)
                        }}
                        className="px-2 py-1 bg-green-600 text-white text-xs rounded hover:bg-green-700"
                      >
                        ✓
                      </button>
                      <button
                        onClick={() => {
                          setEditingId(null)
                          setNewTP('')
                          setNewSL('')
                        }}
                        className="px-2 py-1 bg-gray-600 text-white text-xs rounded hover:bg-gray-700"
                      >
                        ✕
                      </button>
                    </div>
                  ) : (
                    <button
                      onClick={() => {
                        setEditingId(record.id)
                        setNewTP('')
                        setNewSL('')
                      }}
                      className="px-2 py-1 bg-blue-600 text-white text-xs rounded hover:bg-blue-700"
                    >
                      {t_dict.modifyBtn}
                    </button>
                  )}
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>

      <div className="mt-4 text-xs text-gray-400">
        💡 {tpslRecords.length} 个活跃的 TP/SL 记录
      </div>
    </div>
  )
}
