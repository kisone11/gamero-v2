// Package response 提供统一的 HTTP JSON 响应格式封装。
// 所有 API 响应均使用此包提供的函数，确保响应格式一致。
// 响应结构：{ "code": 0, "message": "成功", "data": {...} }
package response

import (
	"net/http"
	"strings"

	"github.com/gamero/gamero/pkg/errors"
	"github.com/gamero/gamero/pkg/logger"
	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"go.uber.org/zap"
)

// Response 统一响应结构体
type Response struct {
	Code    errors.Code `json:"code"`           // 业务错误码，0 表示成功
	Message string      `json:"message"`        // 响应消息
	Data    interface{} `json:"data,omitempty"` // 响应数据（成功时有值）
}

// PageData 分页数据结构
type PageData struct {
	List     interface{} `json:"list"`      // 数据列表
	Total    int64       `json:"total"`     // 总记录数
	Page     int         `json:"page"`      // 当前页码
	PageSize int         `json:"page_size"` // 每页大小
	Pages    int         `json:"pages"`     // 总页数
}

// Success 返回成功响应
func Success(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, Response{
		Code:    errors.CodeSuccess,
		Message: errors.CodeSuccess.Message(),
		Data:    data,
	})
}

// SuccessMsg 返回带自定义消息的成功响应
func SuccessMsg(c *gin.Context, msg string, data interface{}) {
	c.JSON(http.StatusOK, Response{
		Code:    errors.CodeSuccess,
		Message: msg,
		Data:    data,
	})
}

// SuccessPage 返回分页成功响应
func SuccessPage(c *gin.Context, list interface{}, total int64, page, pageSize int) {
	pages := 0
	if pageSize > 0 {
		pages = int((total + int64(pageSize) - 1) / int64(pageSize))
	}
	c.JSON(http.StatusOK, Response{
		Code:    errors.CodeSuccess,
		Message: errors.CodeSuccess.Message(),
		Data: PageData{
			List:     list,
			Total:    total,
			Page:     page,
			PageSize: pageSize,
			Pages:    pages,
		},
	})
}

// Fail 返回失败响应，使用 AppError
func Fail(c *gin.Context, err error) {
	if appErr, ok := errors.IsAppError(err); ok {
		httpStatus := appErrorToHTTPStatus(appErr.Code)
		c.JSON(httpStatus, Response{
			Code:    appErr.Code,
			Message: appErr.Message,
		})
		return
	}
	// 非 AppError，返回内部错误
	logger.Error("unexpected error in handler", zap.Error(err))
	c.JSON(http.StatusInternalServerError, Response{
		Code:    errors.CodeInternalError,
		Message: errors.CodeInternalError.Message(),
	})
}

// FailCode 根据错误码返回失败响应
func FailCode(c *gin.Context, code errors.Code) {
	c.JSON(appErrorToHTTPStatus(code), Response{
		Code:    code,
		Message: code.Message(),
	})
}

// FailMsg 返回带自定义消息的失败响应
func FailMsg(c *gin.Context, code errors.Code, msg string) {
	c.JSON(appErrorToHTTPStatus(code), Response{
		Code:    code,
		Message: msg,
	})
}

// FailBadRequest 返回 400 错误
func FailBadRequest(c *gin.Context, msg string) {
	c.JSON(http.StatusBadRequest, Response{
		Code:    errors.CodeBadRequest,
		Message: msg,
	})
}

// FailUnauthorized 返回 401 错误
func FailUnauthorized(c *gin.Context) {
	c.JSON(http.StatusUnauthorized, Response{
		Code:    errors.CodeUnauthorized,
		Message: errors.CodeUnauthorized.Message(),
	})
}

// FailForbidden 返回 403 错误
func FailForbidden(c *gin.Context) {
	c.JSON(http.StatusForbidden, Response{
		Code:    errors.CodeForbidden,
		Message: errors.CodeForbidden.Message(),
	})
}

// FailNotFound 返回 404 错误
func FailNotFound(c *gin.Context, msg string) {
	c.JSON(http.StatusNotFound, Response{
		Code:    errors.CodeNotFound,
		Message: msg,
	})
}

// FailInternal 返回 500 错误
func FailInternal(c *gin.Context) {
	c.JSON(http.StatusInternalServerError, Response{
		Code:    errors.CodeInternalError,
		Message: errors.CodeInternalError.Message(),
	})
}

// FailTooManyRequests 返回 429 错误（限流）
func FailTooManyRequests(c *gin.Context) {
	c.JSON(http.StatusTooManyRequests, Response{
		Code:    errors.CodeTooManyRequests,
		Message: errors.CodeTooManyRequests.Message(),
	})
}

// ValidationError 处理参数验证错误，返回详细的错误信息
func ValidationError(c *gin.Context, err error) {
	if validationErrs, ok := err.(validator.ValidationErrors); ok {
		// validator 验证错误，构造详细错误信息
		var errMsgs []string
		for _, e := range validationErrs {
			errMsgs = append(errMsgs, formatValidationError(e))
		}
		c.JSON(http.StatusBadRequest, Response{
			Code:    errors.CodeParamInvalid,
			Message: strings.Join(errMsgs, "; "),
		})
		return
	}
	// 其他绑定错误
	c.JSON(http.StatusBadRequest, Response{
		Code:    errors.CodeBadRequest,
		Message: "请求参数错误：" + err.Error(),
	})
}

