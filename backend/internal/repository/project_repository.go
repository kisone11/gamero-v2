// Package repository 提供项目系统的数据库访问层。
// 采用接口 + 实现的模式,封装所有对 Project、ProjectMember、ProjectTimeline 表的操作。
package repository

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/gamero/gamero/internal/model"
	apperrors "github.com/gamero/gamero/pkg/errors"
	"gorm.io/gorm"
)

// ProjectRepository 项目仓储接口
type ProjectRepository interface {
	// ===== 项目基础操作 =====

	// CreateProject 创建项目,返回含 ID 的项目
	CreateProject(ctx context.Context, project *model.Project) error
	// GetProjectByID 根据 ID 查找项目(未删除)
	GetProjectByID(ctx context.Context, id uint64) (*model.Project, error)
	// GetProjectBySlug 根据 slug 查找项目
	GetProjectBySlug(ctx context.Context, slug string) (*model.Project, error)
	// UpdateProject 更新项目指定字段
	UpdateProject(ctx context.Context, id uint64, updates map[string]interface{}) error
	// SoftDeleteProject 软删除项目
	SoftDeleteProject(ctx context.Context, id uint64) error
	// ListProjects 分页查询项目列表,支持按状态/类型/关键词筛选
	ListProjects(ctx context.Context, params *ListProjectsParams) ([]*model.Project, int64, error)
	// ExistsSlug 判断 slug 是否已被占用
	ExistsSlug(ctx context.Context, slug string) (bool, error)
	// IncrFollowerCount 关注数 +1(原子操作)
	IncrFollowerCount(ctx context.Context, projectID uint64, delta int) error

	// ===== 高级查询 =====

	// ListProjectsByHotScore 按热度排行榜(关注数+收藏数+成员数综合排序)
	ListProjectsByHotScore(ctx context.Context, limit int) ([]*model.Project, error)
	// ListProjectsByRating 按评分排序(join game_review 聚合 avg)
	ListProjectsByRating(ctx context.Context, page, pageSize int) ([]*model.Project, []float64, error)
	// ListRecentlyActiveProjects 按更新时间(最近有日志更新的)
	ListRecentlyActiveProjects(ctx context.Context, days int, page, pageSize int) ([]*model.Project, int64, error)
	// GetTagStats 按标签聚合(返回每个 tag 及项目数)
	GetTagStats(ctx context.Context) ([]TagCount, error)
	// GetProjectFullStats 获取项目完整统计(成员数/日志数/关注数/评分)
	GetProjectFullStats(ctx context.Context, projectID uint64) (*ProjectFullStats, error)
	// GetProjectsByIDs 批量获取项目(IN 查询)
	GetProjectsByIDs(ctx context.Context, ids []uint64) ([]*model.Project, error)
	// ListProjectsByOwnerIDs batch query projects by owner IDs (for Feed)
	// Supports status filter, ordered by created_at DESC, max limit items.
	ListProjectsByOwnerIDs(ctx context.Context, ownerIDs []uint64, statuses []model.ProjectStatus, limit int) ([]*model.Project, error)
	// GetProjectIDsByOwnerID 获取指定用户创建的所有项目 ID 列表（用于 Feed）
	GetProjectIDsByOwnerID(ctx context.Context, ownerID uint64) ([]uint64, error)
	// ListUserProjects 查询用户参与的项目（owner + 活跃成员），分页
	ListUserProjects(ctx context.Context, userID uint64, offset, limit int) ([]*model.Project, int64, error)
	// ListProjectsAdvanced 高级筛选(状态+标签+成员数范围+排序)
	ListProjectsAdvanced(ctx context.Context, filter ProjectFilter) ([]*model.Project, int64, error)

	// ===== 成员操作 =====

	// AddMember 添加项目成员
	AddMember(ctx context.Context, member *model.ProjectMember) error
	// RemoveMember 移除项目成员(记录 left_at,保留历史履历,is_active 置 false)
	RemoveMember(ctx context.Context, projectID, userID uint64) error
	// LeaveProject 成员主动离开项目(同 RemoveMember,但由成员自行调用)
	LeaveProject(ctx context.Context, projectID, userID uint64) error
	// UpdateMember 更新成员角色和贡献
	UpdateMember(ctx context.Context, projectID, userID uint64, updates map[string]interface{}) error
	// GetMembers 获取项目所有成员(含用户信息预加载)
	GetMembers(ctx context.Context, projectID uint64) ([]*model.ProjectMember, error)
	// GetMemberByUserID 根据项目ID和用户ID查找成员
	GetMemberByUserID(ctx context.Context, projectID, userID uint64) (*model.ProjectMember, error)
	// IsMember 判断用户是否为项目成员
	IsMember(ctx context.Context, projectID, userID uint64) (bool, error)
	// IsOwner 判断用户是否为项目 owner
	IsOwner(ctx context.Context, projectID, ownerID uint64) (bool, error)
	// UpdateMemberRole 更新指定成员的角色
	UpdateMemberRole(ctx context.Context, projectID, userID uint64, role model.ProjectMemberRole) error
	// UpdateOwner 更新项目的 owner_id
	UpdateOwner(ctx context.Context, projectID, newOwnerID uint64) error

	// ===== 时间轴操作 =====

	// CreateTimelineEvent 创建时间轴事件
	CreateTimelineEvent(ctx context.Context, event *model.ProjectTimeline) error
	// GetTimeline 获取项目时间轴(按时间倒序,分页)
	GetTimeline(ctx context.Context, projectID uint64, offset, limit int) ([]*model.ProjectTimeline, int64, error)
	// ListTimelinesByProjectIDs 批量查询多个项目的时间轴事件(用于动态流)
	// 按 created_at 倒序,最多返回 limit 条
	ListTimelinesByProjectIDs(ctx context.Context, projectIDs []uint64, limit int) ([]*model.ProjectTimeline, error)
	// GetTimelineByID 根据 ID 查询单条时间轴事件(Feed 缓存路径回查 DB 用)
	GetTimelineByID(ctx context.Context, id uint64) (*model.ProjectTimeline, error)
	// GetTimelinesByIDs 按 ID 列表批量获取时间轴事件(Feed 批量渲染用)
	GetTimelinesByIDs(ctx context.Context, ids []uint64) ([]*model.ProjectTimeline, error)

	// ===== 收藏操作 =====

	// CollectProject 收藏项目(已收藏则静默成功)
	CollectProject(ctx context.Context, projectID, userID uint64) error
	// UncollectProject 取消收藏
	UncollectProject(ctx context.Context, projectID, userID uint64) error
	// IsCollected 判断用户是否已收藏
	IsCollected(ctx context.Context, projectID, userID uint64) (bool, error)
	// GetUserCollectedProjects 获取用户收藏的项目列表(分页)
	GetUserCollectedProjects(ctx context.Context, userID uint64, offset, limit int) ([]*model.Project, int64, error)

	// ===== 任务操作 =====
	ListTasks(ctx context.Context, projectID uint64) ([]*model.ProjectTask, error)
	CreateTask(ctx context.Context, task *model.ProjectTask) error
	GetTaskByID(ctx context.Context, projectID, taskID uint64) (*model.ProjectTask, error)
	UpdateTask(ctx context.Context, projectID, taskID uint64, updates map[string]interface{}) error
	DeleteTask(ctx context.Context, projectID, taskID uint64) error
}

