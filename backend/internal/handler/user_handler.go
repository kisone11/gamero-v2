// Package handler 提供 HTTP 请求处理层。
// 本文件包含用户系统所有 API 的 Gin Handler 实现。
// Handler 负责：参数绑定与校验、调用 Service 层、统一响应输出。
package handler

import (
	"strconv"
	"strings"
	"time"

	"github.com/gamero/gamero/internal/service"
	"github.com/gamero/gamero/pkg/auth"
	apperrors "github.com/gamero/gamero/pkg/errors"
	"github.com/gamero/gamero/pkg/logger"
	"github.com/gamero/gamero/pkg/middleware"
	"github.com/gamero/gamero/pkg/pagination"
	"github.com/gamero/gamero/pkg/response"
	"github.com/gamero/gamero/pkg/storage"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// UserHandler 用户相关 API 的 Handler 集合
type UserHandler struct {
	svc service.UserService
}

// NewUserHandler 创建 UserHandler 实例
func NewUserHandler(svc service.UserService) *UserHandler {
	return &UserHandler{svc: svc}
}

// ==================== 验证码 ====================

// SendSMSCode 发送手机号验证码
// POST /api/v1/auth/send-sms-code
// func (h *UserHandler) SendSMSCode(c *gin.Context) {
// 	var req service.SendSMSCodeReq
// 	if err := c.ShouldBindJSON(&req); err != nil {
// 		response.FailBadRequest(c, "请求参数错误："+err.Error())
// 		return
// 	}

// 	if err := h.svc.SendSMSCode(c.Request.Context(), &req); err != nil {
// 		response.Fail(c, err)
// 		return
// 	}

// 	response.SuccessMsg(c, "验证码已发送", nil)
// }

// SendEmailCode 发送邮箱验证码
// POST /api/v1/auth/send-email-code
func (h *UserHandler) SendEmailCode(c *gin.Context) {
	var req service.SendEmailCodeReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailBadRequest(c, "请求参数错误："+err.Error())
		return
	}

	_, err := h.svc.SendEmailCode(c.Request.Context(), &req)
	if err != nil {
		response.Fail(c, err)
		return
	}

	response.SuccessMsg(c, "验证码已发送，请查看邮箱", nil)
}

// ==================== 注册 ====================

// RegisterByPhone 手机号注册
// POST /api/v1/auth/register/phone
// func (h *UserHandler) RegisterByPhone(c *gin.Context) {
// 	var req service.RegisterByPhoneReq
// 	if err := c.ShouldBindJSON(&req); err != nil {
// 		response.FailBadRequest(c, "请求参数错误："+err.Error())
// 		return
// 	}

// 	result, err := h.svc.RegisterByPhone(c.Request.Context(), &req)
// 	if err != nil {
// 		response.Fail(c, err)
// 		return
// 	}

// 	response.SuccessMsg(c, "注册成功", result)
// }

// RegisterByEmail 邮箱注册
// POST /api/v1/auth/register/email
func (h *UserHandler) RegisterByEmail(c *gin.Context) {
	var req service.RegisterByEmailReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailBadRequest(c, "请求参数错误："+err.Error())
		return
	}

	result, err := h.svc.RegisterByEmail(c.Request.Context(), &req)
	if err != nil {
		response.Fail(c, err)
		return
	}

	response.SuccessMsg(c, "注册成功", result)
}

// ==================== 登录 ====================

// LoginByPhone 手机号验证码登录
// POST /api/v1/auth/login/phone
// func (h *UserHandler) LoginByPhone(c *gin.Context) {
// 	var req service.LoginByPhoneReq
// 	if err := c.ShouldBindJSON(&req); err != nil {
// 		response.FailBadRequest(c, "请求参数错误："+err.Error())
// 		return
// 	}

// 	result, err := h.svc.LoginByPhone(c.Request.Context(), &req)
// 	if err != nil {
// 		response.Fail(c, err)
// 		return
// 	}

// 	response.SuccessMsg(c, "登录成功", result)
// }

// LoginByEmail 邮箱密码登录
// POST /api/v1/auth/login/email
func (h *UserHandler) LoginByEmail(c *gin.Context) {
	var req service.LoginByEmailReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailBadRequest(c, "请求参数错误："+err.Error())
		return
	}

	result, err := h.svc.LoginByEmail(c.Request.Context(), &req)
	if err != nil {
		response.Fail(c, err)
		return
	}

	response.SuccessMsg(c, "登录成功", result)
}

// LoginByWechat 微信登录（预留接口，功能未开放）
// POST /api/v1/auth/login/wechat
// 预留数据结构：请求体 { "code": "微信授权code", "state": "防CSRF状态码" }
// 微信登录流程：前端获取 code -> 后端换取 access_token -> 获取 openid/unionid -> 绑定/创建账号
func (h *UserHandler) LoginByWechat(c *gin.Context) {
	response.FailMsg(c, apperrors.CodeServiceUnavail, "微信登录功能暂未开放，敬请期待")
}

