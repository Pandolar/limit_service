package tools

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

// LimitData 限速配置数据结构
type LimitData struct {
	ChatGPT map[string]map[string]string `json:"chatgpt"`
	Other   string                       `json:"other"`
}

// 全局限速数据
var limitData LimitData

// InitStarLimit 初始化限速数据
func InitStarLimit() error {
	// 读取限速配置文件
	file, err := os.Open("./data/limit.json")
	if err != nil {
		return fmt.Errorf("打开限速配置文件失败: %w", err)
	}
	defer file.Close()

	if err := json.NewDecoder(file).Decode(&limitData); err != nil {
		return fmt.Errorf("解析限速配置文件失败: %w", err)
	}

	fmt.Println("限速配置初始化完成")
	return nil
}

// parseLimit 解析限制配置，将 '次数/时间' 解析为计数和时间（秒）
func parseLimit(limitStr string) (int, int, error) {
	parts := strings.Split(limitStr, "/")
	if len(parts) != 2 {
		return 0, 0, fmt.Errorf("限制格式错误: %s", limitStr)
	}

	count, err := strconv.Atoi(parts[0])
	if err != nil {
		return 0, 0, fmt.Errorf("次数解析错误: %w", err)
	}

	duration := parts[1]
	var seconds int

	if strings.HasSuffix(duration, "h") {
		hours, err := strconv.Atoi(strings.TrimSuffix(duration, "h"))
		if err != nil {
			return 0, 0, fmt.Errorf("小时解析错误: %w", err)
		}
		seconds = hours * 3600
	} else if strings.HasSuffix(duration, "d") {
		days, err := strconv.Atoi(strings.TrimSuffix(duration, "d"))
		if err != nil {
			return 0, 0, fmt.Errorf("天数解析错误: %w", err)
		}
		seconds = days * 86400
	} else if strings.HasSuffix(duration, "m") {
		minutes, err := strconv.Atoi(strings.TrimSuffix(duration, "m"))
		if err != nil {
			return 0, 0, fmt.Errorf("分钟解析错误: %w", err)
		}
		seconds = minutes * 60
	} else {
		var err error
		seconds, err = strconv.Atoi(duration)
		if err != nil {
			return 0, 0, fmt.Errorf("秒数解析错误: %w", err)
		}
	}

	return count, seconds, nil
}

// getLimitRules 按顺序查找限制规则
func getLimitRules(packageType, model string) string {
	if chatgptRules, exists := limitData.ChatGPT[packageType]; exists {
		if limit, exists := chatgptRules[model]; exists {
			return limit
		}
		if limit, exists := chatgptRules["other"]; exists {
			return limit
		}
	}
	return limitData.Other
}

// GetStarLimit 检查用户在指定模型下的速率限制，并返回是否允许发送消息
// 参数: xuserid - 用户ID, model - 模型名称
// 返回: (是否允许发送消息, 消息内容, 错误)
func GetStarLimit(xuserid, model string) (bool, string, error) {
	// 获取用户当前激活的套餐信息
	activePackagesKey := fmt.Sprintf("user:%s:active_packages", xuserid)
	activePackagesData, err := RedisClient.Get(activePackagesKey)
	if err != nil {
		return false, "", fmt.Errorf("获取用户套餐信息失败: %w", err)
	}

	packageType := "free" // 默认为免费套餐
	if activePackagesData != nil {
		userData, ok := activePackagesData.(map[string]interface{})
		if ok {
			if chatgptData, exists := userData["ChatGPT"]; exists {
				if chatgptMap, ok := chatgptData.(map[string]interface{}); ok {
					if level, exists := chatgptMap["level"]; exists {
						if levelStr, ok := level.(string); ok {
							packageType = strings.ToLower(levelStr)
						}
					}
				}
			}
		}
	}

	// 获取速率限制规则
	limitStr := getLimitRules(packageType, model)
	if limitStr == "" {
		return false, "未配置速率限制", nil
	}

	// 解析速率限制规则
	maxCount, windowSeconds, err := parseLimit(limitStr)
	if err != nil {
		return false, "", fmt.Errorf("解析限制规则失败: %w", err)
	}

	redisKey := fmt.Sprintf("star_rate_limit:%s:%s:%s", xuserid, packageType, model)
	userPackageKey := fmt.Sprintf("star_rate_limit_package:%s", xuserid)

	// 检查用户当前套餐是否发生变化
	storedPackage, err := RedisClient.GetString(userPackageKey)
	if err != nil && err.Error() != "redis: nil" {
		return false, "", fmt.Errorf("获取存储的套餐信息失败: %w", err)
	}

	if storedPackage != packageType {
		// 如果套餐发生变化，重置计数器并更新套餐信息
		if err := RedisClient.Set(redisKey, 0, time.Duration(windowSeconds)*time.Second); err != nil {
			return false, "", fmt.Errorf("重置计数器失败: %w", err)
		}
		if err := RedisClient.Set(userPackageKey, packageType, 0); err != nil {
			return false, "", fmt.Errorf("更新套餐信息失败: %w", err)
		}
	}

	// 获取当前计数器值
	currentCount, err := RedisClient.GetInt(redisKey)
	if err != nil && err.Error() != "redis: nil" {
		return false, "", fmt.Errorf("获取当前计数失败: %w", err)
	}

	// 检查是否超过速率限制
	if currentCount >= maxCount {
		return false, fmt.Sprintf("超过速率限制：在该%s套餐下，%s模型每%d分钟允许%d条消息，请稍后重试或升级套餐。",
			packageType, model, windowSeconds/60, maxCount), nil
	}

	// 如果没有超过限制，递增计数器
	newCount, err := RedisClient.Incr(redisKey)
	if err != nil {
		return false, "", fmt.Errorf("递增计数器失败: %w", err)
	}

	if newCount == 1 {
		// 如果是第一次递增，设置过期时间
		if err := RedisClient.Expire(redisKey, time.Duration(windowSeconds)*time.Second); err != nil {
			return false, "", fmt.Errorf("设置过期时间失败: %w", err)
		}
	}

	return true, "允许发送消息", nil
} 