# im 部署指南

本项目支持通过 Docker / Docker Compose 一键部署，包含 Go 后端 + MySQL 数据库，无需手动建库建表。

## 一、环境要求

- Docker >= 20.10
- Docker Compose >= 2.0（新版 Docker Desktop 已内置 `docker compose`）

## 二、一键部署（推荐）

```bash
# 1. （可选）自定义配置
cp .env.example .env
#   按需修改 .env 中的端口、密码等

# 2. 构建并启动全部服务
docker compose up -d --build
```

启动完成后：

| 服务 | 地址 |
|------|------|
| 前端首页 | http://localhost:8080/ |
| 登录页 | http://localhost:8080/static/login.html |
| 注册页 | http://localhost:8080/static/register.html |
| 聊天界面 | http://localhost:8080/static/chat.html |
| 安全演示 | http://localhost:8080/static/security.html |
| MySQL（宿主机访问） | localhost:3307 |

默认管理员账号（用于审计中心 `admin.html`）：`admin` / `admin123`

## 三、常用命令

```bash
# 查看实时日志
docker compose logs -f im

# 仅查看后端日志
docker compose logs -f im

# 停止所有服务（保留数据）
docker compose down

# 停止并清空数据库数据（重置）
docker compose down -v

# 重新构建镜像（代码修改后）
docker compose up -d --build
```

## 四、配置项说明

所有配置通过 `.env` 文件或环境变量覆盖，默认值见 `.env.example`：

| 变量 | 默认值 | 说明 |
|------|--------|------|
| `APP_PORT` | `8080` | 后端对外端口 |
| `DB_PORT` | `3307` | MySQL 对外端口（容器内固定 3306） |
| `DB_USER` | `root` | 数据库用户名 |
| `DB_PASSWORD` | `root` | 数据库密码 |
| `DB_NAME` | `im_system` | 数据库名 |
| `DB_MAX_RETRIES` | `30` | 连接 MySQL 的最大重试次数 |
| `DB_RETRY_INTERVAL` | `2s` | 重试间隔 |

## 五、架构说明

```
┌────────────────────────────────────────────────────┐
│                  Docker 网络 (im_net)              │
│                                                    │
│  ┌──────────────┐         ┌─────────────────────┐  │
│  │  im_server   │  3306   │     im_mysql        │  │
│  │  Go + Gin    │────────▶│  MySQL 8.0          │  │
│  │  :8080       │         │  数据持久化 volume   │  │
│  └──────┬───────┘         └─────────────────────┘  │
│         │                                            │
└─────────┼────────────────────────────────────────────┘
          │ 8080 → 8080
          ▼
      浏览器访问
```

- **MySQL** 容器首次启动时会自动执行 `database/schema.sql` 完成建库建表，数据持久化到 `im_mysql_data` 卷。
- **im_server** 容器通过 `depends_on.healthcheck` 等待 MySQL 就绪后再启动，并内置连接重试机制，确保启动顺序可靠。
- 后端容器以非 root 用户运行，仅包含编译后的二进制和前端静态文件，镜像体积小、安全性高。

## 六、单独构建镜像

如需在不使用 compose 的情况下单独构建后端镜像：

```bash
docker build -t im:latest .
docker run -d --name im_server -p 8080:8080 \
  -e DB_HOST=host.docker.internal \
  -e DB_PORT=3307 \
  -e DB_PASSWORD=root \
  im:latest
```

## 七、故障排查

1. **后端启动后报 `failed to ping database`**
   - 检查 MySQL 容器是否健康：`docker compose ps`
   - 查看后端日志：`docker compose logs im`
   - 增大 `DB_MAX_RETRIES` / `DB_RETRY_INTERVAL`

2. **端口被占用**
   - 修改 `.env` 中的 `APP_PORT` 或 `DB_PORT` 为其他空闲端口。

3. **数据库未初始化**
   - 通常因为数据卷已存在旧数据。执行 `docker compose down -v` 后重新启动。
