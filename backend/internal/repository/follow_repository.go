// Package repository 提供关注系统的数据库访问层。
// 实现对 user_follows 和 project_follows 表的所有操作。
// 采用接口 + 实现的模式，便于单元测试时替换 Mock 实现。
package repository

import (
	"context"
	"fmt"
	"strings"

	"github.com/gamero/gamero/internal/model"
	apperrors "github.com/gamero/gamero/pkg/errors"
	"gorm.io/gorm"
)

// FollowRepository 关注系统仓储接口
// 整合用户关注和项目关注的所有数据操作
type FollowRepository interface {
	// ===== 用户关注操作（操作 user_follows 表，对接已有 UserFollow 模型）=====

	// FollowUser 创建用户关注关系（followerID 关注 followeeID）
	FollowUser(ctx context.Context, followerID, followeeID uint64) error
	// UnfollowUser 删除用户关注关系（followerID 取关 followeeID）
	UnfollowUser(ctx context.Context, followerID, followeeID uint64) error
	// IsFollowingUser 判断 followerID 是否关注了 followeeID
	IsFollowingUser(ctx context.Context, followerID, followeeID uint64) (bool, error)
	// GetFollowers 获取 userID 的粉丝列表（谁关注了我），分页返回 UserFollow 记录
	GetFollowers(ctx context.Context, userID uint64, offset, limit int) ([]*model.UserFollow, int64, error)
	// GetFollowing 获取 userID 关注的列表（我关注了谁），分页返回 UserFollow 记录
	GetFollowing(ctx context.Context, userID uint64, offset, limit int) ([]*model.UserFollow, int64, error)
	// GetFollowingIDs 获取 followerID 关注的所有用户 ID（用于动态流查询，不分页）
	GetFollowingIDs(ctx context.Context, followerID uint64) ([]uint64, error)
	// GetFollowerIDsUpToLimit 获取 userID 的粉丝 ID 列表，最多返回 limit 条（用于写扩散）
	// limit 通常设为 FanoutThreshold（1000），超过此数说明是大V，应走读扩散
	GetFollowerIDsUpToLimit(ctx context.Context, userID uint64, limit int) ([]uint64, error)

	// ===== 项目关注操作（操作 project_follows 表）=====

	// FollowProject 创建项目关注关系（userID 关注 projectID）
	FollowProject(ctx context.Context, userID, projectID uint64) error
	// UnfollowProject 删除项目关注关系（userID 取关 projectID）
	UnfollowProject(ctx context.Context, userID, projectID uint64) error
	// IsFollowingProject 判断 userID 是否关注了 projectID
	IsFollowingProject(ctx context.Context, userID, projectID uint64) (bool, error)
	// GetProjectFollowers 获取关注 projectID 的用户列表，分页返回 ProjectFollow 记录
	GetProjectFollowers(ctx context.Context, projectID uint64, offset, limit int) ([]*model.ProjectFollow, int64, error)
	// GetProjectFollowerIDsUpToLimit 获取关注 projectID 的用户 ID 列表，最多返回 limit 条
	// 用于项目状态变更后向关注者写通知（limit 防止关注者极多时一次性操作过大）
	GetProjectFollowerIDsUpToLimit(ctx context.Context, projectID uint64, limit int) ([]uint64, error)
	// GetUserFollowerIDsUpToLimit 获取关注 userID 的用户 ID 列表，最多返回 limit 条
	// 用于帖子发布后向粉丝做 Feed 写扩散
	GetUserFollowerIDsUpToLimit(ctx context.Context, userID uint64, limit int) ([]uint64, error)
	// GetFollowingProjectIDs 获取 userID 关注的所有项目 ID（用于动态流查询，不分页）
	GetFollowingProjectIDs(ctx context.Context, userID uint64) ([]uint64, error)

	// GetFollowingIDsPartitioned 获取 followerID 关注的用户，按粉丝数分为普通（写扩散）和大V（读合并）两组
	// threshold：粉丝数阈值（通常为 cache.FanoutThreshold）
	GetFollowingIDsPartitioned(ctx context.Context, followerID uint64, threshold int) (normalIDs, heavyIDs []uint64, err error)

	// ===== 话题关注操作（操作 topic_follows 表）=====

	// FollowTopic 创建话题关注关系（userID 关注 topicID）
	FollowTopic(ctx context.Context, userID, topicID uint64) error
	// UnfollowTopic 删除话题关注关系（userID 取关 topicID）
	UnfollowTopic(ctx context.Context, userID, topicID uint64) error
	// IsFollowingTopic 判断 userID 是否关注了 topicID
	IsFollowingTopic(ctx context.Context, userID, topicID uint64) (bool, error)
	// GetFollowingTopicIDs 获取 userID 关注的所有话题 ID
	GetFollowingTopicIDs(ctx context.Context, userID uint64) ([]uint64, error)
	// GetFollowedTopics 获取 userID 关注的话题列表，分页返回
	GetFollowedTopics(ctx context.Context, userID uint64, offset, limit int) ([]*model.Topic, int64, error)
	// IncrTopicFollowerCount 更新话题关注人数冗余字段（delta 为 +1 或 -1）
	IncrTopicFollowerCount(ctx context.Context, topicID uint64, delta int) error
}

