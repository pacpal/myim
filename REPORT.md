# 防篡改审计IM即时通讯系统 — 项目报告

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

即时通讯（Instant Messaging, IM）系统已成为企业办公、金融沟通、法律协作等场景的核心基础设施。然而，传统IM系统普遍存在一个被忽视的安全隐患——**数据可被内部人员直接篡改而难以发现**。

在实际应用中，IM系统面临以下数据完整性威胁：

- **内部威胁**：拥有数据库权限的DBA或运维人员可直接执行 `UPDATE messages SET content=...` 修改历史消息内容，而应用层完全无感知。在金融沟通、合同确认等场景中，消息被篡改可能导致严重后果。
- **SQL注入写权限**：当系统存在SQL注入漏洞且数据库账户有写权限时，攻击者可通过注入篡改消息内容。
- **审计日志自身可信度不足**：传统审计日志存储在数据库中，管理员可删除 `DELETE FROM audit_logs` 来抹除操作痕迹，使审计形同虚设。
- **篡改难以定位**：即使发现数据异常，也难以确定哪条记录在何时被谁篡改。

区块链通过分布式共识解决了数据防篡改问题，但对IM这类中心化系统而言，引入完整区块链过于沉重。**哈希链（Hash Chain）** 提供了一种轻量级方案：每条记录存储前一条记录的哈希，形成链式依赖，任何单点篡改都会导致后续哈希链断裂，从而被精确检测。

### 1.2 待解决问题

本项目旨在解决**IM系统消息与审计日志的防篡改问题**，具体包括：

1. **消息完整性保护**：防止消息内容被直接修改数据库而未被发现
2. **篡改精确定位**：当篡改发生时，能准确定位被篡改的记录及其位置
3. **审计日志自身防篡改**：审计日志同样受哈希链保护，防止管理员删除日志
4. **传统Web安全防护**：同时防御SQL注入、XSS、DoS等常见攻击
5. **弱网环境可靠性**：在弱网条件下保证消息可靠投递（作为工程实现约束）

### 1.3 项目意义

1. **创新价值**：提出基于哈希链的IM防篡改审计架构，将密码学哈希链应用于IM消息与审计日志的双重保护，在IM领域较为罕见
2. **实用价值**：为金融沟通、法律协作、合同确认等对数据完整性要求高的场景提供可验证的消息可信度保障
3. **安全价值**：通过四种攻击的攻防对照演示，完整展示Web安全威胁与防御方法
4. **教学价值**：综合实践前后端分离架构、数据库设计、密码学哈希链、实时通信、安全防护

### 1.4 技术选型理由

| 技术 | 选择 | 理由 |
|------|------|------|
| 后端语言 | Go | 高并发、编译型、性能优异 |
| Web框架 | Gin | 轻量高效、API友好 |
| 数据库 | MySQL | 关系型、事务支持、生态成熟 |
| 实时通信 | SSE | 单向持久连接、自动重连、适合弱网 |
| 哈希算法 | SHA-256 | 密码学安全哈希、抗碰撞、性能良好 |
| 密码加密 | bcrypt | 自适应成本、抗彩虹表 |
| 前端 | HTML/CSS/JS | 无需构建工具、轻量、兼容性好 |

---

## 二、需求分析

### 2.1 功能需求

#### 2.1.1 用户管理
- 用户注册（用户名、密码、昵称）
- 用户登录/登出
- 个人资料查看与修改
- 角色区分（普通用户 role=0，审计管理员 role=1）

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

#### 2.1.5 防篡改审计（核心创新）
- 消息哈希链：每条消息存储 prev_hash 与 curr_hash，形成链式依赖
- 完整性校验：扫描全部哈希链，检测内容篡改与链断裂
- 篡改精确定位：准确定位被篡改记录的ID、类型、期望哈希与实际哈希
- 防篡改审计日志：所有敏感操作记录入审计日志，日志自身受哈希链保护
- 实时告警：检测到篡改时通过SSE实时推送告警给管理员
- 审计中心：管理员可查看篡改告警、审计日志、执行完整性校验

