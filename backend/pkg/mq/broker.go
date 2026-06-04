package mq

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"
	"go.uber.org/zap"
)

// Handler 消息处理函数
type Handler func(ctx context.Context, event Event) error

// Broker 消息队列接口
type Broker interface {
	// Publish 发布事件
	Publish(ctx context.Context, event Event) error
	// Subscribe 订阅事件类型，handler 在单独 goroutine 中调用
	Subscribe(eventType EventType, handler Handler)
	// Start 启动消费循环
	Start(ctx context.Context)
	// Close 关闭 Broker
	Close() error
}

// NewEvent 创建新事件（自动填充 ID 和时间）
func NewEvent(eventType EventType, payload interface{}) (Event, error) {
	data, err := json.Marshal(payload)
	if err != nil {
		return Event{}, err
	}
	return Event{
		ID:        uuid.New().String(),
		Type:      eventType,
		Payload:   data,
		CreatedAt: time.Now(),
	}, nil
}

// ===========================================================================
// MemoryBroker — 开发/测试用内存队列
// ===========================================================================

// MemoryBroker 内存消息队列（开发/测试用）
// 生产环境替换为 KafkaBroker
type MemoryBroker struct {
	handlers map[EventType][]Handler
	ch       chan Event
	mu       sync.RWMutex
	logger   *zap.Logger
}

// NewMemoryBroker 创建内存 Broker
func NewMemoryBroker(bufSize int, logger *zap.Logger) *MemoryBroker {
	if bufSize <= 0 {
		bufSize = 1000
	}
	return &MemoryBroker{
		handlers: make(map[EventType][]Handler),
		ch:       make(chan Event, bufSize),
		logger:   logger,
	}
}

// Publish 发布事件到内存队列
func (b *MemoryBroker) Publish(ctx context.Context, event Event) error {
	select {
	case b.ch <- event:
		return nil
	default:
		// 队列满时记录日志，不阻塞
		b.logger.Warn("mq: channel full, dropping event", zap.String("type", string(event.Type)))
		return nil
	}
}

// Subscribe 注册事件处理器
func (b *MemoryBroker) Subscribe(eventType EventType, handler Handler) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.handlers[eventType] = append(b.handlers[eventType], handler)
}

// Start 启动消费循环（非阻塞，在 goroutine 中运行）
func (b *MemoryBroker) Start(ctx context.Context) {
	go func() {
		for {
			select {
			case event := <-b.ch:
				b.dispatch(ctx, event)
			case <-ctx.Done():
				return
			}
		}
	}()
}

// dispatch 将事件分发给所有订阅 handler
func (b *MemoryBroker) dispatch(ctx context.Context, event Event) {
	b.mu.RLock()
	handlers := b.handlers[event.Type]
	b.mu.RUnlock()
	for _, h := range handlers {
		h := h
		go func() {
			defer func() {
				if r := recover(); r != nil {
					b.logger.Error("mq: handler panic recovered",
						zap.String("type", string(event.Type)),
						zap.Any("panic", r))
				}
			}()
			if err := h(ctx, event); err != nil {
				b.logger.Error("mq: handler error",
					zap.String("type", string(event.Type)),
					zap.Error(err))
			}
		}()
	}
}

// Close 关闭 Broker
func (b *MemoryBroker) Close() error {
	close(b.ch)
	return nil
}

// ===========================================================================
// KafkaBroker — 生产级 Kafka 实现
// 依赖：github.com/segmentio/kafka-go v0.4.x
// 运行 `go mod tidy` 自动拉取依赖。
// ===========================================================================