// LoginByGithub GitHub 登录（预留接口，功能未开放）
// POST /api/v1/auth/login/github
// 预留数据结构：请求体 { "code": "GitHub OAuth code", "state": "防CSRF状态码" }
// GitHub OAuth 流程：前端获取 code -> 后端换取 access_token -> 获取用户信息 -> 绑定/创建账号
func (h *UserHandler) LoginByGithub(c *gin.Context) {
	response.FailMsg(c, apperrors.CodeServiceUnavail, "GitHub 登录功能暂未开放，敬请期待")
}

// ==================== Token 管理 ====================

// RefreshToken 用 Refresh Token 换取新的 Token 对
// POST /api/v1/auth/refresh
func (h *UserHandler) RefreshToken(c *gin.Context) {
	var req service.RefreshTokenReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailBadRequest(c, "请求参数错误："+err.Error())
		return
	}

	result, err := h.svc.RefreshToken(c.Request.Context(), &req)
	if err != nil {
		response.Fail(c, err)
		return
	}

	response.Success(c, result)
}

// Logout 登出，将当前 Access Token 加入黑名单
// POST /api/v1/auth/logout（需 JWT 认证）
func (h *UserHandler) Logout(c *gin.Context) {
	// 从 Authorization 头获取 Token 字符串
	authHeader := c.GetHeader("Authorization")
	parts := strings.SplitN(authHeader, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") || parts[1] == "" {
		response.FailCode(c, apperrors.CodeTokenMissing)
		return
	}
	tokenStr := parts[1]

	// 从 Context 中获取已解析的 JWT Claims（由 middleware.JWTAuth 注入）
	claimsVal, exists := c.Get(middleware.ClaimsKey)
	if !exists {
		response.FailCode(c, apperrors.CodeTokenInvalid)
		return
	}

	// 类型断言为 pkg/auth.Claims
	claims, ok := claimsVal.(*auth.Claims)
	if !ok {
		response.FailCode(c, apperrors.CodeTokenInvalid)
		return
	}

	// 获取 Token 过期时间（用于设置 Redis 黑名单 TTL）
	var expiresAt time.Time
	if claims.ExpiresAt != nil {
		expiresAt = claims.ExpiresAt.Time
	} else {
		// ExpiresAt 不应为 nil（在 parseToken 时已验证），兜底给个合理过期时间
		expiresAt = time.Now().Add(2 * time.Hour)
	}

	if err := h.svc.Logout(c.Request.Context(), tokenStr, expiresAt); err != nil {
		response.Fail(c, err)
		return
	}

	response.SuccessMsg(c, "已成功登出", nil)
}

// ==================== 密码管理 ====================

// ChangePassword 修改密码（需已登录，提供旧密码）
// PUT /api/v1/users/me/password
func (h *UserHandler) ChangePassword(c *gin.Context) {
	userID, ok := requireLogin(c)
	if !ok {
		return
	}
	var req service.ChangePasswordReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailBadRequest(c, err.Error())
		return
	}
	if err := h.svc.ChangePassword(c.Request.Context(), userID, &req); err != nil {
		response.Fail(c, err)
		return
	}
	response.SuccessMsg(c, "密码修改成功", nil)
}

// SendResetPasswordCode 发送忘记密码验证码
// POST /api/v1/auth/forgot-password
func (h *UserHandler) SendResetPasswordCode(c *gin.Context) {
	var req service.SendEmailCodeReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailBadRequest(c, err.Error())
		return
	}
	if err := h.svc.SendResetPasswordCode(c.Request.Context(), &req); err != nil {
		response.Fail(c, err)
		return
	}
	response.SuccessMsg(c, "验证码已发送，请查收邮件", nil)
}

// ResetPasswordByCode 通过验证码重置密码
// POST /api/v1/auth/reset-password
func (h *UserHandler) ResetPasswordByCode(c *gin.Context) {
	var req service.ResetPasswordByCodeReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailBadRequest(c, err.Error())
		return
	}
	if err := h.svc.ResetPasswordByCode(c.Request.Context(), &req); err != nil {
		response.Fail(c, err)
		return
	}
	response.SuccessMsg(c, "密码重置成功，请用新密码登录", nil)
}

// GetUserProfile 获取用户个人主页（通过用户名，公开接口，可选 JWT）
// GET /api/v1/users/:id
func (h *UserHandler) GetUserProfile(c *gin.Context) {
	raw := c.Param("id")
	viewerID, _ := middleware.GetUserID(c) // 未登录时为 0

	var profile interface{}
	var err error

	if uid, parseErr := strconv.ParseUint(raw, 10, 64); parseErr == nil {
		// 参数是数字 ID
		profile, err = h.svc.GetUserProfile(c.Request.Context(), uid, viewerID)
	} else {
		// 参数是 username 字符串
		profile, err = h.svc.GetUserProfileByUsername(c.Request.Context(), raw, viewerID)
	}

	if err != nil {
		response.Fail(c, err)
		return
	}

	response.Success(c, profile)
}

