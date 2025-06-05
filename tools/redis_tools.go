package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/go-redis/redis/v8"
	"limit_service/config"
)

// RedisTool Redis工具结构体
type RedisTool struct {
	client *redis.Client
	prefix string
	ctx    context.Context
}

// 全局Redis工具实例
var RedisClient *RedisTool

// InitRedis 初始化Redis连接
func InitRedis() error {
	cfg := config.GetConfig()
	
	rdb := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%d", cfg.Redis.Host, cfg.Redis.Port),
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
	})

	// 测试连接
	ctx := context.Background()
	_, err := rdb.Ping(ctx).Result()
	if err != nil {
		return fmt.Errorf("Redis连接失败: %w", err)
	}

	RedisClient = &RedisTool{
		client: rdb,
		prefix: "star:",
		ctx:    ctx,
	}

	fmt.Println("Redis连接成功")
	return nil
}

// getKey 返回带有前缀的键名
func (r *RedisTool) getKey(key string) string {
	return r.prefix + key
}

// Set 设置键值对到Redis中
func (r *RedisTool) Set(key string, value interface{}, expiration time.Duration) error {
	fullKey := r.getKey(key)
	
	// 如果value是复杂类型，序列化为JSON
	var val string
	switch v := value.(type) {
	case string:
		val = v
	case int, int32, int64, float32, float64, bool:
		val = fmt.Sprintf("%v", v)
	default:
		jsonBytes, err := json.Marshal(v)
		if err != nil {
			return fmt.Errorf("序列化失败: %w", err)
		}
		val = string(jsonBytes)
	}

	return r.client.Set(r.ctx, fullKey, val, expiration).Err()
}

// Get 从Redis中获取键的值
func (r *RedisTool) Get(key string) (interface{}, error) {
	fullKey := r.getKey(key)
	val, err := r.client.Get(r.ctx, fullKey).Result()
	if err != nil {
		if err == redis.Nil {
			return nil, nil // 键不存在
		}
		return nil, err
	}

	// 尝试解析为JSON
	var result interface{}
	if err := json.Unmarshal([]byte(val), &result); err == nil {
		return result, nil
	}

	// 如果不是JSON，返回原始字符串
	return val, nil
}

// GetString 获取字符串值
func (r *RedisTool) GetString(key string) (string, error) {
	fullKey := r.getKey(key)
	return r.client.Get(r.ctx, fullKey).Result()
}

// GetInt 获取整数值
func (r *RedisTool) GetInt(key string) (int, error) {
	val, err := r.GetString(key)
	if err != nil {
		return 0, err
	}
	return strconv.Atoi(val)
}

// Exists 检查键是否存在
func (r *RedisTool) Exists(key string) (bool, error) {
	fullKey := r.getKey(key)
	result, err := r.client.Exists(r.ctx, fullKey).Result()
	return result > 0, err
}

// Delete 删除键
func (r *RedisTool) Delete(key string) error {
	fullKey := r.getKey(key)
	return r.client.Del(r.ctx, fullKey).Err()
}

// Expire 设置过期时间
func (r *RedisTool) Expire(key string, expiration time.Duration) error {
	fullKey := r.getKey(key)
	return r.client.Expire(r.ctx, fullKey, expiration).Err()
}

// Incr 自增
func (r *RedisTool) Incr(key string) (int64, error) {
	fullKey := r.getKey(key)
	return r.client.Incr(r.ctx, fullKey).Result()
}

// TTL 获取键的剩余生存时间
func (r *RedisTool) TTL(key string) (time.Duration, error) {
	fullKey := r.getKey(key)
	return r.client.TTL(r.ctx, fullKey).Result()
} 