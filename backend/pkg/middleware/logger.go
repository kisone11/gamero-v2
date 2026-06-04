// logger.go 提供 HTTP 请求日志中间件，使用 zap 记录每个请求的详细信息。
package middleware

import (
	"bytes"
	"io"
	"net/http"
	"time"

	"github.com/gamero/gamero/pkg/logger"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// responseWriter 包装 gin.ResponseWriter 以捕获响应体和状态码
type responseWriter struct {
	gin.ResponseWriter
	body   *bytes.Buffer
	status int
}

func (w *responseWriter) WriteHeader(statusCode int) {
	w.status = statusCode
	w.ResponseWriter.WriteHeader(statusCode)
}

func (w *responseWriter) Write(b []byte) (int, error) {
	w.body.Write(b)
	return w.ResponseWriter.Write(b)
}

// RequestLogger 请求日志中间件
// 记录每个请求的方法、路径、状态码、耗时、客户端 IP 等信息
func RequestLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		rawQuery := c.Request.URL.RawQuery
		if rawQuery != "" {
			path = path + "?" + rawQuery
		}

		// 读取请求体（用于记录日志），然后恢复以供后续 handler 使用
		var requestBody []byte
		if c.Request.Body != nil && c.Request.ContentLength > 0 && c.Request.ContentLength <= 4096 {
			requestBody, _ := io.ReadAll(c.Request.Body) // Ignore read errors for logging
			c.Request.Body = io.NopCloser(bytes.NewBuffer(requestBody))
		}

		// 替换 ResponseWriter 以捕获响应
		rw := &responseWriter{
			ResponseWriter: c.Writer,
			body:           &bytes.Buffer{},
			status:         http.StatusOK,
		}
		c.Writer = rw

		// 处理请求
		c.Next()

		latency := time.Since(start)
		statusCode := rw.status
		if statusCode == 0 {
			statusCode = c.Writer.Status()
		}

		fields := []zap.Field{
			zap.String("method", c.Request.Method),
			zap.String("path", path),
			zap.Int("status", statusCode),
			zap.Duration("latency", latency),
			zap.String("client_ip", c.ClientIP()),
			zap.String("user_agent", c.Request.UserAgent()),
		}

		// 记录请求体（仅 POST/PUT/PATCH，且大小合理）
		if len(requestBody) > 0 && len(requestBody) <= 4096 {
			fields = append(fields, zap.ByteString("request_body", requestBody))
		}

		// 记录用户 ID（如果已认证）
		if userID, ok := GetUserID(c); ok {
			fields = append(fields, zap.Uint64("user_id", userID))
		}

		// 记录错误信息
		if len(c.Errors) > 0 {
			fields = append(fields, zap.String("errors", c.Errors.String()))
		}

		// 根据状态码决定日志级别
		if statusCode >= 500 {
			logger.Error("HTTP 请求", fields...)
		} else if statusCode >= 400 {
			logger.Warn("HTTP 请求", fields...)
		} else {
			logger.Info("HTTP 请求", fields...)
		}
	}
}