#### 2.1.6 安全攻防演示
- SQL注入攻击与防御
- XSS跨站脚本攻击与防御
- DoS拒绝服务攻击与防御
- 数据篡改攻击与哈希链检测防御（创新点）

### 2.2 非功能需求

| 需求 | 描述 |
|------|------|
| 防篡改 | SHA-256哈希链保护消息与审计日志，篡改可检测可定位 |
| 弱网优化 | SSE自动重连、离线消息存储、心跳保活、消息内容截断 |
| 安全性 | 参数化查询、HTML转义、速率限制、安全响应头、CSP策略 |
| 可扩展性 | 模块化设计、Hub模式管理连接 |
| 响应性 | 前后端分离、异步通信 |

### 2.3 工具与方法基本原理

#### 2.3.1 哈希链（Hash Chain）—— 核心创新

哈希链是一种将密码学哈希函数迭代应用于数据序列，形成链式依赖结构的技术。

**原理**：
```
curr_hash_n = SHA256(prev_hash_{n-1} || content_n || from_user_id_n || created_at_n)
prev_hash_n = curr_hash_{n-1}
```

每条记录存储前一条记录的哈希值（prev_hash）和自身计算出的哈希值（curr_hash）。篡改检测逻辑：

- **内容篡改**：若记录内容被修改但 curr_hash 未更新，重新计算的哈希 ≠ 存储 curr_hash
- **链断裂**：若记录被删除或插入，prev_hash ≠ 上一条记录的 curr_hash

**优势**：
- 轻量级：相比区块链无需共识机制，仅依赖单向哈希函数
- 可定位：能精确指出哪条记录被篡改
- 自保护：审计日志自身也用哈希链，防止管理员删日志

#### 2.3.2 SSE（Server-Sent Events）
SSE是HTML5标准的一部分，允许服务器通过HTTP连接向浏览器推送数据。

**原理**：
- 客户端通过`EventSource` API建立持久HTTP连接
- 服务器以`text/event-stream`格式持续推送数据
- 浏览器自动处理重连（内置retry机制）
- 适合服务器到客户端的单向数据流（如篡改告警推送）

#### 2.3.3 参数化查询（Prepared Statement）
**原理**：SQL语句与数据分离，数据库引擎将参数仅作为数据处理，不会解析为SQL语法。

```
漏洞：SELECT * FROM users WHERE username = '" + input + "'"
安全：SELECT * FROM users WHERE username = ?
```

#### 2.3.4 HTML转义（XSS防御）
**原理**：将HTML特殊字符转换为实体编码，使浏览器将其作为文本显示而非代码执行。

```
<  →  &lt;
>  →  &gt;
"  →  &quot;
'  →  &#39;
&  →  &amp;
```

#### 2.3.5 速率限制（Rate Limiting，DoS防御）
**原理**：基于滑动窗口算法，限制每个IP在时间窗口内的请求次数，超过阈值返回429状态码。

---

## 三、系统设计

### 3.1 整体架构