// ListProjectsParams 项目列表查询参数
type ListProjectsParams struct {
	Keyword string              // 关键词（匹配项目名、简介）
	Status  model.ProjectStatus // 按状态筛选
	Genre   model.ProjectGenre  // 按游戏类型筛选
	OwnerID uint64              // 按创建者筛选
	SortBy  string              // 排序字段：newest（默认）/ hottest / updated
	Offset  int
	Limit   int
}

// ProjectFilter 高级筛选参数
type ProjectFilter struct {
	Status     string
	Tags       []string
	MinMembers int
	MaxMembers int
	SortBy     string // hot/rating/recent/followers
	Page       int
	PageSize   int
}

// ProjectFullStats 项目完整统计
type ProjectFullStats struct {
	MemberCount   int64
	LogCount      int64
	FollowerCount int64
	AvgRating     float64
	ReviewCount   int64
	CollectCount  int64
}

// TagCount 标签统计
type TagCount struct {
	Tag   string
	Count int64
}

// projectRepository 项目仓储实现
type projectRepository struct {
	db *gorm.DB
}

// NewProjectRepository 创建项目仓储实例
func NewProjectRepository(db *gorm.DB) ProjectRepository {
	return &projectRepository{db: db}
}

// ===== 项目基础操作实现 =====

// CreateProject 创建项目
func (r *projectRepository) CreateProject(ctx context.Context, project *model.Project) error {
	if err := r.db.WithContext(ctx).Create(project).Error; err != nil {
		return fmt.Errorf("创建项目失败: %w", err)
	}
	return nil
}

