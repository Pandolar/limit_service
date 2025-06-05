package auditor

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/anknown/ahocorasick" // Aho-Corasick 库
)

// matcher 是 Aho-Corasick 匹配器实例
var matcher *ahocorasick.Matcher

const keywordsFilePath = "./data/keywords.txt" // 关键词文件路径

// InitAuditor 初始化 Aho-Corasick 匹配器
func InitAuditor() error {
	file, err := os.Open(keywordsFilePath)
	if err != nil {
		return fmt.Errorf("打开关键词文件 %s 失败: %w", keywordsFilePath, err)
	}
	defer file.Close()

	var keywords [][]byte
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		word := strings.TrimSpace(scanner.Text())
		if word != "" {
			keywords = append(keywords, []byte(word))
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("读取关键词文件 %s 出错: %w", keywordsFilePath, err)
	}

	if len(keywords) == 0 {
		log.Println("警告: 关键词文件为空或所有行都为空，审核功能可能无效。")
		// 即使没有关键词，也创建一个空的匹配器，以避免 nil panic
		matcher = ahocorasick.NewMatcher([][]byte{})
		return nil
	}

	matcher = ahocorasick.NewMatcher(keywords)
	log.Printf("Aho-Corasick 审核器已使用 %d 个关键词初始化。", len(keywords))
	return nil
}

// StarAudit 检查 prompt 是否包含任何违禁关键词。
// 如果安全（未找到违禁词）则返回 true，否则返回 false。
func StarAudit(prompt string) bool {
	// 如果 prompt 为空字符串，或者 matcher 未初始化（例如关键词文件读取失败但未 panic），则视为安全
	if prompt == "" || matcher == nil {
		return true
	}

	hits := matcher.Match([]byte(prompt))
	if len(hits) > 0 {
		// 为了安全和隐私，通常不在生产日志中记录完整的 prompt，除非有明确的调试需求和合规策略
		log.Printf("发现违禁关键词。关键词: %s", string(hits[0].Word)) // Prompt 详情可按需记录
		return false // 找到违禁词
	}
	return true // 安全
}