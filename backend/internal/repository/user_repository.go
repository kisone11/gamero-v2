// Package repository 提供数据库访问层，封装所有 GORM 操作。
// 采用接口 + 实现的模式，便于单元测试时替换 Mock 实现。
package repository

import (
	"context"
	"errors"
	"time"

	"github.com/gamero/gamero/internal/model"
	apperrors "github.com/gamero/gamero/pkg/errors"
	"gorm.io/gorm"
)

// UserProjectSummary 用户主页展示的项目摘要
type UserProjectSummary struct {
	ID           uint64     `json:"id"`
	Name         string     `json:"name"`
	Slug         string     `json:"slug"`
	Genre        string     `json:"genre"`
	Status       string     `json:"status"`
	CoverKey     string     `json:"-"`
	CoverURL     string     `json:"cover_url"`
	MemberRole   string     `json:"member_role"`
	Contribution string     `json:"contribution"` // 贡献简介
	JoinedAt     time.Time  `json:"joined_at"`    // 加入时间
	LeftAt       *time.Time `json:"left_at"`      // 离开时间（nil=仍在职）
	IsActive     bool       `json:"is_active"`    // 是否仍在职
}

// UserRepository 用户仓储接口
// 定义所有对 User、UserSkill、Portfolio、UserFollow 表的数据操作
type UserRepository interface {
	// ===== User 基础操作 =====

	// CreateUser 创建新用户，返回包含自增 ID 的用户对象
	CreateUser(ctx context.Context, user *model.User) error
	// GetUserByID 根据 ID 查找用户（包含 Skills 和 Portfolio 预加载）
	GetUserByID(ctx context.Context, id uint64) (*model.User, error)
	// GetUsersByIDs 按 ID 列表批量获取用户（基础字段，不预加载关联）
	GetUsersByIDs(ctx context.Context, ids []uint64) ([]*model.User, error)
	// GetUserByUsername 根据用户名查找用户
	GetUserByUsername(ctx context.Context, username string) (*model.User, error)
	// GetUserByPhone 根据手机号查找用户
	GetUserByPhone(ctx context.Context, phone string) (*model.User, error)
	// GetUserByEmail 根据邮箱查找用户
	GetUserByEmail(ctx context.Context, email string) (*model.User, error)
	// UpdateUser 更新用户信息（只更新非零字段）
	UpdateUser(ctx context.Context, id uint64, updates map[string]interface{}) error
	// ExistsUsername 判断用户名是否已存在
	ExistsUsername(ctx context.Context, username string) (bool, error)
	// ExistsPhone 判断手机号是否已存在
	ExistsPhone(ctx context.Context, phone string) (bool, error)
	// ExistsEmail 判断邮箱是否已存在
	ExistsEmail(ctx context.Context, email string) (bool, error)
	// ExistsNickname 判断昵称是否已存在
	ExistsNickname(ctx context.Context, nickname string) (bool, error)

	// ===== UserSkill 技能操作 =====

	// GetSkillsByUserID 获取用户所有技能标签
	GetSkillsByUserID(ctx context.Context, userID uint64) ([]model.UserSkill, error)
	// ReplaceUserSkills 替换用户所有技能标签（先删后增，在事务中执行）
	ReplaceUserSkills(ctx context.Context, userID uint64, skills []model.UserSkill) error
	// CountCustomSkillsByUserID 统计用户自定义技能标签数量
	CountCustomSkillsByUserID(ctx context.Context, userID uint64) (int64, error)

	// ===== Portfolio 作品集操作 =====

	// GetPortfoliosByUserID 获取用户所有作品（按创建时间倒序）
	GetPortfoliosByUserID(ctx context.Context, userID uint64) ([]model.Portfolio, error)
	// GetPortfolioByID 根据 ID 获取单个作品
	GetPortfolioByID(ctx context.Context, id uint64) (*model.Portfolio, error)
	// CreatePortfolio 创建作品
	CreatePortfolio(ctx context.Context, portfolio *model.Portfolio) error
	// UpdatePortfolio 更新作品（只更新非零字段）
	UpdatePortfolio(ctx context.Context, id uint64, updates map[string]interface{}) error
	// DeletePortfolio 删除作品
	DeletePortfolio(ctx context.Context, id uint64) error

	// ===== UserFollow 关注统计 =====

	// CountFollowers 统计被关注数（粉丝数）
	CountFollowers(ctx context.Context, userID uint64) (int64, error)
	// CountFollowing 统计关注数
	CountFollowing(ctx context.Context, userID uint64) (int64, error)

	// ===== 用户主页聚合查询 =====

	// GetUserProjectSummary 查询用户参与的项目摘要列表（联表 project_members + projects）
	GetUserProjectSummary(ctx context.Context, userID uint64, limit int) ([]UserProjectSummary, error)
	// GetUserRecentLogs 查询用户发布的公开日志（status=published, visibility=public）
	GetUserRecentLogs(ctx context.Context, userID uint64, limit int) ([]*model.DevLog, error)
	// GetUserRecentPosts 查询用户发布的帖子
	GetUserRecentPosts(ctx context.Context, userID uint64, limit int) ([]*model.Post, error)
	// CountUserPosts 统计用户帖子总数
	CountUserPosts(ctx context.Context, userID uint64) (int, error)
	// CountUserLogs 统计用户日志总数
	CountUserLogs(ctx context.Context, userID uint64) (int, error)
	// CountUserProjects 统计用户参与项目总数
	CountUserProjects(ctx context.Context, userID uint64) (int, error)

	// SearchUsersByKeyword 按 username / nickname 模糊匹配搜索用户
	SearchUsersByKeyword(ctx context.Context, keyword string, limit int) ([]*model.User, error)
}