// GetProjectByID 根据 ID 查找项目
func (r *projectRepository) GetProjectByID(ctx context.Context, id uint64) (*model.Project, error) {
	var project model.Project
	err := r.db.WithContext(ctx).
		Where("id = ? AND deleted_at IS NULL", id).
		First(&project).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperrors.CodeError(apperrors.CodeProjectNotFound)
		}
		return nil, fmt.Errorf("查询项目失败: %w", err)
	}
	return &project, nil
}

// GetProjectBySlug 根据 slug 查找项目
func (r *projectRepository) GetProjectBySlug(ctx context.Context, slug string) (*model.Project, error) {
	var project model.Project
	err := r.db.WithContext(ctx).
		Where("slug = ? AND deleted_at IS NULL", slug).
		First(&project).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperrors.CodeError(apperrors.CodeProjectNotFound)
		}
		return nil, fmt.Errorf("查询项目失败: %w", err)
	}
	return &project, nil
}

// UpdateProject 更新项目字段
func (r *projectRepository) UpdateProject(ctx context.Context, id uint64, updates map[string]interface{}) error {
	result := r.db.WithContext(ctx).
		Model(&model.Project{}).
		Where("id = ? AND deleted_at IS NULL", id).
		Updates(updates)
	if result.Error != nil {
		return fmt.Errorf("更新项目失败: %w", result.Error)
	}
	return nil
}

// SoftDeleteProject 软删除项目(设置 deleted_at)
func (r *projectRepository) SoftDeleteProject(ctx context.Context, id uint64) error {
	result := r.db.WithContext(ctx).
		Model(&model.Project{}).
		Where("id = ? AND deleted_at IS NULL", id).
		Update("deleted_at", gorm.Expr("NOW()"))
	if result.Error != nil {
		return fmt.Errorf("删除项目失败: %w", result.Error)
	}
	return nil
}

// ListProjects 分页查询项目列表
func (r *projectRepository) ListProjects(ctx context.Context, params *ListProjectsParams) ([]*model.Project, int64, error) {
	query := r.db.WithContext(ctx).Model(&model.Project{}).Where("deleted_at IS NULL")

	// 关键词搜索
	if params.Keyword != "" {
		kw := "%" + params.Keyword + "%"
		query = query.Where("name ILIKE ? OR description ILIKE ?", kw, kw)
	}
	// 状态筛选
	if params.Status != "" {
		query = query.Where("status = ?", params.Status)
	}
	// 游戏类型筛选
	if params.Genre != "" {
		query = query.Where("genre = ?", params.Genre)
	}
	// 按创建者筛选
	if params.OwnerID > 0 {
		query = query.Where("owner_id = ?", params.OwnerID)
	}

	// 统计总数
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("统计项目总数失败: %w", err)
	}

	// 排序
	switch params.SortBy {
	case "hottest":
		query = query.Order("follower_count DESC, created_at DESC")
	case "updated":
		query = query.Order("updated_at DESC")
	default: // newest
		query = query.Order("created_at DESC")
	}

	var projects []*model.Project
	if err := query.Offset(params.Offset).Limit(params.Limit).Find(&projects).Error; err != nil {
		return nil, 0, fmt.Errorf("查询项目列表失败: %w", err)
	}
	return projects, total, nil
}

// ExistsSlug 判断 slug 是否已存在
func (r *projectRepository) ExistsSlug(ctx context.Context, slug string) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&model.Project{}).
		Where("slug = ? AND deleted_at IS NULL", slug).
		Count(&count).Error
	return count > 0, err
}

// IncrFollowerCount 关注数增减(原子操作)
func (r *projectRepository) IncrFollowerCount(ctx context.Context, projectID uint64, delta int) error {
	return r.db.WithContext(ctx).
		Model(&model.Project{}).
		Where("id = ?", projectID).
		Update("follower_count", gorm.Expr("follower_count + ?", delta)).Error
}

// ===== 任务操作实现 =====

func (r *projectRepository) ListTasks(ctx context.Context, projectID uint64) ([]*model.ProjectTask, error) {
	var tasks []*model.ProjectTask
	err := r.db.WithContext(ctx).
		Preload("Creator").
		Preload("Assignee").
		Where("project_id = ?", projectID).
		Order("created_at DESC").
		Find(&tasks).Error
	return tasks, err
}

