# CLAUDE.md

本文件为 Claude Code（claude.ai/code）在此代码库中工作时提供指导。

## 常用命令

```bash
# 开发环境启动
go run main.go -env=dev

# 生产环境启动
go run main.go -env=prod

# 编译
go build -o wssgo main.go

# 编译检查 / 运行测试
go test ./...

# 同步依赖
go mod tidy
```

模块名为 `wssgo`（定义在 `go.mod`）。配置文件通过 `-env` 参数选择：`config/config_dev.toml` 或 `config/config_prod.toml`。

## 外部依赖

运行只需要 **Redis**（支持单机和哨兵模式），无其他外部组件依赖。

## 架构概览

这是一个分布式 WebSocket 网关，负责代理浏览器客户端发出的 HTTP 请求，并在多节点间路由服务端推送消息。

### 启动流程（`main.go`）

1. 按 `-env` 加载配置 → `config.InitServiceConfig`
2. 初始化日志 → `libs.InitLogger`
3. 获取本机 IP → 存入 `config.ServiceConf.LocalIp`（同时作为 RPC 监听地址）
4. 启动 RPC 服务（goroutine）→ `wsServer.InitRpcServer()`
5. 注册 WebSocket HTTP 处理器 → `wsServer.Init()`（初始化 URL/Origin 白名单，挂载 `/ws`）
6. 启动 HTTP 服务 → `httpServer.Init()`（创建独立 `ServeMux`，包裹中间件，调用 `ListenAndServe`）

### WebSocket 层（`wsServer/`）

每个连接用 `wsConnection` 结构体表示（`websocket.go`），启动三个并发 goroutine：
- `wsReadLoop` — 读取原始帧，放入 `inChan`；每次读前刷新读超时（150 秒）
- `wsWriteLoop` — 消费 `outChan`，写入帧
- `procLoop` — 从 `inChan` 读消息，用信号量（并发上限 2）派发 goroutine 调用 `process()`；连接空闲超 100 秒发送心跳

连接建立时（`wsHandler`），`device_id` 查询参数为必填；可选 `uid` Cookie 设置登录用户。服务端生成 UUID 作为 `wssid`，并在第一条消息中返回给客户端。

`WsManager`（`wsManager.go`）是单例 `ClientManager`，通过 `register`/`unregister` channel 和 `ProcLoop` goroutine 维护活跃连接池（`WsClientPools`）。会话写入 Redis 失败时会记录日志并主动断开连接，避免产生幽灵连接。

### 消息处理（`wsServer/task.go`）

WS 消息必须为 JSON，包含以下字段：
- `wssid`、`request_id`、`action` — 原样回显到响应
- `request_type` — `"req&resp"` 或 `"broadcast"`
- `request_data` — 各类型专属载荷

**`req&resp`**：校验 `http_method`（仅允许 GET/POST）、`request_url`、`post_data`、`headers` → 检查 `request_url` 是否在 URL 白名单（`urlWhiteConf.go`，从配置文件加载）→ 通过 `libs/curl` 代理请求 → 将结果响应回原 WS 连接。

**`broadcast`**：调用 `WsManager.DoSendMsgToWssid(wssid, message)`，向本节点的另一个连接推送消息。

### 跨节点推送流程

后端服务向可能在任意节点的用户推送消息时：

1. HTTP `POST /sendmsg?uid=&deviceid=` → `httpServer/index.go`
2. 查询 `uid`/`deviceId` → `model.UserSession.GetInfo()` → Redis hash `ws_go_{uid}` → 返回 `{deviceId: {ws_server_addr, wssid}}`
3. 按 `ws_server_addr` 获取 RPC 客户端 → `httpServer/rpcClient.go:GetRpcClient()`（按需建连，用 `sync.Map` 缓存，无需 etcd）
4. 调用 `Dispatch` RPC → 目标节点 `wsServer/rpcServer.go:TransitData.Dispatch()`
5. 目标节点调用 `DoSendMsgToWssid` → 发送到本地 `wsConnection`

### 会话存储（`model/userSession.go`）

Redis hash，key 为 `ws_go_{uid}`（无 uid 时用 `ws_go_{deviceId}`），field 为 `deviceId`，value 为 JSON `{"ws_server_addr": "...", "wssid": "..."}`。TTL 60 秒，连接建立时刷新。Redis 客户端全局单例，通过 `libs.DefaultRedis()` 获取。

### HTTP 中间件（`httpServer/middleware.go`）

`httpServer.Init()` 中叠加两层中间件：
- `wrapWithBodyLimit` — 通过 `http.MaxBytesReader` 限制请求体大小
- `wrapWithRateLimit` — 基于 IP 的固定窗口限流；支持 `dry_run`（仅记日志）和 `enforce`（返回 429 / errcode 4290）两种模式；内置后台 goroutine 定期清理过期条目

### 配置结构（`config/serviceConf.go`）

全局单例 `config.ServiceConf`，主要节：`[redis]`、`[rpc]`、`[http]`、`[http.ratelimit]`、`[ws]`、`[base]`。

**Redis** 支持两种模式，通过 `[redis].mode` 切换：
- `standalone`：填写 `addr`
- `sentinel`：填写 `sentinel_addrs`（列表）和 `master_name`

**WebSocket** 白名单通过 `[ws]` 配置：
- `allowed_urls`：代理请求的目标 URL 白名单
- `allowed_origins`：允许连接的 Origin 白名单，为空则允许所有来源

### libs/

- `libs/logger.go` — 全局 `libs.Logger`（zap，生产环境写文件，开发环境输出 stdout）
- `libs/redis.go` — `go-redis` 的轻量封装，支持单机/哨兵模式，全局单例
- `libs/curl/` — `task.go` 用于代理请求的 HTTP 客户端，复用全局 Transport
- `libs/tool.go` — `GetLocalIp()`、`MapInterfaceToMapString()`

## WebSocket 测试客户端

项目根目录下的 `cl-test.html`，用浏览器直接打开。页面上有地址输入框，默认为 `ws://localhost:8189/ws?device_id=aaa`，可直接修改地址后点击 Open 连接。

## 开发规范

修改任何 Go 文件后，必须运行 `go build ./...` 验证编译通过，再报告任务完成。

## 沟通方式

用户使用中文提问时，直接用中文回答具体问题，不要反问或要求澄清。

## 故障排查

遇到配置文件或 JSON 格式问题时，先读取文件内容定位语法错误，直接修复，不要猜测是 API key 或其他无关原因。
