# 防篡改审计 IM — 新机快速部署指南

> 目标：在一台全新电脑上，**5 分钟内**把系统跑起来。
> 原则：**零手动建库、零手动建表、零配置即可启动**。

## 一、前置环境（仅需安装一次）

新电脑只需安装 **Docker Desktop**，其余全部由容器自动处理。

| 平台 | 安装方式 | 说明 |
|------|----------|------|
| Windows 10/11 | 下载 [Docker Desktop](https://www.docker.com/products/docker-desktop/) 安装 | 安装后启动 Docker Desktop，等待右下角图标变绿 |
| macOS | 同上（Intel / Apple Silicon 均可） | Apple Silicon 会自动转译 amd64 镜像 |
| Linux | `curl -fsSL https://get.docker.com \| sh` | 自带 compose 插件 |

**验证环境就绪**（在终端执行，两条命令都有输出即可）：

```bash
docker --version          # 需要 >= 20.10
docker compose version    # 需要 >= 2.0
```

> 若 `docker compose` 报命令不存在，请升级 Docker Desktop 或单独安装 compose 插件。

## 二、获取项目代码

```bash
# 方式 A：已有 git
git clone <仓库地址> IM2.0
cd IM2.0

# 方式 B：直接拷贝整个项目文件夹到新电脑，进入目录
cd IM2.0
```

## 三、一键启动（核心步骤）

在项目根目录执行：

```bash
docker compose up -d --build
```

首次启动会自动完成：
1. 拉取 `golang:1.25-alpine` 与 `mysql:8.0` 镜像
2. 编译 Go 后端为静态二进制
3. 启动 MySQL 并自动执行 `database/schema.sql` 建库建表
4. 后端等待 MySQL 健康检查通过后启动
5. 自动创建管理员账号 `admin / admin123`

预计耗时：首次 3–6 分钟（取决于网速），后续启动 < 30 秒。

## 四、快速验证

浏览器打开：

| 页面 | 地址 |
|------|------|
| 首页 | http://localhost:8080/ |
| 登录页 | http://localhost:8080/static/login.html |
| 注册页 | http://localhost:8080/static/register.html |
| 聊天界面 | http://localhost:8080/static/chat.html |
| 安全演示 | http://localhost:8080/static/security.html |
| 审计中心（管理员） | http://localhost:8080/static/admin.html |

**一键自检脚本**（任选其一）：

```bash
# Windows PowerShell
Start-Process http://localhost:8080/

# macOS / Linux
xdg-open http://localhost:8080/  || open http://localhost:8080/
```

用 `admin / admin123` 登录审计中心，能看到篡改告警与审计日志即代表部署成功。

## 五、常用命令速查

```bash
# 查看运行状态
docker compose ps

# 实时查看后端日志（Ctrl+C 退出查看，不影响服务）
docker compose logs -f im

# 查看数据库日志
docker compose logs -f mysql

# 停止全部服务（数据保留）
docker compose down

# 停止并彻底清空数据库（恢复出厂状态）
docker compose down -v

# 修改代码后重新构建并启动
docker compose up -d --build

# 进入后端容器排查问题
docker compose exec im sh
```

## 六、配置项说明（可选）

默认零配置即可启动。如需改端口或密码，复制 `.env.example` 为 `.env` 后修改：

```bash
cp .env.example .env   # Windows: copy .env.example .env
```

| 变量 | 默认值 | 说明 |
|------|--------|------|
| `APP_PORT` | `8080` | 后端对外端口 |
| `DB_PORT` | `3307` | MySQL 对外端口（容器内固定 3306） |
| `DB_USER` | `root` | 数据库用户名 |
| `DB_PASSWORD` | `root` | 数据库密码 |
| `DB_NAME` | `im_system` | 数据库名 |
| `DB_MAX_RETRIES` | `30` | 连接 MySQL 的最大重试次数 |
| `DB_RETRY_INTERVAL` | `2s` | 重试间隔 |

修改 `.env` 后执行 `docker compose up -d` 即可生效。

## 七、架构说明

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

- **MySQL** 容器首次启动自动执行 `database/schema.sql` 建库建表，数据持久化到 `im_mysql_data` 卷。
- **im_server** 通过 `depends_on.healthcheck` 等待 MySQL 就绪后启动，并内置连接重试机制。
- 后端容器以非 root 用户运行，仅含编译二进制和前端静态文件，镜像小、安全性高。

## 八、不用 Docker 的本地启动（可选）

若新电脑已装 Go 1.25+ 和 MySQL 8.0，可跳过 Docker 直接运行：

```bash
# 1. 启动本地 MySQL（端口 3306，root/root）
# 2. 导入建表脚本
mysql -uroot -proot < database/schema.sql
# 3. 启动后端
go run main.go
```

环境变量默认值与本地 MySQL 一致，无需额外配置。如端口/密码不同，请通过环境变量覆盖：

```bash
# Windows PowerShell
$env:DB_PORT="3306"; $env:DB_PASSWORD="你的密码"; go run main.go

# macOS / Linux
DB_PORT=3306 DB_PASSWORD=你的密码 go run main.go
```

## 九、故障排查（新机常见问题）

1. **`docker compose` 命令不存在**
   - 旧版 Docker 需升级到 Docker Desktop，或安装 compose 插件：`apt install docker-compose-plugin`。

2. **后端启动后报 `failed to ping database`**
   - 检查 MySQL 容器健康状态：`docker compose ps`
   - 查看后端日志：`docker compose logs im`
   - 增大 `.env` 中的 `DB_MAX_RETRIES` / `DB_RETRY_INTERVAL`。

3. **端口被占用（`bind: address already in use`）**
   - 修改 `.env` 中的 `APP_PORT` 或 `DB_PORT` 为空闲端口，再 `docker compose up -d`。

4. **数据库未初始化（表不存在）**
   - 数据卷残留旧数据导致 `schema.sql` 未执行。执行 `docker compose down -v` 后重新启动。

5. **镜像拉取失败（网络问题）**
   - 配置国内镜像加速器：Docker Desktop → Settings → Docker Engine 添加
     `"registry-mirrors": ["https://docker.mirrors.ustc.edu.cn"]`
   - 或手动拉取：`docker pull mysql:8.0 && docker pull golang:1.25-alpine`。

6. **Apple Silicon 架构警告**
   - 不影响运行，Docker 会自动通过 Rosetta 转译 amd64 镜像。

7. **浏览器访问 8080 拒绝连接**
   - 确认容器在运行：`docker compose ps` 应显示 `im` 与 `mysql` 均为 `running`。
   - 确认端口映射：`docker compose ps` 中 `im` 行应显示 `0.0.0.0:8080->8080/tcp`。