```
┌──────────────────────────────────────────────────────────────┐
│                         浏览器前端                            │
│  ┌──────────┐ ┌──────────┐ ┌──────────┐ ┌───────────────┐  │
│  │ 登录注册  │ │ 聊天界面  │ │ 安全演示  │ │ 审计中心(管理员)│  │
│  └────┬─────┘ └────┬─────┘ └────┬─────┘ └───────┬───────┘  │
│       │            │            │               │          │
│       │  HTTP API  │  SSE长连接  │   HTTP API    │  SSE告警  │
└───────┼────────────┼────────────┼───────────────┼──────────┘
        │            │            │               │
┌───────▼────────────▼────────────▼───────────────▼──────────┐
│                    Go后端 (Gin)                             │
│  ┌─────────┐ ┌──────────┐ ┌──────────┐ ┌──────────────┐     │
│  │Auth中间件│ │限流中间件│ │CORS中间件│ │Admin中间件   │     │
│  └────┬────┘ └────┬─────┘ └────┬─────┘ └──────┬───────┘     │
│       │           │            │               │            │
│  ┌────▼───────────▼────────────▼───────────────▼─────────┐  │
│  │              Handler路由层                             │  │
│  │ auth friend group message attack integrity audit       │  │
│  └────────────────────┬───────────────────────────────────┘  │
│                       │                                     │
│  ┌──────────┐  ┌──────▼──────┐  ┌────────────────────────┐  │
│  │哈希链工具 │  │ SSE Hub    │  │  审计日志记录器        │  │
│  │(SHA-256) │  │ (连接管理)  │  │  (自身哈希链保护)     │  │
│  └──────────┘  └────────────┘  └────────────────────────┘  │
│  ┌────────────────────────────────────────────────────────┐ │
│  │              Database层 (MySQL)                        │ │
│  └────────────────────────────────────────────────────────┘ │
└────────────────────────────────────────────────────────────┘
```

### 3.2 模块划分

| 模块 | 文件 | 职责 |
|------|------|------|
| 入口 | main.go | 路由注册、中间件配置、启动服务 |
| 数据库 | database/db.go, schema.sql | 连接管理、表结构定义、管理员初始化 |
| 模型 | models/models.go | 数据结构定义 |
| 中间件 | middleware/middleware.go | 认证、限流、CORS、安全头、管理员鉴权 |
| Hub | hub/hub.go | SSE连接管理、消息推送 |
| 认证 | handlers/auth.go | 注册、登录、登出、资料、审计记录 |
| 好友 | handlers/friend.go | 好友请求、好友列表、搜索 |
| 群组 | handlers/group.go | 群组CRUD、成员管理 |
| 消息 | handlers/message.go | 私聊、群聊、SSE、哈希链计算 |
| 攻防 | handlers/attack.go | SQL注入/XSS/DoS演示 |
| **哈希链** | **util/hashchain.go** | **SHA-256哈希链计算** |
| **审计** | **handlers/audit.go** | **防篡改审计日志记录** |
| **完整性** | **handlers/integrity.go** | **完整性校验、篡改演示、管理员接口** |
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
| POST | /api/messages | 发送私聊（含哈希链） | 是 |
| GET | /api/messages/:id | 聊天记录 | 是 |
| DELETE | /api/messages/:id | 删除消息 | 是 |
| POST | /api/groups/:id/messages | 群聊消息（含哈希链） | 是 |
| GET | /api/groups/:id/messages | 群聊记录 | 是 |
| GET | /api/sse | SSE实时推送 | 是 |
| GET | /api/attack/sql/vulnerable | SQL注入漏洞 | 否 |
| GET | /api/attack/sql/safe | SQL注入防御 | 否 |
| POST | /api/attack/xss/vulnerable | XSS漏洞 | 否 |
| POST | /api/attack/xss/safe | XSS防御 | 否 |
| GET | /api/attack/dos/vulnerable | DoS漏洞 | 否 |
| POST | /api/attack/tamper/:id | 数据篡改攻击 | 是 |
| GET | /api/integrity/check | 完整性校验 | 是 |
| GET | /api/admin/audit/logs | 审计日志查询 | 管理员 |
| GET | /api/admin/integrity/alerts | 篡改告警查询 | 管理员 |
| PUT | /api/admin/integrity/alerts/:id/resolve | 处理告警 | 管理员 |

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
│    role      │   │     │    status    │
│    status    │   │     │    created_at│
│    last_login│   │     └──────────────┘
│    created_at│   │
└──────┬───────┘   │
       │           │     ┌──────────────┐
       │ 1         │     │ audit_logs   │ (自身哈希链保护)
       │           │     │──────────────│
       │ N         │     │ PK id        │
