#!/bin/bash

##############################################################################
# NOFX 完整离线构建准备脚本
# 功能：准备所有源代码和依赖，为离线编译构建做准备
##############################################################################

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PACKAGE_DIR="$(dirname "$SCRIPT_DIR")"
SOURCE_DIR="$PACKAGE_DIR/source"

# 颜色定义
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m'

echo -e "${BLUE}========================================${NC}"
echo -e "${BLUE}NOFX 离线构建准备${NC}"
echo -e "${BLUE}========================================${NC}"
echo ""

# 检查是否已经在源目录中
if [ ! -d "$SOURCE_DIR/nofx" ]; then
    echo -e "${YELLOW}⚠️  警告：未找到源代码目录${NC}"
    echo "此脚本需要在离线设备上执行，源代码应该已经在 source/ 目录中"
    exit 1
fi

cd "$SOURCE_DIR/nofx"

echo -e "${GREEN}✓ 进入源代码目录${NC}"
echo ""

# 1. 检查 Go 环境
echo -e "${BLUE}[1/6] 检查 Go 环境...${NC}"
if ! command -v go &> /dev/null; then
    echo -e "${RED}✗ 未找到 Go${NC}"
    echo "请先安装 Go 1.25 或更高版本"
    exit 1
fi

GO_VERSION=$(go version | awk '{print $3}')
echo -e "${GREEN}✓ Go 已安装: $GO_VERSION${NC}"
echo ""

# 2. 检查 Node.js 环境
echo -e "${BLUE}[2/6] 检查 Node.js 环境...${NC}"
if ! command -v node &> /dev/null; then
    echo -e "${RED}✗ 未找到 Node.js${NC}"
    echo "请先安装 Node.js 20 或更高版本"
    exit 1
fi

NODE_VERSION=$(node --version)
echo -e "${GREEN}✓ Node.js 已安装: $NODE_VERSION${NC}"
echo ""

# 3. Go 依赖准备
echo -e "${BLUE}[3/6] 准备 Go 模块依赖...${NC}"
if [ -f "go.mod" ]; then
    echo "   下载 Go 依赖..."
    go mod download || {
        echo -e "${YELLOW}⚠️  部分依赖下载失败，继续...${NC}"
    }
    echo "   整理 Go 模块..."
    go mod tidy
    echo -e "${GREEN}✓ Go 依赖已准备${NC}"
else
    echo -e "${YELLOW}⚠️  未找到 go.mod 文件${NC}"
fi
echo ""

# 4. Node.js 依赖准备
echo -e "${BLUE}[4/6] 准备 Node.js 前端依赖...${NC}"
if [ -d "web" ] && [ -f "web/package.json" ]; then
    cd web
    echo "   安装前端依赖..."
    if [ -f "package-lock.json" ]; then
        npm ci --offline 2>/dev/null || npm install
    else
        npm install
    fi
    echo -e "${GREEN}✓ 前端依赖已安装${NC}"
    cd ..
else
    echo -e "${YELLOW}⚠️  未找到前端项目${NC}"
fi
echo ""

# 5. 显示构建大小
echo -e "${BLUE}[5/6] 计算项目大小...${NC}"
PROJECT_SIZE=$(du -sh . | awk '{print $1}')
echo -e "${GREEN}✓ 项目总大小: $PROJECT_SIZE${NC}"
echo ""

# 6. 生成构建清单
echo -e "${BLUE}[6/6] 生成构建清单...${NC}"
cat > "$PACKAGE_DIR/BUILD_MANIFEST.txt" << 'EOF'
NOFX 离线构建清单
===================

项目结构：
├── source/nofx/          # 完整的源代码
├── scripts/              # 构建和安装脚本
├── config/               # 配置文件模板
├── BUILD_MANIFEST.txt    # 本文件
└── OFFLINE_BUILD_GUIDE.md # 详细构建指南

源代码包含：
- Go 后端 (backend)
  - api/        - API 接口定义
  - auth/       - 认证模块
  - trader/     - 交易逻辑
  - market/     - 市场数据
  - monitor/    - 监控系统
  - 其他模块...

- Node.js 前端 (web)
  - src/        - React 源代码
  - public/     - 静态资源
  - Dockerfile  - Docker 构建文件

构建步骤：
1. bash scripts/build_backend.sh    - 构建 Go 后端
2. bash scripts/build_frontend.sh   - 构建 Node.js 前端
3. bash scripts/build_docker_images.sh - 构建 Docker 镜像

EOF
echo -e "${GREEN}✓ 构建清单已生成${NC}"
echo ""

echo -e "${GREEN}========================================${NC}"
echo -e "${GREEN}离线构建准备完成！${NC}"
echo -e "${GREEN}========================================${NC}"
echo ""
echo "下一步："
echo "1. 复制此文件夹到离线设备"
echo "2. 在离线设备上运行: bash scripts/build_backend.sh"
echo "3. 然后运行: bash scripts/build_frontend.sh"
echo "4. 最后运行: bash scripts/install.sh"
