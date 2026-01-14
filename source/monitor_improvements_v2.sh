#!/bin/bash

# 🎯 NOFX 改进监控仪表板 v2.0
# 实时展示9项改进的运行状态

TRADER_ID="default_trader"
API_URL="http://localhost:8080"

# 彩色输出
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
MAGENTA='\033[0;35m'
NC='\033[0m'

clear_screen() {
    clear
}

show_header() {
    echo -e "${CYAN}"
    echo "╔════════════════════════════════════════════════════════════════╗"
    echo "║         🚀 NOFX Trading System - Improvements Dashboard v2.0   ║"
    echo "║              Enhanced Error Logging & Monitoring               ║"
    echo "╚════════════════════════════════════════════════════════════════╝"
    echo -e "${NC}"
}

check_health() {
    HEALTH=$(curl -s "${API_URL}/api/health" 2>/dev/null)
    
    if [ -z "$HEALTH" ]; then
        echo -e "${RED}❌ Cannot connect to API server${NC}"
        return 1
    fi
    
    echo -e "${GREEN}✅ System Healthy${NC}"
    return 0
}

show_improvements() {
    echo -e "${BLUE}┌─ 9项改进部署清单 ─────────────────────────────────────┐${NC}"
    
    echo -e "${MAGENTA}【高优先级】${NC}"
    echo -e "${GREEN}  ✓${NC} 1. 订单去重优化 - 30分钟时间窗口检查FILLED订单"
    echo -e "${GREEN}  ✓${NC} 2. 动态止损时间策略 - 波动性自适应(3-10分钟)"
    echo -e "${GREEN}  ✓${NC} 3. TP/SL同步验证 - 每5个周期验证一次"
    
    echo -e "${MAGENTA}【中优先级】${NC}"
    echo -e "${GREEN}  ✓${NC} 4. 重试指数退避 - 5次重试(2s→4s→8s→16s→32s)"
    echo -e "${GREEN}  ✓${NC} 5. 动态TP追踪 - 2%利润阈值,50%价格跟随"
    echo -e "${YELLOW}  ◑${NC} 6. 完善错误日志和监控 - 🔴 实现中..."
    
    echo -e "${MAGENTA}【低优先级】${NC}"
    echo -e "  ⭕ 7. 订单分析统计面板 - 计划中"
    echo -e "  ⭕ 8. TP/SL可视化编辑 - 计划中"
    echo -e "  ⭕ 9. 策略回测对比功能 - 计划中"
    
    echo -e "${BLUE}└───────────────────────────────────────────────────────┘${NC}"
}

show_error_monitoring() {
    echo ""
    echo -e "${BLUE}┌─ 错误监控统计 ────────────────────────────────────────┐${NC}"
    
    # 获取错误统计
    STATS=$(curl -s "${API_URL}/api/error-stats" 2>/dev/null)
    
    if [ ! -z "$STATS" ]; then
        ERROR_TYPES=$(echo "$STATS" | jq '.total_error_types' 2>/dev/null)
        if [ ! -z "$ERROR_TYPES" ] && [ "$ERROR_TYPES" != "null" ]; then
            echo -e "  ${YELLOW}错误类型总数:${NC} ${CYAN}${ERROR_TYPES}${NC}"
        fi
    fi
    
    # 获取错误率
    RATE=$(curl -s "${API_URL}/api/error-rate" 2>/dev/null | jq '.error_rate_per_minute' 2>/dev/null)
    if [ ! -z "$RATE" ] && [ "$RATE" != "null" ]; then
        if (( $(echo "$RATE > 0" | bc -l) )); then
            echo -e "  ${YELLOW}每分钟错误数:${NC} ${RED}${RATE}${NC}"
        else
            echo -e "  ${YELLOW}每分钟错误数:${NC} ${GREEN}${RATE}${NC}"
        fi
    fi
    
    # 获取最近错误
    RECENT=$(curl -s "${API_URL}/api/recent-errors?count=3" 2>/dev/null)
    if [ ! -z "$RECENT" ]; then
        COUNT=$(echo "$RECENT" | jq '.count' 2>/dev/null)
        if [ ! -z "$COUNT" ] && [ "$COUNT" != "null" ] && [ "$COUNT" != "0" ]; then
            echo -e "  ${YELLOW}最近错误数:${NC} ${RED}${COUNT}${NC}"
            
            # 显示最近的错误
            echo ""
            echo -e "  ${CYAN}最近的错误记录:${NC}"
            echo "$RECENT" | jq -r '.errors[] | "    [\(.severity)] \(.timestamp) - \(.error_type): \(.message)"' 2>/dev/null | head -3
        else
            echo -e "  ${YELLOW}最近错误数:${NC} ${GREEN}0 (无错误)${NC}"
        fi
    fi
    
    echo -e "${BLUE}└───────────────────────────────────────────────────────┘${NC}"
}

