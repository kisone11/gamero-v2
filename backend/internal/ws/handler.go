package ws

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/gamero/gamero/pkg/auth"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"go.uber.org/zap"
)

// Handler WebSocket HTTP 处理器
type Handler struct {
	hub      *Hub
	logger   *zap.Logger
	upgrader websocket.Upgrader
}

// NewHandler 创建 WebSocket 处理器。
//
// allowOrigins 为允许建立 WebSocket 连接的来源域名列表（来自 config.Server.AllowOrigins）。
// 若列表为空或包含 "*"，则退回到允许所有来源（仅开发环境使用）。
// 生产环境请传入具体域名，如 ["https://gamero.example.com"]。
func NewHandler(hub *Hub, logger *zap.Logger, allowOrigins []string) *Handler {
	originSet := make(map[string]bool, len(allowOrigins))
	allowAll := len(allowOrigins) == 0
	for _, o := range allowOrigins {
		if o == "*" {
			allowAll = true
			break
		}
		originSet[strings.ToLower(strings.TrimRight(o, "/"))] = true
	}

	upgrader := websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin: func(r *http.Request) bool {
			if allowAll {
				return true
			}
			origin := strings.ToLower(strings.TrimRight(r.Header.Get("Origin"), "/"))
			// 若请求没有 Origin 头（非浏览器客户端），直接放行
			if origin == "" {
				return true
			}
			return originSet[origin]
		},
	}

	h := &Handler{
		hub:      hub,
		logger:   logger,
		upgrader: upgrader,
	}
	return h
}

// ServeWS 处理 WebSocket 升级请求
// GET /api/v1/ws?token=<access_token>
// 支持两种认证方式：
//  1. Query param: ?token=xxx（方便前端 WebSocket 连接，因为 WS 不支持自定义 Header）
//  2. Header: Authorization: Bearer xxx
func (h *Handler) ServeWS(c *gin.Context) {
	// 从 query param 或 header 获取 token
	tokenStr := c.Query("token")
	if tokenStr == "" {
		tokenStr = c.GetHeader("Authorization")
		if len(tokenStr) > 7 && tokenStr[:7] == "Bearer " {
			tokenStr = tokenStr[7:]
		}
	}
	if tokenStr == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"code": 4001, "msg": "token required"})
		return
	}

	// 解析 token
	claims, err := auth.Get().ParseAccessToken(tokenStr)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"code": 4001, "msg": "invalid token"})
		return
	}

	// 升级为 WebSocket 连接（CheckOrigin 由 upgrader 内部执行）
	conn, err := h.upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		h.logger.Error("ws: upgrade failed", zap.Error(err))
		return
	}

	userID := claims.UserID

	client := &Client{
		hub:    h.hub,
		userID: userID,
		conn:   conn,
		send:   make(chan []byte, 256),
	}
	h.hub.Register(client)

	// 发送欢迎消息
	type connectedPayload struct {
		UserID uint64 `json:"user_id"`
		Msg    string `json:"msg"`
	}
	welcome, _ := json.Marshal(Message{
		Type: "connected",
		Payload: connectedPayload{
			UserID: userID,
			Msg:    "connected",
		},
	})
	client.send <- welcome

	// 启动读写 goroutine
	go client.writePump()
	go client.readPump()
}
