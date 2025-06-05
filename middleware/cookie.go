package middleware

import (
	"strings"

	"github.com/gin-gonic/gin"
)

// ExtractCookiesMiddleware 提取cookies中间件
// 对应Python中的extract_cookies_middleware函数
func ExtractCookiesMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 获取cookie字符串
		cookieString := c.GetHeader("cookie")
		
		// 解析cookies
		xtoken := ""
		xuserid := ""
		
		if cookieString != "" {
			// 解析cookie字符串
			cookies := parseCookieString(cookieString)
			xtoken = cookies["xtoken"]
			xuserid = cookies["xuserid"]
		}

		// 将解析的值存储到context中，供后续处理器使用
		c.Set("xtoken", xtoken)
		c.Set("xuserid", xuserid)

		// 继续处理下一个中间件或处理器
		c.Next()
	}
}

// parseCookieString 解析cookie字符串
// 模拟Python中SimpleCookie的行为
func parseCookieString(cookieString string) map[string]string {
	cookies := make(map[string]string)
	
	// 按分号分割cookie
	pairs := strings.Split(cookieString, ";")
	
	for _, pair := range pairs {
		// 去除空格
		pair = strings.TrimSpace(pair)
		
		// 按等号分割键值对
		if idx := strings.Index(pair, "="); idx > 0 {
			key := strings.TrimSpace(pair[:idx])
			value := strings.TrimSpace(pair[idx+1:])
			
			// 去除可能的引号
			if len(value) >= 2 && value[0] == '"' && value[len(value)-1] == '"' {
				value = value[1 : len(value)-1]
			}
			
			cookies[key] = value
		}
	}
	
	return cookies
} 