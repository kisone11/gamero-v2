package repository

import (
	"context"
	"strings"

	"github.com/gamero/gamero/internal/model"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// SensitiveWordRepository 敏感词持久化仓库接口
type SensitiveWordRepository interface {
	// LoadAll 全量加载词库（启动时调用）
	LoadAll(ctx context.Context) ([]string, error)
	// Add 批量写入（已存在的词通过 upsert 跳过）
	Add(ctx context.Context, words []string, createdBy uint64) error
	// Remove 删除一个词（忽略不存在的词）
	Remove(ctx context.Context, word string) error
}

type sensitiveWordRepo struct {
	db *gorm.DB
}

// NewSensitiveWordRepository 构造函数
func NewSensitiveWordRepository(db *gorm.DB) SensitiveWordRepository {
	return &sensitiveWordRepo{db: db}
}

// LoadAll 返回全量小写词列表
func (r *sensitiveWordRepo) LoadAll(ctx context.Context) ([]string, error) {
	var rows []model.SensitiveWord
	if err := r.db.WithContext(ctx).Select("word").Find(&rows).Error; err != nil {
		return nil, err
	}
	words := make([]string, 0, len(rows))
	for _, row := range rows {
		words = append(words, row.Word)
	}
	return words, nil
}

// Add 批量 upsert——已有记录按唯一键跳过
func (r *sensitiveWordRepo) Add(ctx context.Context, words []string, createdBy uint64) error {
	if len(words) == 0 {
		return nil
	}
	rows := make([]model.SensitiveWord, 0, len(words))
	seen := make(map[string]struct{}, len(words))
	for _, w := range words {
		w = strings.ToLower(strings.TrimSpace(w))
		if w == "" {
			continue
		}
		if _, dup := seen[w]; dup {
			continue
		}
		seen[w] = struct{}{}
		rows = append(rows, model.SensitiveWord{Word: w, CreatedBy: createdBy})
	}
	if len(rows) == 0 {
		return nil
	}
	// ON CONFLICT DO NOTHING（PostgreSQL / SQLite）
	return r.db.WithContext(ctx).
		Clauses(clause.OnConflict{DoNothing: true}).
		Create(&rows).Error
}

// Remove 删除单个词（不报 404）
func (r *sensitiveWordRepo) Remove(ctx context.Context, word string) error {
	word = strings.ToLower(strings.TrimSpace(word))
	if word == "" {
		return nil
	}
	return r.db.WithContext(ctx).
		Where("word = ?", word).
		Delete(&model.SensitiveWord{}).Error
}