func (r *projectRepository) CreateTask(ctx context.Context, task *model.ProjectTask) error {
	return r.db.WithContext(ctx).Create(task).Error
}

func (r *projectRepository) GetTaskByID(ctx context.Context, projectID, taskID uint64) (*model.ProjectTask, error) {
	var task model.ProjectTask
	err := r.db.WithContext(ctx).
		Preload("Creator").
		Preload("Assignee").
		Where("project_id = ? AND id = ?", projectID, taskID).
		First(&task).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, apperrors.CodeError(apperrors.CodeResourceNotFound)
	}
	return &task, err
}

func (r *projectRepository) UpdateTask(ctx context.Context, projectID, taskID uint64, updates map[string]interface{}) error {
	result := r.db.WithContext(ctx).
		Model(&model.ProjectTask{}).
		Where("project_id = ? AND id = ?", projectID, taskID).
		Updates(updates)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return apperrors.CodeError(apperrors.CodeResourceNotFound)
	}
	return nil
}

func (r *projectRepository) DeleteTask(ctx context.Context, projectID, taskID uint64) error {
	result := r.db.WithContext(ctx).
		Delete(&model.ProjectTask{}, "project_id = ? AND id = ?", projectID, taskID)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return apperrors.CodeError(apperrors.CodeResourceNotFound)
	}
	return nil
}

// ===== 成员操作实现 =====

// AddMember 添加项目成员
func (r *projectRepository) AddMember(ctx context.Context, member *model.ProjectMember) error {
	if err := r.db.WithContext(ctx).Create(member).Error; err != nil {
		if strings.Contains(err.Error(), "duplicate") || strings.Contains(err.Error(), "unique") {
			return apperrors.CodeError(apperrors.CodeMemberAlreadyExists)
		}
		return fmt.Errorf("添加成员失败: %w", err)
	}
	return nil
}

// RemoveMember 移除项目成员:记录 left_at,is_active=false,保留行数据作为履历
func (r *projectRepository) RemoveMember(ctx context.Context, projectID, userID uint64) error {
	return r.setMemberLeft(ctx, projectID, userID)
}

// LeaveProject 成员主动离开:与 RemoveMember 逻辑相同,由成员自行发起
func (r *projectRepository) LeaveProject(ctx context.Context, projectID, userID uint64) error {
	return r.setMemberLeft(ctx, projectID, userID)
}

// setMemberLeft 设置成员离开时间(内部共用)
func (r *projectRepository) setMemberLeft(ctx context.Context, projectID, userID uint64) error {
	now := time.Now()
	result := r.db.WithContext(ctx).
		Model(&model.ProjectMember{}).
		Where("project_id = ? AND user_id = ? AND is_active = true", projectID, userID).
		Updates(map[string]interface{}{
			"left_at":    now,
			"is_active":  false,
			"updated_at": now,
		})
	if result.Error != nil {
		return fmt.Errorf("更新成员离开时间失败: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return apperrors.CodeError(apperrors.CodeMemberNotFound)
	}
	return nil
}

// UpdateMember 更新成员信息
func (r *projectRepository) UpdateMember(ctx context.Context, projectID, userID uint64, updates map[string]interface{}) error {
	result := r.db.WithContext(ctx).
		Model(&model.ProjectMember{}).
		Where("project_id = ? AND user_id = ?", projectID, userID).
		Updates(updates)
	if result.Error != nil {
		return fmt.Errorf("更新成员失败: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return apperrors.CodeError(apperrors.CodeMemberNotFound)
	}
	return nil
}

// GetMembers 获取项目当前在职成员(is_active=true)
func (r *projectRepository) GetMembers(ctx context.Context, projectID uint64) ([]*model.ProjectMember, error) {
	var members []*model.ProjectMember
	err := r.db.WithContext(ctx).
		Where("project_id = ? AND is_active = true", projectID).
		Order("created_at ASC").
		Find(&members).Error
	if err != nil {
		return nil, fmt.Errorf("查询成员列表失败: %w", err)
	}
	return members, nil
}

// GetMemberByUserID 查找在职成员记录(is_active=true)
func (r *projectRepository) GetMemberByUserID(ctx context.Context, projectID, userID uint64) (*model.ProjectMember, error) {
	var member model.ProjectMember
	err := r.db.WithContext(ctx).
		Where("project_id = ? AND user_id = ? AND is_active = true", projectID, userID).
		First(&member).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperrors.CodeError(apperrors.CodeMemberNotFound)
		}
		return nil, fmt.Errorf("查询成员失败: %w", err)
	}
	return &member, nil
}

