// Package config 提供应用程序配置管理功能。
// 使用 viper 读取 YAML 配置文件，支持通过环境变量覆盖配置项。
// 环境变量命名规则：将配置路径的点和减号替换为下划线，并转为大写，加前缀 GAMERO_。
// 例如：database.host -> GAMERO_DATABASE_HOST
package config

import (
	"fmt"
	"strings"
	"time"

	"github.com/spf13/viper"
)

// Config 是应用程序的总配置结构体
type Config struct {
	Server        ServerConfig        `mapstructure:"server"`
	Database      DatabaseConfig      `mapstructure:"database"`
	Redis         RedisConfig         `mapstructure:"redis"`
	Storage       StorageConfig       `mapstructure:"storage"`
	JWT           JWTConfig           `mapstructure:"jwt"`
	SMS           SMSConfig           `mapstructure:"sms"`
	Email         EmailConfig         `mapstructure:"email"`
	Logger        LoggerConfig        `mapstructure:"logger"`
	RateLimit     RateLimitConfig     `mapstructure:"rate_limit"`
	MQ            MQConfig            `mapstructure:"mq"`
	Feed          FeedConfig          `mapstructure:"feed"`
	Elasticsearch ElasticsearchConfig `mapstructure:"elasticsearch"`
	Observability ObservabilityConfig `mapstructure:"observability"`
	Features      FeaturesConfig      `mapstructure:"features"`
	Moderation    ModerationConfig    `mapstructure:"moderation"`
}

// ElasticsearchConfig ES 配置
type ElasticsearchConfig struct {
	Enabled     bool     `mapstructure:"enabled"`
	Addresses   []string `mapstructure:"addresses"`
	Username    string   `mapstructure:"username"`
	Password    string   `mapstructure:"password"`
	IndexPrefix string   `mapstructure:"index_prefix"`
}

// FeedConfig Feed 流配置
type FeedConfig struct {
	FanoutThreshold    int `mapstructure:"fanout_threshold"`    // 粉丝数超过此值走读合并（大V），默认 1000
	MaxFeedLen         int `mapstructure:"max_feed_len"`        // 每用户 Feed ZSet 最大条数，默认 500
	HeavyFanoutWorkers int `mapstructure:"heavy_fanout_workers"` // 批量写扩散并发 worker（预留），默认 10
}

// MQConfig 消息队列配置
type MQConfig struct {
	Provider string   `mapstructure:"provider"` // memory / kafka
	Brokers  []string `mapstructure:"brokers"`  // Kafka broker 地址列表
	BufSize  int      `mapstructure:"buf_size"` // 内存队列缓冲大小（memory 模式）
	// Kafka 专属配置
	GroupID        string `mapstructure:"group_id"`        // 消费者组 ID，默认 "gamero"
	TopicPrefix    string `mapstructure:"topic_prefix"`    // Topic 前缀，默认 "gamero."
	NumPartitions  int    `mapstructure:"num_partitions"`  // 自动创建 Topic 时的分区数，默认 3
	ReplicationFactor int `mapstructure:"replication_factor"` // 副本数，默认 1
	// 生产者配置
	BatchSize    int `mapstructure:"batch_size"`     // 批量提交大小，默认 100
	BatchTimeout int `mapstructure:"batch_timeout"`  // 批量提交超时（毫秒），默认 100
	// 消费者配置
	MinBytes    int `mapstructure:"min_bytes"`    // 最小拉取字节，默认 1
	MaxBytes    int `mapstructure:"max_bytes"`    // 最大拉取字节，默认 10MB
	MaxWaitMs   int `mapstructure:"max_wait_ms"`  // 最长等待（毫秒），默认 500
	CommitInterval int `mapstructure:"commit_interval"` // 自动提交间隔（毫秒），默认 1000
}

// ServerConfig HTTP 服务器配置
type ServerConfig struct {
	Port           int      `mapstructure:"port"`
	Mode           string   `mapstructure:"mode"`
	ReadTimeout    int      `mapstructure:"read_timeout"`
	WriteTimeout   int      `mapstructure:"write_timeout"`
	MaxHeaderBytes int      `mapstructure:"max_header_bytes"`
	// AllowOrigins 允许跨域/WebSocket 的来源域名列表。
	// 生产环境应填写具体域名，如 ["https://gamero.example.com"]。
	// 留空或填 ["*"] 时退回到允许所有来源（仅限开发环境）。
	AllowOrigins   []string `mapstructure:"allow_origins"`
}

