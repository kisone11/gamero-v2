// Package handler 提供发现模块（搜索与推荐）的 HTTP 请求处理层。
// 本文件包含以下 API 的 Gin Handler 实现：
//   - 项目搜索 / 用户搜索 / 日志搜索 / 帖子搜索
//   - 热门项目、热门日志分页、热门帖子
//   - 个性化推荐帖子
package handler

import (
	"strconv"

	"github.com/gamero/gamero/internal/service"
	"github.com/gamero/gamero/pkg/middleware"
	"github.com/gamero/gamero/pkg/pagination"
	"github.com/gamero/gamero/pkg/response"
	"github.com/gin-gonic/gin"
)

// DiscoverHandler 发现模块所有 API 的 Handler 集合
type DiscoverHandler struct {
	svc service.DiscoverService
}

// NewDiscoverHandler 创建 DiscoverHandler 实例
func NewDiscoverHandler(svc service.DiscoverService) *DiscoverHandler {
	return &DiscoverHandler{svc: svc}
}

// ===========================
// 统一搜索 API
// ===========================

// unifiedSearchResponse 统一搜索响应结构
type unifiedSearchResponse struct {
	Projects *searchSection[service.DiscoverProjectItem] `json:"projects,omitempty"`
	Users    *searchSection[service.DiscoverUserItem]   `json:"users,omitempty"`
	Logs     *searchSection[service.DiscoverLogItem]    `json:"logs,omitempty"`
	Posts    *searchSection[service.DiscoverPostItem]   `json:"posts,omitempty"`
}

// searchSection 单一类型搜索结果分区
type searchSection[T any] struct {
	Total int64 `json:"total"`
	List  []*T  `json:"list"`
}

// UnifiedSearch 统一搜索接口，支持 type=all/projects/users/logs/posts
// GET /api/v1/search?q=关键词&type=all&page=1&page_size=20
func (h *DiscoverHandler) UnifiedSearch(c *gin.Context) {
	keyword := c.Query("q")
	if len([]rune(keyword)) > 100 {
		response.FailBadRequest(c, "搜索关键词过长（最多100字）")
		return
	}

	searchType := c.DefaultQuery("type", "all")
	pager := pagination.Parse(c)
	ctx := c.Request.Context()

	var resp unifiedSearchResponse

	switch searchType {
	case "projects":
		items, total, err := h.svc.SearchProjects(ctx, keyword, pager.Page, pager.PageSize)
		if err != nil {
			response.Fail(c, err)
			return
		}
		resp.Projects = &searchSection[service.DiscoverProjectItem]{Total: total, List: items}
	case "users":
		items, total, err := h.svc.SearchUsers(ctx, keyword, pager.Page, pager.PageSize)
		if err != nil {
			response.Fail(c, err)
			return
		}
		resp.Users = &searchSection[service.DiscoverUserItem]{Total: total, List: items}
	case "logs":
		logsUserID, _ := middleware.GetUserID(c)
		items, total, err := h.svc.SearchLogs(ctx, keyword, pager.Page, pager.PageSize, logsUserID)
		if err != nil {
			response.Fail(c, err)
			return
		}
		resp.Logs = &searchSection[service.DiscoverLogItem]{Total: total, List: items}
	case "posts":
		items, total, err := h.svc.SearchPosts(ctx, keyword, pager.Page, pager.PageSize)
		if err != nil {
			response.Fail(c, err)
			return
		}
		resp.Posts = &searchSection[service.DiscoverPostItem]{Total: total, List: items}
	default: // all
		// 并行查询四种类型（goroutine 并发，任一失败不阻断其他）
		userID, _ := middleware.GetUserID(c)
		type searchRes struct {
			name  string
			items interface{}
			total int64
			err   error
		}
		ch := make(chan searchRes, 4)

		go func() {
			items, total, err := h.svc.SearchProjects(ctx, keyword, pager.Page, pager.PageSize)
			ch <- searchRes{"projects", items, total, err}
		}()
		go func() {
			items, total, err := h.svc.SearchUsers(ctx, keyword, pager.Page, pager.PageSize)
			ch <- searchRes{"users", items, total, err}
		}()
		go func() {
			items, total, err := h.svc.SearchLogs(ctx, keyword, pager.Page, pager.PageSize, userID)
			ch <- searchRes{"logs", items, total, err}
		}()
		go func() {
			items, total, err := h.svc.SearchPosts(ctx, keyword, pager.Page, pager.PageSize)
			ch <- searchRes{"posts", items, total, err}
		}()

		for i := 0; i < 4; i++ {
			r := <-ch
			if r.err != nil {
				continue
			}
			switch r.name {
			case "projects":
				resp.Projects = &searchSection[service.DiscoverProjectItem]{Total: r.total, List: r.items.([]*service.DiscoverProjectItem)}
			case "users":
				resp.Users = &searchSection[service.DiscoverUserItem]{Total: r.total, List: r.items.([]*service.DiscoverUserItem)}
			case "logs":
				resp.Logs = &searchSection[service.DiscoverLogItem]{Total: r.total, List: r.items.([]*service.DiscoverLogItem)}
			case "posts":
				resp.Posts = &searchSection[service.DiscoverPostItem]{Total: r.total, List: r.items.([]*service.DiscoverPostItem)}
			}
		}
	}

	response.Success(c, resp)
}

// ===========================
// 搜索 API
// ===========================

