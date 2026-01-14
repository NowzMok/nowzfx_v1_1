# NOFX V1.1 离线安装包 - 文件索引

## 📚 文档文件

### QUICK_START.md
快速开始指南 - 3 分钟上手，包含：
- 一键安装命令
- 常见问题解答
- 基本命令参考
- 推荐首先阅读

### README.md
详细文档 - 完整功能说明，包含：
- 系统要求
- 分步安装指南
- 配置说明
- 故障排除
- 安全建议

### INDEX.md
本文件 - 包内容导航

## 🛠️ 脚本文件

### 安装脚本 (scripts/)

#### install.sh
完整自动化安装脚本
- 一键执行所有构建步骤
- 自动处理依赖下载
- 启动 Docker 服务
- 使用: bash scripts/install.sh

#### prepare_offline_build.sh
构建准备脚本
- 检查编译环境
- 下载 Go/Node 依赖
- 验证项目结构
- 单独使用: bash scripts/prepare_offline_build.sh

#### build_backend.sh
Go 后端构建脚本
- 编译 Go 应用
- 生成二进制文件到 build/backend/
- 可在离线环境重复使用

#### build_frontend.sh
Node.js 前端构建脚本
- 编译 React 应用
- 生成输出到 build/frontend/dist/
- 可在离线环境重复使用

#### build_docker_images.sh
Docker 镜像构建脚本
- 基于编译好的文件构建镜像
- 导出为 .tar.gz 文件
- 存储到 docker_images/
- 依赖 Docker daemon

#### uninstall.sh
卸载脚本
- 停止 Docker 服务
- 删除容器和镜像
- 保留应用数据和配置
- 使用: bash scripts/uninstall.sh

### 工具脚本

#### start.sh
快速启动脚本
- 启动已构建的系统
- 检查镜像和配置
- 一键启动: bash start.sh

#### check_package.sh
包验证脚本
- 验证文件完整性
- 显示详细统计信息
- 诊断缺失的文件

## 📁 目录结构

```
nowzfx_v1_1/
├── source/                           # 源代码目录
│   └── nofx/                        # 完整的 NOFX 项目
│       ├── api/                     # API 接口
│       ├── auth/                    # 认证模块
│       ├── web/                     # 前端 React 应用
│       ├── docker/                  # Docker 配置文件
│       ├── go.mod / go.sum          # Go 依赖
│       ├── docker-compose.yml       # 本地开发配置
│       └── ...
│
├── scripts/                          # 构建和安装脚本
│   ├── install.sh                   # 完整安装脚本
│   ├── prepare_offline_build.sh     # 构建环境准备
│   ├── build_backend.sh             # 后端构建
│   ├── build_frontend.sh            # 前端构建
│   ├── build_docker_images.sh       # 镜像构建
│   └── uninstall.sh                 # 卸载脚本
│
├── config/                           # 配置文件目录
│   └── .env.example                 # 环境变量模板
│
├── docker_images/                    # Docker 镜像存储
│   ├── nofx-backend-1.0.0.tar.gz   # （构建后生成）
│   └── nofx-frontend-1.0.0.tar.gz  # （构建后生成）
│
├── build/                            # 构建输出目录（运行脚本后生成）
│   ├── backend/                     # Go 二进制文件
│   │   └── nofx-server
│   └── frontend/                    # 前端编译产物
│       └── dist/
│
├── data/                             # 应用数据目录（运行时生成）
│   ├── data.db                      # SQLite 数据库
│   └── logs/                        # 日志文件
│
├── docker-compose.yml               # Docker Compose 配置
├── .env                             # 环境变量文件（首次运行时创建）
├── README.md                        # 详细文档
├── QUICK_START.md                   # 快速启动指南
├── INDEX.md                         # 本文件
├── start.sh                         # 快速启动脚本
├── check_package.sh                 # 包验证脚本
└── .gitignore                       # Git 忽略文件

```

## 🚀 快速命令参考

```bash
# 验证包完整性
bash check_package.sh

# 一键安装
bash scripts/install.sh

# 启动已安装的系统
bash start.sh

# 查看日志
docker-compose logs -f

# 停止服务
docker-compose stop

# 卸载
bash scripts/uninstall.sh
```

## 📋 安装流程概览

```
1. 验证系统需求
         ↓
2. 运行 check_package.sh 验证文件
         ↓
3. 运行 scripts/install.sh
         ├─→ 准备构建环境
         ├─→ 编译 Go 后端
         ├─→ 编译 Node 前端
         ├─→ 构建 Docker 镜像
         └─→ 启动服务
         ↓
4. 访问应用 (http://localhost:3000)
```

## ⚙️ 功能特性

### 完全离线部署
- ✅ 包含完整源代码
- ✅ 支持离线编译
- ✅ Docker 镜像可离线加载
- ✅ 不依赖外部服务

### 灵活构建选项
- ✅ 自动化一键安装
- ✅ 分步手动构建
- ✅ 支持部分离线使用
- ✅ Docker 镜像导出

### 完整文档
- ✅ 快速启动指南
- ✅ 详细配置说明
- ✅ 故障排除指南
- ✅ 安全建议

## 🔧 环境变量

编辑 `.env` 文件配置应用：

| 变量 | 说明 | 默认值 |
|------|------|--------|
| JWT_SECRET | JWT 签名密钥 | change-me-... |
| DATA_ENCRYPTION_KEY | 数据加密密钥 | change-me-... |
| DB_TYPE | 数据库类型 | sqlite |
| DB_PATH | 数据库路径 | /app/data/data.db |
| LOG_LEVEL | 日志级别 | info |
| TZ | 时区 | Asia/Shanghai |
| REACT_APP_API_URL | 前端 API 地址 | http://localhost:8080 |

## 📊 系统需求

### 硬件
- CPU: 2+ 核心
- 内存: 4GB+
- 磁盘: 20GB+ (包含源代码和编译产物)

### 软件
- Docker: 20.10+
- Docker Compose: 2.0+
- Go: 1.25+
- Node.js: 20+
- 操作系统: Linux / macOS / Windows (WSL2)

## 📞 获取帮助

### 快速问题
1. 查看 QUICK_START.md
2. 运行 check_package.sh
3. 查看 Docker 日志: docker-compose logs -f

### 详细问题
1. 阅读 README.md
2. 查看脚本注释
3. 检查 Docker 容器状态: docker ps -a

## 📝 版本信息

- **包版本**: 1.1
- **NOFX 版本**: 1.0.0
- **创建时间**: 2026-01-14
- **兼容平台**: Linux, macOS, Windows (WSL2)

---

**开始使用**: 
1. 阅读 QUICK_START.md
2. 运行 bash scripts/install.sh
3. 访问 http://localhost:3000
