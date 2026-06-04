// Package storage 提供基于 AWS S3 兼容接口的对象存储封装。
// 使用标准库 net/http + crypto/hmac + crypto/sha256 实现 AWS Signature V4 签名，
// 兼容腾讯云 COS、阿里云 OSS、AWS S3 等 S3 协议存储服务。
package storage

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"

	"github.com/gamero/gamero/config"
	"github.com/gamero/gamero/pkg/logger"
	"go.uber.org/zap"
)

// global 是全局存储客户端实例
var global *Client

// Client 封装了 S3 兼容存储客户端及配置
type Client struct {
	cfg        config.StorageConfig
	httpClient *http.Client
}

// UploadResult 上传结果
type UploadResult struct {
	ObjectName string // 对象名（路径）
	ETag       string // 文件 ETag
	Size       int64  // 文件大小
}

// Init 初始化 S3 兼容存储客户端
func Init(cfg config.StorageConfig) (*Client, error) {
	client := &Client{
		cfg: cfg,
		httpClient: &http.Client{
			Timeout: 60 * time.Second,
		},
	}

	global = client
	logger.Info("对象存储连接成功",
		zap.String("provider", cfg.Provider),
		zap.String("endpoint", cfg.Endpoint),
		zap.String("bucket", cfg.BucketName),
	)
	return client, nil
}

// Get 返回全局存储客户端
func Get() *Client {
	if global == nil {
		panic("存储未初始化，请先调用 storage.Init()")
	}
	return global
}

// Ping 检查存储连接状态（通过 HEAD 请求检查 bucket）
func Ping() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	c := Get()
	objectURL := c.buildObjectURL("")
	req, err := http.NewRequestWithContext(ctx, http.MethodHead, objectURL, nil)
	if err != nil {
		return fmt.Errorf("构造 Ping 请求失败: %w", err)
	}
	c.signRequest(req, "", nil)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("存储服务连接失败: %w", err)
	}
	defer resp.Body.Close()
	// 403 说明有连接但认证问题，也算可用
	if resp.StatusCode >= 500 {
		return fmt.Errorf("存储服务异常: HTTP %d", resp.StatusCode)
	}
	return nil
}

// BucketName 返回默认 bucket 名称
func (c *Client) BucketName() string {
	return c.cfg.BucketName
}

// GetPublicURL 返回对象的公开访问 URL（CDN 或存储域名）
func (c *Client) GetPublicURL(objectName string) string {
	baseURL := strings.TrimRight(c.cfg.BaseURL, "/")
	objectName = strings.TrimLeft(objectName, "/")
	return baseURL + "/" + objectName
}

// Upload 上传文件
// objectName: 对象在 bucket 中的名称（路径），如 "avatars/user/123.jpg"
// reader: 文件内容读取器
// size: 文件大小（-1 表示未知，将读取全部内容）
// contentType: MIME 类型，如 "image/jpeg"
func (c *Client) Upload(ctx context.Context, objectName string, reader io.Reader, size int64, contentType string) (*UploadResult, error) {
	var bodyBytes []byte
	var err error

	if size < 0 {
		bodyBytes, err = io.ReadAll(reader)
		if err != nil {
			return nil, fmt.Errorf("读取文件内容失败: %w", err)
		}
		size = int64(len(bodyBytes))
	} else {
		bodyBytes, err = io.ReadAll(io.LimitReader(reader, size))
		if err != nil {
			return nil, fmt.Errorf("读取文件内容失败: %w", err)
		}
	}

	// 计算 payload sha256
	payloadHash := hexSHA256(bodyBytes)
	objectURL := c.buildObjectURL(objectName)

	req, err := http.NewRequestWithContext(ctx, http.MethodPut, objectURL, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("构造上传请求失败: %w", err)
	}

	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	req.Header.Set("x-amz-content-sha256", payloadHash)
	req.ContentLength = size

	c.signRequest(req, payloadHash, nil)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("上传文件失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusNoContent {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("上传文件失败: HTTP %d %s", resp.StatusCode, string(body))
	}

	etag := strings.Trim(resp.Header.Get("ETag"), `"`)
	return &UploadResult{
		ObjectName: objectName,
		ETag:       etag,
		Size:       size,
	}, nil
}

