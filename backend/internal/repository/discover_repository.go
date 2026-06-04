// Package repository 提供发现模块（搜索与推荐）的数据库访问层。
// 封装全局搜索、推荐系统、标签聚合、热门内容等所有查询操作。
package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"strings"

	"github.com/gamero/gamero/internal/model"
	"gorm.io/gorm"
)


// DiscoverRepository 发现模块仓储接口
// 包含搜索、热门内容、推广展示等所有数据访问方法
type DiscoverRepository interface {
	// ===========================
	// 搜索相关方法
	// ===========================

	// SearchProjects 按关键词搜索项目（ILIKE 匹配 name 或 description），分页
	SearchProjects(ctx context.Context, keyword string, offset, limit int) ([]*model.Project, int64, error)

	// SearchUsers 按关键词搜索用户（ILIKE 匹配 username 或 nickname），分页
	SearchUsers(ctx context.Context, keyword string, offset, limit int) ([]*model.User, int64, error)

	// SearchLogs 按关键词搜索开发日志（ILIKE 匹配 title），
	// 只搜索 status=published 且 visibility=public 的日志，分页
	SearchLogs(ctx context.Context, keyword string, offset, limit int) ([]*model.DevLog, int64, error)

	// SearchPosts 按关键词搜索社区帖子（title+content 全文搜索），分页
	SearchPosts(ctx context.Context, keyword string, offset, limit int) ([]*model.Post, int64, error)

	// ===========================
	// 热门内容方法
	// ===========================

	// GetProjectsByFollowerCount 按关注数降序获取项目列表（用于热门项目），分页
	GetProjectsByFollowerCount(ctx context.Context, offset, limit int) ([]*model.Project, int64, error)

	// GetTopLogs 按点赞数降序获取已发布公开日志（热门日志），分页
	// logType 非空时仅返回对应类型的日志
	GetTopLogs(ctx context.Context, offset, limit int, logType string) ([]*model.DevLog, int64, error)

	// ===========================
	// 批量查询方法（ES 搜索后回查 DB）
	// ===========================

	// GetProjectsByIDs 按 ID 列表批量获取项目（ES 搜索后回查 DB）
	GetProjectsByIDs(ctx context.Context, ids []uint64) ([]*model.Project, error)
	// GetUsersByIDs 按 ID 列表批量获取用户
	GetUsersByIDs(ctx context.Context, ids []uint64) ([]*model.User, error)
	// GetLogsByIDs 按 ID 列表批量获取日志
	GetLogsByIDs(ctx context.Context, ids []uint64) ([]*model.DevLog, error)

	// ===========================
	// 帖子相关方法
	// ===========================

	// GetTopPosts 按热度排序的帖子（like×2+comment+collect），分页
	GetTopPosts(ctx context.Context, offset, limit int) ([]*model.Post, int64, error)

	// GetDiscoverPostsByIDs 按 ID 列表批量获取帖子（过滤软删除）
	GetDiscoverPostsByIDs(ctx context.Context, ids []uint64) ([]*model.Post, error)

	// GetHotPostIDs 获取热度最高的帖子 ID（兜底推荐）
	GetHotPostIDs(ctx context.Context, excludeIDs []uint64, limit int) ([]uint64, error)

	// GetCoInteractedPostIDs 共现查询：与 postIDs 被同一用户点赞的其他帖子 ID
	GetCoInteractedPostIDs(ctx context.Context, postIDs []uint64, excludeIDs []uint64, limit int) ([]uint64, error)

	// GetUserPreferredTopicIDs 从用户帖子推断偏好话题 ID（L3 降级用）
	GetUserPreferredTopicIDs(ctx context.Context, userID uint64, limit int) ([]uint64, error)

	// GetPostsByTopicIDs 根据话题 ID 列表获取帖子（L4 降级）
	GetPostsByTopicIDs(ctx context.Context, topicIDs []uint64, excludeIDs []uint64, offset, limit int) ([]*model.Post, int64, error)

	// ===========================
	// 日志相关方法
	// ===========================

	// GetLogsByIDsFiltered 按 ID 批量获取日志（过滤软删除 + is_banned 项目 + 仅公开已发布）
	GetLogsByIDsFiltered(ctx context.Context, ids []uint64) ([]*model.DevLog, error)

	// ===========================

	// ===========================
	// 多维行为信号（帖子推荐用）
	// ===========================

	// GetUserInteractedPostIDs 聚合用户对帖子的所有互动（点赞+收藏+评论）
	GetUserInteractedPostIDs(ctx context.Context, userID uint64, limit int) ([]uint64, error)

	// GetSocialFeedPostIDs 社交 CF：获取用户关注的人近期互动过的帖子 ID
	GetSocialFeedPostIDs(ctx context.Context, userID uint64, excludeIDs []uint64, limit int) ([]uint64, error)

	// GetHotPostsByTopics 按话题分层热门帖子
	GetHotPostsByTopics(ctx context.Context, topicIDs []uint64, excludeIDs []uint64, limit int) ([]uint64, error)
}

