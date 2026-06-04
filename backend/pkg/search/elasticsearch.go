package search

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"

	"github.com/elastic/go-elasticsearch/v8"
	"go.uber.org/zap"
)

// ESSearcher Elasticsearch 搜索实现
type ESSearcher struct {
	client      *elasticsearch.Client
	indexPrefix string
	logger      *zap.Logger
}

// NewESSearcher 创建 ES 搜索器
func NewESSearcher(addresses []string, username, password, indexPrefix string, logger *zap.Logger) (*ESSearcher, error) {
	cfg := elasticsearch.Config{
		Addresses: addresses,
		Username:  username,
		Password:  password,
	}
	client, err := elasticsearch.NewClient(cfg)
	if err != nil {
		return nil, fmt.Errorf("search: failed to create ES client: %w", err)
	}
	s := &ESSearcher{
		client:      client,
		indexPrefix: indexPrefix,
		logger:      logger,
	}
	// 初始化 mapping（幂等，索引已存在时跳过）
	ctx := context.Background()
	if err := s.ensureIndices(ctx); err != nil {
		// 初始化失败只警告，不阻断启动（旧索引可能已存在）
		s.logger.Warn("search: failed to ensure ES indices", zap.Error(err))
	}
	return s, nil
}

// ensureIndices 确保所有业务索引存在（带中文分词 mapping）
// 使用 create index API，已存在时返回 400 视为成功（幂等）
func (e *ESSearcher) ensureIndices(ctx context.Context) error {
	indices := map[string]string{
		"projects": projectsMapping,
		"users":    usersMapping,
		"logs":     logsMapping,
		"posts":    postsMapping,
	}
	for name, mapping := range indices {
		if err := e.createIndexIfNotExists(ctx, name, mapping); err != nil {
			return fmt.Errorf("search: ensure index %s: %w", name, err)
		}
	}
	return nil
}

// createIndexIfNotExists 创建索引（已存在时静默跳过）
func (e *ESSearcher) createIndexIfNotExists(ctx context.Context, name, mapping string) error {
	res, err := e.client.Indices.Create(
		e.idx(name),
		e.client.Indices.Create.WithBody(bytes.NewBufferString(mapping)),
		e.client.Indices.Create.WithContext(ctx),
	)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	// 400 resource_already_exists_exception → 忽略
	if res.StatusCode == 400 {
		return nil
	}
	if res.IsError() {
		body, _ := io.ReadAll(res.Body)
		return fmt.Errorf("create index %s: %s", name, string(body))
	}
	e.logger.Info("search: index created", zap.String("index", e.idx(name)))
	return nil
}

// projectsMapping 项目索引 mapping（ik_max_word 中文分词）
const projectsMapping = `{
  "settings": {
    "analysis": {
      "analyzer": {
        "ik_smart_analyzer": {"type": "custom", "tokenizer": "ik_smart"},
        "ik_max_analyzer":   {"type": "custom", "tokenizer": "ik_max_word"}
      }
    }
  },
  "mappings": {
    "properties": {
      "id":           {"type": "long"},
      "name":         {"type": "text", "analyzer": "ik_max_analyzer", "search_analyzer": "ik_smart_analyzer"},
      "description":  {"type": "text", "analyzer": "ik_max_analyzer", "search_analyzer": "ik_smart_analyzer"},
      "genre":        {"type": "keyword"},
      "status":       {"type": "keyword"},
      "style_tags":   {"type": "keyword"},
      "hot_score":    {"type": "float"},
      "created_at":   {"type": "date"}
    }
  }
}`

// usersMapping 用户索引 mapping
const usersMapping = `{
  "settings": {
    "analysis": {
      "analyzer": {
        "ik_smart_analyzer": {"type": "custom", "tokenizer": "ik_smart"},
        "ik_max_analyzer":   {"type": "custom", "tokenizer": "ik_max_word"}
      }
    }
  },
  "mappings": {
    "properties": {
      "id":       {"type": "long"},
      "username": {"type": "keyword"},
      "nickname": {"type": "text", "analyzer": "ik_max_analyzer", "search_analyzer": "ik_smart_analyzer"},
      "bio":      {"type": "text", "analyzer": "ik_max_analyzer", "search_analyzer": "ik_smart_analyzer"},
      "role":     {"type": "keyword"}
    }
  }
}`

// logsMapping 开发日志索引 mapping
const logsMapping = `{
  "settings": {
    "analysis": {
      "analyzer": {
        "ik_smart_analyzer": {"type": "custom", "tokenizer": "ik_smart"},
        "ik_max_analyzer":   {"type": "custom", "tokenizer": "ik_max_word"}
      }
    }
  },
  "mappings": {
    "properties": {
      "id":         {"type": "long"},
      "title":      {"type": "text", "analyzer": "ik_max_analyzer", "search_analyzer": "ik_smart_analyzer"},
      "content":    {"type": "text", "analyzer": "ik_max_analyzer", "search_analyzer": "ik_smart_analyzer"},
      "project_id": {"type": "long"},
      "author_id":  {"type": "long"},
      "created_at": {"type": "date"}
    }
  }
}`

