package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strconv"
	"time"

	"github.com/Pandolar/limit_service/internal/config" // 使用新的模块路径
	"github.com/go-redis/redis/v9"
)

// ClientWrapper 包装了 Redis 客户端
type ClientWrapper struct {
	Client *redis.Client
	Prefix string
}

// RC 是全局 Redis 客户端实例
var RC *ClientWrapper

// InitRedis 初始化全局 Redis 客户端
func InitRedis(cfg config.AppConfig) error {
	rdb := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%d", cfg.RedisHost, cfg.RedisPort),
		Password: cfg.RedisPassword,
		DB:       cfg.RedisDB,
	})

	// Ping 以检查连接
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if _, err := rdb.Ping(ctx).Result(); err != nil {
		return fmt.Errorf("连接 Redis 失败: %w", err)
	}

	RC = &ClientWrapper{
		Client: rdb,
		Prefix: "star:", // 与 Python 的 redis_tools.py 中的前缀相同
	}
	return nil
}

// getKey 辅助函数，用于获取带前缀的键名
func (rc *ClientWrapper) getKey(key string) string {
	return rc.Prefix + key
}

// Set 将键值对存储到 Redis 中。复杂类型会进行 JSON 序列化。
func (rc *ClientWrapper) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	fullKey := rc.getKey(key)
	var dataToStore interface{}

	switch v := value.(type) {
	// 对于简单类型，直接存储
	case string, []byte, int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64, float32, float64, bool:
		dataToStore = v
	default:
		// 对于复杂类型（如 map、slice、struct），序列化为 JSON
		jsonData, err := json.Marshal(v)
		if err != nil {
			return fmt.Errorf("序列化键 %s 的值失败: %w", key, err)
		}
		dataToStore = jsonData
	}

	return rc.Client.Set(ctx, fullKey, dataToStore, expiration).Err()
}

// Get 从 Redis 检索值。如果值看起来像 JSON，它会尝试反序列化 JSON。
// 返回 (值, 是否找到, 错误)
func (rc *ClientWrapper) Get(ctx context.Context, key string) (interface{}, bool, error) {
	fullKey := rc.getKey(key)
	val, err := rc.Client.Get(ctx, fullKey).Result()
	if err == redis.Nil {
		return nil, false, nil // 键不存在
	}
	if err != nil {
		return nil, false, fmt.Errorf("从 Redis 获取键 %s 失败: %w", key, err)
	}

	// 尝试将值作为 JSON 反序列化到 map[string]interface{} 以获得灵活性，
	// 类似于 Python 的 ast.literal_eval 处理 JSON 字符串的行为。
	var jsonData map[string]interface{}
	if err := json.Unmarshal([]byte(val), &jsonData); err == nil {
		// 可以根据具体的数据结构进行更细致的判断
		// 例如，通过检查特定键是否存在来判断是否为 active_packages
		if _, ok := jsonData["ChatGPT"]; ok {
			return jsonData, true, nil
		}
		// 如果是有效的 JSON 但不是已知的特定结构，也可能是一个列表或其他 JSON 对象。
		// 为简单起见，如果能成功反序列化为 map，则返回该 map。
	}

	// 如果不是有效的 JSON，或者是一个简单的字符串，则按原样返回字符串
	return val, true, nil
}

// GetString 从 Redis 检索字符串类型的值。
// 返回 (值, 是否找到, 错误)
func (rc *ClientWrapper) GetString(ctx context.Context, key string) (string, bool, error) {
	fullKey := rc.getKey(key)
	val, err := rc.Client.Get(ctx, fullKey).Result()
	if err == redis.Nil {
		return "", false, nil // 键不存在
	}
	if err != nil {
		return "", false, fmt.Errorf("从 Redis 获取键 %s 失败: %w", key, err)
	}
	return val, true, nil
}

// GetInt 从 Redis 检索整数类型的值。
// 返回 (值, 是否找到, 错误)
func (rc *ClientWrapper) GetInt(ctx context.Context, key string) (int, bool, error) {
	valStr, found, err := rc.GetString(ctx, key)
	if err != nil || !found {
		return 0, found, err
	}
	var valInt int
	valInt, err = strconv.Atoi(valStr)
	if err != nil {
		// 如果值存在但不能转换为整数，记录日志但仍认为键是找到的
		log.Printf("警告：键 %s 的值 (%s) 无法转换为整数: %v", key, valStr, err)
		return 0, true, fmt.Errorf("转换键 %s 的值为整数失败: %w", key, err)
	}
	return valInt, true, nil
}

// Incr 将键的整数值加一。
func (rc *ClientWrapper) Incr(ctx context.Context, key string) (int64, error) {
	fullKey := rc.getKey(key)
	val, err := rc.Client.Incr(ctx, fullKey).Result()
	if err != nil {
		return 0, fmt.Errorf("自增键 %s 失败: %w", key, err)
	}
	return val, nil
}

// Expire 设置键的过期时间。
func (rc *ClientWrapper) Expire(ctx context.Context, key string, expiration time.Duration) error {
	fullKey := rc.getKey(key)
	if err := rc.Client.Expire(ctx, fullKey, expiration).Err(); err != nil {
		return fmt.Errorf("设置键 %s 的过期时间失败: %w", key, err)
	}
	return nil
}