// discoverRepository DiscoverRepository 接口的具体实现
type discoverRepository struct {
	db *gorm.DB
}

// NewDiscoverRepository 创建 discoverRepository 实例
func NewDiscoverRepository(db *gorm.DB) DiscoverRepository {
	return &discoverRepository{db: db}
}

// ===========================
// 搜索实现
// ===========================

// SearchProjects 按关键词搜索项目（PostgreSQL 全文搜索匹配 name 或 description），分页
// 使用 simple 字典支持中文，按 ts_rank 相关性降序排列
func (r *discoverRepository) SearchProjects(ctx context.Context, keyword string, offset, limit int) ([]*model.Project, int64, error) {
	ftsQuery := sanitizeFTSQuery(keyword)
	if ftsQuery == "" {
		return []*model.Project{}, 0, nil
	}

	ftsCondition := "to_tsvector('simple', coalesce(name,'') || ' ' || coalesce(description,'')) @@ to_tsquery('simple', ?)"

	var total int64
	// 先统计总数
	if err := r.db.WithContext(ctx).
		Model(&model.Project{}).
		Where("deleted_at IS NULL AND is_banned = false").
		Where(ftsCondition, ftsQuery).
		Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("搜索项目计数失败: %w", err)
	}

	if total == 0 {
		return []*model.Project{}, 0, nil
	}

	var projects []*model.Project
	// 按 ts_rank 相关性降序排列，相关性相同时按关注数降序
	if err := r.db.WithContext(ctx).
		Where("deleted_at IS NULL AND is_banned = false").
		Where(ftsCondition, ftsQuery).
		Order(gorm.Expr(
			"ts_rank(to_tsvector('simple', coalesce(name,'') || ' ' || coalesce(description,'')), to_tsquery('simple', ?)) DESC, follower_count DESC, created_at DESC",
			ftsQuery,
		)).
		Offset(offset).
		Limit(limit).
		Find(&projects).Error; err != nil {
		return nil, 0, fmt.Errorf("搜索项目失败: %w", err)
	}

	return projects, total, nil
}

// SearchUsers 按关键词搜索用户（PostgreSQL 全文搜索匹配 username 或 bio），分页
// 使用 simple 字典支持中文，按 ts_rank 相关性降序排列
func (r *discoverRepository) SearchUsers(ctx context.Context, keyword string, offset, limit int) ([]*model.User, int64, error) {
	ftsQuery := sanitizeFTSQuery(keyword)
	if ftsQuery == "" {
		return []*model.User{}, 0, nil
	}

	ftsCondition := "to_tsvector('simple', coalesce(username,'') || ' ' || coalesce(bio,'')) @@ to_tsquery('simple', ?)"

	var total int64
	if err := r.db.WithContext(ctx).
		Model(&model.User{}).
		Where("status = ?", model.UserStatusActive).
		Where(ftsCondition, ftsQuery).
		Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("搜索用户计数失败: %w", err)
	}

	if total == 0 {
		return []*model.User{}, 0, nil
	}

	var users []*model.User
	// 按 ts_rank 相关性降序排列，相关性相同时按创建时间降序
	if err := r.db.WithContext(ctx).
		Where("status = ?", model.UserStatusActive).
		Where(ftsCondition, ftsQuery).
		Order(gorm.Expr(
			"ts_rank(to_tsvector('simple', coalesce(username,'') || ' ' || coalesce(bio,'')), to_tsquery('simple', ?)) DESC, created_at DESC",
			ftsQuery,
		)).
		Offset(offset).
		Limit(limit).
		Find(&users).Error; err != nil {
		return nil, 0, fmt.Errorf("搜索用户失败: %w", err)
	}

	return users, total, nil
}