// followRepository 是 FollowRepository 接口的具体实现
type followRepository struct {
	db *gorm.DB
}

// NewFollowRepository 创建 followRepository 实例
func NewFollowRepository(db *gorm.DB) FollowRepository {
	return &followRepository{db: db}
}

// ==================== 用户关注操作实现 ====================

// FollowUser 创建用户关注关系
// 如果已关注则返回冲突错误
func (r *followRepository) FollowUser(ctx context.Context, followerID, followeeID uint64) error {
	follow := &model.UserFollow{
		FollowerID: followerID,
		FollowedID: followeeID,
	}
	if err := r.db.WithContext(ctx).Create(follow).Error; err != nil {
		// 捕获唯一索引冲突
		if strings.Contains(err.Error(), "duplicate") || strings.Contains(err.Error(), "unique") {
			return apperrors.CodeError(apperrors.CodeFollowAlreadyExists)
		}
		return fmt.Errorf("关注用户失败: %w", err)
	}
	return nil
}

// UnfollowUser 删除用户关注关系
// 如果本来没有关注，则返回未找到错误
func (r *followRepository) UnfollowUser(ctx context.Context, followerID, followeeID uint64) error {
	result := r.db.WithContext(ctx).
		Where("follower_id = ? AND followed_id = ?", followerID, followeeID).
		Delete(&model.UserFollow{})
	if result.Error != nil {
		return fmt.Errorf("取关用户失败: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return apperrors.CodeError(apperrors.CodeFollowNotFound)
	}
	return nil
}

// IsFollowingUser 判断 followerID 是否已关注 followeeID
func (r *followRepository) IsFollowingUser(ctx context.Context, followerID, followeeID uint64) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&model.UserFollow{}).
		Where("follower_id = ? AND followed_id = ?", followerID, followeeID).
		Count(&count).Error
	if err != nil {
		return false, fmt.Errorf("查询关注状态失败: %w", err)
	}
	return count > 0, nil
}

// GetFollowers 获取 userID 的粉丝列表（谁关注了 userID）
// 按关注时间倒序，支持分页
func (r *followRepository) GetFollowers(ctx context.Context, userID uint64, offset, limit int) ([]*model.UserFollow, int64, error) {
	var total int64
	if err := r.db.WithContext(ctx).
		Model(&model.UserFollow{}).
		Where("followed_id = ?", userID).
		Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("统计粉丝数失败: %w", err)
	}

	var records []*model.UserFollow
	if err := r.db.WithContext(ctx).
		Where("followed_id = ?", userID).
		Order("created_at DESC").
		Offset(offset).Limit(limit).
		Find(&records).Error; err != nil {
		return nil, 0, fmt.Errorf("查询粉丝列表失败: %w", err)
	}
	return records, total, nil
}

