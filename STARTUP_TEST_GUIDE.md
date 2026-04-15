# go-WSS_GER 启动与测试说明

## 1. 环境准备

- 安装 Go（建议 1.21+，与 `go.mod` 保持兼容）。
- 确认可访问 **Redis**（支持单机或哨兵模式，见第 2 节配置说明）。
- 首次拉取代码后执行：

```bash
go mod tidy
```

## 2. 配置文件说明

项目按环境读取配置：

- 开发环境：`config/config_dev.toml`
- 生产环境：`config/config_prod.toml`

### Redis 配置

支持单机和哨兵两种模式，通过 `mode` 字段切换：

**单机模式：**
```toml
[redis]
mode = "standalone"
addr = "127.0.0.1:6379"
password = ""
db = 0
```

**哨兵模式：**
```toml
[redis]
mode = "sentinel"
sentinel_addrs = ["10.x.x.1:26379", "10.x.x.2:26379", "10.x.x.3:26379"]
master_name = "mymaster"
password = "your_password"
db = 0
```

### RPC 配置

```toml
[rpc]
port = 50051
network = "tcp"
registername = "transitData"
```

> RPC 监听地址由程序启动时自动获取本机 IP，无需手动配置。

### HTTP 配置

```toml
[http]
addr = "0.0.0.0:8189"
read_timeout_sec = 5
write_timeout_sec = 30
idle_timeout_sec = 60
max_body_bytes = 1048576
```

> macOS 下 80 端口需要 root 权限，建议开发时使用 8189 等高位端口。

### WebSocket 白名单配置

```toml
[ws]
# 代理请求允许转发的目标 URL（精确匹配 scheme+host+path）
allowed_urls = ["http://localhost/test"]

# 允许连接的 Origin 白名单（为空则允许所有来源）
allowed_origins = []
```

### 限流配置

```toml
[http.ratelimit]
enabled = true
mode = "dry_run"    # dry_run=仅记录日志，enforce=真实拦截返回 429
strategy = "fixed_window"
window_seconds = 1
max_requests = 30
key_by = "ip"
```

## 3. 启动服务

```bash
# 开发环境
go run main.go -env=dev

# 生产环境
go run main.go -env=prod
```

启动后应能看到：
```
httpServer is run
wsServer is run
rpcServer is run on 192.168.x.x:50051
```

### 优雅关闭

服务支持优雅关闭，向进程发送 `SIGINT`（Ctrl+C）或 `SIGTERM`（Kubernetes pod 停止）后：

1. 清理所有在线连接在 Redis 中的会话记录，防止产生幽灵连接
2. HTTP Server 等待进行中的请求完成（最长 30 秒），之后退出

```bash
# 手动发送 SIGTERM
kill -SIGTERM <pid>
```

## 4. 基础自检

```bash
go test ./...
```

无报错即表示代码编译通过。

## 5. WebSocket 消息协议

### 5.1 连接建立

连接 URL 格式：
```
ws://host:port/ws?device_id=<设备号>
```

- `device_id`：必填，标识客户端设备
- 可选：在请求头携带 `Cookie: uid=<用户ID>`，用于标识登录用户

连接成功后服务端立即下发 `wsInit` 消息：
```json
{
  "errcode": 200,
  "wssid": "f3feefd9-9e5d-450f-900a-873b7a736aa6",
  "request_id": "",
  "response_data": "websocket create success",
  "action": "wsInit"
}
```

> `wssid` 是本次连接的唯一标识，后续所有请求都需要携带。

### 5.2 请求消息结构（客户端 → 服务端）

所有请求均为 JSON 格式，公共字段如下：

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `wssid` | string | 是 | 连接建立时服务端下发的唯一标识 |
| `request_id` | string | 是 | 本次请求的唯一 ID，建议使用 UUID，原样回显到响应 |
| `request_type` | string | 是 | 请求类型，见下方说明 |
| `action` | string | 是 | 业务动作标识，原样回显到响应 |
| `request_data` | object | 是 | 请求数据，内容因 `request_type` 而异 |

### 5.3 request_type：req&resp（HTTP 代理）

客户端通过 WebSocket 发起 HTTP 请求，服务端代理转发后将结果推回。

目标 URL 必须在服务端白名单（`config/config_*.toml` 的 `[ws].allowed_urls`）中，仅支持 GET 和 POST 方法。

**request_data 字段：**

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `http_method` | string | 是 | 请求方法，仅支持 `GET` 或 `POST` |
| `request_url` | string | 是 | 目标 URL，GET 请求参数拼在 URL 后面 |
| `post_data` | string | 是 | POST 请求体，GET 时传空字符串 `""` |
| `headers` | object | 是 | 自定义请求头，无则传 `{}` |

**请求示例：**
```json
{
  "wssid": "f3feefd9-9e5d-450f-900a-873b7a736aa6",
  "request_id": "d4f50517-0005-49f1-bd18-85ab24cfe701",
  "request_type": "req&resp",
  "action": "user.showInfo",
  "request_data": {
    "http_method": "POST",
    "request_url": "http://localhost:8189/test?id=11111",
    "post_data": "msg=ddddd&ww=eee",
    "headers": {
      "test": "www"
    }
  }
}
```

