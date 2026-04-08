# go-WSS_GER
一个go+etcd+rpcx 实现的websocket的服务

#### 安装
go mod init wssgo

go run main.go

### 测试数据
{"wssid":"16b3d4db-4586-4002-8cc8-d5fd0cc877f3","request_id":"d4f50517-0005-49f1-bd18-85ab24cfe701","request_data":{"http_method":"POST","request_url":"http:\/\/localhost\/test?id=11111","post_data":"msg=ddddd&ww=eee","headers":{"test":"www"}},"request_type":"req&resp","action":"user.showInfo"}

### HTTP防护配置（可配置限流）

项目支持在 `config/config_*.toml` 里配置 HTTP 基础防护与“可灰度限流”策略：

```toml
[http]
addr = "0.0.0.0:80"
read_timeout_sec = 5
write_timeout_sec = 30
idle_timeout_sec = 60
max_body_bytes = 1048576

[http.ratelimit]
enabled = true
mode = "dry_run"
strategy = "fixed_window"
window_seconds = 1
max_requests = 30
key_by = "ip"
```

参数说明：

- `http.max_body_bytes`: 单次请求体大小上限（字节），超限会被 HTTP 层拒绝。
- `http.ratelimit.enabled`: 限流总开关。
- `http.ratelimit.mode`:
  - `dry_run`: 只记录命中日志，不真正拦截（推荐先用这个模式观察）。
  - `enforce`: 命中后返回 `429` 和业务码 `4290`。
- `http.ratelimit.strategy`: 目前支持 `fixed_window`（固定时间窗口计数）。
- `http.ratelimit.window_seconds`: 窗口大小（秒）。
- `http.ratelimit.max_requests`: 每个 key 在窗口内可通过的最大请求数。
- `http.ratelimit.key_by`: 目前支持 `ip`。

建议上线步骤：

1. 先开 `enabled=true + mode=dry_run` 观察日志是否有误伤。
2. 根据日志调整 `window_seconds` 与 `max_requests`。
3. 再切换到 `mode=enforce` 正式生效。

### 感言
当初做这个项目的时候，go基本零基础；也是借鉴了一些网上的架构思路觉得不错，就勇敢的挑战了一下；用现在的眼光看当时的实现， 或有种不忍直视的感觉，很多实现好幼幼，不过设计思想还是严格的实现了，心理也暗自骄傲，后期的大量项目未必有这个项目架构设计，挑战性也没这个大（干的时候也只有个方向，什么基础都没有，一点成竹在胸的感觉都没有，全都要探索);把这个项目做完，收获也是丰厚的，对于go语言的感觉豁然开朗，算是完成了入门，什么事情都是要多练，敢干，才能更好的领悟；