// KafkaBrokerConfig KafkaBroker 初始化配置
type KafkaBrokerConfig struct {
	// Brokers Kafka 地址列表，如 ["localhost:9092"]
	Brokers []string
	// GroupID 消费者组 ID（同一组内消息只被消费一次）
	GroupID string
	// TopicPrefix 所有 Topic 的公共前缀，如 "gamero."
	// 最终 Topic 名 = TopicPrefix + EventType，如 "gamero.comment.created"
	TopicPrefix string
	// BatchSize 生产者批量提交大小，0 使用 kafka-go 默认值
	BatchSize int
	// BatchTimeout 生产者批量提交超时
	BatchTimeout time.Duration
	// MinBytes 消费者单次 Fetch 最小字节数
	MinBytes int
	// MaxBytes 消费者单次 Fetch 最大字节数
	MaxBytes int
	// MaxWait 消费者单次 Fetch 最长等待时间
	MaxWait time.Duration
	// CommitInterval 消费者自动提交 offset 间隔（0 = 同步提交）
	CommitInterval time.Duration
	// Logger zap 日志实例
	Logger *zap.Logger
}

// KafkaBroker 生产级 Kafka 消息队列
// - 每个 EventType 对应一个 Kafka Topic（= TopicPrefix + EventType）
// - 生产者：单个共享 kafka.Writer（async batch）
// - 消费者：每个 Topic 独立 kafka.Reader goroutine
type KafkaBroker struct {
	cfg      KafkaBrokerConfig
	writer   *kafka.Writer
	handlers map[EventType][]Handler
	mu       sync.RWMutex
	readers  []*kafka.Reader // 已启动的 reader，用于 Close
	logger   *zap.Logger
}

// NewKafkaBroker 创建 Kafka Broker
// brokers: Kafka 地址列表；logger: zap 实例
// 使用默认配置（GroupID="gamero", TopicPrefix="gamero."）
func NewKafkaBroker(brokers []string, logger *zap.Logger) *KafkaBroker {
	return NewKafkaBrokerWithConfig(KafkaBrokerConfig{
		Brokers:        brokers,
		GroupID:        "gamero",
		TopicPrefix:    "gamero.",
		BatchSize:      100,
		BatchTimeout:   100 * time.Millisecond,
		MinBytes:       1,
		MaxBytes:       10 << 20, // 10 MB
		MaxWait:        500 * time.Millisecond,
		CommitInterval: time.Second,
		Logger:         logger,
	})
}

// NewKafkaBrokerWithConfig 使用完整配置创建 Kafka Broker
func NewKafkaBrokerWithConfig(cfg KafkaBrokerConfig) *KafkaBroker {
	if cfg.GroupID == "" {
		cfg.GroupID = "gamero"
	}
	if cfg.TopicPrefix == "" {
		cfg.TopicPrefix = "gamero."
	}
	if cfg.BatchSize <= 0 {
		cfg.BatchSize = 100
	}
	if cfg.BatchTimeout <= 0 {
		cfg.BatchTimeout = 100 * time.Millisecond
	}
	if cfg.MinBytes <= 0 {
		cfg.MinBytes = 1
	}
	if cfg.MaxBytes <= 0 {
		cfg.MaxBytes = 10 << 20
	}
	if cfg.MaxWait <= 0 {
		cfg.MaxWait = 500 * time.Millisecond
	}
	if cfg.CommitInterval <= 0 {
		cfg.CommitInterval = time.Second
	}

	w := &kafka.Writer{
		Addr:         kafka.TCP(cfg.Brokers...),
		Balancer:     &kafka.LeastBytes{},
		BatchSize:    cfg.BatchSize,
		BatchTimeout: cfg.BatchTimeout,
		// 异步写入：失败会被 Completion 回调记录
		Async:        true,
		Completion: func(messages []kafka.Message, err error) {
			if err != nil && cfg.Logger != nil {
				cfg.Logger.Error("kafka: produce error", zap.Error(err))
			}
		},
	}

	return &KafkaBroker{
		cfg:      cfg,
		writer:   w,
		handlers: make(map[EventType][]Handler),
		logger:   cfg.Logger,
	}
}

// topic 将 EventType 映射为 Kafka Topic 名称
func (k *KafkaBroker) topic(et EventType) string {
	return k.cfg.TopicPrefix + string(et)
}

// Publish 将事件序列化后发布到对应 Kafka Topic
// 使用异步 Writer，失败由 Completion 回调记录日志
func (k *KafkaBroker) Publish(ctx context.Context, event Event) error {
	value, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("kafka: marshal event failed: %w", err)
	}
	msg := kafka.Message{
		Topic: k.topic(event.Type),
		Key:   []byte(event.ID), // 使用事件 ID 作为 key，保证同 key 有序
		Value: value,
	}
	if err := k.writer.WriteMessages(ctx, msg); err != nil {
		return fmt.Errorf("kafka: write message failed: %w", err)
	}
	return nil
}

