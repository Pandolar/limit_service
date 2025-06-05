package config

import (
	"os"
	"strconv"
)

// RedisConfig Redis连接配置
type RedisConfig struct {
	Host     string
	Port     int
	Password string
	DB       int
}

// Config 应用配置
type Config struct {
	Redis RedisConfig
}

// GetConfig 获取应用配置
func GetConfig() *Config {
	return &Config{
		Redis: RedisConfig{
			Host:     getEnvString("REDIS_HOST", "redis"),
			Port:     getEnvInt("REDIS_PORT", 6379),
			Password: getEnvString("REDIS_PASSWORD", ""),
			DB:       getEnvInt("REDIS_DB", 0),
		},
	}
}

// getEnvString 获取字符串环境变量，如果不存在则返回默认值
func getEnvString(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// getEnvInt 获取整数环境变量，如果不存在或转换失败则返回默认值
func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
} 