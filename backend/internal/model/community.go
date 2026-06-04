// Package model 定义社区系统相关的 GORM 数据模型。
// 本文件包含以下模型：
//   - Topic: 话题（帖子分类标签）
//   - Post: 社区帖子
//   - PostComment: 帖子评论（支持二级评论）
//   - PostLike: 帖子点赞
//   - PostCollect: 帖子收藏
//   - PostCommentLike: 帖子评论点赞
//   - Report: 内容举报
package model

import "time"

// Topic 话题表
// 由管理员创建，是帖子的分类标签。
// 对应数据库表 topics
type Topic struct {
	ID            uint64    `gorm:"primaryKey;autoIncrement" json:"id"`
	Name          string    `gorm:"type:varchar(64);not null;uniqueIndex" json:"name"`        // 话题名称，唯一
	Description   string    `gorm:"type:varchar(256);not null;default:''" json:"description"` // 话题描述
	IconKey       string    `gorm:"type:varchar(512);not null;default:''" json:"icon_key"`    // 图标 对象存储 key
	PostCount     int       `gorm:"not null;default:0" json:"post_count"`                     // 帖子数量（冗余）
	FollowerCount int       `gorm:"not null;default:0" json:"follower_count"`                 // 关注人数（冗余）
	IsDefault     bool      `gorm:"not null;default:false" json:"is_default"`                 // 是否为预建默认话题
	CreatedAt     time.Time `gorm:"not null;autoCreateTime" json:"created_at"`
	UpdatedAt     time.Time `gorm:"not null;autoUpdateTime" json:"updated_at"`
}

// TableName 指定表名
func (Topic) TableName() string {
	return "topics"
}

// Post 帖子表
// 用户发布的社区帖子，支持关联话题和项目。
// 对应数据库表 posts
type Post struct {
	ID           uint64     `gorm:"primaryKey;autoIncrement" json:"id"`
	AuthorID     uint64     `gorm:"not null;index" json:"author_id"`                      // 作者 ID，外键 users.id
	Title        string     `gorm:"type:varchar(128);not null" json:"title"`              // 帖子标题，最多 128 字符
	Content      string     `gorm:"type:text;not null" json:"content"`                    // 帖子正文（Markdown），最多 50000 字符
	TopicIDs     string     `gorm:"type:text;not null;default:'[]'" json:"topic_ids"`     // 关联话题 ID 列表，JSON 存储 []uint64，最多 3 个
	Images       string     `gorm:"type:text;not null;default:'[]'" json:"images"`        // 配图 对象存储 key 列表，JSON 存储 []string，最多 9 张
	Videos       string     `gorm:"type:text;not null;default:'[]'" json:"videos"`        // 视频 对象存储 key 列表，JSON 存储 []string，最多 3 个
	ProjectID    uint64     `gorm:"not null;default:0;index" json:"project_id"`           // 关联项目 ID，0 表示不关联
	IsPinned     bool       `gorm:"not null;default:false;index" json:"is_pinned"`        // 是否置顶（管理员操作）
	ViewCount    int        `gorm:"not null;default:0" json:"view_count"`                 // 浏览数（冗余）
	LikeCount    int        `gorm:"not null;default:0" json:"like_count"`                 // 点赞数（冗余）
	CommentCount int        `gorm:"not null;default:0" json:"comment_count"`              // 评论数（冗余）
	CollectCount int        `gorm:"not null;default:0" json:"collect_count"`              // 收藏数（冗余）
	HotScore     float64    `gorm:"type:float;not null;default:0;index" json:"hot_score"` // 热度分数（HackerNews 算法）
	CreatedAt    time.Time  `gorm:"not null;autoCreateTime" json:"created_at"`
	UpdatedAt    time.Time  `gorm:"not null;autoUpdateTime" json:"updated_at"`
	DeletedAt    *time.Time `gorm:"index" json:"deleted_at,omitempty"` // 软删除时间戳
}

// TableName 指定表名
func (Post) TableName() string {
	return "posts"
}

// PostComment 帖子评论表
// 支持两级评论：一级评论（reply_to_id=0）和子评论（reply_to_id 指向父评论 ID）
// 对应数据库表 post_comments
type PostComment struct {
	ID        uint64     `gorm:"primaryKey;autoIncrement" json:"id"`
	PostID    uint64     `gorm:"not null;index" json:"post_id"`              // 所属帖子 ID，外键 posts.id
	AuthorID  uint64     `gorm:"not null;index" json:"author_id"`            // 评论作者 ID，外键 users.id
	Content   string     `gorm:"type:varchar(1000);not null" json:"content"` // 评论内容，最多 1000 字符
	ReplyToID uint64     `gorm:"not null;default:0" json:"reply_to_id"`      // 回复的评论 ID，一级评论为 0
	LikeCount int        `gorm:"not null;default:0" json:"like_count"`       // 评论点赞数（冗余）
	CreatedAt time.Time  `gorm:"not null;autoCreateTime" json:"created_at"`
	UpdatedAt time.Time  `gorm:"not null;autoUpdateTime" json:"updated_at"`
	DeletedAt *time.Time `gorm:"index" json:"deleted_at,omitempty"` // 软删除时间戳
}

