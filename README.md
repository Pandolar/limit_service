# Limit Service - Go 审核与限速服务

基于 Go Gin 框架的高性能用户审核与限速服务，提供内容审核、用户权限验证和请求限速功能。

## 项目概述

本项目是一个面向聊天服务的中间件系统，主要功能包括：
- **内容审核**: 使用 AC 自动机算法检测敏感词汇
- **用户验证**: 基于 token 的用户身份验证和权限检查
- **请求限速**: 灵活的用户请求频率控制
- **缓存机制**: Redis 缓存提升性能和数据持久化

## 项目结构详解

```
limit_service/
├── main.go                    # 程序入口点，服务器启动和路由配置
├── go.mod                     # Go 模块依赖管理文件
├── go.sum                     # 依赖版本锁定文件
├── Dockerfile                 # Docker 容器构建配置
├── Makefile                   # 构建和部署脚本
├── README.md                  # 项目说明文档
├── .gitignore                 # Git 版本控制忽略文件
│
├── api/                       # API 路由层
│   └── audit.go              # 审核接口实现，处理 HTTP 请求和响应
│
├── config/                    # 配置管理层
│   └── config.go             # 环境变量配置和系统参数管理
│
├── middleware/                # 中间件层
│   └── cookie.go             # Cookie 解析中间件，提取用户认证信息
│
├── tools/                     # 业务逻辑工具层
│   ├── redis_tools.go        # Redis 缓存操作封装
│   ├── audit_tools.go        # 内容审核核心算法实现
│   ├── check_tools.go        # 用户验证和权限检查
│   └── limit_tools.go        # 请求限速算法实现
│
├── data/                      # 数据文件
│   ├── keywords.txt          # 敏感词黑名单数据库
│   └── limit.json            # 用户限速配置规则
│
├── tests/                     # 测试文件
│   └── audit_test.go         # 单元测试和集成测试
│
└── scripts/                   # 部署脚本
    └── start.sh              # 服务启动脚本
```

## 核心组件原理

### 1. 主程序 (main.go)

**职责**: 服务器初始化、路由配置、中间件注册

**核心逻辑**:
```go
// 服务器启动流程
1. 初始化 Redis 连接池
2. 加载敏感词库到 AC 自动机
3. 加载用户限速配置
4. 注册 Gin 中间件和路由
5. 启动 HTTP 服务器监听
```

**关键特性**:
- 支持优雅关闭 (Graceful Shutdown)
- 配置热重载机制
- 错误恢复和日志记录

### 2. API 层 (api/audit.go)

**职责**: HTTP 请求处理、参数验证、业务逻辑调用

**接口规范**:
```go
// POST /audit - 内容审核接口
type AuditRequest struct {
    Action   string     `json:"action"`   // 操作类型
    Model    string     `json:"model"`    // 模型标识
    Messages []Message  `json:"messages"` // 消息列表
}

type Message struct {
    Content MessageContent `json:"content"`
}

type MessageContent struct {
    Parts []string `json:"parts"` // 用户输入内容
}
```

**处理流程**:
1. 请求参数解析和验证
2. 用户身份认证 (通过 Cookie)
3. 用户权限检查
4. 请求频率限制检查
5. 内容敏感词审核
6. 返回审核结果

### 3. 中间件层 (middleware/cookie.go)

**职责**: HTTP 请求预处理、用户信息提取

**实现原理**:
```go
// Cookie 解析流程
1. 从 HTTP Header 提取 Cookie
2. 解析用户 token 和相关信息
3. 将用户信息注入到 Context 中
4. 传递给下游处理器
```

**支持的认证信息**:
- `token`: 用户身份令牌
- `user_id`: 用户唯一标识
- `session_id`: 会话标识

### 4. Redis 工具 (tools/redis_tools.go)

**职责**: Redis 缓存操作封装、连接池管理

**核心功能**:
```go
// Redis 操作接口
- Get(key string) (string, error)           // 获取缓存值
- Set(key, value string, expiration time.Duration) error // 设置缓存
- Incr(key string) (int64, error)           // 原子计数器
- HGetAll(key string) (map[string]string, error) // 哈希表操作
- Expire(key string, expiration time.Duration) error // 设置过期时间
```

