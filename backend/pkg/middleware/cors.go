// cors.go 提供 CORS（跨域资源共享）中间件。
package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// CORSConfig CORS 配置
type CORSConfig struct {
	AllowOrigins     []string // 允许的来源，["*"] 表示允许所有
	AllowMethods     []string // 允许的 HTTP 方法
	AllowHeaders     []string // 允许的请求头
	ExposeHeaders    []string // 暴露给客户端的响应头
	AllowCredentials bool     // 是否允许携带 Cookie
	MaxAge           int      // Preflight 请求缓存时间（秒）
}

// DefaultCORSConfig 返回默认的 CORS 配置（允许所有来源，适合开发环境）
func DefaultCORSConfig() CORSConfig {
	return CORSConfig{
		AllowOrigins: []string{"*"},
		AllowMethods: []string{
			http.MethodGet, http.MethodPost, http.MethodPut,
			http.MethodPatch, http.MethodDelete, http.MethodOptions,
			http.MethodHead,
		},
		AllowHeaders: []string{
			"Origin", "Content-Type", "Content-Length",
			"Accept", "Authorization", "X-Requested-With",
			"X-Request-ID", "X-Trace-ID",
		},
		ExposeHeaders:    []string{"Content-Length", "X-Request-ID"},
		AllowCredentials: false,
		MaxAge:           86400, // 24 小时
	}
}

// CORS 返回使用默认配置的 CORS 中间件
func CORS() gin.HandlerFunc {
	return CORSWithConfig(DefaultCORSConfig())
}

// CORSWithConfig 返回自定义配置的 CORS 中间件
func CORSWithConfig(cfg CORSConfig) gin.HandlerFunc {
	allowOriginSet := make(map[string]bool)
	for _, origin := range cfg.AllowOrigins {
		allowOriginSet[origin] = true
	}

	allowMethodsStr := joinStrings(cfg.AllowMethods, ", ")
	allowHeadersStr := joinStrings(cfg.AllowHeaders, ", ")
	exposeHeadersStr := joinStrings(cfg.ExposeHeaders, ", ")
	maxAgeStr := ""
	if cfg.MaxAge > 0 {
		maxAgeStr = intToStr(cfg.MaxAge)
	}

	return func(c *gin.Context) {
		origin := c.Request.Header.Get("Origin")

		// 设置 CORS 响应头
		if allowOriginSet["*"] {
			if cfg.AllowCredentials {
				// 携带 Cookie 时不能使用通配符 *，需要指定具体 origin
				if origin != "" {
					c.Header("Access-Control-Allow-Origin", origin)
					c.Header("Vary", "Origin")
				} else {
					c.Header("Access-Control-Allow-Origin", "*")
				}
			} else {
				c.Header("Access-Control-Allow-Origin", "*")
			}
		} else if origin != "" && allowOriginSet[origin] {
			c.Header("Access-Control-Allow-Origin", origin)
			c.Header("Vary", "Origin")
		}

		if cfg.AllowCredentials {
			c.Header("Access-Control-Allow-Credentials", "true")
		}

		if allowMethodsStr != "" {
			c.Header("Access-Control-Allow-Methods", allowMethodsStr)
		}

		if allowHeadersStr != "" {
			c.Header("Access-Control-Allow-Headers", allowHeadersStr)
		}

		if exposeHeadersStr != "" {
			c.Header("Access-Control-Expose-Headers", exposeHeadersStr)
		}

		// 处理 Preflight 请求（OPTIONS）
		if c.Request.Method == http.MethodOptions {
			if maxAgeStr != "" {
				c.Header("Access-Control-Max-Age", maxAgeStr)
			}
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}

// joinStrings 使用分隔符连接字符串切片
func joinStrings(strs []string, sep string) string {
	if len(strs) == 0 {
		return ""
	}
	result := strs[0]
	for _, s := range strs[1:] {
		result += sep + s
	}
	return result
}

// intToStr 整数转字符串
func intToStr(n int) string {
	if n == 0 {
		return "0"
	}
	result := ""
	neg := n < 0
	if neg {
		n = -n
	}
	for n > 0 {
		result = string(rune('0'+n%10)) + result
		n /= 10
	}
	if neg {
		result = "-" + result
	}
	return result
}
