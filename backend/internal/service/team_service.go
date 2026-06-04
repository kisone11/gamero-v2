// Package service 提供组队系统的业务逻辑层。
// 本文件包含招募、申请、合作评价、人才库、通知等全部业务逻辑：
//   - 招募信息的发布、浏览、更新、关闭
//   - 招募申请的提交、撤回、审批
//   - 合作评价的发表、补充
//   - 人才库浏览
//   - 通知写入（内部使用）
package service

import (
	"go.uber.org/zap"
	"github.com/gamero/gamero/pkg/logger"
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/gamero/gamero/internal/model"
	"github.com/gamero/gamero/internal/repository"
	apperrors "github.com/gamero/gamero/pkg/errors"
	"github.com/gamero/gamero/pkg/filter"
	"github.com/gamero/gamero/pkg/storage"
)

// ===========================
// 请求/响应 DTO
// ===========================

// CreateRecruitmentReq 发布招募请求
type CreateRecruitmentReq struct {
	ProjectID       uint64 `json:"project_id" binding:"required"`
	Position        string `json:"position" binding:"required"`         // program/art/design/sound
	Headcount       int    `json:"headcount" binding:"required"`        // 1-10
	Description     string `json:"description"`                         // 最多200字
	CooperationType string `json:"cooperation_type" binding:"required"` // online/offline/hybrid
	ContactType     string `json:"contact_type"`                        // wechat/qq/platform，默认platform
	ContactInfo     string `json:"contact_info"`                        // 联系方式，platform时为空
	ExpireDays      int    `json:"expire_days"`                         // 有效期天数，默认30，最大90
}

// UpdateRecruitmentReq 更新招募请求（均为可选字段）
type UpdateRecruitmentReq struct {
	Position        *string `json:"position"`
	Headcount       *int    `json:"headcount"`
	Description     *string `json:"description"`
	CooperationType *string `json:"cooperation_type"`
	ContactType     *string `json:"contact_type"`
	ContactInfo     *string `json:"contact_info"`
	ExpireDays      *int    `json:"expire_days"` // 重新设置有效期
}

// ApplyRecruitmentReq 申请加入请求
type ApplyRecruitmentReq struct {
	Position string `json:"position" binding:"required"` // 申请岗位（从招募岗位选）
	Message  string `json:"message"`                     // 个人说明，最多100字
}

// HandleApplicationReq 处理申请请求（通过/拒绝）
type HandleApplicationReq struct {
	Approve bool `json:"approve"` // true=通过，false=拒绝
}

// CreateReviewReq 发表合作评价请求
type CreateReviewReq struct {
	RevieweeID uint64   `json:"reviewee_id" binding:"required"` // 被评人 ID
	Rating     int8     `json:"rating" binding:"required"`      // 1-5星
	Comment    string   `json:"comment"`                        // 文字评语，最多500字
	Tags       []string `json:"tags"`                           // 评价标签
}

// AddReviewSupplementReq 补充评价说明请求
type AddReviewSupplementReq struct {
	Supplement string `json:"supplement" binding:"required"` // 最多200字
}

// ListRecruitmentsReq 招募列表查询参数
type ListRecruitmentsReq struct {
	Page            int    `form:"page"`
	PageSize        int    `form:"page_size"`
	Position        string `form:"position"`         // 岗位筛选
	CooperationType string `form:"cooperation_type"` // 合作方式筛选
	ProjectID       uint64 `form:"project_id"`       // 项目筛选
	OwnerID         uint64 `form:"owner_id"`         // 发布者筛选（查自己的招募时传入）
	Keyword         string `form:"keyword"`          // 关键词（匹配标题/描述）
}

// ListTalentsReq 人才库查询参数
type ListTalentsReq struct {
	Page           int    `form:"page"`
	PageSize       int    `form:"page_size"`
	Category       string `form:"category"`        // 技能分类筛选：program/art/design/sound
	Keyword        string `form:"keyword"`         // 关键词（匹配昵称/用户名）
	Level          string `form:"level"`           // 技能熟练度筛选：beginner/intermediate/advanced
	OnlyAvailable  bool   `form:"only_available"`  // 只显示开放合作的人才
	CoopPreference string `form:"coop_preference"` // 合作方式偏好：online/offline/hybrid
	Location       string `form:"location"`        // 地区筛选
}

// RecruitmentDetail 招募详情响应（含项目基本信息）
type RecruitmentDetail struct {
	*model.Recruitment
	ProjectName string `json:"project_name"` // 项目名称
	ProjectSlug string `json:"project_slug"` // 项目 slug
	CoverURL    string `json:"cover_url"`    // 封面预签名URL
}

// RecruitmentListItem 招募列表项（含项目名称和封面）
type RecruitmentListItem struct {
	ID              uint64                    `json:"id"`
	ProjectID       uint64                    `json:"project_id"`
	OwnerID         uint64                    `json:"owner_id"`
	Position        model.RecruitmentPosition `json:"position"`
	Headcount       int                       `json:"headcount"`
	Description     string                    `json:"description"`
	CooperationType model.CooperationType     `json:"cooperation_type"`
	ContactType     model.ContactType         `json:"contact_type"`
	ContactInfo     string                    `json:"contact_info,omitempty"`
	ExpireAt        time.Time                 `json:"expire_at"`
	ExpiresAt       *time.Time                `json:"expires_at,omitempty"`
	Status          model.RecruitmentStatus   `json:"status"`
	CreatedAt       time.Time                 `json:"created_at"`
	UpdatedAt       time.Time                 `json:"updated_at"`
	ProjectName     string                    `json:"project_name"`
	CoverURL        string                    `json:"cover_url,omitempty"`
}

// ApplicationDetail 申请详情（含申请人基本信息）
type ApplicationDetail struct {
	ID            uint64                    `json:"id"`
	RecruitmentID uint64                    `json:"recruitment_id"`
	ApplicantID   uint64                    `json:"applicant_id"`
	Position      model.RecruitmentPosition `json:"position"`
	Message       string                    `json:"message"`
	Status        model.ApplicationStatus   `json:"status"`
	CreatedAt     time.Time                 `json:"created_at"`
	UpdatedAt     time.Time                 `json:"updated_at"`
	// 申请人信息（查看申请列表时返回）
	ApplicantUsername  string `json:"applicant_username,omitempty"`
	ApplicantNickname  string `json:"applicant_nickname,omitempty"`
	ApplicantAvatarURL string `json:"applicant_avatar_url,omitempty"`
}

