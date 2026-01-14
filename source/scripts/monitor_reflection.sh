#!/bin/bash

# 反思系统实时监控脚本
# 使用方法: ./monitor_reflection.sh

echo "╔════════════════════════════════════════════════════════════╗"
echo "║         🔄 反思系统实时监控 - $(date '+%Y-%m-%d %H:%M:%S')         ║"
echo "╚════════════════════════════════════════════════════════════╝"
echo ""

# 获取脚本所在目录
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="$(dirname "$SCRIPT_DIR")"
cd "$PROJECT_DIR"

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

# 1. 检查进程状态
echo -e "${BLUE}1. 进程状态检查${NC}"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
PROCESS_COUNT=$(ps aux | grep -E "(nofx-app|main\.go)" | grep -v grep | wc -l)
if [ "$PROCESS_COUNT" -gt 0 ]; then
    echo -e "✅ ${GREEN}运行中进程: $PROCESS_COUNT${NC}"
    ps aux | grep -E "(nofx-app|main\.go)" | grep -v grep | head -3
else
    echo -e "❌ ${RED}无运行进程${NC}"
    echo "   建议: cd nofx && ./nofx-app"
fi
echo ""

# 2. 检查数据库状态
echo -e "${BLUE}2. 数据库状态检查${NC}"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
if [ -f "data/data.db" ]; then
    REFLECTIONS=$(sqlite3 data/data.db "SELECT COUNT(*) FROM reflections;" 2>/dev/null || echo "0")
    ADJUSTMENTS=$(sqlite3 data/data.db "SELECT COUNT(*) FROM system_adjustments;" 2>/dev/null || echo "0")
    MEMORY=$(sqlite3 data/data.db "SELECT COUNT(*) FROM ai_learning_memory;" 2>/dev/null || echo "0")
    TRADERS=$(sqlite3 data/data.db "SELECT COUNT(*) FROM traders WHERE is_running = 1;" 2>/dev/null || echo "0")
    TRADES=$(sqlite3 data/data.db "SELECT COUNT(*) FROM trade_history;" 2>/dev/null || echo "0")
    
    echo "📊 反思记录: $REFLECTIONS 条"
    echo "⚡ 调整建议: $ADJUSTMENTS 条"
    echo "🧠 AI记忆: $MEMORY 条"
    echo "👤 活跃交易者: $TRADERS 个"
    echo "📈 交易历史: $TRADES 条"
    
    PENDING_COUNT=$(sqlite3 data/data.db "SELECT COUNT(*) FROM system_adjustments WHERE status = 'pending';" 2>/dev/null || echo "0")
    if [ "$PENDING_COUNT" -gt 0 ]; then
        echo -e "⚠️  ${YELLOW}待处理调整: $PENDING_COUNT 条${NC}"
    fi
else
    echo -e "❌ ${RED}数据库文件不存在${NC}"
fi
echo ""

# 3. 最近反思记录
echo -e "${BLUE}3. 最近反思记录${NC}"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
sqlite3 -header -column data/data.db "SELECT datetime(created_at, 'localtime') as 时间, substr(ai_reflection, 1, 30) || '...' as AI反思, total_trades as 交易数, success_rate as 胜率, total_pn_l as 盈亏 FROM reflections ORDER BY created_at DESC LIMIT 5;" 2>/dev/null || echo "   暂无反思记录"
echo ""

# 4. 待处理调整建议
echo -e "${BLUE}4. 待处理调整建议${NC}"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
sqlite3 -header -column data/data.db "SELECT datetime(created_at, 'localtime') as 时间, substr(adjustment_reason, 1, 35) || '...' as 调整原因, confidence_level as 置信度, status as 状态 FROM system_adjustments WHERE status = 'pending' ORDER BY confidence_level DESC;" 2>/dev/null || echo "   无待处理调整"
echo ""

# 5. AI学习记忆
echo -e "${BLUE}5. AI学习记忆 (Top 5)${NC}"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
sqlite3 -header -column data/data.db "SELECT datetime(created_at, 'localtime') as 时间, substr(ai_learning_advice, 1, 35) || '...' as 学习建议, confidence_accuracy as 置信度 FROM ai_learning_memory ORDER BY created_at DESC LIMIT 5;" 2>/dev/null || echo "   暂无学习记忆"
echo ""

# 6. 调度器状态
echo -e "${BLUE}6. 调度器状态${NC}"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
if [ -f "data/data.db" ]; then
    NEXT_RUN=$(sqlite3 data/data.db "SELECT next_run FROM reflection_schedules LIMIT 1;" 2>/dev/null)
    if [ ! -z "$NEXT_RUN" ]; then
        echo "📅 下次运行: $NEXT_RUN"
        
        # 计算剩余时间
        CURRENT_EPOCH=$(date +%s)
        NEXT_EPOCH=$(date -j -f "%Y-%m-%d %H:%M:%S" "$NEXT_RUN" +%s 2>/dev/null || echo "0")
        if [ "$NEXT_EPOCH" -gt "$CURRENT_EPOCH" ]; then
            REMAINING=$((NEXT_EPOCH - CURRENT_EPOCH))
            HOURS=$((REMAINING / 3600))
            MINUTES=$(((REMAINING % 3600) / 60))
            echo "⏰ 剩余时间: ${HOURS}小时${MINUTES}分钟"
        fi
    else
        echo "⚠️  调度器未初始化"
    fi
