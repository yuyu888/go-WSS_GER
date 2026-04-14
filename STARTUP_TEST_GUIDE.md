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
rpcServer is run on 192.168.x.x:50051
wsServer is run
```

## 4. 基础自检

```bash
go test ./...
```

无报错即表示代码编译通过。

## 5. WebSocket 测试（cl-test.html）

用浏览器直接打开项目根目录下的 `cl-test.html`。

页面顶部有 **WebSocket 地址输入框**，默认为：
```
ws://localhost:8189/ws?device_id=aaa
```

可直接修改地址后点击 **Open** 建立连接，成功后会看到：
- `OPEN`
- 服务端返回的 `wsInit` 消息（包含 `wssid`）

点击 **Send** 发送默认 JSON 消息，成功时会看到 `RESPONSE` 返回结果。

## 6. HTTP 接口测试

### 6.1 首页

```bash
curl "http://127.0.0.1:8189/"
```
期望返回：`hello word`

### 6.2 发送消息到指定 wssid（本节点）

```bash
curl "http://127.0.0.1:8189/sendmsgtowssid?wssid=<wssid>&msg=hello"
```

### 6.3 跨节点推送消息

```bash
curl -X POST "http://127.0.0.1:8189/sendmsg?uid=<uid>&deviceid=<deviceid>" -d "msg=hello"
```

流程：根据 `uid`/`deviceid` 查 Redis 会话 → 找到目标节点 IP → RPC 调用推送。

## 7. 限流验证（可选）

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

## 8. 常见问题排查

- **端口被占用**：修改 `[http].addr` 后重启。
- **连接不上 WebSocket**：检查 `cl-test.html` 地址栏里的端口是否和 `http.addr` 一致。
- **Redis 连接失败**：先用命令行测试连通性，哨兵模式确认 `sentinel_addrs` 和 `master_name` 正确。
- **RPC 异常**：确认 `[rpc].port` 没有被其他进程占用（`lsof -i :50051`）。
- **日志不落文件**：确认 `[base].env = "prod"` 且 `logdir` 目录存在且有写权限。
