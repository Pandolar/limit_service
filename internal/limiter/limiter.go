package limiter // 包名为 limiter

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/Pandolar/limit_service/internal/redis" // 使用新的模块路径
)

// LimitData 存储从 limit.json 解析的速率限制规则
type LimitData struct {
	Chatgpt map[string]map[string]string `json:"chatgpt"`
	Other   string                       `json:"other"`
}

// limitRules 是全局的限速规则实例
var limitRules LimitData

const limitFilePath = "./data/limit.json" // 限速规则文件路径

// InitRateLimiter 从 limit.json 加载限速规则
func InitRateLimiter() error {
	jsonFile, err := os.Open(limitFilePath)
	if err != nil {
		return fmt.Errorf("打开 limit.json 失败: %w", err)
	}
	defer jsonFile.Close()

	byteValue, err := ioutil.ReadAll(jsonFile)
	if err != nil {
		return fmt.Errorf("读取 limit.json 内容失败: %w", err)
	}

	if err := json.Unmarshal(byteValue, &limitRules); err != nil {
		return fmt.Errorf("解析 limit.json 失败: %w", err)
	}
	log.Println("已从 limit.json 加载速率限制规则。")
	return nil
}

// parseLimit 将 "次数/时长" 格式的字符串解析为次数和时间窗口（秒）。
// 例如："5/1h" -> (5, 3600, nil)
func parseLimit(limitStr string) (count int, windowSeconds int, err error) {
	parts := strings.Split(limitStr, "/")
	if len(parts) != 2 {
		return 0, 0, fmt.Errorf("无效的限制格式: %s", limitStr)
	}

	count, err = strconv.Atoi(parts[0])
	if err != nil {
		return 0, 0, fmt.Errorf("限制规则中的次数无效: %s", parts[0])
	}

	durationStr := parts[1]
	var value int
	var unit string

	// 尝试提取数字和单位
	if strings.HasSuffix(durationStr, "h") {
		unit = "h"
		value, err = strconv.Atoi(strings.TrimSuffix(durationStr, "h"))
	} else if strings.HasSuffix(durationStr, "d") {
		unit = "d"
		value, err = strconv.Atoi(strings.TrimSuffix(durationStr, "d"))
	} else if strings.HasSuffix(durationStr, "m") {
		unit = "m"
		value, err = strconv.Atoi(strings.TrimSuffix(durationStr, "m"))
	} else {
		// 如果没有单位后缀，则假定是秒
		value, err = strconv.Atoi(durationStr)
		unit = "s" // 假设是秒
	}

	if err != nil {
		return 0, 0, fmt.Errorf("限制规则中的时长无法解析基础数值: %s, 错误: %v", durationStr, err)
	}

	switch unit {
	case "h":
		windowSeconds = value * 3600
	case "d":
		windowSeconds = value * 86400
	case "m":
		windowSeconds = value * 60
	case "s":
		windowSeconds = value
	default: // 理论上不会到这里，因为前面已经处理了
		return 0, 0, fmt.Errorf("未知的时长单位: %s", unit)
	}

	return count, windowSeconds, nil
}

// getLimitRule 根据套餐名称和模型名称获取限速规则字符串。
func getLimitRule(packageName, modelName string) string {
	if packageRules, ok := limitRules.Chatgpt[packageName]; ok {
		if limit, ok := packageRules[modelName]; ok {
			return limit
		}
		if otherLimit, ok := packageRules["other"]; ok { // 备用规则："other"
			return otherLimit
		}
	}
	return limitRules.Other // 全局备用规则
}