// postsMapping 帖子索引 mapping
const postsMapping = `{
  "settings": {
    "analysis": {
      "analyzer": {
        "ik_smart_analyzer": {"type": "custom", "tokenizer": "ik_smart"},
        "ik_max_analyzer":   {"type": "custom", "tokenizer": "ik_max_word"}
      }
    }
  },
  "mappings": {
    "properties": {
      "id":         {"type": "long"},
      "title":      {"type": "text", "analyzer": "ik_max_analyzer", "search_analyzer": "ik_smart_analyzer"},
      "content":    {"type": "text", "analyzer": "ik_max_analyzer", "search_analyzer": "ik_smart_analyzer"},
      "author_id":  {"type": "long"},
      "created_at": {"type": "date"}
    }
  }
}`

func (e *ESSearcher) idx(name string) string {
	return e.indexPrefix + name
}

// indexDoc 通用文档索引
func (e *ESSearcher) indexDoc(ctx context.Context, index string, id uint64, doc interface{}) error {
	data, err := json.Marshal(doc)
	if err != nil {
		return err
	}
	res, err := e.client.Index(
		e.idx(index),
		bytes.NewReader(data),
		e.client.Index.WithDocumentID(fmt.Sprintf("%d", id)),
		e.client.Index.WithContext(ctx),
	)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.IsError() {
		return fmt.Errorf("search: index %s error: %s", index, res.Status())
	}
	return nil
}

func (e *ESSearcher) IndexProject(ctx context.Context, doc ProjectDoc) error {
	return e.indexDoc(ctx, "projects", doc.ID, doc)
}
func (e *ESSearcher) IndexUser(ctx context.Context, doc UserDoc) error {
	return e.indexDoc(ctx, "users", doc.ID, doc)
}
func (e *ESSearcher) IndexLog(ctx context.Context, doc LogDoc) error {
	return e.indexDoc(ctx, "logs", doc.ID, doc)
}
func (e *ESSearcher) IndexPost(ctx context.Context, doc PostDoc) error {
	return e.indexDoc(ctx, "posts", doc.ID, doc)
}

// searchByKeyword 通用多字段全文搜索
func (e *ESSearcher) searchByKeyword(ctx context.Context, index, keyword string, fields []string, offset, limit int) (Result, error) {
	query := map[string]interface{}{
		"from": offset,
		"size": limit,
		"query": map[string]interface{}{
			"multi_match": map[string]interface{}{
				"query":  keyword,
				"fields": fields,
				"type":   "best_fields",
			},
		},
	}
	data, _ := json.Marshal(query)
	res, err := e.client.Search(
		e.client.Search.WithIndex(e.idx(index)),
		e.client.Search.WithBody(bytes.NewReader(data)),
		e.client.Search.WithContext(ctx),
	)
	if err != nil {
		return Result{}, err
	}
	defer res.Body.Close()
	if res.IsError() {
		return Result{}, fmt.Errorf("search: query %s error: %s", index, res.Status())
	}
	return e.parseResponse(res.Body)
}

func (e *ESSearcher) parseResponse(body io.Reader) (Result, error) {
	var r struct {
		Hits struct {
			Total struct {
				Value int64 `json:"value"`
			} `json:"total"`
			Hits []struct {
				ID    string  `json:"_id"`
				Score float64 `json:"_score"`
			} `json:"hits"`
		} `json:"hits"`
	}
	if err := json.NewDecoder(body).Decode(&r); err != nil {
		return Result{}, err
	}
	hits := make([]Hit, 0, len(r.Hits.Hits))
	for _, h := range r.Hits.Hits {
		var id uint64
		fmt.Sscanf(h.ID, "%d", &id)
		hits = append(hits, Hit{ID: id, Score: h.Score})
	}
	return Result{Hits: hits, Total: r.Hits.Total.Value}, nil
}

func (e *ESSearcher) SearchProjects(ctx context.Context, keyword string, offset, limit int) (Result, error) {
	return e.searchByKeyword(ctx, "projects", keyword, []string{"name^3", "description", "tags^2"}, offset, limit)
}
func (e *ESSearcher) SearchUsers(ctx context.Context, keyword string, offset, limit int) (Result, error) {
	return e.searchByKeyword(ctx, "users", keyword, []string{"username^2", "bio"}, offset, limit)
}
func (e *ESSearcher) SearchLogs(ctx context.Context, keyword string, offset, limit int) (Result, error) {
	return e.searchByKeyword(ctx, "logs", keyword, []string{"title^3", "content"}, offset, limit)
}
func (e *ESSearcher) SearchPosts(ctx context.Context, keyword string, offset, limit int) (Result, error) {
	return e.searchByKeyword(ctx, "posts", keyword, []string{"title^3", "content"}, offset, limit)
}

func (e *ESSearcher) deleteDoc(ctx context.Context, index string, id uint64) error {
	res, err := e.client.Delete(
		e.idx(index),
		fmt.Sprintf("%d", id),
		e.client.Delete.WithContext(ctx),
	)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	return nil
}

func (e *ESSearcher) DeleteProject(ctx context.Context, id uint64) error {
	return e.deleteDoc(ctx, "projects", id)
}
func (e *ESSearcher) DeleteUser(ctx context.Context, id uint64) error {
	return e.deleteDoc(ctx, "users", id)
}
func (e *ESSearcher) DeleteLog(ctx context.Context, id uint64) error {
	return e.deleteDoc(ctx, "logs", id)
}
func (e *ESSearcher) DeletePost(ctx context.Context, id uint64) error {
	return e.deleteDoc(ctx, "posts", id)
}

func (e *ESSearcher) Ping(ctx context.Context) error {
	res, err := e.client.Ping(e.client.Ping.WithContext(ctx))
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.IsError() {
		return fmt.Errorf("search: ES ping failed: %s", res.Status())
	}
	return nil
}