// SearchLogs 按关键词搜索开发日志（PostgreSQL 全文搜索匹配 title），
// 只搜索 status=published 且 visibility=public 的日志，分页
// 使用 simple 字典，仅搜索 title 字段（content 可能很长，避免性能问题），按 ts_rank 相关性降序
func (r *discoverRepository) SearchLogs(ctx context.Context, keyword string, offset, limit int) ([]*model.DevLog, int64, error) {
	ftsQuery := sanitizeFTSQuery(keyword)
	if ftsQuery == "" {
		return []*model.DevLog{}, 0, nil
	}

	// 仅搜索 title，content 字段可能很长，只取 title 做全文索引更高效
	ftsCondition := "to_tsvector('simple', coalesce(title,'')) @@ to_tsquery('simple', ?)"
	bannedFilter := "project_id IS NULL OR project_id IN (SELECT id FROM projects WHERE deleted_at IS NULL AND is_banned = false)"

	var total int64
	if err := r.db.WithContext(ctx).
		Model(&model.DevLog{}).
		Where("deleted_at IS NULL").
		Where("status = ? AND visibility = ?", model.DevLogStatusPublished, model.DevLogVisibilityPublic).
		Where(bannedFilter).
		Where(ftsCondition, ftsQuery).
		Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("搜索日志计数失败: %w", err)
	}

	if total == 0 {
		return []*model.DevLog{}, 0, nil
	}

	var logs []*model.DevLog
	// 按 ts_rank 相关性降序，相关性相同时按点赞数降序
	if err := r.db.WithContext(ctx).
		Where("deleted_at IS NULL").
		Where("status = ? AND visibility = ?", model.DevLogStatusPublished, model.DevLogVisibilityPublic).
		Where(bannedFilter).
		Where(ftsCondition, ftsQuery).
		Order(gorm.Expr(
			"ts_rank(to_tsvector('simple', coalesce(title,'')), to_tsquery('simple', ?)) DESC, like_count DESC, created_at DESC",
			ftsQuery,
		)).
		Offset(offset).
		Limit(limit).
		Find(&logs).Error; err != nil {
		return nil, 0, fmt.Errorf("搜索日志失败: %w", err)
	}

	return logs, total, nil
}
func (r *discoverRepository) GetProjectsByFollowerCount(ctx context.Context, offset, limit int) ([]*model.Project, int64, error) {
	var total int64
	if err := r.db.WithContext(ctx).
		Model(&model.Project{}).
		Where("deleted_at IS NULL AND is_banned = false").
		Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("统计项目总数失败: %w", err)
	}

	if total == 0 {
		return []*model.Project{}, 0, nil
	}

	var projects []*model.Project
	if err := r.db.WithContext(ctx).
		Where("deleted_at IS NULL AND is_banned = false").
		Order("follower_count DESC, updated_at DESC").
		Offset(offset).
		Limit(limit).
		Find(&projects).Error; err != nil {
		return nil, 0, fmt.Errorf("按关注数查询项目失败: %w", err)
	}

	return projects, total, nil
}
func (r *discoverRepository) GetTopLogs(ctx context.Context, offset, limit int, logType string) ([]*model.DevLog, int64, error) {
	// project_id 可为空（不关联项目的独立日志），关联了项目的日志需确保项目未被下架
	baseQuery := r.db.WithContext(ctx).
		Model(&model.DevLog{}).
		Where("deleted_at IS NULL").
		Where("status = ? AND visibility = ?", model.DevLogStatusPublished, model.DevLogVisibilityPublic).
		Where("project_id IS NULL OR project_id IN (SELECT id FROM projects WHERE deleted_at IS NULL AND is_banned = false)")
	if logType != "" {
		baseQuery = baseQuery.Where("log_type = ?", logType)
	}

	var total int64
	if err := baseQuery.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("统计热门日志总数失败: %w", err)
	}

	if total == 0 {
		return []*model.DevLog{}, 0, nil
	}

	fetchQuery := r.db.WithContext(ctx).
		Where("deleted_at IS NULL").
		Where("status = ? AND visibility = ?", model.DevLogStatusPublished, model.DevLogVisibilityPublic).
		Where("project_id IS NULL OR project_id IN (SELECT id FROM projects WHERE deleted_at IS NULL AND is_banned = false)")
	if logType != "" {
		fetchQuery = fetchQuery.Where("log_type = ?", logType)
	}
	var logs []*model.DevLog
	if err := fetchQuery.
		Order("like_count DESC, view_count DESC, created_at DESC").
		Offset(offset).
		Limit(limit).
		Find(&logs).Error; err != nil {
		return nil, 0, fmt.Errorf("查询热门日志失败: %w", err)
	}

	return logs, total, nil
}