// IsMember 判断用户是否为项目成员
func (r *projectRepository) IsMember(ctx context.Context, projectID, userID uint64) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&model.ProjectMember{}).
		Where("project_id = ? AND user_id = ?", projectID, userID).
		Count(&count).Error
	return count > 0, err
}

// IsOwner 判断用户是否为项目 owner
func (r *projectRepository) IsOwner(ctx context.Context, projectID, ownerID uint64) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&model.Project{}).
		Where("id = ? AND owner_id = ? AND deleted_at IS NULL", projectID, ownerID).
		Count(&count).Error
	return count > 0, err
}

// ===== 时间轴操作实现 =====

// CreateTimelineEvent 创建时间轴事件
func (r *projectRepository) CreateTimelineEvent(ctx context.Context, event *model.ProjectTimeline) error {
	if err := r.db.WithContext(ctx).Create(event).Error; err != nil {
		return fmt.Errorf("创建时间轴事件失败: %w", err)
	}
	return nil
}

// GetTimeline 获取项目时间轴(按时间倒序,分页)
func (r *projectRepository) GetTimeline(ctx context.Context, projectID uint64, offset, limit int) ([]*model.ProjectTimeline, int64, error) {
	var total int64
	if err := r.db.WithContext(ctx).
		Model(&model.ProjectTimeline{}).
		Where("project_id = ?", projectID).
		Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("统计时间轴总数失败: %w", err)
	}

	var events []*model.ProjectTimeline
	err := r.db.WithContext(ctx).
		Where("project_id = ?", projectID).
		Order("created_at DESC").
		Offset(offset).Limit(limit).
		Find(&events).Error
	if err != nil {
		return nil, 0, fmt.Errorf("查询时间轴失败: %w", err)
	}
	return events, total, nil
}

// ListTimelinesByProjectIDs 批量查询多个项目的时间轴事件(用于动态流)
// 按 created_at 倒序,最多返回 limit 条
func (r *projectRepository) ListTimelinesByProjectIDs(ctx context.Context, projectIDs []uint64, limit int) ([]*model.ProjectTimeline, error) {
	if len(projectIDs) == 0 {
		return []*model.ProjectTimeline{}, nil
	}
	var events []*model.ProjectTimeline
	if err := r.db.WithContext(ctx).
		Where("project_id IN ?", projectIDs).
		Order("created_at DESC").
		Limit(limit).
		Find(&events).Error; err != nil {
		return nil, fmt.Errorf("查询动态流时间轴失败: %w", err)
	}
	return events, nil
}

// GetTimelineByID 根据 ID 查询单条时间轴事件
func (r *projectRepository) GetTimelineByID(ctx context.Context, id uint64) (*model.ProjectTimeline, error) {
	var event model.ProjectTimeline
	if err := r.db.WithContext(ctx).First(&event, id).Error; err != nil {
		return nil, err
	}
	return &event, nil
}

// GetTimelinesByIDs 按 ID 列表批量获取时间轴事件(Feed 批量渲染用)
func (r *projectRepository) GetTimelinesByIDs(ctx context.Context, ids []uint64) ([]*model.ProjectTimeline, error) {
	if len(ids) == 0 {
		return []*model.ProjectTimeline{}, nil
	}
	var events []*model.ProjectTimeline
	if err := r.db.WithContext(ctx).Where("id IN ?", ids).Find(&events).Error; err != nil {
		return nil, fmt.Errorf("批量查询时间轴事件失败: %w", err)
	}
	return events, nil
}

// UpdateMemberRole 更新指定成员的角色
func (r *projectRepository) UpdateMemberRole(ctx context.Context, projectID, userID uint64, role model.ProjectMemberRole) error {
	return r.db.WithContext(ctx).Model(&model.ProjectMember{}).
		Where("project_id = ? AND user_id = ? AND is_active = true", projectID, userID).
		Update("role", role).Error
}

// UpdateOwner 更新项目的 owner_id
func (r *projectRepository) UpdateOwner(ctx context.Context, projectID, newOwnerID uint64) error {
	return r.db.WithContext(ctx).Model(&model.Project{}).
		Where("id = ?", projectID).
		Update("owner_id", newOwnerID).Error
}

// ===== 收藏操作实现 =====