// GetMyProfile 获取当前登录用户的完整资料（需 JWT）
// GET /api/v1/users/me
func (h *UserHandler) GetMyProfile(c *gin.Context) {
	myID, ok := requireLogin(c)
	if !ok {
		return
	}

	profile, err := h.svc.GetUserProfile(c.Request.Context(), myID, myID)
	if err != nil {
		response.Fail(c, err)
		return
	}

	response.Success(c, profile)
}

// GetUserLogs 分页查询用户已发布的开发日志
// GET /api/v1/users/:id/logs
func (h *UserHandler) GetUserLogs(c *gin.Context) {
	raw := c.Param("id")
	p := pagination.Parse(c)
	page, pageSize := p.Page, p.PageSize

	var uid uint64
	if id, err := strconv.ParseUint(raw, 10, 64); err == nil {
		uid = id
	} else {
		// 按 username 查找
		profile, err := h.svc.GetUserProfileByUsername(c.Request.Context(), raw, 0)
		if err != nil {
			response.Fail(c, err)
			return
		}
		uid = profile.ID
	}

	logs, total, err := h.svc.GetUserLogs(c.Request.Context(), uid, page, pageSize)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.SuccessPage(c, logs, total, page, pageSize)
}

// GetUserEndorsements 分页查询用户收到的合作评价
// GET /api/v1/users/:id/endorsements
func (h *UserHandler) GetUserEndorsements(c *gin.Context) {
	raw := c.Param("id")
	p := pagination.Parse(c)
	page, pageSize := p.Page, p.PageSize

	var uid uint64
	if id, err := strconv.ParseUint(raw, 10, 64); err == nil {
		uid = id
	} else {
		profile, err := h.svc.GetUserProfileByUsername(c.Request.Context(), raw, 0)
		if err != nil {
			response.Fail(c, err)
			return
		}
		uid = profile.ID
	}

	endorsements, total, err := h.svc.GetUserEndorsements(c.Request.Context(), uid, page, pageSize)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.SuccessPage(c, endorsements, total, page, pageSize)
}

// ==================== 个人资料 ====================

// UpdateProfile 更新个人资料（需 JWT）
// PUT /api/v1/users/me/profile
func (h *UserHandler) UpdateProfile(c *gin.Context) {
	userID, ok := requireLogin(c)
	if !ok {
		return
	}

	var req service.UpdateProfileReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailBadRequest(c, "请求参数错误："+err.Error())
		return
	}

	if err := h.svc.UpdateProfile(c.Request.Context(), userID, &req); err != nil {
		response.Fail(c, err)
		return
	}

	response.SuccessMsg(c, "资料更新成功", nil)
}

// UploadAvatarReq 提交头像 key 请求体
type UploadAvatarReq struct {
	Key string `json:"key" binding:"required"`
}

// UploadAvatar 提交头像 key（前端已通过预签名 URL 直传云存储后调用此接口）
// PUT /api/v1/me/avatar
func (h *UserHandler) UploadAvatar(c *gin.Context) {
	userID, ok := requireLogin(c)
	if !ok {
		return
	}

	var req UploadAvatarReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailBadRequest(c, "参数错误：需要 key 字段")
		return
	}

	if err := h.svc.SaveAvatar(c.Request.Context(), userID, req.Key); err != nil {
		response.Fail(c, err)
		return
	}

	// 返回预签名 GET URL 供前端展示（有效期 24 小时）
	presignedURL, err := storage.PresignedGetURL(c.Request.Context(), req.Key, 24*time.Hour)
	if err != nil {
		// 预签名 URL 生成失败不影响头像保存，但应记录日志下便于排查
		logger.Warn("预签名头像 URL 生成失败", zap.Uint64("user_id", userID), zap.String("key", req.Key), zap.Error(err))
		presignedURL = ""
	}
	response.Success(c, gin.H{"url": presignedURL, "key": req.Key})
}

// ==================== 技能标签 ====================

// UpdateSkills 更新用户技能标签（需 JWT）
// PUT /api/v1/users/me/skills
func (h *UserHandler) UpdateSkills(c *gin.Context) {
	userID, ok := requireLogin(c)
	if !ok {
		return
	}

	var req service.UpdateSkillsReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailBadRequest(c, "请求参数错误："+err.Error())
		return
	}

	skills, err := h.svc.UpdateSkills(c.Request.Context(), userID, &req)
	if err != nil {
		response.Fail(c, err)
		return
	}

	response.Success(c, gin.H{"skills": skills})
}

