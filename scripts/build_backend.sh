#!/bin/bash

##############################################################################
# NOFX Go 后端离线构建脚本
# 功能：在离线环境中编译 Go 后端
##############################################################################

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PACKAGE_DIR="$(dirname "$SCRIPT_DIR")"
SOURCE_DIR="$PACKAGE_DIR/source/nofx"
OUTPUT_DIR="$PACKAGE_DIR/build/backend"

# 颜色定义
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m'

echo -e "${BLUE}========================================${NC}"
echo -e "${BLUE}Go 后端离线构建${NC}"
echo -e "${BLUE}========================================${NC}"
echo ""

# 检查源代码
if [ ! -d "$SOURCE_DIR" ]; then
    echo -e "${RED}✗ 错误：未找到源代码目录${NC}"
    echo "   期望位置: $SOURCE_DIR"
    exit 1
fi

# 检查 Go 版本
if ! command -v go &> /dev/null; then
    echo -e "${RED}✗ 错误：Go 未安装${NC}"
    echo "请安装 Go 1.25 或更高版本"
    exit 1
fi

GO_VERSION=$(go version | awk '{print $3}')
echo -e "${GREEN}✓ Go 版本: $GO_VERSION${NC}"
echo ""

# 创建输出目录
mkdir -p "$OUTPUT_DIR"

# 进入源目录
cd "$SOURCE_DIR"
echo -e "${GREEN}✓ 进入源代码目录${NC}"
echo ""

# 获取构建信息
BUILD_TIME=$(date '+%Y-%m-%d %H:%M:%S')
COMMIT_SHA=$(git rev-parse --short HEAD 2>/dev/null || echo "unknown")
VERSION="1.0.0"

# 构建变量
BUILD_FLAGS="-X 'main.Version=$VERSION' -X 'main.BuildTime=$BUILD_TIME' -X 'main.CommitSHA=$COMMIT_SHA'"

echo -e "${BLUE}[1/4] 检查依赖...${NC}"
if [ ! -f "go.mod" ]; then
    echo -e "${RED}✗ 找不到 go.mod 文件${NC}"
    exit 1
fi

echo -e "${YELLOW}   go version: $GO_VERSION${NC}"
echo -e "${YELLOW}   build version: $VERSION${NC}"
echo -e "${YELLOW}   build time: $BUILD_TIME${NC}"
echo ""

echo -e "${BLUE}[2/4] 下载依赖模块...${NC}"
go mod download || {
    echo -e "${YELLOW}⚠️  部分模块可能离线，继续...${NC}"
}
echo ""

echo -e "${BLUE}[3/4] 编译 Go 程序...${NC}"
# 根据系统选择输出文件名
if [[ "$OSTYPE" == "linux-gnu"* ]]; then
    OUTPUT_FILE="$OUTPUT_DIR/nofx-server"
    echo -e "${YELLOW}   目标平台: Linux${NC}"
elif [[ "$OSTYPE" == "darwin"* ]]; then
    OUTPUT_FILE="$OUTPUT_DIR/nofx-server-darwin"
    echo -e "${YELLOW}   目标平台: macOS${NC}"
else
    OUTPUT_FILE="$OUTPUT_DIR/nofx-server"
fi

go build \
    -v \
    -ldflags "$BUILD_FLAGS" \
    -o "$OUTPUT_FILE" \
    .

if [ $? -eq 0 ]; then
    echo -e "${GREEN}✓ 编译成功${NC}"
    BINARY_SIZE=$(du -h "$OUTPUT_FILE" | awk '{print $1}')
    echo -e "${YELLOW}   二进制大小: $BINARY_SIZE${NC}"
else
    echo -e "${RED}✗ 编译失败${NC}"
    exit 1
fi
echo ""

echo -e "${BLUE}[4/4] 验证二进制...${NC}"
if [ -x "$OUTPUT_FILE" ]; then
    echo -e "${GREEN}✓ 二进制可执行${NC}"
    "$OUTPUT_FILE" --version 2>/dev/null || echo -e "${YELLOW}   (版本信息可能不可用)${NC}"
else
    echo -e "${YELLOW}⚠️  二进制不可执行，尝试修复...${NC}"
    chmod +x "$OUTPUT_FILE"
fi
echo ""

echo -e "${GREEN}========================================${NC}"
echo -e "${GREEN}Go 后端构建完成！${NC}"
echo -e "${GREEN}========================================${NC}"
echo ""
echo -e "输出位置: ${YELLOW}$OUTPUT_FILE${NC}"
echo ""
