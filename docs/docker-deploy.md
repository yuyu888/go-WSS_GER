# Docker 部署说明

## 文件说明

```
Dockerfile       多阶段构建，最终镜像基于 alpine，体积约 20MB
.dockerignore    排除 .git、docs、*.md 等无关文件，加快构建
```

---

## 快速开始

### 1. 准备配置文件

镜像内使用 `config/config_prod.toml`（默认）或 `config/config_dev.toml`。

在正式构建前，先修改好配置文件中的 Redis 地址、密码等生产参数：

```toml
# config/config_prod.toml 关键项
[redis]
mode = "sentinel"
sentinel_addrs = ["10.x.x.1:26379", "10.x.x.2:26379", "10.x.x.3:26379"]
master_name = "mymaster"
password = "your_password"

[http]
addr = "0.0.0.0:8080"   # 容器内监听端口，与 Dockerfile EXPOSE 保持一致
```

> **注意**：配置文件连同源码一起打包进镜像。如果不希望密码出现在镜像层中，  
> 请参考下方「配置文件挂载」方案。

---

### 2. 构建镜像

```bash
# 在项目根目录执行
docker build -t wssgo:latest .

# 指定版本 tag
docker build -t wssgo:v1.0.0 .
```

---

### 3. 运行容器

#### 基础运行（使用镜像内配置）

```bash
docker run -d \
  --name wssgo \
  -p 8080:8080 \
  -p 50051:50051 \
  wssgo:latest
```

#### 挂载外部配置文件（推荐生产使用，避免密码打入镜像）

```bash
# 将宿主机的 config/ 目录挂载到容器内
docker run -d \
  --name wssgo \
  -p 8080:8080 \
  -p 50051:50051 \
  -v /your/host/config:/app/config:ro \
  wssgo:latest
```

#### 切换到 dev 环境

```bash
docker run -d \
  --name wssgo-dev \
  -p 8189:8189 \
  -p 50051:50051 \
  -e APP_ENV=dev \
  wssgo:latest
```

#### 挂载日志目录（持久化日志）

```bash
docker run -d \
  --name wssgo \
  -p 8080:8080 \
  -p 50051:50051 \
  -v /var/log/wssgo:/tmp/wssgo \
  wssgo:latest
```

---

## 端口说明

| 端口 | 用途 |
|------|------|
| `8080` | HTTP / WebSocket 服务（对外） |
| `50051` | RPC 服务（仅节点间内部通信，**不要对外暴露**） |

> RPC 端口在多节点部署时需要节点间互通，但不应暴露到公网。  
> K8s 部署时建议通过 ClusterIP Service 打通节点间 RPC，仅对外暴露 HTTP 端口。

---

## 多节点 K8s 部署要点

本服务支持多节点水平扩展，节点间通过 Redis + RPC 路由消息。K8s 部署时注意：

1. **不要使用 ClusterIP 作为 RPC 地址**：服务启动时会自动获取 Pod IP 注册到 Redis，RPC 直接通过 Pod IP 互通，确保 Pod 之间网络可达（同一命名空间通常默认可达）。

2. **HTTP 对外只需一个 Service**：

```yaml
# 对外暴露 HTTP/WS 端口
apiVersion: v1
kind: Service
metadata:
  name: wssgo
spec:
  selector:
    app: wssgo
  ports:
    - name: http
      port: 80
      targetPort: 8080
  type: LoadBalancer   # 或 NodePort、Ingress
```

3. **RPC 端口无需创建 Service**：Pod 之间直接通过 Pod IP 访问 50051，不需要走 Service。

4. **配置文件建议用 ConfigMap 挂载**：

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: wssgo-config
data:
  config_prod.toml: |
    [redis]
    mode = "sentinel"
    ...
---
# 在 Deployment 中挂载
volumes:
  - name: config
    configMap:
      name: wssgo-config
containers:
  - name: wssgo
    volumeMounts:
      - name: config
        mountPath: /app/config
```

---

## 常用命令

```bash
# 查看运行日志
docker logs -f wssgo

# 进入容器排查
docker exec -it wssgo sh

# 停止并删除容器
docker stop wssgo && docker rm wssgo

# 验证服务
curl http://localhost:8080/
# 期望返回：hello word

# WebSocket 测试页面
open http://localhost:8080/test-client
```

---

## 健康检查（可选）

可在 `docker run` 或 K8s 的 `livenessProbe` 中使用 HTTP 健康检查：

```bash
# Docker
docker run -d \
  --health-cmd="wget -qO- http://localhost:8080/ || exit 1" \
  --health-interval=15s \
  --health-timeout=3s \
  --health-retries=3 \
  --name wssgo \
  -p 8080:8080 -p 50051:50051 \
  wssgo:latest
```

```yaml
# K8s livenessProbe
livenessProbe:
  httpGet:
    path: /
    port: 8080
  initialDelaySeconds: 10
  periodSeconds: 15
```