show_detailed_stats() {
    echo ""
    echo -e "${BLUE}┌─ 详细的错误分类统计 ──────────────────────────────────┐${NC}"
    
    STATS=$(curl -s "${API_URL}/api/error-stats" 2>/dev/null)
    
    if [ ! -z "$STATS" ]; then
        # 解析每种错误类型的统计
        echo "$STATS" | jq -r '.stats | to_entries[] | 
            "  \(.value.error_type): \(.value.count) 次 | 最后: \(.value.last_seen)"' 2>/dev/null | head -5
    fi
    
    echo -e "${BLUE}└───────────────────────────────────────────────────────┘${NC}"
}

show_improvement_details() {
    echo ""
    echo -e "${BLUE}┌─ 改进6详情：错误日志和监控 ──────────────────────────┐${NC}"
    
    echo -e "${CYAN}功能描述:${NC}"
    echo "  ✓ 创建了 ErrorTracker 工具类"
    echo "  ✓ 支持错误分类记录 (RETRY/SYNC/EXECUTION/VALIDATION)"
    echo "  ✓ 实现了错误统计和追踪"
    echo "  ✓ 添加了错误查询API接口"
    echo "  ✓ 集成到订单执行重试逻辑中"
    echo "  ✓ 集成到TP/SL同步验证中"
    
    echo ""
    echo -e "${CYAN}新增API接口:${NC}"
    echo "  • GET /api/error-stats - 获取错误统计"
    echo "  • GET /api/recent-errors - 获取最近的错误"
    echo "  • GET /api/error-report - 生成完整报告"
    echo "  • GET /api/error-rate - 获取错误率"
    echo "  • POST /api/clear-errors - 清除统计数据"
    
    echo -e "${BLUE}└───────────────────────────────────────────────────────┘${NC}"
}

show_next_steps() {
    echo ""
    echo -e "${MAGENTA}📋 接下来的工作:${NC}"
    echo "  1. ⭕ 测试错误追踪功能 (在实际交易中观察)"
    echo "  2. ⭕ 集成错误统计到前端仪表板"
    echo "  3. ⭕ 实现告警机制 (错误率超过阈值)"
    echo "  4. ⭕ 添加低优先级功能 (7-9项)"
}

# 主循环
while true; do
    clear_screen
    show_header
    
    if ! check_health; then
        echo -e "${RED}等待API服务启动...${NC}"
        sleep 5
        continue
    fi
    
    echo ""
    show_improvements
    show_error_monitoring
    show_detailed_stats
    show_improvement_details
    show_next_steps
    
    echo ""
    echo -e "${YELLOW}📊 更新时间: $(date '+%Y-%m-%d %H:%M:%S')${NC}"
    echo -e "${CYAN}⏱️  下次刷新: 10秒后${NC}"
    echo -e "${CYAN}💡 提示: 按 Ctrl+C 退出监控${NC}"
    
    sleep 10
done
