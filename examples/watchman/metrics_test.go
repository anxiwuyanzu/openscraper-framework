package watchman

import (
	"github.com/anxiwuyanzu/openscraper-framework/v4/dot"
	"github.com/anxiwuyanzu/openscraper-framework/v4/exp/watchman/common"
	"github.com/anxiwuyanzu/openscraper-framework/v4/exp/watchman/metrics"
	"go.opentelemetry.io/otel/attribute"
	"sync"
	"sync/atomic"
	"time"
)

// 通过定义一个变量实现已记录的指标的暂存.
// 本样例展示了记录爬虫运行中, 各种 HTTP 响应码的数量的记录
// ===== 指标记录系统是读取一个指针指向的值(*ptr), 然后在上报完后将其设置为 0, 达到将暂存数据重置的作用 =====

var (
	statusCodeMap = map[string]*statusCode{}
	// spiders 模拟多个爬虫
	spiders = []string{"keyword", "live", "product"}
	// codes 模拟多种 HTTP 响应码
	codes = []int{200, 304, 400, 401, 429, 500}
)

// statusCode 用于暂存爬虫HTTP响应码个数的结构体
type statusCode struct {
	sync.RWMutex
	codes map[int]*int64
}

func init() {
	// 启用指标记录功能
	metrics.Setup(&common.Config{
		ExporterType:       common.ExporterGRPC,
		ExporterEndpoint:   "127.0.0.1",
		Mission:            "production",
		Business:           "cds",
		MetricSendInterval: 5 * time.Second,
	})
}

func main() {
	// 结束前关闭指标记录功能并清理资源
	defer metrics.Shutdown()

	// 为启动的所有爬虫都开辟一个暂存指标数据的地方
	for _, spider := range spiders {
		RegisterStatusCodeMetrics(spider)
	}

	// 模拟爬虫发起HTTP请求
	for {
		work()
	}
}

// work 模拟爬虫HTTP请求
func work() {
	time.Sleep(time.Second)
	code := codes[time.Now().Unix()%int64(len(codes))]
	spider := spiders[time.Now().Unix()%int64(len(spiders))]

	// 请求完成后, 记录一下
	RecordStatusCode(spider, code)
}

// RegisterStatusCodeMetrics 为爬虫开辟暂存指标数据的空间
func RegisterStatusCodeMetrics(spiderName string) {
	// 防止重复开辟
	if statusCodeMap[spiderName] == nil {
		statusCodeMap[spiderName] = &statusCode{
			codes: map[int]*int64{},
		}
	}
}

// RecordStatusCode 更新暂存的数据
func RecordStatusCode(spiderName string, code int) {
	// 如果该响应码是第一次出现, 则在 map 里新建一下, 防止 panic. 并注册对应的计数器(Gauge)
	if statusCodeMap[spiderName].codes[code] == nil {
		statusCodeMap[spiderName].Lock()
		statusCodeMap[spiderName].codes[code] = new(int64)

		// 注册一个计数器(Gauge)到指标记录系统中(某个爬虫的某种状态码的计数器).
		if err := metrics.RegisterGauge(
			statusCodeMap[spiderName].codes[code],
			"",
			"status_code",
			// attribute 是自定义的一些属性, 用于在 Grafana 筛选数据.
			attribute.String("spider", spiderName),
			attribute.Int("code", code),
		); err != nil {
			dot.Logger().Panic("failed to register gauge")
		}

		statusCodeMap[spiderName].Unlock()
	}

	// 对 map 的值的修改, 实际上并不是对 map 的写操作. 只有获取 key 的读操作.
	// map 的写操作是增删 key-value 键值对.
	statusCodeMap[spiderName].RLock()
	defer statusCodeMap[spiderName].RUnlock()
	atomic.AddInt64(statusCodeMap[spiderName].codes[code], 1)
}