// DatabaseConfig PostgreSQL 数据库配置
type DatabaseConfig struct {
	Host            string   `mapstructure:"host"`
	Port            int      `mapstructure:"port"`
	User            string   `mapstructure:"user"`
	Password        string   `mapstructure:"password"`
	DBName          string   `mapstructure:"dbname"`
	SSLMode         string   `mapstructure:"sslmode"`
	Timezone        string   `mapstructure:"timezone"`
	MaxIdleConns    int      `mapstructure:"max_idle_conns"`
	MaxOpenConns    int      `mapstructure:"max_open_conns"`
	ConnMaxLifetime int      `mapstructure:"conn_max_lifetime"`
	// ReplicaDSNs 只读副本 DSN 列表（为空时不启用读写分离）
	ReplicaDSNs     []string `mapstructure:"replica_dsns"`
}

// DSN 返回 PostgreSQL 连接字符串
func (d DatabaseConfig) DSN() string {
	return fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s TimeZone=%s",
		d.Host, d.Port, d.User, d.Password, d.DBName, d.SSLMode, d.Timezone,
	)
}

// RedisConfig Redis 连接配置
type RedisConfig struct {
	Host         string `mapstructure:"host"`
	Port         int    `mapstructure:"port"`
	Password     string `mapstructure:"password"`
	DB           int    `mapstructure:"db"`
	PoolSize     int    `mapstructure:"pool_size"`
	MinIdleConns int    `mapstructure:"min_idle_conns"`
	DialTimeout  int    `mapstructure:"dial_timeout"`
	ReadTimeout  int    `mapstructure:"read_timeout"`
	WriteTimeout int    `mapstructure:"write_timeout"`
}

// Addr 返回 Redis 地址字符串
func (r RedisConfig) Addr() string {
	return fmt.Sprintf("%s:%d", r.Host, r.Port)
}

// StorageConfig S3 兼容对象存储配置（支持腾讯云 COS、阿里云 OSS、AWS S3 等）
type StorageConfig struct {
	Provider        string `mapstructure:"provider"`          // cos | oss | s3
	Endpoint        string `mapstructure:"endpoint"`          // S3 兼容 endpoint，如 https://cos.ap-guangzhou.myqcloud.com
	Region          string `mapstructure:"region"`            // 地域
	AccessKeyID     string `mapstructure:"access_key_id"`
	SecretAccessKey string `mapstructure:"secret_access_key"`
	BucketName      string `mapstructure:"bucket_name"`
	BaseURL         string `mapstructure:"base_url"`          // 公开访问基础 URL（CDN 或存储域名）
	ForcePathStyle  bool   `mapstructure:"force_path_style"` // 路径风格，默认 false
}

// JWTConfig JWT 令牌配置
type JWTConfig struct {
	AccessSecret  string `mapstructure:"access_secret"`
	RefreshSecret string `mapstructure:"refresh_secret"`
	AccessExpire  int    `mapstructure:"access_expire"`
	RefreshExpire int    `mapstructure:"refresh_expire"`
}

// AccessExpireDuration 返回 Access Token 过期时长
func (j JWTConfig) AccessExpireDuration() time.Duration {
	return time.Duration(j.AccessExpire) * time.Second
}

// RefreshExpireDuration 返回 Refresh Token 过期时长
func (j JWTConfig) RefreshExpireDuration() time.Duration {
	return time.Duration(j.RefreshExpire) * time.Second
}

// SMSConfig 短信服务配置
type SMSConfig struct {
	Provider   string `mapstructure:"provider"`
	SecretID   string `mapstructure:"secret_id"`
	SecretKey  string `mapstructure:"secret_key"`
	SdkAppID   string `mapstructure:"sdk_app_id"`
	SignName   string `mapstructure:"sign_name"`
	TemplateID string `mapstructure:"template_id"`
	Expire     int    `mapstructure:"expire"`
	Interval   int    `mapstructure:"interval"`
}

// EmailConfig 邮件服务配置
type EmailConfig struct {
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	Username string `mapstructure:"username"`
	Password string `mapstructure:"password"`
	From     string `mapstructure:"from"`
	SSL      bool   `mapstructure:"ssl"`
	Expire   int    `mapstructure:"expire"`
	Interval int    `mapstructure:"interval"`
}

