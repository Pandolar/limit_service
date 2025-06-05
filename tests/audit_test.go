package tests

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"limit_service/api"
	"limit_service/middleware"
	"limit_service/tools"
)

// MockRedisTool 模拟Redis工具
type MockRedisTool struct{}

func (m *MockRedisTool) GetString(key string) (string, error) {
	// 模拟token验证失败
	return "", nil
}

func (m *MockRedisTool) Get(key string) (interface{}, error) {
	return nil, nil
}

// setupTestRouter 设置测试路由器
func setupTestRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(middleware.ExtractCookiesMiddleware())
	api.SetupAuditRoutes(router)
	return router
}

// TestRootHandler 测试根路径处理器
func TestRootHandler(t *testing.T) {
	router := setupTestRouter()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, 200, w.Code)
	
	var response map[string]string
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "Hello, Star Limt Server Is Ready", response["message"])
}

// TestAuditHandlerInvalidRequest 测试审核处理器 - 无效请求
func TestAuditHandlerInvalidRequest(t *testing.T) {
	// 跳过需要Redis的测试
	if tools.RedisClient == nil {
		t.Skip("跳过需要Redis连接的测试")
	}
	
	router := setupTestRouter()

	// 测试空请求体
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/audit", bytes.NewBufferString("{}"))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, 400, w.Code)
}

// TestStarAudit 测试审核功能
func TestStarAudit(t *testing.T) {
	// 测试安全内容
	result := tools.StarAudit("这是一个正常的问题")
	assert.True(t, result)

	// 测试非字符串输入
	result = tools.StarAudit(123)
	assert.True(t, result)

	// 测试nil输入
	result = tools.StarAudit(nil)
	assert.True(t, result)
}

// TestParseCookieString 测试cookie解析
func TestParseCookieString(t *testing.T) {
	// 注意：这个函数在middleware包中是私有的，这里只是演示如何测试
	// 实际实现中需要将其导出或创建包装函数
	
	router := setupTestRouter()
	
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/", nil)
	req.Header.Set("Cookie", "xtoken=test_token; xuserid=test_user")
	router.ServeHTTP(w, req)

	assert.Equal(t, 200, w.Code)
}

// TestAuditRequestValidation 测试审核请求验证
func TestAuditRequestValidation(t *testing.T) {
	// 跳过需要Redis的测试
	if tools.RedisClient == nil {
		t.Skip("跳过需要Redis连接的测试")
	}
	
	router := setupTestRouter()

	// 创建有效的审核请求
	auditReq := map[string]interface{}{
		"action": "test",
		"model":  "gpt-4",
		"messages": []map[string]interface{}{
			{
				"content": map[string]interface{}{
					"parts": []string{"测试内容"},
				},
			},
		},
	}

	reqBody, _ := json.Marshal(auditReq)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/audit", bytes.NewBuffer(reqBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Cookie", "xtoken=invalid; xuserid=test")
	router.ServeHTTP(w, req)

	// 应该返回token验证失败或其他错误
	assert.NotEqual(t, 200, w.Code)
}

// BenchmarkStarAudit 性能测试 - 审核功能
func BenchmarkStarAudit(b *testing.B) {
	testString := "这是一个用于性能测试的正常字符串，包含一些常见的中文内容"
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		tools.StarAudit(testString)
	}
}

// TestMain 测试主函数，用于设置测试环境
func TestMain(m *testing.M) {
	// 在这里可以初始化测试需要的资源
	// 注意：在实际测试中，您可能需要使用测试专用的Redis实例
	// 或者mock Redis连接
	
	// 初始化关键词（使用测试数据）
	if err := tools.InitKeyWords(); err != nil {
		// 如果初始化失败，可以跳过相关测试或使用mock
		// log.Printf("警告：无法初始化关键词审核: %v", err)
	}

	// 初始化限速配置
	if err := tools.InitStarLimit(); err != nil {
		// 同样，如果初始化失败，可以使用mock
		// log.Printf("警告：无法初始化限速配置: %v", err)
	}

	// 运行测试
	os.Exit(m.Run())
} 