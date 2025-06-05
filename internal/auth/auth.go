package auth // 包名为 auth

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"

	"github.com/Pandolar/limit_service/internal/redis" // 使用新的模块路径
)

// VerifyTokenNoHeader 验证提供的 xtoken 是否与 Redis 中为 xuserid 存储的令牌匹配。
// 保留此函数名以与 Python 版本保持一致。
func VerifyTokenNoHeader(ctx context.Context, xuserid string, xtoken string) (bool, error) {
	if xuserid == "" || xtoken == "" {
		log.Println("VerifyTokenNoHeader: xuserid 或 xtoken 为空")
		return false, nil // 或者返回一个错误，指示缺少参数
	}
	key := "xtoken_" + xuserid
	expectedToken, found, err := redis.RC.GetString(ctx, key) // 调用 redis 包的全局客户端 RC
	if err != nil {
		log.Printf("用户 %s 的令牌从 Redis 获取失败: %v", xuserid, err)
		// 返回错误，以便上层可以决定如何响应（例如 500 错误）
		return false, fmt.Errorf("redis_get_token_error: %w", err)
	}
	if !found {
		log.Printf("用户 %s 的令牌在 Redis 中未找到", xuserid)
		return false, nil // 令牌未找到，验证失败
	}
	return xtoken == expectedToken, nil
}

// VerifyUserAcard 验证用户是否允许在特定的 "car" 上发出请求。
func VerifyUserAcard(ctx context.Context, xuserid string, carid string) (bool, error) {
	if xuserid == "" || carid == "" {
		log.Println("VerifyUserAcard: xuserid 或 carid 为空")
		return false, nil // 或者返回一个错误
	}

	// 获取用户的激活套餐信息
	userPackagesKey := "user:" + xuserid + ":active_packages"
	// 假设数据在 Redis 中存储为 JSON 字符串
	userPackagesDataStr, found, err := redis.RC.GetString(ctx, userPackagesKey) // 调用 redis 包
	if err != nil {
		log.Printf("用户 %s 的激活套餐信息获取失败: %v", xuserid, err)
		return false, fmt.Errorf("redis_get_user_packages_error: %w", err)
	}

	userLevel := "free" // 默认为 free 套餐
	if found && userPackagesDataStr != "" {
		var packagesData map[string]interface{}
		// 尝试解析 JSON
		if errUnmarshal := json.Unmarshal([]byte(userPackagesDataStr), &packagesData); errUnmarshal == nil {
			// 安全地提取嵌套值
			if chatGPTData, ok := packagesData["ChatGPT"].(map[string]interface{}); ok {
				if levelVal, ok := chatGPTData["level"].(string); ok {
					userLevel = strings.ToLower(levelVal)
				}
			}
		} else {
			log.Printf("为用户 %s 解析 active_packages JSON 失败: %v。数据: %s", xuserid, errUnmarshal, userPackagesDataStr)
			// 如果解析失败，保持默认等级 'free'
		}
	}

	// 等级不是 "free", "base", "mini" 的用户可以使用任何 car
	if userLevel != "free" && userLevel != "base" && userLevel != "mini" {
		return true, nil
	}

	// 对于 "free", "base", "mini" 用户，检查 car 的标签
	carStatusKey := "car_status:" + carid
	carStatusDataStr, found, err := redis.RC.GetString(ctx, carStatusKey) // 调用 redis 包
	if err != nil {
		log.Printf("car %s 的状态信息获取失败: %v", carid, err)
		return false, fmt.Errorf("redis_get_car_status_error: %w", err)
	}
	if !found || carStatusDataStr == "" {
		log.Printf("car %s 的状态信息未找到", carid)
		return false, nil // car 状态未找到
	}

	var carData map[string]interface{}
	if errUnmarshal := json.Unmarshal([]byte(carStatusDataStr), &carData); errUnmarshal != nil {
		log.Printf("为 car %s 解析 car_status JSON 失败: %v。数据: %s", carid, errUnmarshal, carStatusDataStr)
		return false, nil // 无法确定 car 标签
	}

	carLabelVal, ok := carData["label"].(string)
	if !ok {
		log.Printf("car %s 的 car_status 中未找到标签或标签不是字符串", carid)
		return false, nil
	}
	carLabel := strings.ToLower(carLabelVal)

	if carLabel == "mini" || carLabel == "free" {
		return true, nil
	}

	return false, nil
}