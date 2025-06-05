package api

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"limit_service/tools"
)

// Message 消息结构体
type Message struct {
	Content map[string]interface{} `json:"content" binding:"required"`
}

// AuditRequest 审核请求结构体
type AuditRequest struct {
	Action   string    `json:"action" binding:"required"`
	Model    string    `json:"model" binding:"required"`
	Messages []Message `json:"messages" binding:"required"`
}

// AuditResponse 审核响应结构体
type AuditResponse struct {
	Status string `json:"status,omitempty"`
	Error  string `json:"error,omitempty"`
}

// HelloResponse 欢迎响应结构体
type HelloResponse struct {
	Message string `json:"message"`
}

// SetupAuditRoutes 设置审核相关路由
func SetupAuditRoutes(router *gin.Engine) {
	// POST /audit 审核接口
	router.POST("/audit", auditHandler)
	
	// GET / 和 GET /audit 根路径和审核路径
	router.GET("/", rootHandler)
	router.GET("/audit", rootHandler)
}

// auditHandler 处理审核请求
func auditHandler(c *gin.Context) {
	// 从中间件获取token和用户ID
	xtoken, exists := c.Get("xtoken")
	if !exists {
		c.JSON(http.StatusUnauthorized, AuditResponse{Error: "缺少xtoken"})
		return
	}

	xuserid, exists := c.Get("xuserid")
	if !exists {
		c.JSON(http.StatusUnauthorized, AuditResponse{Error: "缺少xuserid"})
		return
	}

	xtokenStr, ok := xtoken.(string)
	if !ok {
		c.JSON(http.StatusUnauthorized, AuditResponse{Error: "xtoken格式错误"})
		return
	}

	xuseridStr, ok := xuserid.(string)
	if !ok {
		c.JSON(http.StatusUnauthorized, AuditResponse{Error: "xuserid格式错误"})
		return
	}

	// 验证token
	isValid, err := tools.VerifyTokenNoHeader(xuseridStr, xtokenStr)
	if err != nil {
		c.JSON(http.StatusInternalServerError, AuditResponse{Error: "验证token失败"})
		return
	}
	if !isValid {
		c.JSON(http.StatusTooManyRequests, AuditResponse{Error: "登录信息已过期，请重新登录"})
		return
	}

	// 解析请求体
	var auditRequest AuditRequest
	if err := c.ShouldBindJSON(&auditRequest); err != nil {
		c.JSON(http.StatusBadRequest, AuditResponse{Error: "请求参数错误: " + err.Error()})
		return
	}

	model := auditRequest.Model
	
	// 获取prompt
	var prompt string
	if len(auditRequest.Messages) > 0 {
		if parts, exists := auditRequest.Messages[0].Content["parts"]; exists {
			if partsSlice, ok := parts.([]interface{}); ok && len(partsSlice) > 0 {
				if promptStr, ok := partsSlice[0].(string); ok {
					prompt = promptStr
				}
			}
		}
	}

	// 获取header信息
	carid := strings.ReplaceAll(c.GetHeader("carid"), " ", "")
	usertoken := strings.TrimPrefix(c.GetHeader("Authorization"), "Bearer ")

	fmt.Printf("carid: %s\n", carid)
	fmt.Printf("usertoken: %s\n", usertoken)
	fmt.Printf("prompt: %s\n", prompt)

	// 内容审核
	if !tools.StarAudit(prompt) {
		c.JSON(http.StatusBadRequest, AuditResponse{Error: "请珍惜账号, 不要提问违禁内容."})
		return
	}

	// 检查速率限制
	isOk, limitMsg, err := tools.GetStarLimit(xuseridStr, model)
	if err != nil {
		c.JSON(http.StatusInternalServerError, AuditResponse{Error: "检查速率限制失败: " + err.Error()})
		return
	}

	if isOk {
		// 校验用户权限是否能在该车提问（就算没过限速也要先看看能不能提问）
		canUse, err := tools.VerifyUserAcard(xuseridStr, carid)
		if err != nil {
			c.JSON(http.StatusInternalServerError, AuditResponse{Error: "验证用户权限失败: " + err.Error()})
			return
		}

		if canUse {
			c.JSON(http.StatusOK, AuditResponse{Status: "ok"})
		} else {
			c.JSON(http.StatusTooManyRequests, AuditResponse{Error: "请右上角切换线路"})
		}
	} else {
		c.JSON(http.StatusTooManyRequests, AuditResponse{Error: limitMsg})
	}
}

// rootHandler 处理根路径请求
func rootHandler(c *gin.Context) {
	c.JSON(http.StatusOK, HelloResponse{Message: "Hello, Star Limt Server Is Ready"})
} 