┌──────▼───────┐   │     │ FK actor_id ─┘
│   friends    │   │     │    actor_name│
│──────────────│   │     │    action    │
│ PK id        │   │     │    detail    │
│ FK user_id ──┘   │     │    ip_address│
│ FK friend_id ────┘     │    prev_hash  │── 哈希链
│    remark    │         │    curr_hash  │── 哈希链
│    created_at│         │    created_at│
└──────────────┘         └──────────────┘

┌──────────────┐         ┌──────────────┐
│   groups     │         │group_members  │
│──────────────│         │──────────────│
│ PK id        │◄────────│ PK id        │
│    name      │         │ FK group_id  │
│ FK owner_id ──►users   │ FK user_id ──►users
│    created_at│         │    role      │
└──────┬───────┘         └──────────────┘
       │
       │ 1         ┌──────────────┐
       │           │  messages    │ (哈希链保护)
       │ N         │──────────────│
┌──────▼───────┐   │ PK id        │
│group_messages│   │ FK from_user──►users
│──────────────│   │ FK to_user ───►users
│ PK id        │   │    content   │
│ FK group_id ─┘   │    prev_hash │── 哈希链
│ FK from_user──►users│  curr_hash  │── 哈希链
│    content   │   │    created_at│
│    prev_hash │── │    status    │
│    curr_hash │── └──────────────┘
│    created_at│
└──────────────┘         ┌──────────────┐
                         │integrity_    │
                         │  alerts      │
                         │──────────────│
                         │ PK id        │
                         │ target_type  │
                         │ target_id    │
                         │ expected_hash│
                         │ actual_hash  │
                         │ reason       │
                         │ handled     │
                         │ created_at  │
                         └──────────────┘
```

### 4.2 表结构说明（12张表）

| 序号 | 表名 | 说明 | 关联 | 哈希链 |
|------|------|------|------|--------|
| 1 | users | 用户账户 | 被多表引用 | - |
| 2 | friends | 好友关系 | user_id, friend_id → users | - |
| 3 | friend_requests | 好友请求 | from_user_id, to_user_id → users | - |
| 4 | groups | 群组 | owner_id → users | - |
| 5 | group_members | 群成员 | group_id → groups, user_id → users | - |
| 6 | messages | 私聊消息 | from_user_id, to_user_id → users | ✅ |
| 7 | group_messages | 群聊消息 | group_id → groups, from_user_id → users | ✅ |
| 8 | message_ack | 消息回执 | message_id → messages, user_id → users | - |
| 9 | offline_messages | 离线消息 | user_id → users, message_id → messages | - |
| 10 | login_logs | 登录日志 | user_id → users | - |
| 11 | audit_logs | 审计日志 | actor_id → users | ✅ |
| 12 | integrity_alerts | 篡改告警 | - | - |

### 4.3 数据库操作类型（大于4种）

| 操作类型 | SQL语句 | 应用场景 |
|----------|---------|----------|
| INSERT | INSERT INTO users... | 注册用户、发送消息、记录审计、生成告警 |
| DELETE | DELETE FROM friends... | 删除好友、解散群组、删除消息 |
| UPDATE | UPDATE users SET status... | 更新在线状态、消息已读、处理告警 |
| SELECT | SELECT * FROM messages... | 查询消息、好友列表、审计日志 |
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
im/
├── main.go                    # 程序入口
├── go.mod / go.sum            # 依赖管理
├── database/
│   ├── db.go                 # 数据库连接 + 管理员初始化
│   └── schema.sql            # 建表脚本（12张表）
├── models/
│   └── models.go              # 数据模型
├── middleware/
│   └── middleware.go          # 认证/限流/CORS/安全头/管理员鉴权
├── hub/
│   └── hub.go                # SSE连接管理
├── util/
│   └── hashchain.go           # SHA-256哈希链计算（核心）
├── handlers/
│   ├── auth.go               # 认证处理 + 审计记录
│   ├── friend.go             # 好友处理
│   ├── group.go              # 群组处理
│   ├── message.go            # 消息处理 + 哈希链计算
│   ├── attack.go             # SQL注入/XSS/DoS演示
│   ├── audit.go              # 防篡改审计日志记录器
│   └── integrity.go          # 完整性校验 + 篡改演示
├── utils/
│   └── utils.go               # 工具函数
└── frontend/
    ├── index.html             # 首页
    ├── login.html             # 登录
    ├── register.html          # 注册
    ├── chat.html              # 聊天界面
    ├── security.html          # 安全演示
    ├── admin.html             # 审计中心（管理员）
    ├── css/style.css          # 样式
    └── js/
        ├── auth.js            # API工具
        ├── chat.js            # 聊天逻辑
        ├── security.js        # 攻防逻辑
        └── admin.js           # 审计中心逻辑
```