// formatValidationError 格式化验证错误信息为友好的中文提示
func formatValidationError(e validator.FieldError) string {
	field := e.Field()
	tag := e.Tag()
	param := e.Param()

	switch tag {
	case "required":
		return field + " 不能为空"
	case "min":
		return field + " 长度不能少于 " + param + " 个字符"
	case "max":
		return field + " 长度不能超过 " + param + " 个字符"
	case "len":
		return field + " 长度必须为 " + param + " 个字符"
	case "email":
		return field + " 必须是有效的邮箱地址"
	case "alphanum":
		return field + " 只能包含字母和数字"
	case "numeric":
		return field + " 必须是数字"
	case "gt":
		return field + " 必须大于 " + param
	case "gte":
		return field + " 必须大于等于 " + param
	case "lt":
		return field + " 必须小于 " + param
	case "lte":
		return field + " 必须小于等于 " + param
	case "oneof":
		return field + " 必须是以下值之一：" + param
	case "url":
		return field + " 必须是有效的 URL"
	case "slug":
		return field + " 只能包含小写字母、数字和连字符"
	case "future_date":
		return field + " 必须是未来的日期"
	case "chinese_phone":
		return field + " 必须是有效的中国大陆手机号"
	case "safe_content":
		return field + " 包含不允许的HTML标签或脚本"
	default:
		return field + " 验证失败：" + tag
	}
}

// appErrorToHTTPStatus 将业务错误码映射到 HTTP 状态码
func appErrorToHTTPStatus(code errors.Code) int {
	switch code {
	case errors.CodeSuccess:
		return http.StatusOK

	// 400 Bad Request：参数错误、输入校验失败、业务前置条件不满足
	case errors.CodeBadRequest, errors.CodeParamInvalid, errors.CodeParamMissing,
		errors.CodeParamTooLong, errors.CodeParamTooShort,
		errors.CodePasswordTooWeak,
		errors.CodeFileTooLarge, errors.CodeFileTypeInvalid, errors.CodeUploadFailed,
		errors.CodeSMSCodeInvalid, errors.CodeSMSCodeExpired, errors.CodeSMSSendFailed,
		errors.CodeEmailCodeInvalid, errors.CodeEmailCodeExpired, errors.CodeEmailSendFailed,
		errors.CodeContentViolation,
		errors.CodeScreenshotLimit, errors.CodeRecruitmentHeadcount, errors.CodeRecruitmentExpireDays,
		errors.CodeDevLogImageLimit, errors.CodeDevLogVersionRequired,
		errors.CodeDevLogCannotUnpublish, errors.CodeDevLogNotPublished,
		errors.CodeApplicationNotPending, errors.CodeApplicationNotWithdrawable,
		errors.CodeApplicationSelfProject,
		errors.CodeProjectNotFinished, errors.CodeNotBothMembers,
		errors.CodeFollowSelf,
		errors.CodeRefreshFailed, errors.CodePasswordIncorrect,
		errors.CodeRecruitmentClosed:
		return http.StatusBadRequest

	// 401 Unauthorized：未登录或 token 失效
	case errors.CodeUnauthorized, errors.CodeTokenInvalid, errors.CodeTokenExpired,
		errors.CodeTokenRevoked, errors.CodeTokenMissing:
		return http.StatusUnauthorized

	// 403 Forbidden：已登录但无权限
	case errors.CodeForbidden, errors.CodePermissionDenied, errors.CodeResourceForbidden,
		errors.CodePostForbidden, errors.CodeCommentForbidden, errors.CodeProjectForbidden,
		errors.CodeRecruitmentForbidden, errors.CodeApplicationForbidden, errors.CodeReviewForbidden,
		errors.CodeDevLogForbidden, errors.CodeUserDisabled:
		return http.StatusForbidden

	// 404 Not Found：资源不存在
	case errors.CodeNotFound, errors.CodeUserNotFound, errors.CodeResourceNotFound,
		errors.CodeGameNotFound, errors.CodePostNotFound, errors.CodeCommentNotFound,
		errors.CodeProjectNotFound, errors.CodeMemberNotFound,
		errors.CodeRecruitmentNotFound, errors.CodeApplicationNotFound,
		errors.CodeReviewNotFound, errors.CodeNotificationNotFound,
		errors.CodeDevLogNotFound, errors.CodeFollowNotFound:
		return http.StatusNotFound

	// 405 Method Not Allowed
	case errors.CodeMethodNotAllowed:
		return http.StatusMethodNotAllowed

	// 409 Conflict：重复创建或状态冲突
	case errors.CodeConflict, errors.CodeUserAlreadyExists, errors.CodeEmailAlreadyExists,
		errors.CodeNicknameAlreadyExists,
		errors.CodePhoneAlreadyExists, errors.CodeUsernameAlreadyUsed, errors.CodeResourceConflict,
		errors.CodeGameAlreadyExist, errors.CodeMemberAlreadyExists,
		errors.CodeApplicationDuplicate, errors.CodeReviewDuplicate, errors.CodeReviewSupplementExists,
		errors.CodeDevLogAlreadyLiked, errors.CodeDevLogAlreadyCollected, errors.CodeDevLogCommentAlreadyLiked,
		errors.CodeFollowAlreadyExists:
		return http.StatusConflict

	// 429 Too Many Requests：频率限制
	case errors.CodeTooManyRequests, errors.CodeSMSCodeFrequency, errors.CodeEmailFrequency:
		return http.StatusTooManyRequests

	// 503 Service Unavailable
	case errors.CodeServiceUnavail:
		return http.StatusServiceUnavailable

	default:
		return http.StatusInternalServerError
	}
}
