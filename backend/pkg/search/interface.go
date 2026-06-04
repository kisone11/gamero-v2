package search

import "context"

// ProjectDoc ES 项目文档
type ProjectDoc struct {
	ID          uint64   `json:"id"`
	Name        string   `json:"name"`
	Slug        string   `json:"slug"`
	Description string   `json:"description"`
	Genre       string   `json:"genre"`
	Tags        []string `json:"tags"`
	Status      string   `json:"status"`
	CreatedAt   int64    `json:"created_at"`
}

// UserDoc ES 用户文档
type UserDoc struct {
	ID        uint64 `json:"id"`
	Username  string `json:"username"`
	Bio       string `json:"bio"`
	CreatedAt int64  `json:"created_at"`
}

// LogDoc ES 日志文档
type LogDoc struct {
	ID        uint64 `json:"id"`
	Title     string `json:"title"`
	Content   string `json:"content"`
	AuthorID  uint64 `json:"author_id"`
	ProjectID uint64 `json:"project_id"`
	CreatedAt int64  `json:"created_at"`
}

// PostDoc ES 帖子文档
type PostDoc struct {
	ID        uint64 `json:"id"`
	Title     string `json:"title"`
	Content   string `json:"content"`
	AuthorID  uint64 `json:"author_id"`
	CreatedAt int64  `json:"created_at"`
}

// Hit 搜索结果单条
type Hit struct {
	ID    uint64  `json:"id"`
	Score float64 `json:"score"`
}

// Result 搜索结果
type Result struct {
	Hits  []Hit `json:"hits"`
	Total int64 `json:"total"`
}

// Searcher 搜索引擎接口（PostgreSQL FTS 和 ES 均实现此接口）
type Searcher interface {
	// IndexProject 索引/更新项目文档
	IndexProject(ctx context.Context, doc ProjectDoc) error
	// IndexUser 索引/更新用户文档
	IndexUser(ctx context.Context, doc UserDoc) error
	// IndexLog 索引/更新日志文档
	IndexLog(ctx context.Context, doc LogDoc) error
	// IndexPost 索引/更新帖子文档
	IndexPost(ctx context.Context, doc PostDoc) error

	// SearchProjects 搜索项目，返回匹配的 ID 列表和总数
	SearchProjects(ctx context.Context, keyword string, offset, limit int) (Result, error)
	// SearchUsers 搜索用户
	SearchUsers(ctx context.Context, keyword string, offset, limit int) (Result, error)
	// SearchLogs 搜索日志
	SearchLogs(ctx context.Context, keyword string, offset, limit int) (Result, error)
	// SearchPosts 搜索帖子
	SearchPosts(ctx context.Context, keyword string, offset, limit int) (Result, error)

	// DeleteProject 删除项目索引
	DeleteProject(ctx context.Context, id uint64) error
	// DeleteUser 删除用户索引
	DeleteUser(ctx context.Context, id uint64) error
	// DeleteLog 删除日志索引
	DeleteLog(ctx context.Context, id uint64) error
	// DeletePost 删除帖子索引
	DeletePost(ctx context.Context, id uint64) error

	// Ping 健康检查
	Ping(ctx context.Context) error
}
