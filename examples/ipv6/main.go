package main

import (
	"fmt"
	"github.com/anxiwuyanzu/openscraper-framework/spider-common-go/v4/dot"
	"github.com/anxiwuyanzu/openscraper-framework/spider-common-go/v4/reqwest"
	"github.com/anxiwuyanzu/openscraper-framework/spider-common-go/v4/reqwest/proxz"
	"github.com/anxiwuyanzu/openscraper-framework/spider-common-go/v4/spider"
	"github.com/sirupsen/logrus"
)

func init() {
	var SomeSpider spider.Anchor = "ipv6"

	SomeSpider.Register(func() *spider.Factory {
		proxy, err := proxz.ParseConfigFromString("http://xigua?ipv6=1")
		if err != nil {
			panic(err)
		}
		m := &Ipv6Middleware{
			client: reqwest.NewClient(nil, reqwest.WithProxyConfig(proxy), reqwest.WithHttpVersion(2)),
		}

		return &spider.Factory{
			SourceFactory: func(itemCh spider.ItemCh, workerNum int, mode int) {
				itemCh <- dot.Item{"id": "xx"}
			},
			SpiderFactory: func(logger *logrus.Entry) spider.Spider {
				app := &Ipv6{}
				app.Use(spider.NewLogger())
				app.Use(m.doRequest)
				return app
			},
			MaxRetryTimes: 0,
		}
	})
}

type Ipv6Middleware struct {
	client reqwest.IClient
}

func (m *Ipv6Middleware) doRequest(ctx spider.Context) {
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

type Ipv6 struct {
	spider.Application
}

// https://api3-normal-c-hl.amemv.com/passport/token/beat/v2/?version_code=14.9.0&js_sdk_version=1.99.0.3&tma_jssdk_version=1.99.0.3&app_name=aweme&app_version=14.9.0&vendor_id=27A21A5E-910A-4CD5-A2F3-00B6C40CF1EC&vid=27A21A5E-910A-4CD5-A2F3-00B6C40CF1EC&device_id=66374238128&channel=App%20Store&mcc_mnc=46011&resolution=828%2A1792&aid=1128&app_id=1128&minor_status=0&screen_width=828&install_id=3624419971306603&openudid=0e6af968bfa6aaf08202e3469d330259205ca716&cdid=0E04B687-D87E-4751-B9C5-E212A6FD1E76&os_api=18&idfv=27A21A5E-910A-4CD5-A2F3-00B6C40CF1EC&ac=WIFI&os_version=13.5.1&ssmix=a&appTheme=dark&device_platform=iphone&build_number=149014&is_vcd=1&device_type=iPhone11%2C8&iid=3624419971306603&idfa=46829C6E-A738-43AB-91E3-586CD81B0887&scene=boot&first_beat=0
func (s *Ipv6) Start(ctx spider.Context) {
	uri := ""
	uri = "http://ipv6.vm3.test-ipv6.com/ip/?callback=?&testdomain=test-ipv6.com&testname=test_aaaa"
	//uri = "https://api3-normal-c-hl.amemv.com/passport/token/beat/v2/?version_code=14.9.0&js_sdk_version=1.99.0.3&tma_jssdk_version=1.99.0.3&app_name=aweme&app_version=14.9.0&vendor_id=27A21A5E-910A-4CD5-A2F3-00B6C40CF1EC&vid=27A21A5E-910A-4CD5-A2F3-00B6C40CF1EC&device_id=66374238128&channel=App%20Store&mcc_mnc=46011&resolution=828%2A1792&aid=1128&app_id=1128&minor_status=0&screen_width=828&install_id=3624419971306603&openudid=0e6af968bfa6aaf08202e3469d330259205ca716&cdid=0E04B687-D87E-4751-B9C5-E212A6FD1E76&os_api=18&idfv=27A21A5E-910A-4CD5-A2F3-00B6C40CF1EC&ac=WIFI&os_version=13.5.1&ssmix=a&appTheme=dark&device_platform=iphone&build_number=149014&is_vcd=1&device_type=iPhone11%2C8&iid=3624419971306603&idfa=46829C6E-A738-43AB-91E3-586CD81B0887&scene=boot&first_beat=0"
	uri = "https://api5-normal-c-hl.amemv.com/aweme/v2/shop/promotion/pack/?is_h5=1&is_native_h5=1"
	req := ctx.NewRequest()
	req.SetRequestURI(uri)
}

func (s *Ipv6) Parse(ctx spider.Context) {
	body, _ := ctx.Request().ResponseBody()

	fmt.Println(string(body))

	ctx.Ok()
}

// 测试目标域名是否支持ipv6
func main() {
	engine := spider.NewEngine()
	engine.Start("ipv6")
}
