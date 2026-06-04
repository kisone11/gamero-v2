// Package metrics 提供 Prometheus 指标定义和 Gin 中间件。
package metrics

import (
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// HTTPRequestsTotal HTTP 请求总数（by method, path, status）
	HTTPRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "gamero_http_requests_total",
			Help: "Total number of HTTP requests",
		},
		[]string{"method", "path", "status"},
	)

	// HTTPRequestDuration HTTP 请求延迟直方图
	HTTPRequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "gamero_http_request_duration_seconds",
			Help:    "HTTP request latency in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method", "path"},
	)

	// ActiveWSConnections 活跃 WebSocket 连接数（Gauge）
	ActiveWSConnections = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "gamero_ws_active_connections",
		Help: "Number of active WebSocket connections",
	})

	// DBQueriesTotal 数据库查询总数
	DBQueriesTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "gamero_db_queries_total",
			Help: "Total number of database queries",
		},
		[]string{"operation"}, // select/insert/update/delete
	)

	// CacheHitsTotal 缓存命中/未命中总数
	CacheHitsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "gamero_cache_operations_total",
			Help: "Total cache operations",
		},
		[]string{"result"}, // hit / miss
	)

	// MQPublishTotal 消息队列发布总数
	MQPublishTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "gamero_mq_publish_total",
			Help: "Total MQ publish operations",
		},
		[]string{"event_type", "result"}, // success / error
	)

	// FeedFanoutDuration Feed 写扩散耗时直方图
	FeedFanoutDuration = promauto.NewHistogram(prometheus.HistogramOpts{
		Name:    "gamero_feed_fanout_duration_seconds",
		Help:    "Duration of feed fanout operations",
		Buckets: []float64{0.001, 0.005, 0.01, 0.05, 0.1, 0.5, 1.0},
	})

	// ContentCreatedTotal 内容创建计数（by type: post/devlog/project/comment）
	ContentCreatedTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "gamero_content_created_total",
			Help: "Total number of content items created",
		},
		[]string{"type"},
	)

	// UserActionTotal 用户行为计数（like/collect/follow/reward）
	UserActionTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "gamero_user_action_total",
			Help: "Total number of user interactions",
		},
		[]string{"action"},
	)

	// SearchQueryTotal 搜索引擎调用计数
	SearchQueryTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "gamero_search_query_total",
			Help: "Total search queries by engine and type",
		},
		[]string{"engine", "type"},
	)

	// FeedMergeTotal 推拉结合读合并触发次数
	FeedMergeTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "gamero_feed_merge_total",
			Help: "Total feed merge operations",
		},
		[]string{"result"}, // heavy_merged / cache_miss / cache_hit
	)

	// WSMessageTotal WebSocket 推送消息计数
	WSMessageTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "gamero_ws_message_total",
			Help: "Total WebSocket messages pushed",
		},
		[]string{"type"},
	)

	// CrowdfundActiveCount 当前活跃众筹项目数
	CrowdfundActiveCount = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "gamero_crowdfund_active_total",
		Help: "Number of currently active crowdfunding campaigns",
	})
)

// PrometheusMiddleware Gin 中间件，记录每个请求的 HTTP 指标
func PrometheusMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		// 使用路由模式（如 /api/v1/users/:id），避免高基数
		path := c.FullPath()
		if path == "" {
			path = "unknown"
		}
		method := c.Request.Method

		c.Next()

		duration := time.Since(start).Seconds()
		status := strconv.Itoa(c.Writer.Status())

		HTTPRequestsTotal.WithLabelValues(method, path, status).Inc()
		HTTPRequestDuration.WithLabelValues(method, path).Observe(duration)
	}
}
