package tools

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	ahocorasick "github.com/petar-dambovaliev/aho-corasick"
)

// 全局Aho-Corasick自动机
var automaton ahocorasick.AhoCorasick
var automatonInitialized bool

// InitKeyWords 初始化关键词自动机
func InitKeyWords() error {
	// 读取关键词文件
	file, err := os.Open("./data/keywords.txt")
	if err != nil {
		return fmt.Errorf("打开关键词文件失败: %w", err)
	}
	defer file.Close()

	var patterns []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		word := strings.TrimSpace(scanner.Text())
		if word != "" {
			patterns = append(patterns, word)
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("读取关键词文件失败: %w", err)
	}

	// 创建Aho-Corasick自动机
	builder := ahocorasick.NewAhoCorasickBuilder(ahocorasick.Opts{
		AsciiCaseInsensitive: false,
		MatchOnlyWholeWords:  false,
		MatchKind:            ahocorasick.LeftMostFirstMatch,
		DFA:                  true,
	})
	
	automaton = builder.Build(patterns)
	automatonInitialized = true
	fmt.Println("关键词自动机初始化完成")
	return nil
}

// StarAudit 审核用户输入的文本
// 参数: prompt - 用户的输入文本
// 返回: true表示安全，false表示包含违禁词
func StarAudit(prompt interface{}) bool {
	// 检查输入是否为字符串
	promptStr, ok := prompt.(string)
	if !ok {
		fmt.Println("prompt不是字符串")
		fmt.Printf("prompt: %v\n", prompt)
		return true // 如果不是字符串，认为是安全的
	}

	// 检查automaton是否已初始化
	if !automatonInitialized {
		fmt.Println("警告：关键词自动机未初始化，跳过审核")
		return true // 如果自动机未初始化，认为是安全的
	}

	// 使用自动机搜索违禁词
	matches := automaton.FindAll(promptStr)
	
	// 如果找到任何违禁词，返回false
	if len(matches) > 0 {
		fmt.Println("发现黑名单")
		for _, match := range matches {
			keyword := promptStr[match.Start():match.End()]
			fmt.Printf("key word: %s\n", keyword)
		}
		fmt.Printf("prompt: %s\n", promptStr)
		return false
	}

	return true
} 