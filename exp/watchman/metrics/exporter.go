package metrics

import (
	"context"
	"encoding/json"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp"
	"go.opentelemetry.io/otel/exporters/stdout/stdoutmetric"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	"os"
	"time"
)

func newConsoleExporter() (sdkmetric.Exporter, error) {
	return stdoutmetric.New(
		stdoutmetric.WithoutTimestamps(),
		stdoutmetric.WithEncoder(json.NewEncoder(os.Stdout)),
		stdoutmetric.WithTemporalitySelector(temporalityDeltaSelector),
	)
}

func newHTTPExporter(host, path string, headers map[string]string) (sdkmetric.Exporter, error) {
	return otlpmetrichttp.New(nil,
		otlpmetrichttp.WithEndpoint(host),
		otlpmetrichttp.WithURLPath(path),
		otlpmetrichttp.WithHeaders(headers),
		otlpmetrichttp.WithCompression(otlpmetrichttp.GzipCompression),
		otlpmetrichttp.WithTimeout(15*time.Second),
		otlpmetrichttp.WithInsecure(),
		otlpmetrichttp.WithTemporalitySelector(temporalityDeltaSelector),
	)
}

func newGRPCExporter(host string, headers map[string]string) (sdkmetric.Exporter, error) {
	return otlpmetricgrpc.New(context.Background(),
		otlpmetricgrpc.WithEndpoint(host),
		otlpmetricgrpc.WithHeaders(headers),
		otlpmetricgrpc.WithCompressor("gzip"),
		otlpmetricgrpc.WithTimeout(15*time.Second),
		otlpmetricgrpc.WithInsecure(),
		otlpmetricgrpc.WithTemporalitySelector(temporalityDeltaSelector),
	)
}

func temporalityDeltaSelector(_ sdkmetric.InstrumentKind) metricdata.Temporality {
	return metricdata.DeltaTemporality
}
