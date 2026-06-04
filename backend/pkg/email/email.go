// Package email 提供邮箱验证码的发送和验证功能。
// 验证码为 6 位数字，存储在 Redis 中，默认 5 分钟过期。
// 同一邮箱 60 秒内不允许重复发送。
package email

import (
	"context"
	"crypto/rand"
	"crypto/tls"
	"fmt"
	"math/big"
	"net"
	"net/smtp"
	"time"

	"github.com/gamero/gamero/config"
	"github.com/gamero/gamero/pkg/cache"
	apperrors "github.com/gamero/gamero/pkg/errors"
	"github.com/gamero/gamero/pkg/logger"
	"go.uber.org/zap"
)

const (
	// codeKeyPrefix 验证码存储 Redis key 前缀
	codeKeyPrefix = "email:code:"
	// intervalKeyPrefix 发送间隔 Redis key 前缀
	intervalKeyPrefix = "email:interval:"
	// attemptKeyPrefix 失败次数 Redis key 前缀
	attemptKeyPrefix = "email:attempt:"
	// codeLength 验证码位数
	codeLength = 6
	// maxAttempts 验证码最大允许错误次数
	maxAttempts = 5
)

// global 是全局邮件服务实例
var global *Service

// Service 邮件服务
type Service struct {
	cfg config.EmailConfig
}

// Init 初始化全局邮件服务
func Init(cfg config.EmailConfig) *Service {
	global = &Service{cfg: cfg}
	return global
}

// Get 返回全局邮件服务
func Get() *Service {
	if global == nil {
		panic("邮件服务未初始化，请先调用 email.Init()")
	}
	return global
}

// SendVerifyCode 发送邮箱验证码
// 如果该邮箱在 interval 秒内已发送，返回频率限制错误
// 返回值：验证码字符串，错误
func (s *Service) SendVerifyCode(ctx context.Context, toEmail string) (string, error) {
	codeKey := codeKeyPrefix + toEmail
	intervalKey := intervalKeyPrefix + toEmail

	// 检查发送间隔
	exists, err := cache.Exists(ctx, intervalKey)
	if err != nil {
		return "", fmt.Errorf("检查发送间隔失败: %w", err)
	}
	if exists {
		return "", apperrors.CodeError(apperrors.CodeEmailFrequency)
	}

	// 生成 6 位随机验证码
	code, err := generateCode(codeLength)
	if err != nil {
		return "", fmt.Errorf("生成验证码失败: %w", err)
	}

	// 存储验证码到 Redis
	expireDuration := time.Duration(s.cfg.Expire) * time.Second
	if err := cache.Set(ctx, codeKey, code, expireDuration); err != nil {
		return "", fmt.Errorf("存储验证码失败: %w", err)
	}

	// 设置发送间隔锁
	intervalDuration := time.Duration(s.cfg.Interval) * time.Second
	if err := cache.Set(ctx, intervalKey, "1", intervalDuration); err != nil {
		return "", fmt.Errorf("设置发送间隔失败: %w", err)
	}

	if s.cfg.Host != "" && s.cfg.Host != "smtp.example.com" {
		if err := s.sendEmail(toEmail, "Gamero 验证码", buildVerifyCodeBody(code, s.cfg.Expire)); err != nil {
			// 发送失败时清理
			if err := cache.Del(ctx, codeKey, intervalKey); err != nil {
				logger.Warn("failed to delete cache", zap.Error(err))
			}
			return "", apperrors.WrapMsg(apperrors.CodeEmailSendFailed, "邮件发送失败", err)
		}
		logger.Info("邮箱验证码已发送", zap.String("email", maskEmail(toEmail)))
		return code, nil
	}

	// 未配置 SMTP 时：直接在控制台输出验证码。
	fmt.Printf("===== 新验证码 = %s (邮箱: %s) =====\n", code, maskEmail(toEmail))
	logger.Info("邮箱验证码已发送", zap.String("email", maskEmail(toEmail)))
	return code, nil
}

// Verify 验证邮箱验证码
// 验证成功后立即删除验证码（一次性）
// 单个验证码最多尝试 maxAttempts 次，超出后删除验证码防止暴力枚举
func (s *Service) Verify(ctx context.Context, email, code string) error {
	codeKey := codeKeyPrefix + email
	attemptKey := attemptKeyPrefix + email

	storedCode, err := cache.GetString(ctx, codeKey)
	if err != nil {
		return apperrors.CodeError(apperrors.CodeEmailCodeExpired)
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
		return apperrors.CodeError(apperrors.CodeEmailCodeExpired)
	}

	if storedCode != code {
		return apperrors.CodeError(apperrors.CodeEmailCodeInvalid)
	}

	// 验证成功，删除验证码和错误计数
	if err := cache.Del(ctx, codeKey, attemptKey); err != nil {
		logger.Warn("failed to delete cache", zap.Error(err))
	}
	return nil
}

