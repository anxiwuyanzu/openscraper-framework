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
	var Http2Spider spider.Anchor = "http2"
	Http2Spider.Register(func() *spider.Factory {
		proxy, err := proxz.ParseConfigFromString("socks://127.0.0.1:8889")
		if err != nil {
			panic(err)
		}

		m := &Http2Middleware{
			client: reqwest.NewClient(nil, reqwest.WithProxyConfig(proxy), reqwest.WithHttpVersion(2)),
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

type Http2Middleware struct {
	client reqwest.IClient
}

func (m *Http2Middleware) doRequest(ctx spider.Context) {
	req := ctx.Request()

	// add common headers in Middleware.
	req.SetHeader("content-type", "application/json")
	req.SetHeader("referer", "https://mix.jinritemai.com")
	req.SetHeader("accept-encoding", "gzip, deflate, br")
	req.SetUserAgent("Mozilla/5.0 (iPhone; CPU iPhone OS 14_6 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Mobile/15E148 BytedanceWebview/d8a21c6 aweme_22.1.0 JsSdk/2.0 NetType/WIFI Channel/App Store ByteLocale/zh Region/CN AppTheme/light WebcastSDK/2620 Region/CN App/aweme AppVersion/22.1.0 VersionCode/221017 Channel/App Store Webcast_ByteLocale/zh AppTheme/light FalconTag/3D1718C4-CB3C-4C5A-A1D1-E1EA4AFADFB1 ")

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
	req.SetRequestURI("https://mon.snssdk.com/monitor_web/settings/browser-settings?bid=mix_h5&store=1")
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
	engine.Start("http2")
}