### 5.2 核心代码说明

#### 5.2.1 哈希链计算（util/hashchain.go）—— 核心创新

```go
// ComputeHash 计算消息哈希链值
// 公式: SHA256(prev_hash || content || fromUserID || createdAt)
func ComputeHash(prevHash, content string, fromUserID int, createdAt string) string {
    data := prevHash + content + strconv.Itoa(fromUserID) + createdAt
    sum := sha256.Sum256([]byte(data))
    return hex.EncodeToString(sum[:])
}

// ComputeAuditHash 计算审计日志哈希链值
// 公式: SHA256(prev_hash || actor_id || action || detail || created_at)
func ComputeAuditHash(prevHash string, actorID int, action, detail, createdAt string) string {
    data := prevHash + strconv.Itoa(actorID) + action + detail + createdAt
    sum := sha256.Sum256([]byte(data))
    return hex.EncodeToString(sum[:])
}
```

#### 5.2.2 发送消息时计算哈希链（handlers/message.go）

发送消息时，先获取上一条消息的 curr_hash 作为 prev_hash，插入后读取 created_at，再计算并更新 curr_hash：

```go
// 获取上一条消息哈希
var prevHash string
database.DB.QueryRow("SELECT curr_hash FROM messages ORDER BY id DESC LIMIT 1").Scan(&prevHash)

// 插入消息（含 prev_hash）
result, err := database.DB.Exec(
    "INSERT INTO messages (from_user_id, to_user_id, content, msg_type, prev_hash) VALUES (?, ?, ?, ?, ?)",
    fromUserID, req.ToUserID, content, req.MsgType, prevHash)

msgID, _ := result.LastInsertId()

// 计算并持久化哈希链值
var createdAt string
database.DB.QueryRow("SELECT created_at FROM messages WHERE id = ?", msgID).Scan(&createdAt)
currHash := util.ComputeHash(prevHash, content, fromUserID, createdAt)
database.DB.Exec("UPDATE messages SET curr_hash = ? WHERE id = ?", currHash, msgID)

// 记录防篡改审计日志
RecordAudit(fromUserID, fromName, "send_msg", "私聊消息#"+id, c.ClientIP())
```

#### 5.2.3 完整性校验（handlers/integrity.go）—— 核心防御

完整性校验扫描全部哈希链，检测两类异常：

```go
func IntegrityCheck(c *gin.Context) {
    // 遍历所有消息
    rows, _ := database.DB.Query(
        "SELECT id, prev_hash, curr_hash, content, from_user_id, created_at FROM messages ORDER BY id ASC")
    expectedPrev := ""
    for rows.Next() {
        // 链断裂检测：prev_hash != 上一条 curr_hash
        if prevHash != expectedPrev {
            alerts = append(alerts, "哈希链断裂：prev_hash 与上一条记录不符")
        }
        // 内容篡改检测：重算哈希 != 存储 curr_hash
        recomputed := util.ComputeHash(prevHash, content, fromUserID, createdAt)
        if recomputed != currHash {
            alerts = append(alerts, "内容被篡改：重新计算的哈希与存储值不符")
        }
        expectedPrev = currHash
    }
    // 同样检测 group_messages 和 audit_logs 链
    // 发现异常时持久化告警 + SSE实时推送管理员
}
```