### 5.4 request_type：broadcast（推送消息）

向本节点上指定 `wssid` 的连接推送一条消息。

**request_data 字段：**

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `wssid` | string | 是 | 目标连接的 wssid |
| `message` | string | 是 | 要推送的消息内容 |

**请求示例：**
```json
{
  "wssid": "f3feefd9-9e5d-450f-900a-873b7a736aa6",
  "request_id": "e7f2c744-1af1-49fb-b1fc-b859b08d9a26",
  "request_type": "broadcast",
  "action": "push.msg",
  "request_data": {
    "wssid": "f3feefd9-9e5d-450f-900a-873b7a736aa6",
    "message": "hello from broadcast"
  }
}
```

### 5.5 响应消息结构（服务端 → 客户端）

| 字段 | 类型 | 说明 |
|------|------|------|
| `errcode` | int | 状态码，200 为成功 |
| `wssid` | string | 连接标识，原样回显 |
| `request_id` | string | 请求 ID，原样回显 |
| `action` | string | 业务动作，原样回显 |
| `response_data` | string | 响应内容，成功时为业务数据，失败时为错误描述 |

### 5.6 错误码说明

| errcode | 说明 |
|---------|------|
| 200 | 成功 |
| 4201 | 缺少 request_id |
| 4202 | 缺少 wssid |
| 4203 | 缺少 request_type |
| 4204 | 缺少 request_data |
| 4205 | 缺少 action |
| 4101 | http_method 缺失或不支持（仅支持 GET/POST） |
| 4102 | 缺少 request_url |
| 4103 | 缺少 post_data |
| 4104 | 缺少 headers |
| 4004 | 消息格式错误，非合法 JSON |
| 4006 | request_type 不支持 |
| 5001 | 代理请求失败（目标服务返回非 200） |

### 5.7 心跳机制

服务端每 **30 秒**发送一次标准 WebSocket **Ping 帧**，浏览器会自动回复 **Pong 帧**，无需客户端额外处理。若 **60 秒**内未收到 Pong，服务端将主动断开连接。

## 6. WebSocket 测试（cl-test.html）

用浏览器直接打开项目根目录下的 `cl-test.html`，或通过 HTTP 访问（推荐，避免 `file://` 跨域问题）：

```
http://localhost:8189/test-client
```

页面顶部有 **WebSocket 地址输入框**，默认为：
```
ws://localhost:8189/ws?device_id=aaa
```

操作步骤：
1. 点击 **Open** 建立连接，成功后 `wssid` 自动显示在页面上并填入发送框
2. 下拉框选择预设用例（`req&resp` 或 `broadcast`），或手动编辑 JSON
3. 点击 **Send** 发送，蓝色为收到的响应，红色为发出的消息

## 7. HTTP 接口测试

### 7.1 首页

```bash
curl "http://127.0.0.1:8189/"
```
期望返回：`hello word`

### 7.2 测试客户端页面

```bash
# 浏览器访问（避免 file:// 跨域问题，推荐）
open http://127.0.0.1:8189/test-client
```

### 7.3 发送消息到指定 wssid（本节点）

```bash
curl "http://127.0.0.1:8189/sendmsgtowssid?wssid=<wssid>&msg=hello"
```

### 7.4 跨节点推送消息

```bash
curl -X POST "http://127.0.0.1:8189/sendmsg?uid=<uid>&deviceid=<deviceid>" -d "msg=hello"
```

流程：根据 `uid`/`deviceid` 查 Redis 会话 → 找到目标节点 IP → RPC 调用推送。

## 8. 限流验证（可选）

### 观察模式（推荐先做）

配置 `mode = "dry_run"`，快速压测：

```bash
for i in {1..100}; do curl -s "http://127.0.0.1:8189/test?id=$i" -d "msg=x" >/dev/null; done
```

期望：请求仍可通过，日志出现 `rate limit dry-run hit`。

### 强制拦截模式

配置 `mode = "enforce"`，重启服务后重复压测，超限请求返回 `429`，响应体含：
```json
{"errcode": 4290, "response_data": "rate limit exceeded"}
```

## 9. 常见问题排查

- **端口被占用**：修改 `[http].addr` 后重启。
- **连接不上 WebSocket**：检查 `cl-test.html` 地址栏里的端口是否和 `http.addr` 一致。
- **Redis 连接失败**：先用命令行测试连通性，哨兵模式确认 `sentinel_addrs` 和 `master_name` 正确。
- **RPC 异常**：确认 `[rpc].port` 没有被其他进程占用（`lsof -i :50051`）。
- **日志不落文件**：确认 `[base].env = "prod"` 且 `logdir` 目录存在且有写权限。
- **代理请求被拒绝**：确认目标 URL 已加入 `[ws].allowed_urls` 白名单。
- **优雅关闭后仍有残留会话**：确认 Redis 连通且 `DelInfo` 无报错，可查日志中 `Shutdown DelInfo failed` 记录。
