// Package ws 提供 WebSocket 实时推送功能。
// 包含连接管理（Hub/Client）、消息推送和通知推送等能力。
// 多实例部署时，通过 Redis Pub/Sub 广播跨实例消息。
package ws

import (
	"context"
	"encoding/json"
	"sync"
	"time"

	"github.com/gamero/gamero/pkg/cache"
	"github.com/gamero/gamero/pkg/metrics"
	"github.com/gorilla/websocket"
	"go.uber.org/zap"
)

const (
	writeWait      = 10 * time.Second    // 写超时
	pongWait       = 60 * time.Second    // 等待 pong 超时
	pingPeriod     = (pongWait * 9) / 10 // 发送 ping 间隔
	maxMessageSize = 512                  // 最大消息大小（字节）
)

// Message WebSocket 推送消息
type Message struct {
	Type    string      `json:"type"`    // 消息类型，如 "notification"
	Payload interface{} `json:"payload"` // 消息内容
}

// Client 单个 WebSocket 连接
type Client struct {
	hub    *Hub
	userID uint64
	conn   *websocket.Conn
	send   chan []byte
}

// Hub WebSocket 连接管理中心
type Hub struct {
	// 按 userID 存储连接（一个用户可能多个 tab）
	clients map[uint64]map[*Client]bool
	mu      sync.RWMutex
	logger  *zap.Logger

	// Redis Pub/Sub channel（多实例广播）
	pubsubChannel string
}

// NewHub 创建 Hub 实例
func NewHub(logger *zap.Logger) *Hub {
	return &Hub{
		clients:       make(map[uint64]map[*Client]bool),
		logger:        logger,
		pubsubChannel: "ws:notifications",
	}
}

// pubsubPayload Redis Pub/Sub 消息格式
type pubsubPayload struct {
	UserID uint64          `json:"user_id"` // 目标用户 ID
	Msg    json.RawMessage `json:"msg"`     // 已序列化的 WebSocket 消息
}

// StartPubSub 启动 Redis Pub/Sub 订阅，接收其他实例发来的推送消息并转发到本地连接。
// 应在 goroutine 中运行，ctx 取消时退出。
func (h *Hub) StartPubSub(ctx context.Context) {
	rdb := cache.Get()
	if rdb == nil {
		h.logger.Warn("redis not available, skipping pubsub subscribe")
		return
	}
	sub := rdb.Subscribe(ctx, h.pubsubChannel)
	defer sub.Close()

	h.logger.Info("ws: pubsub subscriber started", zap.String("channel", h.pubsubChannel))

	ch := sub.Channel()
	for {
		select {
		case <-ctx.Done():
			h.logger.Info("ws: pubsub subscriber stopped")
			return
		case msg, ok := <-ch:
			if !ok {
				return
			}
			var p pubsubPayload
			if err := json.Unmarshal([]byte(msg.Payload), &p); err != nil {
				h.logger.Warn("ws: invalid pubsub payload", zap.Error(err))
				continue
			}
			// 将消息推送到本实例上 userID 对应的连接
			h.SendToUser(p.UserID, p.Msg)
		}
	}
}

// PublishToUser 通过 Redis Pub/Sub 向所有实例广播消息（包含本实例）。
// 调用方不需要关心目标用户连接在哪个实例，统一通过此方法广播。
func (h *Hub) PublishToUser(ctx context.Context, userID uint64, msg []byte) {
	p := pubsubPayload{UserID: userID, Msg: msg}
	data, err := json.Marshal(p)
	if err != nil {
		h.logger.Error("ws: marshal pubsub payload failed", zap.Error(err))
		return
	}
	rdb := cache.Get()
	if rdb == nil {
		// Redis not available, skip pubsub
		return
	}
	if err := rdb.Publish(ctx, h.pubsubChannel, data).Err(); err != nil {
		h.logger.Warn("ws: redis publish failed, falling back to local send",
			zap.Uint64("user_id", userID), zap.Error(err))
		// Redis 不可用时降级为本地推送
		h.SendToUser(userID, msg)
	}
}

// Register 注册客户端连接
func (h *Hub) Register(client *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if _, ok := h.clients[client.userID]; !ok {
		h.clients[client.userID] = make(map[*Client]bool)
	}
	h.clients[client.userID][client] = true
	metrics.ActiveWSConnections.Inc()
	h.logger.Info("ws: client connected", zap.Uint64("user_id", client.userID))
}

// Unregister 注销客户端连接
func (h *Hub) Unregister(client *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if conns, ok := h.clients[client.userID]; ok {
		if _, exists := conns[client]; exists {
			delete(conns, client)
			close(client.send)
			metrics.ActiveWSConnections.Dec()
			if len(conns) == 0 {
				delete(h.clients, client.userID)
			}
		}
	}
	h.logger.Info("ws: client disconnected", zap.Uint64("user_id", client.userID))
}

// SendToUser 向指定用户的所有连接推送消息
func (h *Hub) SendToUser(userID uint64, msg []byte) {
	// 在锁内拷贝 client 列表，避免 Unregister 并发修改 map 导致数据竞争
	h.mu.RLock()
	var clients []*Client
	for client := range h.clients[userID] {
		clients = append(clients, client)
	}
	h.mu.RUnlock()

	for _, client := range clients {
		h.safeSend(client, userID, msg)
	}
}

// safeSend 安全地向 client.send 写入消息。
// Unregister 会 close(client.send)，若在 SendToUser 读取 client 列表后、实际写入前被 close，
// 直接写入 closed channel 会 panic。用 recover 单纯捕获该场景。
func (h *Hub) safeSend(client *Client, userID uint64, msg []byte) {
	defer func() {
		if r := recover(); r != nil {
			// 连接已关闭，静默丢弃
			h.logger.Warn("ws: send on closed channel (client already disconnected)",
				zap.Uint64("user_id", userID))
		}
	}()
	select {
	case client.send <- msg:
	default:
		// 发送缓冲区满，跳过（不阻塞）
		h.logger.Warn("ws: send buffer full, dropping message", zap.Uint64("user_id", userID))
	}
}

// OnlineCount 返回当前在线用户数
func (h *Hub) OnlineCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.clients)
}

// IsOnline 判断用户是否在线
func (h *Hub) IsOnline(userID uint64) bool {
	h.mu.RLock()
	defer h.mu.RUnlock()
	conns, ok := h.clients[userID]
	return ok && len(conns) > 0
}

// writePump 将消息写入 WebSocket 连接（每个 Client 一个 goroutine）
func (c *Client) writePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()
	for {
		select {
		case message, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			w, err := c.conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			w.Write(message)
			// 批量发送队列中的消息
			n := len(c.send)
			for i := 0; i < n; i++ {
				w.Write([]byte("\n"))
				w.Write(<-c.send)
			}
			if err := w.Close(); err != nil {
				return
			}
		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// readPump 读取客户端消息（心跳响应、断线检测）
func (c *Client) readPump() {
	defer func() {
		c.hub.Unregister(c)
		c.conn.Close()
	}()
	c.conn.SetReadLimit(maxMessageSize)
	c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})
	for {
		_, _, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				c.hub.logger.Warn("ws: unexpected close", zap.Error(err))
			}
			break
		}
		// 客户端消息暂不处理（单向推送模式）
	}
}