#### 5.2.4 防篡改审计日志（handlers/audit.go）

审计日志自身也受哈希链保护，防止管理员删除日志：

```go
func RecordAudit(actorID int, actorName, action, detail, ip string) {
    // 获取上一条审计日志哈希
    var prevHash string
    database.DB.QueryRow("SELECT curr_hash FROM audit_logs ORDER BY id DESC LIMIT 1").Scan(&prevHash)

    // 插入审计记录
    res, _ := database.DB.Exec(
        "INSERT INTO audit_logs (actor_id, actor_name, action, detail, ip_address, prev_hash) VALUES (?,?,?,?,?,?)",
        actorID, actorName, action, detail, ip, prevHash)

    // 计算并更新哈希链值
    id, _ := res.LastInsertId()
    var createdAt string
    database.DB.QueryRow("SELECT created_at FROM audit_logs WHERE id = ?", id).Scan(&createdAt)
    currHash := util.ComputeAuditHash(prevHash, actorID, action, detail, createdAt)
    database.DB.Exec("UPDATE audit_logs SET curr_hash = ? WHERE id = ?", currHash, id)
}
```

#### 5.2.5 SSE实时推送（hub/hub.go + handlers/message.go）
- Hub维护用户ID到客户端通道的映射
- 客户端连接时注册，断开时注销
- 发送消息时通过Hub推送到目标用户的SSE连接
- 检测到篡改时实时推送告警给所有管理员
- 心跳机制保持连接活跃

#### 5.2.6 弱网优化策略
1. **SSE自动重连**：浏览器EventSource内置重连机制，断线后3秒自动重连
2. **离线消息存储**：接收方不在线时，消息存入offline_messages表，上线后自动推送
3. **心跳保活**：每30秒发送心跳注释行，防止连接超时
4. **消息内容截断**：限制消息最大长度5000字符，减少带宽占用

#### 5.2.7 安全防御实现
1. **SQL注入防御**：所有数据库查询使用参数化查询（`?`占位符）
2. **XSS防御**：使用`html.EscapeString`转义输出，前端使用`textContent`而非`innerHTML`
3. **DoS防御**：基于IP的滑动窗口速率限制（120次/分钟），登录接口更严格（10次/分钟）
4. **数据篡改防御**：SHA-256哈希链 + 完整性校验 + 防篡改审计日志

### 5.3 功能截图说明

系统运行后可通过以下地址访问：
- 首页：http://localhost:8080/
- 登录：http://localhost:8080/static/login.html
- 注册：http://localhost:8080/static/register.html
- 聊天：http://localhost:8080/static/chat.html
- 安全演示：http://localhost:8080/static/security.html
- 审计中心：http://localhost:8080/static/admin.html（管理员）

#### 主要界面功能：
1. **首页**：展示系统特性，提供登录/注册/安全演示入口
2. **注册页**：输入用户名、昵称、密码注册账户
3. **登录页**：输入用户名密码登录，获取会话Token
4. **聊天界面**：
   - 左侧栏：好友列表/群组列表/好友请求（标签切换）
   - 右侧：聊天区域，支持发送/接收消息
   - 管理员可见"审计中心"入口
5. **安全演示页**：
   - SQL注入：对比漏洞接口与参数化查询防御
   - XSS：对比原始输出与HTML转义防御
   - DoS：对比无限制接口与速率限制防御
   - 数据篡改：模拟篡改数据库 + 哈希链检测（创新点）
6. **审计中心（管理员）**：
   - 篡改告警：查看所有检测到的篡改记录，含期望/实际哈希对比
   - 审计日志：查看所有敏感操作记录，含哈希链值
   - 完整性校验：一键扫描全部哈希链

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

### 6.2 XSS跨站脚本攻击与防御

#### 攻击原理
漏洞接口直接返回用户输入的原始内容，前端使用`innerHTML`渲染，导致脚本执行：
```javascript
// 漏洞：直接渲染原始内容
element.innerHTML = data.content  // <img src=x onerror=alert(1)> 会执行
```

