// Package sensitive 提供基于 DFA（确定性有限自动机）的敏感词过滤功能。
// 内置 50+ 常见敏感词（脏话、政治、色情相关），支持检查和替换两个公开方法。
// 设计上与 pkg/filter 的 AC 自动机互补：
//   - pkg/filter: 多模式 Aho-Corasick，适合大词库、实时流量检测
//   - pkg/sensitive: 轻量 DFA，适合注册/昵称等低频场景的二次校验
package sensitive

import (
	"strings"
	"unicode"
)

// ===========================
// 内置敏感词列表（50+ 词）
// ===========================

// DefaultWords 内置敏感词列表。
// 按类别分组：脏话/粗口、政治敏感、色情相关、诈骗广告。
// 生产环境建议从数据库/外部文件加载，通过 NewDFA(words) 动态初始化。
var DefaultWords = []string{
	// ─── 脏话/粗口（中文）───
	"草泥马",
	"他妈的",
	"傻逼",
	"操你",
	"滚你妈",
	"妈的",
	"卧槽",
	"狗屁",
	"废物",
	"贱人",
	"死妈",
	"臭逼",
	"操蛋",
	"牛逼你妈",
	"去你妈",
	"煞笔",
	"智障",
	"脑残",
	"神经病",
	"你妈死了",

	// ─── 脏话/粗口（英文）───
	"fuck",
	"shit",
	"asshole",
	"bitch",
	"bastard",
	"damn",
	"crap",
	"piss",
	"dick",
	"pussy",
	"cock",
	"cunt",
	"motherfucker",
	"bullshit",
	"fuckoff",
	"whore",

	// ─── 政治敏感（通用词，可根据合规要求调整）───
	"六四",
	"天安门事件",
	"法轮功",
	"达赖喇嘛",
	"台独",
	"港独",
	"藏独",
	"疆独",
	"推翻共产党",
	"颠覆政权",

	// ─── 色情相关 ───
	"av女优",
	"黄片",
	"裸聊",
	"援交",
	"约炮",
	"开房",
	"性爱",
	"一夜情",
	"卖淫",
	"嫖娼",

	// ─── 诈骗/广告 ───
	"加微信",
	"私聊我",
	"免费领取",
	"代开发票",
	"高利贷",
	"洗钱",
	"刷单",
	"兼职日赚",
	"点击领红包",
	"微信号",
}

// ===========================
// DFA 节点定义
// ===========================

// dfaNode DFA 状态节点（Trie 结构实现）
type dfaNode struct {
	children map[rune]*dfaNode
	isEnd    bool   // 是否为某个敏感词的结尾
	word     string // 命中的原始敏感词（用于替换时计算长度）
}

func newDFANode() *dfaNode {
	return &dfaNode{children: make(map[rune]*dfaNode)}
}

// ===========================
// DFA 过滤器
// ===========================

// DFA 确定性有限自动机敏感词过滤器。
// 零外部依赖，纯标准库实现。
type DFA struct {
	root *dfaNode
}

// NewDFA 使用给定词库构建 DFA 过滤器。
// words 中的词会被转换为小写后插入 Trie。
func NewDFA(words []string) *DFA {
	d := &DFA{root: newDFANode()}
	for _, w := range words {
		d.insert(strings.ToLower(strings.TrimSpace(w)))
	}
	return d
}

// insert 向 Trie 插入一个敏感词
func (d *DFA) insert(word string) {
	if word == "" {
		return
	}
	cur := d.root
	for _, ch := range word {
		if cur.children[ch] == nil {
			cur.children[ch] = newDFANode()
		}
		cur = cur.children[ch]
	}
	cur.isEnd = true
	cur.word = word
}

// Check 检查文本是否包含任意敏感词（忽略大小写、跳过空白）。
// 返回 true 表示包含敏感词，false 表示干净。
func (d *DFA) Check(text string) bool {
	normalized := normalize(text)
	runes := []rune(normalized)
	n := len(runes)

	for i := 0; i < n; i++ {
		cur := d.root
		j := i
		for j < n {
			ch := runes[j]
			next, ok := cur.children[ch]
			if !ok {
				break
			}
			cur = next
			if cur.isEnd {
				return true
			}
			j++
		}
	}
	return false
}

// Replace 将文本中的敏感词替换为等长 '*'。
// 若文本中含多个敏感词，全部替换。忽略大小写。
func (d *DFA) Replace(text string) string {
	runes := []rune(text)
	normalized := []rune(normalize(text))
	n := len(runes)
	masked := make([]bool, n)

	for i := 0; i < n; i++ {
		cur := d.root
		j := i
		lastEnd := -1

		for j < n {
			ch := normalized[j]
			next, ok := cur.children[ch]
			if !ok {
				break
			}
			cur = next
			if cur.isEnd {
				lastEnd = j
			}
			j++
		}

		if lastEnd >= 0 {
			// 标记 [i, lastEnd] 范围的字符为敏感
			for k := i; k <= lastEnd; k++ {
				masked[k] = true
			}
		}
	}

	result := make([]rune, n)
	for i, r := range runes {
		if masked[i] {
			result[i] = '*'
		} else {
			result[i] = r
		}
	}
	return string(result)
}

// normalize 将文本转为小写并去除空白字符，用于 DFA 匹配
func normalize(text string) string {
	var sb strings.Builder
	sb.Grow(len(text))
	for _, r := range strings.ToLower(text) {
		if !unicode.IsSpace(r) {
			sb.WriteRune(r)
		}
	}
	return sb.String()
}

// ===========================
// 全局单例 & 便捷函数
// ===========================

var globalDFA = NewDFA(DefaultWords)

// Check 包级便捷函数：使用内置词库检查文本是否含敏感词。
func Check(text string) bool {
	return globalDFA.Check(text)
}

// Replace 包级便捷函数：使用内置词库将敏感词替换为 '*'。
func Replace(text string) string {
	return globalDFA.Replace(text)
}
