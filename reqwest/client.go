package reqwest

import (
	"context"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/anxiwuyanzu/openscraper-framework/v4/dot"
	"github.com/anxiwuyanzu/openscraper-framework/v4/reqwest/proxz"
	_ "github.com/anxiwuyanzu/openscraper-framework/v4/reqwest/proxz/providers"
	"github.com/anxiwuyanzu/openscraper-framework/v4/reqwest/stats"
	"github.com/valyala/fasthttp"
)

type IRequest interface {
	SetMethod(string)
}

type IClient interface {
	DoRequest(req IRequest) error
	DoRequestTimeout(req IRequest, timeout time.Duration) error
	// DoRequestTimeoutAndRetry 强制重试请求; Client 可能内部可能也有retry, 但是通常用在Get,Head请求
	DoRequestTimeoutAndRetry(req IRequest, timeout time.Duration, times int) error
	ProxyString() string
	ReleaseProxy()
	HttpFailedTimes() int
	Proxy() *proxz.ProxyIp
	//ProxyDialer() DialFunc
}

func NewClient(option *Option, opts ...OpOption) IClient {
	if option == nil {
		option = DefaultOption()
	}

	for _, opFunc := range opts {
		opFunc(option)
	}

	if option.Client == "fasthttp" {
		option.HttpVersion = 1
		return NewFastHttpClient(option)
	}
	if option.Client == "tlshttp" {
		return NewTlsClient(option)
	}

	return NewStandardClient(option)
}

// ProxyGuard 通常和client结合; 用在一个client一次只拿一个代理, 直到过期, 或者主动释放
type ProxyGuard struct {
	proxy                *proxz.ProxyIp
	proxyLock            sync.Mutex
	proxyPool            proxz.ProxyPoolManager
	proxyFetcher         proxz.ProxyFetcher
	proxyDialer          DialFunc
	option               *Option
	closeIdleConnections func()
}

func NewProxyGuard(option *Option) *ProxyGuard {
	if option == nil {
		option = DefaultOption()
	}

	return &ProxyGuard{
		proxyLock:            sync.Mutex{},
		option:               option,
		closeIdleConnections: func() {}, // 纯粹是使用的时候不想判断 closeIdleConnections == nil
	}
}

func (pg *ProxyGuard) getDialer() DialFunc {
	proxyConfig := pg.option.ProxyConfig

	var dial DialFunc
	var err error
	if proxyConfig == nil {
		return nil
	}
	if len(proxyConfig.Provider) > 0 {
		if pg.option.ProxyFetchByIpLocation != "" || pg.option.ProxyFetchByCity != "" {
			pg.proxyFetcher = proxyConfig.CreateFetcher()
			if pg.proxyFetcher == nil {
				dot.Logger().Panicf("unknown provider %s", proxyConfig.Provider)
			}
		} else {
			pg.proxyPool = proxyConfig.CreateDefaultPool()
			if pg.proxyPool == nil {
				dot.Logger().Panicf("unknown provider %s", proxyConfig.Provider)
			}
		}

		dial = func(ctx context.Context, network, addr string) (net.Conn, error) {
			return proxz.PoolDial(pg, addr, pg.option.DialTimeout)
		}
	} else if proxyConfig.Protocol == proxz.ProtocolHttp {
		// 如 127.0.0.1:8888  user:pwd@127.0.0.1:8888 不包含http://
		if len(proxyConfig.Proxy) == 0 {
			panic("proxy is not set")
		}
		dial = HttpProxyDialer(proxyConfig.Proxy, pg.option.DialTimeout)
		pg.proxy = &proxz.ProxyIp{Protocol: proxz.ProtocolHttp, Host: proxyConfig.Proxy}
	} else if proxyConfig.Protocol == proxz.ProtocolSocks {
		if len(proxyConfig.Proxy) == 0 {
			panic("proxy is not set")
		}
		pg.proxy = &proxz.ProxyIp{Protocol: proxz.ProtocolSocks, Host: proxyConfig.Proxy}
		dial, err = SocksDialer(proxyConfig.Proxy, pg.option.DialTimeout)
		if err != nil {
			dot.Logger().Panic(fmt.Sprintf("failed to dial socks: %s; err: %s", proxyConfig.Proxy, err.Error()))
		}
	} else {
		dialer := &fasthttp.TCPDialer{Concurrency: 1000}
		dial = func(ctx context.Context, network, addr string) (net.Conn, error) {
			return dialer.DialTimeout(addr, pg.option.DialTimeout)
		}
	}

	pg.proxyDialer = dial
	return dial
}

// AcquireProxy 实现 ProxyPoolManager; 基本目标是让 Client 同时只有一个 Proxy;
// 但是有个小问题值得思考, 如果 proxy 是无效的, 会导致...
func (pg *ProxyGuard) AcquireProxy() (*proxz.ProxyIp, error) {
	pg.proxyLock.Lock()
	defer pg.proxyLock.Unlock()

	var err error
	var create bool
	if pg.proxy == nil || pg.proxy.Invalid {
		create = true
	} else if pg.proxy != nil && time.Until(pg.proxy.Expire) < pg.option.ProxyDropIn {
		create = true
	}

	if create {
		if pg.proxyPool != nil {
			pg.proxy, err = pg.proxyPool.AcquireProxy()
			if err != nil {
				if err == fasthttp.ErrTimeout {
					dot.Logger().WithError(err).Error("failed to acquire proxy")
				}
				return pg.proxy, err
			}
		} else if pg.proxyFetcher != nil {
			proxy := pg.proxyFetcher.FetchByIpLocation(pg.option.ProxyFetchByCity, pg.option.ProxyFetchByIpLocation)
			if proxy != nil {
				pg.proxy = proxy
				return pg.proxy, err
			}
		}

	}
	return pg.proxy, err
}

func (pg *ProxyGuard) Proxy() *proxz.ProxyIp {
	return pg.proxy
}

func (pg *ProxyGuard) ProxyString() string {
	return fmt.Sprintf("%s", pg.proxy)
}

// ReleaseProxy 释放代理
func (pg *ProxyGuard) ReleaseProxy() {
	if pg.proxyPool == nil {
		return
	}
	stats.IncrReleaseProxies()
	pg.proxyLock.Lock()
	defer pg.proxyLock.Unlock()

	pg.proxy = nil
	pg.closeIdleConnections()
}

// CheckProxyAndCloseIdleConn 检查代理是否即将过期, 避免因为代理过期导致请求失败, 影响错误的统计
func (pg *ProxyGuard) CheckProxyAndCloseIdleConn() {
	pg.proxyLock.Lock()
	defer pg.proxyLock.Unlock()

	if pg.proxyPool == nil || pg.proxy == nil || pg.proxy.Expire.IsZero() {
		return
	}
	if pg.proxy != nil && time.Until(pg.proxy.Expire) < pg.option.ProxyDropIn {
		pg.proxy = nil
		pg.closeIdleConnections()
	}
}

func (pg *ProxyGuard) ProxyDialer() DialFunc {
	if pg.proxyDialer == nil {
		pg.proxyDialer = pg.getDialer()
	}
	return pg.proxyDialer
}
