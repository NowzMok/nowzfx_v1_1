#!/bin/bash

##############################################################################
# NOFX Docker 镜像离线构建脚本
# 功能：在编译好源代码后，构建 Docker 镜像
##############################################################################

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PACKAGE_DIR="$(dirname "$SCRIPT_DIR")"
SOURCE_DIR="$PACKAGE_DIR/source/nofx"
BUILD_DIR="$PACKAGE_DIR/build"
IMAGES_DIR="$PACKAGE_DIR/docker_images"

# 颜色定义
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m'

echo -e "${BLUE}========================================${NC}"
echo -e "${BLUE}Docker 镜像离线构建${NC}"
echo -e "${BLUE}========================================${NC}"
echo ""

# 检查 Docker
if ! command -v docker &> /dev/null; then
    echo -e "${RED}✗ 错误：Docker 未安装或不在 PATH 中${NC}"
    echo "请先安装 Docker"
    exit 1
fi

DOCKER_VERSION=$(docker --version)
echo -e "${GREEN}✓ $DOCKER_VERSION${NC}"
echo ""

# 检查 Docker Daemon
if ! docker info &> /dev/null; then
    echo -e "${RED}✗ 错误：Docker 守护进程未运行${NC}"
    echo "请启动 Docker"
    exit 1
fi

echo -e "${GREEN}✓ Docker 守护进程正在运行${NC}"
echo ""

# 创建镜像输出目录
mkdir -p "$IMAGES_DIR"

echo -e "${BLUE}[1/3] 检查源代码...${NC}"
if [ ! -d "$SOURCE_DIR" ]; then
    echo -e "${RED}✗ 错误：未找到源代码${NC}"
    exit 1
fi

if [ ! -d "$SOURCE_DIR/docker" ]; then
    echo -e "${RED}✗ 错误：未找到 docker 目录${NC}"
    exit 1
fi

echo -e "${GREEN}✓ 源代码目录完整${NC}"
echo ""

echo -e "${BLUE}[2/3] 构建 Docker 镜像...${NC}"

# 构建后端镜像
echo -e "${YELLOW}   构建后端镜像 (nofx-backend:latest)...${NC}"
cd "$SOURCE_DIR"

if [ -f "docker/Dockerfile.backend" ]; then
    docker build \
        -f docker/Dockerfile.backend \
        -t nofx-backend:latest \
        -t nofx-backend:1.0.0 \
        --progress=plain \
        .
    
    if [ $? -eq 0 ]; then
        echo -e "${GREEN}✓ 后端镜像构建成功${NC}"
    else
        echo -e "${RED}✗ 后端镜像构建失败${NC}"
        exit 1
    fi
else
    echo -e "${RED}✗ 找不到 docker/Dockerfile.backend${NC}"
    exit 1
fi

echo ""

# 构建前端镜像
echo -e "${YELLOW}   构建前端镜像 (nofx-frontend:latest)...${NC}"

if [ -f "docker/Dockerfile.frontend" ]; then
    docker build \
        -f docker/Dockerfile.frontend \
        -t nofx-frontend:latest \
        -t nofx-frontend:1.0.0 \
        --progress=plain \
        .
    
    if [ $? -eq 0 ]; then
        echo -e "${GREEN}✓ 前端镜像构建成功${NC}"
    else
        echo -e "${RED}✗ 前端镜像构建失败${NC}"
        exit 1
    fi
else
    echo -e "${RED}✗ 找不到 docker/Dockerfile.frontend${NC}"
    exit 1
fi

echo ""

echo -e "${BLUE}[3/3] 导出 Docker 镜像...${NC}"

# 导出后端镜像
echo -e "${YELLOW}   导出后端镜像...${NC}"
BACKEND_IMAGE="$IMAGES_DIR/nofx-backend-1.0.0.tar.gz"
docker save nofx-backend:latest | gzip > "$BACKEND_IMAGE"

if [ -f "$BACKEND_IMAGE" ]; then
    BACKEND_SIZE=$(du -h "$BACKEND_IMAGE" | awk '{print $1}')
    echo -e "${GREEN}✓ 后端镜像已导出${NC}"
    echo -e "${YELLOW}   文件大小: $BACKEND_SIZE${NC}"
else
    echo -e "${RED}✗ 后端镜像导出失败${NC}"
    exit 1
fi

echo ""

# 导出前端镜像
echo -e "${YELLOW}   导出前端镜像...${NC}"
FRONTEND_IMAGE="$IMAGES_DIR/nofx-frontend-1.0.0.tar.gz"
docker save nofx-frontend:latest | gzip > "$FRONTEND_IMAGE"

if [ -f "$FRONTEND_IMAGE" ]; then
    FRONTEND_SIZE=$(du -h "$FRONTEND_IMAGE" | awk '{print $1}')
    echo -e "${GREEN}✓ 前端镜像已导出${NC}"
    echo -e "${YELLOW}   文件大小: $FRONTEND_SIZE${NC}"
else
    echo -e "${RED}✗ 前端镜像导出失败${NC}"
    exit 1
fi

echo ""

echo -e "${GREEN}========================================${NC}"
echo -e "${GREEN}Docker 镜像构建完成！${NC}"
echo -e "${GREEN}========================================${NC}"
echo ""
echo -e "镜像位置:"
echo -e "${YELLOW}  后端: $BACKEND_IMAGE${NC}"
echo -e "${YELLOW}  前端: $FRONTEND_IMAGE${NC}"
echo ""
