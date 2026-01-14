# NOFX 快速部署指南

## 📋 前置要求

- Docker 20.10+ 
- Docker Compose 2.0+

## 🚀 快速部署（3 步）

### 1️⃣ 下载 docker-compose.yml

```bash
# 选择其中一个方式：

# 方式 A：从 GitHub 克隆整个项目
git clone https://github.com/NowzMok/nowzfx.git
cd nowzfx

# 方式 B：只下载 docker-compose.yml
curl -O https://raw.githubusercontent.com/NowzMok/nowzfx/main/docker-compose.simple.yml
mv docker-compose.simple.yml docker-compose.yml
```

### 2️⃣ 创建 .env 文件（可选，使用默认值则跳过）

```bash
# 复制模板（如果存在）
cp .env.example .env

# 或手动创建并修改
cat > .env << 'EOF'
JWT_SECRET=your-secret-key-change-me
DATA_ENCRYPTION_KEY=your-encryption-key-change-me
RSA_PRIVATE_KEY=
DB_TYPE=sqlite
DB_PATH=/app/data/data.db
LOG_LEVEL=info
TZ=Asia/Shanghai
REACT_APP_API_URL=http://localhost:8080
EOF

# 生成强密钥（推荐）
openssl rand -base64 32  # 用于 JWT_SECRET
openssl rand -base64 32  # 用于 DATA_ENCRYPTION_KEY
```

### 3️⃣ 启动服务

```bash
# 启动所有服务
docker-compose up -d

# 查看服务状态
docker-compose ps

# 查看日志
docker-compose logs -f
```

## ✅ 验证部署

```bash
# 检查后端健康
curl http://localhost:8080/api/health

# 访问前端
# 打开浏览器: http://localhost:3000
```

## 🛑 停止和清理

```bash
# 停止服务（保留数据）
docker-compose stop

# 重启服务
docker-compose restart

# 完全删除容器（保留数据和镜像）
docker-compose down

# 删除一切（包括镜像）
docker-compose down --rmi all
```

## 📝 常用命令

```bash
# 查看实时日志
docker-compose logs -f nofx-backend
docker-compose logs -f nofx-frontend

# 进入容器
docker-compose exec nofx-backend bash
docker-compose exec nofx-frontend bash

# 查看容器状态
docker-compose ps

# 重启服务
docker-compose restart nofx-backend
docker-compose restart nofx-frontend
```

## 🔧 修改配置

```bash
# 1. 停止服务
docker-compose stop

# 2. 编辑 .env 文件
nano .env

# 3. 重启服务
docker-compose up -d
```

## 🚨 故障排除

### 端口已被占用

编辑 `docker-compose.yml`，修改 ports：

```yaml
# 改为其他端口
ports:
  - "8081:8080"   # 后端使用 8081
  - "3001:80"     # 前端使用 3001
```

### 容器无法启动

```bash
# 查看详细日志
docker-compose logs nofx-backend

# 清理并重试
docker-compose down
docker-compose up -d
```

### 无法连接到后端

```bash
# 查看后端容器日志
docker-compose logs nofx-backend

# 测试连接
curl http://localhost:8080/api/health

# 检查网络
docker network inspect nofx_nofx-network
```

## 📊 系统要求

- **CPU**: 2+ 核
- **内存**: 4GB+
- **磁盘**: 10GB+
- **网络**: 首次拉取镜像需要网络连接

## 💾 数据备份

```bash
# 备份数据
cp -r data data.backup-$(date +%Y%m%d)

# 完整备份
tar czf nofx-backup-$(date +%Y%m%d).tar.gz data/ .env docker-compose.yml

# 恢复备份
tar xzf nofx-backup-20240114.tar.gz
docker-compose up -d
```

## 🔐 安全建议

⚠️ **生产环境必须修改以下内容：**

1. **JWT_SECRET** - 生成新的强密钥
   ```bash
   openssl rand -base64 32
   ```

2. **DATA_ENCRYPTION_KEY** - 生成新的加密密钥
   ```bash
   openssl rand -base64 32
   ```

3. **修改默认密码** - 确保 .env 文件不被提交到版本控制

4. **定期备份** - 定期备份 data/ 目录

## 📚 更多信息

- 完整文档: 查看项目中的 NETWORK_DEPLOYMENT_GUIDE.md
- API 文档: http://localhost:8080/docs
- 前端应用: http://localhost:3000

---

**祝您部署顺利！** 🎉
