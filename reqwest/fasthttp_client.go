package reqwest

import (
	"crypto/tls"
	"errors"
	"github.com/anxiwuyanzu/openscraper-framework/spider-common-go/v4/dot"
	"github.com/anxiwuyanzu/openscraper-framework/spider-common-go/v4/reqwest/stats"
	"github.com/sirupsen/logrus"
	"github.com/valyala/fasthttp"
	"sync"
	"sync/atomic"
	"time"
)

var (
	ErrConnectionClosed = errors.New("connection closed")
)

// FastHttpClient 负责 fasthttp 请求, 包含代理功能
type FastHttpClient struct {
	sync.Mutex
	*ProxyGuard

	innerClient     *fasthttp.Client
	logger          *logrus.Entry
	option          *Option
	httpFailedTimes int32
}

func NewFastHttpClient(option *Option, opts ...OpOption) *FastHttpClient {
	if option == nil {
		option = DefaultOption()
	}
	for _, opFunc := range opts {
		opFunc(option)
	}
	logger := dot.Logger()

	f := &FastHttpClient{
		logger:     logger,
		option:     option,
		ProxyGuard: NewProxyGuard(option),
	}

	f.createClient()
	return f
}

func (c *FastHttpClient) createClient() {
	c.Lock()
	defer c.Unlock()

	dial := c.getDialer()

	c.innerClient = &fasthttp.Client{
		Dial:                     dial.ToFastHttpDialFunc(),
		Name:                     "",
		NoDefaultUserAgentHeader: true,
		ReadBufferSize:           c.option.ReadBufferSize,
		ReadTimeout:              c.option.Timeout,
		WriteTimeout:             c.option.Timeout,
		MaxIdleConnDuration:      c.option.Idle,
		MaxConnsPerHost:          c.option.MaxConnsPerHost,
		// 当代理断开连接，client会按MaxIdemponentCallAttempts设置的次数自动重试，重试过程不会报错；
		MaxIdemponentCallAttempts: c.option.MaxCallAttempts,
		TLSConfig: &tls.Config{
			InsecureSkipVerify: true,
		},
	}
	c.closeIdleConnections = c.innerClient.CloseIdleConnections
}

func (c *FastHttpClient) DoRequest(req IRequest) error {
	res := req.(IFastHttpRequest)
	return c.DoTimeout(res.GetRequest(), res.GetResponse(), c.option.Timeout)
}

func (c *FastHttpClient) DoRequestTimeout(req IRequest, timeout time.Duration) error {
	res := req.(IFastHttpRequest)
	return c.DoTimeout(res.GetRequest(), res.GetResponse(), timeout)
}

func (c *FastHttpClient) DoRequestTimeoutAndRetry(req IRequest, timeout time.Duration, times int) error {
	res := req.(IFastHttpRequest)
	var err error
	for times > 0 {
		times = times - 1
		err = c.DoTimeout(res.GetRequest(), res.GetResponse(), timeout)
		if err == nil {
			return nil
		}
	}
	return err
}

func (c *FastHttpClient) DoTimeout(req *fasthttp.Request, resp *fasthttp.Response, timeout time.Duration) error {
	c.option.Acquire()
	defer c.option.Release()

	c.CheckProxyAndCloseIdleConn()
	stats.IncrRequest()

	err := c.innerClient.DoTimeout(req, resp, timeout)
	if err != nil {
		atomic.AddInt32(&c.httpFailedTimes, 1)
		stats.IncrRequestFailed()

		// 返回这个错误, 一般是proxy失效了, 大概率所有的 Connection 都被服务端关闭了; 直接换Proxy, 主动关闭 Connection
		// 而不是一个个 Connection 重试一遍
		if errors.Is(err, fasthttp.ErrConnectionClosed) {
			c.ReleaseProxy()
			return ErrConnectionClosed
		}
		if err.Error() == "tls: first record does not look like a TLS handshake" {
			c.ReleaseProxy()
			return ErrConnectionClosed
		}
		if errors.Is(err, fasthttp.ErrTLSHandshakeTimeout) {
			c.ReleaseProxy()
			return fasthttp.ErrTLSHandshakeTimeout
		}
		return err
	}

	atomic.StoreInt32(&c.httpFailedTimes, 0)
	return nil
}

func (c *FastHttpClient) Do(req *fasthttp.Request, resp *fasthttp.Response) error {
	return c.DoTimeout(req, resp, c.option.Timeout)
}

func (c *FastHttpClient) DoRedirects(req *fasthttp.Request, resp *fasthttp.Response, maxRedirectsCount int) error {
	c.option.Acquire()
	defer c.option.Release()

	c.CheckProxyAndCloseIdleConn()
	return c.innerClient.DoRedirects(req, resp, maxRedirectsCount)
}

func (c *FastHttpClient) InnerClient() *fasthttp.Client {
	return c.innerClient
}

func (c *FastHttpClient) HttpFailedTimes() int {
	return int(c.httpFailedTimes)
}
