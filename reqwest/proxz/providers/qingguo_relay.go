package providers

import (
	"fmt"
	"io"
	"net/http"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/anxiwuyanzu/openscraper-framework/v4/dot"
	"github.com/anxiwuyanzu/openscraper-framework/v4/reqwest/proxz"
	"github.com/anxiwuyanzu/openscraper-framework/v4/reqwest/stats"
	"github.com/sirupsen/logrus"
	"github.com/tidwall/gjson"
)

func init() {
	proxz.RegisterProvider(ProviderQingGuoRelay, &qingGuoRelaySingleton{
		fetcher: make(map[string]proxz.ProxyFetcher),
	})
}

type qingGuoRelaySingleton struct {
	fetcher map[string]proxz.ProxyFetcher
}

func (p *qingGuoRelaySingleton) BuildFetcher(config *proxz.ProxyConfig) proxz.ProxyFetcher {
	topic := config.Params.Get("topic")
	if len(topic) == 0 {
		topic = "default"
	}

	city := config.Params.Get("city")
	if city == "random" {
		city = ZhiMaGetRandomCity()
	}

	channel := getQingGuoChannel(config)
	key := topic + "-" + city + "-" + channel

	if mgr, ok := p.fetcher[key]; ok {
		return mgr
	}

	minimum := config.GetSizeOrDefault(30)
	keepAlive, _ := strconv.Atoi(config.Params.Get("keep_alive_sec"))

	fetcher := NewQingGuoUnifyFetcher(key, minimum, channel, topic, city, config.Protocol, keepAlive)
	p.fetcher[key] = fetcher
	return fetcher
}

func getQingGuoChannel(c *proxz.ProxyConfig) string {
	channel := c.Params.Get("chan")
	if channel == "10m" {
		return "10m"
	} else if channel == "15m" {
		return "15m"
	}
	return "3m"
}

// QingGuoUnifyFetcher 芝麻代理对请求 ip 有并发控制, 所以维护一个全局的获取 ip 的方式是必要的
// See https://github.com/anxiwuyanzu/openscraper-framework/proxy-relay/tree/v2-feature-get-ip
type QingGuoUnifyFetcher struct {
	sync.Mutex
	key           string
	getIpUrl      string
	proxies       chan *proxz.ProxyIp
	minimum       int
	getIpState    uint32
	proxyProtocol proxz.Protocol
	proxyType     string
	topic         string
	logger        *logrus.Entry
	city          string
	keepAliveSec  int
}

func NewQingGuoUnifyFetcher(key string, minimum int, proxyType, topic, city string, protocol proxz.Protocol, keepAlive int) proxz.ProxyFetcher {
	logger := dot.Logger().WithFields(logrus.Fields{"pt": proxyType, "topic": topic})

	logger.WithField("minimum", minimum).WithField("city", city).Info("QINGGUO Unify Fetcher is init")

	unifyAcquireUrl := dot.Conf().Proxy.QingGuoAcquireIpServer
	if len(unifyAcquireUrl) == 0 {
		unifyAcquireUrl = "http://qingguo-acquire-ip.cmm-crawler-intranet.k8s.limayao.com"
	}

	if len(topic) == 0 {
		topic = "default"
	}

	fetcher := &QingGuoUnifyFetcher{
		key:           key,
		getIpUrl:      unifyAcquireUrl,
		proxies:       make(chan *proxz.ProxyIp, minimum*15),
		proxyType:     proxyType,
		city:          city,
		minimum:       minimum,
		proxyProtocol: protocol,
		topic:         topic,
		logger:        logger,
		keepAliveSec:  keepAlive,
	}
	return fetcher
}

func (f *QingGuoUnifyFetcher) Key() string {
	return ProviderQingGuoRelay + "/" + f.key
}

func (f *QingGuoUnifyFetcher) ProxyCh() chan *proxz.ProxyIp {
	return f.proxies
}

func (f *QingGuoUnifyFetcher) TryFetchProxy() {
	if len(f.proxies) < f.minimum {
		go f.FetchProxy(f.proxyType)
	}
}

func (f *QingGuoUnifyFetcher) FetchProxy(proxyType string) {
	if !atomic.CompareAndSwapUint32(&f.getIpState, 0, 1) {
		return
	}

	defer atomic.CompareAndSwapUint32(&f.getIpState, 1, 0)

	start := time.Now()
	var uri string
	if len(f.city) > 0 {
		uri = fmt.Sprintf("%s/get_city_ip?city=%s&proxy_type=%s&size=%d&topic=%s&keep_alive_sec=%d", f.getIpUrl, f.city, proxyType, f.minimum, f.topic, f.keepAliveSec)
	} else {
		uri = fmt.Sprintf("%s/get_ip?topic=%s&proxy_type=%s&size=%d&keep_alive_sec=%d", f.getIpUrl, f.topic, proxyType, f.minimum, f.keepAliveSec)
	}

	resp, err := http.Get(uri)
	if err != nil {
		f.logger.WithField("url", f.getIpUrl).Error(err)
		return
	}
	defer func() {
		_ = resp.Body.Close()
	}()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		f.logger.Error(err)
		return
	}

	proxies := gjson.GetBytes(body, "data").Array()
	f.logger.WithFields(logrus.Fields{
		"took": time.Now().Sub(start).Milliseconds(),
		"cnt":  len(proxies),
		"pt":   proxyType,
	}).Info("request proxy from qingguo-unify success")

	stats.IncrAcquireProxies(uint32(len(proxies)))

	for _, proxy := range proxies {
		host := proxy.Get("host").String()
		expireTime := proxy.Get("expire").String()
		expire, _ := time.Parse(time.RFC3339, expireTime)
		city := proxy.Get("city").String()
		f.proxies <- &proxz.ProxyIp{
			Host:     host,
			Expire:   expire,
			CityCode: city,
			Protocol: f.proxyProtocol,
			Provider: ProviderQingGuoRelay,
			AuthUser: proxy.Get("auth_user").String(),
			AuthPwd:  proxy.Get("auth_pwd").String(),
		}
	}
}

func (f *QingGuoUnifyFetcher) FetchByIpLocation(city, ip string) *proxz.ProxyIp {
	start := time.Now()
	uri := fmt.Sprintf("%s/by_ip_location?size=1&city=%s&ip=%s&default_city=%s", f.getIpUrl, city, ip, f.city)

	resp, err := http.Get(uri)
	if err != nil {
		f.logger.WithField("url", f.getIpUrl).Error(err)
		return nil
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		f.logger.Error(err)
		return nil
	}

	proxy := gjson.GetBytes(body, "data.0")
	f.logger.WithFields(logrus.Fields{
		"took":      time.Since(start).Milliseconds(),
		"cnt":       1,
		"resp_msg":  gjson.GetBytes(body, "msg").String(),
		"resp_city": proxy.Get("city").String(),
	}).Info("request proxy from qingguo-unify success")

	host := proxy.Get("host").String()
	expireTime := proxy.Get("expire").String()
	expire, _ := time.Parse(time.RFC3339, expireTime)

	return &proxz.ProxyIp{
		Host:     host,
		Expire:   expire,
		CityCode: proxy.Get("city").String(),
		Protocol: f.proxyProtocol,
		Provider: ProviderQingGuoRelay,
		AuthUser: proxy.Get("auth_user").String(),
		AuthPwd:  proxy.Get("auth_pwd").String(),
	}
}
