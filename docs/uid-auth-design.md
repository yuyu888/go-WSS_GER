# WebSocket 连接用户身份认证方案分析

## 现状与问题

### 当前实现

WebSocket 握手时，服务端从 Cookie 中读取 `uid` 作为登录用户标识：

```go
// wsServer/websocket.go
loginUid, err := req.Cookie("uid")
if err == nil {
    wsConn.loginUid = loginUid.Value
}
```

该 Cookie 由以下接口写入：

```go
// httpServer/index.go
uid := query.Get("uid")
if uid != "" {
    c1 := http.Cookie{Name: "uid", Value: uid, HttpOnly: true}
    http.SetCookie(w, &c1)
}
```

### 根本问题：uid 未经验证，可任意伪造

任何人只需访问 `GET /?uid=目标用户ID` 即可将任意 uid 写入自己的 Cookie，随后以该 uid 建立 WebSocket 连接：

```
GET /?uid=victim_uid          →  Set-Cookie: uid=victim_uid
ws://host/ws?device_id=aaa   →  服务端以 victim_uid 注册连接
```

**后果**：后端对 `victim_uid` 的推送消息会发到攻击者的连接，造成消息泄漏。

Cookie 的 `HttpOnly` 属性只防止 JavaScript 读取，并不能阻止客户端伪造 Cookie 值。

---

## 可选方案

### 方案一：Query 参数传签名 Token

连接 URL 携带由业务后端签发的短期 Token（如 JWT 或 HMAC 签名串）：

```
ws://host/ws?device_id=aaa&token=<signed_token>
```

WSS 服务端验证签名，从 Payload 中提取 uid：

```
业务后端签发：HMAC-SHA256(uid + device_id + timestamp, secret_key)
WSS 验证：重新计算签名并比对，检查 timestamp 防重放
```

| | |
|---|---|
| **优点** | 实现简单，无需额外存储 |
| **缺点** | Token 会出现在服务器日志、代理日志、浏览器历史中，存在泄漏风险；不适合生产环境 |

**适用场景**：内网服务、受控环境、快速验证阶段。

---

### 方案二：一次性 Ticket 换 uid（推荐）

WebSocket 握手前，客户端先向业务后端申请一个短期、一次性的 ticket，再用 ticket 建立 WS 连接：

```
1. 客户端（已登录）→ 业务后端：POST /auth/ws-ticket
2. 业务后端 → Redis：SET ws_ticket:{uuid} {uid}  EX 30
3. 业务后端 → 客户端：{"ticket": "uuid", "ttl": 30}
4. 客户端 → WSS：ws://host/ws?device_id=aaa&ticket=uuid
5. WSS → Redis：GETDEL ws_ticket:{uuid}  → 取出 uid，同时消费掉
```

关键特性：
- ticket **生命周期 10~30 秒**，超时自动失效
- **一次性消费**，用后即删，重放无效
- ticket 本身不携带任何用户信息，泄漏了也无法推断 uid

| | |
|---|---|
| **优点** | 安全性高，ticket 短命且一次性，泄漏危害极低 |
| **缺点** | 业务方需改造，客户端多一次 HTTP 请求 |

**适用场景**：互联网生产环境，用户数据安全要求较高的场景。

---

### 方案三：业务后端颁发签名 Cookie（当前方案的正确做法）

Cookie 方案本身没有问题，问题在于 Cookie 的值是客户端可控的明文。

正确做法是由**业务后端在登录时颁发签名 Cookie**（如 JWT 或加密 Session ID），WSS 服务端验证 Cookie 的签名：

```
登录时：业务后端 Set-Cookie: session=<signed_value>; HttpOnly; Secure; SameSite=Strict
WS 握手：WSS 验证 session Cookie 签名 → 提取 uid
```

| | |
|---|---|
| **优点** | 浏览器自动携带，无需客户端额外处理；与现有登录体系集成自然 |
| **缺点** | WSS 服务需要能验证业务后端签发的 Cookie（共享密钥或调用验证接口） |

**适用场景**：WSS 与业务后端同域部署，且已有成熟的 Cookie 登录体系。

---

## 针对本项目的建议

本项目定位为**独立的 WebSocket 网关**，不承担登录鉴权职责，uid 由上游业务系统提供。

**推荐采用方案二（Ticket 换 uid）**，理由：

1. 网关与业务后端职责分离，网关只需从 Redis 取 ticket 兑换 uid，不需要知道业务的 session 机制
2. 对现有代码改动最小：仅修改 `wsHandler` 中读取 uid 的部分，增加一个 `model.WsTicket` Redis 操作
3. 业务后端改造成本低：生成 UUID 写入 Redis 即可，无需引入 JWT 库

**次选方案一（签名 Token）**：如果你的部署环境是内网或 K8s 集群内部调用，且接受日志中出现 token，方案一改动最小，可作为过渡方案。

---

## 近期需要处理的问题

除认证问题外，`httpHandlerIndex` 中允许任意设置 uid Cookie 的逻辑**应当移除**，它既是安全漏洞也是无效代码：

```go
// 应删除以下逻辑
uid := query.Get("uid")
if uid != "" {
    c1 := http.Cookie{Name: "uid", Value: uid, HttpOnly: true}
    http.SetCookie(w, &c1)
}
```