// Download 下载文件，返回 io.ReadCloser，调用方负责关闭
func (c *Client) Download(ctx context.Context, objectName string) (io.ReadCloser, error) {
	objectURL := c.buildObjectURL(objectName)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, objectURL, nil)
	if err != nil {
		return nil, fmt.Errorf("构造下载请求失败: %w", err)
	}
	req.Header.Set("x-amz-content-sha256", "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855")
	c.signRequest(req, "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855", nil)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("下载文件失败: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return nil, fmt.Errorf("下载文件失败: HTTP %d", resp.StatusCode)
	}
	return resp.Body, nil
}

// Delete 删除文件
func (c *Client) Delete(ctx context.Context, objectName string) error {
	objectURL := c.buildObjectURL(objectName)

	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, objectURL, nil)
	if err != nil {
		return fmt.Errorf("构造删除请求失败: %w", err)
	}
	emptyHash := "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"
	req.Header.Set("x-amz-content-sha256", emptyHash)
	c.signRequest(req, emptyHash, nil)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("删除文件失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("删除文件失败: HTTP %d %s", resp.StatusCode, string(body))
	}
	return nil
}

// DeleteMultiple 批量删除文件（逐一删除）
func (c *Client) DeleteMultiple(ctx context.Context, objectNames []string) error {
	for _, name := range objectNames {
		if err := c.Delete(ctx, name); err != nil {
			return err
		}
	}
	return nil
}

// PresignedGetURL 生成预签名下载 URL
// expiry: URL 有效期
func (c *Client) PresignedGetURL(ctx context.Context, objectName string, expiry time.Duration) (string, error) {
	return c.presignURL(ctx, http.MethodGet, objectName, expiry)
}

// PresignedPutURL 生成预签名上传 URL（用于前端直传）
// expiry: URL 有效期
func (c *Client) PresignedPutURL(ctx context.Context, objectName string, expiry time.Duration) (string, error) {
	return c.presignURL(ctx, http.MethodPut, objectName, expiry)
}

// ===== 内部方法 =====

// buildObjectURL 构建对象 URL
// ForcePathStyle=true  → 路径风格：<endpoint>/<bucket>/<object>
// ForcePathStyle=false → 虚拟主机风格：<endpoint-protocol>//<bucket>.<endpoint-host>/<object>
func (c *Client) buildObjectURL(objectName string) string {
	if c.cfg.ForcePathStyle {
		endpoint := strings.TrimRight(c.cfg.Endpoint, "/")
		bucket := c.cfg.BucketName
		objectName = strings.TrimLeft(objectName, "/")
		if objectName == "" {
			return fmt.Sprintf("%s/%s/", endpoint, bucket)
		}
		return fmt.Sprintf("%s/%s/%s", endpoint, bucket, objectName)
	}

	// 虚拟主机风格
	scheme, host := c.parseEndpoint()
	bucket := c.cfg.BucketName
	objectName = strings.TrimLeft(objectName, "/")
	if objectName == "" {
		return fmt.Sprintf("%s://%s.%s/", scheme, bucket, host)
	}
	return fmt.Sprintf("%s://%s.%s/%s", scheme, bucket, host, objectName)
}

// parseEndpoint 解析 endpoint，返回 host 和 scheme
func (c *Client) parseEndpoint() (scheme, host string) {
	endpoint := c.cfg.Endpoint
	if strings.HasPrefix(endpoint, "https://") {
		return "https", strings.TrimPrefix(endpoint, "https://")
	}
	if strings.HasPrefix(endpoint, "http://") {
		return "http", strings.TrimPrefix(endpoint, "http://")
	}
	return "https", endpoint
}

