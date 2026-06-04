// Package service 提供热度分数计算逻辑。
package service

import (
	"math"
	"time"

	"github.com/gamero/gamero/internal/model"
)

// CalcPostHotScore 计算帖子热度分数（HackerNews 式算法）
//
// 公式：
//
//	hot_score = (like_count + comment_count * 0.5 + view_count * 0.1) / pow(age_hours + 2, 1.5)
//	age_hours = (now - created_at).Hours()
//
// 时效惩罚因子让新帖子更易上榜，随时间衰减。
func CalcPostHotScore(post *model.Post) float64 {
	ageHours := time.Since(post.CreatedAt).Hours()
	if ageHours < 0 {
		ageHours = 0
	}
	numerator := float64(post.LikeCount) + float64(post.CommentCount)*0.5 + float64(post.ViewCount)*0.1
	denominator := math.Pow(ageHours+2, 1.5)
	return numerator / denominator
}

// HotScore 通用热度分数（含新鲜度加成）
// 基础公式: (likes + comments*0.5 + views*0.1) / pow(hours+2, 1.5)
// 新鲜度: <7天内容 ×1.5
func HotScore(likes, comments, views int64, createdAt time.Time) float64 {
	ageHours := time.Since(createdAt).Hours()
	if ageHours < 0 {
		ageHours = 0
	}
	numerator := float64(likes) + float64(comments)*0.5 + float64(views)*0.1
	denominator := math.Pow(ageHours+2, 1.5)
	score := numerator / denominator
	// 新鲜度加成: <7天内容 ×1.5
	if ageHours < 24*7 {
		score *= 1.5
	}
	return score
}
