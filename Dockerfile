# 使用官方Go镜像作为构建环境
FROM golang:1.21-alpine AS builder

# 设置工作目录
WORKDIR /app

# 安装必要的包
RUN apk add --no-cache git

# 复制go mod文件
COPY go.mod go.sum ./

# 下载依赖
RUN go mod download

# 复制源代码
COPY . .

# 构建应用
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o main .

# 使用轻量级的alpine镜像作为运行环境
FROM alpine:latest

# 设置工作目录
WORKDIR /root/

# 从构建阶段复制可执行文件
COPY --from=builder /app/main .

# 复制数据文件
COPY --from=builder /app/data ./data

# 暴露端口
EXPOSE 19892

# 运行应用
CMD ["./main"] 