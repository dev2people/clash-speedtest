# 阶段 1: 构建
FROM golang:1.24 AS builder

WORKDIR /app

# 配置 Go 国内代理以及禁用 cgo
ENV CGO_ENABLED=0 \
    GOPROXY=https://goproxy.cn,direct

# 优先复制 go.mod 与 go.sum 进行依赖下载，充分利用 Docker 缓存
COPY go.mod go.sum ./
RUN go mod download

# 复制源代码并编译
COPY . .
RUN go build -buildvcs=false -ldflags="-s -w" -o /app/clash-speedtest .

# 阶段 2: 最终镜像
FROM alpine:latest

# 从构建镜像中复制 CA 根证书（无需在运行时联网 apk add）
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

# 暴露应用程序使用的端口
EXPOSE 8080

# 从构建阶段复制二进制文件
COPY --from=builder /app/clash-speedtest /clash-speedtest

# 设置启动命令
ENTRYPOINT ["/clash-speedtest", "-web", "-port", "8080"]