// ===========================
// FTS 工具函数
// ===========================

// sanitizeFTSQuery 将用户输入的关键词转为安全的 to_tsquery 参数。
// 移除 PostgreSQL tsquery 特殊字符，多词之间用 & 连接（AND 语义），支持中文。
// 若关键词全为特殊字符，则返回空字符串（调用方应降级或返回空结果）。
func sanitizeFTSQuery(keyword string) string {
	// 移除 PostgreSQL tsquery 特殊字符：| ! & ( ) ' < > : * \\
	re := regexp.MustCompile(`[|!&()'<>:*\\\\]`)
	clean := re.ReplaceAllString(keyword, " ")
	words := strings.Fields(clean)
	if len(words) == 0 {
		return ""
	}
	return strings.Join(words, " & ")
}

func (r *discoverRepository) GetProjectsByIDs(ctx context.Context, ids []uint64) ([]*model.Project, error) {
	if len(ids) == 0 {
		return []*model.Project{}, nil
	}
	var projects []*model.Project
	if err := r.db.WithContext(ctx).
		Where("id IN ? AND deleted_at IS NULL AND is_banned = false", ids).
		Find(&projects).Error; err != nil {
		return nil, err
	}
	return projects, nil
}

// GetUsersByIDs 按 ID 列表批量查用户
func (r *discoverRepository) GetUsersByIDs(ctx context.Context, ids []uint64) ([]*model.User, error) {
	if len(ids) == 0 {
		return []*model.User{}, nil
	}
	var users []*model.User
	if err := r.db.WithContext(ctx).Where("id IN ?", ids).Find(&users).Error; err != nil {
		return nil, err
	}
	return users, nil
}

// GetLogsByIDs 按 ID 列表批量查日志
func (r *discoverRepository) GetLogsByIDs(ctx context.Context, ids []uint64) ([]*model.DevLog, error) {
	if len(ids) == 0 {
		return []*model.DevLog{}, nil
	}
	var logs []*model.DevLog
	if err := r.db.WithContext(ctx).Where("id IN ?", ids).Find(&logs).Error; err != nil {
		return nil, err
	}
	return logs, nil
}
func (r *discoverRepository) SearchPosts(ctx context.Context, keyword string, offset, limit int) ([]*model.Post, int64, error) {
	ftsQuery := sanitizeFTSQuery(keyword)
	if ftsQuery == "" {
		return []*model.Post{}, 0, nil
	}

	ftsCondition := "to_tsvector('simple', coalesce(title,'') || ' ' || coalesce(content,'')) @@ to_tsquery('simple', ?)"

	var total int64
	if err := r.db.WithContext(ctx).
		Model(&model.Post{}).
		Where("deleted_at IS NULL").
		Where(ftsCondition, ftsQuery).
		Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("搜索帖子计数失败: %w", err)
	}

	if total == 0 {
		return []*model.Post{}, 0, nil
	}

	var posts []*model.Post
	if err := r.db.WithContext(ctx).
		Where("deleted_at IS NULL").
		Where(ftsCondition, ftsQuery).
		Order(gorm.Expr(
			"ts_rank(to_tsvector('simple', coalesce(title,'') || ' ' || coalesce(content,'')), to_tsquery('simple', ?)) DESC, like_count DESC, created_at DESC",
			ftsQuery,
		)).
		Offset(offset).
		Limit(limit).
		Find(&posts).Error; err != nil {
		return nil, 0, fmt.Errorf("搜索帖子失败: %w", err)
	}

	return posts, total, nil
}

