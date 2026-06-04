// Package repository 提供人才库查询的数据库访问层扩展。
// 本文件新增 TalentQuerier 接口及其实现，供组队系统人才库功能使用。
// 注意：userRepository 同时实现了 UserRepository 和 TalentQuerier 两个接口。
package repository

import (
	"context"
	"errors"
	"strings"

	"github.com/gamero/gamero/internal/model"
	apperrors "github.com/gamero/gamero/pkg/errors"
	"gorm.io/gorm"
)

// ListTalentsParams 人才库查询参数
type ListTalentsParams struct {
	Page            int    // 页码，从 1 开始
	PageSize        int    // 每页大小
	Category        string // 技能分类筛选：program/art/design/sound
	Keyword         string // 关键词（匹配昵称/用户名）
	Level           string // 技能熟练度筛选：beginner/intermediate/advanced
	OnlyAvailable   bool   // 是否只显示开放合作的人才（is_available=true）
	CoopPreference  string // 合作方式偏好筛选：online/offline/hybrid
	Location        string // 地区筛选（模糊匹配）
}

// TalentUser 人才库用户信息（含技能和参与项目数）
type TalentUser struct {
	model.User
	ProjectCount int64 `json:"project_count"` // 参与项目数（含 owner 和 member）
}

// TalentQuerier 人才库查询接口
// userRepository 实现此接口，以便在 TeamService 中使用
type TalentQuerier interface {
	// ListTalents 分页查询人才库
	// 排序规则：先按参与项目数倒序，再按注册时间倒序
	ListTalents(ctx context.Context, params *ListTalentsParams) ([]*TalentUser, int64, error)
}

// TalentInvitationRepository 人才邀请数据访问接口
type TalentInvitationRepository interface {
	// CreateInvitation 创建邀请（检查重复：同一项目同一岗位同一人只能有一条pending邀请）
	CreateInvitation(ctx context.Context, inv *model.TalentInvitation) error
	// GetInvitationByID 查询邀请详情
	GetInvitationByID(ctx context.Context, id uint64) (*model.TalentInvitation, error)
	// UpdateInvitationStatus 更新邀请状态
	UpdateInvitationStatus(ctx context.Context, id uint64, status model.InvitationStatus) error
	// GetPendingInvitation 查询同项目同岗位同人的待处理邀请（用于重复检查）
	GetPendingInvitation(ctx context.Context, projectID, talentID uint64, position model.RecruitmentPosition) (*model.TalentInvitation, error)
	// ListInvitationsByTalent 查询被邀请人收到的邀请列表（分页，按创建时间倒序）
	ListInvitationsByTalent(ctx context.Context, talentID uint64, status string, offset, limit int) ([]*model.TalentInvitation, int64, error)
	// ListInvitationsByProject 查询项目发出的邀请列表（分页，按创建时间倒序）
	ListInvitationsByProject(ctx context.Context, projectID uint64, status string, offset, limit int) ([]*model.TalentInvitation, int64, error)
	// DeleteTerminatedInvitation 删除已终结（declined/withdrawn/expired）的邀请记录，为重邀腾位
	DeleteTerminatedInvitation(ctx context.Context, projectID, talentID uint64, position model.RecruitmentPosition) error
}

// NewTalentInvitationRepository 创建 TalentInvitationRepository 实例
func NewTalentInvitationRepository(db *gorm.DB) TalentInvitationRepository {
	return &talentInvitationRepo{db: db}
}

type talentInvitationRepo struct{ db *gorm.DB }

func (r *talentInvitationRepo) CreateInvitation(ctx context.Context, inv *model.TalentInvitation) error {
	return r.db.WithContext(ctx).Create(inv).Error
}

func (r *talentInvitationRepo) GetInvitationByID(ctx context.Context, id uint64) (*model.TalentInvitation, error) {
	var inv model.TalentInvitation
	if err := r.db.WithContext(ctx).First(&inv, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperrors.New(apperrors.CodeNotFound, "邀请不存在")
		}
		return nil, err
	}
	return &inv, nil
}

func (r *talentInvitationRepo) UpdateInvitationStatus(ctx context.Context, id uint64, status model.InvitationStatus) error {
	result := r.db.WithContext(ctx).
		Model(&model.TalentInvitation{}).
		Where("id = ?", id).
		Update("status", status)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return apperrors.New(apperrors.CodeNotFound, "邀请不存在")
	}
	return nil
}

