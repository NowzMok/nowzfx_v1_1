# NowzFX - AI驱动的加密货币交易平台

一个基于AI的加密货币自动交易系统，支持多交易所、多策略并行运行。

## 🌟 主要功能

- 🤖 **AI决策引擎**：集成OpenAI/Claude等大语言模型进行交易决策
- 📊 **多策略支持**：支持趋势跟踪、网格交易、动态对冲等多种策略
- 🔄 **多交易所**：支持Binance、OKX等主流交易所
- 📈 **实时监控**：Web界面实时查看交易状态和持仓情况
- 🔐 **安全可靠**：支持API密钥加密存储、风控管理
- 🐳 **容器化部署**：支持Docker一键部署

## 🏗️ 技术栈

### 后端
- **Go 1.21+**: 高性能并发处理
- **Gin**: Web框架
- **SQLite**: 本地数据存储
- **WebSocket**: 实时数据推送

### 前端
- **React 18**: 用户界面
- **TypeScript**: 类型安全
- **Vite**: 快速构建
- **TailwindCSS**: 样式框架

## 📦 快速开始

### 使用Docker Compose部署（推荐）

1. 克隆项目并配置环境变量：
```bash
git clone <repository-url>
cd nowzfx
cp .env.example .env
# 编辑.env文件，填入您的API密钥
```

2. 启动服务：
```bash
docker compose -f docker-compose.complete.yml up -d --build
```

3. 访问服务：
- 前端界面：http://localhost:3000
- 后端API：http://localhost:8080
- 性能监控：http://localhost:6060/debug/pprof/

### 本地开发

#### 后端
```bash
# 安装依赖
go mod download

# 运行
go run main.go
```

#### 前端
```bash
cd web
npm install
npm run dev
```

## ⚙️ 配置说明

在`.env`文件中配置以下参数：

```env
# AI配置
AI_PROVIDER=openai
AI_API_KEY=your-api-key
AI_BASE_URL=https://api.openai.com/v1
AI_MODEL=gpt-4o-mini

# 交易配置
ENABLE_REAL_TRADING=false  # 生产环境设为true
MAX_POSITION_SIZE=1000
RISK_PER_TRADE=0.02

# 日志配置
LOG_LEVEL=info
```

## 📚 文档

- [部署指南](手动部署指南.md)
- [架构设计](ARCHITECTURE_REDESIGN.md)
- [API文档](docs/api.md)

## 🔒 安全提示

- ⚠️ **永远不要**将`.env`文件提交到版本控制
- ⚠️ **永远不要**在公共场所分享API密钥
- ⚠️ 建议在测试网络上充分测试后再使用真实资金
- ⚠️ 启用实盘交易前请确保理解所有风险

## 📝 许可证

本项目采用MIT许可证 - 详见 [LICENSE](LICENSE) 文件

## 🤝 贡献

欢迎提交Issue和Pull Request！

## ⚠️ 免责声明

本软件仅供学习和研究使用。使用本软件进行实盘交易的所有风险由用户自行承担。