#### 防御方法（3种）
1. **HTML转义**（主要）：将 `<` `>` `"` `'` `&` 转为实体编码
2. **CSP策略**：`Content-Security-Policy: script-src 'self'` 阻止内联脚本
3. **HttpOnly Cookie**：防止XSS窃取会话Cookie

### 6.3 DoS拒绝服务攻击与防御

#### 攻击原理
漏洞接口无任何限制，执行重计算操作，攻击者并发大量请求耗尽服务器资源。

#### 防御方法（3种）
1. **速率限制**（主要）：滑动窗口算法，120次/分钟/IP
2. **连接数限制**：限制并发连接数
3. **请求频率控制**：登录接口更严格（10次/分钟）

### 6.4 数据篡改攻击与哈希链检测（创新点）

#### 攻击原理
攻击者（如DBA或通过SQL注入写权限）直接修改数据库中消息内容，但不更新哈希值：
```sql
-- 攻击者直接执行
UPDATE messages SET content = '老板说明天涨薪取消' WHERE id = 8
-- curr_hash 未更新，但应用层无感知
```

#### 防御方法：哈希链检测

**哈希链结构**：
```
消息1: prev_hash=""      curr_hash=SHA256(""||content1||user1||time1)
消息2: prev_hash=hash1    curr_hash=SHA256(hash1||content2||user2||time2)
消息3: prev_hash=hash2    curr_hash=SHA256(hash2||content3||user3||time3)
```

**检测逻辑**：
1. **内容篡改检测**：重新计算 `SHA256(prev_hash||content||from_user_id||created_at)`，若 ≠ 存储 `curr_hash`，则内容被篡改
2. **链断裂检测**：若 `prev_hash` ≠ 上一条记录的 `curr_hash`，则记录被删除或插入

**审计日志自身防篡改**：审计日志表也采用哈希链结构，管理员删除 `DELETE FROM audit_logs` 会导致链断裂被检测。

#### 演示流程
1. 在IM中发送几条消息（自动计算哈希链）
2. 在安全演示页填入消息ID，点击"执行篡改"模拟直接改库
3. 点击"执行完整性校验"，系统精确定位被篡改记录：
   - 显示目标类型与ID
   - 显示期望哈希（重算）与实际哈希（存储）对比
   - 显示篡改原因
4. 管理员在审计中心收到实时SSE告警

#### 测试结果
```
篡改前校验: 完整 ✅ alerts=0
篡改后校验: 已发现篡改 🚨 alerts=1
  → target_id=8, target_type=message
  → reason=内容被篡改：重新计算的哈希与存储值不符
  → expected=bab9e040191a366bb04aef544c2d37cf2c37e4963dde1c584fc8ca02f57cbc94
  → actual=01cac1eea9c90485d95a96eed29965d9585e93855e00f510a179b9325c396761
```

### 6.5 防御方法总结

| 攻击类型 | 防御方法1 | 防御方法2 | 防御方法3 |
|----------|-----------|-----------|-----------|
| SQL注入 | 参数化查询 | 输入验证 | 最小权限原则 |
| XSS | HTML转义 | CSP策略 | HttpOnly Cookie |
| DoS | 速率限制 | 连接数限制 | 请求频率控制 |
| 数据篡改 | SHA-256哈希链 | 审计日志防篡改 | 完整性定期校验 |

---

## 七、测试与运行

### 7.1 环境搭建与启动

1. **启动MySQL**（Docker方式）：
   ```bash
   docker run -d --name im_mysql -p 3307:3306 \
     -e MYSQL_ROOT_PASSWORD=root \
     -e MYSQL_DATABASE=im_system \
     mysql:latest
   ```

2. **初始化数据库**：
   ```bash
   Get-Content database/schema.sql -Raw | docker exec -i im_mysql mysql -uroot -proot im_system
   ```

