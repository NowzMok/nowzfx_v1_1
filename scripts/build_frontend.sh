#!/bin/bash

##############################################################################
# NOFX Node.js 前端离线构建脚本
# 功能：在离线环境中编译 Node.js 前端
##############################################################################

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PACKAGE_DIR="$(dirname "$SCRIPT_DIR")"
SOURCE_DIR="$PACKAGE_DIR/source/nofx/web"
OUTPUT_DIR="$PACKAGE_DIR/build/frontend"

# 颜色定义
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m'

echo -e "${BLUE}========================================${NC}"
echo -e "${BLUE}Node.js 前端离线构建${NC}"
echo -e "${BLUE}========================================${NC}"
echo ""

# 检查源代码
if [ ! -d "$SOURCE_DIR" ]; then
    echo -e "${RED}✗ 错误：未找到前端源代码目录${NC}"
    echo "   期望位置: $SOURCE_DIR"
    exit 1
fi

# 检查 Node.js 版本
if ! command -v node &> /dev/null; then
    echo -e "${RED}✗ 错误：Node.js 未安装${NC}"
    echo "请安装 Node.js 20 或更高版本"
    exit 1
fi

NODE_VERSION=$(node --version)
NPM_VERSION=$(npm --version)

echo -e "${GREEN}✓ Node.js 版本: $NODE_VERSION${NC}"
echo -e "${GREEN}✓ npm 版本: $NPM_VERSION${NC}"
echo ""

# 创建输出目录
mkdir -p "$OUTPUT_DIR"

# 进入前端目录
cd "$SOURCE_DIR"
echo -e "${GREEN}✓ 进入前端源代码目录${NC}"
echo ""

echo -e "${BLUE}[1/4] 检查依赖文件...${NC}"
if [ ! -f "package.json" ]; then
    echo -e "${RED}✗ 找不到 package.json 文件${NC}"
    exit 1
fi

if [ -f "package-lock.json" ]; then
    echo -e "${YELLOW}   发现 package-lock.json${NC}"
else
    echo -e "${YELLOW}   未找到 package-lock.json，将创建${NC}"
fi
echo ""

echo -e "${BLUE}[2/4] 安装依赖...${NC}"
if [ -f "package-lock.json" ]; then
    echo -e "${YELLOW}   使用 npm ci (精确版本)${NC}"
    npm ci --offline 2>/dev/null || npm install --legacy-peer-deps
else
    echo -e "${YELLOW}   使用 npm install${NC}"
    npm install --legacy-peer-deps
fi
echo -e "${GREEN}✓ 依赖安装完成${NC}"
echo ""

echo -e "${BLUE}[3/4] 编译前端...${NC}"
if [ -f "package.json" ] && grep -q '"build"' package.json; then
    echo -e "${YELLOW}   构建前端应用...${NC}"
    npm run build
    
    if [ -d "dist" ]; then
        echo -e "${GREEN}✓ 编译成功${NC}"
        BUILD_SIZE=$(du -sh dist | awk '{print $1}')
        echo -e "${YELLOW}   构建输出大小: $BUILD_SIZE${NC}"
    else
        echo -e "${YELLOW}⚠️  未找到输出目录 'dist'${NC}"
    fi
else
    echo -e "${YELLOW}⚠️  未找到 build 脚本，跳过编译${NC}"
fi
echo ""

echo -e "${BLUE}[4/4] 准备部署文件...${NC}"
# 复制编译后的文件到输出目录
if [ -d "dist" ]; then
    cp -r dist "$OUTPUT_DIR/dist" 2>/dev/null || true
    echo -e "${GREEN}✓ 编译文件已复制到 $OUTPUT_DIR${NC}"
fi

# 复制 package.json 和 package-lock.json 用于生产构建
cp package.json "$OUTPUT_DIR/" 2>/dev/null || true
[ -f "package-lock.json" ] && cp package-lock.json "$OUTPUT_DIR/" || true

echo ""

echo -e "${GREEN}========================================${NC}"
echo -e "${GREEN}前端构建完成！${NC}"
echo -e "${GREEN}========================================${NC}"
echo ""
echo -e "输出位置: ${YELLOW}$OUTPUT_DIR${NC}"
echo ""
