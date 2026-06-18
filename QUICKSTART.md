# IM系统 快速使用

## 一、环境要求

- **Go** 1.21+
- **MySQL** 5.7+ / 8.0（默认端口 3307，可在 `.env` 修改）

## 二、快速启动（3 步）

### 1. 初始化数据库

启动 MySQL 后，执行建表脚本：

```bash
mysql -u root -p < database/schema.sql
```

这会创建 `im_system` 数据库及全部表，并自动初始化管理员账号。

### 2. 配置 `.env`

编辑项目根目录的 `.env`，按你的 MySQL 实际情况修改：

```env
DB_HOST=127.0.0.1
DB_PORT=3307
DB_USER=root
DB_PASSWORD=root        # 改成你的真实密码
DB_NAME=im_system
PORT=8080               # 服务端口
```

### 3. 启动服务

```bash
go run main.go
```

看到以下日志即启动成功：

```
Database connected successfully
IM系统启动成功，访问 http://localhost:8080
```

## 三、访问地址

| 页面 | 地址 | 说明 |
|------|------|------|
| 首页 | http://localhost:8080/ | 系统入口 |
| 注册 | http://localhost:8080/static/register.html | 新用户注册 |
| 登录 | http://localhost:8080/static/login.html | 用户登录 |
| 聊天 | http://localhost:8080/static/chat.html | IM 主界面（需登录） |
| 安全演示 | http://localhost:8080/static/security.html | SQL注入/XSS/DoS/篡改演示 |
| 审计中心 | http://localhost:8080/static/admin.html | 管理员专用 |

## 四、默认账号

| 角色 | 用户名 | 密码 | 说明 |
|------|--------|------|------|
| 管理员 | `admin` | `admin123` | 系统启动时自动创建，可访问审计中心 |
| 普通用户 | 自行注册 | 自行设置 | 通过注册页面创建 |

## 五、核心功能

### 1. 即时通讯
- 私聊 / 群聊
- SSE 实时推送（弱网自动重连）
- 离线消息持久化，上线自动送达

### 2. 安全攻防演示（security.html）
- **SQL 注入**：对比字符串拼接 vs 参数化查询
- **XSS 跨站脚本**：对比 innerHTML 渲染 vs HTML 转义
- **DoS 拒绝服务**：对比无限制 vs 速率限制（120 次/分钟/IP）
- **数据篡改**：模拟改库不更新哈希，再用哈希链检测

### 3. 防篡改审计（admin.html，管理员专用）
- **篡改告警**：实时展示被检测到的篡改记录
- **审计日志**：哈希链保护的操作日志，防删除/篡改
- **完整性校验**：一键扫描全部哈希链，定位异常记录

## 六、攻防演示流程

以「数据篡改」为例：

1. 用任意账号登录聊天页，发几条消息
2. 打开 `security.html`，找到「数据篡改」卡片
3. 填入某条消息 ID 和篡改内容，点「执行篡改」
4. 点「执行完整性校验」→ 系统精确定位被篡改的记录
5. 用 `admin` / `admin123` 登录 `admin.html` → 查看实时告警和审计日志链

## 七、编译为可执行文件

```bash
# 编译（生成 im.exe）
go build -o im.exe .

# 运行（需保持 .env 和 frontend 目录在同目录下）
.\im.exe
```

> 注意：前端文件不在 exe 内，部署时需连同 `frontend/` 文件夹一起拷贝。

## 八、配置项说明

| 环境变量 | 默认值 | 说明 |
|---------|--------|------|
| `DB_HOST` | `127.0.0.1` | 数据库地址 |
| `DB_PORT` | `3307` | 数据库端口 |
| `DB_USER` | `root` | 数据库用户名 |
| `DB_PASSWORD` | `root` | 数据库密码 |
| `DB_NAME` | `im_system` | 数据库名 |
| `DB_MAX_RETRIES` | `30` | 连接失败重试次数 |
| `DB_RETRY_INTERVAL` | `2s` | 重试间隔 |
| `PORT` | `8080` | 服务监听端口 |

## 九、常见问题

**Q：启动报错「数据库连接失败」？**
A：检查 MySQL 是否启动、`.env` 里的密码/端口是否正确、`im_system` 库是否已建表。

**Q：审计中心显示「需要管理员权限」？**
A：用 `admin` / `admin123` 登录后再访问 `admin.html`。

**Q：修改 `.env` 后不生效？**
A：需重启服务（`go run main.go` 或重新运行 exe）。