// ==================== 作品集 ====================

// GetMyPortfolio 获取自己的作品集（需 JWT）
// GET /api/v1/users/me/portfolio
func (h *UserHandler) GetMyPortfolio(c *gin.Context) {
	userID, ok := requireLogin(c)
	if !ok {
		return
	}

	portfolios, err := h.svc.GetMyPortfolio(c.Request.Context(), userID)
	if err != nil {
		response.Fail(c, err)
		return
	}

	response.Success(c, gin.H{"portfolio": portfolios})
}

// CreatePortfolio 添加作品（需 JWT）
// POST /api/v1/users/me/portfolio
func (h *UserHandler) CreatePortfolio(c *gin.Context) {
	userID, ok := requireLogin(c)
	if !ok {
		return
	}

	var req service.CreatePortfolioReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailBadRequest(c, "请求参数错误："+err.Error())
		return
	}

	portfolio, err := h.svc.CreatePortfolio(c.Request.Context(), userID, &req)
	if err != nil {
		response.Fail(c, err)
		return
	}

	response.SuccessMsg(c, "作品添加成功", portfolio)
}

// UpdatePortfolio 更新作品（需 JWT）
// PUT /api/v1/users/me/portfolio/:id
func (h *UserHandler) UpdatePortfolio(c *gin.Context) {
	userID, ok := requireLogin(c)
	if !ok {
		return
	}

	portfolioID, err := parseUint64Param(c, "id")
	if err != nil {
		response.FailBadRequest(c, "作品 ID 格式错误")
		return
	}

	var req service.UpdatePortfolioReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailBadRequest(c, "请求参数错误："+err.Error())
		return
	}

	portfolio, err := h.svc.UpdatePortfolio(c.Request.Context(), userID, portfolioID, &req)
	if err != nil {
		response.Fail(c, err)
		return
	}

	response.SuccessMsg(c, "作品更新成功", portfolio)
}

// DeletePortfolio 删除作品（需 JWT）
// DELETE /api/v1/users/me/portfolio/:id
func (h *UserHandler) DeletePortfolio(c *gin.Context) {
	userID, ok := requireLogin(c)
	if !ok {
		return
	}

	portfolioID, err := parseUint64Param(c, "id")
	if err != nil {
		response.FailBadRequest(c, "作品 ID 格式错误")
		return
	}

	if err := h.svc.DeletePortfolio(c.Request.Context(), userID, portfolioID); err != nil {
		response.Fail(c, err)
		return
	}

	response.SuccessMsg(c, "作品已删除", nil)
}

// ==================== 用户搜索 ====================

// SearchUsers 按 username/nickname 模糊搜索用户（用于 DM 发起时选人）
// GET /api/v1/users/search?q={keyword}&limit=10
func (h *UserHandler) SearchUsers(c *gin.Context) {
	keyword := c.Query("q")
	if keyword == "" {
		response.FailBadRequest(c, "搜索关键词不能为空（参数 q）")
		return
	}
	if len([]rune(keyword)) > 50 {
		response.FailBadRequest(c, "搜索关键词过长（最夐50字）")
		return
	}

	limitStr := c.DefaultQuery("limit", "10")
	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit <= 0 {
		limit = 10
	}
	if limit > 50 {
		limit = 50
	}

	items, err := h.svc.SearchUsers(c.Request.Context(), keyword, limit)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.Success(c, items)
}

// ==================== Handler 内部辅助函数 ====================

// ==================== 账号注销 ====================

// DeleteAccount 注销账号（GDPR 合规，需用户主动确认）
// DELETE /api/v1/users/me
func (h *UserHandler) DeleteAccount(c *gin.Context) {
	userID, ok := requireLogin(c)
	if !ok {
		return
	}
	var req service.DeleteAccountReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailBadRequest(c, err.Error())
		return
	}
	if err := h.svc.DeleteAccount(c.Request.Context(), userID, &req); err != nil {
		response.Fail(c, err)
		return
	}

	// 注销成功后立即吊销当前 Access Token
	authHeader := c.GetHeader("Authorization")
	parts := strings.SplitN(authHeader, " ", 2)
	if len(parts) == 2 && strings.EqualFold(parts[0], "Bearer") && parts[1] != "" {
		claimsVal, exists := c.Get(middleware.ClaimsKey)
		if exists {
			if claims, ok2 := claimsVal.(*auth.Claims); ok2 {
				expiresAt := time.Now().Add(2 * time.Hour)
				if claims.ExpiresAt != nil {
					expiresAt = claims.ExpiresAt.Time
				}
				_ = auth.RevokeToken(c.Request.Context(), parts[1], expiresAt)
			}
		}
	}

	response.SuccessMsg(c, "账号已注销，再见", nil)
}
