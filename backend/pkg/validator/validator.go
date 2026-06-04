// Package validator 提供自定义验证规则，扩展 Gin 的默认验证器。
// 注册自定义验证标签：slug, future_date, chinese_phone, safe_content 等。
package validator

import (
	"reflect"
	"regexp"
	"strings"
	"time"

	"github.com/gin-gonic/gin/binding"
	"github.com/go-playground/validator/v10"
)

var (
	// slugRegex slug 格式：小写字母、数字、连字符（不能以连字符开头或结尾）
	slugRegex = regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)
	// chinesePhoneRegex 中国大陆手机号：1开头11位数字
	chinesePhoneRegex = regexp.MustCompile(`^1[3-9]\d{9}$`)
	// htmlTagRegex 检测 HTML 标签
	htmlTagRegex = regexp.MustCompile(`<[^>]*>`)
	// scriptTagRegex 检测 script 标签
	scriptTagRegex = regexp.MustCompile(`(?i)<script[^>]*>.*?</script>`)
)

// RegisterCustomValidators 注册所有自定义验证规则到 Gin 的验证器
func RegisterCustomValidators() error {
	if v, ok := binding.Validator.Engine().(*validator.Validate); ok {
		// 注册 slug 验证规则
		if err := v.RegisterValidation("slug", validateSlug); err != nil {
			return err
		}
		// 注册 future_date 验证规则
		if err := v.RegisterValidation("future_date", validateFutureDate); err != nil {
			return err
		}
		// 注册 chinese_phone 验证规则
		if err := v.RegisterValidation("chinese_phone", validateChinesePhone); err != nil {
			return err
		}
		// 注册 safe_content 验证规则
		if err := v.RegisterValidation("safe_content", validateSafeContent); err != nil {
			return err
		}
	}
	return nil
}

// validateSlug 验证 slug 格式：只允许小写字母、数字、连字符
// 不能以连字符开头或结尾，不能包含连续连字符
func validateSlug(fl validator.FieldLevel) bool {
	slug := fl.Field().String()
	if slug == "" {
		return true // 空值由 required 标签处理
	}
	return slugRegex.MatchString(slug)
}

// validateFutureDate 验证日期必须是未来时间
// 支持格式：RFC3339, 2006-01-02T15:04:05Z07:00, time.Time 类型
func validateFutureDate(fl validator.FieldLevel) bool {
	field := fl.Field()
	
	var t time.Time
	// 字符串类型，尝试解析为时间
	if field.Kind() == reflect.String {
		str := field.String()
		if str == "" {
			return true // 空值由 required 标签处理
		}
		var err error
		// 尝试多种时间格式
		t, err = time.Parse(time.RFC3339, str)
		if err != nil {
			t, err = time.Parse("2006-01-02T15:04:05Z07:00", str)
		}
		if err != nil {
			t, err = time.Parse("2006-01-02 15:04:05", str)
		}
		if err != nil {
			t, err = time.Parse("2006-01-02", str)
		}
		if err != nil {
			return false // 无法解析的时间格式
		}
	} else if field.Kind() == reflect.Interface {
		// 接口类型，尝试转换为 time.Time
		if timeVal, ok := field.Interface().(time.Time); ok {
			t = timeVal
		} else {
			return false
		}
	} else {
		return false
	}
	
	// 检查是否为未来时间（至少在当前时间 1 分钟之后）
	return t.After(time.Now().Add(time.Minute))
}

// validateChinesePhone 验证中国大陆手机号格式
// 格式：1开头，第二位 3-9，共 11 位数字
func validateChinesePhone(fl validator.FieldLevel) bool {
	phone := fl.Field().String()
	if phone == "" {
		return true // 空值由 required 标签处理
	}
	return chinesePhoneRegex.MatchString(phone)
}

// validateSafeContent 验证内容不包含 HTML/script 标签
// 防止 XSS 攻击
func validateSafeContent(fl validator.FieldLevel) bool {
	content := fl.Field().String()
	if content == "" {
		return true // 空值由 required 标签处理
	}
	
	// 检测 script 标签（大小写不敏感）
	if scriptTagRegex.MatchString(content) {
		return false
	}
	
	// 检测其他 HTML 标签
	if htmlTagRegex.MatchString(content) {
		return false
	}
	
	return true
}

// ValidateEnum 辅助函数：验证字符串是否在允许的枚举值列表中
func ValidateEnum(value string, allowedValues []string) bool {
	for _, allowed := range allowedValues {
		if value == allowed {
			return true
		}
	}
	return false
}

// SanitizeContent 辅助函数：清理内容中的危险字符
// 移除 HTML 标签，保留纯文本
func SanitizeContent(content string) string {
	// 移除 script 标签及其内容
	content = scriptTagRegex.ReplaceAllString(content, "")
	// 移除其他 HTML 标签
	content = htmlTagRegex.ReplaceAllString(content, "")
	// 移除多余空白字符
	content = strings.TrimSpace(content)
	return content
}
