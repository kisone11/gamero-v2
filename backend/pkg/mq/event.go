// Package mq 提供消息队列基础设施。
// 使用接口抽象，默认提供内存队列实现（开发/测试用），预留 Kafka 实现接口。
package mq

import "time"

// EventType 事件类型
type EventType string

const (
	// EventCommentCreated 评论创建（触发：@提及通知、作者通知）
	EventCommentCreated EventType = "comment.created"
	// EventPostLiked 帖子被点赞
	EventPostLiked EventType = "post.liked"
	// EventLogCommentCreated 日志评论创建（触发：@提及通知）
	EventLogCommentCreated EventType = "log.comment.created"
	// EventHotScoreRefresh 触发热度分数刷新
	EventHotScoreRefresh EventType = "hot_score.refresh"
	// EventRecruitExpiry 触发招募过期检查
	EventRecruitExpiry EventType = "recruit.expiry_check"
	// EventUserFollowed 用户被关注（触发：通知）
	EventUserFollowed EventType = "user.followed"
	// EventProjectUpdated 项目更新（触发：关注者 Feed）
	EventProjectUpdated EventType = "project.updated"
	// EventPostPublished 帖子发布（触发：作者粉丝 Feed 扩散）
	EventPostPublished EventType = "post.published"
	// EventDevLogPublished 开发日志发布（触发：关注者 Feed 扩散）
	EventDevLogPublished EventType = "devlog.published"
	// EventProjectMemberJoined 项目成员加入（触发：项目关注者通知）
	EventProjectMemberJoined EventType = "project.member_joined"
)

// Event 消息队列事件
type Event struct {
	ID        string    `json:"id"`
	Type      EventType `json:"type"`
	Payload   []byte    `json:"payload"`
	CreatedAt time.Time `json:"created_at"`
}

// CommentCreatedPayload 评论创建事件载荷
type CommentCreatedPayload struct {
	CommentID uint64 `json:"comment_id"`
	PostID    uint64 `json:"post_id"`
	LogID     uint64 `json:"log_id"`
	AuthorID  uint64 `json:"author_id"`
	Content   string `json:"content"`
}

// UserFollowedPayload 关注事件载荷
type UserFollowedPayload struct {
	FollowerID uint64 `json:"follower_id"`
	FolloweeID uint64 `json:"followee_id"`
}

// ProjectUpdatedPayload 项目更新载荷
type ProjectUpdatedPayload struct {
	ProjectID  uint64 `json:"project_id"`
	UpdateType string `json:"update_type"` // log/release
}

// DevLogPublishedPayload 开发日志发布事件载荷（用于关注者 Feed 扩散）
type DevLogPublishedPayload struct {
	LogID     uint64 `json:"log_id"`
	AuthorID  uint64 `json:"author_id"`
	ProjectID uint64 `json:"project_id"`
	Title     string `json:"title"`
}

// ProjectMemberJoinedPayload 项目成员加入事件载荷
type ProjectMemberJoinedPayload struct {
	ProjectID uint64 `json:"project_id"`
	UserID    uint64 `json:"user_id"`
	Role      string `json:"role"`
}

// PostPublishedPayload 帖子发布事件载荷（用于作者粉丝 Feed 扩散）
type PostPublishedPayload struct {
	PostID   uint64 `json:"post_id"`
	AuthorID uint64 `json:"author_id"`
}
