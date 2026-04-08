# go-WSS_GER 启动与测试说明

## 1. 环境准备

- 安装 Go（建议与 `go.mod` 兼容的版本，至少可执行 `go mod tidy` 和 `go test`）。
- 确认可访问以下依赖服务：
  - Redis（见 `config/config_*.toml` 的 `[redis]`）
  - Etcd（见 `config/config_*.toml` 的 `[etcd]`）
- 首次拉取代码后执行：

```bash
go mod tidy
```

## 2. 配置文件说明

项目按环境读取配置：

- 开发环境：`config/config_dev.toml`
- 生产环境：`config/config_prod.toml`

关键配置建议先确认：

- `[http]`
  - `addr`：HTTP 监听地址（默认 `0.0.0.0:80`）
- `[rpc]`
  - `addr`：只填写 IP/主机名（不带端口）
  - `port`：RPC 端口
- `[http.ratelimit]`
  - `enabled`：是否启用限流
  - `mode`：`dry_run`（仅记录）或 `enforce`（真实拦截）

## 3. 启动服务

### 3.1 开发环境启动

```bash
go run main.go -env=dev
```

### 3.2 生产环境启动

```bash
go run main.go -env=prod
```

启动后应能看到类似日志：

- `wsServer is run`
- `httpServer is run`

## 4. 基础自检

在项目根目录执行：

```bash
go test ./...
```

如果输出均为 `[no test files]` 且无报错，表示代码已成功编译通过。

## 5. WebSocket 测试（cl-test.html）

### 5.1 打开测试页

- 直接用浏览器打开项目根目录下的 `cl-test.html`。

### 5.2 建立连接

- 点击 `Open` 按钮。
- 页面会连接：`ws://localhost/ws?device_id=aaa`。
- 成功后会看到 `OPEN` 以及服务端 `wsInit` 返回消息。

> 如果你的 `http.addr` 不是 `0.0.0.0:80`（例如 `:8080`），请把 `cl-test.html` 里的地址改为 `ws://localhost:8080/ws?device_id=aaa`。

### 5.3 发送请求

- 文本框中默认有一段 JSON，可直接点击 `Send`。
- 成功时会看到 `RESPONSE` 返回结果。

## 6. HTTP 接口测试

### 6.1 首页

```bash
curl "http://127.0.0.1/"
```

期望返回：`hello word`

### 6.2 发送消息到指定 wssid

```bash
curl "http://127.0.0.1/sendmsgtowssid?wssid=<你的wssid>&msg=hello"
```

### 6.3 转发消息接口

```bash
curl -X POST "http://127.0.0.1/sendmsg?uid=<uid>&deviceid=<deviceid>" -d "msg=hello"
```

## 7. 限流验证（可选）

### 7.1 观察模式（推荐先做）

配置：

- `http.ratelimit.enabled = true`
- `http.ratelimit.mode = "dry_run"`

然后快速压测：

```bash
for i in {1..100}; do curl -s "http://127.0.0.1/test?id=$i" -d "msg=x" >/dev/null; done
```

期望：请求仍可通过，但日志出现 `rate limit dry-run hit`。

### 7.2 强制模式

配置：

- `http.ratelimit.enabled = true`
- `http.ratelimit.mode = "enforce"`

重启服务后重复压测，请求超限时会返回 `429`，响应体含：

- `errcode: 4290`
- `response_data: "rate limit exceeded"`

## 8. 常见问题排查

- 端口被占用：修改 `[http].addr` 后重启。
- 连接不上 WebSocket：检查 `cl-test.html` 里的 ws 地址端口是否和 `http.addr` 一致。
- RPC 异常：确认 `[rpc].addr` 为纯 IP/主机名，端口填写在 `[rpc].port`。
- Redis/Etcd 连接失败：先用命令行连通性测试对应地址。

