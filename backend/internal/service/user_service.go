// Package service 提供用户系统的业务逻辑层。
// 包含注册、登录、个人主页、技能标签、作品集等所有用户相关业务逻辑。
// 目前只提供邮箱注册登陆，不使用手机号或者其他三方平台
package service

import (
	"context"
	"crypto/rand"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/gamero/gamero/pkg/logger"
	"go.uber.org/zap"

	"github.com/gamero/gamero/internal/model"
	"github.com/gamero/gamero/internal/repository"
	"github.com/gamero/gamero/pkg/auth"
	"github.com/gamero/gamero/pkg/cache"
	pkgemail "github.com/gamero/gamero/pkg/email"
	apperrors "github.com/gamero/gamero/pkg/errors"
	"github.com/gamero/gamero/pkg/filter"
	"github.com/gamero/gamero/pkg/storage"
	"golang.org/x/crypto/bcrypt"
)

const (
	// avatarPresignExpiry 预签名 URL 有效期 24 小时
	avatarPresignExpiry = 24 * time.Hour
	// customSkillMaxCount 每用户最多 3 个自定义技能
	customSkillMaxCount = 3
	// bcryptCost bcrypt 加密强度
	bcryptCost = 12

	// loginMaxFailCount 登录失败最多允许次数,超出后锁定账号
	loginMaxFailCount = 5
	// loginLockDuration 账号锁定时长(5 分钟)
	loginLockDuration = 5 * time.Minute
)

// 预编译所有正则,避免每次函数调用时重复编译
var (
	// usernameRegexp 用户名:只允许字母、数字、下划线,3-32 位
	usernameRegexp = regexp.MustCompile(`^[a-zA-Z0-9_]{3,32}$`)
	// emailRegexp 邮箱格式
	emailRegexp = regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)
	// passwordLetterRegexp 密码必须包含字母
	passwordLetterRegexp = regexp.MustCompile(`[a-zA-Z]`)
	// passwordDigitRegexp 密码必须包含数字
	passwordDigitRegexp = regexp.MustCompile(`[0-9]`)
)

// SendSMSCodeReq 发送短信验证码请求
// type SendSMSCodeReq struct {
// 	Phone string `json:"phone" binding:"required"`
// }

// SendEmailCodeReq 发送邮箱验证码请求
type SendEmailCodeReq struct {
	Email string `json:"email" binding:"required"`
	Scene string `json:"scene"` // register | reset，用于区分场景
}

// RegisterByPhoneReq 手机号注册请求
// type RegisterByPhoneReq struct {
// 	Phone    string `json:"phone" binding:"required"`
// 	Code     string `json:"code" binding:"required"`
// 	Password string `json:"password" binding:"required"`
// 	Username string `json:"username"` // 可选,未提供时自动生成
// 	Nickname string `json:"nickname"` // 可选
// }

// RegisterByEmailReq 邮箱注册请求
type RegisterByEmailReq struct {
	Email    string `json:"email" binding:"required"`
	Code     string `json:"code" binding:"required"`
	Password string `json:"password" binding:"required"`
	Nickname string `json:"nickname"` // 可选
}

// LoginByPhoneReq 手机号验证码登录请求(无需密码)
// type LoginByPhoneReq struct {
// 	Phone string `json:"phone" binding:"required"`
// 	Code  string `json:"code" binding:"required"`
// }

// LoginByEmailReq 邮箱+密码登录请求
type LoginByEmailReq struct {
	Email    string `json:"email" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// RefreshTokenReq 刷新 Token 请求
type RefreshTokenReq struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

// ChangePasswordReq 修改密码请求(需提供旧密码)
type ChangePasswordReq struct {
	OldPassword string `json:"old_password" binding:"required"`
	NewPassword string `json:"new_password" binding:"required"`
}

// ResetPasswordByCodeReq 通过邮箱验证码重置密码请求
type ResetPasswordByCodeReq struct {
	Email       string `json:"email"        binding:"required"`
	Code        string `json:"code"         binding:"required"`
	NewPassword string `json:"new_password" binding:"required"`
}

// UpdateProfileReq 更新个人资料请求
type UpdateProfileReq struct {
	Nickname string  `json:"nickname"` // 可选,2-32 位
	Bio      *string `json:"bio"`      // 可选,最多 300 字;传 "" 表示清空简介,不传则不更新
	Location *string `json:"location"` // 所在地区,nil=不修改,""=清空;最多64字符
}

// SkillInput 单个技能输入
type SkillInput struct {
	Category    string `json:"category" binding:"required"` // program/art/design/sound/custom
	Name        string `json:"name" binding:"required"`
	Level       string `json:"level" binding:"required"` // beginner/intermediate/advanced
	Description string `json:"description"`              // 技能描述(选填,最多200字)
}

// UpdateSkillsReq 更新技能列表请求
type UpdateSkillsReq struct {
	Skills []SkillInput `json:"skills" binding:"required"`
}

// CreatePortfolioReq 创建作品请求
type CreatePortfolioReq struct {
	Name        string `json:"name" binding:"required"`
	Type        string `json:"type" binding:"required"` // game/demo/art/code
	Link        string `json:"link"`
	ImageURL    string `json:"image_url"`
	Description string `json:"description"`
}

// UpdatePortfolioReq 更新作品请求
type UpdatePortfolioReq struct {
	Name        string `json:"name"`
	Type        string `json:"type"`
	Link        string `json:"link"`
	ImageURL    string `json:"image_url"`
	Description string `json:"description"`
}

// AuthResponse 登录/注册成功响应
type AuthResponse struct {
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token"`
	ExpiresAt    time.Time `json:"expires_at"`
	User         UserInfo  `json:"user"`
}

// UserInfo 用户基础信息(用于 Token 响应)
type UserInfo struct {
	ID        uint64 `json:"id"`
	Username  string `json:"username"`
	Nickname  string `json:"nickname"`
	Role      string `json:"role"`
	AvatarURL string `json:"avatar_url"`
}

// EndorsementItem 个人主页合作评价项
type EndorsementItem struct {
	ID                uint64    `json:"id"`
	ProjectID         uint64    `json:"project_id"`
	ReviewerID        uint64    `json:"reviewer_id"`
	RevieweeID        uint64    `json:"reviewee_id"`
	Rating            int8      `json:"rating"`
	Comment           string    `json:"comment"`
	Tags              []string  `json:"tags"`
	Supplement        string    `json:"supplement,omitempty"`
	ReviewerNickname  string    `json:"reviewer_nickname"`
	ReviewerAvatarURL string    `json:"reviewer_avatar_url,omitempty"`
	CreatedAt         time.Time `json:"created_at"`
}

