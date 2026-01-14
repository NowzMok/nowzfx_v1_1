#!/bin/bash

##############################################################################
# NOFX 包验证脚本
# 功能：验证离线包的完整性
##############################################################################

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PACKAGE_DIR="$SCRIPT_DIR"

# 颜色定义
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m'

echo -e "${BLUE}========================================${NC}"
echo -e "${BLUE}NOFX 离线包验证${NC}"
echo -e "${BLUE}========================================${NC}"
echo ""

TOTAL=0
PASSED=0

# 检查函数
check_file() {
    local file=$1
    local name=$2
    
    TOTAL=$((TOTAL + 1))
    
    if [ -f "$file" ]; then
        SIZE=$(du -h "$file" | awk '{print $1}')
        echo -e "${GREEN}✓${NC} $name (大小: $SIZE)"
        PASSED=$((PASSED + 1))
    else
        echo -e "${RED}✗${NC} $name (缺失)"
    fi
}

check_dir() {
    local dir=$1
    local name=$2
    
    TOTAL=$((TOTAL + 1))
    
    if [ -d "$dir" ]; then
        FILE_COUNT=$(find "$dir" -type f | wc -l)
        echo -e "${GREEN}✓${NC} $name (包含 $FILE_COUNT 个文件)"
        PASSED=$((PASSED + 1))
    else
        echo -e "${RED}✗${NC} $name (缺失)"
    fi
}

# 检查目录结构
echo -e "${BLUE}[1/4] 检查目录结构...${NC}"
check_dir "$PACKAGE_DIR/scripts" "scripts/ 目录"
check_dir "$PACKAGE_DIR/config" "config/ 目录"
check_dir "$PACKAGE_DIR/source" "source/ 源代码目录"
echo ""

# 检查脚本文件
echo -e "${BLUE}[2/4] 检查脚本文件...${NC}"
check_file "$PACKAGE_DIR/scripts/install.sh" "install.sh"
check_file "$PACKAGE_DIR/scripts/uninstall.sh" "uninstall.sh"
check_file "$PACKAGE_DIR/scripts/start.sh" "start.sh"
check_file "$PACKAGE_DIR/scripts/build_backend.sh" "build_backend.sh"
check_file "$PACKAGE_DIR/scripts/build_frontend.sh" "build_frontend.sh"
check_file "$PACKAGE_DIR/scripts/build_docker_images.sh" "build_docker_images.sh"
check_file "$PACKAGE_DIR/scripts/prepare_offline_build.sh" "prepare_offline_build.sh"
check_file "$PACKAGE_DIR/check_package.sh" "check_package.sh"
echo ""

# 检查配置文件
echo -e "${BLUE}[3/4] 检查配置文件...${NC}"
check_file "$PACKAGE_DIR/docker-compose.yml" "docker-compose.yml"
check_file "$PACKAGE_DIR/config/.env.example" ".env.example"
check_file "$PACKAGE_DIR/README.md" "README.md"
check_file "$PACKAGE_DIR/QUICK_START.md" "QUICK_START.md"
echo ""

# 检查源代码
echo -e "${BLUE}[4/4] 检查源代码...${NC}"
if [ -d "$PACKAGE_DIR/source/nofx" ]; then
    SOURCE_FILES=$(find "$PACKAGE_DIR/source/nofx" -type f | wc -l)
    SOURCE_SIZE=$(du -sh "$PACKAGE_DIR/source/nofx" | awk '{print $1}')
    echo -e "${GREEN}✓${NC} source/nofx/ (包含 $SOURCE_FILES 个文件，大小 $SOURCE_SIZE)"
    PASSED=$((PASSED + 1))
else
    echo -e "${RED}✗${NC} source/nofx/ (缺失)"
fi
echo ""

# Docker 镜像检查
if [ -d "$PACKAGE_DIR/docker_images" ]; then
    IMAGE_COUNT=$(find "$PACKAGE_DIR/docker_images" -type f | wc -l)
    if [ $IMAGE_COUNT -gt 0 ]; then
        IMAGE_SIZE=$(du -sh "$PACKAGE_DIR/docker_images" | awk '{print $1}')
        echo -e "${GREEN}ℹ${NC} docker_images/ 包含 $IMAGE_COUNT 个镜像文件 (大小 $IMAGE_SIZE)"
    else
        echo -e "${YELLOW}⚠${NC} docker_images/ 目录为空（需要先构建镜像）"
    fi
fi
echo ""

# 显示总结
echo -e "${BLUE}========================================${NC}"
echo -e "${BLUE}验证结果${NC}"
echo -e "${BLUE}========================================${NC}"
echo ""
echo -e "检查项: ${YELLOW}$PASSED / $TOTAL${NC} 已通过"
echo ""

if [ $PASSED -eq $TOTAL ]; then
    echo -e "${GREEN}✓ 离线包验证成功！${NC}"
    echo ""
    echo "下一步："
    echo "1. 查看快速启动: cat QUICK_START.md"
    echo "2. 或直接安装: bash scripts/install.sh"
else
    echo -e "${YELLOW}⚠ 部分文件缺失，请检查${NC}"
    echo ""
    echo "缺失项可能影响安装，请确保所有文件都已复制。"
fi
echo ""
