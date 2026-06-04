// Package errors 定义应用程序统一错误码和错误类型。
// 错误码规范：
//   - 1xxx: 通用错误
//   - 2xxx: 用户相关错误
//   - 3xxx: 认证/授权错误
//   - 4xxx: 资源相关错误
//   - 5xxx: 服务内部错误
package errors

import "fmt"

// Code 错误码类型
type Code int

const (
	// 通用错误码 1xxx
	CodeSuccess          Code = 0
	CodeBadRequest       Code = 1001
	CodeUnauthorized     Code = 1002
	CodeForbidden        Code = 1003
	CodeNotFound         Code = 1004
	CodeMethodNotAllowed Code = 1005
	CodeConflict         Code = 1006
	CodeTooManyRequests  Code = 1007
	CodeInternalError    Code = 1500
	CodeServiceUnavail   Code = 1503

	// 参数验证错误 11xx
	CodeParamInvalid  Code = 1100
	CodeParamMissing  Code = 1101
	CodeParamTooLong  Code = 1102
	CodeParamTooShort Code = 1103

	// 用户相关错误 2xxx
	CodeUserNotFound          Code = 2001
	CodeUserAlreadyExists     Code = 2002
	CodeUserDisabled          Code = 2003
	CodePasswordIncorrect     Code = 2004
	CodePasswordTooWeak       Code = 2005
	CodeEmailAlreadyExists    Code = 2006
	CodePhoneAlreadyExists    Code = 2007
	CodeUsernameAlreadyUsed   Code = 2008
	CodeNicknameAlreadyExists Code = 2009

	// 认证/授权错误 3xxx
	CodeTokenInvalid     Code = 3001
	CodeTokenExpired     Code = 3002
	CodeTokenRevoked     Code = 3003
	CodeTokenMissing     Code = 3004
	CodeRefreshFailed    Code = 3005
	CodePermissionDenied Code = 3006

	// 验证码错误 31xx
	CodeSMSCodeInvalid   Code = 3101
	CodeSMSCodeExpired   Code = 3102
	CodeSMSCodeFrequency Code = 3103
	CodeSMSSendFailed    Code = 3104
	CodeEmailCodeInvalid Code = 3105
	CodeEmailCodeExpired Code = 3106
	CodeEmailFrequency   Code = 3107
	CodeEmailSendFailed  Code = 3108

	// 资源相关错误 4xxx
	CodeResourceNotFound  Code = 4001
	CodeResourceForbidden Code = 4002
	CodeResourceConflict  Code = 4003
	CodeUploadFailed      Code = 4004
	CodeFileTooLarge      Code = 4005
	CodeFileTypeInvalid   Code = 4006

	// 项目相关错误 41xx
	CodeGameNotFound       Code = 4101
	CodeGameAlreadyExist   Code = 4102
	CodeProjectNotFound    Code = 4103
	CodeProjectForbidden   Code = 4104
	CodeMemberAlreadyExists Code = 4105
	CodeMemberNotFound     Code = 4106
	CodeScreenshotLimit    Code = 4108

	// 帖子相关错误 42xx
	CodePostNotFound  Code = 4201
	CodePostForbidden Code = 4202

	// 评论相关错误 43xx
	CodeCommentNotFound  Code = 4301
	CodeCommentForbidden Code = 4302

	// 组队系统相关错误 44xx
	CodeRecruitmentNotFound      Code = 4401 // 招募信息不存在
	CodeRecruitmentForbidden     Code = 4402 // 无权操作此招募
	CodeRecruitmentClosed        Code = 4403 // 招募已关闭或过期
	CodeApplicationNotFound      Code = 4404 // 申请不存在
	CodeApplicationForbidden     Code = 4405 // 无权操作此申请
	CodeApplicationDuplicate     Code = 4406 // 已申请过该招募
	CodeApplicationSelfProject   Code = 4407 // 不能申请自己项目的招募
	CodeApplicationNotPending    Code = 4408 // 申请状态不是待处理，无法操作
	CodeApplicationNotWithdrawable Code = 4409 // 申请不可撤回（已处理）
	CodeReviewNotFound           Code = 4410 // 评价不存在
	CodeReviewForbidden          Code = 4411 // 无权操作此评价
	CodeReviewDuplicate          Code = 4412 // 已对该成员发表过评价
	CodeReviewSupplementExists   Code = 4413 // 已有补充说明，不可重复补充
	CodeProjectNotFinished       Code = 4414 // 项目未结束，无法评价
	CodeNotBothMembers           Code = 4415 // 双方不都是项目成员
	CodeNotificationNotFound     Code = 4416 // 通知不存在
	CodeRecruitmentHeadcount     Code = 4417 // 需求人数不在有效范围
	CodeRecruitmentExpireDays    Code = 4418 // 有效期天数不在有效范围

	// 开发日志相关错误 45xx
	CodeDevLogNotFound           Code = 4501 // 开发日志不存在
	CodeDevLogForbidden          Code = 4502 // 无权操作此日志
	CodeDevLogAlreadyLiked       Code = 4503 // 已点赞，不可重复点赞
	CodeDevLogAlreadyCollected   Code = 4504 // 已收藏，不可重复收藏
	CodeDevLogNotPublished       Code = 4505 // 日志未发布，不可评论
	CodeDevLogImageLimit         Code = 4506 // 日志配图数量已达上限（最多6张）
	CodeDevLogVersionRequired    Code = 4507 // 版本更新日志必须填写版本号
	CodeDevLogCannotUnpublish    Code = 4508 // 已发布的日志不可改回草稿
	CodeDevLogCommentAlreadyLiked Code = 4509 // 评论已点赞，不可重复点赞

	// 关注系统相关错误 46xx
	CodeFollowAlreadyExists Code = 4601 // 已关注，不可重复关注
	CodeFollowNotFound      Code = 4602 // 关注关系不存在（取关时）
	CodeFollowSelf          Code = 4603 // 不能关注自己

	// 内容安全
	CodeContentViolation Code = 4701 // 内容包含敏感词或违规内容

	// 实名认证（KYC）50xx
	CodeKYCRequired Code = 5001 // 该操作需先完成实名认证

	// Chat 群聊系统 5100 段
	CodeChatRoomNotFound     Code = 5101
	CodeChatNotMember        Code = 5102
	CodeChatMessageNotFound  Code = 5103
	CodeChatMessageForbidden Code = 5104
	CodeChatRecallExpired    Code = 5105 // 超过24h无法撤回
	CodeChatFileTypeInvalid  Code = 5106
	CodeChatFileTooLarge     Code = 5107

)