// ===========================
// 帖子新增实现
// ===========================

// GetTopPosts 按热度（like×2+comment+collect）降序获取帖子，分页
func (r *discoverRepository) GetTopPosts(ctx context.Context, offset, limit int) ([]*model.Post, int64, error) {
	var total int64
	if err := r.db.WithContext(ctx).
		Model(&model.Post{}).
		Where("deleted_at IS NULL").
		Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("统计热门帖子总数失败: %w", err)
	}
	if total == 0 {
		return []*model.Post{}, 0, nil
	}
	var posts []*model.Post
	if err := r.db.WithContext(ctx).
		Where("deleted_at IS NULL").
		Order("(like_count * 2 + comment_count + collect_count) DESC, created_at DESC").
		Offset(offset).Limit(limit).
		Find(&posts).Error; err != nil {
		return nil, 0, fmt.Errorf("查询热门帖子失败: %w", err)
	}
	return posts, total, nil
}

// GetDiscoverPostsByIDs 按 ID 列表批量获取帖子（过滤软删除）
func (r *discoverRepository) GetDiscoverPostsByIDs(ctx context.Context, ids []uint64) ([]*model.Post, error) {
	if len(ids) == 0 {
		return []*model.Post{}, nil
	}
	var posts []*model.Post
	if err := r.db.WithContext(ctx).
		Where("id IN ? AND deleted_at IS NULL", ids).
		Find(&posts).Error; err != nil {
		return nil, err
	}
	return posts, nil
}

// GetHotPostIDs 获取热度最高的帖子 ID（兜底推荐）
func (r *discoverRepository) GetHotPostIDs(ctx context.Context, excludeIDs []uint64, limit int) ([]uint64, error) {
	if limit <= 0 {
		limit = 50
	}
	query := r.db.WithContext(ctx).
		Model(&model.Post{}).
		Select("id").
		Where("deleted_at IS NULL").
		Order("(like_count * 2 + comment_count + collect_count) DESC, created_at DESC").
		Limit(limit)
	if len(excludeIDs) > 0 {
		query = query.Where("id NOT IN ?", excludeIDs)
	}
	var ids []uint64
	if err := query.Pluck("id", &ids).Error; err != nil {
		return nil, fmt.Errorf("获取热门帖子 ID 失败: %w", err)
	}
	if ids == nil {
		ids = []uint64{}
	}
	return ids, nil
}

// GetCoInteractedPostIDs 共现查询：与 postIDs 被同一用户交互（点赞/收藏/评论）的其他帖子 ID
// 使用 UNION ALL 合并点赞、收藏、评论三种交互信号的共现结果，按共现次数降序排列
func (r *discoverRepository) GetCoInteractedPostIDs(ctx context.Context, postIDs []uint64, excludeIDs []uint64, limit int) ([]uint64, error) {
	if len(postIDs) == 0 {
		return []uint64{}, nil
	}
	if limit <= 0 {
		limit = 50
	}
	var ids []uint64

	baseSQL := `
		SELECT post_id FROM (
			SELECT pl2.post_id FROM post_likes pl1
			JOIN post_likes pl2 ON pl1.user_id = pl2.user_id
			WHERE pl1.post_id IN ?
			UNION ALL
			SELECT pc2.post_id FROM post_collects pc1
			JOIN post_collects pc2 ON pc1.user_id = pc2.user_id
			WHERE pc1.post_id IN ?
			UNION ALL
			SELECT pcm2.post_id FROM post_comments pcm1
			JOIN post_comments pcm2 ON pcm1.user_id = pcm2.user_id
			WHERE pcm1.post_id IN ? AND pcm1.deleted_at IS NULL AND pcm2.deleted_at IS NULL
		) t
	`
	if len(excludeIDs) > 0 {
		baseSQL += " WHERE t.post_id NOT IN ?"
	}
	baseSQL += " GROUP BY t.post_id ORDER BY COUNT(*) DESC LIMIT ?"

	args := []interface{}{postIDs, postIDs, postIDs}
	if len(excludeIDs) > 0 {
		args = append(args, excludeIDs)
	}
	args = append(args, limit)

	if err := r.db.WithContext(ctx).Raw(baseSQL, args...).Pluck("post_id", &ids).Error; err != nil {
		return nil, fmt.Errorf("共现帖子查询失败: %w", err)
	}
	if ids == nil {
		ids = []uint64{}
	}
	return ids, nil
}

