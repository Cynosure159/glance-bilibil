# 第一阶段：构建阶段
FROM golang:1.25.4-alpine AS builder

# 设置工作目录
WORKDIR /app

# 复制源码
COPY . .
RUN go mod download

# 编译二进制文件，启用静态链接以兼容 Alpine
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -installsuffix cgo -o glance-bilibil .

# 第二阶段：运行阶段（基于 Alpine，轻量且可调试）
FROM alpine:3.20

# 安装必要工具
RUN apk --no-cache add ca-certificates && \
    mkdir -p /config

# 复制编译好的二进制文件
COPY --from=builder /app/glance-bilibil /glance-bilibil

# 设置挂载点
VOLUME ["/config"]

# 设置环境变量
ENV CONFIG_PATH=/config/config.json

# 暴露端口 (默认 8082)
EXPOSE 8082

# 设置启动命令
CMD ["/glance-bilibil"]