// sendEmail 通过 SMTP 发送邮件
func (s *Service) sendEmail(to, subject, body string) error {
	host := s.cfg.Host
	port := s.cfg.Port
	username := s.cfg.Username
	password := s.cfg.Password
	from := s.cfg.From

	// 构建邮件内容
	msg := fmt.Sprintf(
		"From: %s\r\nTo: %s\r\nSubject: %s\r\nContent-Type: text/html; charset=UTF-8\r\n\r\n%s",
		from, to, subject, body,
	)

	addr := net.JoinHostPort(host, fmt.Sprintf("%d", port))
	mailFrom := username
	if mailFrom == "" {
		mailFrom = "noreply@gamero.local"
	}

	if s.cfg.SSL {
		// SSL/TLS 连接
		tlsCfg := &tls.Config{
			InsecureSkipVerify: false,
			ServerName:         host,
		}

		conn, err := tls.Dial("tcp", addr, tlsCfg)
		if err != nil {
			return fmt.Errorf("TLS 连接失败: %w", err)
		}
		defer conn.Close()

		client, err := smtp.NewClient(conn, host)
		if err != nil {
			return fmt.Errorf("创建 SMTP 客户端失败: %w", err)
		}
		defer client.Close()

		if username != "" {
			auth := smtp.PlainAuth("", username, password, host)
			if err := client.Auth(auth); err != nil {
				return fmt.Errorf("SMTP 认证失败: %w", err)
			}
		}

		if err := client.Mail(mailFrom); err != nil {
			return fmt.Errorf("设置发件人失败: %w", err)
		}

		if err := client.Rcpt(to); err != nil {
			return fmt.Errorf("设置收件人失败: %w", err)
		}

		w, err := client.Data()
		if err != nil {
			return fmt.Errorf("获取邮件 Data writer 失败: %w", err)
		}

		if _, err := fmt.Fprint(w, msg); err != nil {
			return fmt.Errorf("写入邮件内容失败: %w", err)
		}

		return w.Close()
	}

	// STARTTLS 或普通 SMTP
	conn, err := net.DialTimeout("tcp", addr, 10*time.Second)
	if err != nil {
		return fmt.Errorf("连接 SMTP 服务器失败: %w", err)
	}
	defer conn.Close()

	client, err := smtp.NewClient(conn, host)
	if err != nil {
		return fmt.Errorf("创建 SMTP 客户端失败: %w", err)
	}
	defer client.Close()

	// 尝试 STARTTLS
	if ok, _ := client.Extension("STARTTLS"); ok {
		cfg := &tls.Config{ServerName: host}
		if err := client.StartTLS(cfg); err != nil {
			return fmt.Errorf("STARTTLS 失败: %w", err)
		}
	}

	if username != "" {
		auth := smtp.PlainAuth("", username, password, host)
		if err := client.Auth(auth); err != nil {
			return fmt.Errorf("SMTP 认证失败: %w", err)
		}
	}

	if err := client.Mail(mailFrom); err != nil {
		return fmt.Errorf("设置发件人失败: %w", err)
	}

	if err := client.Rcpt(to); err != nil {
		return fmt.Errorf("设置收件人失败: %w", err)
	}

	w, err := client.Data()
	if err != nil {
		return fmt.Errorf("获取邮件 Data writer 失败: %w", err)
	}

	if _, err := fmt.Fprint(w, msg); err != nil {
		return fmt.Errorf("写入邮件内容失败: %w", err)
	}

	return w.Close()
}

// buildVerifyCodeBody 构建验证码邮件 HTML 内容
func buildVerifyCodeBody(code string, expireSeconds int) string {
	expireMinutes := expireSeconds / 60
	return fmt.Sprintf(`
<!DOCTYPE html>
<html>
<head><meta charset="UTF-8"></head>
<body style="font-family: Arial, sans-serif; max-width: 600px; margin: 0 auto; padding: 20px;">
  <h2 style="color: #333;">Gamero 验证码</h2>
  <p>您的验证码为：</p>
  <div style="font-size: 32px; font-weight: bold; color: #FF6B35; letter-spacing: 8px; padding: 20px; background: #f5f5f5; text-align: center; border-radius: 8px;">
    %s
  </div>
  <p style="color: #666; margin-top: 20px;">验证码有效期 <strong>%d 分钟</strong>，请勿泄露给他人。</p>
  <p style="color: #999; font-size: 12px;">如果这不是您的操作，请忽略此邮件。</p>
  <hr style="border: none; border-top: 1px solid #eee;">
  <p style="color: #999; font-size: 12px;">此邮件由 Gamero 系统自动发送，请勿直接回复。</p>
</body>
</html>
`, code, expireMinutes)
}

// generateCode 生成指定位数的随机数字验证码
func generateCode(length int) (string, error) {
	max := new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(length)), nil)
	n, err := rand.Int(rand.Reader, max)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf(fmt.Sprintf("%%0%dd", length), n), nil
}

// maskEmail 遮盖邮箱（用于日志）
func maskEmail(email string) string {
	for i, c := range email {
		if c == '@' {
			if i <= 2 {
				return email[:1] + "***" + email[i:]
			}
			return email[:2] + "***" + email[i:]
		}
	}
	return email
}

// ===== 包级别便捷函数 =====

// SendVerifyCode 使用全局服务发送邮箱验证码
func SendVerifyCode(ctx context.Context, toEmail string) (string, error) {
	return Get().SendVerifyCode(ctx, toEmail)
}

// Verify 使用全局服务验证邮箱验证码
func Verify(ctx context.Context, email, code string) error {
	return Get().Verify(ctx, email, code)
}