// ReviewDetail 评价详情（含评价人信息）
type ReviewDetail struct {
	ID         uint64    `json:"id"`
	ProjectID  uint64    `json:"project_id"`
	ReviewerID uint64    `json:"reviewer_id"`
	RevieweeID uint64    `json:"reviewee_id"`
	Rating     int8      `json:"rating"`
	Comment    string    `json:"comment"`
	Tags       []string  `json:"tags"`
	Supplement string    `json:"supplement,omitempty"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
	// 评价人基本信息
	ReviewerUsername  string `json:"reviewer_username"`
	ReviewerNickname  string `json:"reviewer_nickname"`
	ReviewerAvatarURL string `json:"reviewer_avatar_url,omitempty"`
}

// RatingSummary 评分摘要
type RatingSummary struct {
	AverageRating float64 `json:"average_rating"` // 平均分
	Count         int64   `json:"count"`          // 评价总数
}

// TalentListItem 人才库列表项
type TalentListItem struct {
	ID             uint64            `json:"id"`
	Username       string            `json:"username"`
	Nickname       string            `json:"nickname"`
	AvatarURL      string            `json:"avatar_url,omitempty"`
	Bio            string            `json:"bio,omitempty"`
	Skills         []model.UserSkill `json:"skills"`
	ProjectCount   int64             `json:"project_count"`             // 参与项目数
	IsAvailable    bool              `json:"is_available"`              // 是否开放合作
	CoopPreference string            `json:"coop_preference,omitempty"` // 合作方式偏好
	Location       string            `json:"location,omitempty"`        // 所在地区
}

// InviteTalentReq 发送人才邀请请求
type InviteTalentReq struct {
	TalentID uint64 `json:"talent_id" binding:"required"` // 被邀请人用户 ID
	Position string `json:"position" binding:"required"`  // 邀请岗位
	Message  string `json:"message"`                      // 邀请说明（最多200字）
}

// InvitationDetail 邀请详情（含项目和人才信息）
type InvitationDetail struct {
	*model.TalentInvitation
	ProjectName      string `json:"project_name"`
	InviterUsername  string `json:"inviter_username"`
	InviterNickname  string `json:"inviter_nickname"`
	InviterAvatarURL string `json:"inviter_avatar_url,omitempty"`
	TalentUsername   string `json:"talent_username"`
	TalentNickname   string `json:"talent_nickname"`
	TalentAvatarURL  string `json:"talent_avatar_url,omitempty"`
}

// SetAvailabilityReq 设置人才可用性请求
type SetAvailabilityReq struct {
	IsAvailable    bool   `json:"is_available"`
	CoopPreference string `json:"coop_preference"` // online/offline/hybrid，可为空
	Location       string `json:"location"`        // 所在地区，可为空
}

// ===========================
// Service 接口
// ===========================

// TeamService 组队系统业务逻辑接口
type TeamService interface {
	// ===== 招募 =====

	// CreateRecruitment 发布招募（验证 ownerID 是项目 owner）
	CreateRecruitment(ctx context.Context, ownerID uint64, req *CreateRecruitmentReq) (*model.Recruitment, error)
	// GetRecruitment 获取招募详情（含项目基本信息和封面预签名URL）
	GetRecruitment(ctx context.Context, id uint64) (*RecruitmentDetail, error)
	// UpdateRecruitment 更新招募信息（验证 ownerID）
	UpdateRecruitment(ctx context.Context, ownerID, id uint64, req *UpdateRecruitmentReq) (*model.Recruitment, error)
	// CloseRecruitment 关闭招募（验证 ownerID）
	CloseRecruitment(ctx context.Context, ownerID, id uint64) error
	// ReopenRecruitment 重开招募（验证 ownerID）
	ReopenRecruitment(ctx context.Context, ownerID, id uint64) error
	// ListRecruitments 分页查询招募列表（含项目名称和封面URL）
	ListRecruitments(ctx context.Context, req *ListRecruitmentsReq) ([]*RecruitmentListItem, int64, error)
	// CloseExpiredRecruitments 批量关闭已过期的招募帖，返回关闭数量
	CloseExpiredRecruitments(ctx context.Context) (int64, error)
	// ListExpiringSoonRecruitments 查询 withinDays 天内即将过期的招募（status=open）
	ListExpiringSoonRecruitments(ctx context.Context, withinDays int) ([]*RecruitmentListItem, error)

	// ===== 申请 =====

	// ApplyRecruitment 提交申请（验证不能申请自己项目的招募，不能重复申请）
	ApplyRecruitment(ctx context.Context, applicantID, recruitmentID uint64, req *ApplyRecruitmentReq) (*model.RecruitmentApplication, error)
	// HandleApplication 处理申请（approve=true通过，false拒绝；验证 ownerID 权限；通知申请人）
	HandleApplication(ctx context.Context, ownerID, applicationID uint64, approve bool) (*model.RecruitmentApplication, error)
	// WithdrawApplication 撤回申请（验证申请人，只能撤回 pending 状态）
	WithdrawApplication(ctx context.Context, applicantID, recruitmentID uint64) error
	// GetMyApplications 查询我的申请列表（分页）
	GetMyApplications(ctx context.Context, applicantID uint64, page, pageSize int) ([]*ApplicationDetail, int64, error)
	// GetRecruitmentApplications 查看招募下的申请列表（验证 ownerID 权限，分页）
	GetRecruitmentApplications(ctx context.Context, ownerID, recruitmentID uint64, page, pageSize int) ([]*ApplicationDetail, int64, error)

	// ===== 合作评价 =====

	// CreateReview 发表合作评价（验证项目已结束、双方都是成员、未重复评价）
	CreateReview(ctx context.Context, reviewerID, projectID uint64, req *CreateReviewReq) (*model.CollaborationReview, error)
	// AddReviewSupplement 补充评价说明（验证 reviewerID，只能补充一次）
	AddReviewSupplement(ctx context.Context, reviewerID, reviewID uint64, req *AddReviewSupplementReq) (*model.CollaborationReview, error)
	// GetUserReviews 获取用户收到的评价列表（分页，含评价人信息）
	GetUserReviews(ctx context.Context, userID uint64, page, pageSize int) ([]*ReviewDetail, int64, error)
	// GetUserRatingSummary 获取用户评分摘要
	GetUserRatingSummary(ctx context.Context, userID uint64) (*RatingSummary, error)

	// ===== 人才库 =====

	// ListTalents 人才库列表（按参与项目数倒序，再按注册时间倒序；分页）
	ListTalents(ctx context.Context, req *ListTalentsReq) ([]*TalentListItem, int64, error)

	// SetAvailability 设置自己是否开放合作（加入/退出人才库）
	SetAvailability(ctx context.Context, userID uint64, req *SetAvailabilityReq) error

	// ===== 人才邀请 =====

	// InviteTalent 项目主向人才库中的开发者发送邀请
	// inviterID 必须是 projectID 的 owner；talentID 为被邀请人
	InviteTalent(ctx context.Context, inviterID, projectID, talentID uint64, req *InviteTalentReq) (*model.TalentInvitation, error)
	// WithdrawInvitation 项目主撤回邀请（inviterID 需是邀请发送人）
	WithdrawInvitation(ctx context.Context, inviterID, invitationID uint64) error
	// RespondInvitation 被邀请人回应邀请（accept=true 接受，false 拒绝）
	RespondInvitation(ctx context.Context, talentID, invitationID uint64, accept bool) error
	// GetMyInvitations 被邀请人查看收到的邀请列表（分页）
	GetMyInvitations(ctx context.Context, talentID uint64, status string, page, pageSize int) ([]*InvitationDetail, int64, error)
	// GetProjectInvitations 项目主查看项目发出的邀请列表（分页）
	GetProjectInvitations(ctx context.Context, inviterID, projectID uint64, status string, page, pageSize int) ([]*InvitationDetail, int64, error)

	// SetNotificationService 注入统一通知服务（可选）
	SetNotificationService(svc NotificationService)
}

// ===========================
// Service 实现
// ===========================

// teamService 组队业务逻辑实现
type teamService struct {
	repo           repository.TeamRepository
	projectRepo    repository.ProjectRepository
	userRepo       repository.UserRepository
	storage        *storage.Client
	invitationRepo repository.TalentInvitationRepository // 人才邀请仓储（可选，见 SetInvitationRepo）
	*NotificationClient
}

// NewTeamService 创建组队服务实例
func NewTeamService(
	repo repository.TeamRepository,
	projectRepo repository.ProjectRepository,
	userRepo repository.UserRepository,
	stor *storage.Client,
) TeamService {
	svc := &teamService{
		repo:        repo,
		projectRepo: projectRepo,
		userRepo:    userRepo,
		storage:     stor,
		NotificationClient: &NotificationClient{},
	}
	return svc
}

// SetInvitationRepo 注入邀请仓储（供 router 初始化时注入）
func (s *teamService) SetInvitationRepo(repo repository.TalentInvitationRepository) {
	s.invitationRepo = repo
}


// ===========================
// 内部工具函数
// ===========================

// encodeLinks 将 []string 序列化为 JSON 字符串
func encodeLinks(links []string) string {
	if links == nil {
		links = []string{}
	}
	b, _ := json.Marshal(links)
	return string(b)
}

// decodeLinks 将 JSON 字符串反序列化为 []string
func decodeLinks(jsonStr string) []string {
	if jsonStr == "" || jsonStr == "null" {
		return []string{}
	}
	var result []string
	if err := json.Unmarshal([]byte(jsonStr), &result); err != nil {
		return []string{}
	}
	return result
}

func (s *teamService) presignURL(ctx context.Context, key string) string {
	if key == "" || s.storage == nil {
		return ""
	}
	url, err := s.storage.PresignedGetURL(ctx, key, 24*time.Hour)
	if err != nil {
		return s.storage.GetPublicURL(key)
	}
	return url
}

// buildRecruitmentListItem 将 Recruitment 和项目信息组装为列表项
func (s *teamService) buildRecruitmentListItem(ctx context.Context, rec *model.Recruitment, projectName, coverKey string) *RecruitmentListItem {
	return &RecruitmentListItem{
		ID:              rec.ID,
		ProjectID:       rec.ProjectID,
		OwnerID:         rec.OwnerID,
		Position:        rec.Position,
		Headcount:       rec.Headcount,
		Description:     rec.Description,
		CooperationType: rec.CooperationType,
		ContactType:     rec.ContactType,
		ContactInfo:     rec.ContactInfo,
		ExpireAt:        rec.ExpireAt,
		ExpiresAt:       rec.ExpiresAt,
		Status:          rec.Status,
		CreatedAt:       rec.CreatedAt,
		UpdatedAt:       rec.UpdatedAt,
		ProjectName:     projectName,
		CoverURL:        s.presignURL(ctx, coverKey),
	}
}

// buildApplicationDetail 将 RecruitmentApplication 组装为详情 DTO
func (s *teamService) buildApplicationDetail(ctx context.Context, app *model.RecruitmentApplication, withApplicant bool) *ApplicationDetail {
	detail := &ApplicationDetail{
		ID:            app.ID,
		RecruitmentID: app.RecruitmentID,
		ApplicantID:   app.ApplicantID,
		Position:      app.Position,
		Message:       app.Message,
		Status:        app.Status,
		CreatedAt:     app.CreatedAt,
		UpdatedAt:     app.UpdatedAt,
	}
	// 查询申请人信息（用于 owner 查看申请列表）
	if withApplicant {
		user, err := s.userRepo.GetUserByID(ctx, app.ApplicantID)
		if err == nil {
			detail.ApplicantUsername = user.Username
			detail.ApplicantNickname = user.Nickname
			detail.ApplicantAvatarURL = s.presignURL(ctx, user.AvatarKey)
		}
	}
	return detail
}

// writeNotification 写入通知记录（内部调用，失败不影响主流程）
func (s *teamService) writeNotification(ctx context.Context, userID uint64, senderID uint64, ntype model.NotificationType, title, content string, metadata map[string]interface{}) {
	metaBytes, _ := json.Marshal(metadata)
	n := &model.Notification{
		UserID:   userID,
		SenderID: senderID,
		Type:     ntype,
		Title:    title,
		Content:  content,
		Metadata: string(metaBytes),
	}
	// 写入通知（副作用 cache.Incr + WS push 由 NotificationService.afterSend 异步处理）
	s.SendModel(ctx, n)
}

// ===========================
// 招募实现
// ===========================

// CreateRecruitment 发布招募信息
// 验证：ownerID 必须是项目 owner；字段格式合法；有效期范围
func (s *teamService) CreateRecruitment(ctx context.Context, ownerID uint64, req *CreateRecruitmentReq) (*model.Recruitment, error) {
	// 验证岗位是否合法
	pos := model.RecruitmentPosition(req.Position)
	if !model.ValidRecruitmentPositions[pos] {
		return nil, apperrors.New(apperrors.CodeParamInvalid, "岗位分类无效，请选择 program/art/design/sound")
	}

	// 验证合作方式
	coop := model.CooperationType(req.CooperationType)
	if !model.ValidCooperationTypes[coop] {
		return nil, apperrors.New(apperrors.CodeParamInvalid, "合作方式无效，请选择 online/offline/hybrid")
	}

	// 联系方式类型，默认 platform
	contactType := model.ContactTypePlatform
	if req.ContactType != "" {
		contactType = model.ContactType(req.ContactType)
		if !model.ValidContactTypes[contactType] {
			return nil, apperrors.New(apperrors.CodeParamInvalid, "联系方式类型无效，请选择 wechat/qq/platform")
		}
	}

	// platform 类型时 contact_info 应为空
	if contactType == model.ContactTypePlatform {
		req.ContactInfo = ""
	} else if req.ContactInfo == "" {
		return nil, apperrors.New(apperrors.CodeParamMissing, "联系方式不能为空")
	}

	// 验证需求人数
	if req.Headcount < model.RecruitmentMinHeadcount || req.Headcount > model.RecruitmentMaxHeadcount {
		return nil, apperrors.CodeError(apperrors.CodeRecruitmentHeadcount)
	}

	// 验证说明长度
	if len([]rune(req.Description)) > 200 {
		return nil, apperrors.New(apperrors.CodeParamTooLong, "需求说明不能超过200字")
	}
	if req.Description != "" && filter.Contains(req.Description) {
		return nil, apperrors.New(apperrors.CodeContentViolation, "需求说明包含敏感词，请修改后重试")
	}

	// 计算有效期
	expireDays := req.ExpireDays
	if expireDays <= 0 {
		expireDays = model.RecruitmentDefaultDays
	}
	if expireDays > model.RecruitmentMaxDays {
		return nil, apperrors.CodeError(apperrors.CodeRecruitmentExpireDays)
	}

	// 验证项目存在且 ownerID 是 owner
	project, err := s.projectRepo.GetProjectByID(ctx, req.ProjectID)
	if err != nil {
		return nil, err
	}
	if project.OwnerID != ownerID {
		return nil, apperrors.CodeError(apperrors.CodeProjectForbidden)
	}

	expireTime := time.Now().Add(time.Duration(expireDays) * 24 * time.Hour)
	rec := &model.Recruitment{
		ProjectID:       req.ProjectID,
		OwnerID:         ownerID,
		Position:        pos,
		Headcount:       req.Headcount,
		Description:     req.Description,
		CooperationType: coop,
		ContactType:     contactType,
		ContactInfo:     req.ContactInfo,
		ExpireAt:        expireTime,
		ExpiresAt:       &expireTime, // 与 ExpireAt 保持同步，供定时批量关闭使用
		Status:          model.RecruitmentStatusOpen,
	}

	if err := s.repo.CreateRecruitment(ctx, rec); err != nil {
		return nil, apperrors.Wrap(apperrors.CodeInternalError, err)
	}
	return rec, nil
}

// GetRecruitment 获取招募详情，附带项目名称和封面预签名URL
func (s *teamService) GetRecruitment(ctx context.Context, id uint64) (*RecruitmentDetail, error) {
	rec, err := s.repo.GetRecruitmentByID(ctx, id)
	if err != nil {
		return nil, err
	}

	detail := &RecruitmentDetail{Recruitment: rec}

	// 查询关联项目信息
	project, err := s.projectRepo.GetProjectByID(ctx, rec.ProjectID)
	if err == nil {
		detail.ProjectName = project.Name
		detail.ProjectSlug = project.Slug
		detail.CoverURL = s.presignURL(ctx, project.CoverKey)
	}

	return detail, nil
}

// UpdateRecruitment 更新招募信息（验证 ownerID 权限）
func (s *teamService) UpdateRecruitment(ctx context.Context, ownerID, id uint64, req *UpdateRecruitmentReq) (*model.Recruitment, error) {
	rec, err := s.repo.GetRecruitmentByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// 验证权限
	if rec.OwnerID != ownerID {
		return nil, apperrors.CodeError(apperrors.CodeRecruitmentForbidden)
	}

	updates := map[string]interface{}{}

	if req.Position != nil {
		pos := model.RecruitmentPosition(*req.Position)
		if !model.ValidRecruitmentPositions[pos] {
			return nil, apperrors.New(apperrors.CodeParamInvalid, "岗位分类无效")
		}
		updates["position"] = pos
	}

	if req.Headcount != nil {
		if *req.Headcount < model.RecruitmentMinHeadcount || *req.Headcount > model.RecruitmentMaxHeadcount {
			return nil, apperrors.CodeError(apperrors.CodeRecruitmentHeadcount)
		}
		updates["headcount"] = *req.Headcount
	}

	if req.Description != nil {
		if len([]rune(*req.Description)) > 200 {
			return nil, apperrors.New(apperrors.CodeParamTooLong, "需求说明不能超过200字")
		}
		updates["description"] = *req.Description
	}

	if req.CooperationType != nil {
		coop := model.CooperationType(*req.CooperationType)
		if !model.ValidCooperationTypes[coop] {
			return nil, apperrors.New(apperrors.CodeParamInvalid, "合作方式无效")
		}
		updates["cooperation_type"] = coop
	}

	if req.ContactType != nil {
		ct := model.ContactType(*req.ContactType)
		if !model.ValidContactTypes[ct] {
			return nil, apperrors.New(apperrors.CodeParamInvalid, "联系方式类型无效")
		}
		updates["contact_type"] = ct
		if ct == model.ContactTypePlatform {
			updates["contact_info"] = ""
		}
	}

	if req.ContactInfo != nil {
		updates["contact_info"] = *req.ContactInfo
	}

	if req.ExpireDays != nil {
		if *req.ExpireDays <= 0 || *req.ExpireDays > model.RecruitmentMaxDays {
			return nil, apperrors.CodeError(apperrors.CodeRecruitmentExpireDays)
		}
		newExpireAt := time.Now().Add(time.Duration(*req.ExpireDays) * 24 * time.Hour)
		updates["expire_at"] = newExpireAt
		updates["expires_at"] = newExpireAt // 同步更新，供定时批量关闭使用
	}

	if len(updates) > 0 {
		if err := s.repo.UpdateRecruitment(ctx, id, updates); err != nil {
			return nil, apperrors.Wrap(apperrors.CodeInternalError, err)
		}
	}

	// 返回最新招募记录
	return s.repo.GetRecruitmentByID(ctx, id)
}

// CloseRecruitment 关闭招募（验证 ownerID 权限）
func (s *teamService) CloseRecruitment(ctx context.Context, ownerID, id uint64) error {
	rec, err := s.repo.GetRecruitmentByID(ctx, id)
	if err != nil {
		return err
	}
	if rec.OwnerID != ownerID {
		return apperrors.CodeError(apperrors.CodeRecruitmentForbidden)
	}
	if rec.Status == model.RecruitmentStatusClosed {
		return nil // 幂等，已关闭则直接返回
	}
	return s.repo.CloseRecruitment(ctx, id)
}

// ReopenRecruitment 重开招募（验证 ownerID 权限）
func (s *teamService) ReopenRecruitment(ctx context.Context, ownerID, id uint64) error {
	rec, err := s.repo.GetRecruitmentByID(ctx, id)
	if err != nil {
		return err
	}
	if rec.OwnerID != ownerID {
		return apperrors.CodeError(apperrors.CodeRecruitmentForbidden)
	}
	if rec.Status == model.RecruitmentStatusOpen {
		return nil // 幂等，已开放则直接返回
	}
	return s.repo.ReopenRecruitment(ctx, id)
}

// ListRecruitments 分页查询招募列表，含项目名称和封面预签名URL
func (s *teamService) ListRecruitments(ctx context.Context, req *ListRecruitmentsReq) ([]*RecruitmentListItem, int64, error) {
	// 分页参数默认值
	page := req.Page
	if page < 1 {
		page = 1
	}
	pageSize := req.PageSize
	if pageSize < 1 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}

	params := &repository.ListRecruitmentsParams{
		Page:            page,
		PageSize:        pageSize,
		Position:        model.RecruitmentPosition(req.Position),
		CooperationType: model.CooperationType(req.CooperationType),
		ProjectID:       req.ProjectID,
		OwnerID:         req.OwnerID,
		Keyword:         req.Keyword,
	}

	list, total, err := s.repo.ListRecruitments(ctx, params)
	if err != nil {
		return nil, 0, apperrors.Wrap(apperrors.CodeInternalError, err)
	}

	// 批量查询关联的项目信息（去重项目ID）
	projectIDSet := make(map[uint64]struct{})
	for _, rec := range list {
		projectIDSet[rec.ProjectID] = struct{}{}
	}
	projectInfoMap := make(map[uint64]*model.Project)
	for pid := range projectIDSet {
		p, err := s.projectRepo.GetProjectByID(ctx, pid)
		if err == nil {
			projectInfoMap[pid] = p
		}
	}

	// 组装响应
	items := make([]*RecruitmentListItem, 0, len(list))
	for _, rec := range list {
		var projectName, coverKey string
		if p, ok := projectInfoMap[rec.ProjectID]; ok {
			projectName = p.Name
			coverKey = p.CoverKey
		}
		items = append(items, s.buildRecruitmentListItem(ctx, rec, projectName, coverKey))
	}

	return items, total, nil
}

// CloseExpiredRecruitments 批量关闭已过期的招募帖
func (s *teamService) CloseExpiredRecruitments(ctx context.Context) (int64, error) {
	return s.repo.CloseExpiredRecruitments(ctx)
}

// ListExpiringSoonRecruitments 查询 withinDays 天内即将过期的招募（status=open）
func (s *teamService) ListExpiringSoonRecruitments(ctx context.Context, withinDays int) ([]*RecruitmentListItem, error) {
	recs, err := s.repo.ListExpiringSoonRecruitments(ctx, withinDays)
	if err != nil {
		return nil, apperrors.Wrap(apperrors.CodeInternalError, err)
	}

	// 批量查询关联的项目信息
	projectIDSet := make(map[uint64]struct{})
	for _, rec := range recs {
		projectIDSet[rec.ProjectID] = struct{}{}
	}
	projectInfoMap := make(map[uint64]*model.Project)
	for pid := range projectIDSet {
		p, err := s.projectRepo.GetProjectByID(ctx, pid)
		if err == nil {
			projectInfoMap[pid] = p
		}
	}

	items := make([]*RecruitmentListItem, 0, len(recs))
	for _, rec := range recs {
		var projectName, coverKey string
		if p, ok := projectInfoMap[rec.ProjectID]; ok {
			projectName = p.Name
			coverKey = p.CoverKey
		}
		items = append(items, s.buildRecruitmentListItem(ctx, rec, projectName, coverKey))
	}
	return items, nil
}

// ===========================
// 申请实现
// ===========================

// ApplyRecruitment 提交申请
// 验证：不能申请自己项目的招募；不能重复申请；招募必须是开放状态
func (s *teamService) ApplyRecruitment(ctx context.Context, applicantID, recruitmentID uint64, req *ApplyRecruitmentReq) (*model.RecruitmentApplication, error) {
	// 查询招募
	rec, err := s.repo.GetRecruitmentByID(ctx, recruitmentID)
	if err != nil {
		return nil, err
	}

	// 招募必须处于开放且未过期状态
	if !rec.IsOpen() {
		return nil, apperrors.CodeError(apperrors.CodeRecruitmentClosed)
	}

	// 申请岗位必须是招募岗位
	pos := model.RecruitmentPosition(req.Position)
	if pos != rec.Position {
		return nil, apperrors.New(apperrors.CodeParamInvalid, fmt.Sprintf("申请岗位 %s 与招募岗位 %s 不匹配", req.Position, rec.Position))
	}

	// 不能申请自己项目的招募
	if rec.OwnerID == applicantID {
		return nil, apperrors.CodeError(apperrors.CodeApplicationSelfProject)
	}

	// 不能重复申请
	hasApplied, err := s.repo.HasApplied(ctx, applicantID, recruitmentID)
	if err != nil {
		return nil, apperrors.Wrap(apperrors.CodeInternalError, err)
	}
	if hasApplied {
		return nil, apperrors.CodeError(apperrors.CodeApplicationDuplicate)
	}

	// 验证个人说明长度
	if len([]rune(req.Message)) > 100 {
		return nil, apperrors.New(apperrors.CodeParamTooLong, "个人说明不能超过100字")
	}
	if req.Message != "" && filter.Contains(req.Message) {
		return nil, apperrors.New(apperrors.CodeContentViolation, "个人说明包含敏感词，请修改后重试")
	}

	app := &model.RecruitmentApplication{
		RecruitmentID: recruitmentID,
		ApplicantID:   applicantID,
		Position:      pos,
		Message:       req.Message,
		Status:        model.ApplicationStatusPending,
	}

	if err := s.repo.CreateApplication(ctx, app); err != nil {
		return nil, apperrors.Wrap(apperrors.CodeInternalError, err)
	}

	// 给项目 owner 写通知：有人申请加入你的项目
	applicant, _ := s.userRepo.GetUserByID(ctx, applicantID)
	applicantName := "某位开发者"
	if applicant != nil {
		applicantName = applicant.Nickname
		if applicantName == "" {
			applicantName = applicant.Username
		}
	}
	project, _ := s.projectRepo.GetProjectByID(ctx, rec.ProjectID)
	projectName := "你的项目"
	if project != nil {
		projectName = project.Name
	}
	s.writeNotification(ctx, rec.OwnerID, applicantID,
		model.NotificationTypeRecruitmentApplied,
		"有人申请加入你的项目",
		fmt.Sprintf("%s 申请加入 %s（%s岗位）", applicantName, projectName, string(pos)),
		map[string]interface{}{
			"recruitment_id": recruitmentID,
			"application_id": app.ID,
			"applicant_id":   applicantID,
			"project_id":     rec.ProjectID,
		},
	)

	return app, nil
}

// HandleApplication 处理申请（通过/拒绝）
// 验证：ownerID 必须是招募的 owner；申请必须是 pending 状态
func (s *teamService) HandleApplication(ctx context.Context, ownerID, applicationID uint64, approve bool) (*model.RecruitmentApplication, error) {
	app, err := s.repo.GetApplicationByID(ctx, applicationID)
	if err != nil {
		return nil, err
	}

	// 验证申请状态
	if app.Status != model.ApplicationStatusPending {
		return nil, apperrors.CodeError(apperrors.CodeApplicationNotPending)
	}

	// 查询关联招募，验证操作者权限
	rec, err := s.repo.GetRecruitmentByID(ctx, app.RecruitmentID)
	if err != nil {
		return nil, err
	}
	if rec.OwnerID != ownerID {
		return nil, apperrors.CodeError(apperrors.CodeApplicationForbidden)
	}

	// 更新申请状态
	var newStatus model.ApplicationStatus
	var notifType model.NotificationType
	var notifTitle, notifContent string

	// 提前查询项目名（approve/reject 都需要）
	projectName := "该项目"
	if p, err := s.projectRepo.GetProjectByID(ctx, rec.ProjectID); err == nil {
		projectName = p.Name
	}

	if approve {
		newStatus = model.ApplicationStatusApproved
		notifType = model.NotificationTypeApplicationApproved
		notifTitle = "你的申请已通过"
		notifContent = fmt.Sprintf("你申请加入 %s（%s岗位）的请求已通过，欢迎加入！", projectName, string(app.Position))
	} else {
		newStatus = model.ApplicationStatusRejected
		notifType = model.NotificationTypeApplicationRejected
		notifTitle = "你的申请未通过"
		notifContent = fmt.Sprintf("你申请加入 %s（%s岗位）的请求未通过，请继续加油！", projectName, string(app.Position))
	}

	if err := s.repo.UpdateApplicationStatus(ctx, applicationID, newStatus); err != nil {
		return nil, apperrors.Wrap(apperrors.CodeInternalError, err)
	}

	// 申请通过：写入 project_members，并联动群聊成员
	if approve {
		if err := s.projectRepo.AddMember(ctx, &model.ProjectMember{
			ProjectID: rec.ProjectID,
			UserID:    app.ApplicantID,
			Role:      model.ProjectMemberRoleMember,
		}); err != nil {
			logger.Warn("failed to add project member", zap.Error(err))
		}
	}

	// 通知申请人
	s.writeNotification(ctx, app.ApplicantID, rec.OwnerID, notifType, notifTitle, notifContent,
		map[string]interface{}{
			"application_id": applicationID,
			"recruitment_id": app.RecruitmentID,
			"project_id":     rec.ProjectID,
		},
	)

	// 返回最新申请记录
	app.Status = newStatus
	return app, nil
}

// WithdrawApplication 撤回申请（只能撤回 pending 状态）
func (s *teamService) WithdrawApplication(ctx context.Context, applicantID, recruitmentID uint64) error {
	app, err := s.repo.GetApplicationByApplicantAndRecruitment(ctx, applicantID, recruitmentID)
	if err != nil {
		return err
	}

	// 只能撤回 pending 状态的申请
	if app.Status != model.ApplicationStatusPending {
		return apperrors.CodeError(apperrors.CodeApplicationNotWithdrawable)
	}

	return s.repo.UpdateApplicationStatus(ctx, app.ID, model.ApplicationStatusWithdrawn)
}

// GetMyApplications 查询我的申请列表（分页）
func (s *teamService) GetMyApplications(ctx context.Context, applicantID uint64, page, pageSize int) ([]*ApplicationDetail, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}

	list, total, err := s.repo.GetApplicationsByApplicantID(ctx, applicantID, page, pageSize)
	if err != nil {
		return nil, 0, apperrors.Wrap(apperrors.CodeInternalError, err)
	}

	items := make([]*ApplicationDetail, 0, len(list))
	for _, app := range list {
		items = append(items, s.buildApplicationDetail(ctx, app, false))
	}
	return items, total, nil
}

// GetRecruitmentApplications 查看招募下的申请列表（验证 ownerID 权限，分页）
func (s *teamService) GetRecruitmentApplications(ctx context.Context, ownerID, recruitmentID uint64, page, pageSize int) ([]*ApplicationDetail, int64, error) {
	// 验证权限
	rec, err := s.repo.GetRecruitmentByID(ctx, recruitmentID)
	if err != nil {
		return nil, 0, err
	}
	if rec.OwnerID != ownerID {
		return nil, 0, apperrors.CodeError(apperrors.CodeRecruitmentForbidden)
	}

	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}

	list, total, err := s.repo.GetApplicationsByRecruitmentID(ctx, recruitmentID, page, pageSize)
	if err != nil {
		return nil, 0, apperrors.Wrap(apperrors.CodeInternalError, err)
	}

	// 批量查申请人信息（避免 N+1）
	applicantIDs := make([]uint64, 0, len(list))
	for _, app := range list {
		applicantIDs = append(applicantIDs, app.ApplicantID)
	}
	applicantMap := make(map[uint64]*model.User)
	if users, err := s.userRepo.GetUsersByIDs(ctx, applicantIDs); err == nil {
		for _, u := range users {
			applicantMap[u.ID] = u
		}
	}

	items := make([]*ApplicationDetail, 0, len(list))
	for _, app := range list {
		detail := s.buildApplicationDetail(ctx, app, false)
		if u, ok := applicantMap[app.ApplicantID]; ok {
			detail.ApplicantUsername = u.Username
			detail.ApplicantNickname = u.Nickname
			detail.ApplicantAvatarURL = s.presignURL(ctx, u.AvatarKey)
		}
		items = append(items, detail)
	}
	return items, total, nil
}

// ===========================
// 合作评价实现
// ===========================

// CreateReview 发表合作评价
// 验证：项目已结束（launched或abandoned）；双方都是项目成员；未重复评价
func (s *teamService) CreateReview(ctx context.Context, reviewerID, projectID uint64, req *CreateReviewReq) (*model.CollaborationReview, error) {
	// 不能评价自己
	if reviewerID == req.RevieweeID {
		return nil, apperrors.New(apperrors.CodeParamInvalid, "不能评价自己")
	}

	// 验证评分范围
	if req.Rating < 1 || req.Rating > 5 {
		return nil, apperrors.New(apperrors.CodeParamInvalid, "评分须在 1-5 之间")
	}

	// 验证评语长度和敏感词
	if len([]rune(req.Comment)) > 500 {
		return nil, apperrors.New(apperrors.CodeParamTooLong, "评语不能超过500字")
	}
	if req.Comment != "" && filter.Contains(req.Comment) {
		return nil, apperrors.New(apperrors.CodeContentViolation, "评语包含敏感词，请修改后重试")
	}

	// 验证评价标签（只允许预设标签）
	for _, tag := range req.Tags {
		valid := false
		for _, preset := range model.PresetReviewTags {
			if tag == preset {
				valid = true
				break
			}
		}
		if !valid {
			return nil, apperrors.New(apperrors.CodeParamInvalid, fmt.Sprintf("评价标签 \"%s\" 不在预设列表中", tag))
		}
	}

	// 查询项目
	project, err := s.projectRepo.GetProjectByID(ctx, projectID)
	if err != nil {
		return nil, err
	}

	// 验证项目已结束
	if project.Status != model.ProjectStatusLaunched && project.Status != model.ProjectStatusAbandoned {
		return nil, apperrors.CodeError(apperrors.CodeProjectNotFinished)
	}

	// 验证双方都是项目成员
	isReviewerMember, err := s.projectRepo.IsMember(ctx, projectID, reviewerID)
	if err != nil {
		return nil, apperrors.Wrap(apperrors.CodeInternalError, err)
	}
	isRevieweeMember, err := s.projectRepo.IsMember(ctx, projectID, req.RevieweeID)
	if err != nil {
		return nil, apperrors.Wrap(apperrors.CodeInternalError, err)
	}
	if !isReviewerMember || !isRevieweeMember {
		return nil, apperrors.CodeError(apperrors.CodeNotBothMembers)
	}

	// 验证未重复评价
	hasReviewed, err := s.repo.HasReviewed(ctx, projectID, reviewerID, req.RevieweeID)
	if err != nil {
		return nil, apperrors.Wrap(apperrors.CodeInternalError, err)
	}
	if hasReviewed {
		return nil, apperrors.CodeError(apperrors.CodeReviewDuplicate)
	}

	review := &model.CollaborationReview{
		ProjectID:  projectID,
		ReviewerID: reviewerID,
		RevieweeID: req.RevieweeID,
		Rating:     req.Rating,
		Comment:    req.Comment,
		Tags:       encodeLinks(req.Tags),
	}

	if err := s.repo.CreateReview(ctx, review); err != nil {
		return nil, apperrors.Wrap(apperrors.CodeInternalError, err)
	}

	// 通知被评价人（失败不阻断）
	reviewer, _ := s.userRepo.GetUserByID(ctx, reviewerID)
	reviewerName := "一位队友"
	if reviewer != nil && reviewer.Nickname != "" {
		reviewerName = reviewer.Nickname
	} else if reviewer != nil {
		reviewerName = reviewer.Username
	}
	s.writeNotification(ctx, req.RevieweeID, reviewerID,
		model.NotificationTypeSystem,
		"你收到了一条评价",
		fmt.Sprintf("%s 对你在 %s 的合作给出了 %d 星评价", reviewerName, project.Name, req.Rating),
		map[string]interface{}{
			"review_id":  review.ID,
			"project_id": projectID,
			"rating":     req.Rating,
		},
	)

	return review, nil
}
func (s *teamService) AddReviewSupplement(ctx context.Context, reviewerID, reviewID uint64, req *AddReviewSupplementReq) (*model.CollaborationReview, error) {
	// 验证补充说明长度和敏感词
	if len([]rune(req.Supplement)) > 200 {
		return nil, apperrors.New(apperrors.CodeParamTooLong, "补充说明不能超过200字")
	}
	if req.Supplement != "" && filter.Contains(req.Supplement) {
		return nil, apperrors.New(apperrors.CodeContentViolation, "补充说明包含敏感词，请修改后重试")
	}

	review, err := s.repo.GetReviewByID(ctx, reviewID)
	if err != nil {
		return nil, err
	}

	// 验证操作权限（只有评价人可以补充）
	if review.ReviewerID != reviewerID {
		return nil, apperrors.CodeError(apperrors.CodeReviewForbidden)
	}

	// AddSupplement 内部验证是否已有补充，会返回错误
	if err := s.repo.AddSupplement(ctx, reviewID, req.Supplement); err != nil {
		return nil, err
	}

	// 返回最新评价记录
	return s.repo.GetReviewByID(ctx, reviewID)
}

// GetUserReviews 获取用户收到的评价列表（含评价人基本信息）
func (s *teamService) GetUserReviews(ctx context.Context, userID uint64, page, pageSize int) ([]*ReviewDetail, int64, error) {
	// 根据用户名查找用户
	user, err := s.userRepo.GetUserByID(ctx, userID)
	if err != nil {
		return nil, 0, err
	}

	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}

	list, total, err := s.repo.GetReviewsByRevieweeID(ctx, user.ID, page, pageSize)
	if err != nil {
		return nil, 0, apperrors.Wrap(apperrors.CodeInternalError, err)
	}

	// 批量获取评价人信息（避免 N+1）
	reviwerIDSet := make(map[uint64]struct{}, len(list))
	for _, r := range list {
		reviwerIDSet[r.ReviewerID] = struct{}{}
	}
	reviwerIDs := make([]uint64, 0, len(reviwerIDSet))
	for id := range reviwerIDSet {
		reviwerIDs = append(reviwerIDs, id)
	}
	reviwerMap := make(map[uint64]*model.User, len(reviwerIDs))
	if len(reviwerIDs) > 0 {
		if users, err := s.userRepo.GetUsersByIDs(ctx, reviwerIDs); err == nil {
			for _, u := range users {
				reviwerMap[u.ID] = u
			}
		}
	}

	items := make([]*ReviewDetail, 0, len(list))
	for _, review := range list {
		detail := &ReviewDetail{
			ID:         review.ID,
			ProjectID:  review.ProjectID,
			ReviewerID: review.ReviewerID,
			RevieweeID: review.RevieweeID,
			Rating:     review.Rating,
			Comment:    review.Comment,
			Tags:       decodeLinks(review.Tags),
			Supplement: review.Supplement,
			CreatedAt:  review.CreatedAt,
			UpdatedAt:  review.UpdatedAt,
		}
		if reviewer, ok := reviwerMap[review.ReviewerID]; ok {
			detail.ReviewerUsername = reviewer.Username
			detail.ReviewerNickname = reviewer.Nickname
			detail.ReviewerAvatarURL = s.presignURL(ctx, reviewer.AvatarKey)
		}
		items = append(items, detail)
	}

	return items, total, nil
}

// GetUserRatingSummary 获取用户评分摘要
func (s *teamService) GetUserRatingSummary(ctx context.Context, userID uint64) (*RatingSummary, error) {
	user, err := s.userRepo.GetUserByID(ctx, userID)
	if err != nil {
		return nil, err
	}

	avg, count, err := s.repo.GetAverageRating(ctx, user.ID)
	if err != nil {
		return nil, apperrors.Wrap(apperrors.CodeInternalError, err)
	}

	return &RatingSummary{
		AverageRating: avg,
		Count:         count,
	}, nil
}

// ===========================
// 人才库实现
// ===========================

// ListTalents 人才库列表查询
// 排序：先按参与项目数倒序，再按注册时间倒序
func (s *teamService) ListTalents(ctx context.Context, req *ListTalentsReq) ([]*TalentListItem, int64, error) {
	if req.Page < 1 {
		req.Page = 1
	}
	if req.PageSize < 1 {
		req.PageSize = 20
	}
	if req.PageSize > 100 {
		req.PageSize = 100
	}

	// 通过 userRepo 查询人才库
	talentRepo, ok := s.userRepo.(repository.TalentQuerier)
	if !ok {
		return nil, 0, apperrors.New(apperrors.CodeInternalError, "人才库查询不支持")
	}

	params := &repository.ListTalentsParams{
		Page:           req.Page,
		PageSize:       req.PageSize,
		Category:       req.Category,
		Keyword:        req.Keyword,
		Level:          req.Level,
		OnlyAvailable:  req.OnlyAvailable,
		CoopPreference: req.CoopPreference,
		Location:       req.Location,
	}

	users, total, err := talentRepo.ListTalents(ctx, params)
	if err != nil {
		return nil, 0, apperrors.Wrap(apperrors.CodeInternalError, err)
	}

	items := make([]*TalentListItem, 0, len(users))
	for _, u := range users {
		item := &TalentListItem{
			ID:             u.ID,
			Username:       u.Username,
			Nickname:       u.Nickname,
			AvatarURL:      s.presignURL(ctx, u.AvatarKey),
			Bio:            u.Bio,
			Skills:         u.Skills,
			ProjectCount:   u.ProjectCount,
			IsAvailable:    u.IsAvailable,
			CoopPreference: u.CoopPreference,
			Location:       u.Location,
		}
		items = append(items, item)
	}

	return items, total, nil
}

// SetAvailability 设置自己是否开放合作
func (s *teamService) SetAvailability(ctx context.Context, userID uint64, req *SetAvailabilityReq) error {
	updates := map[string]interface{}{
		"is_available": req.IsAvailable,
	}
	if req.CoopPreference != "" {
		if _, valid := model.ValidCooperationTypes[model.CooperationType(req.CoopPreference)]; !valid {
			return apperrors.New(apperrors.CodeParamInvalid, "无效的合作方式偏好")
		}
		updates["coop_preference"] = req.CoopPreference
	} else {
		updates["coop_preference"] = nil
	}
	if len(req.Location) > 64 {
		return apperrors.New(apperrors.CodeParamInvalid, "地区信息最多64个字符")
	}
	updates["location"] = req.Location

	return s.userRepo.UpdateUser(ctx, userID, updates)
}

// ================================
// 人才邀请实现
// ================================

const invitationExpireDays = 7 // 邀请有效期（天）

// InviteTalent 项目主向人才发送邀请
func (s *teamService) InviteTalent(ctx context.Context, inviterID, projectID, talentID uint64, req *InviteTalentReq) (*model.TalentInvitation, error) {
	if s.invitationRepo == nil {
		return nil, apperrors.New(apperrors.CodeInternalError, "邀请功能未初始化")
	}

	// 验证项目存在且 inviterID 是 owner
	project, err := s.projectRepo.GetProjectByID(ctx, projectID)
	if err != nil {
		return nil, err
	}
	if project.OwnerID != inviterID {
		return nil, apperrors.New(apperrors.CodeForbidden, "只有项目负责人可以发送邀请")
	}

	// 不能邀请自己（提前到 IsAvailable 检查之前，语义更准确）
	if talentID == inviterID {
		return nil, apperrors.New(apperrors.CodeParamInvalid, "不能邀请自己")
	}

	// 验证被邀请人存在
	talent, err := s.userRepo.GetUserByID(ctx, talentID)
	if err != nil {
		return nil, apperrors.New(apperrors.CodeNotFound, "用户不存在")
	}
	if !talent.IsAvailable {
		return nil, apperrors.New(apperrors.CodeParamInvalid, "该用户未开放合作邀请")
	}

	// 验证岗位
	position := model.RecruitmentPosition(req.Position)
	if !model.ValidRecruitmentPositions[position] {
		return nil, apperrors.New(apperrors.CodeParamInvalid, "无效的邀请岗位")
	}

	// 清理已终结的旧邀请（declined/withdrawn/expired），使重新邀请不受唯一索引阻挡
	if err := s.invitationRepo.DeleteTerminatedInvitation(ctx, projectID, talentID, position); err != nil {
		return nil, apperrors.Wrap(apperrors.CodeInternalError, err)
	}

	// 检查是否已有待处理邀请
	existing, err := s.invitationRepo.GetPendingInvitation(ctx, projectID, talentID, position)
	if err != nil {
		return nil, apperrors.Wrap(apperrors.CodeInternalError, err)
	}
	if existing != nil {
		return nil, apperrors.New(apperrors.CodeConflict, "已存在待处理的邀请，请等待对方回应")
	}

	// 验证说明长度
	if len([]rune(req.Message)) > 200 {
		return nil, apperrors.New(apperrors.CodeParamInvalid, "邀请说明最多200字")
	}

	inv := &model.TalentInvitation{
		ProjectID: projectID,
		InviterID: inviterID,
		TalentID:  talentID,
		Position:  position,
		Message:   req.Message,
		Status:    model.InvitationStatusPending,
		ExpireAt:  time.Now().AddDate(0, 0, invitationExpireDays),
	}
	if err := s.invitationRepo.CreateInvitation(ctx, inv); err != nil {
		return nil, apperrors.Wrap(apperrors.CodeInternalError, err)
	}

	// 推送通知给被邀请人
	s.writeNotification(ctx, talentID, inviterID, model.NotificationTypeTalentInvited,
		"你收到了一封项目邀请",
		fmt.Sprintf("项目《%s》邀请你担任 %s", project.Name, req.Position),
		map[string]interface{}{"invitation_id": inv.ID, "project_id": projectID})

	return inv, nil
}

// WithdrawInvitation 撤回邀请
func (s *teamService) WithdrawInvitation(ctx context.Context, inviterID, invitationID uint64) error {
	if s.invitationRepo == nil {
		return apperrors.New(apperrors.CodeInternalError, "邀请功能未初始化")
	}
	inv, err := s.invitationRepo.GetInvitationByID(ctx, invitationID)
	if err != nil {
		return err
	}
	if inv.InviterID != inviterID {
		return apperrors.New(apperrors.CodeForbidden, "只能撤回自己发送的邀请")
	}
	if inv.Status != model.InvitationStatusPending {
		return apperrors.New(apperrors.CodeParamInvalid, "只能撤回待处理的邀请")
	}
	return s.invitationRepo.UpdateInvitationStatus(ctx, invitationID, model.InvitationStatusWithdrawn)
}

// RespondInvitation 被邀请人回应邀请
func (s *teamService) RespondInvitation(ctx context.Context, talentID, invitationID uint64, accept bool) error {
	if s.invitationRepo == nil {
		return apperrors.New(apperrors.CodeInternalError, "邀请功能未初始化")
	}
	inv, err := s.invitationRepo.GetInvitationByID(ctx, invitationID)
	if err != nil {
		return err
	}
	if inv.TalentID != talentID {
		return apperrors.New(apperrors.CodeForbidden, "只能回应自己收到的邀请")
	}
	if inv.Status != model.InvitationStatusPending {
		return apperrors.New(apperrors.CodeParamInvalid, "该邀请已处理或已过期")
	}
	if inv.IsExpired() {
		if err := s.invitationRepo.UpdateInvitationStatus(ctx, invitationID, model.InvitationStatusExpired); err != nil {
			logger.Warn("failed to update invitation status", zap.Error(err))
		}
		return apperrors.New(apperrors.CodeParamInvalid, "该邀请已过期")
	}

	newStatus := model.InvitationStatusDeclined
	notifType := model.NotificationTypeInviteDeclined
	notifTitle := "邀请被拒绝"
	if accept {
		newStatus = model.InvitationStatusAccepted
		notifType = model.NotificationTypeInviteAccepted
		notifTitle = "邀请已接受"
	}
	if err := s.invitationRepo.UpdateInvitationStatus(ctx, invitationID, newStatus); err != nil {
		return err
	}

	// 邀请接受：写入 project_members，并联动群聊成员
	if accept {
		if addErr := s.projectRepo.AddMember(ctx, &model.ProjectMember{
			ProjectID: inv.ProjectID,
			UserID:    talentID,
			Role:      model.ProjectMemberRoleMember,
		}); addErr != nil {
			// 已是成员时跳过，其他错误仅记录日志，不回滚已更新的邀请状态
			if ae, ok := apperrors.IsAppError(addErr); !ok || ae.Code != apperrors.CodeMemberAlreadyExists {
				logger.Warn("failed to add project member on invitation accept", zap.Uint64("project_id", inv.ProjectID), zap.Uint64("user_id", talentID), zap.Error(addErr))
			}
		}
	}

	// 通知邀请方
	s.writeNotification(ctx, inv.InviterID, talentID, notifType, notifTitle,
		fmt.Sprintf("你发送的邀请已被%s", map[bool]string{true: "接受", false: "拒绝"}[accept]),
		map[string]interface{}{"invitation_id": invitationID})

	return nil
}

// GetMyInvitations 获取我收到的邀请列表
// validInvitationStatusFilter 邀请列表允许的 status 过滤值（空字符串表示不过滤）
var validInvitationStatusFilter = map[string]bool{
	"":                                      true,
	string(model.InvitationStatusPending):   true,
	string(model.InvitationStatusAccepted):  true,
	string(model.InvitationStatusDeclined):  true,
	string(model.InvitationStatusWithdrawn): true,
	string(model.InvitationStatusExpired):   true,
}

func (s *teamService) GetMyInvitations(ctx context.Context, talentID uint64, status string, page, pageSize int) ([]*InvitationDetail, int64, error) {
	if s.invitationRepo == nil {
		return nil, 0, apperrors.New(apperrors.CodeInternalError, "邀请功能未初始化")
	}
	if !validInvitationStatusFilter[status] {
		return nil, 0, apperrors.New(apperrors.CodeParamInvalid, "status 参数非法")
	}
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 20
	}
	offset := (page - 1) * pageSize
	items, total, err := s.invitationRepo.ListInvitationsByTalent(ctx, talentID, status, offset, pageSize)
	if err != nil {
		return nil, 0, apperrors.Wrap(apperrors.CodeInternalError, err)
	}
	details, err := s.enrichInvitations(ctx, items)
	return details, total, err
}

// GetProjectInvitations 获取项目发出的邀请列表（owner 查看）
func (s *teamService) GetProjectInvitations(ctx context.Context, inviterID, projectID uint64, status string, page, pageSize int) ([]*InvitationDetail, int64, error) {
	if s.invitationRepo == nil {
		return nil, 0, apperrors.New(apperrors.CodeInternalError, "邀请功能未初始化")
	}
	if !validInvitationStatusFilter[status] {
		return nil, 0, apperrors.New(apperrors.CodeParamInvalid, "status 参数非法")
	}
	project, err := s.projectRepo.GetProjectByID(ctx, projectID)
	if err != nil {
		return nil, 0, err
	}
	if project.OwnerID != inviterID {
		return nil, 0, apperrors.New(apperrors.CodeForbidden, "只有项目负责人可以查看邀请列表")
	}
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 20
	}
	offset := (page - 1) * pageSize
	items, total, err := s.invitationRepo.ListInvitationsByProject(ctx, projectID, status, offset, pageSize)
	if err != nil {
		return nil, 0, apperrors.Wrap(apperrors.CodeInternalError, err)
	}
	details, err := s.enrichInvitations(ctx, items)
	return details, total, err
}

// enrichInvitations 为邀请列表批量填充项目名称和用户信息
func (s *teamService) enrichInvitations(ctx context.Context, items []*model.TalentInvitation) ([]*InvitationDetail, error) {
	if len(items) == 0 {
		return nil, nil
	}

	// 收集项目 ID 和用户 ID（進行批量查询，避免 N+1）
	projectIDSet := make(map[uint64]struct{}, len(items))
	userIDSet := make(map[uint64]struct{}, len(items)*2)
	for _, inv := range items {
		projectIDSet[inv.ProjectID] = struct{}{}
		userIDSet[inv.InviterID] = struct{}{}
		userIDSet[inv.TalentID] = struct{}{}
	}

	// 批量查项目名（项目只需要 Name，逐个查即可，但应尽量少查）
	projectMap := make(map[uint64]string, len(projectIDSet))
	for pid := range projectIDSet {
		if p, err := s.projectRepo.GetProjectByID(ctx, pid); err == nil {
			projectMap[pid] = p.Name
		}
	}

	// 批量查用户（一次 IN 查询）
	userIDs := make([]uint64, 0, len(userIDSet))
	for uid := range userIDSet {
		userIDs = append(userIDs, uid)
	}
	userMap := make(map[uint64]*model.User, len(userIDs))
	if users, err := s.userRepo.GetUsersByIDs(ctx, userIDs); err == nil {
		for _, u := range users {
			userMap[u.ID] = u
		}
	}

	details := make([]*InvitationDetail, 0, len(items))
	for _, inv := range items {
		d := &InvitationDetail{TalentInvitation: inv}
		d.ProjectName = projectMap[inv.ProjectID]
		if inviter, ok := userMap[inv.InviterID]; ok {
			d.InviterUsername = inviter.Username
			d.InviterNickname = inviter.Nickname
			d.InviterAvatarURL = s.presignURL(ctx, inviter.AvatarKey)
		}
		if talent, ok := userMap[inv.TalentID]; ok {
			d.TalentUsername = talent.Username
			d.TalentNickname = talent.Nickname
			d.TalentAvatarURL = s.presignURL(ctx, talent.AvatarKey)
		}
		details = append(details, d)
	}
	return details, nil
}