func (r *talentInvitationRepo) GetPendingInvitation(ctx context.Context, projectID, talentID uint64, position model.RecruitmentPosition) (*model.TalentInvitation, error) {
	var inv model.TalentInvitation
	err := r.db.WithContext(ctx).
		Where("project_id = ? AND talent_id = ? AND position = ? AND status = ?",
			projectID, talentID, position, model.InvitationStatusPending).
		First(&inv).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &inv, nil
}

func (r *talentInvitationRepo) ListInvitationsByTalent(ctx context.Context, talentID uint64, status string, offset, limit int) ([]*model.TalentInvitation, int64, error) {
	db := r.db.WithContext(ctx).Model(&model.TalentInvitation{}).Where("talent_id = ?", talentID)
	if status != "" {
		db = db.Where("status = ?", status)
	}
	var total int64
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var items []*model.TalentInvitation
	if err := db.Order("created_at DESC").Offset(offset).Limit(limit).Find(&items).Error; err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

func (r *talentInvitationRepo) ListInvitationsByProject(ctx context.Context, projectID uint64, status string, offset, limit int) ([]*model.TalentInvitation, int64, error) {
	db := r.db.WithContext(ctx).Model(&model.TalentInvitation{}).Where("project_id = ?", projectID)
	if status != "" {
		db = db.Where("status = ?", status)
	}
	var total int64
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var items []*model.TalentInvitation
	if err := db.Order("created_at DESC").Offset(offset).Limit(limit).Find(&items).Error; err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

func (r *talentInvitationRepo) DeleteTerminatedInvitation(ctx context.Context, projectID, talentID uint64, position model.RecruitmentPosition) error {
	return r.db.WithContext(ctx).
		Where("project_id = ? AND talent_id = ? AND position = ? AND status IN ?",
			projectID, talentID, position,
			[]string{string(model.InvitationStatusDeclined), string(model.InvitationStatusWithdrawn), string(model.InvitationStatusExpired)}).
		Delete(&model.TalentInvitation{}).Error
}

// ListTalents 实现人才库分页查询
// 连接 user_skills 表进行技能筛选，连接 project_members 统计参与项目数
func (r *userRepository) ListTalents(ctx context.Context, params *ListTalentsParams) ([]*TalentUser, int64, error) {
	// 基础查询：从 users 表出发，LEFT JOIN project_members 统计参与项目数
	baseDB := r.db.WithContext(ctx).Table("users u").
		Select(`u.id, u.username, u.nickname, u.avatar_key, u.bio, u.status, u.role,
			u.is_available, u.coop_preference, u.location, u.created_at, u.updated_at,
			COALESCE(pm.project_count, 0) as project_count`).
		Joins(`LEFT JOIN (
			SELECT user_id, COUNT(DISTINCT project_id) as project_count
			FROM project_members
			WHERE is_active = true
			GROUP BY user_id
		) pm ON pm.user_id = u.id`).
		Where("u.status = ?", model.UserStatusActive)

	// 只显示开放合作的人才
	if params.OnlyAvailable {
		baseDB = baseDB.Where("u.is_available = true")
	}

	// 合作方式偏好筛选
	if params.CoopPreference != "" {
		baseDB = baseDB.Where("u.coop_preference = ?", params.CoopPreference)
	}

	// 地区筛选（模糊匹配）
	if params.Location != "" {
		loc := "%" + strings.TrimSpace(params.Location) + "%"
		baseDB = baseDB.Where("u.location LIKE ?", loc)
	}

	// 关键词筛选（匹配昵称或用户名）
	if params.Keyword != "" {
		keyword := "%" + strings.TrimSpace(params.Keyword) + "%"
		baseDB = baseDB.Where("(u.nickname LIKE ? OR u.username LIKE ?)", keyword, keyword)
	}

	// 技能分类筛选：使用子查询避免 JOIN + GROUP BY 的 SQL 兼容问题
	if params.Category != "" || params.Level != "" {
		skillSub := r.db.WithContext(ctx).Table("user_skills s").
			Select("s.user_id").
			Where("s.user_id = u.id")
		if params.Category != "" {
			skillSub = skillSub.Where("s.category = ?", params.Category)
		}
		if params.Level != "" {
			skillSub = skillSub.Where("s.level = ?", params.Level)
		}
		baseDB = baseDB.Where("EXISTS (?)", skillSub)
	}

	// 统计总数
	var total int64
	countDB := r.db.WithContext(ctx).Table("users u").
		Where("u.status = ?", model.UserStatusActive)
	if params.OnlyAvailable {
		countDB = countDB.Where("u.is_available = true")
	}
	if params.CoopPreference != "" {
		countDB = countDB.Where("u.coop_preference = ?", params.CoopPreference)
	}
	if params.Location != "" {
		loc := "%" + strings.TrimSpace(params.Location) + "%"
		countDB = countDB.Where("u.location LIKE ?", loc)
	}
	if params.Keyword != "" {
		keyword := "%" + strings.TrimSpace(params.Keyword) + "%"
		countDB = countDB.Where("(u.nickname LIKE ? OR u.username LIKE ?)", keyword, keyword)
	}
	if params.Category != "" || params.Level != "" {
		skillSub := r.db.WithContext(ctx).Table("user_skills s").
			Select("s.user_id").
			Where("s.user_id = u.id")
		if params.Category != "" {
			skillSub = skillSub.Where("s.category = ?", params.Category)
		}
		if params.Level != "" {
			skillSub = skillSub.Where("s.level = ?", params.Level)
		}
		countDB = countDB.Where("EXISTS (?)", skillSub)
	}
	if err := countDB.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if total == 0 {
		return []*TalentUser{}, 0, nil
	}

	// 分页查询，排序：参与项目数倒序，注册时间倒序
	type rawUser struct {
		ID             uint64           `gorm:"column:id"`
		Username       string           `gorm:"column:username"`
		Nickname       string           `gorm:"column:nickname"`
		AvatarKey      string           `gorm:"column:avatar_key"`
		Bio            string           `gorm:"column:bio"`
		Status         model.UserStatus `gorm:"column:status"`
		Role           model.UserRole   `gorm:"column:role"`
		IsAvailable    bool             `gorm:"column:is_available"`
		CoopPreference string           `gorm:"column:coop_preference"`
		Location       string           `gorm:"column:location"`
		ProjectCount   int64            `gorm:"column:project_count"`
	}

	var rows []rawUser
	offset := (params.Page - 1) * params.PageSize
	if err := baseDB.
		Order("project_count DESC, u.created_at DESC").
		Offset(offset).
		Limit(params.PageSize).
		Scan(&rows).Error; err != nil {
		return nil, 0, err
	}

	// 批量查询用户技能（避免 N+1 问题）
	userIDs := make([]uint64, 0, len(rows))
	for _, row := range rows {
		userIDs = append(userIDs, row.ID)
	}

	skillMap := make(map[uint64][]model.UserSkill)
	if len(userIDs) > 0 {
		var skills []model.UserSkill
		if err := r.db.WithContext(ctx).
			Where("user_id IN ?", userIDs).
			Order("created_at ASC").
			Find(&skills).Error; err == nil {
			for _, sk := range skills {
				skillMap[sk.UserID] = append(skillMap[sk.UserID], sk)
			}
		}
	}

	// 组装结果
	result := make([]*TalentUser, 0, len(rows))
	for _, row := range rows {
		talent := &TalentUser{
			User: model.User{
				ID:             row.ID,
				Username:       row.Username,
				Nickname:       row.Nickname,
				AvatarKey:      row.AvatarKey,
				Bio:            row.Bio,
				Status:         row.Status,
				Role:           row.Role,
				IsAvailable:    row.IsAvailable,
				CoopPreference: row.CoopPreference,
				Location:       row.Location,
			},
			ProjectCount: row.ProjectCount,
		}
		if skills, ok := skillMap[row.ID]; ok {
			talent.User.Skills = skills
		} else {
			talent.User.Skills = []model.UserSkill{}
		}
		result = append(result, talent)
	}

	return result, total, nil
}

// verifyTalentQuerierImpl 编译期检查 userRepository 实现了 TalentQuerier 接口
var _ TalentQuerier = (*userRepository)(nil)

// PresignGetFunc 预签名函数类型（便于 storage.Client 解耦）
type PresignGetFunc func(ctx context.Context, key string, expiry interface{}) (string, error)

// ensureGormDB 从 UserRepository 获取底层 *gorm.DB（仅用于组队系统内部联查）
// 注意：只有 userRepository 结构体才有 db 字段；若 UserRepository 是 mock，此方法返回 nil
func ExtractDB(repo UserRepository) *gorm.DB {
	if r, ok := repo.(*userRepository); ok {
		return r.db
	}
	return nil
}
