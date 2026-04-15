# ---- 构建阶段 ----
FROM golang:1.21-alpine AS builder

# 安装编译依赖（rpcx 依赖 gcc）
RUN apk add --no-cache gcc musl-dev

WORKDIR /build

# 先复制依赖文件，利用 Docker 层缓存加速重复构建
COPY go.mod go.sum ./
RUN go mod download

# 复制源码并编译成静态二进制
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o wssgo main.go

# ---- 运行阶段 ----
FROM alpine:3.19

# 时区设置（可按需改为 Asia/Shanghai）
RUN apk add --no-cache tzdata ca-certificates \
    && cp /usr/share/zoneinfo/Asia/Shanghai /etc/localtime \
    && echo "Asia/Shanghai" > /etc/timezone \
    && apk del tzdata

WORKDIR /app

# 从构建阶段复制二进制
COPY --from=builder /build/wssgo .

# 复制配置目录和测试页面
COPY config/ ./config/
COPY cl-test.html .

# 创建日志目录
RUN mkdir -p /tmp/wssgo

# HTTP 端口 / RPC 端口
EXPOSE 8080 50051

# 默认以 prod 环境启动，可通过 ENV 覆盖
ENV APP_ENV=prod

ENTRYPOINT ["sh", "-c", "./wssgo -env=${APP_ENV}"]
