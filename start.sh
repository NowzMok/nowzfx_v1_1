#!/bin/bash

##############################################################################
# NOFX 启动脚本 (一键启动)
# 功能：快速启动已构建的 NOFX 系统
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
echo -e "${BLUE}NOFX 启动${NC}"
echo -e "${BLUE}========================================${NC}"
echo ""

# 检查 Docker
if ! command -v docker &> /dev/null; then
    echo -e "${RED}✗ 错误：Docker 未安装${NC}"
    exit 1
fi

# 检查 docker-compose
if ! command -v docker-compose &> /dev/null; then
    echo -e "${RED}✗ 错误：docker-compose 未安装${NC}"
    exit 1
fi

cd "$PACKAGE_DIR"

# 检查 docker-compose.yml
if [ ! -f "docker-compose.yml" ]; then
    echo -e "${RED}✗ 错误：docker-compose.yml 不存在${NC}"
    exit 1
fi

# 检查镜像
echo -e "${BLUE}[1/3] 检查 Docker 镜像...${NC}"

BACKEND_EXISTS=$(docker images | grep -c "nofx-backend" || true)
FRONTEND_EXISTS=$(docker images | grep -c "nofx-frontend" || true)

if [ $BACKEND_EXISTS -eq 0 ] || [ $FRONTEND_EXISTS -eq 0 ]; then
    echo -e "${YELLOW}⚠️  未找到镜像，需要先构建...${NC}"
    echo ""
    
    # 尝试从 docker_images 目录加载
    if [ -d "docker_images" ]; then
        BACKEND_TAR=$(find docker_images -name "nofx-backend*.tar.gz" -o -name "nofx-backend*.tar" | head -1)
        FRONTEND_TAR=$(find docker_images -name "nofx-frontend*.tar.gz" -o -name "nofx-frontend*.tar" | head -1)
        
        if [ -n "$BACKEND_TAR" ]; then
            echo -e "${YELLOW}   从文件加载后端镜像...${NC}"
            docker load -i "$BACKEND_TAR"
        fi
        
        if [ -n "$FRONTEND_TAR" ]; then
            echo -e "${YELLOW}   从文件加载前端镜像...${NC}"
            docker load -i "$FRONTEND_TAR"
        fi
    fi
fi

if docker images | grep -q "nofx-backend"; then
    echo -e "${GREEN}✓ 后端镜像已就绪${NC}"
else
    echo -e "${RED}✗ 后端镜像缺失，请先运行 scripts/build_docker_images.sh${NC}"
    exit 1
fi

if docker images | grep -q "nofx-frontend"; then
    echo -e "${GREEN}✓ 前端镜像已就绪${NC}"
else
    echo -e "${RED}✗ 前端镜像缺失，请先运行 scripts/build_docker_images.sh${NC}"
    exit 1
fi

echo ""

# 检查配置文件
echo -e "${BLUE}[2/3] 检查配置文件...${NC}"

if [ ! -f ".env" ]; then
    echo -e "${YELLOW}   复制环境变量模板...${NC}"
    if [ -f "config/.env.example" ]; then
        cp config/.env.example .env
        echo -e "${GREEN}✓ 配置文件已创建${NC}"
        echo -e "${YELLOW}⚠️  请检查并修改 .env 文件中的敏感信息${NC}"
    fi
else
    echo -e "${GREEN}✓ 配置文件已存在${NC}"
fi

echo ""

# 启动服务
echo -e "${BLUE}[3/3] 启动服务...${NC}"

# 创建数据目录
mkdir -p data

# 启动 docker-compose
docker-compose up -d

if [ $? -eq 0 ]; then
    echo -e "${GREEN}✓ 服务启动成功${NC}"
    echo ""
    
    # 等待服务完全启动
    echo -e "${YELLOW}   等待服务启动中...${NC}"
    sleep 3
    
    # 显示状态
    echo ""
    docker-compose ps
    echo ""
    
    echo -e "${GREEN}========================================${NC}"
    echo -e "${GREEN}NOFX 已启动！${NC}"
    echo -e "${GREEN}========================================${NC}"
    echo ""
    echo -e "访问地址:"
    echo -e "  前端: ${YELLOW}http://localhost:3000${NC}"
    echo -e "  后端: ${YELLOW}http://localhost:8080${NC}"
    echo ""
    echo -e "有用命令:"
    echo -e "  查看日志:   ${YELLOW}docker-compose logs -f${NC}"
    echo -e "  停止服务:   ${YELLOW}docker-compose stop${NC}"
    echo -e "  重启服务:   ${YELLOW}docker-compose restart${NC}"
    echo ""
else
    echo -e "${RED}✗ 服务启动失败${NC}"
    echo "运行以下命令查看错误信息:"
    echo -e "  ${YELLOW}docker-compose logs${NC}"
    exit 1
fi
