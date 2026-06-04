// Package sms 提供短信验证码的生成、存储和验证功能。
// 验证码为 6 位数字，存储在 Redis 中，默认 5 分钟过期。
// 同一手机号 60 秒内不允许重复发送。
//
// 注意：本包实现验证码的逻辑部分，实际短信发送需接入第三方服务商 SDK。
// 当前实现提供基础发送接口，可根据 config.SMSConfig 中的 provider 字段扩展。
package sms

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha1"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/gamero/gamero/config"
	"github.com/gamero/gamero/pkg/cache"
	apperrors "github.com/gamero/gamero/pkg/errors"
	"github.com/gamero/gamero/pkg/logger"
	"go.uber.org/zap"
)

const (
	// codeKeyPrefix 验证码存储 Redis key 前缀
	codeKeyPrefix = "sms:code:"
	// intervalKeyPrefix 发送间隔 Redis key 前缀
	intervalKeyPrefix = "sms:interval:"
	// attemptKeyPrefix 失败次数 Redis key 前缀
	attemptKeyPrefix = "sms:attempt:"
	// codeLength 验证码位数
	codeLength = 6
	// maxAttempts 验证码最大允许错误次数
	maxAttempts = 5
)

// global 是全局 SMS 服务实例
var global *Service

// Service 短信服务
type Service struct {
	cfg config.SMSConfig
}

// Init 初始化全局短信服务
func Init(cfg config.SMSConfig) *Service {
	global = &Service{cfg: cfg}
	return global
}

// Get 返回全局短信服务
func Get() *Service {
	if global == nil {
		panic("短信服务未初始化，请先调用 sms.Init()")
	}
	return global
}

// Send 发送短信验证码到指定手机号
// 如果该手机号在 interval 秒内已发送，返回频率限制错误
func (s *Service) Send(ctx context.Context, phone string) (string, error) {
	// 检查发送间隔
	intervalKey := intervalKeyPrefix + phone
	exists, err := cache.Exists(ctx, intervalKey)
	if err != nil {
		return "", fmt.Errorf("检查发送间隔失败: %w", err)
	}
	if exists {
		return "", apperrors.CodeError(apperrors.CodeSMSCodeFrequency)
	}

	// 生成 6 位随机验证码
	code, err := generateCode(codeLength)
	if err != nil {
		return "", fmt.Errorf("生成验证码失败: %w", err)
	}

	// 存储验证码到 Redis
	codeKey := codeKeyPrefix + phone
	expireDuration := time.Duration(s.cfg.Expire) * time.Second
	if err := cache.Set(ctx, codeKey, code, expireDuration); err != nil {
		return "", fmt.Errorf("存储验证码失败: %w", err)
	}

	// 设置发送间隔锁
	intervalDuration := time.Duration(s.cfg.Interval) * time.Second
	if err := cache.Set(ctx, intervalKey, "1", intervalDuration); err != nil {
		return "", fmt.Errorf("设置发送间隔失败: %w", err)
	}

	// 实际发送短信（通过第三方 SDK）
	if err := s.doSend(ctx, phone, code); err != nil {
		// 发送失败时清理验证码和间隔锁
		if err := cache.Del(ctx, codeKey, intervalKey); err != nil {
			logger.Warn("failed to delete cache", zap.Error(err))
		}
		return "", apperrors.WrapMsg(apperrors.CodeSMSSendFailed, "短信发送失败", err)
	}

	logger.Info("短信验证码已发送", zap.String("phone", maskPhone(phone)))
	return code, nil
}

// Verify 验证手机号对应的验证码
// 验证成功后立即删除验证码（一次性）
// 单个验证码最多尝试 maxAttempts 次，超出后删除验证码防止暴力枚举
func (s *Service) Verify(ctx context.Context, phone, code string) error {
	codeKey := codeKeyPrefix + phone
	attemptKey := attemptKeyPrefix + phone

	storedCode, err := cache.GetString(ctx, codeKey)
	if err != nil {
		// redis.Nil 表示 key 不存在（验证码已过期或未发送）
		return apperrors.CodeError(apperrors.CodeSMSCodeExpired)
	}

	// 错误次数检查
	attempts, _ := cache.IncrBy(ctx, attemptKey, 1)
	if attempts == 1 {
		// 首次计数，设置 TTL（与验证码相同过期时间）
		if err := cache.Expire(ctx, attemptKey, time.Duration(s.cfg.Expire)*time.Second); err != nil {
			logger.Warn("failed to update cache expiry", zap.Error(err))
		}
	}
	if attempts > maxAttempts {
		// 超出最大尝试次数，删除验证码强制失效
		if err := cache.Del(ctx, codeKey, attemptKey); err != nil {
			logger.Warn("failed to delete cache", zap.Error(err))
		}
		return apperrors.CodeError(apperrors.CodeSMSCodeExpired)
	}

	if storedCode != code {
		return apperrors.CodeError(apperrors.CodeSMSCodeInvalid)
	}

	// 验证成功，删除验证码和错误计数
	if err := cache.Del(ctx, codeKey, attemptKey); err != nil {
		logger.Warn("failed to delete cache", zap.Error(err))
	}
	return nil
}