// messages 错误码对应的中文描述
var messages = map[Code]string{
	CodeSuccess:          "成功",
	CodeBadRequest:       "请求参数错误",
	CodeUnauthorized:     "未登录或登录已过期",
	CodeForbidden:        "无权限访问",
	CodeNotFound:         "资源不存在",
	CodeMethodNotAllowed: "请求方法不允许",
	CodeConflict:         "资源冲突",
	CodeTooManyRequests:  "请求过于频繁，请稍后再试",
	CodeInternalError:    "服务器内部错误",
	CodeServiceUnavail:   "服务暂不可用",

	CodeParamInvalid:  "参数格式不正确",
	CodeParamMissing:  "缺少必要参数",
	CodeParamTooLong:  "参数值过长",
	CodeParamTooShort: "参数值过短",

	CodeUserNotFound:          "用户不存在",
	CodeUserAlreadyExists:     "用户已存在",
	CodeUserDisabled:          "账号已被禁用",
	CodePasswordIncorrect:     "密码错误",
	CodePasswordTooWeak:       "密码强度不足",
	CodeEmailAlreadyExists:    "邮箱已被注册",
	CodePhoneAlreadyExists:    "手机号已被注册",
	CodeUsernameAlreadyUsed:   "用户名已被使用",
	CodeNicknameAlreadyExists: "昵称已被使用",

	CodeTokenInvalid:     "Token 无效",
	CodeTokenExpired:     "Token 已过期",
	CodeTokenRevoked:     "Token 已吊销",
	CodeTokenMissing:     "Token 缺失",
	CodeRefreshFailed:    "刷新 Token 失败",
	CodePermissionDenied: "权限不足",

	CodeSMSCodeInvalid:   "短信验证码错误",
	CodeSMSCodeExpired:   "短信验证码已过期",
	CodeSMSCodeFrequency: "短信发送过于频繁，请稍后再试",
	CodeSMSSendFailed:    "短信发送失败",
	CodeEmailCodeInvalid: "邮箱验证码错误",
	CodeEmailCodeExpired: "邮箱验证码已过期",
	CodeEmailFrequency:   "邮件发送过于频繁，请稍后再试",
	CodeEmailSendFailed:  "邮件发送失败",

	CodeResourceNotFound:  "资源不存在",
	CodeResourceForbidden: "无权操作此资源",
	CodeResourceConflict:  "资源冲突",
	CodeUploadFailed:      "文件上传失败",
	CodeFileTooLarge:      "文件超过大小限制",
	CodeFileTypeInvalid:   "文件类型不支持",

	CodeGameNotFound:        "游戏不存在",
	CodeGameAlreadyExist:    "游戏已存在",
	CodeProjectNotFound:     "项目不存在",
	CodeProjectForbidden:    "无权操作此项目",
	CodeMemberAlreadyExists: "该用户已是项目成员",
	CodeMemberNotFound:      "成员不存在",
	CodeScreenshotLimit:     "截图数量已达上限（最多10张）",

	CodePostNotFound:  "帖子不存在",
	CodePostForbidden: "无权操作此帖子",

	CodeCommentNotFound:  "评论不存在",
	CodeCommentForbidden: "无权操作此评论",

	CodeRecruitmentNotFound:        "招募信息不存在",
	CodeRecruitmentForbidden:       "无权操作此招募",
	CodeRecruitmentClosed:          "招募已关闭或过期",
	CodeApplicationNotFound:        "申请记录不存在",
	CodeApplicationForbidden:       "无权操作此申请",
	CodeApplicationDuplicate:       "已申请过该招募，不可重复申请",
	CodeApplicationSelfProject:     "不能申请自己项目的招募",
	CodeApplicationNotPending:      "申请状态不是待处理，无法继续操作",
	CodeApplicationNotWithdrawable: "申请已处理，不可撤回",
	CodeReviewNotFound:             "评价记录不存在",
	CodeReviewForbidden:            "无权操作此评价",
	CodeReviewDuplicate:            "已对该成员发表过评价，不可重复评价",
	CodeReviewSupplementExists:     "已有补充说明，不可再次补充",
	CodeProjectNotFinished:         "项目未结束（需要 launched 或 abandoned 状态）",
	CodeNotBothMembers:             "评价人或被评人不是该项目成员",
	CodeNotificationNotFound:       "通知不存在",
	CodeRecruitmentHeadcount:       "需求人数须在 1-10 之间",
	CodeRecruitmentExpireDays:      "有效期须在 1-90 天之间",

	CodeDevLogNotFound:            "开发日志不存在",
	CodeDevLogForbidden:           "无权操作此日志",
	CodeDevLogAlreadyLiked:        "已点赞，不可重复点赞",
	CodeDevLogAlreadyCollected:    "已收藏，不可重复收藏",
	CodeDevLogNotPublished:        "日志未发布，不可进行此操作",
	CodeDevLogImageLimit:          "日志配图数量已达上限（最多6张）",
	CodeDevLogVersionRequired:     "版本更新日志必须填写版本号",
	CodeDevLogCannotUnpublish:     "已发布的日志不可改回草稿",
	CodeDevLogCommentAlreadyLiked: "评论已点赞，不可重复点赞",

	CodeFollowAlreadyExists: "已关注，不可重复关注",
	CodeFollowNotFound:      "关注关系不存在",
	CodeFollowSelf:          "不能关注自己",

	CodeContentViolation: "内容包含敏感词，请修改后重试",

	CodeKYCRequired: "该操作需先完成实名认证",

	CodeChatRoomNotFound:     "群聊房间不存在",
	CodeChatNotMember:        "您不是该群聊的成员",
	CodeChatMessageNotFound:  "消息不存在",
	CodeChatMessageForbidden: "无权操作此消息",
	CodeChatRecallExpired:    "消息发送超过24小时，无法撤回",
	CodeChatFileTypeInvalid:  "文件类型不支持",
	CodeChatFileTooLarge:     "文件超过大小限制",

}

