// Package filter 提供基于 Aho-Corasick 自动机的敏感词过滤功能（纯 Go 实现，无外部依赖）。
//
// 使用方式：
//
//	filter.Init(filter.DefaultSensitiveWords) // 应用启动时初始化
//	if filter.Contains(text) { ... }          // 检测是否含敏感词
//	clean := filter.Replace(text)             // 将敏感词替换为 ***
package filter

import (
	"strings"
	"sync"
	"unicode"
)

// node AC 自动机节点
type node struct {
	fail     *node
	isEnd    bool
	word     string // 命中的敏感词（小写）
	childMap map[rune]*node
}

func newNode() *node {
	return &node{childMap: make(map[rune]*node)}
}

// SensitiveFilter AC 自动机敏感词过滤器（线程安全）
type SensitiveFilter struct {
	root  *node
	mu    sync.RWMutex
	words []string // 保存当前词库（用于 AddWords 重建时合并）
}

// NewSensitiveFilter 创建过滤器并加载词库
func NewSensitiveFilter(words []string) *SensitiveFilter {
	sf := &SensitiveFilter{root: newNode()}
	for _, w := range words {
		w = strings.ToLower(strings.TrimSpace(w))
		if w != "" {
			sf.addWord(w)
			sf.words = append(sf.words, w)
		}
	}
	sf.build()
	return sf
}

// addWord 向 Trie 插入单词（小写，不加锁，仅在构建阶段调用）
func (sf *SensitiveFilter) addWord(word string) {
	cur := sf.root
	for _, ch := range word {
		if cur.childMap[ch] == nil {
			cur.childMap[ch] = newNode()
		}
		cur = cur.childMap[ch]
	}
	cur.isEnd = true
	cur.word = word
}

// build 通过 BFS 构建 fail 指针
func (sf *SensitiveFilter) build() {
	queue := make([]*node, 0, 64)
	for _, child := range sf.root.childMap {
		child.fail = sf.root
		queue = append(queue, child)
	}
	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]
		for ch, child := range cur.childMap {
			// 沿 fail 链找到最长可匹配后缀
			fail := cur.fail
			for fail != nil && fail.childMap[ch] == nil {
				fail = fail.fail
			}
			if fail == nil {
				child.fail = sf.root
			} else {
				child.fail = fail.childMap[ch]
				if child.fail == child {
					child.fail = sf.root
				}
			}
			queue = append(queue, child)
		}
	}
}

// Contains 判断文本是否包含任意敏感词（忽略大小写，跳过空白字符）
func (sf *SensitiveFilter) Contains(text string) bool {
	sf.mu.RLock()
	defer sf.mu.RUnlock()

	cur := sf.root
	for _, ch := range strings.ToLower(text) {
		if unicode.IsSpace(ch) {
			continue
		}
		// 沿 fail 链回退，直到找到匹配或回到根
		for cur != sf.root && cur.childMap[ch] == nil {
			cur = cur.fail
		}
		if next, ok := cur.childMap[ch]; ok {
			cur = next
		}
		if cur.isEnd {
			return true
		}
		// 检查 fail 链上是否有结束节点（处理后缀敏感词）
		for tmp := cur.fail; tmp != nil && tmp != sf.root; tmp = tmp.fail {
			if tmp.isEnd {
				return true
			}
		}
	}
	return false
}

// Replace 将文本中的敏感词替换为等长 '*'
func (sf *SensitiveFilter) Replace(text string) string {
	sf.mu.RLock()
	defer sf.mu.RUnlock()

	runes := []rune(text)
	result := make([]rune, len(runes))
	copy(result, runes)

	cur := sf.root
	lowerRunes := []rune(strings.ToLower(text))

	for i, ch := range lowerRunes {
		for cur != sf.root && cur.childMap[ch] == nil {
			cur = cur.fail
		}
		if next, ok := cur.childMap[ch]; ok {
			cur = next
		}
		if cur.isEnd {
			wordLen := len([]rune(cur.word))
			start := i - wordLen + 1
			if start < 0 {
				start = 0
			}
			for j := start; j <= i && j < len(result); j++ {
				result[j] = '*'
			}
		}
	}
	return string(result)
}

// AddWords 动态追加敏感词（重建自动机，立即生效）
// 注意：追加操作会合并已有词库和新词，重建整棵 Trie。
func (sf *SensitiveFilter) AddWords(newWords []string) {
	sf.mu.Lock()
	defer sf.mu.Unlock()

	// 合并词库（去重）
	merged := make([]string, 0, len(sf.words)+len(newWords))
	seen := make(map[string]struct{}, len(sf.words))
	for _, w := range sf.words {
		seen[w] = struct{}{}
		merged = append(merged, w)
	}
	for _, w := range newWords {
		w = strings.ToLower(strings.TrimSpace(w))
		if w == "" {
			continue
		}
		if _, ok := seen[w]; !ok {
			seen[w] = struct{}{}
			merged = append(merged, w)
		}
	}

	// 重建自动机
	sf.root = newNode()
	sf.words = merged
	for _, w := range merged {
		sf.addWord(w)
	}
	sf.build()
}

// Words 返回当前词库的拷贝（用于管理员查看）
func (sf *SensitiveFilter) Words() []string {
	sf.mu.RLock()
	defer sf.mu.RUnlock()
	result := make([]string, len(sf.words))
	copy(result, sf.words)
	return result
}

// RemoveWord 删除一个敏感词并重建自动机（小写匹配，立即生效）
func (sf *SensitiveFilter) RemoveWord(word string) {
	word = strings.ToLower(strings.TrimSpace(word))
	if word == "" {
		return
	}
	sf.mu.Lock()
	defer sf.mu.Unlock()

	// 过滤掉该词
	filtered := sf.words[:0]
	for _, w := range sf.words {
		if w != word {
			filtered = append(filtered, w)
		}
	}
	if len(filtered) == len(sf.words) {
		return // 词不存在，无需重建
	}

	// 重建自动机
	sf.root = newNode()
	sf.words = filtered
	for _, w := range filtered {
		sf.addWord(w)
	}
	sf.build()
}

// ===========================
// 全局单例
// ===========================

var (
	global *SensitiveFilter
	once   sync.Once
)

// Init 初始化全局过滤器（only-once，应在应用启动时调用）
func Init(words []string) {
	once.Do(func() {
		global = NewSensitiveFilter(words)
	})
}

// Get 获取全局过滤器。若未调用 Init，返回空过滤器（不会 panic）
func Get() *SensitiveFilter {
	if global == nil {
		return NewSensitiveFilter(nil)
	}
	return global
}

// Contains 包级便捷函数：判断文本是否含敏感词
func Contains(text string) bool {
	return Get().Contains(text)
}

// Replace 包级便捷函数：替换文本中的敏感词
func Replace(text string) string {
	return Get().Replace(text)
}

// RemoveWord 包级便捷函数：删除一个敏感词
func RemoveWord(word string) {
	Get().RemoveWord(word)
}

// DefaultSensitiveWords 默认敏感词列表（中英文各示例，生产环境应从 DB/文件加载）
var DefaultSensitiveWords = []string{
	// --- 暴力/仇恨 ---
	"fuck", "shit", "asshole", "bitch", "bastard",
	// --- 中文示例（脱敏符号占位，实际部署请从外部文件加载） ---
	"草泥马", "他妈的", "傻逼", "操你", "滚你妈",
	// --- 广告/诈骗 ---
	"加微信", "私聊我", "免费领取",
}
