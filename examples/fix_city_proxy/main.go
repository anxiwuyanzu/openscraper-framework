package main

import (
	"fmt"
	"github.com/anxiwuyanzu/openscraper-framework/v4/dot"
	"github.com/anxiwuyanzu/openscraper-framework/v4/reqwest"
	"github.com/anxiwuyanzu/openscraper-framework/v4/reqwest/proxz"
	_ "github.com/anxiwuyanzu/openscraper-framework/v4/reqwest/proxz/providers"
	"github.com/anxiwuyanzu/openscraper-framework/v4/spider"
	log "github.com/sirupsen/logrus"
)

// 本爬虫用来测试设置固定省份的代理
func init() {
	var SomeSpiderName spider.Anchor = "ns/spider"
	SomeSpiderName.Register(func() *spider.Factory {
		proxy, err := proxz.ParseConfigFromString("http://zhima-relay?chan=zl&topic=default&size=10&city=random")
		if err != nil {
			panic(err)
		}

		m := &Middleware{
			client: reqwest.NewClient(nil, reqwest.WithProxyConfig(proxy)),
		}
		return &spider.Factory{
			SourceFactory: func(itemCh spider.ItemCh, workerNum, mode int) {
				itemCh <- dot.Item{"id": "xx1"}
				itemCh <- dot.Item{"id": "xx2"}
			},
			SpiderFactory: func(logger *log.Entry) spider.Spider {
				s := &SomeSpider{}
				s.Use(spider.NewLoggerWithUri())
				s.Use(m.doRequest)
				return s
			},
			WorkerNum:     1,
			MaxRetryTimes: 0,
		}
	})
}

type Middleware struct {
	client reqwest.IClient
}

func (m *Middleware) doRequest(ctx spider.Context) {
	req := ctx.Request()

	// add common headers in Middleware.
	req.SetUserAgent("curl")

	err := m.client.DoRequest(req)
	if err != nil {
		ctx.Fail(err)
		return
	}
	ctx.ParseResponse()
	ctx.Next()

	// 丢弃代理
	m.client.ReleaseProxy()
}

type SomeSpider struct {
	spider.Application
}

// Start build basic request in Start
func (s *SomeSpider) Start(ctx spider.Context) {
	req := ctx.NewRequest()
	req.SetMethod("GET")
	req.SetRequestURI("http://cip.cc/")
}

func (s *SomeSpider) Parse(ctx spider.Context) {
	body, err := ctx.Request().ResponseBody()
	if err != nil {
		ctx.Fail(err)
	}

	fmt.Println(string(body))
	ctx.Ok()
}

func main() {
	engine := spider.NewEngine()
	engine.Start("ns/spider")
}
