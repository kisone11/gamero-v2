// recovery.go 提供 Panic Recovery 中间件，捕获 panic 并返回 500 错误响应。
package middleware

import (
	"fmt"
	"net/http"
	"runtime/debug"

	"github.com/gamero/gamero/pkg/logger"
	"github.com/gamero/gamero/pkg/response"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// Recovery Panic Recovery 中间件
// 捕获处理器中的 panic，记录堆栈信息并返回 500 响应
func Recovery() gin.HandlerFunc {
	return RecoveryWithHandler(defaultPanicHandler)
}

// defaultPanicHandler 默认 panic 处理函数
func defaultPanicHandler(c *gin.Context, err interface{}) {
	stack := debug.Stack()
	logger.Error("Panic 捕获",
		zap.Any("error", err),
		zap.String("method", c.Request.Method),
		zap.String("path", c.Request.URL.Path),
		zap.String("client_ip", c.ClientIP()),
		zap.ByteString("stack", stack),
	)

	c.JSON(http.StatusInternalServerError, response.Response{
		Code:    1500,
		Message: "服务器内部错误",
	})
}

// RecoveryWithHandler 自定义 panic 处理函数的 Recovery 中间件
func RecoveryWithHandler(handler func(c *gin.Context, err interface{})) gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				// 检查是否为客户端断开连接（写入已关闭连接的错误，不需要记录）
				if isBrokenPipe(err) {
					logger.Warn("客户端断开连接",
						zap.Any("error", err),
						zap.String("path", c.Request.URL.Path),
					)
					c.Abort()
					return
				}

				handler(c, err)
				c.Abort()
			}
		}()
		c.Next()
	}
}

// isBrokenPipe 判断是否为客户端连接断开错误
func isBrokenPipe(err interface{}) bool {
	errStr := fmt.Sprintf("%v", err)
	return errStr == "write tcp: broken pipe" ||
		errStr == "write: broken pipe" ||
		errStr == "connection reset by peer"
}
