// Package tracing 提供 OpenTelemetry 链路追踪初始化功能。
// 开发环境默认输出到 stdout；生产环境可替换为 Jaeger/OTLP exporter。
package tracing

import (
	"context"
	"fmt"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
	"go.uber.org/zap"
)

// TracerConfig 链路追踪配置（与 config.TracingConfig 字段保持一致，避免循环依赖）
type TracerConfig struct {
	Enabled     bool    `mapstructure:"enabled"`
	ServiceName string  `mapstructure:"service_name"`
	SampleRate  float64 `mapstructure:"sample_rate"` // 0.0~1.0，默认 0.1
	// Exporter: "stdout" | "jaeger" | "zipkin"（当前只实现 stdout）
	Exporter string `mapstructure:"exporter"`
	Endpoint string `mapstructure:"endpoint"` // Jaeger/Zipkin 地址（预留）
}

// Init 初始化全局 TracerProvider。
// 返回 shutdown 函数，调用方应在程序退出时调用以刷新并关闭 exporter。
func Init(cfg TracerConfig, logger *zap.Logger) (shutdown func(context.Context) error, err error) {
	if !cfg.Enabled {
		// 未启用时返回 noop shutdown，全局 provider 保持默认 noop 实现
		return func(ctx context.Context) error { return nil }, nil
	}

	sampleRate := cfg.SampleRate
	if sampleRate <= 0 {
		sampleRate = 0.1
	}
	if sampleRate > 1.0 {
		sampleRate = 1.0
	}

	res, err := resource.Merge(
		resource.Default(),
		resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceName(cfg.ServiceName),
			semconv.ServiceVersion("1.0.0"),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("tracing: create resource failed: %w", err)
	}

	// stdout exporter（开发调试用；生产环境替换为 OTLP/Jaeger exporter）
	exporter, err := stdouttrace.New(stdouttrace.WithPrettyPrint())
	if err != nil {
		return nil, fmt.Errorf("tracing: create stdout exporter failed: %w", err)
	}

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(res),
		sdktrace.WithSampler(sdktrace.TraceIDRatioBased(sampleRate)),
	)

	otel.SetTracerProvider(tp)
	logger.Info("OpenTelemetry 链路追踪已启用",
		zap.String("service", cfg.ServiceName),
		zap.Float64("sample_rate", sampleRate),
		zap.String("exporter", cfg.Exporter),
	)

	return tp.Shutdown, nil
}