// userRepository 是 UserRepository 接口的具体实现
type userRepository struct {
	db *gorm.DB
}

// NewUserRepository 创建 userRepository 实例
func NewUserRepository(db *gorm.DB) UserRepository {
	return &userRepository{db: db}
}

// ==================== User 基础操作实现 ====================

// CreateUser 创建新用户
func (r *userRepository) CreateUser(ctx context.Context, user *model.User) error {
	result := r.db.WithContext(ctx).Create(user)
	if result.Error != nil {
		return result.Error
	}
	return nil
}

// GetUserByID 根据 ID 查找用户，同时预加载 Skills 和 Portfolio
func (r *userRepository) GetUserByID(ctx context.Context, id uint64) (*model.User, error) {
	var user model.User
	result := r.db.WithContext(ctx).
		Preload("Skills").
		Preload("Portfolio").
		First(&user, id)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, apperrors.CodeError(apperrors.CodeUserNotFound)
		}
		return nil, result.Error
	}
	return &user, nil
}

// GetUsersByIDs 按 ID 列表批量获取用户（基础字段，不预加载关联）
// 用于列表展示场景（如拉黑列表、推荐用户等），避免 N+1 查询。
func (r *userRepository) GetUsersByIDs(ctx context.Context, ids []uint64) ([]*model.User, error) {
	if len(ids) == 0 {
		return []*model.User{}, nil
	}
	var users []*model.User
	if err := r.db.WithContext(ctx).
		Where("id IN ?", ids).
		Find(&users).Error; err != nil {
		return nil, err
	}
	return users, nil
}

func (r *userRepository) GetUserByUsername(ctx context.Context, username string) (*model.User, error) {
	var user model.User
	result := r.db.WithContext(ctx).
		Preload("Skills").
		Preload("Portfolio").
		Where("username = ?", username).
		First(&user)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, apperrors.CodeError(apperrors.CodeUserNotFound)
		}
		return nil, result.Error
	}
	return &user, nil
}

// GetUserByPhone 根据手机号查找用户
func (r *userRepository) GetUserByPhone(ctx context.Context, phone string) (*model.User, error) {
	var user model.User
	result := r.db.WithContext(ctx).
		Where("phone = ?", phone).
		First(&user)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, apperrors.CodeError(apperrors.CodeUserNotFound)
		}
		return nil, result.Error
	}
	return &user, nil
}

// GetUserByEmail 根据邮箱查找用户
func (r *userRepository) GetUserByEmail(ctx context.Context, email string) (*model.User, error) {
	var user model.User
	result := r.db.WithContext(ctx).
		Where("email = ?", email).
		First(&user)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, apperrors.CodeError(apperrors.CodeUserNotFound)
		}
		return nil, result.Error
	}
	return &user, nil
}

// UpdateUser 更新用户信息，只更新 updates 中指定的字段
func (r *userRepository) UpdateUser(ctx context.Context, id uint64, updates map[string]interface{}) error {
	result := r.db.WithContext(ctx).
		Model(&model.User{}).
		Where("id = ?", id).
		Updates(updates)
	if result.Error != nil {
		return result.Error
	}
	return nil
}

// ExistsUsername 判断用户名是否已存在
func (r *userRepository) ExistsUsername(ctx context.Context, username string) (bool, error) {
	var count int64
	result := r.db.WithContext(ctx).
		Model(&model.User{}).
		Where("username = ?", username).
		Count(&count)
	if result.Error != nil {
		return false, result.Error
	}
	return count > 0, nil
}

