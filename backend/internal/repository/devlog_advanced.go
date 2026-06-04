// devlog_advanced.go - DevLog Repository 高级查询实现
package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/gamero/gamero/internal/model"
	"gorm.io/gorm"
)

// ===== 高级查询类型定义 =====

// ProjectLogStats 项目日志统计
type ProjectLogStats struct {
	TotalLogs     int64
	DevlogCount   int64
	ReleaseCount  int64
	TutorialCount int64
	TotalLikes    int64
	TotalComments int64
}

// MonthlyLogCount 月度日志统计
type MonthlyLogCount struct {
	Year  int
	Month int
	Count int64
}

// LogCommentNode 日志评论树节点
type LogCommentNode struct {
	model.DevLogComment
	Replies    []*model.DevLogComment
	ReplyCount int64
}

// ===== DevLogRepository 高级查询实现 =====

// GetProjectLogStats 按项目聚合日志统计
func (r *devLogRepository) GetProjectLogStats(ctx context.Context, projectID uint64) (*ProjectLogStats, error) {
	var stats ProjectLogStats

	err := r.db.WithContext(ctx).Raw(`
		SELECT 
			COUNT(*) as total_logs,
			SUM(CASE WHEN log_type = 'devlog' THEN 1 ELSE 0 END) as devlog_count,
			SUM(CASE WHEN log_type = 'release' THEN 1 ELSE 0 END) as release_count,
			SUM(CASE WHEN log_type = 'tutorial' THEN 1 ELSE 0 END) as tutorial_count,
			COALESCE(SUM(like_count), 0) as total_likes,
			COALESCE(SUM(comment_count), 0) as total_comments
		FROM dev_logs
		WHERE project_id = ? AND deleted_at IS NULL
	`, projectID).Scan(&stats).Error

	if err != nil {
		return nil, fmt.Errorf("获取项目日志统计失败: %w", err)
	}

	return &stats, nil
}

// GetLogTimeline 日志时间线(按月聚合数量)
func (r *devLogRepository) GetLogTimeline(ctx context.Context, projectID uint64) ([]MonthlyLogCount, error) {
	var timeline []MonthlyLogCount

	err := r.db.WithContext(ctx).Raw(`
		SELECT 
			EXTRACT(YEAR FROM created_at)::int as year,
			EXTRACT(MONTH FROM created_at)::int as month,
			COUNT(*) as count
		FROM dev_logs
		WHERE project_id = ? AND deleted_at IS NULL
		GROUP BY year, month
		ORDER BY year DESC, month DESC
	`, projectID).Scan(&timeline).Error

	if err != nil {
		return nil, fmt.Errorf("获取日志时间线失败: %w", err)
	}

	return timeline, nil
}

// ListHotLogs 热门日志排行
func (r *devLogRepository) ListHotLogs(ctx context.Context, days int, logType string, page, pageSize int) ([]*model.DevLog, int64, error) {
	offset := (page - 1) * pageSize
	cutoffTime := time.Now().AddDate(0, 0, -days)

	query := r.db.WithContext(ctx).Model(&model.DevLog{}).
		Where("deleted_at IS NULL AND created_at >= ?", cutoffTime)

	// 日志类型筛选
	if logType != "" {
		query = query.Where("log_type = ?", logType)
	}

	// 统计总数
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("统计热门日志总数失败: %w", err)
	}

	// 热度分数 = 点赞数*2 + 评论数*3 + 收藏数*1.5 + 浏览数*0.1
	var logs []*model.DevLog
	err := r.db.WithContext(ctx).
		Table("dev_logs").
		Select(`*, (like_count * 2 + comment_count * 3 + collect_count * 1.5 + view_count * 0.1) as hot_score`).
		Where("deleted_at IS NULL AND created_at >= ?", cutoffTime).
		Scopes(func(db *gorm.DB) *gorm.DB {
			if logType != "" {
				return db.Where("log_type = ?", logType)
			}
			return db
		}).
		Order("hot_score DESC, created_at DESC").
		Offset(offset).Limit(pageSize).
		Find(&logs).Error

	if err != nil {
		return nil, 0, fmt.Errorf("查询热门日志失败: %w", err)
	}

	return logs, total, nil
}

