# NOFX - AI Trading Platform

[![Go](https://img.shields.io/badge/Go-1.21+-00ADD8?style=flat&logo=go)](https://golang.org/)
[![React](https://img.shields.io/badge/React-18+-61DAFB?style=flat&logo=react)](https://reactjs.org/)
[![License](https://img.shields.io/badge/License-AGPL--3.0-blue.svg)](LICENSE)

> Open-source AI-powered cryptocurrency trading system

## ✨ Features

- 🤖 Multi-AI: DeepSeek, Qwen, GPT, Claude, Gemini, Grok, Kimi
- 🏦 Multi-Exchange: Binance, Bybit, OKX, Bitget, Hyperliquid
- 📊 Markets: Crypto, Stocks, Forex, Metals
- ⚡ Strategy Studio: Visual builder with indicators
- 🎯 AI Competition: Multi-model performance comparison
- 💻 Web Dashboard: Real-time P/L and position tracking

## 🚀 Quick Start

### Docker (Recommended)

```bash
git clone https://github.com/NowzMok/nowzfx.git
cd nowzfx
cp .env.example .env
# Edit .env with your API keys
docker compose -f docker-compose.complete.yml up -d --build
```

Access: http://localhost:3000

### Manual

```bash
# Prerequisites: Go 1.21+, Node.js 18+, TA-Lib
go mod download && go build -o nofx && ./nofx
cd web && npm install && npm run dev  # new terminal
```

## 📖 Setup

1. Add AI API keys in web interface
2. Configure exchange credentials
3. Build strategy in Studio
4. Create and start traders

## ⚠️ Risk Warning

Experimental software. AI trading carries risks. Use small amounts for testing.

## 📚 Docs

- [Architecture](docs/architecture/README.md)
- [FAQ](docs/faq/README.md)
- [Contributing](CONTRIBUTING.md)

## 📞 Contact

- Issues: [github.com/NowzMok/nowzfx/issues](https://github.com/NowzMok/nowzfx/issues)
- Telegram: [t.me/nofx_dev_community](https://t.me/nofx_dev_community)

## 📄 License

AGPL-3.0 - See [LICENSE](LICENSE)
