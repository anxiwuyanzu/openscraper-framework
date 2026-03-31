package common

import (
	"context"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.17.0"
)

// NewResources name 可以作为爬虫名称, 如author, dy-live, keyword14-cn, namespace可以作为爬虫类型, 如xhs,cds,cqq
// resource 类似于metadata, 它储存的所有信息都会传给span
// 最基本的 namespace和name 由参数传入, 其余自定义属性使用环境变量 "OTEL_RESOURCE_ATTRIBUTES" 指定,
// 以"K=V"格式书写, 并以","分隔. 如:
// OTEL_RESOURCE_ATTRIBUTES="IP=1.1.1.1, Port=80, Path=/localhost"
func NewResources(name, namespace string) (*resource.Resource, error) {
	res, err := resource.New(
		context.Background(),
		resource.WithFromEnv(),
		resource.WithHost(),
		resource.WithOS(),
		resource.WithProcess(),
		resource.WithContainer())
	if err != nil {
		return nil, err
	}
	res, err = resource.Merge(
		res,
		resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceName(name),
			semconv.ServiceNamespace(namespace),
		),
	)
	if err != nil {
		return nil, err
	}
	return res, nil
}
