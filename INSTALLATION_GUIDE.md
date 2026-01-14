# NowzFX v1.1 - 完整可安装项目

🎯 这是一个完整的、开箱即用的 NowzFX 量化交易系统项目，包含所有源代码、部署脚本和配置文件。

## 📦 项目包含内容

```
nowzfx_v1_1/
├── source/                    # 完整的源代码
│   ├── main.go               # 后端入口
│   ├── web/                  # 前端应用（React + TypeScript）
│   ├── api/                  # API 处理器
│   ├── trader/               # 交易模块
│   ├── backtest/             # 回测模块
│   ├── store/                # 数据存储
│   ├── docker/               # Docker 配置
│   └── ...                   # 其他模块
├── scripts/                  # 部署和构建脚本
│   ├── build_backend.sh      # 构建后端
│   ├── build_frontend.sh     # 构建前端
│   ├── build_docker_images.sh # 构建 Docker 镜像
│   ├── install.sh            # 一键安装
│   └── ...
├── config/                   # 配置文件
│   └── .env.example          # 环境变量示例
├── docker-compose.yml        # Docker Compose 配置
├── 00-START-HERE.txt         # 快速开始指南
├── README.md                 # 项目说明
├── QUICK_START.md            # 快速开始步骤
└── SETUP_GUIDE.md            # 详细设置指南
```

## 🚀 快速开始

### 方法 1：使用 Docker（推荐，最快）

1. **克隆或下载项目**
   ```bash
   git clone https://github.com/NowzMok/nowzfx_v1_1.git
   cd nowzfx_v1_1
   ```

2. **配置环境变量**
   ```bash
   cp config/.env.example config/.env
   # 编辑 config/.env，设置你的 API 密钥和配置
   ```

3. **启动服务**
   ```bash
   docker-compose up -d
   ```

4. **访问应用**
   - 前端：http://localhost:3000
   - 后端 API：http://localhost:8080

### 方法 2：本地开发安装

1. **克隆项目**
   ```bash
   git clone https://github.com/NowzMok/nowzfx_v1_1.git
   cd nowzfx_v1_1
   ```

2. **运行自动化脚本（需要 Go 和 Node.js）**
   ```bash
   chmod +x scripts/build_backend.sh scripts/build_frontend.sh
   scripts/build_backend.sh
   scripts/build_frontend.sh
   ```

3. **配置和启动**
   ```bash
   cp config/.env.example config/.env
   # 根据需要编辑 .env
   docker-compose up -d
   ```

### 方法 3：离线 U 盘安装

1. 复制整个 `nowzfx_v1_1` 文件夹到 U 盘
2. 在目标机器上执行：
   ```bash
   ./start.sh
   # 或
   bash scripts/install.sh
   ```

## 🔧 系统要求

### 最小要求
- **CPU**: 2核心
- **内存**: 4GB RAM
- **存储**: 5GB 可用空间
- **操作系统**: Linux / macOS / Windows (WSL2)

### 推荐配置
- **CPU**: 4核心或以上
- **内存**: 8GB 或以上
- **存储**: 20GB 或以上 SSD
- **网络**: 稳定的互联网连接（用于数据源和交易）

### 软件依赖
- **Docker**: 20.10+
- **Docker Compose**: 2.0+
- 或
- **Go**: 1.25+
- **Node.js**: 20+

## 📋 服务配置

### 后端服务（nofx-backend）
- 端口：8080
- 框架：Go + Gin
- 数据库：SQLite
- 功能：交易执行、回测、监控、API

### 前端服务（nofx-frontend）
- 端口：3000
- 框架：React 18 + TypeScript
- 工具：Vite + Tailwind CSS
- 功能：交易界面、仪表板、配置管理

## 🔐 环保变量配置

关键环境变量：

```env
# API 密钥（必需）
JWT_SECRET=your_jwt_secret_key_here
DATA_ENCRYPTION_KEY=your_encryption_key_here
RSA_PRIVATE_KEY=your_rsa_private_key_here

# 交易交所配置
BINANCE_API_KEY=your_binance_key
BINANCE_API_SECRET=your_binance_secret

# 其他配置
TZ=Asia/Shanghai
LOG_LEVEL=info
REACT_APP_API_URL=http://localhost:8080
```

详细配置说明见 `config/.env.example`

## 📚 文档

- **00-START-HERE.txt** - 快速开始导航
- **QUICK_START.md** - 3 步快速启动
- **SETUP_GUIDE.md** - 详细配置指南
- **INDEX.md** - 文件索引
- **README.md** - 项目详细说明
- **source/README.md** - 源代码说明

## 🛠️ 常用命令

```bash
# 启动所有服务
docker-compose up -d

# 查看日志
docker-compose logs -f

# 重启服务
docker-compose restart

# 停止服务
docker-compose down

# 完全清理（包括数据卷）
docker-compose down -v

# 查看正在运行的容器
docker-compose ps

# 进入容器 shell
docker-compose exec nofx-backend sh
docker-compose exec nofx-frontend sh
```

## 🐛 故障排查

### 容器无法启动
```bash
# 查看详细错误日志
docker-compose logs nofx-backend
docker-compose logs nofx-frontend

# 检查环境变量
cat config/.env

# 重建镜像
docker-compose build --no-cache
```

### 端口被占用
```bash
# 修改 docker-compose.yml 中的端口映射
# 将 "3000:3000" 改为 "3001:3000" 等
```

### 数据库问题
```bash
# 重置数据库
docker-compose exec nofx-backend rm -f /app/data/data.db
docker-compose restart nofx-backend
```

## 📞 支持

- 📖 [完整文档](source/README.md)
- 🐛 [报告问题](https://github.com/NowzMok/nowzfx_v1_1/issues)
- 💬 [讨论](https://github.com/NowzMok/nowzfx_v1_1/discussions)

## 📄 许可证

MIT License - 详见 [LICENSE](LICENSE) 文件

## ⚠️ 免责声明

本项目仅供教育和研究使用。使用本软件进行交易所产生的任何财务损失，作者不承担责任。
始终在进行任何交易活动之前进行彻底的测试和风险评估。

## 🙏 感谢

感谢所有贡献者和用户的支持！

---

**最后更新**: 2025年1月14日
**当前版本**: v1.1.0
**GitHub**: https://github.com/NowzMok/nowzfx_v1_1