3. **启动后端**：
   ```bash
   go run main.go
   ```
   服务启动后自动创建管理员账号（admin / admin123）

4. **访问系统**：
   - 打开浏览器访问 http://localhost:8080/

### 7.2 测试用例

| 测试项 | 操作 | 预期结果 | 实际结果 |
|--------|------|----------|----------|
| 用户注册 | 输入用户名密码注册 | 注册成功 | ✅通过 |
| 用户登录 | 输入正确密码登录 | 返回Token+role | ✅通过 |
| 管理员登录 | admin/admin123登录 | role=1，显示审计入口 | ✅通过 |
| 搜索用户 | 输入关键词搜索 | 返回匹配用户 | ✅通过 |
| 好友请求 | 发送好友请求 | 请求发送成功 | ✅通过 |
| 接受请求 | 接受好友请求 | 双向添加好友 | ✅通过 |
| 发送私聊 | 发送消息给好友 | 消息存储+哈希链计算 | ✅通过 |
| 查看记录 | 查看聊天历史 | 返回消息列表 | ✅通过 |
| 创建群组 | 创建新群组 | 群组创建成功 | ✅通过 |
| 群聊消息 | 发送群聊消息 | 所有成员收到+哈希链 | ✅通过 |
| SQL注入 | 漏洞接口注入 | 返回所有用户 | ✅攻击成功 |
| SQL防御 | 参数化查询 | 返回空结果 | ✅防御成功 |
| XSS攻击 | 发送脚本标签 | 脚本执行 | ✅攻击成功 |
| XSS防御 | HTML转义 | 显示为文本 | ✅防御成功 |
| DoS攻击 | 并发请求 | 服务器处理 | ✅攻击成功 |
| DoS防御 | 速率限制 | 部分请求被拒 | ✅防御成功 |
| 数据篡改 | 模拟改库不更新哈希 | 篡改成功 | ✅攻击成功 |
| 完整性校验(篡改前) | 执行校验 | 完整 ✅ | ✅通过 |
| 完整性校验(篡改后) | 执行校验 | 发现篡改 🚨 | ✅防御成功 |
| 篡改定位 | 查看告警详情 | 精确定位ID+哈希对比 | ✅通过 |
| 审计日志 | 查看审计中心 | 显示操作记录+哈希链 | ✅通过 |
| 实时告警 | 篡改后管理员收告警 | SSE推送红色告警 | ✅通过 |

### 7.3 数据库操作统计

| 操作类型 | 数量 | 示例 |
|----------|------|------|
| INSERT | 10处 | 注册、发消息、建群、好友请求、记录审计、生成告警 |
| SELECT | 14处 | 查消息、查好友、查群成员、查审计日志、查告警 |
| UPDATE | 6处 | 更新状态、接受请求、已读、处理告警、更新哈希 |
| DELETE | 4处 | 删好友、删消息、解散群 |
| JOIN | 6处 | 多表关联查询 |
| 事务 | 2处 | 接受好友、创建群组 |

### 7.4 总结

本项目实现了一个**基于哈希链的防篡改审计IM即时通讯系统**，具备以下特点：

1. **核心创新——防篡改审计**：采用SHA-256哈希链保护消息与审计日志，实现篡改检测与精确定位。审计日志自身也受哈希链保护，防止管理员删除日志抹除痕迹
2. **完整功能**：用户管理、好友系统、群组聊天、实时消息
3. **四种攻防演示**：SQL注入、XSS、DoS、数据篡改的完整攻防对照，其中数据篡改+哈希链检测为创新点
4. **数据库设计**：12张关联表，6种操作类型，完整ER关系，3张表采用哈希链保护
5. **弱网优化**：SSE自动重连、离线消息、心跳保活、消息截断（作为工程实现约束）
6. **前后端分离**：Go后端 + 原生Web前端，RESTful API，管理员审计中心
7. **实时告警**：检测到篡改时通过SSE实时推送告警给管理员

系统已通过全部功能测试和安全攻防测试，满足所有需求要求。