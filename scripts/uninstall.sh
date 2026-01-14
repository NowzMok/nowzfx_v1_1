#!/bin/bash

##############################################################################
# NOFX 卸载脚本
# 功能：安全地卸载 NOFX，保留数据目录
##############################################################################

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PACKAGE_DIR="$(dirname "$SCRIPT_DIR")"

# 颜色定义
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m'

echo -e "${BLUE}========================================${NC}"
echo -e "${BLUE}NOFX 卸载${NC}"
echo -e "${BLUE}========================================${NC}"
echo ""

echo -e "${YELLOW}⚠️  警告：此操作将停止并删除 Docker 容器和镜像${NC}"
echo -e "${YELLOW}数据文件将被保留在 data/ 目录中${NC}"
echo ""
echo "确认卸载? (y/N)"
read -r response

if [ "$response" != "y" ]; then
    echo -e "${YELLOW}已取消${NC}"
    exit 0
fi

cd "$PACKAGE_DIR"

echo -e "${BLUE}[1/3] 停止服务...${NC}"
docker-compose down 2>/dev/null || true
echo -e "${GREEN}✓ 服务已停止${NC}"
echo ""

echo -e "${BLUE}[2/3] 删除镜像...${NC}"
docker rmi nofx-backend:latest 2>/dev/null || true
docker rmi nofx-backend:1.0.0 2>/dev/null || true
docker rmi nofx-frontend:latest 2>/dev/null || true
docker rmi nofx-frontend:1.0.0 2>/dev/null || true
echo -e "${GREEN}✓ 镜像已删除${NC}"
echo ""

echo -e "${BLUE}[3/3] 清理环境...${NC}"
echo -e "${YELLOW}   数据目录已保留: $PACKAGE_DIR/data${NC}"
echo -e "${YELLOW}   配置文件已保留: $PACKAGE_DIR/.env${NC}"
echo ""

echo -e "${GREEN}========================================${NC}"
echo -e "${GREEN}卸载完成${NC}"
echo -e "${GREEN}========================================${NC}"
echo ""
