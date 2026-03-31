package main

import (
	"fmt"
	"github.com/anxiwuyanzu/openscraper-framework/spider-common-go/v4/dot"
	"github.com/anxiwuyanzu/openscraper-framework/spider-common-go/v4/reqwest"
	"github.com/anxiwuyanzu/openscraper-framework/spider-common-go/v4/reqwest/proxz"
	"github.com/anxiwuyanzu/openscraper-framework/spider-common-go/v4/spider"
	log "github.com/sirupsen/logrus"
)

func init() {
	spider.Anchor("my/spider").Register(func() *spider.Factory {
		proxy, err := proxz.ParseConfigFromString("socks://zhima-relay?chan=10m&topic=default&size=3&provider=fly,qingguo")
		if err != nil {
			panic(err)
		}

		option := reqwest.DefaultOption()
		option.ProxyConfig = proxy
		// option.ProxyFetchByIpLocation = "103.63.155.35"

		m := &MyMiddleware{
			client: reqwest.NewClient(option),
		}
		return &spider.Factory{
			SourceFactory: func(itemCh spider.ItemCh, workerNum, mode int) {
				itemCh <- dot.Item{"id": "xx"}
			},
			SpiderFactory: func(logger *log.Entry) spider.Spider {
				s := &SomeSpider{}
				s.Use(spider.NewLoggerWithUri())
				s.Use(m.doRequest)
				return s
			},
			MaxRetryTimes: 0,
		}
	})

	spider.Anchor("my/spider2").Register(func() *spider.Factory {
		proxy, err := proxz.ParseConfigFromString("socks://zhima-relay?chan=10m&topic=default1&size=3&provider=fly,qingguo")
		if err != nil {
			panic(err)
		}

		option := reqwest.DefaultOption()
		option.ProxyConfig = proxy
		// option.ProxyFetchByIpLocation = "103.63.155.35"

		m := &MyMiddleware{
			client: reqwest.NewClient(option),
		}
		return &spider.Factory{
			SourceFactory: func(itemCh spider.ItemCh, workerNum, mode int) {
				itemCh <- dot.Item{"id": "xx"}
			},
			SpiderFactory: func(logger *log.Entry) spider.Spider {
				s := &SomeSpider{}
				s.Use(spider.NewLoggerWithUri())
				s.Use(m.doRequest)
				return s
			},
			MaxRetryTimes: 0,
		}
	})
}

type MyMiddleware struct {
	client reqwest.IClient
}

func (m *MyMiddleware) doRequest(ctx spider.Context) {
	req := ctx.Request()

	err := m.client.DoRequest(req)
	if err != nil {
		ctx.Fail(err)
		return
	}
	ctx.ParseResponse()
	ctx.Next()
}

type SomeSpider struct {
	spider.Application
}

// Start build basic request in Start
func (s *SomeSpider) Start(ctx spider.Context) {
	req := ctx.NewRequest()
	req.SetMethod("GET")
	req.SetHeader("user-agent", "curl/7.81.0")
	req.SetRequestURI("http://cip.cc")
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
	config := []spider.GroupConfig{
		{Spider: "my/spider"},
		{Spider: "my/spider2"},
	}
	engine.StartSpiderGroup(config)
	//time.Sleep(5 * time.Second)
}
