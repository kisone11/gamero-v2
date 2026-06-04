// Package service 提供业务逻辑层的共享图片处理工具函数。
package service

import (
	"context"
	"encoding/json"
	"time"

	"github.com/gamero/gamero/pkg/logger"
	"github.com/gamero/gamero/pkg/storage"
	"go.uber.org/zap"
)

// buildImageURLs 将对象存储 key 列表的 JSON 字符串转换为预签名 URL 列表（有效期 24h）。
func buildImageURLs(ctx context.Context, imagesJSON string) []string {
	keys := parseImageKeys(imagesJSON)
	urls := make([]string, 0, len(keys))
	for _, key := range keys {
		if key == "" {
			continue
		}
		url, err := storage.Get().PresignedGetURL(ctx, key, 24*time.Hour)
		if err != nil {
			url = storage.Get().GetPublicURL(key)
		}
		urls = append(urls, url)
	}
	return urls
}

// parseImageKeys 解析帖子图片或视频的 JSON 字符串为 key 列表。
func parseImageKeys(imagesJSON string) []string {
	var keys []string
	if imagesJSON != "" && imagesJSON != "[]" {
		if err := json.Unmarshal([]byte(imagesJSON), &keys); err != nil {
			logger.Warn("failed to unmarshal JSON", zap.Error(err))
		}
	}
	return keys
}
