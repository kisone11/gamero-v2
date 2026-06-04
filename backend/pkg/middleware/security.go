// Package middleware 提供 HTTP 安全响应头中间件。
// 向所有响应注入安全头，防御常见 Web 攻击（MIME 嗅探、点击劫持、信息泄露等）。
package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// SecurityHeaders 注入安全响应头中间件。
// 建议挂载在所有路由组最外层（engine.Use 级别）。
func SecurityHeaders() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 防止 MIME 嗅探（浏览器不猜内容类型）
		c.Header("X-Content-Type-Options", "nosniff")
		// 禁止页面被嵌入 iframe（防点击劫持）
		c.Header("X-Frame-Options", "DENY")
		// 关闭旧版 XSS filter（现代浏览器不再推荐）
		c.Header("X-XSS-Protection", "0")
		// 严格 referrer 策略（跨域请求不携带 path）
		c.Header("Referrer-Policy", "strict-origin-when-cross-origin")
		// 禁止页面访问摄像头/麦克风/地理位置（非游戏功能）
		c.Header("Permissions-Policy", "camera=(), microphone=(), geolocation=()")
		// 启用 HSTS（强制 HTTPS，有效期 1 年，包含子域名）
		c.Header("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		// 不缓存认证相关接口
		if c.FullPath() != "" {
			c.Header("Cache-Control", "no-store")
		}
		c.Next()
	}
}

// maxJSONBodyBytes JSON/表单请求体最大允许大小：4MB（文件上传单独限制）
const maxJSONBodyBytes = 4 << 20 // 4MB

// MaxBodySize 限制 JSON/表单请求体大小的中间件，防止请求体炸弹攻击。
// 文件上传路由应跳过此中间件（配置了独立的 size 校验）。
func MaxBodySize() gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.ContentLength > maxJSONBodyBytes {
			c.AbortWithStatusJSON(http.StatusRequestEntityTooLarge, gin.H{
				"code": 40013,
				"msg":  "请求体超过最大限制（4MB）",
			})
			return
		}
		// 即使 Content-Length 无，也限制实际读取大小（防止 chunked encoding 绕过）
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxJSONBodyBytes)
		c.Next()
	}
}