// GetFollowing 获取 userID 关注的用户列表（userID 关注了谁）
// 按关注时间倒序，支持分页
func (r *followRepository) GetFollowing(ctx context.Context, userID uint64, offset, limit int) ([]*model.UserFollow, int64, error) {
	var total int64
	if err := r.db.WithContext(ctx).
		Model(&model.UserFollow{}).
		Where("follower_id = ?", userID).
		Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("统计关注数失败: %w", err)
	}

	var records []*model.UserFollow
	if err := r.db.WithContext(ctx).
		Where("follower_id = ?", userID).
		Order("created_at DESC").
		Offset(offset).Limit(limit).
		Find(&records).Error; err != nil {
		return nil, 0, fmt.Errorf("查询关注列表失败: %w", err)
	}
	return records, total, nil
}

// GetFollowingIDs 获取 followerID 关注的所有用户 ID（不分页，用于动态流）
func (r *followRepository) GetFollowingIDs(ctx context.Context, followerID uint64) ([]uint64, error) {
	type idRow struct {
		FollowedID uint64
	}
	var rows []idRow
	if err := r.db.WithContext(ctx).
		Model(&model.UserFollow{}).
		Select("followed_id").
		Where("follower_id = ?", followerID).
		Scan(&rows).Error; err != nil {
		return nil, fmt.Errorf("查询关注用户 ID 列表失败: %w", err)
	}
	ids := make([]uint64, 0, len(rows))
	for _, row := range rows {
		ids = append(ids, row.FollowedID)
	}
	return ids, nil
}

// GetFollowerIDsUpToLimit 获取 userID 的粉丝 ID 列表，最多返回 limit 条。
// 用于写扩散场景：调用方传入 FanoutThreshold（1000），若实际粉丝数超过 limit，
// 说明是大V，写扩散将不覆盖全部粉丝（大V走读扩散）。
func (r *followRepository) GetFollowerIDsUpToLimit(ctx context.Context, userID uint64, limit int) ([]uint64, error) {
	type idRow struct {
		FollowerID uint64
	}
	var rows []idRow
	if err := r.db.WithContext(ctx).
		Model(&model.UserFollow{}).
		Select("follower_id").
		Where("followed_id = ?", userID).
		Limit(limit).
		Scan(&rows).Error; err != nil {
		return nil, fmt.Errorf("查询粉丝 ID 列表失败: %w", err)
	}
	ids := make([]uint64, 0, len(rows))
	for _, row := range rows {
		ids = append(ids, row.FollowerID)
	}
	return ids, nil
}

// ==================== 项目关注操作实现 ====================

// FollowProject 创建项目关注关系
// 如果已关注则返回冲突错误
func (r *followRepository) FollowProject(ctx context.Context, userID, projectID uint64) error {
	follow := &model.ProjectFollow{
		UserID:    userID,
		ProjectID: projectID,
	}
	if err := r.db.WithContext(ctx).Create(follow).Error; err != nil {
		// 捕获唯一索引冲突
		if strings.Contains(err.Error(), "duplicate") || strings.Contains(err.Error(), "unique") {
			return apperrors.CodeError(apperrors.CodeFollowAlreadyExists)
		}
		return fmt.Errorf("关注项目失败: %w", err)
	}
	return nil
}

// UnfollowProject 删除项目关注关系
// 如果本来没有关注，则返回未找到错误
func (r *followRepository) UnfollowProject(ctx context.Context, userID, projectID uint64) error {
	result := r.db.WithContext(ctx).
		Where("user_id = ? AND project_id = ?", userID, projectID).
		Delete(&model.ProjectFollow{})
	if result.Error != nil {
		return fmt.Errorf("取关项目失败: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return apperrors.CodeError(apperrors.CodeFollowNotFound)
	}
	return nil
}

// IsFollowingProject 判断 userID 是否已关注 projectID
func (r *followRepository) IsFollowingProject(ctx context.Context, userID, projectID uint64) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&model.ProjectFollow{}).
		Where("user_id = ? AND project_id = ?", userID, projectID).
		Count(&count).Error
	if err != nil {
		return false, fmt.Errorf("查询项目关注状态失败: %w", err)
	}
	return count > 0, nil
}

