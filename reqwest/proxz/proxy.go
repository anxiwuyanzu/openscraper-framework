package proxz

import (
	"fmt"
	"net/url"
	"strconv"
	"time"

	"github.com/anxiwuyanzu/openscraper-framework/v4/dot"
)

type Protocol uint8

const (
	ProtocolNone  Protocol = 0
	ProtocolHttp  Protocol = 1
	ProtocolSocks Protocol = 2
)

type ProxyIp struct {
	Host        string
	Expire      time.Time
	Protocol    Protocol
	CityCode    string
	Invalid     bool
	Provider    string
	SubProvider string
	AuthUser    string
	AuthPwd     string
}

func (p *ProxyIp) String() string {
	return p.Host
}

// ProxyConfig 代理配置; 设计目标: 1. 可以描述所有代理; 2. 可以方便环境变量配置方式(一行表达)
type ProxyConfig struct {
	Protocol    Protocol
	Provider    string
	Proxy       string
	PoolBuilder PoolBuilder
	Params      url.Values
}

func DefaultProxyConfig() *ProxyConfig {
	return &ProxyConfig{Protocol: ProtocolNone}
}

func ParseProxyConfigFromConfig() (*ProxyConfig, error) {
	proxy := dot.Conf().Proxy.Proxy
	if len(proxy) == 0 || proxy == "direct" {
		return DefaultProxyConfig(), nil
	}
	return ParseConfigFromString(proxy)
}

// ParseConfigFromString like:
// http://127.0.0.1:8889; socks://127.0.0.1:8889; http://username:pwd@127.0.0.1:8889
// http://zhima?chan=zl&topic=default&size=100
// http://relay?sub=mix
// http://xigua?ipv6=1
func ParseConfigFromString(str string) (*ProxyConfig, error) {
	parsed, err := url.Parse(str)
	if err != nil {
		return nil, err
	}

	protocol := ProtocolNone
	switch parsed.Scheme {
	case "http", "https":
		protocol = ProtocolHttp
	case "socks", "socks5", "sock":
		protocol = ProtocolSocks
	default:
		return nil, fmt.Errorf("unkonow protocol :%s", parsed.Scheme)
	}

	config := &ProxyConfig{Protocol: protocol}
	if _, ok := providerBuilders[parsed.Host]; ok { // 外部提供商
		config.Provider = parsed.Host
		config.Params = parsed.Query()
		config.PoolBuilder = NewClassicPool
	} else {
		config.Proxy = str[len(parsed.Scheme)+3:]
	}
	return config, nil
}

// CreateDefaultPool 创建默认的 PoolManager. 也可根据自己需求创建PoolManager
func (c *ProxyConfig) CreateDefaultPool() ProxyPoolManager {
	return CreateProxyPool(c)
}

// CreateFetcher 根据配置创建相应提供商的 Proxy Fetcher
func (c *ProxyConfig) CreateFetcher() ProxyFetcher {
	return CreateProxyFetcher(c)
}

func (c *ProxyConfig) GetSizeOrDefault(def int) int {
	size, _ := strconv.Atoi(c.Params.Get("size"))
	if size == 0 {
		return def
	}
	return size
}

func (c *ProxyConfig) GetDropProxyPoint() time.Duration {
	dropIn, _ := strconv.Atoi(c.Params.Get("drop_in"))
	if dropIn == 0 {
		return 5 * time.Second
	}
	return time.Duration(dropIn) * time.Second
}

func (c *ProxyConfig) GetFilterProxySec() int64 {
	filterIn, _ := strconv.Atoi(c.Params.Get("filter_in"))
	return int64(filterIn)
}

func (c *ProxyConfig) GetFetchProxyTimeout() time.Duration {
	timeout, _ := strconv.Atoi(c.Params.Get("timeout"))
	if timeout == 0 {
		return 15 * time.Second
	}
	return time.Duration(timeout) * time.Second
}
