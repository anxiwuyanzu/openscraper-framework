package proxz

import (
	"context"
	"errors"
	"net"
	"sync"
	"time"

	"github.com/anxiwuyanzu/openscraper-framework/spider-common-go/v4/dot"
	"github.com/anxiwuyanzu/openscraper-framework/spider-common-go/v4/reqwest/stats"
	"github.com/valyala/fasthttp/fasthttpproxy"
	goproxy "golang.org/x/net/proxy"
)

var (
	ErrNoProxy = errors.New("failed to get proxy")
	poolLock   = sync.Mutex{}

	providerBuilders = make(map[string]FetcherBuilder)
	providerPool     = make(map[string]ProxyPoolManager)
)

// RegisterProvider 第三方注册 FetcherBuilder 到这里, 以便后续通过配置构建
func RegisterProvider(name string, builder FetcherBuilder) {
	providerBuilders[name] = builder
}

// FetcherBuilder 定义创建 ProxyFetcher 的接口
type FetcherBuilder interface {
	// BuildFetcher 返回 ProxyFetcher
	BuildFetcher(config *ProxyConfig) ProxyFetcher
}

// ProxyFetcher 定义获取代理的接口
type ProxyFetcher interface {
	// Key 返回 ProxyFetcher 的唯一标识
	Key() string
	// TryFetchProxy 尝试去第三方(芝麻 西瓜等)获取代理
	TryFetchProxy()
	// ProxyCh 返回代理的ch
	ProxyCh() chan *ProxyIp
	FetchByIpLocation(city, ip string) *ProxyIp
}

// PoolBuilder 定义用来创建 ProxyPoolManager 的方法
type PoolBuilder func(fetcher ProxyFetcher, config *ProxyConfig) ProxyPoolManager

// ProxyPoolManager 管理代理池, 一般 ProxyPoolManager 拥有一个 ProxyFetcher,
// ProxyPoolManager 管理怎么调用 ProxyFetcher
// 你可以轻松的自定义自己的 ProxyPoolManager, 通过配置传入
type ProxyPoolManager interface {
	AcquireProxy() (*ProxyIp, error)
}

// CreateProxyPool 创建 ProxyPoolManager, 默认使用 ClassicPool
func CreateProxyPool(config *ProxyConfig) ProxyPoolManager {
	poolLock.Lock()
	defer poolLock.Unlock()

	if builder, ok := providerBuilders[config.Provider]; ok {
		fetcher := builder.BuildFetcher(config)

		if pool, ok := providerPool[fetcher.Key()]; ok { // 同一种 fetcher 只需要一个pool
			return pool
		}

		var pool ProxyPoolManager
		if config.PoolBuilder == nil {
			pool = NewClassicPool(fetcher, config)
		} else {
			pool = config.PoolBuilder(fetcher, config)
		}
		providerPool[fetcher.Key()] = pool
		return pool
	}
	return nil
}

func CreateProxyFetcher(config *ProxyConfig) ProxyFetcher {
	poolLock.Lock()
	defer poolLock.Unlock()

	if builder, ok := providerBuilders[config.Provider]; ok {
		return builder.BuildFetcher(config)
	}

	return nil
}

// PoolDial 为 Client 提供 Dial
func PoolDial(mgr ProxyPoolManager, addr string, timeout time.Duration) (net.Conn, error) {
	start := time.Now()

	proxy, err := mgr.AcquireProxy()
	if err != nil || proxy == nil {
		dot.Logger().WithError(err).Warn("Could not get proxy")
		return nil, ErrNoProxy
	}

	if ts := time.Since(start).Milliseconds(); ts > 2000 {
		dot.Logger().WithField("took", ts).Info("wait for get proxy")
	}

	stats.IncrDialProxies()

	var conn net.Conn
	switch proxy.Protocol {
	case ProtocolHttp:
		host := proxy.Host
		if len(proxy.AuthUser) > 0 {
			host = proxy.AuthUser + ":" + proxy.AuthPwd + "@" + proxy.Host
		}
		dialer := fasthttpproxy.FasthttpHTTPDialerTimeout(host, timeout)
		conn, err = dialer(addr)
	case ProtocolSocks:
		var auth *goproxy.Auth
		if len(proxy.AuthUser) > 0 {
			auth = &goproxy.Auth{User: proxy.AuthUser, Password: proxy.AuthPwd}
		}

		var dialer goproxy.Dialer
		dialer, err = goproxy.SOCKS5("tcp", proxy.Host, auth, goproxy.Direct)
		if err != nil {
			stats.IncrDialFailedProxies()
			return nil, err
		}
		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		defer cancel()
		conn, err = dialer.(goproxy.ContextDialer).DialContext(ctx, "tcp", addr)
	default:
		dot.Logger().Fatalf("do not support protocol: %d", proxy.Protocol)
	}

	if err != nil {
		proxy.Invalid = true
		stats.IncrDialFailedProxies()
	}
	return conn, err
}
