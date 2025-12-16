# 阶段 1: 构建
FROM golang:1.24 AS builder
WORKDIR /app
# **** 关键步骤：将源代码复制到镜像中 ****
COPY . .
# 禁用 cgo 以实现完全静态链接
ENV CGO_ENABLED=0
# 编译 Go 应用程序
RUN go build -ldflags="-s -w" -o /app/clash-speedtest .
# 阶段 2: 最终镜像
FROM alpine:3.18
# 安装 ca-certificates 以确保 HTTPS/SSL 正常工作
RUN apk update && apk add --no-cache ca-certificates
# 暴露应用程序使用的端口
EXPOSE 8080
# 从构建阶段复制二进制文件
COPY --from=builder /app/clash-speedtest /clash-speedtest
# 设置启动命令
ENTRYPOINT ["/clash-speedtest", "-web", "-port", "8080"]