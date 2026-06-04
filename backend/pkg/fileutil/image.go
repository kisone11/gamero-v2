// Package fileutil 提供文件类型检测等通用工具。
package fileutil

import (
	"fmt"
	"io"
	"net/http"
)

// AllowedImageMIME 合法图片 MIME 类型及对应文件扩展名
var AllowedImageMIME = map[string]string{
	"image/jpeg": "jpg",
	"image/png":  "png",
	"image/gif":  "gif",
	"image/webp": "webp",
}

// DetectImageMIME 通过读取文件头（magic bytes）检测真实 MIME 类型。
// 返回 MIME 类型字符串（如 "image/jpeg"）和文件扩展名（如 "jpg"）。
// 若文件类型不在 AllowedImageMIME 白名单内，返回错误。
//
// 注意：r 必须支持从头开始读取（调用方负责传入未读的 io.Reader）。
// 函数只读取前 512 字节用于检测，不消耗后续数据。
func DetectImageMIME(r io.Reader) (mime, ext string, sniffed []byte, err error) {
	// 读取前 512 字节用于 MIME 嗅探
	buf := make([]byte, 512)
	n, readErr := io.ReadAtLeast(r, buf, 1)
	if readErr != nil {
		return "", "", nil, fmt.Errorf("读取文件头失败: %w", readErr)
	}
	buf = buf[:n]

	detected := http.DetectContentType(buf)

	fileExt, ok := AllowedImageMIME[detected]
	if !ok {
		return "", "", nil, fmt.Errorf("不支持的文件类型 %q，仅允许 JPEG/PNG/GIF/WebP", detected)
	}

	return detected, fileExt, buf, nil
}