// doSend 实际发送短信，根据配置分发到不同服务商
// 本地/开发模式：不调用云服务，直接打印验证码到日志
// 生产模式：根据 Provider 字段路由到对应服务商实现
func (s *Service) doSend(ctx context.Context, phone, code string) error {
	if s.cfg.Provider == "local" || s.cfg.SecretID == "" || s.cfg.SecretID == "your-secret-id" {
		logger.Info("[SMS] 开发模式 — 验证码打印（生产环境需配置 SMS 服务商）",
			zap.String("phone", maskPhone(phone)),
			zap.String("code", code),
		)
		return nil
	}

	switch s.cfg.Provider {
	case "tencent":
		return s.sendViaTencent(phone, code)
	case "aliyun":
		return s.sendViaAliyun(phone, code)
	default:
		logger.Warn("[SMS] 未知服务商，跳过实际发送", zap.String("provider", s.cfg.Provider))
		return nil
	}
}

// sendViaTencent 腾讯云短信发送（TC3-HMAC-SHA256 签名，无需外部 SDK）
//
// 所需配置字段：SecretID、SecretKey、SdkAppID、SignName、TemplateID
// 文档：https://cloud.tencent.com/document/api/382/55981
func (s *Service) sendViaTencent(phone, code string) error {
	const (
		service   = "sms"
		host      = "sms.tencentcloudapi.com"
		region    = "ap-guangzhou"
		version   = "2021-01-11"
		action    = "SendSms"
		algorithm = "TC3-HMAC-SHA256"
	)

	now := time.Now().UTC()
	timestamp := strconv.FormatInt(now.Unix(), 10)
	date := now.Format("2006-01-02")

	// 1. 构建请求体
	payload, _ := json.Marshal(map[string]interface{}{
		"SmsSdkAppId":      s.cfg.SdkAppID,
		"SignName":         s.cfg.SignName,
		"TemplateId":       s.cfg.TemplateID,
		"TemplateParamSet": []string{code},
		"PhoneNumberSet":   []string{"+86" + phone},
	})

	// 2. 构建规范请求串
	hashedPayload := fmt.Sprintf("%x", sha256.Sum256(payload))
	headers := "content-type;host;x-tc-action"
	canonicalRequest := strings.Join([]string{
		"POST",
		"/",
		"",
		"content-type:application/json\nhost:" + host + "\nx-tc-action:" + strings.ToLower(action) + "\n",
		headers,
		hashedPayload,
	}, "\n")

	// 3. 构建待签字符串
	credentialScope := date + "/" + service + "/tc3_request"
	hashedCanonical := fmt.Sprintf("%x", sha256.Sum256([]byte(canonicalRequest)))
	stringToSign := strings.Join([]string{algorithm, timestamp, credentialScope, hashedCanonical}, "\n")

	// 4. 计算签名
	hmacSHA256 := func(key, data []byte) []byte {
		h := hmac.New(sha256.New, key)
		h.Write(data)
		return h.Sum(nil)
	}
	secretDate := hmacSHA256([]byte("TC3"+s.cfg.SecretKey), []byte(date))
	secretService := hmacSHA256(secretDate, []byte(service))
	secretSigning := hmacSHA256(secretService, []byte("tc3_request"))
	signature := fmt.Sprintf("%x", hmacSHA256(secretSigning, []byte(stringToSign)))

	// 5. 构建 Authorization
	authorization := fmt.Sprintf(
		"%s Credential=%s/%s, SignedHeaders=%s, Signature=%s",
		algorithm, s.cfg.SecretID, credentialScope, headers, signature,
	)

	// 6. 发送 HTTP 请求
	req, err := http.NewRequest("POST", "https://"+host, bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("tencentSMS: 建立请求失败: %w", err)
	}
	req.Header.Set("Authorization", authorization)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Host", host)
	req.Header.Set("X-TC-Action", action)
	req.Header.Set("X-TC-Timestamp", timestamp)
	req.Header.Set("X-TC-Version", version)
	req.Header.Set("X-TC-Region", region)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("tencentSMS: 请求失败: %w", err)
	}
	defer resp.Body.Close()

	var result struct {
		Response struct {
			SendStatusSet []struct {
				Code    string `json:"Code"`
				Message string `json:"Message"`
			} `json:"SendStatusSet"`
			Error *struct {
				Code    string `json:"Code"`
				Message string `json:"Message"`
			} `json:"Error"`
		} `json:"Response"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return fmt.Errorf("tencentSMS: 解析响应失败: %w", err)
	}
	if result.Response.Error != nil {
		return fmt.Errorf("tencentSMS: API错误 [%s] %s",
			result.Response.Error.Code, result.Response.Error.Message)
	}
	if len(result.Response.SendStatusSet) > 0 && result.Response.SendStatusSet[0].Code != "Ok" {
		return fmt.Errorf("tencentSMS: 发送失败 [%s] %s",
			result.Response.SendStatusSet[0].Code,
			result.Response.SendStatusSet[0].Message)
	}

	logger.Info("[SMS] 短信发送成功（腾讯云）", zap.String("phone", maskPhone(phone)))
	return nil
}

// sendViaAliyun 阿里云短信发送（HMAC-SHA1 签名，无需外部 SDK）
//
// 所需配置字段：SecretID（AccessKeyId）、SecretKey、SignName、TemplateID（TemplateCode）
// 文档：https://help.aliyun.com/document_detail/101414.html
func (s *Service) sendViaAliyun(phone, code string) error {
	now := time.Now().UTC()
	nonce := fmt.Sprintf("%d", now.UnixNano())
	timestampStr := now.Format("2006-01-02T15:04:05Z")

	params := map[string]string{
		"AccessKeyId":      s.cfg.SecretID,
		"Action":           "SendSms",
		"Format":           "JSON",
		"PhoneNumbers":     phone,
		"SignatureMethod":  "HMAC-SHA1",
		"SignatureNonce":   nonce,
		"SignatureVersion": "1.0",
		"SignName":         s.cfg.SignName,
		"TemplateCode":     s.cfg.TemplateID,
		"TemplateParam":    `{"code":"` + code + `"}`,
		"Timestamp":        timestampStr,
		"Version":          "2017-05-25",
	}

	// 按字母顶序排序参数
	keys := make([]string, 0, len(params))
	for k := range params {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	pairs := make([]string, 0, len(keys))
	for _, k := range keys {
		pairs = append(pairs, url.QueryEscape(k)+"="+url.QueryEscape(params[k]))
	}
	canonical := strings.Join(pairs, "&")
	stringToSign := "GET&" + url.QueryEscape("/") + "&" + url.QueryEscape(canonical)

	mac := hmac.New(sha1.New, []byte(s.cfg.SecretKey+"&"))
	mac.Write([]byte(stringToSign))
	sig := base64.StdEncoding.EncodeToString(mac.Sum(nil))

	reqURL := "https://dysmsapi.aliyuncs.com/?" + canonical + "&Signature=" + url.QueryEscape(sig)
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(reqURL)
	if err != nil {
		return fmt.Errorf("aliyunSMS: 请求失败: %w", err)
	}
	defer resp.Body.Close()

	var result struct {
		Code    string `json:"Code"`
		Message string `json:"Message"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return fmt.Errorf("aliyunSMS: 解析响应失败: %w", err)
	}
	if result.Code != "OK" {
		return fmt.Errorf("aliyunSMS: 发送失败 [%s] %s", result.Code, result.Message)
	}

	logger.Info("[SMS] 短信发送成功（阿里云）", zap.String("phone", maskPhone(phone)))
	return nil
}

// generateCode 生成指定位数的随机数字验证码
func generateCode(length int) (string, error) {
	max := new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(length)), nil)
	n, err := rand.Int(rand.Reader, max)
	if err != nil {
		return "", err
	}
	// 补零对齐
	return fmt.Sprintf(fmt.Sprintf("%%0%dd", length), n), nil
}

// maskPhone 遮盖手机号中间 4 位（用于日志）
func maskPhone(phone string) string {
	if len(phone) < 7 {
		return phone
	}
	return phone[:3] + "****" + phone[len(phone)-4:]
}

// ===== 包级别便捷函数 =====

// Send 使用全局服务发送短信验证码
func Send(ctx context.Context, phone string) (string, error) {
	return Get().Send(ctx, phone)
}

// Verify 使用全局服务验证短信验证码
func Verify(ctx context.Context, phone, code string) error {
	return Get().Verify(ctx, phone, code)
}
