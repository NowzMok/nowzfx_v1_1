#!/bin/bash

##############################################################################
# NOFX 完整离线安装脚本
# 功能：一键安装整个 NOFX 系统（包括从源代码构建）
##############################################################################

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PACKAGE_DIR="$(dirname "$SCRIPT_DIR")"

# 颜色定义
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m'

echo -e "${BLUE}========================================${NC}"
echo -e "${BLUE}NOFX 完整离线安装${NC}"
echo -e "${BLUE}========================================${NC}"
echo ""

# 检查权限
if [ "$EUID" -ne 0 ]; then 
    echo -e "${YELLOW}⚠️  建议以 root 身份运行此脚本${NC}"
    echo "继续安装? (y/n) "
    read -r response
    if [ "$response" != "y" ]; then
        exit 1
    fi
    echo ""
fi

# 步骤 1: 准备构建环境
echo -e "${BLUE}[1/5] 准备构建环境...${NC}"
bash "$SCRIPT_DIR/prepare_offline_build.sh" || {
    echo -e "${YELLOW}⚠️  准备步骤出现问题，继续...${NC}"
}
echo ""

# 步骤 2: 构建后端
echo -e "${BLUE}[2/5] 构建 Go 后端...${NC}"
bash "$SCRIPT_DIR/build_backend.sh" || {
    echo -e "${RED}✗ 后端构建失败${NC}"
    exit 1
}
echo ""

# 步骤 3: 构建前端
echo -e "${BLUE}[3/5] 构建 Node.js 前端...${NC}"
bash "$SCRIPT_DIR/build_frontend.sh" || {
    echo -e "${RED}✗ 前端构建失败${NC}"
    exit 1
}
echo ""

# 步骤 4: 构建 Docker 镜像
echo -e "${BLUE}[4/5] 构建 Docker 镜像...${NC}"
bash "$SCRIPT_DIR/build_docker_images.sh" || {
    echo -e "${RED}✗ Docker 镜像构建失败${NC}"
    exit 1
}
echo ""

# 步骤 5: 启动服务
echo -e "${BLUE}[5/5] 启动 NOFX 服务...${NC}"

# 检查 docker-compose
if ! command -v docker-compose &> /dev/null; then
    echo -e "${RED}✗ 错误：docker-compose 未安装${NC}"
    exit 1
fi

# 进入包目录
cd "$PACKAGE_DIR"

# 创建配置文件
if [ ! -f ".env" ]; then
    echo -e "${YELLOW}   创建 .env 配置文件...${NC}"
    if [ -f "config/.env.example" ]; then
        cp config/.env.example .env
        echo -e "${GREEN}✓ 配置文件已创建${NC}"
    fi
fi

# 启动服务
echo -e "${YELLOW}   启动 Docker Compose...${NC}"
docker-compose up -d

# 等待服务启动
echo -e "${YELLOW}   等待服务启动...${NC}"
sleep 5

# 检查服务状态
echo -e "${YELLOW}   检查服务状态...${NC}"
docker-compose ps

echo ""
echo -e "${GREEN}========================================${NC}"
echo -e "${GREEN}NOFX 安装完成！${NC}"
echo -e "${GREEN}========================================${NC}"
echo ""
echo -e "服务地址:"
echo -e "  前端: ${YELLOW}http://localhost:3000${NC}"
echo -e "  后端: ${YELLOW}http://localhost:8080${NC}"
echo ""
echo -e "常用命令:"
echo -e "  查看日志: ${YELLOW}docker-compose logs -f${NC}"
echo -e "  停止服务: ${YELLOW}docker-compose down${NC}"
echo -e "  重启服务: ${YELLOW}docker-compose restart${NC}"
echo ""