// GetProjectFollowers 获取关注 projectID 的用户列表
// 按关注时间倒序，支持分页
func (r *followRepository) GetProjectFollowers(ctx context.Context, projectID uint64, offset, limit int) ([]*model.ProjectFollow, int64, error) {
	var total int64
	if err := r.db.WithContext(ctx).
		Model(&model.ProjectFollow{}).
		Where("project_id = ?", projectID).
		Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("统计项目关注者数失败: %w", err)
	}

	var records []*model.ProjectFollow
	if err := r.db.WithContext(ctx).
		Where("project_id = ?", projectID).
		Order("created_at DESC").
		Offset(offset).Limit(limit).
		Find(&records).Error; err != nil {
		return nil, 0, fmt.Errorf("查询项目关注者列表失败: %w", err)
	}
	return records, total, nil
}

// GetProjectFollowerIDsUpToLimit 获取关注 projectID 的用户 ID 列表，最多返回 limit 条。
// 按关注时间正序取最早的 limit 位关注者，用于项目状态变更通知写入。
func (r *followRepository) GetProjectFollowerIDsUpToLimit(ctx context.Context, projectID uint64, limit int) ([]uint64, error) {
	type idRow struct {
		UserID uint64
	}
	var rows []idRow
	if err := r.db.WithContext(ctx).
		Model(&model.ProjectFollow{}).
		Select("user_id").
		Where("project_id = ?", projectID).
		Order("created_at ASC").
		Limit(limit).
		Scan(&rows).Error; err != nil {
		return nil, fmt.Errorf("查询项目关注者 ID 列表失败: %w", err)
	}
	ids := make([]uint64, 0, len(rows))
	for _, row := range rows {
		ids = append(ids, row.UserID)
	}
	return ids, nil
}

// GetUserFollowerIDsUpToLimit 获取关注 userID 的用户 ID 列表，最多返回 limit 条。
// 按关注时间正序取最早的 limit 位粉丝，用于帖子发布 Feed 写扩散。
func (r *followRepository) GetUserFollowerIDsUpToLimit(ctx context.Context, userID uint64, limit int) ([]uint64, error) {
	type idRow struct {
		FollowerID uint64
	}
	var rows []idRow
	if err := r.db.WithContext(ctx).
		Model(&model.UserFollow{}).
		Select("follower_id").
		Where("followed_id = ?", userID).
		Order("created_at ASC").
		Limit(limit).
		Scan(&rows).Error; err != nil {
		return nil, fmt.Errorf("查询用户粉丝 ID 列表失败: %w", err)
	}
	ids := make([]uint64, 0, len(rows))
	for _, row := range rows {
		ids = append(ids, row.FollowerID)
	}
	return ids, nil
}

// GetFollowingProjectIDs 获取 userID 关注的所有项目 ID（不分页，用于动态流）
func (r *followRepository) GetFollowingProjectIDs(ctx context.Context, userID uint64) ([]uint64, error) {
	type idRow struct {
		ProjectID uint64
	}
	var rows []idRow
	if err := r.db.WithContext(ctx).
		Model(&model.ProjectFollow{}).
		Select("project_id").
		Where("user_id = ?", userID).
		Scan(&rows).Error; err != nil {
		return nil, fmt.Errorf("查询关注项目 ID 列表失败: %w", err)
	}
	ids := make([]uint64, 0, len(rows))
	for _, row := range rows {
		ids = append(ids, row.ProjectID)
	}
	return ids, nil
}

