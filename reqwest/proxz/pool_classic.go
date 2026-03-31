package proxz

import (
	"github.com/anxiwuyanzu/openscraper-framework/v4/dot"
	"github.com/anxiwuyanzu/openscraper-framework/v4/reqwest/stats"
	"github.com/valyala/fasthttp"
	"sync"
	"time"
)

// ClassicPool 管理代理获取和过期
// 将管理代理的策略部分独立出来
type ClassicPool struct {
	sync.Mutex
	dropProxyPoint  time.Duration // 在代理过期时间小于 dropProxyPoint 时丢弃
	filterProxySec  int64
	getProxyTimeout time.Duration // 获取 proxy 超时时间
	fetcher         ProxyFetcher
	current         *ProxyIp
	timeFilter      map[string]int64
}

func NewClassicPool(fetcher ProxyFetcher, config *ProxyConfig) ProxyPoolManager {
	dropProxyPoint := config.GetDropProxyPoint()
	if dropProxyPoint == 0 {
		dropProxyPoint = 5 * time.Second
	}

	getProxyTimeout := config.GetFetchProxyTimeout()
	if getProxyTimeout == 0 {
		getProxyTimeout = 15 * time.Second
	}

	return &ClassicPool{
		fetcher:         fetcher,
		dropProxyPoint:  dropProxyPoint,
		filterProxySec:  config.GetFilterProxySec(),
		getProxyTimeout: getProxyTimeout,
		timeFilter:      make(map[string]int64, 1000),
	}
}

func (m *ClassicPool) AcquireProxy() (*ProxyIp, error) {
	tc := fasthttp.AcquireTimer(m.getProxyTimeout)
	defer fasthttp.ReleaseTimer(tc)

	for {
		m.fetcher.TryFetchProxy()

		select {
		case proxy := <-m.fetcher.ProxyCh():
			now := time.Now()
			// 如果代理快过期，丢弃
			if proxy.Expire.Sub(now) < m.dropProxyPoint {
				//dot.Logger().WithField("expire", proxy.Expire).Info("proxy is expired")
				stats.IncrExpiredProxies()
				continue
			}
			// 如果代理上次使用时间不满足filterProxySec，丢弃
			if m.filterProxySec > 0 && !m.filter(proxy.Host, now.Unix()) {
				continue
			}

			return proxy, nil
		case <-tc.C:
			stats.IncrAcquireProxiesTimeout()
			return nil, fasthttp.ErrTimeout
		}
	}
}

func (m *ClassicPool) filter(ip string, now int64) bool {
	m.Lock()
	defer m.Unlock()
	if ts, ok := m.timeFilter[ip]; ok && now-ts < m.filterProxySec {
		dot.Logger().WithField("cnt", len(m.timeFilter)).WithField("ip", ip).WithField("ut", ts).Info("filter proxy")
		return false
	}

	if len(m.timeFilter) > 30000 { // 避免map过大
		for p, ts := range m.timeFilter {
			if now-ts > 10800 { // 3 hours
				delete(m.timeFilter, p)
			}
		}
	}

	m.timeFilter[ip] = now
	return true
}