// ExistsPhone 判断手机号是否已存在
func (r *userRepository) ExistsPhone(ctx context.Context, phone string) (bool, error) {
	var count int64
	result := r.db.WithContext(ctx).
		Model(&model.User{}).
		Where("phone = ?", phone).
		Count(&count)
	if result.Error != nil {
		return false, result.Error
	}
	return count > 0, nil
}

// ExistsEmail 判断邮箱是否已存在
func (r *userRepository) ExistsEmail(ctx context.Context, email string) (bool, error) {
	var count int64
	result := r.db.WithContext(ctx).
		Model(&model.User{}).
		Where("email = ?", email).
		Count(&count)
	if result.Error != nil {
		return false, result.Error
	}
	return count > 0, nil
}

// ExistsNickname 判断昵称是否已存在
func (r *userRepository) ExistsNickname(ctx context.Context, nickname string) (bool, error) {
	var count int64
	result := r.db.WithContext(ctx).
		Model(&model.User{}).
		Where("nickname = ?", nickname).
		Count(&count)
	if result.Error != nil {
		return false, result.Error
	}
	return count > 0, nil
}

// ==================== UserSkill 技能操作实现 ====================

// GetSkillsByUserID 获取用户所有技能标签
func (r *userRepository) GetSkillsByUserID(ctx context.Context, userID uint64) ([]model.UserSkill, error) {
	var skills []model.UserSkill
	result := r.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Order("created_at ASC").
		Find(&skills)
	if result.Error != nil {
		return nil, result.Error
	}
	return skills, nil
}

// ReplaceUserSkills 原子性替换用户所有技能标签
// 在事务中先删除当前所有技能，再批量插入新技能
func (r *userRepository) ReplaceUserSkills(ctx context.Context, userID uint64, skills []model.UserSkill) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 删除该用户所有现有技能
		if err := tx.Where("user_id = ?", userID).Delete(&model.UserSkill{}).Error; err != nil {
			return err
		}
		// 如果新技能列表不为空，批量插入
		if len(skills) > 0 {
			if err := tx.Create(&skills).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

// CountCustomSkillsByUserID 统计用户自定义技能标签数量
func (r *userRepository) CountCustomSkillsByUserID(ctx context.Context, userID uint64) (int64, error) {
	var count int64
	result := r.db.WithContext(ctx).
		Model(&model.UserSkill{}).
		Where("user_id = ? AND is_custom = true", userID).
		Count(&count)
	if result.Error != nil {
		return 0, result.Error
	}
	return count, nil
}

// ==================== Portfolio 作品集操作实现 ====================

// GetPortfoliosByUserID 获取用户所有作品，按创建时间倒序
func (r *userRepository) GetPortfoliosByUserID(ctx context.Context, userID uint64) ([]model.Portfolio, error) {
	var portfolios []model.Portfolio
	result := r.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Order("created_at DESC").
		Find(&portfolios)
	if result.Error != nil {
		return nil, result.Error
	}
	return portfolios, nil
}

// GetPortfolioByID 根据 ID 获取单个作品
func (r *userRepository) GetPortfolioByID(ctx context.Context, id uint64) (*model.Portfolio, error) {
	var portfolio model.Portfolio
	result := r.db.WithContext(ctx).First(&portfolio, id)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, apperrors.CodeError(apperrors.CodeResourceNotFound)
		}
		return nil, result.Error
	}
	return &portfolio, nil
}

// CreatePortfolio 创建作品条目
func (r *userRepository) CreatePortfolio(ctx context.Context, portfolio *model.Portfolio) error {
	result := r.db.WithContext(ctx).Create(portfolio)
	if result.Error != nil {
		return result.Error
	}
	return nil
}

// UpdatePortfolio 更新作品信息，只更新 updates 中指定的字段
func (r *userRepository) UpdatePortfolio(ctx context.Context, id uint64, updates map[string]interface{}) error {
	result := r.db.WithContext(ctx).
		Model(&model.Portfolio{}).
		Where("id = ?", id).
		Updates(updates)
	if result.Error != nil {
		return result.Error
	}
	return nil
}

// DeletePortfolio 删除作品条目
func (r *userRepository) DeletePortfolio(ctx context.Context, id uint64) error {
	result := r.db.WithContext(ctx).Delete(&model.Portfolio{}, id)
	if result.Error != nil {
		return result.Error
	}
	return nil
}

// ==================== UserFollow 关注统计实现 ====================

