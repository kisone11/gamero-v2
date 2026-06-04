package mq

import (
	"context"
	"encoding/json"
)

// SubscriberDeps 订阅处理器依赖（通过函数类型注入，避免循环 import）
type SubscriberDeps struct {
	// SendMentionNotifications 解析 content 中的 @mention，发送通知
	SendMentionNotifications func(ctx context.Context, content string, authorID uint64, entityType string, entityID uint64)
	// NotifyLogAuthor 通知日志作者有新评论
	NotifyLogAuthor func(ctx context.Context, logID uint64, commenterID uint64, content string)
	// FanoutProjectUpdate 将项目更新推入关注者 Feed ZSet（写扩散）
	// projectID: 项目 ID，updateType: "log"/"release"，score: 发布时间戳
	FanoutProjectUpdate func(ctx context.Context, projectID uint64, updateType string, score float64)
	// FanoutUserPost 将帖子推入作者粉丝的 Feed ZSet（写扩散）
	// authorID: 作者 ID，postID: 帖子 ID，score: 发布时间戳
	FanoutUserPost func(ctx context.Context, authorID uint64, postID uint64, score float64)
}

// RegisterSubscribers 注册所有业务事件订阅。
// 在 router.Setup 中初始化 Broker 后调用。
func RegisterSubscribers(b Broker, deps SubscriberDeps) {
	// 评论创建 → @提及通知
	b.Subscribe(EventCommentCreated, newCommentMentionHandler(deps))
	// 评论创建 → 作者通知（日志评论通知日志作者）
	b.Subscribe(EventCommentCreated, newCommentAuthorNotifyHandler(deps))
	// 项目更新（日志/版本）→ 关注者 Feed 扩散
	b.Subscribe(EventProjectUpdated, newProjectFeedHandler(deps))
	// 里程碑完成 → 关注者 Feed 扩散
	// 帖子发布 → 作者粉丝 Feed 扩散
	b.Subscribe(EventPostPublished, newPostPublishedFeedHandler(deps))
}

// newCommentMentionHandler 处理评论中的 @提及通知
func newCommentMentionHandler(deps SubscriberDeps) Handler {
	return func(ctx context.Context, event Event) error {
		var p CommentCreatedPayload
		if err := json.Unmarshal(event.Payload, &p); err != nil {
			return err
		}
		if deps.SendMentionNotifications == nil {
			return nil
		}
		entityType := "post"
		entityID := p.PostID
		if p.LogID > 0 {
			entityType = "log"
			entityID = p.LogID
		}
		deps.SendMentionNotifications(ctx, p.Content, p.AuthorID, entityType, entityID)
		return nil
	}
}

// newCommentAuthorNotifyHandler 日志评论时通知日志作者
func newCommentAuthorNotifyHandler(deps SubscriberDeps) Handler {
	return func(ctx context.Context, event Event) error {
		var p CommentCreatedPayload
		if err := json.Unmarshal(event.Payload, &p); err != nil {
			return err
		}
		if p.LogID > 0 && deps.NotifyLogAuthor != nil {
			deps.NotifyLogAuthor(ctx, p.LogID, p.AuthorID, p.Content)
		}
		return nil
	}
}

// newProjectFeedHandler 项目更新（日志/版本）Feed 扩散
func newProjectFeedHandler(deps SubscriberDeps) Handler {
	return func(ctx context.Context, event Event) error {
		if deps.FanoutProjectUpdate == nil {
			return nil
		}
		var p ProjectUpdatedPayload
		if err := json.Unmarshal(event.Payload, &p); err != nil {
			return err
		}
		score := float64(event.CreatedAt.Unix())
		deps.FanoutProjectUpdate(ctx, p.ProjectID, p.UpdateType, score)
		return nil
	}
}


// newPostPublishedFeedHandler 帖子发布 → 作者粉丝 Feed 扩散
func newPostPublishedFeedHandler(deps SubscriberDeps) Handler {
	return func(ctx context.Context, event Event) error {
		if deps.FanoutUserPost == nil {
			return nil
		}
		var p PostPublishedPayload
		if err := json.Unmarshal(event.Payload, &p); err != nil {
			return err
		}
		score := float64(event.CreatedAt.Unix())
		deps.FanoutUserPost(ctx, p.AuthorID, p.PostID, score)
		return nil
	}
}


