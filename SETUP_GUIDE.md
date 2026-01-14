# 📦 NOFX V1.1 完整离线安装包 - 使用指南

## 概述

您现在拥有一个**完全自包含的离线安装包** `nowzfx_v1_1`，包含：

- ✅ **完整源代码** - Go 后端 + Node.js 前端
- ✅ **自动化脚本** - 一键构建和安装
- ✅ **Docker 配置** - 完整的容器化部署
- ✅ **详细文档** - 快速启动 + 完整指南
- ✅ **离线能力** - 可在没有网络的设备上运行

---

## 🎯 文件夹说明

### 已创建的文件夹结构

```
nowzfx_v1_1/                           ← 主文件夹（复制到 U 盘时用这个）
├── source/                            # 源代码目录（目前为空，需要复制）
│   └── nofx/                         # 将在这里复制您的 NOFX 项目
├── scripts/                           # 所有构建脚本
│   ├── install.sh                    # ⭐ 一键安装脚本
│   ├── prepare_offline_build.sh      # 准备构建环境
│   ├── build_backend.sh              # 编译 Go 后端
│   ├── build_frontend.sh             # 编译 Node 前端
│   ├── build_docker_images.sh        # 构建 Docker 镜像
│   └── uninstall.sh                  # 卸载脚本
├── config/
│   └── .env.example                  # 环境变量模板
├── docker_images/                    # Docker 镜像存储（自动生成）
├── build/                            # 编译输出目录（自动生成）
├── data/                             # 应用数据目录（自动生成）
├── docker-compose.yml                # Docker 编排配置
├── 📄 00-START-HERE.txt              # 快速开始文本
├── 📄 README.md                      # 详细完整文档
├── 📄 QUICK_START.md                 # 快速启动指南
├── 📄 INDEX.md                       # 文件导航
├── 🔧 prepare_source.sh              # 源代码准备脚本
├── 🔧 check_package.sh               # 包验证工具
└── 🔧 start.sh                       # 快速启动脚本
```

---

## ⚡ 使用步骤（3 个命令）

### 前置条件

确保已安装：
- Docker (20.10+)
- Docker Compose (2.0+)
- Go (1.25+)
- Node.js (20+)

### 步骤 1: 准备源代码

```bash
cd nowzfx_v1_1
bash prepare_source.sh /Users/nowzmok/Desktop/圣灵/nonowz/nofx
```

**说明**：
- 这会将您的 NOFX 项目复制到 `source/nofx` 目录
- 需要指定您的 NOFX 项目的实际路径

### 步骤 2: 赋予执行权限

```bash
chmod +x scripts/*.sh *.sh
```

### 步骤 3: 一键安装

```bash
sudo bash scripts/install.sh
```

或者不用 sudo（某些命令可能提示权限问题）：

```bash
bash scripts/install.sh
```

---

## 📋 完整构建流程

一键安装脚本会自动执行以下步骤：

```
1. 准备构建环境
   ├─ 检查 Go/Node.js 版本
   ├─ 下载 Go 依赖 (需要网络)
   └─ 安装 Node.js 依赖 (需要网络)

2. 编译 Go 后端
   ├─ 进入源代码目录
   ├─ 编译 Go 程序
   └─ 生成二进制文件到 build/backend/

3. 编译 Node.js 前端
   ├─ 安装前端依赖
   ├─ 构建 React 应用
   └─ 生成输出到 build/frontend/dist/

4. 构建 Docker 镜像
   ├─ 基于编译文件构建镜像
   ├─ 镜像 1: nofx-backend:latest
   ├─ 镜像 2: nofx-frontend:latest
   └─ 导出为 .tar.gz 到 docker_images/

5. 启动服务
   ├─ 创建 Docker Compose 配置
   ├─ 启动后端服务 (端口 8080)
   └─ 启动前端服务 (端口 3000)
```

---

## ✅ 验证安装

### 安装完成后

1. **检查服务状态**
   ```bash
   docker-compose ps
   ```
   应该看到两个容器都是 `Up` 状态

