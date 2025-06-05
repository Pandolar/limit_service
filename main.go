package main

import (
	"fmt"
	"log"

	"github.com/gin-gonic/gin"
	"limit_service/api"
	"limit_service/middleware"
	"limit_service/tools"
)

func main() {
	// 初始化Redis连接
	if err := tools.InitRedis(); err != nil {
		log.Fatalf("初始化Redis失败: %v", err)
	}

	// 初始化关键词审核
	if err := tools.InitKeyWords(); err != nil {
		log.Fatalf("初始化关键词审核失败: %v", err)
	}

	// 初始化限速配置
	if err := tools.InitStarLimit(); err != nil {
		log.Fatalf("初始化限速配置失败: %v", err)
	}

	// 创建Gin路由器
	// 在生产环境中，可以使用gin.SetMode(gin.ReleaseMode)
	router := gin.Default()

	// 添加cookie提取中间件
	router.Use(middleware.ExtractCookiesMiddleware())

	// 设置路由
	api.SetupAuditRoutes(router)

	// 启动服务器
	fmt.Println("服务器启动在端口 19892")
	if err := router.Run(":19892"); err != nil {
		log.Fatalf("启动服务器失败: %v", err)
	}
} 