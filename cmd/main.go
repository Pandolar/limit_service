package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Pandolar/limit_service/internal/api"     // 使用新的模块路径
	"github.com/Pandolar/limit_service/internal/auditor" // 使用新的模块路径
	"github.com/Pandolar/limit_service/internal/auth"    // 使用新的模块路径
	"github.com/Pandolar/limit_service/internal/config"  // 使用新的模块路径
	"github.com/Pandolar/limit_service/internal/limiter" // 使用新的模块路径
	"github.com/Pandolar/limit_service/internal/redis"   // 使用新的模块路径
	"github.com/gin-gonic/gin"
)

// extractCookiesMiddleware 从 Cookie 中提取 xtoken 和 xuserid
// 并将它们设置到 Gin 的上下文中。
func extractCookiesMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		xtoken, err := c.Cookie("xtoken")
		if err != nil {
			// 如果 Cookie 不存在或获取出错，设置为空字符串，后续逻辑会处理认证失败
			xtoken = ""
			if !errors.Is(err, http.ErrNoCookie) { // 只记录非“无此cookie”的错误
				log.Printf("提取 xtoken Cookie 时出错: %v", err)
			}
		}
		c.Set("xtoken", xtoken)

		xuserid, err := c.Cookie("xuserid")
		if err != nil {
			xuserid = ""
			if !errors.Is(err, http.ErrNoCookie) {
				log.Printf("提取 xuserid Cookie 时出错: %v", err)
			}
		}
		c.Set("xuserid", xuserid)

		c.Next() // 继续处理请求链中的下一个处理器
	}
}

func main() {
	// 1. 加载配置
	config.LoadConfig() // 调用 config 包的函数
	log.Printf("配置已加载: RedisHost=%s, ServerPort=%d", config.Cfg.RedisHost, config.Cfg.ServerPort)

	// 2. 初始化 Redis 客户端
	if err := redis.InitRedis(config.Cfg); err != nil { // 调用 redis 包的函数
		log.Fatalf("初始化 Redis 客户端失败: %v", err)
	}
	log.Println("Redis 客户端初始化成功。")

	// 3. 初始化审核器 (Aho-Corasick)
	if err := auditor.InitAuditor(); err != nil { // 调用 auditor 包的函数
		// 根据需求，这里可以是 log.Fatalf 使程序退出，或者只是记录错误并允许程序继续运行（审核功能可能降级）
		log.Printf("警告：初始化审核器失败: %v。审核功能可能无法正常工作。", err)
	}

	// 4. 初始化速率限制器 (加载 limit.json)
	if err := limiter.InitRateLimiter(); err != nil { // 调用 limiter 包的函数
		log.Fatalf("初始化速率限制器失败: %v", err)
	}

	// 5. 设置 Gin 路由器
	// gin.SetMode(gin.ReleaseMode) // 在生产环境中取消注释此行
	router := gin.Default() // Default() 包含了 Logger 和 Recovery 中间件

	// 应用自定义中间件
	router.Use(extractCookiesMiddleware())

	// 注册路由
	// api.AuditHandler 和 api.RootHandler 来自导入的 api 包
	router.POST("/audit", api.AuditHandler)
	router.GET("/audit", api.RootHandler) // 根路径下的 /audit GET
	router.GET("/", api.RootHandler)      // 应用的根路径 GET

	// 6. 启动 HTTP 服务器
	serverAddr := fmt.Sprintf("0.0.0.0:%d", config.Cfg.ServerPort)
	srv := &http.Server{
		Addr:         serverAddr,
		Handler:      router,
		ReadTimeout:  10 * time.Second, // 示例：添加读取超时
		WriteTimeout: 10 * time.Second, // 示例：添加写入超时
	}

	log.Printf("服务器正在 %s 上启动", serverAddr)

	// 实现优雅停机
	go func() {
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("监听服务失败: %s\n", err)
		}
	}()

	// 等待中断信号以优雅地关闭服务器
	quit := make(chan os.Signal, 1)
	// signal.Notify 会将指定的信号转发到 quit 通道
	// syscall.SIGINT 对应 Ctrl+C
	// syscall.SIGTERM 是通用的终止信号
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit // 阻塞直到接收到信号
	log.Println("服务器正在关闭...")

	// 创建一个带有超时的上下文，用于服务器关闭
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second) // 5秒关闭超时
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Fatal("服务器强制关闭:", err)
	}

	log.Println("服务器已退出。")
}