**连接管理**:
- 连接池复用，提高性能
- 自动重连机制
- 连接健康检查
- 支持 Redis Cluster 和哨兵模式

### 5. 审核工具 (tools/audit_tools.go)

**职责**: 内容审核核心算法、敏感词检测

**AC 自动机原理**:
```
AC (Aho-Corasick) 自动机是多模式字符串匹配算法:
1. 构建 Trie 树存储所有敏感词
2. 构建失败函数 (failure function)
3. 单次扫描文本，同时匹配多个模式
4. 时间复杂度: O(n + m + z)
   - n: 文本长度
   - m: 所有模式长度之和  
   - z: 匹配结果数量
```

**实现特性**:
- 大小写不敏感匹配
- 支持中文、英文、数字混合内容
- 内存高效，单次构建多次使用
- 支持敏感词库热更新

**检测流程**:
```go
1. 初始化时加载 keywords.txt 构建 AC 自动机
2. 接收用户输入文本
3. 使用 AC 自动机扫描文本
4. 返回检测到的敏感词位置和类型
```

### 6. 验证工具 (tools/check_tools.go)

**职责**: 用户身份验证、权限检查、业务规则验证

**验证层次**:
```go
// 三层验证机制
1. Token 有效性验证
   - Token 格式检查
   - Token 过期时间验证
   - Token 签名验证

2. 用户权限验证  
   - 用户状态检查 (正常/封禁/限制)
   - 角色权限验证
   - 资源访问权限

3. 业务规则验证
   - 请求参数完整性
   - 业务逻辑约束
   - 数据一致性检查
```

**缓存策略**:
- 用户信息缓存 (TTL: 30分钟)
- 权限信息缓存 (TTL: 10分钟)
- 黑名单缓存 (TTL: 5分钟)

### 7. 限速工具 (tools/limit_tools.go)

**职责**: 请求频率控制、防刷机制

**限速算法**: 滑动窗口 + 令牌桶混合算法

```go
// 滑动窗口算法 (精确控制)
1. 维护用户最近 N 秒的请求时间戳列表
2. 移除超过时间窗口的旧请求记录  
3. 检查当前窗口内请求数量
4. 根据配置决定是否允许请求

// 令牌桶算法 (平滑限流)  
1. 为每个用户维护令牌桶
2. 固定速率向桶中添加令牌
3. 请求消耗令牌，无令牌则拒绝
4. 桶容量限制突发请求
```

**限速配置** (data/limit.json):
```json
{
  "default": {
    "requests_per_minute": 60,    // 每分钟请求数
    "burst_size": 10,             // 突发请求容量
    "window_size": 60             // 滑动窗口大小(秒)
  },
  "vip_users": {
    "requests_per_minute": 300,   // VIP 用户更高限额
    "burst_size": 50,
    "window_size": 60
  }
}
```

**Redis 存储结构**:
```
rate_limit:{user_id}:requests  -> 请求时间戳列表
rate_limit:{user_id}:tokens    -> 令牌桶状态
rate_limit:{user_id}:config    -> 用户限速配置
```

## 数据流架构

```
HTTP Request
     ↓
[Cookie Middleware] ← 解析用户信息
     ↓
[API Handler] ← 参数验证
     ↓
[Check Tools] ← 用户验证 → [Redis Cache]
     ↓
[Limit Tools] ← 频率检查 → [Redis Counter]
     ↓  
[Audit Tools] ← 内容审核 → [AC Automaton]
     ↓
HTTP Response
```

## 性能特性

### 1. 高并发处理
- **Gin 框架**: 基于 Go 原生 net/http，支持数万并发连接
- **Goroutine 池**: 自动管理协程，避免过度创建
- **无状态设计**: 支持水平扩展和负载均衡

### 2. 内存优化
- **AC 自动机**: 一次构建，多次使用，内存占用稳定
- **Redis 连接池**: 复用连接，减少建连开销
- **数据结构优化**: 使用高效的数据结构存储临时数据

