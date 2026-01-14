# NOFX V1.1 快速启动指南

## ⚡ 3 分钟快速启动

### 前置要求
- ✅ Docker 已安装（版本 20.10+）
- ✅ Docker Compose 已安装（版本 2.0+）
- ✅ Go 已安装（版本 1.25+，用于编译后端）
- ✅ Node.js 已安装（版本 20+，用于编译前端）

### 一键安装

```bash
# 1. 进入目录
cd nowzfx_v1_1

# 2. 赋予权限
chmod +x scripts/*.sh

# 3. 运行安装（需要网络用于下载依赖）
sudo bash scripts/install.sh
```

等待安装完成（通常 5-15 分钟，取决于网络和硬件）。

### 验证安装

```bash
# 查看服务状态
docker-compose ps

# 查看实时日志
docker-compose logs -f
```

### 访问应用

打开浏览器访问：
- **前端**: http://localhost:3000
- **后端**: http://localhost:8080

## 📍 目录导航

### 首次安装

1. **查看系统要求**: 阅读本文档的前置要求部分
2. **运行安装**: 执行上面的一键安装命令
3. **访问应用**: 安装完成后打开浏览器

### 配置修改

需要修改环境变量？

```bash
# 1. 停止服务
docker-compose down

# 2. 编辑配置
nano .env
# 或
vim .env

# 3. 重启服务
docker-compose up -d
```

### 查看日志

```bash
# 后端日志
docker-compose logs -f nofx-backend

# 前端日志
docker-compose logs -f nofx-frontend

# 全部日志
docker-compose logs -f
```

### 停止和启动

```bash
# 停止所有服务
docker-compose stop

# 启动所有服务
docker-compose start

# 重启所有服务
docker-compose restart

# 删除容器（保留数据和镜像）
docker-compose down
```

### 完整卸载

```bash
bash scripts/uninstall.sh
```

## 🔍 进阶操作

### 手动分步构建

如果你想了解构建过程的细节，可以分别运行：

```bash
# 1. 准备构建环境
bash scripts/prepare_offline_build.sh

# 2. 构建后端
bash scripts/build_backend.sh

# 3. 构建前端
bash scripts/build_frontend.sh

# 4. 构建 Docker 镜像
bash scripts/build_docker_images.sh

# 5. 启动服务
docker-compose up -d
```

### 更新数据库

```bash
# 进入后端容器
docker exec -it nofx-backend /bin/bash

# 在容器内执行数据库迁移（如果有）
# ./nofx-server migrate
```

### 导出数据

```bash
# 备份数据库
cp data/data.db data/data.db.backup

# 完整备份
tar czf nofx-backup-$(date +%Y%m%d).tar.gz data/ .env
```

## ❓ 常见问题

### Q: 安装需要多久？
**A**: 取决于网络速度和硬件。通常 5-15 分钟。首次安装需要下载依赖，后续安装会快得多。

### Q: 可以离线安装吗？
**A**: 首次安装需要网络下载依赖。之后，可以将整个 `nowzfx_v1_1` 文件夹复制到其他离线设备，直接使用已构建的 Docker 镜像。

### Q: 如何修改端口？
**A**: 编辑 `docker-compose.yml` 中的 `ports` 部分：
```yaml
ports:
  - "YOUR_PORT:3000"  # 前端
  - "YOUR_BACKEND_PORT:8080"  # 后端
```

### Q: 数据存储在哪里？
**A**: 所有数据存储在 `data/` 目录中。停止容器后数据保持不变。

### Q: 忘记修改密钥怎么办？
**A**: 
```bash
# 停止服务
docker-compose down

# 生成新密钥
openssl rand -base64 32

# 编辑 .env
nano .env  # 更新 JWT_SECRET 和 DATA_ENCRYPTION_KEY

# 重启服务
docker-compose up -d
```

## 📚 详细文档

更多详细信息，请查看 `README.md`：

```bash
cat README.md
```

或

```bash
less README.md
```

## 🚨 故障排除

### 无法连接到后端

```bash
# 检查后端容器状态
docker ps | grep nofx-backend

# 查看后端日志
docker logs nofx-backend

# 测试后端连接
curl http://localhost:8080/health
```

### 前端无法加载

```bash
# 检查前端容器
docker ps | grep nofx-frontend

# 查看前端日志
docker logs nofx-frontend

# 检查端口
lsof -i :3000
```

### 磁盘空间不足

```bash
# 清理 Docker 资源
docker system prune

# 清理构建缓存
docker builder prune
```

## 📞 获取帮助

遇到问题？

1. **查看完整日志**: `docker-compose logs -f`
2. **阅读详细文档**: `README.md`
3. **检查 Docker 状态**: `docker-compose ps`
4. **重启服务**: `docker-compose restart`

---

**祝您使用愉快！** 🎉
