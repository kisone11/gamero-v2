// upload_limits.go 提供文件上传大小校验工具。
package fileutil

import "fmt"

// 上传大小常量（单位：字节）
const (
	MaxAvatarSize     = 5 << 20  // 5MB  — 头像
	MaxImageSize      = 10 << 20 // 10MB — 封面/截图/帖子图片
	MaxVideoSize      = 100 << 20 // 100MB — 开发日志视频
	MaxDocumentSize   = 10 << 20 // 10MB — 商品封面等文档类图片
)

// CheckFileSize 校验文件大小是否在允许范围内。
// fileSize 单位为字节，maxBytes 为最大允许字节数。
// 超出则返回描述性错误；未超出返回 nil。
func CheckFileSize(fileSize, maxBytes int64, label string) error {
	if fileSize <= 0 {
		return fmt.Errorf("文件大小无效")
	}
	if fileSize > maxBytes {
		return fmt.Errorf("%s 文件大小 %.1fMB 超过限制 %.0fMB",
			label,
			float64(fileSize)/(1<<20),
			float64(maxBytes)/(1<<20))
	}
	return nil
}