2. **查看日志**
   ```bash
   docker-compose logs -f
   ```
   检查是否有错误信息

3. **访问应用**
   - 前端: http://localhost:3000
   - 后端: http://localhost:8080

---

## 🛠️ 分步操作（手动控制）

如果想逐步控制安装过程，可以分别运行脚本：

### 1. 验证包完整性

```bash
bash check_package.sh
```

### 2. 准备构建环境

```bash
bash scripts/prepare_offline_build.sh
```

### 3. 构建后端

```bash
bash scripts/build_backend.sh
```

输出文件: `build/backend/nofx-server`

### 4. 构建前端

```bash
bash scripts/build_frontend.sh
```

输出文件: `build/frontend/dist/`

### 5. 构建 Docker 镜像

```bash
bash scripts/build_docker_images.sh
```

生成文件:
- `docker_images/nofx-backend-1.0.0.tar.gz`
- `docker_images/nofx-frontend-1.0.0.tar.gz`

### 6. 启动服务

```bash
docker-compose up -d
```

### 7. 验证运行

```bash
docker-compose ps
docker-compose logs -f
```

---

## 📊 配置管理

### 修改环境变量

首次运行时，会自动从 `config/.env.example` 复制 `.env` 文件。

要修改配置：

```bash
# 停止服务
docker-compose down

# 编辑配置文件
nano .env

# 重要配置项：
# JWT_SECRET           - JWT 签名密钥
# DATA_ENCRYPTION_KEY  - 数据加密密钥
# DB_TYPE             - 数据库类型 (sqlite)
# DB_PATH             - 数据库路径

# 重启服务
docker-compose up -d
```

### 生成安全的密钥

```bash
# 生成 32 字符的随机密钥
openssl rand -base64 32

# 或使用 Python
python3 -c "import secrets; print(secrets.token_urlsafe(32))"
```

---

## 🔄 常用命令

### Docker Compose 命令

```bash
# 启动服务
docker-compose up -d

# 停止服务（保留容器）
docker-compose stop

# 启动已停止的服务
docker-compose start

# 重启服务
docker-compose restart

# 删除容器（保留数据和镜像）
docker-compose down

# 查看容器状态
docker-compose ps

# 查看实时日志
docker-compose logs -f

# 查看特定服务日志
docker-compose logs -f nofx-backend
docker-compose logs -f nofx-frontend

# 进入容器交互
docker-compose exec nofx-backend bash
docker-compose exec nofx-frontend bash
```

### 卸载

```bash
# 安全卸载（保留数据）
bash scripts/uninstall.sh

# 或手动卸载
docker-compose down
docker rmi nofx-backend:latest nofx-backend:1.0.0
docker rmi nofx-frontend:latest nofx-frontend:1.0.0
```

---

## 💾 数据管理

### 数据位置

所有应用数据存储在 `data/` 目录：

```
data/
├── data.db          # SQLite 数据库
└── logs/            # 应用日志（如有）
```

### 备份数据

```bash
# 备份整个数据目录
cp -r data data.backup-$(date +%Y%m%d-%H%M%S)

# 只备份数据库
cp data/data.db data/data.db.backup

# 压缩备份
tar czf nofx-backup-$(date +%Y%m%d).tar.gz data/ .env
```

### 恢复数据

```bash
# 停止服务
docker-compose down

# 恢复备份
rm -rf data/*
cp -r data.backup-20240114-120000/* data/

# 或恢复单个数据库
cp data/data.db.backup data/data.db

# 重启服务
docker-compose up -d
```

---

## 🚚 复制到 U 盘（离线部署）

### 完整步骤

1. **验证包完整性**
   ```bash
   bash check_package.sh
   ```

2. **复制到 U 盘**
   ```bash
   cp -r nowzfx_v1_1 /Volumes/YOUR_USB_NAME/
   ```

3. **在离线设备上安装**
   ```bash
   cd /Volumes/YOUR_USB_NAME/nowzfx_v1_1
   bash scripts/install.sh
   ```