// CollectProject 收藏项目(已收藏则静默成功)
func (r *projectRepository) CollectProject(ctx context.Context, projectID, userID uint64) error {
	collect := &model.ProjectCollect{ProjectID: projectID, UserID: userID}
	result := r.db.WithContext(ctx).
		Where("project_id = ? AND user_id = ?", projectID, userID).
		FirstOrCreate(collect)
	return result.Error
}

// UncollectProject 取消收藏
func (r *projectRepository) UncollectProject(ctx context.Context, projectID, userID uint64) error {
	return r.db.WithContext(ctx).
		Where("project_id = ? AND user_id = ?", projectID, userID).
		Delete(&model.ProjectCollect{}).Error
}

// IsCollected 判断用户是否已收藏
func (r *projectRepository) IsCollected(ctx context.Context, projectID, userID uint64) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&model.ProjectCollect{}).
		Where("project_id = ? AND user_id = ?", projectID, userID).
		Count(&count).Error
	return count > 0, err
}

// GetUserCollectedProjects 获取用户收藏的项目列表(分页,按收藏时间倒序)
func (r *projectRepository) GetUserCollectedProjects(ctx context.Context, userID uint64, offset, limit int) ([]*model.Project, int64, error) {
	var total int64
	if err := r.db.WithContext(ctx).
		Model(&model.ProjectCollect{}).
		Where("user_id = ?", userID).
		Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var collects []model.ProjectCollect
	if err := r.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Order("created_at DESC").
		Offset(offset).Limit(limit).
		Find(&collects).Error; err != nil {
		return nil, 0, err
	}

	ids := make([]uint64, 0, len(collects))
	for _, c := range collects {
		ids = append(ids, c.ProjectID)
	}
	if len(ids) == 0 {
		return []*model.Project{}, total, nil
	}

	var projects []*model.Project
	if err := r.db.WithContext(ctx).
		Where("id IN ? AND deleted_at IS NULL", ids).
		Find(&projects).Error; err != nil {
		return nil, 0, err
	}
	return projects, total, nil
}

// ===== 高级查询实现 =====

// ListProjectsByHotScore 按热度排行榜(关注数+收藏数+成员数综合排序)
func (r *projectRepository) ListProjectsByHotScore(ctx context.Context, limit int) ([]*model.Project, error) {
	var projects []*model.Project
	// 热度分数 = 关注数*2 + 收藏数*1.5 + 成员数*0.5
	err := r.db.WithContext(ctx).
		Table("projects p").
		Select(`p.*, 
			(p.follower_count * 2 + 
			COALESCE((SELECT COUNT(*) FROM project_collects WHERE project_id = p.id), 0) * 1.5 + 
			COALESCE((SELECT COUNT(*) FROM project_members WHERE project_id = p.id AND is_active = true), 0) * 0.5
			) as hot_score`).
		Where("p.deleted_at IS NULL").
		Order("hot_score DESC, p.created_at DESC").
		Limit(limit).
		Find(&projects).Error
	if err != nil {
		return nil, fmt.Errorf("查询热门项目失败: %w", err)
	}
	return projects, nil
}

// ListProjectsByRating 按评分排序(join game_review 聚合 avg)
func (r *projectRepository) ListProjectsByRating(ctx context.Context, page, pageSize int) ([]*model.Project, []float64, error) {
	type ProjectWithRating struct {
		model.Project
		AvgRating float64
	}

	offset := (page - 1) * pageSize
	var results []ProjectWithRating

	err := r.db.WithContext(ctx).
		Table("projects p").
		Select(`p.*, COALESCE(AVG(gr.rating), 0) as avg_rating`).
		Joins("LEFT JOIN game_reviews gr ON gr.project_id = p.id").
		Where("p.deleted_at IS NULL").
		Group("p.id").
		Having("COUNT(gr.id) > 0"). // 只返回有评分的项目
		Order("avg_rating DESC, p.created_at DESC").
		Offset(offset).Limit(pageSize).
		Find(&results).Error

	if err != nil {
		return nil, nil, fmt.Errorf("查询评分排行榜失败: %w", err)
	}

	projects := make([]*model.Project, len(results))
	ratings := make([]float64, len(results))
	for i, r := range results {
		projects[i] = &r.Project
		ratings[i] = r.AvgRating
	}

	return projects, ratings, nil
}

