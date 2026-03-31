package trace

import (
	"context"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"os"
	"time"
)

// NewConsoleExporter 创建一个输出到stdout的OTel Exporter
func newConsoleExporter() (sdktrace.SpanExporter, error) {
	return stdouttrace.New(
		stdouttrace.WithWriter(os.Stdout),
		stdouttrace.WithPrettyPrint(),
	)
}

// NewHTTPExporter 创建一个以HTTP协议输出到Collector的OTel Exporter
func newHTTPExporter(host, path string, headers map[string]string) (sdktrace.SpanExporter, error) {
	return otlptracehttp.New(
		context.Background(),
		otlptracehttp.WithEndpoint(host),
		otlptracehttp.WithURLPath(path),
		otlptracehttp.WithHeaders(headers),
		otlptracehttp.WithCompression(otlptracehttp.GzipCompression),
		otlptracehttp.WithTimeout(15*time.Second),
		otlptracehttp.WithInsecure(),
	)
}

// NewGRPCExporter 创建一个以GRPC协议输出到Collector的OTel Exporter
func newGRPCExporter(host string, headers map[string]string) (sdktrace.SpanExporter, error) {
	return otlptracegrpc.New(
		context.Background(),
		otlptracegrpc.WithEndpoint(host),
		otlptracegrpc.WithHeaders(headers),
		otlptracegrpc.WithTimeout(15*time.Second),
		otlptracegrpc.WithInsecure(),
	)
}
