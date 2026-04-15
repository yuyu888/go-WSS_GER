# go-WSS_GER

一个基于 Go + rpcx 实现的分布式 WebSocket 网关服务。

## 功能简介

- 浏览器通过 WebSocket 长连接接入，服务端为每个连接分配唯一 `wssid`
- 支持 `req&resp` 模式：客户端通过 WS 发起 HTTP 代理请求，服务端转发并将结果推回
- 支持 `broadcast` 模式：向本节点其他连接推送消息
- 支持跨节点推送：通过 Redis 查找用户所在节点，RPC 转发消息
- 内置 HTTP 限流（固定窗口，支持 `dry_run` / `enforce` 两种模式）
- 支持 WebSocket Origin 白名单和代理 URL 白名单
- 支持优雅关闭（Graceful Shutdown）：收到 SIGINT/SIGTERM 后清理 Redis 会话，等待进行中的 HTTP 请求完成后退出

## 外部依赖

只需要 **Redis**，支持单机和哨兵模式。

## 快速启动

```bash
go mod tidy
go run main.go -env=dev
```

详细配置和测试说明见 [STARTUP_TEST_GUIDE.md](./STARTUP_TEST_GUIDE.md)。

## 测试数据

WebSocket 消息示例（`req&resp` 模式）：

```json
{
  "wssid": "16b3d4db-4586-4002-8cc8-d5fd0cc877f3",
  "request_id": "d4f50517-0005-49f1-bd18-85ab24cfe701",
  "request_data": {
    "http_method": "POST",
    "request_url": "http://localhost/test?id=11111",
    "post_data": "msg=ddddd&ww=eee",
    "headers": {"test": "www"}
  },
  "request_type": "req&resp",
  "action": "user.showInfo"
}
```

## HTTP 防护配置（限流）

在 `config/config_*.toml` 里配置：

```toml
[http.ratelimit]
enabled = true
mode = "dry_run"       # dry_run=仅记录，enforce=真实拦截返回 429
strategy = "fixed_window"
window_seconds = 1
max_requests = 30
key_by = "ip"
```

建议上线步骤：
1. 先开 `enabled=true + mode=dry_run` 观察日志是否有误伤
2. 根据日志调整 `window_seconds` 与 `max_requests`
3. 再切换到 `mode=enforce` 正式生效

## Redis 配置

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
