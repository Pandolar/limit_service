package api // 包名为 api

import (
	"log"
	"net/http"
	"strings"

	"github.com/Pandolar/limit_service/internal/auditor" // 使用新的模块路径
	"github.com/Pandolar/limit_service/internal/auth"    // 使用新的模块路径
	"github.com/Pandolar/limit_service/internal/limiter" // 使用新的模块路径
	"github.com/gin-gonic/gin"
)

// Message 结构体对应 Python 中的 Pydantic Message 模型
type Message struct {
	Content map[string]interface{} `json:"content"`
}

// AuditRequest 结构体对应 Python 中的 Pydantic AuditRequest 模型
type AuditRequest struct {
	Action   string    `json:"action" binding:"required"` // 添加 binding:"required" 确保字段存在
	Model    string    `json:"model" binding:"required"`
	Messages []Message `json:"messages" binding:"required,dive"` // dive 会对切片内元素也进行校验（如果 Message 有校验标签）
}

// AuditHandler 处理 /audit POST 请求
func AuditHandler(c *gin.Context) {
	// 从中间件获取 xtoken 和 xuserid
	xtokenVal, _ := c.Get("xtoken")
	xuseridVal, _ := c.Get("xuserid")

	xtoken, okToken := xtokenVal.(string)
	xuserid, okUserid := xuseridVal.(string)

	if !okToken || !okUserid || xtoken == "" || xuserid == "" {
		log.Println("AuditHandler: xtoken 或 xuserid 无效或缺失")
		c.JSON(http.StatusUnauthorized, gin.H{"error": "无效的认证凭据 (Invalid credentials)"})
		return
	}

	// 1. 验证令牌
	// 调用 auth 包中的 VerifyTokenNoHeader 函数
	tokenOk, err := auth.VerifyTokenNoHeader(c.Request.Context(), xuserid, xtoken)
	if err != nil {
		log.Printf("用户 %s 的令牌验证过程中发生错误: %v", xuserid, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "令牌验证时发生内部服务器错误"})
		return
	}
	if !tokenOk {
		// Python 版本使用 429，但 401 Unauthorized 或 403 Forbidden 可能更符合语义
		c.JSON(http.StatusUnauthorized, gin.H{"error": "登录信息已过期或无效，请重新登录"})
		return
	}

	var auditReq AuditRequest
	if err := c.ShouldBindJSON(&auditReq); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的请求体: " + err.Error()})
		return
	}

	// 安全地提取 prompt
	var prompt string
	if len(auditReq.Messages) > 0 && auditReq.Messages[0].Content != nil {
		if partsVal, ok := auditReq.Messages[0].Content["parts"]; ok {
			// 假设 "parts" 是一个字符串列表
			if partsList, okList := partsVal.([]interface{}); okList && len(partsList) > 0 {
				if partStr, okStr := partsList[0].(string); okStr {
					prompt = partStr
				}
			}
		}
	}
	// 如果未找到 "parts" 或其内容不符合预期，prompt 将保持为空字符串，与 Python 的默认行为类似

	// 记录请求头信息（与 Python 代码中的 print 对应）
	carID := strings.ReplaceAll(c.GetHeader("carid"), " ", "")
	userTokenHeader := strings.ReplaceAll(strings.TrimPrefix(c.GetHeader("Authorization"), "Bearer "), " ", "")
	// 注意：在生产环境中应谨慎记录敏感信息如 token 和 prompt
	log.Printf("AuditHandler - 请求信息: UserID=%s, CarID=%s, UserTokenHeader (present)=%t, Prompt (前50字符): %.50s",
		xuserid, carID, userTokenHeader != "", prompt)

	// 2. 内容审核
	// 调用 auditor 包中的 StarAudit 函数
	if !auditor.StarAudit(prompt) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请珍惜账号, 不要提问违禁内容."}) // 400 状态码与 Python 一致
		return
	}

	// 3. 速率限制检查
	// 调用 limiter 包中的 GetStarLimit 函数
	isLimitOk, limitMsg, err := limiter.GetStarLimit(c.Request.Context(), xuserid, auditReq.Model)
	if err != nil {
		log.Printf("用户 %s, 模型 %s 的速率限制检查过程中发生错误: %v", xuserid, auditReq.Model, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "速率限制检查时发生内部服务器错误"})
		return
	}

	if !isLimitOk {
		c.JSON(http.StatusTooManyRequests, gin.H{"error": limitMsg}) // 429 状态码与 Python 一致
		return
	}

	// 如果速率限制通过 (isLimitOk 为 true)，则检查用户在此 car 上的权限
	// 这与 Python 的逻辑一致: `if is_ok: ... ret_ = await verify_user_acard`

	// 4. 验证用户在此 "car" 上的权限
	// 调用 auth 包中的 VerifyUserAcard 函数
	canUseCar, err := auth.VerifyUserAcard(c.Request.Context(), xuserid, carID)
	if err != nil {
		log.Printf("用户 %s, car %s 的用户权限验证过程中发生错误: %v", xuserid, carID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "用户权限检查时发生内部服务器错误"})
		return
	}

	if canUseCar {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	} else {
		// Python 版本使用 429，语义上也可以考虑 403 Forbidden
		c.JSON(http.StatusTooManyRequests, gin.H{"error": "请右上角切换线路"})
	}
}

// RootHandler 处理 GET / 和 GET /audit 请求
func RootHandler(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "你好，Star Limit 服务器已就绪 (Go 版本)"})
}