// GetRelatedLogs 相关日志推荐(同项目/同作者/相似标签)
func (r *devLogRepository) GetRelatedLogs(ctx context.Context, logID uint64, limit int) ([]*model.DevLog, error) {
	// 先获取原日志
	var originLog model.DevLog
	if err := r.db.WithContext(ctx).First(&originLog, logID).Error; err != nil {
		return nil, fmt.Errorf("查询原始日志失败: %w", err)
	}

	// 推荐策略: 同项目 > 同作者 > 相似标签
	var relatedLogs []*model.DevLog

	// 1. 优先同项目的其他日志
	if err := r.db.WithContext(ctx).
		Where("project_id = ? AND id != ? AND deleted_at IS NULL", originLog.ProjectID, logID).
		Order("created_at DESC").
		Limit(limit).
		Find(&relatedLogs).Error; err != nil {
		return nil, fmt.Errorf("查询同项目日志失败: %w", err)
	}

	// 如果不够，补充同作者的日志
	if len(relatedLogs) < limit {
		remaining := limit - len(relatedLogs)
		var authorLogs []*model.DevLog
		if err := r.db.WithContext(ctx).
			Where("author_id = ? AND id != ? AND project_id != ? AND deleted_at IS NULL",
				originLog.AuthorID, logID, originLog.ProjectID).
			Order("created_at DESC").
			Limit(remaining).
			Find(&authorLogs).Error; err == nil {
			relatedLogs = append(relatedLogs, authorLogs...)
		}
	}

	return relatedLogs, nil
}

// ===== DevLogCommentRepository 高级查询实现 =====

// GetLogCommentTree 评论树(楼中楼)
func (r *devLogCommentRepository) GetLogCommentTree(ctx context.Context, logID uint64, page, pageSize int) ([]*LogCommentNode, int64, error) {
	offset := (page - 1) * pageSize

	// 统计一级评论总数
	var total int64
	if err := r.db.WithContext(ctx).
		Model(&model.DevLogComment{}).
		Where("log_id = ? AND reply_to_id = 0 AND deleted_at IS NULL", logID).
		Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("统计一级评论失败: %w", err)
	}

	// 查询一级评论
	var parentComments []*model.DevLogComment
	if err := r.db.WithContext(ctx).
		Where("log_id = ? AND reply_to_id = 0 AND deleted_at IS NULL", logID).
		Order("created_at ASC").
		Offset(offset).Limit(pageSize).
		Find(&parentComments).Error; err != nil {
		return nil, 0, fmt.Errorf("查询一级评论失败: %w", err)
	}

	if len(parentComments) == 0 {
		return []*LogCommentNode{}, total, nil
	}

	// 提取一级评论 ID
	parentIDs := make([]uint64, len(parentComments))
	for i, c := range parentComments {
		parentIDs[i] = c.ID
	}

	// 批量查询子评论
	var childComments []*model.DevLogComment
	if err := r.db.WithContext(ctx).
		Where("log_id = ? AND reply_to_id IN ? AND deleted_at IS NULL", logID, parentIDs).
		Order("created_at ASC").
		Find(&childComments).Error; err != nil {
		return nil, 0, fmt.Errorf("查询子评论失败: %w", err)
	}

	// 组装评论树
	childMap := make(map[uint64][]*model.DevLogComment)
	for _, child := range childComments {
		childMap[child.ReplyToID] = append(childMap[child.ReplyToID], child)
	}

	// 构建评论树节点
	nodes := make([]*LogCommentNode, len(parentComments))
	for i, parent := range parentComments {
		replies := childMap[parent.ID]
		if replies == nil {
			replies = []*model.DevLogComment{}
		}
		nodes[i] = &LogCommentNode{
			DevLogComment: *parent,
			Replies:       replies,
			ReplyCount:    int64(len(replies)),
		}
	}

	return nodes, total, nil
}

// SearchLogComments 日志评论全文搜索
func (r *devLogCommentRepository) SearchLogComments(ctx context.Context, logID uint64, query string) ([]*model.DevLogComment, error) {
	var comments []*model.DevLogComment

	keyword := "%" + query + "%"
	err := r.db.WithContext(ctx).
		Where("log_id = ? AND content ILIKE ? AND deleted_at IS NULL", logID, keyword).
		Order("created_at DESC").
		Find(&comments).Error

	if err != nil {
		return nil, fmt.Errorf("搜索日志评论失败: %w", err)
	}

	return comments, nil
}