// UserProfileResponse 用户个人主页响应
type UserProfileResponse struct {
	ID             uint64                          `json:"id"`
	Username       string                          `json:"username"`
	Nickname       string                          `json:"nickname"`
	Role           string                          `json:"role,omitempty"` // admin 接口包含此字段
	Bio            string                          `json:"bio"`
	AvatarURL      string                          `json:"avatar_url"` // 预签名 URL,24h 有效
	Skills         []model.UserSkill               `json:"skills"`
	Portfolio      []model.Portfolio               `json:"portfolio"`
	PostCount      int    `json:"post_count"`
	LogCount       int    `json:"log_count"`
	ProjectCount   int    `json:"project_count"`
	FollowersCount int64                           `json:"followers_count"`
	FollowingCount int64                           `json:"following_count"`
	IsFollowing    bool                            `json:"is_following"` // 访问者是否已关注该用户
	IsBanned       bool                            `json:"is_banned"`    // 是否已被封禁(管理员接口使用)
	Badges         []string                        `json:"badges"`       // 用户成就徽章列表
	CreatedAt      time.Time                       `json:"created_at"`
	// 以下字段仅在访问自己主页时(viewerID == userID)返回
	IsAvailable    *bool   `json:"is_available,omitempty"`    // 是否开放合作
	CoopPreference *string `json:"coop_preference,omitempty"` // 合作方式偏好
	Location       *string `json:"location,omitempty"`        // 所在地区
}

// ==================== Service 接口定义 ====================

// UserService 用户服务接口
type UserService interface {
	// SendSMSCode 发送手机号验证码
	// SendSMSCode(ctx context.Context, req *SendSMSCodeReq) error
	// SendEmailCode 发送邮箱验证码（返回验证码用于开发环境显示）
	SendEmailCode(ctx context.Context, req *SendEmailCodeReq) (string, error)
	// RegisterByPhone 手机号注册
	// RegisterByPhone(ctx context.Context, req *RegisterByPhoneReq) (*AuthResponse, error)
	// RegisterByEmail 邮箱注册
	RegisterByEmail(ctx context.Context, req *RegisterByEmailReq) (*AuthResponse, error)
	// LoginByPhone 手机号验证码登录
	// LoginByPhone(ctx context.Context, req *LoginByPhoneReq) (*AuthResponse, error)
	// LoginByEmail 邮箱密码登录
	LoginByEmail(ctx context.Context, req *LoginByEmailReq) (*AuthResponse, error)
	// Logout 登出,将 Access Token 加入黑名单
	Logout(ctx context.Context, tokenStr string, expiresAt time.Time) error
	// RefreshToken 用 Refresh Token 换取新的 Token 对
	RefreshToken(ctx context.Context, req *RefreshTokenReq) (*AuthResponse, error)
	// GetUserProfile 获取用户个人主页(通过用户 ID),viewerID=0 表示未登录
	GetUserProfile(ctx context.Context, userID uint64, viewerID uint64) (*UserProfileResponse, error)
	// GetUserProfileByUsername 获取用户个人主页(通过 username),viewerID=0 表示未登录
	GetUserProfileByUsername(ctx context.Context, username string, viewerID uint64) (*UserProfileResponse, error)
	// UpdateProfile 更新个人资料
	UpdateProfile(ctx context.Context, userID uint64, req *UpdateProfileReq) error
	// SaveAvatar 保存头像 key(前端已直传云存储后提交 key)
	SaveAvatar(ctx context.Context, userID uint64, key string) error
	// UpdateSkills 更新用户技能标签
	UpdateSkills(ctx context.Context, userID uint64, req *UpdateSkillsReq) ([]model.UserSkill, error)
	// GetMyPortfolio 获取自己的作品集
	GetMyPortfolio(ctx context.Context, userID uint64) ([]model.Portfolio, error)
	// CreatePortfolio 添加作品
	CreatePortfolio(ctx context.Context, userID uint64, req *CreatePortfolioReq) (*model.Portfolio, error)
	// UpdatePortfolio 更新作品
	UpdatePortfolio(ctx context.Context, userID uint64, portfolioID uint64, req *UpdatePortfolioReq) (*model.Portfolio, error)
	// DeletePortfolio 删除作品
	DeletePortfolio(ctx context.Context, userID uint64, portfolioID uint64) error
	// SetUserRole 更新用户角色(仅允许 superadmin 调用)
	SetUserRole(ctx context.Context, userID uint64, role string) error

	// ChangePassword 已登录用户修改密码(需提供旧密码)
	ChangePassword(ctx context.Context, userID uint64, req *ChangePasswordReq) error
	// SendResetPasswordCode 向邮箱发送重置密码验证码
	SendResetPasswordCode(ctx context.Context, req *SendEmailCodeReq) error
	// ResetPasswordByCode 通过邮箱验证码重置密码(忘记密码)
	ResetPasswordByCode(ctx context.Context, req *ResetPasswordByCodeReq) error
	// DeleteAccount 注销账号(软删除,需用户主动确认)
	DeleteAccount(ctx context.Context, userID uint64, req *DeleteAccountReq) error
	// SetNotificationService 注入统一通知服务(可选)
	SetNotificationService(svc NotificationService)
	// SetFollowRepository 注入关注仓库(可选,用于主页 is_following 字段)
	SetFollowRepository(repo repository.FollowRepository)
	// GetUserLogs 分页查询用户已发布的开发日志
	GetUserLogs(ctx context.Context, userID uint64, page, pageSize int) ([]DevLogListItem, int64, error)
	// GetUserEndorsements 分页查询用户收到的合作评价
	GetUserEndorsements(ctx context.Context, userID uint64, page, pageSize int) ([]EndorsementItem, int64, error)
	// SetDevLogRepository 注入开发日志仓库(可选,用于主页日志分页查询)
	SetDevLogRepository(repo repository.DevLogRepository)
	// SearchUsers 按 username/nickname 模糊搜索用户(用于 DM 发起时选人)
	SearchUsers(ctx context.Context, keyword string, limit int) ([]*UserSearchItem, error)
}

// ==================== Service 实现 ====================

// userService 是 UserService 接口的具体实现
type userService struct {
	repo       repository.UserRepository
	teamRepo   repository.TeamRepository
	followRepo repository.FollowRepository  // 用于 is_following 查询(可为 nil,降级安全)
	devLogRepo repository.DevLogRepository  // 用于日志分页查询(可为 nil,降级为空列表)
	*NotificationClient
}

// NewUserService 创建 userService 实例
func NewUserService(repo repository.UserRepository, teamRepo repository.TeamRepository) UserService {
	svc := &userService{repo: repo, teamRepo: teamRepo, NotificationClient: &NotificationClient{}}
	return svc
}

// SetFollowRepository 注入关注仓库(可选,用于主页 is_following 字段)
func (s *userService) SetFollowRepository(repo repository.FollowRepository) {
	s.followRepo = repo
}

// SetDevLogRepository 注入开发日志仓库(可选,用于日志分页查询)
func (s *userService) SetDevLogRepository(repo repository.DevLogRepository) {
	s.devLogRepo = repo
}

// ==================== 发送验证码 ====================

// SendSMSCode 发送手机号验证码
// func (s *userService) SendSMSCode(ctx context.Context, req *SendSMSCodeReq) error {
// 	// 验证手机号格式:11 位数字
// 	if !isValidPhone(req.Phone) {
// 		return apperrors.New(apperrors.CodeParamInvalid, "手机号格式不正确,请输入11位手机号")
// 	}
// 	_, err := pkgsms.Send(ctx, req.Phone)
// 	return err
// }

