package reqwest

import (
	"context"
	"time"

	"github.com/anxiwuyanzu/openscraper-framework/spider-common-go/v4/dot"
	"github.com/anxiwuyanzu/openscraper-framework/spider-common-go/v4/reqwest/proxz"
	"golang.org/x/sync/semaphore"
)

type Option struct {
	dot.Reqwest

	ProxyConfig *proxz.ProxyConfig
	Limit       *semaphore.Weighted

	// ProxyFetchByIpLocation 设置代理城市,传入ip, 程序会自动通过 ip 找到同源 city
	ProxyFetchByIpLocation string
	// ProxyFetchByCity 设置代理城市,传入city code
	ProxyFetchByCity string
}

func (o *Option) Acquire() {
	if o.Limit != nil {
		_ = o.Limit.Acquire(context.Background(), 1)
		return
	}

	if o.Concurrence > 0 {
		if v := dot.Value("DOT.GLOBAL.CONCURRENCE"); v != nil {
			o.Limit = v.(*semaphore.Weighted)
			_ = o.Limit.Acquire(context.Background(), 1)
			return
		}

		v := dot.WithImmutableValue("DOT.GLOBAL.CONCURRENCE", semaphore.NewWeighted(o.Concurrence))
		o.Limit = v.(*semaphore.Weighted)
		_ = o.Limit.Acquire(context.Background(), 1)
	}
}

func (o *Option) Release() {
	if o.Limit != nil {
		o.Limit.Release(1)
	}
}

// DefaultOption 读取配置文件中的设置
func DefaultOption() *Option {
	defaultProxyConfig, _ := proxz.ParseProxyConfigFromConfig()

	return &Option{
		Reqwest:     dot.Conf().Reqwest,
		ProxyConfig: defaultProxyConfig,
	}
}

func DefaultOptionWithProxy(proxyConfig *proxz.ProxyConfig) *Option {
	if proxyConfig == nil {
		proxyConfig = proxz.DefaultProxyConfig()
	}

	return &Option{
		Reqwest:     dot.Conf().Reqwest,
		ProxyConfig: proxyConfig,
	}
}

type OpOption func(*Option)

func WithIdle(idle time.Duration) OpOption {
	return func(op *Option) { op.Idle = idle }
}

func WithTimeout(timeout time.Duration) OpOption {
	return func(op *Option) { op.Timeout = timeout }
}

func WithDialTimeout(timeout time.Duration) OpOption {
	return func(op *Option) { op.DialTimeout = timeout }
}

func WithMaxConnsPerHost(maxConnsPerHost int) OpOption {
	return func(op *Option) { op.MaxConnsPerHost = maxConnsPerHost }
}

func WithHttpVersion(version int) OpOption {
	return func(op *Option) { op.HttpVersion = version }
}

func WithProxyConfig(proxyConfig *proxz.ProxyConfig) OpOption {
	return func(op *Option) { op.ProxyConfig = proxyConfig }
}

func WithLimit(limit *semaphore.Weighted) OpOption {
	return func(op *Option) { op.Limit = limit }
}