### 所需空间

- **最小**: 15GB (仅源代码)
- **典型**: 25GB (含编译产物)
- **完整**: 30GB+ (含所有中间文件)

---

## ❓ 常见问题

### Q1: 需要网络连接吗？

**A**: 首次安装时需要网络用于：
- 下载 Go modules (go mod download)
- 下载 npm packages (npm install)
- Docker 镜像操作

之后可以完全离线运行。

### Q2: 如何在离线环境中使用？

**A**: 
1. 在有网络的机器上完整运行一次 `install.sh`
2. 将 docker 镜像导出: `docker save nofx-backend:latest | gzip > ...tar.gz`
3. 将整个 `nowzfx_v1_1` 文件夹复制到 U 盘
4. 在离线设备上，镜像会从 `docker_images/` 自动加载

### Q3: 端口已被占用怎么办？

**A**: 编辑 `docker-compose.yml`，修改 ports 部分：
```yaml
ports:
  - "3001:80"     # 改为 3001
  - "8081:8080"   # 改为 8081
```

### Q4: 如何重新构建？

**A**: 
```bash
# 清理旧的镜像
docker-compose down
docker rmi nofx-backend:latest nofx-frontend:latest

# 删除构建产物
rm -rf build/

# 重新构建
bash scripts/install.sh
```

### Q5: 数据会丢失吗？

**A**: 不会。使用 `docker-compose down` 只删除容器，不删除：
- `data/` 目录中的数据
- `.env` 配置文件
- Docker 镜像

只有使用 `uninstall.sh` 或手动 `docker rmi` 时才会删除镜像。

---

## 📚 文档导航

| 文件 | 内容 | 适合场景 |
|------|------|---------|
| 00-START-HERE.txt | 极简快速开始 | 第一次打开文件夹 |
| QUICK_START.md | 3 分钟快速指南 | 急着想运行 |
| README.md | 完整详细文档 | 深入了解所有功能 |
| INDEX.md | 文件索引和导航 | 寻找特定功能 |

---

## 🔐 安全建议

1. **修改默认密钥** ⚠️ 重要
   ```bash
   # 生成新密钥
   openssl rand -base64 32
   
   # 更新 .env 文件
   JWT_SECRET=<生成的密钥>
   DATA_ENCRYPTION_KEY=<生成的密钥>
   ```

2. **备份重要数据**
   ```bash
   cp -r data data.backup
   cp .env .env.backup
   ```

3. **网络隔离** (如在离线环境)
   - 物理隔离网络
   - 或使用防火墙限制访问

4. **定期更新**
   - 更新 Docker 基础镜像
   - 更新依赖包

---

## 📞 故障排除

### Docker 镜像构建失败

```bash
# 查看详细错误
docker build -f docker/Dockerfile.backend . --progress=plain

# 清理 Docker 缓存
docker builder prune
```

### 依赖下载失败

```bash
# 清理 npm 缓存
npm cache clean --force

# 清理 Go 模块缓存
rm -rf ~/go/pkg/mod

# 重新下载
bash scripts/prepare_offline_build.sh
```

### 端口冲突

```bash
# 查找占用的进程
lsof -i :3000
lsof -i :8080

# 修改 docker-compose.yml 中的端口映射
```

### 权限错误

```bash
# 赋予脚本执行权限
chmod +x scripts/*.sh *.sh

# 或使用 sudo
sudo bash scripts/install.sh
```

---

## 📝 版本信息

- **包版本**: 1.1
- **NOFX 版本**: 1.0.0
- **创建时间**: 2026-01-14
- **支持平台**: Linux, macOS, Windows (WSL2)

---

## 🎉 现在开始

1. 阅读 `00-START-HERE.txt` 或 `QUICK_START.md`
2. 运行 `bash prepare_source.sh /path/to/nofx`
3. 运行 `bash scripts/install.sh`
4. 访问 http://localhost:3000

**祝您使用愉快！**
