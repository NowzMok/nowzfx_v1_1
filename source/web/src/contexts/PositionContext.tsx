/**
 * PositionContext.tsx - 持仓上下文
 *
 * 跨组件共享当前持仓数据，用于动态止损过滤
 * 使用 Set 存储 OPEN 状态的币种，供 PendingOrdersTable 快速查询
 */

import React, { createContext, useContext, useState, useCallback } from 'react'
import type { Position } from '../types'

interface PositionContextType {
  positions: Position[]
  openSymbols: Set<string> // OPEN 状态的币种集合
  setPositions: (positions: Position[]) => void
}

const PositionContext = createContext<PositionContextType | undefined>(undefined)

/**
 * PositionProvider 组件 - 提供持仓上下文
 */
export function PositionProvider({ children }: { children: React.ReactNode }) {
  const [positions, setPositionsState] = useState<Position[]>([])

  // 计算 openSymbols（OPEN 状态的币种集合）
  const openSymbols = React.useMemo(() => {
    const symbols = new Set<string>()
    positions.forEach((pos) => {
      if (pos.status === 'OPEN') {
        symbols.add(pos.symbol)
      }
    })
    console.log('[PositionContext] 计算 openSymbols:', Array.from(symbols))
    console.log('[PositionContext] 持仓总数:', positions.length, '开仓数:', symbols.size)
    return symbols
  }, [positions])

  // 更新 positions 的 wrapper
  const setPositions = useCallback((newPositions: Position[]) => {
    setPositionsState(newPositions)
  }, [])

  const value: PositionContextType = {
    positions,
    openSymbols,
    setPositions,
  }

  return (
    <PositionContext.Provider value={value}>
      {children}
    </PositionContext.Provider>
  )
}

/**
 * usePositions 钩子 - 在组件中访问持仓上下文
 */
export function usePositions(): PositionContextType {
  const context = useContext(PositionContext)
  if (context === undefined) {
    throw new Error('usePositions 必须在 PositionProvider 内使用')
  }
  return context
}
