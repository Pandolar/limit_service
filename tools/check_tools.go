package tools

import (
	"fmt"
	"strings"

	"github.com/go-redis/redis/v8"
)

// VerifyTokenNoHeader 验证用户token（不使用header）
func VerifyTokenNoHeader(xuserid, xtoken string) (bool, error) {
	expectedToken, err := RedisClient.GetString(fmt.Sprintf("xtoken_%s", xuserid))
	if err != nil {
		if err == redis.Nil {
			return false, nil // token不存在
		}
		return false, err
	}
	
	return xtoken == expectedToken, nil
}

// VerifyUserAcard 校验用户是否可以在指定车提问
// 参数: xuserid - 用户ID, carid - 车ID
// 返回: true表示可以提问
func VerifyUserAcard(xuserid, carid string) (bool, error) {
	// 获取用户激活的套餐信息
	redisUserData, err := RedisClient.Get(fmt.Sprintf("user:%s:active_packages", xuserid))
	if err != nil {
		return false, fmt.Errorf("获取用户套餐信息失败: %w", err)
	}

	var level string
	if redisUserData == nil {
		level = "free"
	} else {
		// 解析用户数据
		userData, ok := redisUserData.(map[string]interface{})
		if !ok {
			return false, fmt.Errorf("用户数据格式错误")
		}
		
		chatgptData, ok := userData["ChatGPT"].(map[string]interface{})
		if !ok {
			level = "free"
		} else {
			levelData, ok := chatgptData["level"].(string)
			if !ok {
				level = "free"
			} else {
				level = strings.ToLower(levelData)
			}
		}
	}

	// 如果不是免费号或基础号，可以使用任何车
	if level != "free" && level != "base" && level != "mini" {
		return true, nil
	}

	// 获取车的状态信息
	redisCarData, err := RedisClient.Get(fmt.Sprintf("car_status:%s", carid))
	if err != nil {
		return false, fmt.Errorf("获取车状态信息失败: %w", err)
	}

	if redisCarData == nil {
		return false, fmt.Errorf("车状态信息不存在")
	}

	carData, ok := redisCarData.(map[string]interface{})
	if !ok {
		return false, fmt.Errorf("车数据格式错误")
	}

	carLabelData, ok := carData["label"].(string)
	if !ok {
		return false, fmt.Errorf("车标签数据格式错误")
	}

	carLabel := strings.ToLower(carLabelData)
	
	// 如果车标签是mini或free，允许访问
	if carLabel == "mini" || carLabel == "free" {
		return true, nil
	}

	return false, nil
} 