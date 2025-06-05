# Stage 1: 构建 Go 应用程序
FROM golang:1.24.3-alpine AS builder
# 使用 1.24.3 版本的 golang alpine 镜像作为构建环境

LABEL maintainer="Pandolar"
LABEL stage="builder"

WORKDIR /app

# 优化依赖下载：首先只复制 go.mod 和 go.sum
# 这样 Docker 可以缓存依赖层，除非这两个文件发生变化
COPY go.mod go.sum ./
RUN go mod download && go mod verify

# 复制应用程序的其余源代码
COPY . .

# 构建应用程序
# CGO_ENABLED=0 确保静态链接（如果可能），减少对系统库的依赖
# -ldflags="-s -w" 剥离调试信息，减小二进制文件体积
# -o /app/limit-server 指定输出文件名和路径
RUN CGO_ENABLED=0 GOOS=linux go build -v -ldflags="-s -w" -o /app/limit-server ./cmd/server/main.go

# Stage 2: 创建一个精简的生产镜像
FROM alpine:latest
# 使用最新的 alpine 镜像作为运行环境，它非常小

LABEL stage="runtime"

WORKDIR /app

# 从构建阶段 (builder) 复制编译好的二进制文件
COPY --from=builder /app/limit-server /app/limit-server

# 复制应用程序需要的数据文件 (例如配置文件、模板等)
# 确保 data 目录及其内容被复制到镜像的 /app/data/ 路径下
COPY data/ ./data/

# 暴露应用程序运行的端口
EXPOSE 19892

# （可选）为了安全，可以创建一个非 root 用户来运行应用程序
# RUN addgroup -S appgroup && adduser -S appuser -G appgroup
# USER appuser

# 容器启动时执行的命令
CMD ["/app/limit-server"]