package trace

import (
	"context"
	"github.com/anxiwuyanzu/openscraper-framework/spider-common-go/v4/dot"
	"github.com/anxiwuyanzu/openscraper-framework/spider-common-go/v4/exp/watchman/common"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
)

var (
	register               bool
	tracerProviderShutdown func(context.Context) error
	tracer                 trace.Tracer
)

// Setup setups OTel tracer
func Setup(cfg *common.Config) {
	res, err := common.NewResources(cfg.Mission, cfg.Business)
	if err != nil {
		dot.Logger().WithError(err).Panic("failed to setup trace engine")
	}

	var exp sdktrace.SpanExporter
	switch cfg.ExporterType {
	case common.ExporterConsole:
		exp, err = newConsoleExporter()
	case common.ExporterGRPC:
		exp, err = newGRPCExporter(cfg.ExporterEndpoint, cfg.ExporterHeader)
	case common.ExporterHTTP:
		exp, err = newHTTPExporter(cfg.ExporterEndpoint, cfg.ExporterEndpointPath, cfg.ExporterHeader)
	}
	if err != nil {
		dot.Logger().WithError(err).Panic("failed to create trace exporter")
	}

	setTraceProvider(exp, res)
	tracer = otel.GetTracerProvider().Tracer("")
	register = true
}

func Shutdown() {
	if register {
		_ = tracerProviderShutdown(context.Background())
	}
}

// NewSpan create a span from tracer and return the span's context.
// rawCtx: used as the span's foundation, for example, when rawCtx is spider.Context, created span will contain dot.Item and dot.Value
// returned context is used to get the span from a different place. If you keep the span yourself(for example,
// save it in spider.Context's Values), or the span is ended inside current scope, you don't need the save it(use _ to ignore it)
func NewSpan(rawCtx context.Context, spanName string) (context.Context, *Span) {
	if !register {
		dot.Logger().Panic("trace provider not registered")
	}

	newCtx, span := tracer.Start(rawCtx, spanName)
	return newCtx, &Span{span: span}
}

// GetSpanByContext use NewSpan returned context to find the corresponding span
func GetSpanByContext(c context.Context) *Span {
	if !register {
		dot.Logger().Panic("trace provider not registered")
	}
	return &Span{span: trace.SpanFromContext(c)}
}

func setTraceProvider(exporter sdktrace.SpanExporter, res *resource.Resource) {
	traceProvider := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(res),
	)

	otel.SetTracerProvider(traceProvider)
	tracerProviderShutdown = traceProvider.Shutdown
}
