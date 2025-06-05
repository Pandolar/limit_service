# 环境变量配置说明

## 方式一：命令行直接设置（推荐测试用）

```bash
# 基本配置（Redis无密码）
REDIS_HOST=localhost REDIS_PORT=6379 REDIS_PASSWORD="" REDIS_DB=0 go run main.go

# 有密码的Redis
REDIS_HOST=localhost REDIS_PORT=6379 REDIS_PASSWORD="your_password" REDIS_DB=0 go run main.go

# 远程Redis
REDIS_HOST=192.168.1.100 REDIS_PORT=6379 REDIS_PASSWORD="your_password" REDIS_DB=1 go run main.go
```

## 方式二：Shell export（会话级别）

```bash
# 设置环境变量
export REDIS_HOST=localhost
export REDIS_PORT=6379
export REDIS_PASSWORD=""
export REDIS_DB=0

# 启动程序
go run main.go

# 或者编译后运行
go build -o limit_service main.go
./limit_service
```

## 方式三：创建.env文件

1. 创建 `.env` 文件：
```bash
cat > .env << EOF
REDIS_HOST=localhost
REDIS_PORT=6379
REDIS_PASSWORD=
REDIS_DB=0
EOF
```

2. 然后运行程序（需要先安装godotenv支持）

## 方式四：使用Makefile

```bash
# 使用make命令运行（会自动设置环境变量）
make run

# 或者
make dev-setup  # 创建.env文件
make run
```

## 常用Redis配置示例

### 本地Redis（无认证）
```bash
REDIS_HOST=localhost
REDIS_PORT=6379
REDIS_PASSWORD=""
REDIS_DB=0
```

### 本地Redis（有认证）
```bash
REDIS_HOST=localhost
REDIS_PORT=6379
REDIS_PASSWORD="your_redis_password"
REDIS_DB=0
```

### Docker Redis
```bash
REDIS_HOST=127.0.0.1
REDIS_PORT=6379
REDIS_PASSWORD=""
REDIS_DB=0
```

### 云Redis（如阿里云/腾讯云）
```bash
REDIS_HOST=your-redis-host.redis.rds.aliyuncs.com
REDIS_PORT=6379
REDIS_PASSWORD="your_password"
REDIS_DB=0
```

## 测试连接

启动程序后，如果看到以下信息说明Redis连接成功：
```
Redis连接成功
关键词自动机初始化完成
限速配置初始化完成
服务器启动在端口 19892
```

如果连接失败，会显示具体的错误信息。 