// GetUserPreferredTopicIDs 从用户发帖历史中推断偏好话题 ID（取最近20帖，按频率排序）
func (r *discoverRepository) GetUserPreferredTopicIDs(ctx context.Context, userID uint64, limit int) ([]uint64, error) {
	var topicIDsJSON []string
	if err := r.db.WithContext(ctx).
		Model(&model.Post{}).
		Select("topic_ids").
		Where("author_id = ? AND deleted_at IS NULL AND topic_ids != '' AND topic_ids != '[]'", userID).
		Order("created_at DESC").
		Limit(20).
		Pluck("topic_ids", &topicIDsJSON).Error; err != nil {
		return nil, err
	}
	freq := make(map[uint64]int)
	for _, j := range topicIDsJSON {
		var ids []uint64
		if err := json.Unmarshal([]byte(j), &ids); err == nil {
			for _, id := range ids {
				freq[id]++
			}
		}
	}
	type kv struct {
		id    uint64
		count int
	}
	kvs := make([]kv, 0, len(freq))
	for id, c := range freq {
		kvs = append(kvs, kv{id, c})
	}
	sort.Slice(kvs, func(i, j int) bool { return kvs[i].count > kvs[j].count })
	if limit <= 0 {
		limit = 5
	}
	result := make([]uint64, 0, limit)
	for i, k := range kvs {
		if i >= limit {
			break
		}
		result = append(result, k.id)
	}
	return result, nil
}

// buildTopicIDsClauses 构建 topic_ids JSON 字段的精确匹配条件。
// topic_ids 格式为 "[1,10,21]"，用正则确保数字边界匹配，避免 LIKE '%1%' 误匹配 10/11/21 等。
func buildTopicIDsClauses(topicIDs []uint64) string {
	clauses := make([]string, 0, len(topicIDs))
	for _, tid := range topicIDs {
		// 匹配：数组开头 "[id," 或 "[id]" 或 ",id," 或 ",id]"
		clauses = append(clauses, fmt.Sprintf(`topic_ids ~ '(^\[|,)%d(,|\])'`, tid))
	}
	return strings.Join(clauses, " OR ")
}

// GetPostsByTopicIDs 根据话题 ID 列表获取帖子（L2 降级，按热度排序）
func (r *discoverRepository) GetPostsByTopicIDs(ctx context.Context, topicIDs []uint64, excludeIDs []uint64, offset, limit int) ([]*model.Post, int64, error) {
	if len(topicIDs) == 0 {
		return []*model.Post{}, 0, nil
	}
	// 构建 OR 条件：JSON 字段中精确匹配任意一个 topicID
	orClause := buildTopicIDsClauses(topicIDs)

	baseQ := r.db.WithContext(ctx).Model(&model.Post{}).Where("deleted_at IS NULL").Where(orClause)
	if len(excludeIDs) > 0 {
		baseQ = baseQ.Where("id NOT IN ?", excludeIDs)
	}
	var total int64
	if err := baseQ.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if total == 0 {
		return []*model.Post{}, 0, nil
	}
	var posts []*model.Post
	q2 := r.db.WithContext(ctx).Where("deleted_at IS NULL").Where(orClause)
	if len(excludeIDs) > 0 {
		q2 = q2.Where("id NOT IN ?", excludeIDs)
	}
	if err := q2.Order("(like_count*2+comment_count+collect_count) DESC, created_at DESC").
		Offset(offset).Limit(limit).Find(&posts).Error; err != nil {
		return nil, 0, err
	}
	return posts, total, nil
}

// ===========================
// 日志新增实现
// ===========================