// CountFollowers 统计用户的粉丝（被关注）数
func (r *userRepository) CountFollowers(ctx context.Context, userID uint64) (int64, error) {
	var count int64
	result := r.db.WithContext(ctx).
		Model(&model.UserFollow{}).
		Where("followed_id = ?", userID).
		Count(&count)
	if result.Error != nil {
		return 0, result.Error
	}
	return count, nil
}

// CountFollowing 统计用户关注的人数
func (r *userRepository) CountFollowing(ctx context.Context, userID uint64) (int64, error) {
	var count int64
	result := r.db.WithContext(ctx).
		Model(&model.UserFollow{}).
		Where("follower_id = ?", userID).
		Count(&count)
	if result.Error != nil {
		return 0, result.Error
	}
	return count, nil
}

// ==================== 用户主页聚合查询实现 ====================

// GetUserProjectSummary 查询用户参与的项目摘要列表（含历史离职记录）
// 按加入时间倒序，最多返回 limit 条（在职优先，之后按加入时间倒序）
func (r *userRepository) GetUserProjectSummary(ctx context.Context, userID uint64, limit int) ([]UserProjectSummary, error) {
	var results []UserProjectSummary
	err := r.db.WithContext(ctx).
		Table("project_members pm").
		Select("p.id, p.name, p.slug, p.genre, p.status, p.cover_key, "+
			"pm.role AS member_role, pm.contribution, pm.joined_at, pm.left_at, pm.is_active").
		Joins("JOIN projects p ON p.id = pm.project_id AND p.deleted_at IS NULL").
		Where("pm.user_id = ?", userID).
		Order("pm.is_active DESC, pm.joined_at DESC").
		Limit(limit).
		Scan(&results).Error
	if err != nil {
		return nil, err
	}
	if results == nil {
		results = []UserProjectSummary{}
	}
	return results, nil
}

// SearchUsersByKeyword 按 username / nickname 模糊匹配搜索用户（不含已注销账号）
func (r *userRepository) SearchUsersByKeyword(ctx context.Context, keyword string, limit int) ([]*model.User, error) {
	var users []*model.User
	like := "%" + keyword + "%"
	err := r.db.WithContext(ctx).
		Where("(username LIKE ? OR nickname LIKE ?)", like, like).
		Order("id ASC").
		Limit(limit).
		Find(&users).Error
	if err != nil {
		return nil, err
	}
	if users == nil {
		users = []*model.User{}
	}
	return users, nil
}

// GetUserRecentLogs 查询用户发布的公开日志（status=published, visibility=public），按发布时间倒序
func (r *userRepository) GetUserRecentLogs(ctx context.Context, userID uint64, limit int) ([]*model.DevLog, error) {
	var logs []*model.DevLog
	err := r.db.WithContext(ctx).
		Where("author_id = ? AND status = ? AND visibility = ? AND deleted_at IS NULL",
			userID, model.DevLogStatusPublished, model.DevLogVisibilityPublic).
		Order("created_at DESC").
		Limit(limit).
		Find(&logs).Error
	if err != nil {
		return nil, err
	}
	return logs, nil
}

// GetUserRecentPosts 查询用户发布的帖子
func (r *userRepository) GetUserRecentPosts(ctx context.Context, userID uint64, limit int) ([]*model.Post, error) {
	var posts []*model.Post
	err := r.db.WithContext(ctx).
		Where("author_id = ? AND deleted_at IS NULL").
		Order("created_at DESC").
		Limit(limit).
		Find(&posts).Error

	if err != nil {
		return nil, err
	}

	return posts, nil
}

// CountUserPosts 统计用户帖子总数
func (r *userRepository) CountUserPosts(ctx context.Context, userID uint64) (int, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&model.Post{}).
		Where("author_id = ? AND deleted_at IS NULL", userID).Count(&count).Error
	return int(count), err
}

// CountUserLogs 统计用户已发布的公开日志总数（排除草稿、私密日志和软删除）
func (r *userRepository) CountUserLogs(ctx context.Context, userID uint64) (int, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&model.DevLog{}).
		Where("author_id = ? AND status = ? AND visibility = ? AND deleted_at IS NULL",
			userID, model.DevLogStatusPublished, model.DevLogVisibilityPublic).Count(&count).Error
	return int(count), err
}

// CountUserProjects 统计用户参与项目总数（排除已删除和 abandoned 的项目）
func (r *userRepository) CountUserProjects(ctx context.Context, userID uint64) (int, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&model.ProjectMember{}).
		Joins("JOIN projects ON projects.id = project_members.project_id AND projects.deleted_at IS NULL").
		Where("project_members.user_id = ? AND project_members.left_at IS NULL", userID).
		Where("projects.status != ?", model.ProjectStatusAbandoned).
		Count(&count).Error
	return int(count), err
}
