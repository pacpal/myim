# 弱网优化IM即时通讯系统 — 项目报告

## 目录
1. [绪论](#一绪论)
2. [需求分析](#二需求分析)
3. [系统设计](#三系统设计)
4. [数据库设计](#四数据库设计)
5. [实现过程](#五实现过程)
6. [安全攻防演示](#六安全攻防演示)
7. [测试与运行](#七测试与运行)

---

## 一、绪论

### 1.1 项目背景

随着移动互联网的普及，即时通讯（Instant Messaging, IM）已成为人们日常沟通的重要方式。然而，在地铁、电梯、偏远地区、跨国网络等弱网环境下，传统IM系统常出现消息丢失、延迟过高、连接频繁断开等问题，严重影响用户体验。

弱网环境的主要特征包括：
- **高延迟**：网络往返时间（RTT）可能超过1000ms
- **高丢包率**：数据包丢失率可达5%~20%
- **带宽受限**：可用带宽可能低于100KB/s
- **连接不稳定**：TCP连接频繁断开与重连

### 1.2 待解决问题

本项目旨在解决**弱网环境下的即时通讯可靠性问题**，具体包括：
1. 消息在弱网下的可靠投递（不丢失、不重复）
2. 断线后的自动重连与离线消息补发
3. 服务器资源的合理保护（防止恶意攻击耗尽资源）
4. 用户数据的安全性（防止SQL注入、XSS等攻击）

### 1.3 项目意义

1. **实用价值**：为弱网环境用户提供稳定可靠的通讯服务
2. **安全价值**：通过攻防演示帮助理解Web安全威胁与防御方法
3. **教学价值**：完整展示前后端分离架构、数据库设计、实时通信、安全防护的综合实践

### 1.4 技术选型理由

| 技术 | 选择 | 理由 |
|------|------|------|
| 后端语言 | Go | 高并发、编译型、性能优异 |
| Web框架 | Gin | 轻量高效、API友好 |
| 数据库 | MySQL | 关系型、事务支持、生态成熟 |
| 实时通信 | SSE | 单向持久连接、自动重连、适合弱网 |
| 前端 | HTML/CSS/JS | 无需构建工具、轻量、兼容性好 |

---

## 二、需求分析

### 2.1 功能需求

#### 2.1.1 用户管理
- 用户注册（用户名、密码、昵称）
- 用户登录/登出
- 个人资料查看与修改

#### 2.1.2 好友管理
- 搜索用户
- 发送/接收好友请求
- 接受/拒绝好友请求
- 删除好友
- 查看好友列表（含在线状态）

#### 2.1.3 群组管理
- 创建群组
- 查看群组列表
- 查看群成员
- 添加/移除群成员
- 解散群组

#### 2.1.4 消息通讯
- 一对一私聊
- 群组聊天
- 消息历史记录查询
- 消息删除
- 消息状态追踪（已发送/已送达/已读）
- 离线消息存储与自动推送

#### 2.1.5 安全攻防演示
- SQL注入攻击与防御
- XSS跨站脚本攻击与防御
- DoS拒绝服务攻击与防御

### 2.2 非功能需求

| 需求 | 描述 |
|------|------|
| 弱网优化 | SSE自动重连、离线消息存储、心跳保活、消息内容截断 |
| 安全性 | 参数化查询、HTML转义、速率限制、安全响应头 |
| 可扩展性 | 模块化设计、Hub模式管理连接 |
| 响应性 | 前后端分离、异步通信 |

### 2.3 工具与方法基本原理

#### 2.3.1 SSE（Server-Sent Events）
SSE是HTML5标准的一部分，允许服务器通过HTTP连接向浏览器推送数据。

**原理**：
- 客户端通过`EventSource` API建立持久HTTP连接
- 服务器以`text/event-stream`格式持续推送数据
- 浏览器自动处理重连（内置retry机制）
- 适合服务器到客户端的单向数据流

**弱网优势**：
- 基于HTTP，穿透防火墙能力强
- 浏览器自动重连，无需额外代码
- 轻量级协议头，带宽占用小
- 支持事件类型分类（event字段）

#### 2.3.2 参数化查询（Prepared Statement）
**原理**：SQL语句与数据分离，数据库引擎将参数仅作为数据处理，不会解析为SQL语法。

```
漏洞：SELECT * FROM users WHERE username = '" + input + "'"
安全：SELECT * FROM users WHERE username = ?
```

#### 2.3.3 HTML转义（XSS防御）
**原理**：将HTML特殊字符转换为实体编码，使浏览器将其作为文本显示而非代码执行。

```
<  →  &lt;
>  →  &gt;
"  →  &quot;
'  →  &#39;
&  →  &amp;
```

#### 2.3.4 速率限制（Rate Limiting，DoS防御）
**原理**：基于滑动窗口算法，限制每个IP在时间窗口内的请求次数，超过阈值返回429状态码。

---

## 三、系统设计

### 3.1 整体架构

```
┌─────────────────────────────────────────────────────────┐
│                      浏览器前端                          │
│  ┌──────────┐  ┌──────────┐  ┌──────────────────────┐  │
│  │ 登录注册  │  │ 聊天界面  │  │  安全攻防演示界面     │  │
│  └────┬─────┘  └────┬─────┘  └──────────┬───────────┘  │
│       │              │                   │              │
│       │   HTTP API   │   SSE长连接        │  HTTP API    │
└───────┼──────────────┼───────────────────┼──────────────┘
        │              │                   │
┌───────▼──────────────▼───────────────────▼──────────────┐
│                    Go后端 (Gin)                           │
│  ┌─────────┐ ┌──────────┐ ┌──────────┐ ┌──────────────┐  │
│  │Auth中间件│ │限流中间件│ │CORS中间件│ │安全头中间件  │  │
│  └────┬────┘ └────┬─────┘ └────┬─────┘ └──────┬───────┘  │
│       │           │            │               │          │
│  ┌────▼───────────▼────────────▼───────────────▼───────┐ │
│  │              Handler路由层                           │ │
│  │  auth.go  friend.go  group.go  message.go  attack.go │ │
│  └────────────────────┬─────────────────────────────────┘ │
│                       │                                   │
│  ┌────────────────────▼─────────────────────────────────┐│
│  │              SSE Hub (连接管理)                        ││
│  └────────────────────┬─────────────────────────────────┘│
│  ┌────────────────────▼─────────────────────────────────┐│
│  │              Database层 (MySQL)                       ││
│  └──────────────────────────────────────────────────────┘│
└──────────────────────────────────────────────────────────┘
```

### 3.2 模块划分

| 模块 | 文件 | 职责 |
|------|------|------|
| 入口 | main.go | 路由注册、中间件配置、启动服务 |
| 数据库 | database/db.go, schema.sql | 连接管理、表结构定义 |
| 模型 | models/models.go | 数据结构定义 |
| 中间件 | middleware/middleware.go | 认证、限流、CORS、安全头 |
| Hub | hub/hub.go | SSE连接管理、消息推送 |
| 认证 | handlers/auth.go | 注册、登录、登出、资料 |
| 好友 | handlers/friend.go | 好友请求、好友列表、搜索 |
| 群组 | handlers/group.go | 群组CRUD、成员管理 |
| 消息 | handlers/message.go | 私聊、群聊、SSE、历史记录 |
| 攻防 | handlers/attack.go | SQL注入/XSS/DoS演示 |
| 工具 | utils/utils.go | HTML转义、输入验证、Token生成 |
| 前端 | frontend/ | HTML页面、CSS样式、JS逻辑 |

### 3.3 API设计

| 方法 | 路径 | 功能 | 认证 |
|------|------|------|------|
| POST | /api/register | 注册 | 否 |
| POST | /api/login | 登录 | 否 |
| POST | /api/logout | 登出 | 是 |
| GET | /api/profile | 获取资料 | 是 |
| PUT | /api/profile | 修改资料 | 是 |
| GET | /api/friends | 好友列表 | 是 |
| POST | /api/friends/request | 发送好友请求 | 是 |
| GET | /api/friends/requests | 好友请求列表 | 是 |
| PUT | /api/friends/requests/:id/accept | 接受请求 | 是 |
| PUT | /api/friends/requests/:id/reject | 拒绝请求 | 是 |
| DELETE | /api/friends/:id | 删除好友 | 是 |
| GET | /api/users/search | 搜索用户 | 是 |
| GET | /api/groups | 群组列表 | 是 |
| POST | /api/groups | 创建群组 | 是 |
| GET | /api/groups/:id/members | 群成员 | 是 |
| POST | /api/groups/:id/members | 添加成员 | 是 |
| DELETE | /api/groups/:id/members | 移除成员 | 是 |
| DELETE | /api/groups/:id | 解散群组 | 是 |
| POST | /api/messages | 发送私聊 | 是 |
| GET | /api/messages/:id | 聊天记录 | 是 |
| DELETE | /api/messages/:id | 删除消息 | 是 |
| POST | /api/groups/:id/messages | 群聊消息 | 是 |
| GET | /api/groups/:id/messages | 群聊记录 | 是 |
| GET | /api/sse | SSE实时推送 | 是 |
| GET | /api/attack/sql/vulnerable | SQL注入漏洞 | 否 |
| GET | /api/attack/sql/safe | SQL注入防御 | 否 |
| POST | /api/attack/xss/vulnerable | XSS漏洞 | 否 |
| POST | /api/attack/xss/safe | XSS防御 | 否 |
| GET | /api/attack/dos/vulnerable | DoS漏洞 | 否 |

---

## 四、数据库设计

### 4.1 ER图（实体关系图）

```
┌──────────────┐         ┌──────────────┐
│    users     │         │ login_logs   │
│──────────────│         │──────────────│
│ PK id        │◄──┐     │ PK id        │
│    username  │   │     │ FK user_id ──┘
│    password  │   │     │    username  │
│    nickname  │   │     │    ip_address│
│    avatar    │   │     │    user_agent│
│    status    │   │     │    status    │
│    last_login│   │     │    created_at│
│    created_at│   │     └──────────────┘
└──────┬───────┘   │
       │           │
       │ 1         │
       │           │
       │ N         │
┌──────▼───────┐   │     ┌──────────────┐
│   friends    │   │     │ friend_req   │
│──────────────│   │     │──────────────│
│ PK id        │   │     │ PK id        │
│ FK user_id ──┘   │     │ FK from_user──┘
│ FK friend_id ────┘     │ FK to_user ───►users
│    remark    │         │    message   │
│    created_at│         │    status    │
└──────────────┘         └──────────────┘

┌──────────────┐         ┌──────────────┐
│   groups     │         │ group_members│
│──────────────│         │──────────────│
│ PK id        │◄────────│ PK id        │
│    name      │         │ FK group_id  │
│ FK owner_id ──►users   │ FK user_id ──►users
│    avatar    │         │    role      │
│    description│        │    joined_at  │
│    created_at│         └──────────────┘
└──────┬───────┘
       │
       │ 1
       │
       │ N
┌──────▼───────┐         ┌──────────────┐
│group_messages│         │   messages   │
│──────────────│         │──────────────│
│ PK id        │         │ PK id        │
│ FK group_id ─┘         │ FK from_user──►users
│ FK from_user──►users   │ FK to_user ───►users
│    content   │         │    content   │
│    msg_type  │         │    msg_type  │
│    created_at│         │    status    │
└──────────────┘         │    created_at│
                         └──────┬───────┘
                                │
                                │ 1
                                │
                         ┌──────▼───────┐
                         │ message_ack  │
                         │──────────────│
                         │ PK id        │
                         │ FK message_id│
                         │ FK user_id ──►users
                         │    ack_type  │
                         │    ack_time  │
                         └──────────────┘

                         ┌──────────────┐
                         │offline_msgs  │
                         │──────────────│
                         │ PK id        │
                         │ FK user_id ──►users
                         │ FK message_id│
                         │    delivered │
                         │    created_at│
                         └──────────────┘
```

### 4.2 表结构说明（10张表）

| 序号 | 表名 | 说明 | 关联 |
|------|------|------|------|
| 1 | users | 用户账户 | 被多表引用 |
| 2 | friends | 好友关系 | user_id, friend_id → users |
| 3 | friend_requests | 好友请求 | from_user_id, to_user_id → users |
| 4 | groups | 群组 | owner_id → users |
| 5 | group_members | 群成员 | group_id → groups, user_id → users |
| 6 | messages | 私聊消息 | from_user_id, to_user_id → users |
| 7 | group_messages | 群聊消息 | group_id → groups, from_user_id → users |
| 8 | message_ack | 消息回执 | message_id → messages, user_id → users |
| 9 | offline_messages | 离线消息 | user_id → users, message_id → messages |
| 10 | login_logs | 登录日志 | user_id → users |

### 4.3 数据库操作类型（大于4种）

| 操作类型 | SQL语句 | 应用场景 |
|----------|---------|----------|
| INSERT | INSERT INTO users... | 注册用户、发送消息、创建群组 |
| DELETE | DELETE FROM friends... | 删除好友、解散群组、删除消息 |
| UPDATE | UPDATE users SET status... | 更新在线状态、消息已读、请求状态 |
| SELECT | SELECT * FROM messages... | 查询消息、好友列表、群成员 |
| JOIN | JOIN users ON... | 多表关联查询（好友+用户信息、群成员+用户信息） |
| 事务 | BEGIN/COMMIT | 接受好友请求（双向添加）、创建群组（建群+加群主） |

---

## 五、实现过程

### 5.1 编程语言与环境配置

#### 5.1.1 开发环境
- **操作系统**：Windows
- **Go版本**：Go 1.25.0
- **数据库**：MySQL 8.0（Docker容器，端口3307）
- **浏览器**：Chrome/Edge

#### 5.1.2 依赖包
```
github.com/gin-gonic/gin          # Web框架
github.com/go-sql-driver/mysql    # MySQL驱动
golang.org/x/crypto/bcrypt        # 密码加密
```

#### 5.1.3 项目结构
```
IM2.0/
├── main.go                    # 程序入口
├── go.mod / go.sum            # 依赖管理
├── database/
│   ├── db.go                 # 数据库连接
│   └── schema.sql            # 建表脚本
├── models/
│   └── models.go              # 数据模型
├── middleware/
│   └── middleware.go          # 认证/限流/CORS/安全头
├── hub/
│   └── hub.go                # SSE连接管理
├── handlers/
│   ├── auth.go               # 认证处理
│   ├── friend.go             # 好友处理
│   ├── group.go              # 群组处理
│   ├── message.go            # 消息处理
│   └── attack.go             # 攻防演示
├── utils/
│   └── utils.go               # 工具函数
└── frontend/
    ├── index.html             # 首页
    ├── login.html             # 登录
    ├── register.html          # 注册
    ├── chat.html              # 聊天界面
    ├── security.html          # 安全演示
    ├── css/style.css          # 样式
    └── js/
        ├── auth.js            # API工具
        ├── chat.js            # 聊天逻辑
        └── security.js        # 攻防逻辑
```

### 5.2 核心代码说明

#### 5.2.1 数据库连接（database/db.go）
使用连接池管理数据库连接，设置最大连接数和空闲连接数。

#### 5.2.2 SSE实时推送（hub/hub.go + handlers/message.go）
- Hub维护用户ID到客户端通道的映射
- 客户端连接时注册，断开时注销
- 发送消息时通过Hub推送到目标用户的SSE连接
- 心跳机制保持连接活跃

#### 5.2.3 弱网优化策略
1. **SSE自动重连**：浏览器EventSource内置重连机制，断线后3秒自动重连
2. **离线消息存储**：接收方不在线时，消息存入offline_messages表，上线后自动推送
3. **心跳保活**：每30秒发送心跳注释行，防止连接超时
4. **消息内容截断**：限制消息最大长度5000字符，减少带宽占用

#### 5.2.4 安全防御实现
1. **SQL注入防御**：所有数据库查询使用参数化查询（`?`占位符）
2. **XSS防御**：使用`html.EscapeString`转义输出，前端使用`textContent`而非`innerHTML`
3. **DoS防御**：基于IP的滑动窗口速率限制（120次/分钟），登录接口更严格（10次/分钟）

### 5.3 功能截图说明

系统运行后可通过以下地址访问：
- 首页：http://localhost:8080/
- 登录：http://localhost:8080/static/login.html
- 注册：http://localhost:8080/static/register.html
- 聊天：http://localhost:8080/static/chat.html
- 安全演示：http://localhost:8080/static/security.html

#### 主要界面功能：
1. **首页**：展示系统特性，提供登录/注册/安全演示入口
2. **注册页**：输入用户名、昵称、密码注册账户
3. **登录页**：输入用户名密码登录，获取会话Token
4. **聊天界面**：
   - 左侧栏：好友列表/群组列表/好友请求（标签切换）
   - 右侧：聊天区域，支持发送/接收消息
   - 添加好友：搜索用户名并发送好友请求
   - 创建群组：输入群名和描述创建群
5. **安全演示页**：
   - SQL注入：对比漏洞接口与参数化查询防御
   - XSS：对比原始输出与HTML转义防御
   - DoS：对比无限制接口与速率限制防御

---

## 六、安全攻防演示

### 6.1 SQL注入攻击与防御

#### 攻击原理
漏洞接口使用字符串拼接构造SQL：
```go
query := "SELECT id, username, nickname FROM users WHERE username = '" + username + "'"
```
攻击者输入 `' OR '1'='1`，SQL变为：
```sql
SELECT id, username, nickname FROM users WHERE username = '' OR '1'='1'
```
WHERE条件恒真，返回所有用户数据。

#### 防御方法（3种）
1. **参数化查询**（主要）：
   ```go
   db.Query("SELECT ... WHERE username = ?", username)
   ```
2. **输入验证**：用户名只允许字母数字下划线
3. **最小权限**：数据库用户仅授予必要权限

#### 测试结果
- 漏洞接口：输入`' OR '1'='1` → 返回所有用户数据（攻击成功）
- 防御接口：输入`' OR '1'='1` → 返回空结果（防御成功）

### 6.2 XSS跨站脚本攻击与防御

#### 攻击原理
漏洞接口直接返回原始内容，前端使用`innerHTML`渲染：
```javascript
element.innerHTML = data.raw_content;  // <script>被执行
```
攻击者发送`<script>alert('XSS')</script>`，脚本在受害者浏览器执行。

#### 防御方法（3种）
1. **HTML转义**（主要）：
   ```go
   html.EscapeString(content)  // < → &lt; > → &gt;
   ```
2. **CSP策略**：设置Content-Security-Policy响应头
3. **HttpOnly Cookie**：防止JavaScript读取Cookie

#### 测试结果
- 漏洞接口：返回`<script>alert('XSS')</script>`，前端innerHTML渲染后脚本执行
- 防御接口：返回`&lt;script&gt;...`，前端渲染为文本，脚本不执行

### 6.3 DoS拒绝服务攻击与防御

#### 攻击原理
漏洞接口无任何限制，每次请求执行1000万次循环计算，并发请求耗尽CPU资源。

#### 防御方法（3种）
1. **速率限制**（主要）：基于IP的滑动窗口算法，120次/分钟
2. **连接数限制**：数据库连接池上限25个
3. **登录频率限制**：登录接口单独限制10次/分钟

#### 测试结果
- 发送150个并发请求到受保护接口
- 结果：113个成功(200)，37个被限流(429)
- 速率限制有效拦截了超量请求

---

## 七、测试与运行

### 7.1 启动步骤

1. **启动MySQL**（Docker方式）：
   ```bash
   docker run -d --name im_mysql -p 3307:3306 \
     -e MYSQL_ROOT_PASSWORD=root \
     -e MYSQL_DATABASE=im_system \
     mysql:latest
   ```

2. **初始化数据库**：
   ```bash
   docker cp database/schema.sql im_mysql:/schema.sql
   docker exec im_mysql sh -c "mysql -uroot -proot < /schema.sql"
   ```

3. **启动后端**：
   ```bash
   go run main.go
   ```

4. **访问系统**：
   - 打开浏览器访问 http://localhost:8080/

### 7.2 测试用例

| 测试项 | 操作 | 预期结果 | 实际结果 |
|--------|------|----------|----------|
| 用户注册 | 输入用户名密码注册 | 注册成功 | ✅通过 |
| 用户登录 | 输入正确密码登录 | 返回Token | ✅通过 |
| 搜索用户 | 输入关键词搜索 | 返回匹配用户 | ✅通过 |
| 好友请求 | 发送好友请求 | 请求发送成功 | ✅通过 |
| 接受请求 | 接受好友请求 | 双向添加好友 | ✅通过 |
| 发送私聊 | 发送消息给好友 | 消息存储并推送 | ✅通过 |
| 查看记录 | 查看聊天历史 | 返回消息列表 | ✅通过 |
| 创建群组 | 创建新群组 | 群组创建成功 | ✅通过 |
| 群聊消息 | 发送群聊消息 | 所有成员收到 | ✅通过 |
| SQL注入 | 漏洞接口注入 | 返回所有用户 | ✅攻击成功 |
| SQL防御 | 参数化查询 | 返回空结果 | ✅防御成功 |
| XSS攻击 | 发送脚本标签 | 脚本执行 | ✅攻击成功 |
| XSS防御 | HTML转义 | 显示为文本 | ✅防御成功 |
| DoS攻击 | 并发请求 | 服务器处理 | ✅攻击成功 |
| DoS防御 | 速率限制 | 部分请求被拒 | ✅防御成功 |

### 7.3 数据库操作统计

| 操作类型 | 数量 | 示例 |
|----------|------|------|
| INSERT | 8处 | 注册、发消息、建群、好友请求 |
| SELECT | 12处 | 查消息、查好友、查群成员 |
| UPDATE | 5处 | 更新状态、接受请求、已读 |
| DELETE | 4处 | 删好友、删消息、解散群 |
| JOIN | 6处 | 多表关联查询 |
| 事务 | 2处 | 接受好友、创建群组 |

### 7.4 总结

本项目实现了一个面向弱网环境优化的IM即时通讯系统，具备以下特点：

1. **完整功能**：用户管理、好友系统、群组聊天、实时消息
2. **弱网优化**：SSE自动重连、离线消息、心跳保活、消息截断
3. **安全防护**：SQL注入、XSS、DoS三种攻击的完整攻防演示
4. **数据库设计**：10张关联表，6种操作类型，完整ER关系
5. **前后端分离**：Go后端 + 原生Web前端，RESTful API

系统已通过全部功能测试和安全攻防测试，满足所有需求要求。