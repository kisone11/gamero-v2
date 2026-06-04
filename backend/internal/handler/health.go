// Package handler 提供 HTTP 请求处理器。
// health.go 实现健康检查接口，检查各组件连接状态。
package handler

import (
	"net/http"
	"time"

	"github.com/gamero/gamero/pkg/cache"
	"github.com/gamero/gamero/pkg/database"
	"github.com/gamero/gamero/pkg/storage"
	"github.com/gin-gonic/gin"
)

// componentStatus 组件状态
type componentStatus struct {
	Status  string `json:"status"`           // "ok" 或 "error"
	Message string `json:"message,omitempty"` // 错误时的详细信息
}

// healthResponse 健康检查响应体
type healthResponse struct {
	Status     string                     `json:"status"`     // 整体状态："ok" 或 "degraded"
	Timestamp  time.Time                  `json:"timestamp"`  // 检查时间
	Components map[string]componentStatus `json:"components"` // 各组件状态
}

// Health 健康检查处理器
// GET /health
// 检查服务本身、PostgreSQL、Redis、对象存储的连接状态
func Health(c *gin.Context) {
	resp := &healthResponse{
		Status:     "ok",
		Timestamp:  time.Now(),
		Components: make(map[string]componentStatus),
	}

	// 检查数据库
	if err := database.Ping(); err != nil {
		resp.Components["database"] = componentStatus{
			Status:  "error",
			Message: err.Error(),
		}
		resp.Status = "degraded"
	} else {
		resp.Components["database"] = componentStatus{Status: "ok"}
	}

	// 检查 Redis
	if err := cache.Ping(); err != nil {
		resp.Components["redis"] = componentStatus{
			Status:  "error",
			Message: err.Error(),
		}
		resp.Status = "degraded"
	} else {
		resp.Components["redis"] = componentStatus{Status: "ok"}
	}

	// 检查对象存储
	if err := storage.Ping(); err != nil {
		resp.Components["storage"] = componentStatus{
			Status:  "error",
			Message: err.Error(),
		}
		resp.Status = "degraded"
	} else {
		resp.Components["storage"] = componentStatus{Status: "ok"}
	}

	// 确定 HTTP 状态码
	httpStatus := http.StatusOK
	if resp.Status != "ok" {
		httpStatus = http.StatusServiceUnavailable
	}

	c.JSON(httpStatus, resp)
}