// presignURL 生成预签名 URL（query 参数签名）
func (c *Client) presignURL(_ context.Context, method, objectName string, expiry time.Duration) (string, error) {
	now := time.Now().UTC()
	datetime := now.Format("20060102T150405Z")
	date := now.Format("20060102")
	region := c.cfg.Region
	if region == "" {
		region = "us-east-1"
	}

	credential := fmt.Sprintf("%s/%s/%s/s3/aws4_request",
		c.cfg.AccessKeyID, date, region)

	_, host := c.parseEndpoint()
	bucket := c.cfg.BucketName
	objectName = strings.TrimLeft(objectName, "/")

	var canonicalURI, signingHost string
	if c.cfg.ForcePathStyle {
		// 路径风格: canonical URI = /<bucket>/<object>, host = endpoint host
		canonicalURI = "/" + bucket + "/" + pathEscape(objectName)
		signingHost = host
	} else {
		// 虚拟主机风格: canonical URI = /<object>, host = <bucket>.<endpoint>
		canonicalURI = "/" + pathEscape(objectName)
		signingHost = bucket + "." + host
	}

	expirySeconds := int64(expiry.Seconds())

	// 构造 query 参数（需排序）
	queryParams := url.Values{}
	queryParams.Set("X-Amz-Algorithm", "AWS4-HMAC-SHA256")
	queryParams.Set("X-Amz-Credential", credential)
	queryParams.Set("X-Amz-Date", datetime)
	queryParams.Set("X-Amz-Expires", fmt.Sprintf("%d", expirySeconds))
	queryParams.Set("X-Amz-SignedHeaders", "host")
	canonicalQueryString := encodeQueryParams(queryParams)

	// 规范化 headers（仅签名 host）
	canonicalHeaders := fmt.Sprintf("host:%s\n", signingHost)
	signedHeaders := "host"

	// 构造 CanonicalRequest
	canonicalRequest := strings.Join([]string{
		method,
		canonicalURI,
		canonicalQueryString,
		canonicalHeaders,
		signedHeaders,
		"UNSIGNED-PAYLOAD",
	}, "\n")

	// StringToSign
	scope := fmt.Sprintf("%s/%s/s3/aws4_request", date, region)
	stringToSign := strings.Join([]string{
		"AWS4-HMAC-SHA256",
		datetime,
		scope,
		hexSHA256([]byte(canonicalRequest)),
	}, "\n")

	// 签名密钥
	signingKey := sigV4Key(c.cfg.SecretAccessKey, date, region)
	signature := hexHMAC(signingKey, []byte(stringToSign))

	// 最终 URL（与 buildObjectURL 格式一致）
	finalURL := c.buildObjectURL(objectName) + "?" + canonicalQueryString + "&X-Amz-Signature=" + signature
	return finalURL, nil
}

// signRequest 为 http.Request 添加 AWS SigV4 Authorization header
func (c *Client) signRequest(req *http.Request, payloadHash string, extraHeaders map[string]string) {
	now := time.Now().UTC()
	datetime := now.Format("20060102T150405Z")
	date := now.Format("20060102")
	region := c.cfg.Region
	if region == "" {
		region = "us-east-1"
	}

	// 确定用于签名的主机名
	_, rawHost := c.parseEndpoint()
	signingHost := rawHost
	if !c.cfg.ForcePathStyle {
		signingHost = c.cfg.BucketName + "." + rawHost
	}

	// 添加必要 header
	req.Header.Set("x-amz-date", datetime)
	if req.Header.Get("host") == "" {
		req.Header.Set("host", signingHost)
	}
	if payloadHash != "" && req.Header.Get("x-amz-content-sha256") == "" {
		req.Header.Set("x-amz-content-sha256", payloadHash)
	}
	for k, v := range extraHeaders {
		req.Header.Set(k, v)
	}

	// 收集需要签名的 header（小写、排序）
	signedHeaderKeys := []string{"host", "x-amz-content-sha256", "x-amz-date"}
	var validKeys []string
	for _, k := range signedHeaderKeys {
		if req.Header.Get(k) != "" || k == "host" {
			validKeys = append(validKeys, k)
		}
	}
	sort.Strings(validKeys)

	// 构造 CanonicalHeaders
	var canonicalHeadersBuf strings.Builder
	for _, k := range validKeys {
		val := req.Header.Get(k)
		if k == "host" {
			val = signingHost
		}
		canonicalHeadersBuf.WriteString(k)
		canonicalHeadersBuf.WriteString(":")
		canonicalHeadersBuf.WriteString(strings.TrimSpace(val))
		canonicalHeadersBuf.WriteString("\n")
	}
	canonicalHeaders := canonicalHeadersBuf.String()
	signedHeaders := strings.Join(validKeys, ";")

	// 规范化 URI
	rawPath := req.URL.Path
	if rawPath == "" {
		rawPath = "/"
	}
	// 规范化查询字符串
	canonicalQueryString := encodeQueryParams(req.URL.Query())

	// CanonicalRequest
	if payloadHash == "" {
		payloadHash = "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"
	}
	canonicalRequest := strings.Join([]string{
		req.Method,
		rawPath,
		canonicalQueryString,
		canonicalHeaders,
		signedHeaders,
		payloadHash,
	}, "\n")

	// StringToSign
	scope := fmt.Sprintf("%s/%s/s3/aws4_request", date, region)
	stringToSign := strings.Join([]string{
		"AWS4-HMAC-SHA256",
		datetime,
		scope,
		hexSHA256([]byte(canonicalRequest)),
	}, "\n")

	// 签名密钥
	signingKey := sigV4Key(c.cfg.SecretAccessKey, date, region)
	signature := hexHMAC(signingKey, []byte(stringToSign))

	// Authorization header
	credential := fmt.Sprintf("%s/%s/%s/s3/aws4_request",
		c.cfg.AccessKeyID, date, region)
	authorization := fmt.Sprintf(
		"AWS4-HMAC-SHA256 Credential=%s, SignedHeaders=%s, Signature=%s",
		credential, signedHeaders, signature,
	)
	req.Header.Set("Authorization", authorization)
}