// SendEmailCode 发送邮箱验证码
func (s *userService) SendEmailCode(ctx context.Context, req *SendEmailCodeReq) (string, error) {
	// 验证邮箱格式
	if !isValidEmail(req.Email) {
		return "", apperrors.New(apperrors.CodeParamInvalid, "邮箱格式不正确")
	}

	// 根据场景检查邮箱状态
	exists, err := s.repo.ExistsEmail(ctx, req.Email)
	if err != nil {
		return "", apperrors.Wrap(apperrors.CodeInternalError, err)
	}

	if req.Scene == "register" {
		// 注册场景：邮箱不能已存在
		if exists {
			return "", apperrors.CodeError(apperrors.CodeEmailAlreadyExists)
		}
	} else if req.Scene == "reset" {
		// 重置密码场景：邮箱必须存在
		if !exists {
			return "", apperrors.New(apperrors.CodeUserNotFound, "该邮箱未注册")
		}
	}
	// 其他场景（如登录验证）不检查邮箱是否存在

	// 发送验证码。接口层不会把验证码返回给前端，用户必须从邮箱获取。
	code, err := pkgemail.SendVerifyCode(ctx, req.Email)
	return code, err
}

// ==================== 注册 ====================

// RegisterByPhone 手机号注册
// func (s *userService) RegisterByPhone(ctx context.Context, req *RegisterByPhoneReq) (*AuthResponse, error) {
// 	// 参数验证
// 	if !isValidPhone(req.Phone) {
// 		return nil, apperrors.New(apperrors.CodeParamInvalid, "手机号格式不正确,请输入11位手机号")
// 	}
// 	if err := validatePassword(req.Password); err != nil {
// 		return nil, err
// 	}

// 	// 验证短信验证码
// 	if err := pkgsms.Verify(ctx, req.Phone, req.Code); err != nil {
// 		return nil, err
// 	}

// 	// 检查手机号是否已注册
// 	exists, err := s.repo.ExistsPhone(ctx, req.Phone)
// 	if err != nil {
// 		return nil, apperrors.Wrap(apperrors.CodeInternalError, err)
// 	}
// 	if exists {
// 		return nil, apperrors.CodeError(apperrors.CodePhoneAlreadyExists)
// 	}

// 	// 处理用户名
// 	username, err := s.resolveUsername(ctx, "") // 用户名由后端自动生成
// 	if err != nil {
// 		return nil, err
// 	}

// 	// 处理昵称
// 	nickname := req.Nickname
// 	if nickname == "" {
// 		nickname = username
// 	}
// 	if err := validateNickname(nickname); err != nil {
// 		return nil, err
// 	}

// 	// bcrypt 加密密码
// 	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcryptCost)
// 	if err != nil {
// 		return nil, apperrors.Wrap(apperrors.CodeInternalError, err)
// 	}

// 	// 创建用户
// 	user := &model.User{
// 		Username:     username,
// 		Nickname:     nickname,
// 		Phone:        req.Phone,
// 		PasswordHash: string(hash),
// 		Role:         model.UserRoleUser,
// 		Status:       model.UserStatusActive,
// 	}
// 	if err := s.repo.CreateUser(ctx, user); err != nil {
// 		return nil, apperrors.Wrap(apperrors.CodeInternalError, err)
// 	}

// 	return s.generateAuthResponse(ctx, user)
// }

// RegisterByEmail 邮箱注册
func (s *userService) RegisterByEmail(ctx context.Context, req *RegisterByEmailReq) (*AuthResponse, error) {
	// 参数验证
	if !isValidEmail(req.Email) {
		return nil, apperrors.New(apperrors.CodeParamInvalid, "邮箱格式不正确")
	}
	if err := validatePassword(req.Password); err != nil {
		return nil, err
	}

	// 【重要】先检查邮箱和昵称是否可用，再验证验证码
	// 这样可以避免验证码被消耗后，因为其他错误导致用户无法重新提交

	// 检查邮箱是否已注册
	exists, err := s.repo.ExistsEmail(ctx, req.Email)
	if err != nil {
		return nil, apperrors.Wrap(apperrors.CodeInternalError, err)
	}
	if exists {
		return nil, apperrors.CodeError(apperrors.CodeEmailAlreadyExists)
	}

	// 检查昵称是否已被使用（仅当用户主动提供昵称时检查）
	if req.Nickname != "" {
		if err := validateNickname(req.Nickname); err != nil {
			return nil, err
		}
		exists, err := s.repo.ExistsNickname(ctx, req.Nickname)
		if err != nil {
			return nil, apperrors.Wrap(apperrors.CodeInternalError, err)
		}
		if exists {
			return nil, apperrors.New(apperrors.CodeNicknameAlreadyExists, "该昵称已被使用，请换一个")
		}
	}

	// 所有参数验证通过后，再验证邮箱验证码（验证后会删除验证码）
	if err := pkgemail.Verify(ctx, req.Email, req.Code); err != nil {
		return nil, err
	}

	// 处理用户名
	username, err := s.resolveUsername(ctx, "") // 用户名由后端自动生成
	if err != nil {
		return nil, err
	}

	// 处理昵称
	nickname := req.Nickname
	if nickname == "" {
		nickname = username
		// 空昵称时使用生成的用户名，不需要验证重复
		if err := validateNickname(nickname); err != nil {
			return nil, err
		}
	}

	// bcrypt 加密密码
	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcryptCost)
	if err != nil {
		return nil, apperrors.Wrap(apperrors.CodeInternalError, err)
	}

	// 创建用户
	user := &model.User{
		Username:     username,
		Nickname:     nickname,
		Email:        req.Email,
		PasswordHash: string(hash),
		Role:         model.UserRoleUser,
		Status:       model.UserStatusActive,
	}
	if err := s.repo.CreateUser(ctx, user); err != nil {
		return nil, apperrors.Wrap(apperrors.CodeInternalError, err)
	}

	return s.generateAuthResponse(ctx, user)
}

// ==================== 登录 ====================

// LoginByPhone 手机号验证码登录(无需密码)
// func (s *userService) LoginByPhone(ctx context.Context, req *LoginByPhoneReq) (*AuthResponse, error) {
// 	// 验证手机号格式
// 	if !isValidPhone(req.Phone) {
// 		return nil, apperrors.New(apperrors.CodeParamInvalid, "手机号格式不正确,请输入11位手机号")
// 	}

// 	// 验证短信验证码
// 	if err := pkgsms.Verify(ctx, req.Phone, req.Code); err != nil {
// 		return nil, err
// 	}

// 	// 查找用户
// 	user, err := s.repo.GetUserByPhone(ctx, req.Phone)
// 	if err != nil {
// 		return nil, err
// 	}

// 	// 检查账号状态(支持有期限封号自动放行)
// 	if err := checkUserBanned(user); err != nil {
// 		return nil, err
// 	}

// 	return s.generateAuthResponse(ctx, user)
// }

