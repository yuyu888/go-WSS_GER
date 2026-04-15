# 调用场景说明

本文档覆盖 WSS 网关的全部调用场景，包括调用方法、链路说明和实际例子。

---

## 目录

1. [建立 WebSocket 连接](#1-建立-websocket-连接)
2. [req&resp：通过 WS 代理 HTTP 请求](#2-reqresp通过-ws-代理-http-请求)
3. [broadcast：本节点跨连接推送](#3-broadcast本节点跨连接推送)
4. [sendmsgtowssid：后端向指定 wssid 推送（本节点）](#4-sendmsgtowssid后端向指定-wssid-推送本节点)
5. [sendmsg：后端向指定用户推送（跨节点）](#5-sendmsg后端向指定用户推送跨节点)
6. [RPC 内部调用（50051）](#6-rpc-内部调用50051)

---

## 1. 建立 WebSocket 连接

### 调用方式

```
ws://<host>:<port>/ws?device_id=<设备号>
```

**参数说明：**

| 参数 | 位置 | 必填 | 说明 |
|------|------|------|------|
| `device_id` | Query | 是 | 客户端设备唯一标识，自定义字符串 |
| `uid` | Cookie | 否 | 登录用户 ID，未携带则以 device_id 作为会话 key |

### 链路说明

```
客户端
  → HTTP Upgrade → wsHandler
      ↓ 校验 device_id
      ↓ 读取 Cookie: uid
      ↓ 生成 wssid（UUID）
      ↓ 向客户端发送 wsInit 响应
      ↓ 注册到 WsManager
          ↓ 写入本地 WsClientPools（进程内存）
          ↓ 写入 Redis Hash: ws_go_{uid}  field=device_id  value={ws_server_addr, wssid}
          ↓ 写入 Redis String: ws_go_wssid_{wssid}  value=本节点IP
      ↓ 启动三个 goroutine：wsReadLoop / wsWriteLoop / procLoop
```

### 实际例子

```javascript
// 浏览器端
const ws = new WebSocket('ws://localhost:8080/ws?device_id=my-phone-001');

ws.onmessage = (e) => {
  const data = JSON.parse(e.data);
  if (data.action === 'wsInit') {
    console.log('连接成功，wssid:', data.wssid);
    // 后续请求都需要携带此 wssid
  }
};
```

**服务端立即下发的 wsInit 响应：**

```json
{
  "errcode": 200,
  "wssid": "f3feefd9-9e5d-450f-900a-873b7a736aa6",
  "request_id": "",
  "response_data": "websocket create success",
  "action": "wsInit"
}
```

---

## 2. req&resp：通过 WS 代理 HTTP 请求

客户端通过 WebSocket 发起 HTTP 请求，由服务端代理转发，结果推回到同一连接。

目标 URL 必须在配置文件 `[ws].allowed_urls` 白名单内。

### 请求结构

```json
{
  "wssid": "f3feefd9-9e5d-450f-900a-873b7a736aa6",
  "request_id": "自定义唯一ID，原样回显",
  "request_type": "req&resp",
  "action": "自定义动作名，原样回显",
  "request_data": {
    "http_method": "GET 或 POST",
    "request_url": "目标URL（必须在白名单内）",
    "post_data": "POST请求体，GET时传空字符串",
    "headers": {}
  }
}
```

### 链路说明

```
客户端发送 WS 消息
  → procLoop → process()
      ↓ 解析 JSON，校验公共字段（wssid/request_id/action/request_type/request_data）
      ↓ request_type = "req&resp" → doRequestBusiness()
          ↓ 校验 http_method（仅 GET/POST）
          ↓ 校验 request_url 在白名单内（精确匹配 scheme+host+path）
          ↓ libs/curl 发起 HTTP 请求到目标服务
          ↓ 目标服务返回结果
      ↓ 将结果通过原 WS 连接推回客户端
```

### 实际例子

**GET 请求：**

```json
{
  "wssid": "f3feefd9-9e5d-450f-900a-873b7a736aa6",
  "request_id": "req-001",
  "request_type": "req&resp",
  "action": "user.getInfo",
  "request_data": {
    "http_method": "GET",
    "request_url": "http://localhost/test?id=12345",
    "post_data": "",
    "headers": {}
  }
}
```

**POST 请求（含自定义 Header）：**

```json
{
  "wssid": "f3feefd9-9e5d-450f-900a-873b7a736aa6",
  "request_id": "req-002",
  "request_type": "req&resp",
  "action": "order.submit",
  "request_data": {
    "http_method": "POST",
    "request_url": "http://localhost/test",
    "post_data": "amount=100&item_id=999",
    "headers": {
      "X-Token": "your-auth-token",
      "Content-Type": "application/x-www-form-urlencoded"
    }
  }
}
```

**服务端响应（成功）：**

```json
{
  "errcode": 200,
  "wssid": "f3feefd9-9e5d-450f-900a-873b7a736aa6",
  "request_id": "req-002",
  "action": "order.submit",
  "response_data": "目标服务返回的原始内容"
}
```

---

## 3. broadcast：本节点跨连接推送

由某个 WebSocket 连接（A）向另一个连接（B）推送消息。

B 必须在同一节点，或服务端能通过 Redis 查到 B 所在节点并 RPC 转发（跨节点）。

### 请求结构

```json
{
  "wssid": "发送方的wssid",
  "request_id": "req-003",
  "request_type": "broadcast",
  "action": "chat.message",
  "request_data": {
    "wssid": "接收方的wssid",
    "message": "消息内容"
  }
}
```

### 链路说明

```
客户端 A 发送 WS 消息（request_type=broadcast）
  → process() → doSendMsgToWssid()
      ↓ 查本地 WsClientPools，目标 wssid 是否在当前节点
      ├─ 在本节点 → 直接推送给目标 WS 连接（客户端 B）
      └─ 不在本节点
            ↓ 查 Redis: ws_go_wssid_{目标wssid} → 获取目标节点 IP
            ↓ 调用 CrossNodeSendFunc（httpServer 注入的 RPC 回调）
            ↓ GetRpcClient(目标节点IP) → RPC 调用 Dispatch()
            ↓ 目标节点推送给客户端 B
  → 向客户端 A 回复操作结果
```

### 实际例子

```json
{
  "wssid": "f3feefd9-9e5d-450f-900a-873b7a736aa6",
  "request_id": "req-003",
  "request_type": "broadcast",
  "action": "chat.send",
  "request_data": {
    "wssid": "a1b2c3d4-0000-0000-0000-111122223333",
    "message": "你好，这是一条实时消息"
  }
}
```

**客户端 B 收到的内容**（原始字节，非 JSON 结构）：

```
你好，这是一条实时消息
```

**客户端 A 收到的回执：**

```json
{
  "errcode": 200,
  "wssid": "f3feefd9-9e5d-450f-900a-873b7a736aa6",
  "request_id": "req-003",
  "action": "chat.send",
  "response_data": "信息：你好，这是一条实时消息 发送给 a1b2c3d4-0000-0000-0000-111122223333"
}
```

---

## 4. sendmsgtowssid：后端向指定 wssid 推送（本节点）

后端服务知道目标连接的 `wssid`，且确认该连接在**当前节点**，直接推送。

### 调用方式

```
GET /sendmsgtowssid?wssid=<wssid>&msg=<消息内容>
```

### 链路说明

```
后端服务
  → GET /sendmsgtowssid?wssid=xxx&msg=yyy
      ↓ 从 WsClientPools 查找 wssid 对应的连接
      ↓ 找到 → 将 msg 写入该连接的 outChan
      ↓ wsWriteLoop 消费 outChan，推送给客户端
```

> **注意**：此接口不查 Redis，不走 RPC，只推送当前节点上的连接。  
> 多节点部署时，请改用 `/sendmsg`（第5节）。

### 实际例子

```bash
curl "http://localhost:8080/sendmsgtowssid?wssid=f3feefd9-9e5d-450f-900a-873b7a736aa6&msg=你有一条新消息"
```

**返回：**

```
信息：你有一条新消息 发送给 f3feefd9-9e5d-450f-900a-873b7a736aa6
```

---

## 5. sendmsg：后端向指定用户推送（跨节点）

后端服务向指定用户（`uid` 或 `deviceid`）的**所有设备**推送消息，自动处理跨节点路由。这是**生产环境最常用的推送入口**。

### 调用方式

```
POST /sendmsg?uid=<用户ID>&deviceid=<设备ID>

Body（表单格式）：
  msg=<消息内容>
```

**参数组合：**

| uid | deviceid | 行为 |
|-----|----------|------|
| 有 | 有 | 推送该用户指定设备的连接 |
| 有 | 无 | 推送该用户所有设备的连接 |
| 无 | 有 | 推送该设备 ID 关联的连接 |
| 无 | 无 | 返回 4001 错误 |

### 链路说明

```
后端服务
  → POST /sendmsg?uid=user_a
      ↓ 参数校验（uid/deviceid 至少一个，msg 不为空）
      ↓ usersession.GetInfo(uid, deviceid)
          ↓ Redis HGetAll: ws_go_{uid} → {deviceId: {ws_server_addr, wssid}}
      ↓ 遍历所有设备
          ↓ 解析 ws_server_addr（目标节点 IP）
          ├─ 目标节点 = 本节点 → DoSendMsgToWssid（直接推送）
          └─ 目标节点 ≠ 本节点
                ↓ GetRpcClient(目标节点IP:50051)
                ↓ RpcCall → 目标节点 Dispatch()
                ↓ 目标节点 DoSendMsgToWssid → 推送给客户端
      ↓ 返回 {"errcode":200, "response_data":"sent to N device(s)"}
```

### 实际例子

**推送给用户所有设备（最常见）：**

```bash
curl -X POST "http://localhost:8080/sendmsg?uid=user_a" \
  -d "msg=你有一条新通知"
```

**推送给用户指定设备：**

```bash
curl -X POST "http://localhost:8080/sendmsg?uid=user_a&deviceid=my-phone-001" \
  -d "msg=仅发给手机端"
```

**推送给未登录用户（仅有 device_id）：**

```bash
curl -X POST "http://localhost:8080/sendmsg?deviceid=my-phone-001" \
  -d "msg=设备推送"
```

**成功响应：**

```json
{"errcode": 200, "response_data": "sent to 2 device(s)"}
```

**错误响应示例：**

```json
{"errcode": 4001, "response_data": "uid, deviceId is empty"}
{"errcode": 4002, "response_data": "msg is empty"}
{"errcode": 5003, "response_data": "get user info failed: ..."}
```

---

## 6. RPC 内部调用（50051）

**此端口仅供节点间自动调用，外部系统不应直接访问。**

### 调用方式

由 `/sendmsg` 和 `broadcast` 跨节点时自动触发，调用方是同一套服务的其他节点。

**RPC 方法：**

```
服务名：transitData（由配置 [rpc].registername 决定）
方法名：Dispatch
参数：  Message{Content: string, Wssid: string}
返回：  Reply{Status: int, Data: interface{}}
```

### 链路说明

```
节点A（调用方）
  → GetRpcClient("10.0.0.5:50051")    // 按需建连，sync.Map 缓存
  → rpcClient.Call("Dispatch", msg)
      ↓
节点B（目标节点，10.0.0.5）
  → TransitData.Dispatch()
  → WsManager.DoSendMsgToWssid(wssid, content)
  → 从本地 WsClientPools 找到连接
  → 推送消息给客户端
```

### 为什么不对外暴露

- RPC 接口无鉴权，任何能访问 50051 的调用方都可以向任意 wssid 推送任意内容
- K8s 部署时只需要 Pod 之间网络互通，通过 NetworkPolicy 限制外部访问即可
- 外部推送统一走 `/sendmsg`（HTTP），由该接口内部决定是否走 RPC

---

## 错误码汇总

| errcode | 含义 | 触发场景 |
|---------|------|----------|
| 200 | 成功 | — |
| 4001 | 缺少 device_id / uid+deviceid 均为空 | WS 握手 / sendmsg |
| 4002 | msg 为空 | sendmsg |
| 4004 | 消息非合法 JSON | WS 消息解析 |
| 4006 | 不支持的 request_type | WS 消息处理 |
| 4101 | http_method 缺失或不支持 | req&resp |
| 4102 | 缺少 request_url | req&resp |
| 4103 | 缺少 post_data | req&resp |
| 4104 | 缺少 headers | req&resp |
| 4201 | 缺少 request_id | WS 公共字段校验 |
| 4202 | 缺少 wssid | WS 公共字段校验 |
| 4203 | 缺少 request_type | WS 公共字段校验 |
| 4204 | 缺少 request_data | WS 公共字段校验 |
| 4205 | 缺少 action | WS 公共字段校验 |
| 4290 | 触发限流 | HTTP 限流（enforce 模式） |
| 5001 | 代理请求失败（目标返回非 200） | req&resp |
| 5003 | 查询 Redis 会话失败 | sendmsg |

---

## 调用场景速查

| 我想做的事 | 使用哪个入口 |
|-----------|-------------|
| 建立 WebSocket 长连接 | `ws://host/ws?device_id=xxx` |
| 通过 WS 发起 HTTP 请求并获取结果 | WS 消息 `request_type=req&resp` |
| 从一个 WS 连接向另一个连接发消息 | WS 消息 `request_type=broadcast` |
| 后端向指定用户的所有设备推送 | `POST /sendmsg?uid=xxx` |
| 后端向指定设备推送 | `POST /sendmsg?uid=xxx&deviceid=yyy` |
| 后端向本节点指定 wssid 推送 | `GET /sendmsgtowssid?wssid=xxx&msg=yyy` |
| 节点间转发消息（内部自动） | RPC 50051（无需手动调用） |
