package config

import (
	"log"
	"os"
	"strconv"

	"github.com/joho/godotenv" // 可选：用于加载 .env 文件
)

// AppConfig 保存应用程序配置
type AppConfig struct {
	RedisHost     string
	RedisPort     int
	RedisPassword string
	RedisDB       int
	ServerPort    int
}

// Cfg 是全局配置实例
var Cfg AppConfig

// LoadConfig 从环境变量加载配置
// 如果需要，可以扩展此函数以从文件（例如 config.yaml）加载配置
func LoadConfig() {
	// 可选：首先加载 .env 文件。对本地开发很有用。
	// 在生产环境中，通常直接设置环境变量。
	if err := godotenv.Load(); err != nil {
		log.Println("未找到 .env 文件，将使用环境变量或默认值")
	}

	Cfg.RedisHost = getEnv("REDIS_HOST", "localhost") // 将默认值更改为 localhost 以匹配常见的开发设置
	Cfg.RedisPort = getEnvAsInt("REDIS_PORT", 6379)
	Cfg.RedisPassword = getEnv("REDIS_PASSWORD", "") // 空字符串表示没有密码
	Cfg.RedisDB = getEnvAsInt("REDIS_DB", 0)
	Cfg.ServerPort = getEnvAsInt("SERVER_PORT", 19892) // 端口来自 Python 的 limit_main.py
}

// getEnv 获取环境变量的值，如果未设置则返回默认值
func getEnv(key string, defaultVal string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultVal
}

// getEnvAsInt 获取环境变量的值并转换为整数，如果未设置或转换失败则返回默认值
func getEnvAsInt(key string, defaultVal int) int {
	valueStr := getEnv(key, "")
	if value, err := strconv.Atoi(valueStr); err == nil {
		return value
	}
	return defaultVal
}