// ===== AWS SigV4 工具函数 =====

// hexSHA256 计算数据的 SHA-256 哈希并返回十六进制字符串
func hexSHA256(data []byte) string {
	h := sha256.Sum256(data)
	return hex.EncodeToString(h[:])
}

// hmacSHA256 计算 HMAC-SHA256
func hmacSHA256(key, data []byte) []byte {
	h := hmac.New(sha256.New, key)
	h.Write(data)
	return h.Sum(nil)
}

// hexHMAC 计算 HMAC-SHA256 并返回十六进制字符串
func hexHMAC(key, data []byte) string {
	return hex.EncodeToString(hmacSHA256(key, data))
}

// sigV4Key 派生 SigV4 签名密钥
// SigningKey = HMAC(HMAC(HMAC(HMAC("AWS4"+secretKey, date), region), "s3"), "aws4_request")
func sigV4Key(secretKey, date, region string) []byte {
	kDate := hmacSHA256([]byte("AWS4"+secretKey), []byte(date))
	kRegion := hmacSHA256(kDate, []byte(region))
	kService := hmacSHA256(kRegion, []byte("s3"))
	kSigning := hmacSHA256(kService, []byte("aws4_request"))
	return kSigning
}

// encodeQueryParams 将 url.Values 编码为规范化查询字符串（按 key 排序）
func encodeQueryParams(params url.Values) string {
	if len(params) == 0 {
		return ""
	}
	keys := make([]string, 0, len(params))
	for k := range params {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var parts []string
	for _, k := range keys {
		vals := params[k]
		sort.Strings(vals)
		for _, v := range vals {
			parts = append(parts, url.QueryEscape(k)+"="+url.QueryEscape(v))
		}
	}
	return strings.Join(parts, "&")
}

// pathEscape 对 object key 中的每段做 URL 编码，但保留斜线
func pathEscape(objectName string) string {
	parts := strings.Split(objectName, "/")
	escaped := make([]string, len(parts))
	for i, p := range parts {
		escaped[i] = url.PathEscape(p)
	}
	return strings.Join(escaped, "/")
}

// ===== 包级别便捷函数 =====

// Upload 使用全局客户端上传文件
func Upload(ctx context.Context, objectName string, reader io.Reader, size int64, contentType string) (*UploadResult, error) {
	return Get().Upload(ctx, objectName, reader, size, contentType)
}

// Download 使用全局客户端下载文件
func Download(ctx context.Context, objectName string) (io.ReadCloser, error) {
	return Get().Download(ctx, objectName)
}

// Delete 使用全局客户端删除文件
func Delete(ctx context.Context, objectName string) error {
	return Get().Delete(ctx, objectName)
}

// PresignedGetURL 使用全局客户端生成预签名下载 URL
func PresignedGetURL(ctx context.Context, objectName string, expiry time.Duration) (string, error) {
	return Get().PresignedGetURL(ctx, objectName, expiry)
}

// PresignedPutURL 使用全局客户端生成预签名上传 URL
func PresignedPutURL(ctx context.Context, objectName string, expiry time.Duration) (string, error) {
	return Get().PresignedPutURL(ctx, objectName, expiry)
}