// LoggerConfig 日志配置
type LoggerConfig struct {
	Level      string `mapstructure:"level"`
	Format     string `mapstructure:"format"`
	OutputPath string `mapstructure:"output_path"`
	MaxSize    int    `mapstructure:"max_size"`
	MaxBackups int    `mapstructure:"max_backups"`
	MaxAge     int    `mapstructure:"max_age"`
	Compress   bool   `mapstructure:"compress"`
}

// RateLimitConfig 限流配置
type RateLimitConfig struct {
	Enabled   bool    `mapstructure:"enabled"`
	Rate      float64 `mapstructure:"rate"`
	Burst     int     `mapstructure:"burst"`
	KeyPrefix string  `mapstructure:"key_prefix"`
}

// ObservabilityConfig 可观测性配置（Prometheus 指标 + OpenTelemetry 链路追踪）
type ObservabilityConfig struct {
	Metrics MetricsConfig `mapstructure:"metrics"`
	Tracing TracingConfig `mapstructure:"tracing"`
}

// MetricsConfig Prometheus 指标暴露配置
type MetricsConfig struct {
	Enabled bool   `mapstructure:"enabled"`
	Path    string `mapstructure:"path"` // 默认 /metrics
}

// TracingConfig OpenTelemetry 链路追踪配置
// 与 pkg/tracing.TracerConfig 保持字段同步，分开定义以避免循环依赖
type TracingConfig struct {
	Enabled     bool    `mapstructure:"enabled"`
	ServiceName string  `mapstructure:"service_name"`
	SampleRate  float64 `mapstructure:"sample_rate"` // 0.0~1.0，默认 0.1
	Exporter    string  `mapstructure:"exporter"`    // stdout | jaeger | zipkin
	Endpoint    string  `mapstructure:"endpoint"`    // Jaeger/Zipkin 地址
}

// ModerationConfig 内容审核回调配置
type ModerationConfig struct {
	// CallbackSecret 用于验证腾讯云 COS 审核回调签名的预共享密钥
	CallbackSecret string `mapstructure:"callback_secret"`
	// AllowedCallbackIPs 腾讯云回调 IP 白名单，逗号分隔
	AllowedCallbackIPs string `mapstructure:"allowed_callback_ips"`
}

// global 是全局配置实例
var global *Config

// Load 从指定路径加载配置文件，支持环境变量覆盖
func Load(configPath string) (*Config, error) {
	v := viper.New()

	// 设置配置文件路径
	v.SetConfigFile(configPath)
	v.SetConfigType("yaml")

	// 支持环境变量覆盖，前缀为 GAMERO
	v.SetEnvPrefix("GAMERO")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_", "-", "_"))
	v.AutomaticEnv()

	// 读取配置文件
	if err := v.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("读取配置文件失败: %w", err)
	}

	cfg := &Config{}
	if err := v.Unmarshal(cfg); err != nil {
		return nil, fmt.Errorf("解析配置文件失败: %w", err)
	}

	global = cfg
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("配置校验失败: %w", err)
	}
	return cfg, nil
}

// Validate 校验关键配置项，防止以占位符或空值启动生产服务。
// 以下情况会返回 error：
//   - DB/Redis 密码为空且 DSN 也为空
//   - JWT secret 含占位符（由 auth.Init 额外 panic 兜底，此处提前拦截）
//   - 支付已启用但 payment.provider 无对应密钥
func (c *Config) Validate() error {
	var errs []string

	// DB host 必须配置（DSN 由方法动态生成）
	if c.Database.Host == "" {
		errs = append(errs, "database.host 不能为空")
	}

	// Redis host
	if c.Redis.Host == "" {
		errs = append(errs, "redis.host 不能为空")
	}

	// JWT secret 占位符检查
	for _, s := range []string{c.JWT.AccessSecret, c.JWT.RefreshSecret} {
		if strings.Contains(s, "change-in-production") {
			errs = append(errs, "jwt secret 仍为占位符，生产环境必须替换")
			break
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("配置项错误: %s", strings.Join(errs, "; "))
	}
	return nil
}

// Get 返回全局配置实例，如果未初始化则 panic
func Get() *Config {
	if global == nil {
		panic("配置未初始化，请先调用 config.Load()")
	}
	return global
}

// MustLoad 加载配置，失败则 panic
func MustLoad(configPath string) *Config {
	cfg, err := Load(configPath)
	if err != nil {
		panic(fmt.Sprintf("加载配置失败: %v", err))
	}
	return cfg
}

// FeaturesConfig 功能开关预留，当前无活跃开关。
type FeaturesConfig struct {
}