// ListRecentlyActiveProjects 按更新时间(最近有日志更新的)
func (r *projectRepository) ListRecentlyActiveProjects(ctx context.Context, days int, page, pageSize int) ([]*model.Project, int64, error) {
	offset := (page - 1) * pageSize
	cutoffTime := time.Now().AddDate(0, 0, -days)

	// 统计总数
	var total int64
	if err := r.db.WithContext(ctx).
		Table("projects p").
		Joins("INNER JOIN dev_logs dl ON dl.project_id = p.id").
		Where("p.deleted_at IS NULL AND dl.created_at >= ?", cutoffTime).
		Group("p.id").
		Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("统计活跃项目总数失败: %w", err)
	}

	var projects []*model.Project
	err := r.db.WithContext(ctx).
		Table("projects p").
		Select("p.*, MAX(dl.created_at) as last_log_at").
		Joins("INNER JOIN dev_logs dl ON dl.project_id = p.id").
		Where("p.deleted_at IS NULL AND dl.created_at >= ?", cutoffTime).
		Group("p.id").
		Order("last_log_at DESC").
		Offset(offset).Limit(pageSize).
		Find(&projects).Error

	if err != nil {
		return nil, 0, fmt.Errorf("查询活跃项目失败: %w", err)
	}

	return projects, total, nil
}

// GetTagStats 按标签聚合(返回每个 tag 及项目数)
func (r *projectRepository) GetTagStats(ctx context.Context) ([]TagCount, error) {
	var tagCounts []TagCount
	// 假设 tags 存储为 JSON 数组或逗号分隔的字符串，这里使用 PostgreSQL 的 unnest 函数
	// 如果是 MySQL，需要使用不同的方法
	err := r.db.WithContext(ctx).
		Raw(`
			SELECT 
				tag,
				COUNT(*) as count
			FROM (
				SELECT unnest(string_to_array(tags, ',')) as tag
				FROM projects
				WHERE deleted_at IS NULL AND tags != '' AND tags IS NOT NULL
			) t
			GROUP BY tag
			ORDER BY count DESC
		`).Scan(&tagCounts).Error

	if err != nil {
		return nil, fmt.Errorf("统计标签失败: %w", err)
	}
	return tagCounts, nil
}

// GetProjectFullStats 获取项目完整统计
func (r *projectRepository) GetProjectFullStats(ctx context.Context, projectID uint64) (*ProjectFullStats, error) {
	var stats ProjectFullStats

	err := r.db.WithContext(ctx).Raw(`
		SELECT 
			(SELECT COUNT(*) FROM project_members WHERE project_id = ? AND is_active = true) as member_count,
			(SELECT COUNT(*) FROM dev_logs WHERE project_id = ?) as log_count,
			(SELECT follower_count FROM projects WHERE id = ?) as follower_count,
			(SELECT COALESCE(AVG(rating), 0) FROM game_reviews WHERE project_id = ?) as avg_rating,
			(SELECT COUNT(*) FROM game_reviews WHERE project_id = ?) as review_count,
			(SELECT COUNT(*) FROM project_collects WHERE project_id = ?) as collect_count
	`, projectID, projectID, projectID, projectID, projectID, projectID).Scan(&stats).Error

	if err != nil {
		return nil, fmt.Errorf("获取项目统计失败: %w", err)
	}

	return &stats, nil
}

// GetProjectsByIDs 批量获取项目(IN 查询)
func (r *projectRepository) GetProjectsByIDs(ctx context.Context, ids []uint64) ([]*model.Project, error) {
	if len(ids) == 0 {
		return []*model.Project{}, nil
	}

	var projects []*model.Project
	err := r.db.WithContext(ctx).
		Where("id IN ? AND deleted_at IS NULL", ids).
		Find(&projects).Error

	if err != nil {
		return nil, fmt.Errorf("批量查询项目失败: %w", err)
	}
	return projects, nil
}

// ListProjectsByOwnerIDs batch query projects by owner IDs (for Feed)
func (r *projectRepository) ListProjectsByOwnerIDs(ctx context.Context, ownerIDs []uint64, statuses []model.ProjectStatus, limit int) ([]*model.Project, error) {
	if len(ownerIDs) == 0 {
		return []*model.Project{}, nil
	}
	var projects []*model.Project
	query := r.db.WithContext(ctx).
		Where("owner_id IN ? AND deleted_at IS NULL", ownerIDs)
	if len(statuses) > 0 {
		query = query.Where("status IN ?", statuses)
	}
	if err := query.
		Order("created_at DESC").
		Limit(limit).
		Find(&projects).Error; err != nil {
		return nil, fmt.Errorf("batch query owner projects failed: %w", err)
	}
	return projects, nil
}

