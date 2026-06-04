// Package handler 提供统一动态流的 HTTP 处理器。
// 本文件包含 /api/v1/feed 端点的实现。
package handler

import (
	"strconv"

	"github.com/gamero/gamero/internal/service"
	"github.com/gamero/gamero/pkg/response"
	"github.com/gin-gonic/gin"
)

// FeedHandler 动态流模块的 Handler 集合
type FeedHandler struct {
	svc service.FeedService
}

// NewFeedHandler 创建 FeedHandler 实例
func NewFeedHandler(svc service.FeedService) *FeedHandler {
	return &FeedHandler{svc: svc}
}

// GetFeed 获取当前用户的统一动态流（游标分页）
// GET /api/v1/feed?cursor=<unix_nano>&limit=20
//
// cursor: 上一页最后一条的 created_at UnixNano 时间戳，0 表示从最新开始。
// limit: 每页条数，默认 20，最大 100。
func (h *FeedHandler) GetFeed(c *gin.Context) {
	userID, ok := requireLogin(c)
	if !ok {
		return
	}

	// 解析 limit
	limit := 20
	if limitStr := c.Query("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
			limit = l
			if limit > 100 {
				limit = 100
			}
		}
	}

	// 解析 cursor（UnixNano 时间戳）
	var cursor int64
	if cursorStr := c.Query("cursor"); cursorStr != "" {
		cursor, _ = strconv.ParseInt(cursorStr, 10, 64)
	}

	items, nextCursor, err := h.svc.GetFeed(c.Request.Context(), userID, cursor, limit)
	if err != nil {
		response.Fail(c, err)
		return
	}

	response.Success(c, gin.H{
		"items":       items,
		"next_cursor": nextCursor,
	})
}
