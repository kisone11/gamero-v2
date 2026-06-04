// Package handler 提供内容审核回调的 HTTP Handler。
package handler

import (
	"io"
	"net/http"
	"strings"

	"github.com/gamero/gamero/internal/service"
	"github.com/gamero/gamero/pkg/logger"
	"github.com/gamero/gamero/pkg/response"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// ModerationHandler 内容审核回调 Handler
type ModerationHandler struct {
	svc        service.ModerationService
	allowedIPs string
}

// NewModerationHandler 创建 ModerationHandler
func NewModerationHandler(svc service.ModerationService, allowedIPs string) *ModerationHandler {
	return &ModerationHandler{svc: svc, allowedIPs: allowedIPs}
}

// HandleCallback 处理腾讯云 COS 审核回调
// POST /api/v1/callback/moderation
func (h *ModerationHandler) HandleCallback(c *gin.Context) {
	// IP 白名单校验
	if h.allowedIPs != "" {
		clientIP := c.ClientIP()
		allowed := false
		for _, ip := range strings.Split(h.allowedIPs, ",") {
			if strings.TrimSpace(ip) == clientIP {
				allowed = true
				break
			}
		}
		if !allowed {
			logger.Warn("moderation: IP not allowed", zap.String("ip", clientIP))
			c.AbortWithStatus(http.StatusForbidden)
			return
		}
	}

	// 读取回调签名（腾讯云通过 header 传递）
	signature := c.GetHeader("X-Cos-Signature")
	if signature == "" {
		signature = c.Query("sign")
	}

	// 读取请求体
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		logger.Warn("moderation: failed to read body", zap.Error(err))
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}

	if err := h.svc.HandleCallback(c.Request.Context(), signature, body); err != nil {
		logger.Warn("moderation: callback processing failed", zap.Error(err))
		response.Fail(c, err)
		return
	}

	// 始终返回 200 给腾讯云以避免回调重试
	response.Success(c, gin.H{"code": 0, "message": "success"})
}