// GetProjectIDsByOwnerID 获取指定用户创建的所有项目 ID 列表（用于 Feed）
func (r *projectRepository) GetProjectIDsByOwnerID(ctx context.Context, ownerID uint64) ([]uint64, error) {
	var ids []uint64
	err := r.db.WithContext(ctx).
		Model(&model.Project{}).
		Where("owner_id = ? AND deleted_at IS NULL", ownerID).
		Pluck("id", &ids).Error
	if err != nil {
		return nil, fmt.Errorf("查询用户项目ID失败: %w", err)
	}
	return ids, nil
}

// ListUserProjects 查询用户参与的项目（owner + 活跃成员），分页
func (r *projectRepository) ListUserProjects(ctx context.Context, userID uint64, offset, limit int) ([]*model.Project, int64, error) {
	// 统计总数
	var total int64
	if err := r.db.WithContext(ctx).
		Model(&model.Project{}).
		Joins("LEFT JOIN project_members pm ON pm.project_id = projects.id AND pm.is_active = true").
		Where("(projects.owner_id = ? OR pm.user_id = ?) AND projects.deleted_at IS NULL", userID, userID).
		Distinct("projects.id").
		Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("统计用户项目总数失败: %w", err)
	}

	// 查询项目列表
	var projects []*model.Project
	if err := r.db.WithContext(ctx).
		Joins("LEFT JOIN project_members pm ON pm.project_id = projects.id AND pm.is_active = true").
		Where("(projects.owner_id = ? OR pm.user_id = ?) AND projects.deleted_at IS NULL", userID, userID).
		Group("projects.id").
		Order("projects.created_at DESC").
		Offset(offset).Limit(limit).
		Find(&projects).Error; err != nil {
		return nil, 0, fmt.Errorf("查询用户项目列表失败: %w", err)
	}
	if projects == nil {
		projects = []*model.Project{}
	}
	return projects, total, nil
}

// ListProjectsAdvanced 高级筛选(状态+标签+成员数范围+排序)
func (r *projectRepository) ListProjectsAdvanced(ctx context.Context, filter ProjectFilter) ([]*model.Project, int64, error) {
	query := r.db.WithContext(ctx).Table("projects p").Where("p.deleted_at IS NULL")

	// 状态筛选
	if filter.Status != "" {
		query = query.Where("p.status = ?", filter.Status)
	}

	// 标签筛选 (假设 tags 为逗号分隔字符串)
	if len(filter.Tags) > 0 {
		for _, tag := range filter.Tags {
			query = query.Where("p.tags LIKE ?", "%"+tag+"%")
		}
	}

	// 成员数筛选
	if filter.MinMembers > 0 || filter.MaxMembers > 0 {
		query = query.Joins(`LEFT JOIN (
			SELECT project_id, COUNT(*) as member_count 
			FROM project_members 
			WHERE is_active = true 
			GROUP BY project_id
		) pm ON pm.project_id = p.id`)

		if filter.MinMembers > 0 {
			query = query.Where("COALESCE(pm.member_count, 0) >= ?", filter.MinMembers)
		}
		if filter.MaxMembers > 0 {
			query = query.Where("COALESCE(pm.member_count, 0) <= ?", filter.MaxMembers)
		}
	}

	// 统计总数
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("统计项目总数失败: %w", err)
	}

	// 排序
	switch filter.SortBy {
	case "hot":
		query = query.Select("p.*").
			Order("p.follower_count DESC, p.created_at DESC")
	case "rating":
		query = query.Select("p.*, COALESCE(AVG(gr.rating), 0) as avg_rating").
			Joins("LEFT JOIN game_reviews gr ON gr.project_id = p.id").
			Group("p.id").
			Order("avg_rating DESC")
	case "recent":
		query = query.Select("p.*").Order("p.updated_at DESC")
	case "followers":
		query = query.Select("p.*").Order("p.follower_count DESC")
	default:
		query = query.Select("p.*").Order("p.created_at DESC")
	}

	// 分页
	offset := (filter.Page - 1) * filter.PageSize
	var projects []*model.Project
	if err := query.Offset(offset).Limit(filter.PageSize).Find(&projects).Error; err != nil {
		return nil, 0, fmt.Errorf("查询项目列表失败: %w", err)
	}

	return projects, total, nil
}