### 3. 缓存策略
- **多层缓存**: 内存缓存 + Redis 缓存
- **缓存预热**: 启动时预加载热点数据
- **缓存失效**: 智能缓存更新和失效机制

## 部署架构

### 1. 单机部署
```
[Load Balancer] → [Limit Service] → [Redis]
                       ↓
                  [Log Files]
```

### 2. 集群部署  
```
[Load Balancer] → [Limit Service 1] → [Redis Cluster]
                ↘ [Limit Service 2] ↗
                ↘ [Limit Service N] ↗
                       ↓
                [Centralized Logging]
```

## 环境变量配置

| 变量名 | 默认值 | 说明 |
|--------|--------|------|
| `REDIS_HOST` | `localhost` | Redis 服务器地址 |
| `REDIS_PORT` | `6379` | Redis 服务器端口 |
| `REDIS_PASSWORD` | `` | Redis 认证密码 |
| `REDIS_DB` | `0` | Redis 数据库编号 |
| `GIN_MODE` | `debug` | Gin 运行模式 (debug/release) |
| `SERVER_PORT` | `19892` | HTTP 服务器监听端口 |

## 快速开始

### 1. 本地开发

```bash
# 克隆项目
git clone <repository-url>
cd limit_service

# 安装依赖
go mod download

# 启动 Redis (Docker)
docker run -d --name redis -p 6379:6379 redis:latest

# 运行服务
go run main.go
```

### 2. Docker 部署

```bash
# 构建镜像
docker build -t limit_service:latest .

# 运行服务
docker run -d \
  --name limit_service \
  -p 19892:19892 \
  -e REDIS_HOST=redis \
  -e REDIS_PORT=6379 \
  --link redis:redis \
  limit_service:latest
```

### 3. 使用 Makefile

```bash
# 开发环境运行
make run

# 生产环境构建
make build

# Docker 部署
make docker-build
make docker-run

# 运行测试
make test
```

## API 使用示例

### 内容审核接口

```bash
curl -X POST http://localhost:19892/audit \
  -H "Content-Type: application/json" \
  -H "Cookie: token=your_token_here; user_id=12345" \
  -d '{
    "action": "chat",
    "model": "gpt-3.5-turbo", 
    "messages": [
      {
        "content": {
          "parts": ["这是要审核的内容"]
        }
      }
    ]
  }'
```

### 健康检查

```bash
curl http://localhost:19892/
# 响应: {"message": "Hello, Star Limt Server Is Ready"}
```

## 监控指标

系统提供以下关键指标监控：

1. **请求指标**
   - QPS (每秒请求数)
   - 响应时间分布
   - 错误率统计

2. **业务指标**  
   - 审核通过率
   - 限速触发率
   - 用户验证成功率

3. **系统指标**
   - 内存使用率
   - CPU 使用率  
   - Redis 连接数
   - Goroutine 数量

## 故障排查

### 常见问题

1. **Redis 连接失败**
   ```bash
   # 检查 Redis 服务状态
   redis-cli ping
   
   # 检查网络连通性
   telnet redis_host 6379
   ```

2. **敏感词库加载失败**
   ```bash
   # 检查文件权限
   ls -la data/keywords.txt
   
   # 检查文件编码
   file data/keywords.txt
   ```

3. **性能问题排查**
   ```bash
   # 查看 goroutine 数量
   curl http://localhost:19892/debug/pprof/goroutine
   
   # 内存使用分析
   go tool pprof http://localhost:19892/debug/pprof/heap
   ```

## 扩展开发

### 1. 添加新的审核规则

1. 在 `tools/audit_tools.go` 中扩展 `CheckContent` 函数
2. 添加新的检测逻辑和规则
3. 更新相关测试用例

### 2. 实现新的限速策略

1. 在 `tools/limit_tools.go` 中添加新的限速算法
2. 扩展配置文件格式
3. 更新缓存键名和数据结构

### 3. 添加新的 API 接口

1. 在 `api/` 目录下创建新的处理器文件
2. 在 `main.go` 中注册新路由
3. 添加相应的中间件和验证逻辑 