// TableName 指定表名
func (PostComment) TableName() string {
	return "post_comments"
}

// PostLike 帖子点赞表
// 唯一索引 (post_id, user_id) 防止重复点赞
// 对应数据库表 post_likes
type PostLike struct {
	ID        uint64    `gorm:"primaryKey;autoIncrement" json:"id"`
	PostID    uint64    `gorm:"not null;uniqueIndex:idx_post_user_like" json:"post_id"` // 帖子 ID，外键 posts.id
	UserID    uint64    `gorm:"not null;uniqueIndex:idx_post_user_like" json:"user_id"` // 用户 ID，外键 users.id
	CreatedAt time.Time `gorm:"not null;autoCreateTime" json:"created_at"`
}

// TableName 指定表名
func (PostLike) TableName() string {
	return "post_likes"
}

// PostCollect 帖子收藏表
// 唯一索引 (post_id, user_id) 防止重复收藏
// 对应数据库表 post_collects
type PostCollect struct {
	ID        uint64    `gorm:"primaryKey;autoIncrement" json:"id"`
	PostID    uint64    `gorm:"not null;uniqueIndex:idx_post_user_collect" json:"post_id"` // 帖子 ID，外键 posts.id
	UserID    uint64    `gorm:"not null;uniqueIndex:idx_post_user_collect" json:"user_id"` // 用户 ID，外键 users.id
	CreatedAt time.Time `gorm:"not null;autoCreateTime" json:"created_at"`
}

// TableName 指定表名
func (PostCollect) TableName() string {
	return "post_collects"
}

// PostCommentLike 帖子评论点赞表
// 唯一索引 (comment_id, user_id) 防止重复点赞
// 对应数据库表 post_comment_likes
type PostCommentLike struct {
	ID        uint64    `gorm:"primaryKey;autoIncrement" json:"id"`
	CommentID uint64    `gorm:"not null;uniqueIndex:idx_post_comment_user_like" json:"comment_id"` // 评论 ID，外键 post_comments.id
	UserID    uint64    `gorm:"not null;uniqueIndex:idx_post_comment_user_like" json:"user_id"`    // 用户 ID，外键 users.id
	CreatedAt time.Time `gorm:"not null;autoCreateTime" json:"created_at"`
}

// TableName 指定表名
func (PostCommentLike) TableName() string {
	return "post_comment_likes"
}

// Report 内容举报表
// 唯一索引 (reporter_id, target_type, target_id) 防止重复举报同一内容
// 对应数据库表 reports
type Report struct {
	ID         uint64    `gorm:"primaryKey;autoIncrement" json:"id"`
	ReporterID uint64    `gorm:"not null;index;uniqueIndex:idx_report_unique" json:"reporter_id"`            // 举报人 ID，外键 users.id
	TargetType string    `gorm:"type:varchar(16);not null;uniqueIndex:idx_report_unique" json:"target_type"` // 目标类型：post / comment
	TargetID   uint64    `gorm:"not null;uniqueIndex:idx_report_unique" json:"target_id"`                    // 目标 ID（帖子 ID 或评论 ID）
	Reason     string    `gorm:"type:varchar(32);not null" json:"reason"`                                    // 举报原因：porn/spam/abuse/violation/other
	Supplement string    `gorm:"type:varchar(200);not null;default:''" json:"supplement"`                    // 补充说明，最多 200 字
	Status     string    `gorm:"type:varchar(16);not null;default:'pending'" json:"status"`                  // 举报状态：pending/escalated/handled/rejected
	AdminNote  string    `gorm:"type:varchar(500);not null;default:''" json:"admin_note,omitempty"`          // 管理员处理备注
	CreatedAt  time.Time `gorm:"not null;autoCreateTime" json:"created_at"`
	UpdatedAt  time.Time `gorm:"not null;autoUpdateTime" json:"updated_at"`
}

// TableName 指定表名
func (Report) TableName() string {
	return "reports"
}

// TopicFollow 话题关注表
// 记录用户关注话题的关系，唯一约束 (topic_id, user_id) 防止重复关注。
// 对应数据库表 topic_follows
type TopicFollow struct {
	ID        uint64    `gorm:"primaryKey;autoIncrement" json:"id"`
	TopicID   uint64    `gorm:"not null;uniqueIndex:idx_topic_follow_pair;index" json:"topic_id"` // 话题 ID，外键 topics.id
	UserID    uint64    `gorm:"not null;uniqueIndex:idx_topic_follow_pair;index" json:"user_id"`  // 用户 ID，外键 users.id
	CreatedAt time.Time `gorm:"not null;autoCreateTime" json:"created_at"`
}

// TableName 指定表名
func (TopicFollow) TableName() string {
	return "topic_follows"
}
