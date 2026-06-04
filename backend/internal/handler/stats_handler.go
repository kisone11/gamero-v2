// Package handler 提供创作者数据统计的 HTTP 请求处理层。
// 包含以下接口：
//   - GET /api/v1/stats/me?days=30     — 查看我的创作者统计数据
//   - GET /api/v1/projects/:id/stats?days=30 — 查看指定项目统计数据
//
// 所有接口均需 JWT 认证。
package handler

import (
	"strconv"

	"github.com/gamero/gamero/internal/service"
	"github.com/gamero/gamero/pkg/response"
	"github.com/gin-gonic/gin"
)

// StatsHandler 统计数据处理器
type StatsHandler struct {
	statsSvc service.StatsService
}

// NewStatsHandler 创建 StatsHandler 实例
func NewStatsHandler(statsSvc service.StatsService) *StatsHandler {
	return &StatsHandler{statsSvc: statsSvc}
}

// myStatsResponse 前端 MyStats 兼容格式（flat 结构）
type myStatsResponse struct {
	TotalProjects      int64 `json:"total_projects"`
	TotalLogs          int64 `json:"total_logs"`
	TotalPosts         int64 `json:"total_posts"`
	TotalFollowers     int64 `json:"total_followers"`
	TotalFollowing     int64 `json:"total_following"`
	TotalLikesReceived int64 `json:"total_likes_received"`
	TotalViews         int64 `json:"total_views"`
}

// GetMyStats 获取当前用户的创作者统计数据
// GET /api/v1/stats/me?days=30
func (h *StatsHandler) GetMyStats(c *gin.Context) {
	userID, ok := requireLogin(c)
	if !ok {
		return
	}

	days, _ := strconv.Atoi(c.DefaultQuery("days", "30"))

	stats, err := h.statsSvc.GetMyStats(c.Request.Context(), userID, days)
	if err != nil {
		response.Fail(c, err)
		return
	}

	// 将 Total 字段平铺为前端期望的 MyStats 格式
	var flat myStatsResponse
	if stats.Total != nil {
		flat = myStatsResponse{
			TotalProjects:      stats.Total.TotalProjects,
			TotalLogs:          stats.Total.TotalLogs,
			TotalPosts:         stats.Total.TotalPosts,
			TotalFollowers:     stats.Total.TotalFollowers,
			TotalFollowing:     stats.Total.TotalFollowing,
			TotalLikesReceived: stats.Total.TotalLikesReceived,
			TotalViews:         stats.Total.TotalViews,
		}
	}
	response.Success(c, flat)
}

// GetProjectStats 获取指定项目的统计数据（仅项目负责人可查看）
// GET /api/v1/projects/:id/stats?days=30
func (h *StatsHandler) GetProjectStats(c *gin.Context) {
	projectIDStr := c.Param("id")
	projectID, err := strconv.ParseUint(projectIDStr, 10, 64)
	if err != nil || projectID == 0 {
		response.FailBadRequest(c, "无效的项目 ID")
		return
	}

	requesterID, ok := requireLogin(c)
	if !ok {
		return
	}

	days, _ := strconv.Atoi(c.DefaultQuery("days", "30"))

	stats, err := h.statsSvc.GetProjectStats(c.Request.Context(), projectID, requesterID, days)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.Success(c, stats)
}
