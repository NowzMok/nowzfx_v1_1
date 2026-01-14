#!/bin/bash

##############################################################################
# 准备源代码脚本
# 功能：将 NOFX 项目复制到 source/ 目录
# 使用说明：bash prepare_source.sh /path/to/nofx
##############################################################################

set -e

# 颜色定义
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m'

echo -e "${BLUE}========================================${NC}"
echo -e "${BLUE}NOFX 源代码准备脚本${NC}"
echo -e "${BLUE}========================================${NC}"
echo ""

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PACKAGE_DIR="$SCRIPT_DIR"
SOURCE_DEST="$PACKAGE_DIR/source/nofx"

# 检查参数
if [ -z "$1" ]; then
    echo -e "${YELLOW}使用方法:${NC}"
    echo "  bash prepare_source.sh /path/to/nofx"
    echo ""
    echo -e "${YELLOW}示例:${NC}"
    echo "  bash prepare_source.sh ~/Desktop/圣灵/nonowz/nofx"
    echo ""
    exit 1
fi

SOURCE_PATH="$1"

# 验证源目录
if [ ! -d "$SOURCE_PATH" ]; then
    echo -e "${RED}✗ 错误：源目录不存在${NC}"
    echo "  期望位置: $SOURCE_PATH"
    exit 1
fi

if [ ! -f "$SOURCE_PATH/go.mod" ]; then
    echo -e "${RED}✗ 错误：不是有效的 NOFX 项目目录${NC}"
    echo "  缺失 go.mod 文件"
    exit 1
fi

echo -e "${GREEN}✓ 找到 NOFX 项目${NC}"
echo ""

# 检查目标目录
if [ -d "$SOURCE_DEST" ]; then
    echo -e "${YELLOW}⚠️  目标目录已存在: $SOURCE_DEST${NC}"
    echo "是否覆盖? (y/n)"
    read -r response
    if [ "$response" != "y" ]; then
        echo "已取消"
        exit 0
    fi
    rm -rf "$SOURCE_DEST"
fi

echo -e "${BLUE}[1/3] 复制源代码...${NC}"
mkdir -p "$PACKAGE_DIR/source"
cp -r "$SOURCE_PATH" "$SOURCE_DEST"
echo -e "${GREEN}✓ 源代码已复制${NC}"
echo ""

# 计算大小
echo -e "${BLUE}[2/3] 计算项目大小...${NC}"
SOURCE_SIZE=$(du -sh "$SOURCE_DEST" | awk '{print $1}')
FILE_COUNT=$(find "$SOURCE_DEST" -type f | wc -l)
echo -e "${GREEN}✓ 项目大小: $SOURCE_SIZE ($FILE_COUNT 个文件)${NC}"
echo ""

# 清理 Git 历史（可选，节省空间）
echo -e "${BLUE}[3/3] 清理不必要文件...${NC}"

# 删除 Git 历史（如果存在）
if [ -d "$SOURCE_DEST/.git" ]; then
    echo -e "${YELLOW}   删除 .git 目录...${NC}"
    rm -rf "$SOURCE_DEST/.git"
    CLEANED_SIZE=$(du -sh "$SOURCE_DEST" | awk '{print $1}')
    echo -e "${GREEN}✓ 清理后大小: $CLEANED_SIZE${NC}"
fi

# 删除 node_modules（如果存在）
if [ -d "$SOURCE_DEST/web/node_modules" ]; then
    echo -e "${YELLOW}   删除 web/node_modules...${NC}"
    rm -rf "$SOURCE_DEST/web/node_modules"
fi

# 删除其他缓存
find "$SOURCE_DEST" -name "__pycache__" -type d -exec rm -rf {} + 2>/dev/null || true
find "$SOURCE_DEST" -name ".DS_Store" -delete 2>/dev/null || true

echo ""

echo -e "${GREEN}========================================${NC}"
echo -e "${GREEN}源代码准备完成！${NC}"
echo -e "${GREEN}========================================${NC}"
echo ""
echo -e "源代码位置: ${YELLOW}$SOURCE_DEST${NC}"
echo ""
echo "下一步:"
echo "1. 验证包: bash check_package.sh"
echo "2. 安装:   bash scripts/install.sh"
echo ""
