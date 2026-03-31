package metrics

import (
	"context"
	"github.com/anxiwuyanzu/openscraper-framework/spider-common-go/v4/dot"
	"github.com/anxiwuyanzu/openscraper-framework/spider-common-go/v4/exp/watchman/common"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	"sync/atomic"
	"time"
)

var (
	register         bool
	meter            metric.Meter
	providerShutdown func(context.Context) error
)

// Setup 启用并初始化 Metric 指标记录功能
func Setup(cfg *common.Config) {
	res, err := common.NewResources(cfg.Mission, cfg.Business)
	if err != nil {
		dot.Logger().WithError(err).Panic("failed to setup metrics engine")
	}

	var exp sdkmetric.Exporter
	switch cfg.ExporterType {
	case common.ExporterConsole:
		exp, err = newConsoleExporter()
	case common.ExporterGRPC:
		exp, err = newGRPCExporter(cfg.ExporterEndpoint, cfg.ExporterHeader)
	case common.ExporterHTTP:
		exp, err = newHTTPExporter(cfg.ExporterEndpoint, cfg.ExporterEndpointPath, cfg.ExporterHeader)
	}
	if err != nil {
		dot.Logger().WithError(err).Panic("failed to create metrics exporter")
	}

	setMeterProvider(res, exp, cfg.MetricSendInterval)
	meter = otel.GetMeterProvider().Meter("")
	register = true
}

// Shutdown 停止 Metric 指标记录功能
func Shutdown() {
	if register {
		_ = providerShutdown(context.Background())
	}
}

// RegisterGauge 注册一个计数器, 如成功率计数器、平均耗时计数器
func RegisterGauge(ptr *int64, unit, name string, attrs ...attribute.KeyValue) error {
	var err error
	if len(unit) != 0 {
		_, err = meter.Int64ObservableGauge(
			name,
			metric.WithUnit(unit),
			metric.WithInt64Callback(func(ctx context.Context, observer metric.Int64Observer) error {
				val := *ptr
				observer.Observe(val, metric.WithAttributes(attrs...))
				atomic.StoreInt64(ptr, 0)
				return nil
			}),
		)
	} else {
		_, err = meter.Int64ObservableGauge(name, metric.WithInt64Callback(func(ctx context.Context, observer metric.Int64Observer) error {
			val := *ptr
			observer.Observe(val, metric.WithAttributes(attrs...))
			atomic.StoreInt64(ptr, 0)
			return nil
		}))
	}

	return err
}

func setMeterProvider(res *resource.Resource, exp sdkmetric.Exporter, interval time.Duration) {
	var meterProvider *sdkmetric.MeterProvider
	meterProvider = sdkmetric.NewMeterProvider(
		sdkmetric.WithResource(res),
		sdkmetric.WithReader(sdkmetric.NewPeriodicReader(
			exp,
			sdkmetric.WithTimeout(15*time.Second),
			sdkmetric.WithInterval(interval)),
		),
	)
	otel.SetMeterProvider(meterProvider)
	providerShutdown = meterProvider.Shutdown
}