// GetFollowingIDsPartitioned 获取 followerID 关注的用户，按粉丝数分为普通（写扩散）和大V（读合并）两组
func (r *followRepository) GetFollowingIDsPartitioned(ctx context.Context, followerID uint64, threshold int) (normalIDs, heavyIDs []uint64, err error) {
	type row struct {
		FolloweeID    uint64 `gorm:"column:followee_id"`
		FollowerCount int64  `gorm:"column:follower_count"`
	}
	var rows []row
	err = r.db.WithContext(ctx).
		Table("user_follows uf").
		Select("uf.followee_id, u.follower_count").
		Joins("JOIN users u ON u.id = uf.followee_id").
		Where("uf.follower_id = ? AND uf.deleted_at IS NULL", followerID).
		Scan(&rows).Error
	if err != nil {
		return nil, nil, err
	}
	for _, row := range rows {
		if int(row.FollowerCount) >= threshold {
			heavyIDs = append(heavyIDs, row.FolloweeID)
		} else {
			normalIDs = append(normalIDs, row.FolloweeID)
		}
	}
	return normalIDs, heavyIDs, nil
}

// ===========================
// Topic 关注操作实现
// ===========================

// FollowTopic 创建话题关注关系，已存在则幂等返回成功
func (r *followRepository) FollowTopic(ctx context.Context, userID, topicID uint64) error {
	record := &model.TopicFollow{TopicID: topicID, UserID: userID}
	result := r.db.WithContext(ctx).
		Where(model.TopicFollow{TopicID: topicID, UserID: userID}).
		FirstOrCreate(record)
	return result.Error
}

// UnfollowTopic 删除话题关注关系
func (r *followRepository) UnfollowTopic(ctx context.Context, userID, topicID uint64) error {
	return r.db.WithContext(ctx).
		Where("topic_id = ? AND user_id = ?", topicID, userID).
		Delete(&model.TopicFollow{}).Error
}

// IsFollowingTopic 判断用户是否关注了某话题
func (r *followRepository) IsFollowingTopic(ctx context.Context, userID, topicID uint64) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&model.TopicFollow{}).
		Where("topic_id = ? AND user_id = ?", topicID, userID).
		Count(&count).Error
	return count > 0, err
}

// GetFollowingTopicIDs 获取用户关注的所有话题 ID
func (r *followRepository) GetFollowingTopicIDs(ctx context.Context, userID uint64) ([]uint64, error) {
	type idRow struct{ TopicID uint64 }
	var rows []idRow
	err := r.db.WithContext(ctx).
		Model(&model.TopicFollow{}).
		Select("topic_id").
		Where("user_id = ?", userID).
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	ids := make([]uint64, 0, len(rows))
	for _, row := range rows {
		ids = append(ids, row.TopicID)
	}
	return ids, nil
}

// GetFollowedTopics 获取用户关注的话题列表（分页），按关注时间倒序
func (r *followRepository) GetFollowedTopics(ctx context.Context, userID uint64, offset, limit int) ([]*model.Topic, int64, error) {
	var total int64
	if err := r.db.WithContext(ctx).
		Model(&model.TopicFollow{}).
		Where("user_id = ?", userID).
		Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if total == 0 {
		return []*model.Topic{}, 0, nil
	}

	type joinRow struct {
		model.Topic
		FollowedAt string `gorm:"column:followed_at"`
	}
	var rows []joinRow
	err := r.db.WithContext(ctx).
		Table("topic_follows tf").
		Select("t.*, tf.created_at AS followed_at").
		Joins("JOIN topics t ON t.id = tf.topic_id").
		Where("tf.user_id = ?", userID).
		Order("tf.created_at DESC").
		Offset(offset).Limit(limit).
		Scan(&rows).Error
	if err != nil {
		return nil, 0, err
	}
	topics := make([]*model.Topic, 0, len(rows))
	for i := range rows {
		topics = append(topics, &rows[i].Topic)
	}
	return topics, total, nil
}

// IncrTopicFollowerCount 原子更新话题关注人数冗余字段
func (r *followRepository) IncrTopicFollowerCount(ctx context.Context, topicID uint64, delta int) error {
	return r.db.WithContext(ctx).
		Model(&model.Topic{}).
		Where("id = ?", topicID).
		UpdateColumn("follower_count", gorm.Expr("follower_count + ?", delta)).Error
}