// GetStarLimit 检查用户在指定模型下的速率限制。
// 返回 (是否允许发送消息 bool, 提示消息 string, 错误 error)
func GetStarLimit(ctx context.Context, xuserid string, model string) (bool, string, error) {
	// 获取用户的当前激活套餐信息
	activePackagesKey := "user:" + xuserid + ":active_packages"
	activePackagesDataStr, found, err := redis.RC.GetString(ctx, activePackagesKey) // 调用 redis 包
	if err != nil {
		return false, "", fmt.Errorf("为用户 %s 获取激活套餐失败: %w", xuserid, err)
	}

	packageName := "free" // 默认套餐
	if found && activePackagesDataStr != "" {
		var packagesData map[string]interface{}
		if errJson := json.Unmarshal([]byte(activePackagesDataStr), &packagesData); errJson == nil {
			if chatGPTData, ok := packagesData["ChatGPT"].(map[string]interface{}); ok {
				if levelVal, ok := chatGPTData["level"].(string); ok {
					packageName = strings.ToLower(levelVal)
				}
			}
		} else {
			log.Printf("警告：为用户 %s 解析 active_packages JSON 失败，将使用默认套餐 'free'。错误: %v", xuserid, errJson)
		}
	}

	limitStr := getLimitRule(packageName, model)
	if limitStr == "" {
		// 如果没有配置具体的规则，可以默认允许或拒绝，这里选择拒绝并提示
		return false, "未配置速率限制", nil
	}

	maxCount, windowSeconds, err := parseLimit(limitStr)
	if err != nil {
		return false, "", fmt.Errorf("解析限速规则 '%s' 失败: %w", limitStr, err)
	}
	if windowSeconds <= 0 { // 确保窗口时间有效
		return false, "", fmt.Errorf("无效的限速时间窗口: %d 秒", windowSeconds)
	}

	redisKey := fmt.Sprintf("star_rate_limit:%s:%s:%s", xuserid, packageName, model)
	userPackageKey := fmt.Sprintf("star_rate_limit_package:%s", xuserid)

	// 检查用户当前套餐是否发生变化
	storedPackage, foundStoredPackage, errStoredPackage := redis.RC.GetString(ctx, userPackageKey) // 调用 redis 包
	if errStoredPackage != nil {
		// 如果读取存储的套餐信息失败，记录警告但继续，这可能导致套餐变更时速率限制不重置
		log.Printf("警告：为用户 %s 获取已存储套餐失败: %v", xuserid, errStoredPackage)
	}

	if !foundStoredPackage || storedPackage != packageName {
		// 套餐发生变化或首次使用，重置计数器并更新存储的套餐信息
		// 设置计数器键的过期时间
		if err := redis.RC.Set(ctx, redisKey, "0", time.Duration(windowSeconds)*time.Second); err != nil { // 调用 redis 包
			return false, "", fmt.Errorf("为用户 %s 重置速率限制计数器失败: %w", xuserid, err)
		}
		// 用户套餐键本身可以不设过期或设一个较长的过期时间
		if err := redis.RC.Set(ctx, userPackageKey, packageName, 0); err != nil { // 0 表示无过期, 调用 redis 包
			return false, "", fmt.Errorf("为用户 %s 设置用户套餐键失败: %w", xuserid, err)
		}
	}

	// 获取当前计数器值
	currentCount, foundCount, errCount := redis.RC.GetInt(ctx, redisKey) // 调用 redis 包
	if errCount != nil && foundCount { // 仅当键存在但解析为整数失败时才认为是严重错误
		return false, "", fmt.Errorf("为用户 %s 获取或解析速率限制计数失败: %w", xuserid, errCount)
	}
	// 如果键不存在（例如，因为过期或首次在新窗口内请求），计数视为0
	if !foundCount {
		currentCount = 0
	}

	if currentCount >= maxCount {
		msg := fmt.Sprintf("超过速率限制：在该 %s 套餐下，%s 模型每 %d 分钟允许 %d 条消息，请稍后重试或升级套餐。",
			packageName, model, windowSeconds/60, maxCount)
		return false, msg, nil
	}

	// 如果没有超过限制，递增计数器
	newCount, err := redis.RC.Incr(ctx, redisKey) // 调用 redis 包
	if err != nil {
		return false, "", fmt.Errorf("为用户 %s 递增速率限制计数器失败: %w", xuserid, err)
	}

	// 如果是窗口内的第一次递增（即 newCount == 1），则设置过期时间
	// 这确保了滑动窗口的正确性
	if newCount == 1 {
		err := redis.RC.Expire(ctx, redisKey, time.Duration(windowSeconds)*time.Second) // 调用 redis 包
		if err != nil {
			// 记录错误，但不因此次失败而使请求失败。
			// 计数器已递增，最坏的情况是如果此设置失败，它会提前或延迟过期。
			log.Printf("警告：为速率限制键 %s 设置过期时间失败: %v", redisKey, err)
		}
	}

	return true, "允许发送消息", nil
}