// GetLogsByIDsFiltered 按 ID 批量获取日志（过滤软删除 + is_banned 项目 + 仅公开已发布）
func (r *discoverRepository) GetLogsByIDsFiltered(ctx context.Context, ids []uint64) ([]*model.DevLog, error) {
	if len(ids) == 0 {
		return []*model.DevLog{}, nil
	}
	var logs []*model.DevLog
	if err := r.db.WithContext(ctx).
		Where("id IN ? AND deleted_at IS NULL AND status = ? AND visibility = ?", ids, model.DevLogStatusPublished, model.DevLogVisibilityPublic).
		Where("project_id IS NULL OR project_id IN (SELECT id FROM projects WHERE deleted_at IS NULL AND is_banned = false)").
		Find(&logs).Error; err != nil {
		return nil, err
	}
	return logs, nil
}
func (r *discoverRepository) GetUserInteractedPostIDs(ctx context.Context, userID uint64, limit int) ([]uint64, error) {
	if limit <= 0 {
		limit = 50
	}
	var ids []uint64
	err := r.db.WithContext(ctx).Raw(`
		SELECT DISTINCT post_id FROM (
			SELECT post_id, created_at FROM post_likes    WHERE user_id = ?
			UNION ALL
			SELECT post_id, created_at FROM post_collects WHERE user_id = ?
			UNION ALL
			SELECT post_id, created_at FROM post_comments WHERE user_id = ? AND deleted_at IS NULL
		) t
		ORDER BY created_at DESC
		LIMIT ?
	`, userID, userID, userID, limit).Pluck("post_id", &ids).Error
	if err != nil {
		return nil, fmt.Errorf("聚合用户帖子互动失败: %w", err)
	}
	if ids == nil {
		ids = []uint64{}
	}
	return ids, nil
}
func (r *discoverRepository) GetSocialFeedPostIDs(ctx context.Context, userID uint64, excludeIDs []uint64, limit int) ([]uint64, error) {
	if limit <= 0 {
		limit = 50
	}
	var ids []uint64
	err := r.db.WithContext(ctx).Raw(`
		SELECT DISTINCT t.post_id
		FROM (
			SELECT pl.post_id, pl.created_at
			FROM post_likes pl
			JOIN user_follows uf ON uf.followed_id = pl.user_id
			WHERE uf.follower_id = ?
			UNION ALL
			SELECT pc.post_id, pc.created_at
			FROM post_collects pc
			JOIN user_follows uf ON uf.followed_id = pc.user_id
			WHERE uf.follower_id = ?
		) t
		JOIN posts p ON p.id = t.post_id
		WHERE p.deleted_at IS NULL
		ORDER BY t.created_at DESC
		LIMIT ?
	`, userID, userID, limit).Pluck("t.post_id", &ids).Error
	if err != nil {
		return nil, fmt.Errorf("社交 CF 帖子查询失败: %w", err)
	}
	if ids == nil {
		ids = []uint64{}
	}
	// 应用层 exclude 过滤
	if len(excludeIDs) > 0 {
		exSet := make(map[uint64]struct{}, len(excludeIDs))
		for _, id := range excludeIDs {
			exSet[id] = struct{}{}
		}
		filtered := ids[:0]
		for _, id := range ids {
			if _, ok := exSet[id]; !ok {
				filtered = append(filtered, id)
			}
		}
		return filtered, nil
	}
	return ids, nil
}
func (r *discoverRepository) GetHotPostsByTopics(ctx context.Context, topicIDs []uint64, excludeIDs []uint64, limit int) ([]uint64, error) {
	if len(topicIDs) == 0 || limit <= 0 {
		return []uint64{}, nil
	}
	orClause := buildTopicIDsClauses(topicIDs)
	query := r.db.WithContext(ctx).
		Model(&model.Post{}).
		Select("id").
		Where("deleted_at IS NULL").
		Where(orClause).
		Order("(like_count*2+comment_count+collect_count) DESC, created_at DESC").
		Limit(limit)
	if len(excludeIDs) > 0 {
		query = query.Where("id NOT IN ?", excludeIDs)
	}
	var ids []uint64
	if err := query.Pluck("id", &ids).Error; err != nil {
		return nil, fmt.Errorf("按话题获取热门帖子 ID 失败: %w", err)
	}
	if ids == nil {
		ids = []uint64{}
	}
	return ids, nil
}
