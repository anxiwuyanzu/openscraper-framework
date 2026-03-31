package reqwest

import (
	"context"
	"crypto/tls"
	"github.com/anxiwuyanzu/openscraper-framework/spider-common-go/v4/dot"
	"github.com/anxiwuyanzu/openscraper-framework/spider-common-go/v4/reqwest/stats"
	"github.com/sirupsen/logrus"
	"net/http"
	"sync"
	"sync/atomic"
	"time"
)

// StandardClient 负责 net/http 请求, 包含代理功能;
type StandardClient struct {
	sync.Mutex
	*ProxyGuard
	innerClient     *http.Client
	logger          *logrus.Entry
	option          *Option
	httpFailedTimes int32
}

func NewStandardClient(option *Option, opts ...OpOption) *StandardClient {
	if option == nil {
		option = DefaultOption()
	}
	for _, opFunc := range opts {
		opFunc(option)
	}

	c := &StandardClient{
		ProxyGuard: NewProxyGuard(option),
		logger:     dot.Logger(),
		option:     option,
	}
	c.createClient()

	return c
}

func (c *StandardClient) createClient() {
	c.Lock()
	defer c.Unlock()

	dial := c.getDialer()

	var tr http.RoundTripper
	if c.option.HttpVersion <= 2 {
		trv1v2 := &http.Transport{
			DialContext:        dial,
			TLSClientConfig:    &tls.Config{InsecureSkipVerify: true},
			MaxConnsPerHost:    c.option.MaxConnsPerHost,
			IdleConnTimeout:    c.option.Idle,
			DisableCompression: true,
		}

		if c.option.HttpVersion == 2 {
			trv1v2.ForceAttemptHTTP2 = true
			//trv1v2.ResponseHeaderTimeout = time.Second * 10
			//trv1v2.ExpectContinueTimeout = time.Second * 10
		}
		tr = trv1v2
	} else {
		panic("unknown http version")
	}

	client := &http.Client{
		Transport: tr,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
		//Timeout:   c.option.Timeout,
	}

	c.innerClient = client
	c.closeIdleConnections = c.innerClient.CloseIdleConnections
}

// Do execute raw http.Request
func (c *StandardClient) Do(req *http.Request) (*http.Response, error) {
	c.option.Acquire()
	defer c.option.Release()

	c.CheckProxyAndCloseIdleConn()
	return c.innerClient.Do(req)
}

// DoRequest execute StandardRequest
func (c *StandardClient) DoRequest(req IRequest) error {
	return c.DoRequestTimeout(req, c.option.Timeout)
}

// DoRequestTimeout testme
func (c *StandardClient) DoRequestTimeout(req IRequest, timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	res := req.(IStandardRequest)
	httpReq := res.GetRequest()
	httpReq = httpReq.WithContext(ctx)

	resp, err := c.Do(httpReq)
	if err != nil {
		atomic.AddInt32(&c.httpFailedTimes, 1)
		stats.IncrRequestFailed()
		return err
	}
	atomic.StoreInt32(&c.httpFailedTimes, 0)
	return res.SetResponse(resp)
}

func (c *StandardClient) DoRequestTimeoutAndRetry(req IRequest, timeout time.Duration, times int) error {
	var err error
	for times > 0 {
		times = times - 1
		err = c.DoRequestTimeout(req, timeout)
		if err == nil {
			return nil
		}
	}
	return err
}

func (c *StandardClient) InnerClient() *http.Client {
	return c.innerClient
}

func (c *StandardClient) HttpFailedTimes() int {
	return int(c.httpFailedTimes)
}
