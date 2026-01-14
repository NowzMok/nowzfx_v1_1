#!/bin/bash

# 自动监听文件变化并重启后端和前端

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
BACKEND_PID=""
FRONTEND_PID=""

# 颜色输出
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# 清理函数
cleanup() {
    echo -e "${YELLOW}[监听器] 正在清理进程...${NC}"
    
    if [ ! -z "$BACKEND_PID" ] && kill -0 "$BACKEND_PID" 2>/dev/null; then
        echo -e "${YELLOW}[后端] 停止后端服务 (PID: $BACKEND_PID)${NC}"
        kill $BACKEND_PID 2>/dev/null || true
        sleep 1
    fi
    
    if [ ! -z "$FRONTEND_PID" ] && kill -0 "$FRONTEND_PID" 2>/dev/null; then
        echo -e "${YELLOW}[前端] 停止前端服务 (PID: $FRONTEND_PID)${NC}"
        kill $FRONTEND_PID 2>/dev/null || true
        sleep 1
    fi
    
    echo -e "${GREEN}[完成] 所有进程已停止${NC}"
    exit 0
}

# 设置信号处理
trap cleanup SIGINT SIGTERM

# 启动后端
start_backend() {
    echo -e "${BLUE}[后端] 编译后端服务器...${NC}"
    cd "$SCRIPT_DIR"
    
    if go build -o nofx-server . 2>&1; then
        echo -e "${GREEN}[后端] 编译成功${NC}"
        
        # 杀死旧的后端进程
        if [ ! -z "$BACKEND_PID" ] && kill -0 "$BACKEND_PID" 2>/dev/null; then
            echo -e "${YELLOW}[后端] 停止旧进程${NC}"
            kill $BACKEND_PID 2>/dev/null || true
            sleep 1
        fi
        
        # 启动新的后端进程
        echo -e "${GREEN}[后端] 启动后端服务器...${NC}"
        nohup ./nofx-server > /tmp/nofx-backend.log 2>&1 &
        BACKEND_PID=$!
        echo -e "${GREEN}[后端] 后端服务已启动 (PID: $BACKEND_PID)${NC}"
        sleep 2
        
        # 验证健康检查
        if curl -s http://localhost:8080/api/health > /dev/null 2>&1; then
            echo -e "${GREEN}[后端] 健康检查通过 ✓${NC}"
        else
            echo -e "${RED}[后端] 健康检查失败${NC}"
        fi
    else
        echo -e "${RED}[后端] 编译失败${NC}"
        tail -20 /tmp/nofx-build.log 2>/dev/null || echo "查看构建日志失败"
    fi
}

# 启动前端
start_frontend() {
    echo -e "${BLUE}[前端] 启动前端开发服务器...${NC}"
    cd "$SCRIPT_DIR/web"
    
    # 杀死旧的前端进程
    if [ ! -z "$FRONTEND_PID" ] && kill -0 "$FRONTEND_PID" 2>/dev/null; then
        echo -e "${YELLOW}[前端] 停止旧进程${NC}"
        kill $FRONTEND_PID 2>/dev/null || true
        sleep 1
    fi
    
    npm run dev > /tmp/nofx-frontend.log 2>&1 &
    FRONTEND_PID=$!
    echo -e "${GREEN}[前端] 前端服务已启动 (PID: $FRONTEND_PID)${NC}"
    sleep 3
    
    # 验证前端是否运行
    if curl -s http://localhost:3000 > /dev/null 2>&1; then
        echo -e "${GREEN}[前端] 前端服务已响应 ✓${NC}"
    else
        echo -e "${YELLOW}[前端] 前端服务正在启动...${NC}"
    fi
}

# 主函数
main() {
    echo -e "${BLUE}========================================${NC}"
    echo -e "${BLUE}   NOFX 自动开发服务器${NC}"
    echo -e "${BLUE}========================================${NC}"
    echo ""
    
    # 初始启动
    start_backend
    start_frontend
    
    echo -e "${GREEN}[监听器] 监听文件变化中...${NC}"
    echo -e "${BLUE}后端日志: tail -f /tmp/nofx-backend.log${NC}"
    echo -e "${BLUE}前端日志: tail -f /tmp/nofx-frontend.log${NC}"
    echo -e "${YELLOW}按 Ctrl+C 停止监听${NC}"
    echo ""
    
    # 监听 Go 文件变化
    while true; do
        # 检查 Go 文件是否有变化（排除 vendor, bin 目录）
        CHANGED_FILES=$(find "$SCRIPT_DIR" -type f -name "*.go" \
            ! -path "*/vendor/*" \
            ! -path "*/.git/*" \
            ! -path "*/bin/*" \
            ! -path "*/__debug*" \
            -newermt '1 second ago' 2>/dev/null)
        
        if [ ! -z "$CHANGED_FILES" ]; then
            echo -e "${YELLOW}[变化检测] 检测到 Go 文件变化${NC}"
            echo "$CHANGED_FILES" | sed 's/^/  - /'
            start_backend
        fi
        
        # 检查前端文件变化（React 文件会被 Vite 自动热加载，但我们可以监听编译错误）
        if [ ! -z "$FRONTEND_PID" ] && ! kill -0 "$FRONTEND_PID" 2>/dev/null; then
            echo -e "${RED}[前端] 前端进程已停止，重启中...${NC}"
            start_frontend
        fi
        
        # 检查后端是否崩溃
        if [ ! -z "$BACKEND_PID" ] && ! kill -0 "$BACKEND_PID" 2>/dev/null; then
            echo -e "${RED}[后端] 后端进程已停止，重启中...${NC}"
            start_backend
        fi
        
        sleep 2
    done
}

# 启动
main