else
    echo "❌ 无法检查调度器状态"
fi
echo ""

# 7. API端点测试
echo -e "${BLUE}7. API端点测试${NC}"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
TRADER_ID=$(sqlite3 data/data.db "SELECT id FROM traders WHERE is_running = 1 LIMIT 1;" 2>/dev/null)
if [ ! -z "$TRADER_ID" ]; then
    echo "交易者ID: $TRADER_ID"
    
    # 测试统计端点
    if curl -s "http://localhost:8080/api/reflection/$TRADER_ID/stats" > /dev/null 2>&1; then
        echo -e "✅ ${GREEN}统计端点: 可用${NC}"
    else
        echo -e "❌ ${RED}统计端点: 不可用${NC}"
    fi
    
    # 测试最近记录端点
    if curl -s "http://localhost:8080/api/reflection/$TRADER_ID/recent" > /dev/null 2>&1; then
        echo -e "✅ ${GREEN}最近记录端点: 可用${NC}"
    else
        echo -e "❌ ${RED}最近记录端点: 不可用${NC}"
    fi
else
    echo "⚠️  无活跃交易者"
fi
echo ""

# 8. 日志检查
echo -e "${BLUE}8. 最近日志条目${NC}"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
LOG_FILE="data/nofx_$(date +%Y-%m-%d).log"
if [ -f "$LOG_FILE" ]; then
    RECENT_REFLECTION=$(grep -i reflection "$LOG_FILE" | tail -3)
    if [ ! -z "$RECENT_REFLECTION" ]; then
        echo "$RECENT_REFLECTION"
    else
        echo "   今日无反思相关日志"
    fi
else
    echo "   日志文件不存在: $LOG_FILE"
fi
echo ""

# 9. 问题诊断
echo -e "${BLUE}9. 问题诊断${NC}"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
ISSUES=0

# 检查1: 进程是否运行
if [ "$PROCESS_COUNT" -eq 0 ]; then
    echo -e "❌ ${RED}问题: 反思系统进程未运行${NC}"
    echo "   解决方案: cd nofx && ./nofx-app"
    ISSUES=$((ISSUES + 1))
fi

# 检查2: 是否有交易数据
TRADE_COUNT=$(sqlite3 data/data.db "SELECT COUNT(*) FROM trade_history;" 2>/dev/null || echo "0")
if [ "$TRADE_COUNT" -eq 0 ]; then
    echo -e "⚠️  ${YELLOW}问题: 无交易历史数据${NC}"
    echo "   说明: 需要交易数据才能生成反思"
    echo "   方案: 运行交易者或手动插入测试数据"
    ISSUES=$((ISSUES + 1))
fi

# 检查3: 是否有反思记录
REFLECTION_COUNT=$(sqlite3 data/data.db "SELECT COUNT(*) FROM reflections;" 2>/dev/null || echo "0")
if [ "$REFLECTION_COUNT" -eq 0 ] && [ "$TRADE_COUNT" -gt 0 ]; then
    echo -e "⚠️  ${YELLOW}问题: 有交易数据但无反思记录${NC}"
    echo "   说明: 可能未到达调度时间"
    echo "   方案: 手动触发分析或等待调度"
    ISSUES=$((ISSUES + 1))
fi

# 检查4: API是否响应
if [ ! -z "$TRADER_ID" ]; then
    if ! curl -s "http://localhost:8080/api/reflection/$TRADER_ID/stats" > /dev/null 2>&1; then
        echo -e "❌ ${RED}问题: API端点无响应${NC}"
        echo "   解决方案: 检查服务器是否运行在8080端口"
        ISSUES=$((ISSUES + 1))
    fi
fi

if [ "$ISSUES" -eq 0 ]; then
    echo -e "✅ ${GREEN}所有检查通过，系统运行正常!${NC}"
fi
echo ""

# 10. 快速操作指南
echo -e "${BLUE}10. 快速操作指南${NC}"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "查看完整文档:"
echo "  cat REFLECTION_MONITORING_GUIDE.md"
echo ""
echo "手动触发分析:"
echo "  curl -X POST http://localhost:8080/api/reflection/{TRADER_ID}/analyze \\"
echo "    -H 'Content-Type: application/json' \\"
echo "    -d '{\"type\":\"performance\"}'"
echo ""
echo "插入测试数据:"
echo "  sqlite3 data/data.db < scripts/test_data.sql"
echo ""
echo "查看实时日志:"
echo "  tail -f data/nofx_$(date +%Y-%m-%d).log | grep -i reflection"
echo ""

echo "╔════════════════════════════════════════════════════════════╗"
echo "║                    监控完成                                ║"
echo "╚════════════════════════════════════════════════════════════╝"
