package common

import (
	"time"
)

type exporter string

const (
	ExporterHTTP    exporter = "http"
	ExporterGRPC    exporter = "grpc"
	ExporterConsole exporter = "console"
)

type Config struct {
	ExporterType exporter
	// ExporterEndpoint OTel-Exporter 的地址, 仅包含 HOST:PORT
	ExporterEndpoint string
	// ExporterEndpointPath 设置 OTel-Exporter 的Path, 一般不需要修改, 按照Readme里的设置即可. gRPC 不需要设置.
	ExporterEndpointPath string
	// ExporterHeader 设置请求头, 一般不需要使用, 最常见的用处是设置 Authorize
	ExporterHeader map[string]string
	// Mission 设置本次运行的是什么爬虫, 用于 Grafana 筛选数据, 本质是 OTel Name
	Mission string
	// Business 设置本次运行的爬虫属于哪个业务, 用于 Grafana 筛选数据, 本质是 OTel Namespace
	Business string
	// MetricSendInterval 设置 Metric 上报数据的频率
	MetricSendInterval time.Duration
}
