# NOFX 网络安装部署方案

## 📋 部署前检查清单

### ✅ 项目完整性验证

```
✅ 后端代码
  ✓ go.mod - Go 模块定义
  ✓ go.sum - 依赖锁定
  ✓ main.go - 程序入口
  ✓ api/ - API 接口实现
  ✓ auth/ - 认证模块
  ✓ trader/ - 交易逻辑
  ✓ docker/Dockerfile.backend - 后端 Docker 配置

✅ 前端代码
  ✓ web/package.json - NPM 配置
  ✓ web/package-lock.json - 依赖锁定
  ✓ web/src/ - React 源代码
  ✓ docker/Dockerfile.frontend - 前端 Docker 配置

✅ 文档
  ✓ README.md - 项目说明
  ✓ docker-compose.yml - Docker 编排配置
```

---

## 🚀 网络安装方式

### 方式 1：直接从 GitHub 仓库安装（推荐）

```bash
# 1. 克隆项目
git clone https://github.com/YOUR_USERNAME/nofx.git
cd nofx

# 2. 复制环境变量模板
cp .env.example .env

# 3. 修改环境变量（重要）
nano .env
# 修改以下字段：
# - JWT_SECRET (生成: openssl rand -base64 32)
# - DATA_ENCRYPTION_KEY (生成: openssl rand -base64 32)
# - BACKEND_PORT (默认 8080)
# - FRONTEND_PORT (默认 3000)

# 4. 启动服务
docker-compose up -d

# 5. 验证服务
docker-compose ps
docker-compose logs -f
```

**访问应用**：
- 前端: http://localhost:3000
- 后端: http://localhost:8080

---

### 方式 2：从 Docker Hub 拉取预构建镜像

如果您已将镜像上传到 Docker Hub：

```bash
# 编辑 docker-compose.yml，改为：
# image: YOUR_DOCKERHUB_USERNAME/nofx-backend:latest
# image: YOUR_DOCKERHUB_USERNAME/nofx-frontend:latest

docker-compose pull
docker-compose up -d
```

---

### 方式 3：本地构建（无网络或自定义构建）

```bash
git clone https://github.com/YOUR_USERNAME/nofx.git
cd nofx

# 构建后端
docker build -f docker/Dockerfile.backend -t nofx-backend:latest .

# 构建前端
docker build -f docker/Dockerfile.frontend -t nofx-frontend:latest ./web

# 启动
docker-compose up -d
```

---

## 📦 Docker 镜像信息

### 后端镜像 (nofx-backend)
- **基础镜像**: golang:1.25-alpine
- **大小**: ~450-500MB
- **入口点**: `/app/nofx-server`
- **暴露端口**: 8080
- **依赖**: Go 1.25+, TA-Lib 0.4.0

### 前端镜像 (nofx-frontend)  
- **基础镜像**: node:20-alpine → nginx:alpine (多阶段构建)
- **大小**: ~80-100MB
- **入口点**: Nginx 服务
- **暴露端口**: 80 (容器内) → 3000 (宿主机)
- **依赖**: Node.js 20+

---

## ⚙️ 环境变量说明

**复制 `.env.example` 为 `.env`，并修改以下内容**：

```bash
# ===== 安全配置（必须修改！）=====
JWT_SECRET=change-me-to-secure-random-key-at-least-32-chars
DATA_ENCRYPTION_KEY=change-me-to-secure-random-key-for-encryption
RSA_PRIVATE_KEY=  # 可选

# ===== 数据库配置 =====
DB_TYPE=sqlite
DB_PATH=/app/data/data.db

# ===== 应用配置 =====
LOG_LEVEL=info
TZ=Asia/Shanghai

# ===== 端口配置 =====
BACKEND_PORT=8080
FRONTEND_PORT=3000

# ===== 前端 API 地址 =====
REACT_APP_API_URL=http://localhost:8080
```

### 生成安全密钥

```bash
# 生成 JWT_SECRET
openssl rand -base64 32

# 或使用 Python
python3 -c "import secrets; print(secrets.token_urlsafe(32))"
```

---

## 📊 系统要求

### 硬件
- **CPU**: 2+ 核心
- **内存**: 4GB+
- **磁盘**: 10GB+

### 软件
- **Docker**: 20.10+
- **Docker Compose**: 2.0+
- **网络**: 用于首次拉取镜像

---

## 🔧 常用命令

```bash
# 启动服务
docker-compose up -d

# 停止服务
docker-compose stop

# 查看日志
docker-compose logs -f

# 查看特定服务日志
docker-compose logs -f nofx-backend
docker-compose logs -f nofx-frontend

# 重启服务
docker-compose restart

# 完整关闭（删除容器）
docker-compose down

# 删除镜像
docker-compose down --rmi all

# 进入容器
docker-compose exec nofx-backend bash
docker-compose exec nofx-frontend bash
```

---

## 🚨 故障排除

### 无法连接到后端

```bash
# 检查容器状态
docker-compose ps

# 查看后端日志
docker-compose logs nofx-backend

# 测试连接
curl http://localhost:8080/api/health
```

### 前端无法加载

```bash
# 查看前端日志
docker-compose logs nofx-frontend

# 查看 Nginx 配置
docker-compose exec nofx-frontend cat /etc/nginx/conf.d/default.conf
```

### 磁盘空间不足

```bash
# 清理 Docker
docker system prune -a

# 查看镜像大小
docker images

# 删除指定镜像
docker rmi IMAGE_ID
```

---

## 📝 上传到 GitHub 步骤

如果还没有上传，请按以下步骤操作：

### 1. 初始化 Git 仓库

```bash
cd /Users/nowzmok/Desktop/圣灵/nonowz/nofx
git init
git config user.name "Your Name"
git config user.email "your.email@example.com"
```

### 2. 添加文件

```bash
# 添加所有文件
git add .

# 查看将被提交的文件
git status

# 提交
git commit -m "Initial commit: NOFX trading platform"
```

### 3. 添加远程仓库

```bash
# 在 GitHub 创建仓库后，运行：
git remote add origin https://github.com/YOUR_USERNAME/nofx.git
git branch -M main
git push -u origin main
```

### 4. 创建 Release

```bash
# 打标签
git tag -a v1.0.0 -m "Version 1.0.0"
git push origin v1.0.0
```

---

## ✅ 部署验证清单

部署完成后，请检查以下项目：

- [ ] Docker 容器正在运行 (`docker-compose ps` 显示都是 Up)
- [ ] 后端服务响应 (curl http://localhost:8080/api/health)
- [ ] 前端可访问 (浏览器打开 http://localhost:3000)
- [ ] 日志无错误 (docker-compose logs 正常)
- [ ] 数据目录存在 (ls data/)
- [ ] 环境变量已配置 (.env 文件存在且正确)

---

## 📞 支持

遇到问题？请检查：

1. **Docker 日志**: `docker-compose logs -f`
2. **容器状态**: `docker-compose ps`
3. **网络连接**: `docker network inspect nofx_nofx-network`
4. **磁盘空间**: `df -h`
5. **内存占用**: `docker stats`

---

## 版本信息

- **项目**: NOFX Trading Platform
- **版本**: 1.0.0
- **创建日期**: 2026-01-14
- **兼容平台**: Linux, macOS, Windows (WSL2)

---

**祝您部署顺利！** 🚀
