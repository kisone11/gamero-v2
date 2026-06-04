// Package service 提供评论 @提及相关的工具函数。
package service

import "regexp"

// mentionRegexp 匹配 @username 模式（字母、数字、下划线，3-32 个字符）
var mentionRegexp = regexp.MustCompile(`@([a-zA-Z0-9_]{3,32})`)

// maxMentionsPerComment 每条评论最多触发的提及通知数（防滥用）
const maxMentionsPerComment = 5

// ExtractMentions 从文本中提取 @username 列表（去重，最多返回 maxMentionsPerComment 个）
func ExtractMentions(content string) []string {
	matches := mentionRegexp.FindAllStringSubmatch(content, -1)
	seen := make(map[string]bool)
	result := make([]string, 0, maxMentionsPerComment)
	for _, m := range matches {
		username := m[1]
		if !seen[username] {
			seen[username] = true
			result = append(result, username)
		}
		if len(result) >= maxMentionsPerComment {
			break
		}
	}
	return result
}

// extractMentions 内部别名，保持已有调用不变
func extractMentions(content string) []string {
	return ExtractMentions(content)
}