// Subscribe 注册事件处理器
// 注意：必须在 Start() 之前调用，Start() 会为每个已订阅的 Topic 启动 Reader goroutine
func (k *KafkaBroker) Subscribe(eventType EventType, handler Handler) {
	k.mu.Lock()
	defer k.mu.Unlock()
	k.handlers[eventType] = append(k.handlers[eventType], handler)
}

// Start 为每个已订阅的 EventType 启动独立的 Kafka Reader goroutine
// 每个 Topic 只创建一个 Reader（属于同一消费者组），Handler 并发执行
func (k *KafkaBroker) Start(ctx context.Context) {
	k.mu.RLock()
	topics := make([]EventType, 0, len(k.handlers))
	for et := range k.handlers {
		topics = append(topics, et)
	}
	k.mu.RUnlock()

	for _, et := range topics {
		et := et
		r := kafka.NewReader(kafka.ReaderConfig{
			Brokers:        k.cfg.Brokers,
			GroupID:        k.cfg.GroupID,
			Topic:          k.topic(et),
			MinBytes:       k.cfg.MinBytes,
			MaxBytes:       k.cfg.MaxBytes,
			MaxWait:        k.cfg.MaxWait,
			CommitInterval: k.cfg.CommitInterval,
			// 消费者组模式下从最早未消费 offset 开始（新组首次消费 latest）
			StartOffset: kafka.LastOffset,
		})

		k.mu.Lock()
		k.readers = append(k.readers, r)
		k.mu.Unlock()

		go k.consume(ctx, et, r)
	}
}

// consume 持续从 Kafka Reader 拉取消息并分发 handler
func (k *KafkaBroker) consume(ctx context.Context, et EventType, r *kafka.Reader) {
	for {
		m, err := r.ReadMessage(ctx)
		if err != nil {
			// ctx 取消时退出，其他错误记录后继续重试
			if ctx.Err() != nil {
				return
			}
			k.logger.Error("kafka: read message error",
				zap.String("topic", k.topic(et)),
				zap.Error(err),
			)
			// 简单退避，避免死循环刷日志
			select {
			case <-ctx.Done():
				return
			case <-time.After(time.Second):
			}
			continue
		}

		var event Event
		if err := json.Unmarshal(m.Value, &event); err != nil {
			k.logger.Warn("kafka: unmarshal event failed",
				zap.String("topic", k.topic(et)),
				zap.Error(err),
			)
			continue
		}

		k.dispatch(ctx, event)
	}
}

// dispatch 将事件并发分发给所有已注册 handler
func (k *KafkaBroker) dispatch(ctx context.Context, event Event) {
	k.mu.RLock()
	handlers := k.handlers[event.Type]
	k.mu.RUnlock()

	for _, h := range handlers {
		h := h
		go func() {
			defer func() {
				if r := recover(); r != nil {
					k.logger.Error("kafka: handler panic recovered",
						zap.String("type", string(event.Type)),
						zap.Any("panic", r))
				}
			}()
			if err := h(ctx, event); err != nil {
				k.logger.Error("kafka: handler error",
					zap.String("type", string(event.Type)),
					zap.Error(err),
				)
			}
		}()
	}
}

// Close 关闭所有 Reader 和 Writer
func (k *KafkaBroker) Close() error {
	var errs []error

	// 关闭所有消费者 Reader
	k.mu.Lock()
	readers := k.readers
	k.mu.Unlock()
	for _, r := range readers {
		if err := r.Close(); err != nil {
			errs = append(errs, err)
		}
	}

	// 关闭生产者 Writer（会 flush 剩余 batch）
	if err := k.writer.Close(); err != nil {
		errs = append(errs, err)
	}

	if len(errs) > 0 {
		return fmt.Errorf("kafka broker close errors: %v", errs)
	}
	return nil
}