// Message 根据错误码返回对应的中文描述
func (c Code) Message() string {
	if msg, ok := messages[c]; ok {
		return msg
	}
	return "未知错误"
}

// AppError 应用程序错误，包含错误码和详细信息
type AppError struct {
	Code    Code   // 业务错误码
	Message string // 错误描述
	Err     error  // 原始错误（可选）
}

// Error 实现 error 接口
func (e *AppError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("[%d] %s: %v", e.Code, e.Message, e.Err)
	}
	return fmt.Sprintf("[%d] %s", e.Code, e.Message)
}

// Unwrap 返回原始错误，支持 errors.Is/As
func (e *AppError) Unwrap() error {
	return e.Err
}

// New 创建一个 AppError
func New(code Code, msg string) *AppError {
	return &AppError{
		Code:    code,
		Message: msg,
	}
}

// Newf 创建带格式化消息的 AppError
func Newf(code Code, format string, args ...interface{}) *AppError {
	return &AppError{
		Code:    code,
		Message: fmt.Sprintf(format, args...),
	}
}

// Wrap 包装原始错误为 AppError
func Wrap(code Code, err error) *AppError {
	return &AppError{
		Code:    code,
		Message: code.Message(),
		Err:     err,
	}
}

// WrapMsg 包装原始错误，自定义消息
func WrapMsg(code Code, msg string, err error) *AppError {
	return &AppError{
		Code:    code,
		Message: msg,
		Err:     err,
	}
}

// CodeError 使用错误码的默认消息创建 AppError
func CodeError(code Code) *AppError {
	return &AppError{
		Code:    code,
		Message: code.Message(),
	}
}

// IsAppError 判断是否为 AppError 类型
func IsAppError(err error) (*AppError, bool) {
	if err == nil {
		return nil, false
	}
	appErr, ok := err.(*AppError)
	return appErr, ok
}
