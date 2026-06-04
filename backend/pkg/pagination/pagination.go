// Package pagination 提供统一的分页参数解析和处理功能。
// 支持从 Gin 请求上下文中解析 page 和 page_size 参数，
// 并提供默认值和最大值限制。
package pagination

import (
	"strconv"

	"github.com/gin-gonic/gin"
)

const (
	// DefaultPage 默认页码
	DefaultPage = 1
	// DefaultPageSize 默认每页大小
	DefaultPageSize = 20
	// MaxPageSize 每页最大记录数
	MaxPageSize = 100
	// MinPageSize 每页最小记录数
	MinPageSize = 1
)

// Params 分页参数
type Params struct {
	Page     int `form:"page" json:"page"`           // 页码，从 1 开始
	PageSize int `form:"page_size" json:"page_size"` // 每页大小
}

// Offset 计算 SQL OFFSET 值
func (p *Params) Offset() int {
	return (p.Page - 1) * p.PageSize
}

// Limit 返回 SQL LIMIT 值
func (p *Params) Limit() int {
	return p.PageSize
}

// Validate 验证并修正分页参数
func (p *Params) Validate() {
	if p.Page < 1 {
		p.Page = DefaultPage
	}
	if p.PageSize < MinPageSize {
		p.PageSize = DefaultPageSize
	}
	if p.PageSize > MaxPageSize {
		p.PageSize = MaxPageSize
	}
}

// Parse 从 Gin 上下文中解析分页参数
// 自动设置默认值并限制最大值
func Parse(c *gin.Context) *Params {
	p := &Params{
		Page:     DefaultPage,
		PageSize: DefaultPageSize,
	}

	if pageStr := c.Query("page"); pageStr != "" {
		if page, err := strconv.Atoi(pageStr); err == nil {
			p.Page = page
		}
	}

	if pageSizeStr := c.Query("page_size"); pageSizeStr != "" {
		if pageSize, err := strconv.Atoi(pageSizeStr); err == nil {
			p.PageSize = pageSize
		}
	}

	p.Validate()
	return p
}

// ParseWithDefault 从 Gin 上下文中解析分页参数，支持自定义默认值
func ParseWithDefault(c *gin.Context, defaultPage, defaultPageSize int) *Params {
	p := &Params{
		Page:     defaultPage,
		PageSize: defaultPageSize,
	}

	if pageStr := c.Query("page"); pageStr != "" {
		if page, err := strconv.Atoi(pageStr); err == nil {
			p.Page = page
		}
	}

	if pageSizeStr := c.Query("page_size"); pageSizeStr != "" {
		if pageSize, err := strconv.Atoi(pageSizeStr); err == nil {
			p.PageSize = pageSize
		}
	}

	p.Validate()
	return p
}

// CalcPages 计算总页数
func CalcPages(total int64, pageSize int) int {
	if pageSize <= 0 {
		return 0
	}
	return int((total + int64(pageSize) - 1) / int64(pageSize))
}

// ParsePageSize 仅解析 page_size 参数，不解析 page，用于 cursor 分页场景。
// defaultSize 为默认 page_size，最大值为 MaxPageSize。
func ParsePageSize(c *gin.Context, defaultSize int) int {
	size := defaultSize
	if s := c.Query("page_size"); s != "" {
		if n, err := strconv.Atoi(s); err == nil {
			size = n
		}
	}
	if size < 1 {
		size = 1
	}
	if size > MaxPageSize {
		size = MaxPageSize
	}
	return size
}
