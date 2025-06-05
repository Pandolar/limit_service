#!/bin/bash

# Limit Service 启动脚本

set -e

echo "================================"
echo "启动 Limit Service Go版本"
echo "================================"

# 检查Go环境
if ! command -v go &> /dev/null; then
    echo "错误: 未找到Go环境，请先安装Go 1.21+"
    exit 1
fi

# 检查Go版本
GO_VERSION=$(go version | awk '{print $3}' | sed 's/go//')
echo "当前Go版本: $GO_VERSION"

# 检查依赖文件
if [ ! -f "go.mod" ]; then
    echo "错误: 未找到go.mod文件，请确保在项目根目录运行此脚本"
    exit 1
fi

# 检查数据文件
if [ ! -d "data" ]; then
    echo "警告: 未找到data目录，创建中..."
    mkdir -p data
fi

if [ ! -f "data/keywords.txt" ]; then
    echo "警告: 未找到keywords.txt，创建默认文件..."
    echo "测试黑名单a" > data/keywords.txt
fi

if [ ! -f "data/limit.json" ]; then
    echo "警告: 未找到limit.json，创建默认文件..."
    cat > data/limit.json << 'EOF'
{
  "chatgpt": {
    "free": {
      "other": "5/1h"
    },
    "base": {
      "auto": "50/3h",
      "text-davinci-002-render-sha": "100/3h",
      "gpt-4o-mini": "100/3h",
      "gpt-4o": "15/3h",
      "gpt-4": "10/3h",
      "gpt-4o-canmore": "10/3h",
      "o1-preview": "2/168h",
      "o1-mini": "5/168h"
    },
    "pro": {
      "auto": "1000/3h",
      "text-davinci-002-render-sha": "1000/3h",
      "gpt-4o-mini": "300/3h",
      "gpt-4o": "60/3h",
      "gpt-4": "30/3h",
      "gpt-4o-canmore": "30/3h",
      "o1-preview": "30/168h",
      "o1-mini": "100/168h"
    }
  },
  "other": "40/3h"
}
EOF
fi

# 设置环境变量
export REDIS_HOST=${REDIS_HOST:-"localhost"}
export REDIS_PORT=${REDIS_PORT:-"6379"}
export REDIS_PASSWORD=${REDIS_PASSWORD:-""}
export REDIS_DB=${REDIS_DB:-"0"}

echo "环境变量设置:"
echo "  REDIS_HOST: $REDIS_HOST"
echo "  REDIS_PORT: $REDIS_PORT"
echo "  REDIS_DB: $REDIS_DB"

# 下载依赖
echo "下载Go依赖..."
go mod download
go mod tidy

echo "================================"
echo "启动服务器..."
echo "服务地址: http://localhost:19892"
echo "按 Ctrl+C 停止服务"
echo "================================"

# 启动服务
go run main.go 