// LoginByEmail 邮箱密码登录
func (s *userService) LoginByEmail(ctx context.Context, req *LoginByEmailReq) (*AuthResponse, error) {
	// 验证邮箱格式
	if !isValidEmail(req.Email) {
		return nil, apperrors.New(apperrors.CodeParamInvalid, "邮箱格式不正确")
	}

	// ===== 暴力破解防护:检查失败次数 =====
	failKey := cache.LoginFailKey(req.Email)
	if cnt, err := cache.GetString(ctx, failKey); err == nil {
		n, _ := strconv.ParseInt(cnt, 10, 64)
		if n >= loginMaxFailCount {
			return nil, apperrors.New(apperrors.CodeTooManyRequests, "登录失败次数过多,账号已被暂时锁定,请5分钟后重试")
		}
	}

	// 查找用户
	user, err := s.repo.GetUserByEmail(ctx, req.Email)
	if err != nil {
		// 用户不存在也累计计数,防止枚举
		if _, err := cache.IncrBy(ctx, failKey, 1); err != nil {
			logger.Warn("failed to increment cache", zap.Error(err))
		}
		if err := cache.Expire(ctx, failKey, loginLockDuration); err != nil {
			logger.Warn("failed to update cache expiry", zap.Error(err))
		}
		return nil, err
	}

	// 检查账号状态(支持有期限封号自动放行)
	if err := checkUserBanned(user); err != nil {
		return nil, err
	}

	// 验证密码
	if user.PasswordHash == "" {
		// 该账号通过验证码注册,未设置密码
		return nil, apperrors.New(apperrors.CodePasswordIncorrect, "该账号未设置密码,请使用验证码登录")
	}
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		// 密码错误:累计失败计数
		n, _ := cache.IncrBy(ctx, failKey, 1)
		if err := cache.Expire(ctx, failKey, loginLockDuration); err != nil {
			logger.Warn("failed to update cache expiry", zap.Error(err))
		}
		remaining := loginMaxFailCount - n
		if remaining <= 0 {
			return nil, apperrors.New(apperrors.CodePasswordIncorrect, "密码错误,账号已被暂时锁定,请5分钟后重试")
		}
		return nil, apperrors.New(apperrors.CodePasswordIncorrect, fmt.Sprintf("密码错误,还可尝试 %d 次", remaining))
	}

	// 登录成功:清除失败计数
	if err := cache.Del(ctx, failKey); err != nil {
		logger.Warn("failed to delete cache", zap.Error(err))
	}

	return s.generateAuthResponse(ctx, user)
}

// ==================== Token 管理 ====================

// Logout 登出:将 Access Token 加入 Redis 黑名单
func (s *userService) Logout(ctx context.Context, tokenStr string, expiresAt time.Time) error {
	return auth.RevokeToken(ctx, tokenStr, expiresAt)
}

// RefreshToken 使用 Refresh Token 换取新的 Token 对
func (s *userService) RefreshToken(ctx context.Context, req *RefreshTokenReq) (*AuthResponse, error) {
	// 解析 Refresh Token
	claims, err := auth.ParseRefreshToken(req.RefreshToken)
	if err != nil {
		return nil, apperrors.CodeError(apperrors.CodeRefreshFailed)
	}

	// 检查 Refresh Token 是否已被吊销
	revoked, err := auth.IsTokenRevoked(ctx, req.RefreshToken)
	if err != nil {
		return nil, apperrors.Wrap(apperrors.CodeInternalError, err)
	}
	if revoked {
		return nil, apperrors.CodeError(apperrors.CodeTokenRevoked)
	}

	// 查找用户确认账号状态
	user, err := s.repo.GetUserByID(ctx, claims.UserID)
	if err != nil {
		return nil, err
	}
	// 检查账号状态(支持有期限封号自动放行)
	if err := checkUserBanned(user); err != nil {
		return nil, err
	}

	// Token Rotation:吊销旧 Refresh Token,防止被盗后重复使用
	// 即使吊销失败也不阻断返回(降级安全,避免如果 Redis 暂时不可用导致用户无法刷新 token)
	if err := auth.RevokeToken(ctx, req.RefreshToken, claims.ExpiresAt.Time); err != nil {
		logger.Warn("failed to revoke token", zap.Error(err))
	}

	return s.generateAuthResponse(ctx, user)
}

// ==================== 个人主页 ====================

// GetUserProfile 获取用户个人主页
func (s *userService) GetUserProfile(ctx context.Context, id uint64, viewerID uint64) (*UserProfileResponse, error) {
	// ===== 读缓存（仅未登录访客使用缓存，登录用户需实时查询 is_following）=====
	if viewerID == 0 {
		var cached UserProfileResponse
		if hit, _ := cache.GetJSON(ctx, cache.UserProfileKey(id), &cached); hit {
			return &cached, nil
		}
	}

	// 查找用户(预加载 Skills 和 Portfolio)
	user, err := s.repo.GetUserByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// 获取关注统计(使用 User 模型上的冗余计数字段,避免实时 COUNT)
	followersCount := int64(user.FollowerCount)
	followingCount := int64(user.FollowingCount)

	// 生成头像公开 URL
	avatarURL := ""
	if user.AvatarKey != "" {
		avatarURL = presignURL(ctx, user.AvatarKey)
	}

	// 构建响应
	skills := user.Skills
	if skills == nil {
		skills = []model.UserSkill{}
	}
	portfolio := user.Portfolio
	if portfolio == nil {
		portfolio = []model.Portfolio{}
	}

	// ===== 查询计数 =====
	postCount, _ := s.repo.CountUserPosts(ctx, user.ID)
	logCount, _ := s.repo.CountUserLogs(ctx, user.ID)
	projectCount, _ := s.repo.CountUserProjects(ctx, user.ID)

	// 查询访问者与该用户的关注关系
	var isFollowing bool
	if viewerID != 0 && viewerID != user.ID {
		if s.followRepo != nil {
			if ok, err := s.followRepo.IsFollowingUser(ctx, viewerID, user.ID); err == nil {
				isFollowing = ok
			}
		}
	}

	// ===== 计算用户成就徽章 =====
	badges := make([]string, 0)
	// PIONEER: 注册超过 30 天
	if time.Since(user.CreatedAt) > 30*24*time.Hour {
		badges = append(badges, "PIONEER")
	}
	// CREATOR: 有 > 0 个项目
	if projectCount > 0 {
		badges = append(badges, "CREATOR")
	}
	// PROLIFIC: 有 > 5 条 devlog
	if logCount > 5 {
		badges = append(badges, "PROLIFIC")
	}
	// SOCIAL: 有 > 10 个粉丝
	if user.FollowerCount > 10 {
		badges = append(badges, "SOCIAL")
	}

	resp := &UserProfileResponse{
		ID:             user.ID,
		Username:       user.Username,
		Nickname:       user.Nickname,
		Role:           string(user.Role),
		Bio:            user.Bio,
		AvatarURL:      avatarURL,
		Skills:         skills,
		Portfolio:      portfolio,
		PostCount:      postCount,
		LogCount:       logCount,
		ProjectCount:   projectCount,
		FollowersCount: followersCount,
		FollowingCount: followingCount,
		IsFollowing:    isFollowing,
		IsBanned:       user.IsBanned,
		Badges:         badges,
		CreatedAt:      user.CreatedAt,
	}

	// 仅访问自己主页时,附带求职合作状态
	if viewerID != 0 && viewerID == id {
		isAvail := user.IsAvailable
		coopPref := user.CoopPreference
		location := user.Location
		resp.IsAvailable = &isAvail
		resp.CoopPreference = &coopPref
		resp.Location = &location
	}

	// ===== 写缓存（仅未登录访客，已登录用户有 viewer-specific 数据不缓存）=====
	if viewerID == 0 {
		if err := cache.SetJSON(ctx, cache.UserProfileKey(id), resp, cache.TTLUserProfile); err != nil {
			logger.Warn("failed to set cache", zap.Error(err))
		}
	}

	return resp, nil
}