// SearchProjects 搜索项目（支持分页）
// GET /api/v1/search/projects?q=关键词&page=1&page_size=20
// q 为空时返回热门项目列表
func (h *DiscoverHandler) SearchProjects(c *gin.Context) {
	keyword := c.Query("q")
	if len([]rune(keyword)) > 100 {
		response.FailBadRequest(c, "搜索关键词过长（最多100字）")
		return
	}

	pager := pagination.Parse(c)

	items, total, err := h.svc.SearchProjects(c.Request.Context(), keyword, pager.Page, pager.PageSize)
	if err != nil {
		response.Fail(c, err)
		return
	}

	response.SuccessPage(c, items, total, pager.Page, pager.PageSize)
}

// SearchUsers 搜索用户（支持分页）
// GET /api/v1/search/users?q=关键词&page=1&page_size=20
func (h *DiscoverHandler) SearchUsers(c *gin.Context) {
	keyword := c.Query("q")
	if keyword == "" {
		response.FailBadRequest(c, "搜索关键词不能为空（参数 q）")
		return
	}
	if len([]rune(keyword)) > 100 {
		response.FailBadRequest(c, "搜索关键词过长（最多100字）")
		return
	}

	pager := pagination.Parse(c)

	items, total, err := h.svc.SearchUsers(c.Request.Context(), keyword, pager.Page, pager.PageSize)
	if err != nil {
		response.Fail(c, err)
		return
	}

	response.SuccessPage(c, items, total, pager.Page, pager.PageSize)
}

// SearchLogs 搜索开发日志（支持分页，只搜索已发布公开日志）
// GET /api/v1/search/logs?q=关键词&page=1&page_size=20
// q 为空时返回热门日志列表
func (h *DiscoverHandler) SearchLogs(c *gin.Context) {
	keyword := c.Query("q")
	if len([]rune(keyword)) > 100 {
		response.FailBadRequest(c, "搜索关键词过长（最多100字）")
		return
	}

	pager := pagination.Parse(c)

	userID, _ := middleware.GetUserID(c)
		items, total, err := h.svc.SearchLogs(c.Request.Context(), keyword, pager.Page, pager.PageSize, userID)
	if err != nil {
		response.Fail(c, err)
		return
	}

	response.SuccessPage(c, items, total, pager.Page, pager.PageSize)
}

// SearchPosts 搜索社区帖子
// GET /api/v1/search/posts?q=keyword&page=1&page_size=20
// q 为空时返回热门帖子列表
func (h *DiscoverHandler) SearchPosts(c *gin.Context) {
	keyword := c.Query("q")
	if len([]rune(keyword)) > 100 {
		response.FailBadRequest(c, "搜索关键词过长（最多100字）")
		return
	}

	pager := pagination.Parse(c)

	items, total, err := h.svc.SearchPosts(c.Request.Context(), keyword, pager.Page, pager.PageSize)
	if err != nil {
		response.Fail(c, err)
		return
	}

	response.SuccessPage(c, items, total, pager.Page, pager.PageSize)
}

// ===========================
// 热门/最新内容 API（公开）
// ===========================

// GetHotProjects 热门项目：按关注数排序（前20条）
// GET /api/v1/discover/hot-projects?limit=20
func (h *DiscoverHandler) GetHotProjects(c *gin.Context) {
	limit := parseDiscoverLimit(c, 20, 100)

	userID, _ := middleware.GetUserID(c)
		items, err := h.svc.GetHotProjects(c.Request.Context(), limit, userID)
	if err != nil {
		response.Fail(c, err)
		return
	}

	response.Success(c, gin.H{
		"list":  items,
		"count": len(items),
	})
}

// ===========================
// 工具函数
// ===========================

// parseDiscoverLimit 从 query 参数 limit 解析上限值
// defaultVal: 默认值；maxVal: 最大值上限
func parseDiscoverLimit(c *gin.Context, defaultVal, maxVal int) int {
	limit := defaultVal
	if limitStr := c.Query("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
			limit = l
			if limit > maxVal {
				limit = maxVal
			}
		}
	}
	return limit
}

// GetHotLogsPaged 热门日志分页（公开接口）
// GET /api/v1/discover/hot-logs-paged?page=1&page_size=20&log_type=devlog
func (h *DiscoverHandler) GetHotLogsPaged(c *gin.Context) {
	pager := pagination.Parse(c)
	logType := c.Query("log_type")
	userID, _ := middleware.GetUserID(c)
		items, total, err := h.svc.GetHotLogsPaged(c.Request.Context(), pager.Page, pager.PageSize, logType, userID)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.SuccessPage(c, items, total, pager.Page, pager.PageSize)
}

// GetHotPosts 热门帖子分页（公开接口）
// GET /api/v1/discover/hot-posts?page=1&page_size=20
func (h *DiscoverHandler) GetHotPosts(c *gin.Context) {
	pager := pagination.Parse(c)
	items, total, err := h.svc.GetHotPosts(c.Request.Context(), pager.Page, pager.PageSize)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.SuccessPage(c, items, total, pager.Page, pager.PageSize)
}

// GetRecommendedPosts 个性化推荐帖子（需JWT，三级降级）
// GET /api/v1/discover/recommended-posts?page=1&page_size=20
func (h *DiscoverHandler) GetRecommendedPosts(c *gin.Context) {
	userID, ok := requireLogin(c)
	if !ok {
		return
	}
	pager := pagination.Parse(c)
	items, total, err := h.svc.GetRecommendedPosts(c.Request.Context(), userID, pager.Page, pager.PageSize)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.SuccessPage(c, items, total, pager.Page, pager.PageSize)
}
