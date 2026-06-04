// Package logger 提供基于 zap 的结构化日志封装。
// 支持按日期切割日志文件（通过 lumberjack），区分多个日志级别。
// 初始化后可通过包级别函数直接使用，也可获取底层 *zap.Logger 实例。
package logger

import (
	"os"
	"path/filepath"
	"time"

	"github.com/gamero/gamero/config"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

// global 是全局 logger 实例
var global *zap.Logger

// Init 根据配置初始化全局 logger
func Init(cfg config.LoggerConfig) error {
	// 解析日志级别
	level, err := parseLevel(cfg.Level)
	if err != nil {
		return err
	}

	// 配置 encoder（日志格式）
	encoderCfg := zapcore.EncoderConfig{
		TimeKey:        "time",
		LevelKey:       "level",
		NameKey:        "logger",
		CallerKey:      "caller",
		FunctionKey:    zapcore.OmitKey,
		MessageKey:     "msg",
		StacktraceKey:  "stacktrace",
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeLevel:    zapcore.LowercaseLevelEncoder,
		EncodeTime:     customTimeEncoder,
		EncodeDuration: zapcore.SecondsDurationEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
	}

	var encoder zapcore.Encoder
	if cfg.Format == "console" {
		encoderCfg.EncodeLevel = zapcore.CapitalColorLevelEncoder
		encoder = zapcore.NewConsoleEncoder(encoderCfg)
	} else {
		encoder = zapcore.NewJSONEncoder(encoderCfg)
	}

	// 构建 WriteSyncer（输出目标）
	var cores []zapcore.Core

	// 始终输出到 stdout
	consoleCore := zapcore.NewCore(
		zapcore.NewConsoleEncoder(func() zapcore.EncoderConfig {
			consoleCfg := encoderCfg
			consoleCfg.EncodeLevel = zapcore.CapitalColorLevelEncoder
			return consoleCfg
		}()),
		zapcore.AddSync(os.Stdout),
		level,
	)
	cores = append(cores, consoleCore)

	// 如果配置了文件输出，添加文件 writer
	if cfg.OutputPath != "" {
		// 确保日志目录存在
		logDir := filepath.Dir(cfg.OutputPath)
		if err := os.MkdirAll(logDir, 0755); err != nil {
			return err
		}

		fileWriter := &lumberjack.Logger{
			Filename:   cfg.OutputPath,
			MaxSize:    cfg.MaxSize,
			MaxBackups: cfg.MaxBackups,
			MaxAge:     cfg.MaxAge,
			Compress:   cfg.Compress,
			LocalTime:  true,
		}

		fileCore := zapcore.NewCore(encoder, zapcore.AddSync(fileWriter), level)
		cores = append(cores, fileCore)
	}

	// 合并所有 core
	core := zapcore.NewTee(cores...)

	// 创建 logger
	global = zap.New(core,
		zap.AddCaller(),
		zap.AddCallerSkip(1),
		zap.AddStacktrace(zapcore.ErrorLevel),
	)

	return nil
}

// customTimeEncoder 自定义时间格式
func customTimeEncoder(t time.Time, enc zapcore.PrimitiveArrayEncoder) {
	enc.AppendString(t.Format("2006-01-02 15:04:05.000"))
}

// parseLevel 解析日志级别字符串
func parseLevel(levelStr string) (zapcore.Level, error) {
	var level zapcore.Level
	if err := level.UnmarshalText([]byte(levelStr)); err != nil {
		return zapcore.InfoLevel, err
	}
	return level, nil
}

// Get 返回全局 logger 实例
func Get() *zap.Logger {
	if global == nil {
		// 未初始化时返回一个基础 logger（输出到 stdout）
		l, _ := zap.NewDevelopment()
		return l
	}
	return global
}

// Sync 刷新日志缓冲区，程序退出前应调用
func Sync() {
	if global != nil {
		if err := global.Sync(); err != nil {
			// Ignore sync errors on shutdown
		}
	}
}

// Debug 记录 Debug 级别日志
func Debug(msg string, fields ...zap.Field) {
	Get().Debug(msg, fields...)
}

// Info 记录 Info 级别日志
func Info(msg string, fields ...zap.Field) {
	Get().Info(msg, fields...)
}

// Warn 记录 Warn 级别日志
func Warn(msg string, fields ...zap.Field) {
	Get().Warn(msg, fields...)
}

// Error 记录 Error 级别日志
func Error(msg string, fields ...zap.Field) {
	Get().Error(msg, fields...)
}

// Fatal 记录 Fatal 级别日志，然后调用 os.Exit(1)
func Fatal(msg string, fields ...zap.Field) {
	Get().Fatal(msg, fields...)
}

// With 返回携带指定字段的子 logger
func With(fields ...zap.Field) *zap.Logger {
	return Get().With(fields...)
}

// Named 返回命名子 logger
func Named(name string) *zap.Logger {
	return Get().Named(name)
}