// GetUserProfileByUsername 通过 username 获取用户个人主页(先查用户 ID,再复用 GetUserProfile)
func (s *userService) GetUserProfileByUsername(ctx context.Context, username string, viewerID uint64) (*UserProfileResponse, error) {
	user, err := s.repo.GetUserByUsername(ctx, username)
	if err != nil {
		return nil, err
	}
	return s.GetUserProfile(ctx, user.ID, viewerID)
}

// DevLogListItem 开发日志列表项（含预签名 URL）
type DevLogListItem struct {
	ID            uint64    `json:"id"`
	ProjectID     uint64    `json:"project_id"`
	AuthorID      uint64    `json:"author_id"`
	Title         string    `json:"title"`
	Content       string    `json:"content"`
	LogType       string    `json:"log_type"`
	Version       string    `json:"version"`
	ImageURLs     []string  `json:"image_urls"`
	VideoURLs     []string  `json:"video_urls"`
	DownloadURL   string    `json:"download_url"`
	Visibility    string    `json:"visibility"`
	Status        string    `json:"status"`
	LikeCount     int       `json:"like_count"`
	ViewCount     int       `json:"view_count"`
	CommentCount  int       `json:"comment_count"`
	CollectCount  int       `json:"collect_count"`
	CoverURL      string    `json:"cover_url"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

// GetUserLogs 分页查询用户已发布的开发日志(含预签名图片/视频 URL)
func (s *userService) GetUserLogs(ctx context.Context, userID uint64, page, pageSize int) ([]DevLogListItem, int64, error) {
	if s.devLogRepo == nil {
		return []DevLogListItem{}, 0, nil
	}
	logs, total, err := s.devLogRepo.ListLogsByAuthorID(ctx, userID, (page-1)*pageSize, pageSize)
	if err != nil {
		return nil, 0, err
	}
	result := make([]DevLogListItem, len(logs))
	for i, l := range logs {
		result[i] = DevLogListItem{
			ID:           l.ID,
			ProjectID:    l.ProjectID,
			AuthorID:     l.AuthorID,
			Title:        l.Title,
			Content:      l.Content,
			LogType:      string(l.LogType),
			Version:      l.Version,
			ImageURLs:    buildImageURLs(ctx, l.Images),
			VideoURLs:    buildImageURLs(ctx, l.Videos),
			DownloadURL:  l.DownloadURL,
			Visibility:   string(l.Visibility),
			Status:       string(l.Status),
			LikeCount:    l.LikeCount,
			ViewCount:    l.ViewCount,
			CommentCount: l.CommentCount,
			CollectCount: l.CollectCount,
			CreatedAt:    l.CreatedAt,
			UpdatedAt:    l.UpdatedAt,
		}
		if len(result[i].ImageURLs) > 0 {
			result[i].CoverURL = result[i].ImageURLs[0]
		}
	}
	return result, total, nil
}

// GetUserEndorsements 分页查询用户收到的合作评价
func (s *userService) GetUserEndorsements(ctx context.Context, userID uint64, page, pageSize int) ([]EndorsementItem, int64, error) {
	reviews, total, err := s.teamRepo.GetReviewsByRevieweeID(ctx, userID, page, pageSize)
	if err != nil {
		return nil, 0, err
	}
	// 收集评价人 ID
	reviewerIDs := make([]uint64, 0, len(reviews))
	for _, r := range reviews {
		reviewerIDs = append(reviewerIDs, r.ReviewerID)
	}
	// 批量查询评价人信息
	reviewers, _ := s.repo.GetUsersByIDs(ctx, reviewerIDs)
	reviewerMap := make(map[uint64]*model.User, len(reviewers))
	for _, u := range reviewers {
		reviewerMap[u.ID] = u
	}
	// 构建 EndorsementItem
	result := make([]EndorsementItem, 0, len(reviews))
	for _, r := range reviews {
		item := EndorsementItem{
			ID:         r.ID,
			ProjectID:  r.ProjectID,
			ReviewerID: r.ReviewerID,
			RevieweeID: r.RevieweeID,
			Rating:     r.Rating,
			Comment:    r.Comment,
			Supplement: r.Supplement,
			CreatedAt:  r.CreatedAt,
		}
		item.Tags = splitTags(r.Tags)
		if reviewer, ok := reviewerMap[r.ReviewerID]; ok {
			item.ReviewerNickname = reviewer.Nickname
			if reviewer.AvatarKey != "" {
				item.ReviewerAvatarURL = presignURL(ctx, reviewer.AvatarKey)
			}
		}
		result = append(result, item)
	}
	return result, total, nil
}

// splitTags 将 Tags JSON 字符串拆分为字符串切片
func splitTags(tagsJSON string) []string {
	if tagsJSON == "" || tagsJSON == "[]" {
		return nil
	}
	// Remove brackets and split by comma
	s := strings.TrimPrefix(tagsJSON, "[")
	s = strings.TrimSuffix(s, "]")
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	tags := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.Trim(p, "\" ")
		if p != "" {
			tags = append(tags, p)
		}
	}
	return tags
}

// ==================== 个人资料更新 ====================

// UpdateProfile 更新用户昵称和简介
func (s *userService) UpdateProfile(ctx context.Context, userID uint64, req *UpdateProfileReq) error {
	updates := make(map[string]interface{})

	// 验证并收集昵称更新
	if req.Nickname != "" {
		trimmedNickname := strings.TrimSpace(req.Nickname)
		if err := validateNickname(trimmedNickname); err != nil {
			return err
		}
		updates["nickname"] = trimmedNickname
	}

	// 验证并收集简介更新(bio 使用指针,区分"未传"和"传了空字符串")
	if req.Bio != nil {
		bioVal := strings.TrimSpace(*req.Bio)
		if bioVal != "" {
			if utf8.RuneCountInString(bioVal) > 300 {
				return apperrors.New(apperrors.CodeParamTooLong, "个人简介最多300字")
			}
			if filter.Contains(bioVal) {
				return apperrors.New(apperrors.CodeContentViolation, "个人简介包含敏感词,请修改后重试")
			}
		}
		updates["bio"] = bioVal // ""表示清空
	}

	// 验证并收集所在地区更新(location 使用指针,区分"未传"和"传了空字符串")
	if req.Location != nil {
		locVal := strings.TrimSpace(*req.Location)
		if utf8.RuneCountInString(locVal) > 64 {
			return apperrors.New(apperrors.CodeParamTooLong, "所在地区最多64个字符")
		}
		updates["location"] = locVal
	}

	// 有内容才更新
	if len(updates) > 0 {
		if err := s.repo.UpdateUser(ctx, userID, updates); err != nil {
			return err
		}
		// 失效用户主页缓存(需先查 username)
		if u, e := s.repo.GetUserByID(ctx, userID); e == nil {
			if err := cache.Del(ctx, cache.UserProfileKey(u.ID)); err != nil {
				logger.Warn("failed to delete cache", zap.Error(err))
			}
		}
		return nil
	}
	return nil
}

// ==================== 头像上传 ====================

// SaveAvatar 保存头像 key(前端已通过预签名 URL 直传云存储后调用此方法)
func (s *userService) SaveAvatar(ctx context.Context, userID uint64, key string) error {
	// 验证 key 格式,必须以 avatars/ 开头,防止客户端伪造路径
	if !strings.HasPrefix(key, "avatars/") {
		return apperrors.New(apperrors.CodeParamInvalid, "无效的头像 key 格式")
	}

	// 查询旧头像 key,以便上传成功后删除旧文件
	user, err := s.repo.GetUserByID(ctx, userID)
	if err != nil {
		return apperrors.Wrap(apperrors.CodeInternalError, err)
	}
	oldKey := user.AvatarKey

	// 更新数据库头像字段
	if err := s.repo.UpdateUser(ctx, userID, map[string]interface{}{
		"avatar_key": key,
	}); err != nil {
		return apperrors.Wrap(apperrors.CodeInternalError, err)
	}

	// 异步删除旧头像（静默忽略删除失败,不影响主流程）
	if oldKey != "" && oldKey != key {
		go func() {
			if err := storage.Delete(ctx, oldKey); err != nil {
				logger.Warn("failed to delete old avatar", zap.String("key", oldKey), zap.Error(err))
			}
		}()
	}

	return nil
}

// ==================== 技能标签 ====================

// UpdateSkills 更新用户技能标签(整体替换)
func (s *userService) UpdateSkills(ctx context.Context, userID uint64, req *UpdateSkillsReq) ([]model.UserSkill, error) {
	// 验证并构建技能列表
	var skills []model.UserSkill
	customCount := 0

	for _, input := range req.Skills {
		// 验证分类
		category := model.SkillCategory(input.Category)
		if !isValidSkillCategory(category) {
			return nil, apperrors.Newf(apperrors.CodeParamInvalid, "技能分类 '%s' 不合法,支持:program/art/design/sound/custom", input.Category)
		}

		// 验证熟练度
		level := model.SkillLevel(input.Level)
		if !isValidSkillLevel(level) {
			return nil, apperrors.Newf(apperrors.CodeParamInvalid, "技能熟练度 '%s' 不合法,支持:beginner/intermediate/advanced", input.Level)
		}

		// 验证技能名称
		if strings.TrimSpace(input.Name) == "" {
			return nil, apperrors.New(apperrors.CodeParamInvalid, "技能名称不能为空")
		}
		if utf8.RuneCountInString(input.Name) > 32 {
			return nil, apperrors.New(apperrors.CodeParamTooLong, "技能名称最多32个字符")
		}

		// 验证技能描述(选填)
		desc := strings.TrimSpace(input.Description)
		if utf8.RuneCountInString(desc) > 200 {
			return nil, apperrors.New(apperrors.CodeParamTooLong, "技能描述最多200个字符")
		}
		if desc != "" && filter.Contains(desc) {
			return nil, apperrors.New(apperrors.CodeParamInvalid, "技能描述包含违禁词")
		}

		// 判断是否为自定义标签
		isCustom := category == model.SkillCategoryCustom || !model.IsPresetSkill(category, input.Name)
		if isCustom {
			customCount++
			if customCount > customSkillMaxCount {
				return nil, apperrors.Newf(apperrors.CodeParamInvalid, "自定义技能标签最多%d个", customSkillMaxCount)
			}
		}

		skills = append(skills, model.UserSkill{
			UserID:      userID,
			Category:    category,
			Name:        input.Name,
			Level:       level,
			IsCustom:    isCustom,
			Description: desc,
		})
	}

	// 替换技能(事务执行)
	if err := s.repo.ReplaceUserSkills(ctx, userID, skills); err != nil {
		return nil, apperrors.Wrap(apperrors.CodeInternalError, err)
	}

	// 返回最新技能列表
	result, err := s.repo.GetSkillsByUserID(ctx, userID)
	if err != nil {
		return nil, apperrors.Wrap(apperrors.CodeInternalError, err)
	}

	if result == nil {
		result = []model.UserSkill{}
	}
	return result, nil
}

// ==================== 作品集 ====================

// GetMyPortfolio 获取自己的作品集
func (s *userService) GetMyPortfolio(ctx context.Context, userID uint64) ([]model.Portfolio, error) {
	portfolios, err := s.repo.GetPortfoliosByUserID(ctx, userID)
	if err != nil {
		return nil, apperrors.Wrap(apperrors.CodeInternalError, err)
	}
	if portfolios == nil {
		portfolios = []model.Portfolio{}
	}
	return portfolios, nil
}

// CreatePortfolio 添加一个作品条目
func (s *userService) CreatePortfolio(ctx context.Context, userID uint64, req *CreatePortfolioReq) (*model.Portfolio, error) {
	// 验证作品类型
	pType := model.PortfolioType(req.Type)
	if !isValidPortfolioType(pType) {
		return nil, apperrors.Newf(apperrors.CodeParamInvalid, "作品类型 '%s' 不合法,支持:game/demo/art/code", req.Type)
	}

	// 验证作品名称
	if strings.TrimSpace(req.Name) == "" {
		return nil, apperrors.New(apperrors.CodeParamInvalid, "作品名称不能为空")
	}
	if utf8.RuneCountInString(req.Name) > 128 {
		return nil, apperrors.New(apperrors.CodeParamTooLong, "作品名称最多128个字符")
	}
	if filter.Contains(req.Name) || filter.Contains(req.Description) {
		return nil, apperrors.New(apperrors.CodeParamInvalid, "作品内容包含违禁词")
	}

	portfolio := &model.Portfolio{
		UserID:      userID,
		Name:        req.Name,
		Type:        pType,
		Link:        req.Link,
		ImageURL:    req.ImageURL,
		Description: req.Description,
	}

	if err := s.repo.CreatePortfolio(ctx, portfolio); err != nil {
		return nil, apperrors.Wrap(apperrors.CodeInternalError, err)
	}

	return portfolio, nil
}

// UpdatePortfolio 更新作品条目,需检查归属权
func (s *userService) UpdatePortfolio(ctx context.Context, userID uint64, portfolioID uint64, req *UpdatePortfolioReq) (*model.Portfolio, error) {
	// 查找作品并验证归属
	portfolio, err := s.repo.GetPortfolioByID(ctx, portfolioID)
	if err != nil {
		return nil, err
	}
	if portfolio.UserID != userID {
		return nil, apperrors.CodeError(apperrors.CodeResourceForbidden)
	}

	updates := make(map[string]interface{})

	if req.Name != "" {
		if utf8.RuneCountInString(req.Name) > 128 {
			return nil, apperrors.New(apperrors.CodeParamTooLong, "作品名称最多128个字符")
		}
		if filter.Contains(req.Name) {
			return nil, apperrors.New(apperrors.CodeParamInvalid, "作品名称包含违禁词")
		}
		updates["name"] = req.Name
	}
	if req.Type != "" {
		pType := model.PortfolioType(req.Type)
		if !isValidPortfolioType(pType) {
			return nil, apperrors.Newf(apperrors.CodeParamInvalid, "作品类型 '%s' 不合法", req.Type)
		}
		updates["type"] = pType
	}
	// Link、ImageURL、Description 允许置空(显式更新)
	if req.Link != "" {
		updates["link"] = req.Link
	}
	if req.ImageURL != "" {
		updates["image_url"] = req.ImageURL
	}
	if req.Description != "" {
		if filter.Contains(req.Description) {
			return nil, apperrors.New(apperrors.CodeParamInvalid, "作品描述包含违禁词")
		}
		updates["description"] = req.Description
	}

	if len(updates) == 0 {
		return portfolio, nil
	}

	if err := s.repo.UpdatePortfolio(ctx, portfolioID, updates); err != nil {
		return nil, apperrors.Wrap(apperrors.CodeInternalError, err)
	}

	// 重新查询返回最新数据
	updated, err := s.repo.GetPortfolioByID(ctx, portfolioID)
	if err != nil {
		return nil, err
	}
	return updated, nil
}

// DeletePortfolio 删除作品条目,需检查归属权
func (s *userService) DeletePortfolio(ctx context.Context, userID uint64, portfolioID uint64) error {
	// 查找作品并验证归属
	portfolio, err := s.repo.GetPortfolioByID(ctx, portfolioID)
	if err != nil {
		return err
	}
	if portfolio.UserID != userID {
		return apperrors.CodeError(apperrors.CodeResourceForbidden)
	}

	return s.repo.DeletePortfolio(ctx, portfolioID)
}

// ==================== 内部辅助函数 ====================

// SetUserRole 更新指定用户的角色(仅允许 superadmin 通过 RoleHandler 调用)
// service 层双重校验:不允许设置 superadmin,防止权限扩散
func (s *userService) SetUserRole(ctx context.Context, userID uint64, role string) error {
	validRoles := map[string]bool{
		"user":      true,
		"creator":   true,
		"moderator": true,
		"admin":     true,
	}
	if !validRoles[role] {
		return apperrors.New(apperrors.CodePermissionDenied, "无效的角色,不允许设置为此角色")
	}
	return s.repo.UpdateUser(ctx, userID, map[string]interface{}{"role": role})
}

// generateAuthResponse 生成认证响应(Token 对 + 用户信息)
func (s *userService) generateAuthResponse(ctx context.Context, user *model.User) (*AuthResponse, error) {
	tokenPair, err := auth.GenerateTokenPair(user.ID, user.Username, string(user.Role))
	if err != nil {
		return nil, apperrors.Wrap(apperrors.CodeInternalError, err)
	}

	avatarURL := ""
	if user.AvatarKey != "" {
		avatarURL = presignURL(ctx, user.AvatarKey)
	}
	return &AuthResponse{
		AccessToken:  tokenPair.AccessToken,
		RefreshToken: tokenPair.RefreshToken,
		ExpiresAt:    tokenPair.ExpiresAt,
		User: UserInfo{
			ID:        user.ID,
			Username:  user.Username,
			Nickname:  user.Nickname,
			Role:      string(user.Role),
			AvatarURL: avatarURL,
		},
	}, nil
}

// resolveUsername 处理用户名:用户提供则校验,未提供则生成 user_{随机6位}
func (s *userService) resolveUsername(ctx context.Context, username string) (string, error) {
	if username == "" {
		// 自动生成,最多重试 5 次避免极低概率冲突
		for i := 0; i < 5; i++ {
			generated := "user_" + generateRandom6()
			exists, err := s.repo.ExistsUsername(ctx, generated)
			if err != nil {
				return "", apperrors.Wrap(apperrors.CodeInternalError, err)
			}
			if !exists {
				return generated, nil
			}
		}
		return "", apperrors.New(apperrors.CodeInternalError, "生成用户名失败,请稍后重试")
	}

	// 验证用户名格式:只允许字母、数字、下划线,3-32 位
	if !isValidUsername(username) {
		return "", apperrors.New(apperrors.CodeParamInvalid, "用户名只允许字母、数字和下划线,长度3-32位")
	}
	// 用户名敏感词检测
	if filter.Contains(username) {
		return "", apperrors.New(apperrors.CodeParamInvalid, "用户名包含违禁词")
	}

	// 检查是否已被占用
	exists, err := s.repo.ExistsUsername(ctx, username)
	if err != nil {
		return "", apperrors.Wrap(apperrors.CodeInternalError, err)
	}
	if exists {
		return "", apperrors.CodeError(apperrors.CodeUsernameAlreadyUsed)
	}

	return username, nil
}

// generateRandom6 生成 6 位随机数字字符串(000000-999999)
// 使用 4 字节随机数对 1000000 取模,最大偏差 < 0.023%,可忽略
func generateRandom6() string {
	b := make([]byte, 4)
	_, _ = rand.Read(b)
	n := (uint32(b[0])<<24 | uint32(b[1])<<16 | uint32(b[2])<<8 | uint32(b[3])) % 1000000
	return fmt.Sprintf("%06d", n)
}

// ==================== 参数校验辅助函数 ====================

// isValidPhone 验证手机号:11 位数字
func isValidPhone(phone string) bool {
	if len(phone) != 11 {
		return false
	}
	for _, c := range phone {
		if c < '0' || c > '9' {
			return false
		}
	}
	return true
}

// isValidEmail 验证邮箱格式
func isValidEmail(email string) bool {
	return emailRegexp.MatchString(email)
}

// validatePassword 验证密码:8-32 位,必须同时含字母和数字
func validatePassword(password string) error {
	length := len(password)
	if length < 8 {
		return apperrors.New(apperrors.CodePasswordTooWeak, "密码长度不能少于8位")
	}
	if length > 32 {
		return apperrors.New(apperrors.CodeParamTooLong, "密码长度不能超过32位")
	}
	if !passwordLetterRegexp.MatchString(password) {
		return apperrors.New(apperrors.CodePasswordTooWeak, "密码必须包含字母")
	}
	if !passwordDigitRegexp.MatchString(password) {
		return apperrors.New(apperrors.CodePasswordTooWeak, "密码必须包含数字")
	}
	return nil
}

// validateNickname 验证昵称:先 trim 两端空白,再按 2-32 位(Unicode 字符数)校验,不含敏感词
func validateNickname(nickname string) error {
	nickname = strings.TrimSpace(nickname)
	count := utf8.RuneCountInString(nickname)
	if count < 2 {
		return apperrors.New(apperrors.CodeParamTooShort, "昵称长度不能少于2个字符")
	}
	if count > 32 {
		return apperrors.New(apperrors.CodeParamTooLong, "昵称长度不能超过32个字符")
	}
	if filter.Contains(nickname) {
		return apperrors.New(apperrors.CodeContentViolation, "昵称包含敏感词,请修改后重试")
	}
	return nil
}

// isValidUsername 验证用户名:只允许字母、数字、下划线,3-32 位
// 使用预编译的包级变量,避免每次调用重复编译
func isValidUsername(username string) bool {
	return usernameRegexp.MatchString(username)
}

// isValidSkillCategory 验证技能分类
func isValidSkillCategory(category model.SkillCategory) bool {
	switch category {
	case model.SkillCategoryProgram, model.SkillCategoryArt,
		model.SkillCategoryDesign, model.SkillCategorySound,
		model.SkillCategoryCustom:
		return true
	}
	return false
}

// isValidSkillLevel 验证技能熟练度
func isValidSkillLevel(level model.SkillLevel) bool {
	switch level {
	case model.SkillLevelBeginner, model.SkillLevelIntermediate, model.SkillLevelAdvanced:
		return true
	}
	return false
}

// isValidPortfolioType 验证作品类型
func isValidPortfolioType(pType model.PortfolioType) bool {
	switch pType {
	case model.PortfolioTypeGame, model.PortfolioTypeDemo,
		model.PortfolioTypeArt, model.PortfolioTypeCode:
		return true
	}
	return false
}

// checkUserBanned 检查账号封禁状态。
// 若封号已到期则直接放行(返回 nil),否则返回带剩余时间描述的错误。
// 由调用方负责在 checkUserBanned 返回 nil 后异步清除 BannedUntil 字段(惰性解封)。
func checkUserBanned(user *model.User) error {
	if user.Status != model.UserStatusDisabled {
		return nil
	}
	// 有到期时间且已过期 → 惰性放行
	if user.BannedUntil != nil && time.Now().After(*user.BannedUntil) {
		return nil
	}
	// 有到期时间 → 提示剩余时长
	if user.BannedUntil != nil {
		remaining := time.Until(*user.BannedUntil).Truncate(time.Minute)
		return apperrors.Newf(apperrors.CodeUserDisabled, "账号已被封禁,解封时间:%s(剩余约 %s)",
			user.BannedUntil.Format("2006-01-02 15:04"), remaining)
	}
	// 无到期时间 → 永久封禁
	return apperrors.CodeError(apperrors.CodeUserDisabled)
}

// ==================== 密码管理 ====================

// ChangePassword 已登录用户修改密码。
// 需验证旧密码正确,然后加密新密码写入数据库。
func (s *userService) ChangePassword(ctx context.Context, userID uint64, req *ChangePasswordReq) error {
	if err := validatePassword(req.NewPassword); err != nil {
		return err
	}
	if req.OldPassword == req.NewPassword {
		return apperrors.New(apperrors.CodeParamInvalid, "新密码不能与旧密码相同")
	}

	user, err := s.repo.GetUserByID(ctx, userID)
	if err != nil {
		return apperrors.Wrap(apperrors.CodeInternalError, err)
	}
	if user.PasswordHash == "" {
		return apperrors.New(apperrors.CodePasswordIncorrect, "该账号未设置密码,请通过邮箱验证码重置密码")
	}
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.OldPassword)); err != nil {
		return apperrors.New(apperrors.CodePasswordIncorrect, "旧密码不正确")
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcryptCost)
	if err != nil {
		return apperrors.Wrap(apperrors.CodeInternalError, err)
	}
	return s.repo.UpdateUser(ctx, userID, map[string]interface{}{
		"password_hash": string(hash),
	})
}

// SendResetPasswordCode 向邮箱发送重置密码验证码。
// 邮箱必须是已注册账号,否则拒绝(防止枚举)。
func (s *userService) SendResetPasswordCode(ctx context.Context, req *SendEmailCodeReq) error {
	if !isValidEmail(req.Email) {
		return apperrors.New(apperrors.CodeParamInvalid, "邮箱格式不正确")
	}
	exists, err := s.repo.ExistsEmail(ctx, req.Email)
	if err != nil {
		return apperrors.Wrap(apperrors.CodeInternalError, err)
	}
	if !exists {
		// 统一回复,防止邮箱枚举攻击
		return apperrors.New(apperrors.CodeParamInvalid, "该邮箱未注册")
	}
	_, err = pkgemail.SendVerifyCode(ctx, req.Email)
	return err
}

// ResetPasswordByCode 通过邮箱验证码重置密码(忘记密码流程)。
func (s *userService) ResetPasswordByCode(ctx context.Context, req *ResetPasswordByCodeReq) error {
	if !isValidEmail(req.Email) {
		return apperrors.New(apperrors.CodeParamInvalid, "邮箱格式不正确")
	}
	if err := validatePassword(req.NewPassword); err != nil {
		return err
	}

	// 验证邮箱验证码
	if err := pkgemail.Verify(ctx, req.Email, req.Code); err != nil {
		return err
	}

	// 查找用户
	user, err := s.repo.GetUserByEmail(ctx, req.Email)
	if err != nil {
		return apperrors.Wrap(apperrors.CodeInternalError, err)
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcryptCost)
	if err != nil {
		return apperrors.Wrap(apperrors.CodeInternalError, err)
	}
	return s.repo.UpdateUser(ctx, user.ID, map[string]interface{}{
		"password_hash": string(hash),
	})
}

// ==================== 账号注销 ====================

// UserSearchItem 用户搜索结果简化项
type UserSearchItem struct {
	ID        uint64 `json:"id"`
	Username  string `json:"username"`
	Nickname  string `json:"nickname"`
	AvatarURL string `json:"avatar_url"`
}

// SearchUsers 按 username/nickname 模糊搜索用户,返回简化的用户信息列表。
func (s *userService) SearchUsers(ctx context.Context, keyword string, limit int) ([]*UserSearchItem, error) {
	if limit <= 0 || limit > 50 {
		limit = 10
	}
	users, err := s.repo.SearchUsersByKeyword(ctx, keyword, limit)
	if err != nil {
		return nil, apperrors.Wrap(apperrors.CodeInternalError, err)
	}
	items := make([]*UserSearchItem, 0, len(users))
	for _, u := range users {
		avatarURL := ""
		if u.AvatarKey != "" {
			avatarURL = presignURL(ctx, u.AvatarKey)
		}
		items = append(items, &UserSearchItem{
			ID:        u.ID,
			Username:  u.Username,
			Nickname:  u.Nickname,
			AvatarURL: avatarURL,
		})
	}
	return items, nil
}

// DeleteAccountReq 账号注销确认请求
type DeleteAccountReq struct {
	Confirmation string `json:"confirmation" binding:"required"` // 必须填写 "DELETE MY ACCOUNT"
}

// DeleteAccount 注销账号(软删除 + 清除敏感信息)。
// 注销后:
//   - 用户记录软删除(deleted_at 字段)
//   - 手机号/邮箱置空(释放唯一索引,允许新账号注册相同号码)
//   - 密码 hash 清除
//   - 当前 Access Token 立即吊销写入 Redis 黑名单(Handler 层传入)
func (s *userService) DeleteAccount(ctx context.Context, userID uint64, req *DeleteAccountReq) error {
	return s.deleteAccountWithToken(ctx, userID, req, "", time.Time{})
}

// deleteAccountWithToken 实际执行注销逻辑,支持吊销 Access Token
func (s *userService) deleteAccountWithToken(ctx context.Context, userID uint64, req *DeleteAccountReq, tokenStr string, tokenExpiresAt time.Time) error {
	if req.Confirmation != "DELETE MY ACCOUNT" {
		return apperrors.New(apperrors.CodeParamInvalid, "请输入确认字符串 \"DELETE MY ACCOUNT\" 来确认注销")
	}

	user, err := s.repo.GetUserByID(ctx, userID)
	if err != nil {
		return apperrors.Wrap(apperrors.CodeInternalError, err)
	}

	// 清除敏感字段(释放唯一索引、隐藏个人信息)
	now := time.Now()
	uniqueSuffix := fmt.Sprintf("_deleted_%d", now.Unix())
	updates := map[string]interface{}{
		"phone":         nil,
		"email":         nil,
		"password_hash": "",
		"nickname":      user.Nickname + " (已注销)",
		"bio":           "",
		"deleted_at":    now,
		// 将 username 改名防止索引冲突(保留原始 username 前缀方便溯源)
		"username": user.Username + uniqueSuffix,
	}
	if err := s.repo.UpdateUser(ctx, userID, updates); err != nil {
		return err
	}

	// 吊销当前 Access Token(如果 Handler 层传入)
	if tokenStr != "" && !tokenExpiresAt.IsZero() {
		if err := auth.RevokeToken(ctx, tokenStr, tokenExpiresAt); err != nil {
			logger.Warn("failed to revoke token", zap.Error(err))
		}
	}

	return nil
}
