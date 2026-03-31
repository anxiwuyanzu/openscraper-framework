package providers

import (
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"github.com/anxiwuyanzu/openscraper-framework/spider-common-go/v4/dot"
	"github.com/anxiwuyanzu/openscraper-framework/spider-common-go/v4/reqwest/proxz"
	"github.com/anxiwuyanzu/openscraper-framework/spider-common-go/v4/reqwest/stats"
	"github.com/sirupsen/logrus"
	"github.com/tidwall/gjson"
)

func init() {
	proxz.RegisterProvider(ProviderZhiMaRelay, &zhiMaRelaySingleton{
		fetcher: make(map[string]proxz.ProxyFetcher),
	})
}

type zhiMaRelaySingleton struct {
	fetcher map[string]proxz.ProxyFetcher
}

func (p *zhiMaRelaySingleton) BuildFetcher(config *proxz.ProxyConfig) proxz.ProxyFetcher {
	topic := config.Params.Get("topic")
	if len(topic) == 0 {
		topic = "default"
	}

	city := config.Params.Get("city")
	if city == "random" {
		city = ZhiMaGetRandomCity()
	}
	channel := config.Params.Get("chan")
	key := topic + "-" + city + "-" + channel

	if mgr, ok := p.fetcher[key]; ok {
		return mgr
	}

	minimum := config.GetSizeOrDefault(30)
	provider := config.Params.Get("provider")

	fetcher := NewZhiMaUnifyFetcher(key, minimum, channel, topic, city, provider, config.Protocol)
	p.fetcher[key] = fetcher
	return fetcher
}

type zhiMaProxyType uint8

const (
	ZhimaTypeZL zhiMaProxyType = iota
	ZhimaTypeSD
	ZhimaTypeDX
	ZhimaTypeDw
	ZhimaTypePg
	ZhimaTypeYs
	ZhimaTypeDh

	zmALL zhiMaProxyType = 33
)

// ZhiMaUnifyFetcher 芝麻代理对请求 ip 有并发控制, 所以维护一个全局的获取 ip 的方式是必要的
// See https://github.com/anxiwuyanzu/openscraper-framework/proxy-relay/tree/v2-feature-get-ip
type ZhiMaUnifyFetcher struct {
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
	provider      string
}

func NewZhiMaUnifyFetcher(key string, minimum int, proxyType, topic, city, provider string, protocol proxz.Protocol) proxz.ProxyFetcher {
	logger := dot.Logger().WithFields(logrus.Fields{"pt": proxyType, "topic": topic})

	logger.WithFields(logrus.Fields{
		"minimum":  minimum,
		"provider": provider,
		"city":     city,
	}).Info("ZHIMA Unify Fetcher is init")

	unifyAcquireUrl := dot.Conf().Proxy.ZhiMaAcquireIpServer
	if len(unifyAcquireUrl) == 0 {
		unifyAcquireUrl = "http://zhima-acquire-ip.example.com"
	}

	if len(topic) == 0 {
		topic = "default"
	}

	fetcher := &ZhiMaUnifyFetcher{
		key:           key,
		getIpUrl:      unifyAcquireUrl,
		proxies:       make(chan *proxz.ProxyIp, minimum*15),
		city:          city,
		minimum:       minimum,
		proxyProtocol: protocol,
		proxyType:     proxyType,
		provider:      provider,
		topic:         topic,
		logger:        logger,
	}
	fetcher.SetWhiteIp()
	return fetcher
}

func (f *ZhiMaUnifyFetcher) SetWhiteIp() {
	// TrySetWhiteIp(setZhiMaWhiteIp)
}

func (f *ZhiMaUnifyFetcher) Key() string {
	return ProviderZhiMaRelay + "/" + f.key
}

func (f *ZhiMaUnifyFetcher) ProxyCh() chan *proxz.ProxyIp {
	return f.proxies
}

func (f *ZhiMaUnifyFetcher) TryFetchProxy() {
	if len(f.proxies) < f.minimum {
		go f.FetchProxy(f.proxyType)
	}
}

func (f *ZhiMaUnifyFetcher) FetchProxy(proxyType string) {
	if !atomic.CompareAndSwapUint32(&f.getIpState, 0, 1) {
		return
	}

	defer atomic.CompareAndSwapUint32(&f.getIpState, 1, 0)

	start := time.Now()
	var uri string
	if len(f.city) > 0 {
		uri = fmt.Sprintf("%s/get_city_ip?city=%s&proxy_type=%s&size=%d&provider=%s",
			f.getIpUrl, f.city, proxyType, f.minimum, f.provider)
	} else {
		uri = fmt.Sprintf("%s/get_ip?topic=%s&proxy_type=%s&size=%d&provider=%s",
			f.getIpUrl, f.topic, proxyType, f.minimum, f.provider)
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
		"took": time.Since(start).Milliseconds(),
		"cnt":  len(proxies),
		"pt":   proxyType,
	}).Info("request proxy from zhima-unify success")

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
			Provider: proxy.Get("provider").String(),
			Protocol: f.proxyProtocol,
			AuthUser: proxy.Get("auth_user").String(),
			AuthPwd:  proxy.Get("auth_pwd").String(),
		}
	}
}

func (f *ZhiMaUnifyFetcher) FetchByIpLocation(city, ip string) *proxz.ProxyIp {
	start := time.Now()
	uri := fmt.Sprintf("%s/by_ip_location?size=1&city=%s&ip=%s&default_city=%s&provider=%s", f.getIpUrl, city, ip, f.city, f.provider)

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
	}).Info("request proxy from zhima-unify success")

	host := proxy.Get("host").String()
	expireTime := proxy.Get("expire").String()
	expire, _ := time.Parse(time.RFC3339, expireTime)

	return &proxz.ProxyIp{
		Host:     host,
		Expire:   expire,
		CityCode: proxy.Get("city").String(),
		Protocol: f.proxyProtocol,
		Provider: proxy.Get("provider").String(),
		AuthUser: proxy.Get("auth_user").String(),
		AuthPwd:  proxy.Get("auth_pwd").String(),
	}
}

var AllCities = map[string]string{"北京市市辖区": "110100", "北京市县": "110200", "天津市市辖区": "120100", "天津市县": "120200", "河北省石家庄市": "130100", "河北省唐山市": "130200", "河北省秦皇岛市": "130300", "河北省邯郸市": "130400", "河北省邢台市": "130500", "河北省保定市": "130600", "河北省张家口市": "130700", "河北省承德市": "130800", "河北省沧州市": "130900", "河北省廊坊市": "131000", "河北省衡水市": "131100", "山西省太原市": "140100", "山西省大同市": "140200", "山西省阳泉市": "140300", "山西省长治市": "140400", "山西省晋城市": "140500", "山西省朔州市": "140600", "山西省晋中市": "140700", "山西省运城市": "140800", "山西省忻州市": "140900", "山西省临汾市": "141000", "山西省吕梁市": "141100", "内蒙古呼和浩特市": "150100", "内蒙古包头市": "150200", "内蒙古乌海市": "150300", "内蒙古赤峰市": "150400", "内蒙古通辽市": "150500", "内蒙古鄂尔多斯市": "150600", "内蒙古呼伦贝尔市": "150700", "内蒙古巴彦淖尔市": "150800", "内蒙古乌兰察布市": "150900", "内蒙古兴安盟": "152200", "辽宁省沈阳市": "210100", "辽宁省大连市": "210200", "辽宁省大连市市辖区": "210201", "辽宁省大连市中山区": "210202", "辽宁省鞍山市": "210300", "辽宁省抚顺市": "210400", "辽宁省本溪市": "210500", "辽宁省丹东市": "210600", "辽宁省锦州市": "210700", "辽宁省锦州市市辖区": "210701", "辽宁省营口市": "210800", "辽宁省阜新市": "210900", "辽宁省辽阳市": "211000", "辽宁省盘锦市": "211100", "辽宁省铁岭市": "211200", "辽宁省朝阳市": "211300", "辽宁省葫芦岛市": "211400", "吉林省长春市": "220100", "吉林省吉林市": "220200", "吉林省四平市": "220300", "吉林省辽源市": "220400", "吉林省通化市": "220500", "吉林省白山市": "220600", "吉林省松原市": "220700", "吉林省白城市": "220800", "吉林省延边朝鲜族自治州": "222400", "黑龙江省哈尔滨市": "230100", "黑龙江省齐齐哈尔市": "230200", "黑龙江省鸡西市": "230300", "黑龙江省鹤岗市": "230400", "黑龙江省双鸭山市": "230500", "黑龙江省大庆市": "230600", "黑龙江省伊春市": "230700", "黑龙江省佳木斯市": "230800", "黑龙江省七台河市": "230900", "黑龙江省牡丹江市": "231000", "黑龙江省黑河市": "231100", "黑龙江省绥化市": "231200", "黑龙江省大兴安岭地区": "232700", "上海市市辖区": "310100", "上海市县": "310200", "江苏省南京市": "320100", "江苏省无锡市": "320200", "江苏省徐州市": "320300", "江苏省常州市": "320400", "江苏省苏州市": "320500", "江苏省南通市": "320600", "江苏省连云港市": "320700", "江苏省连云港市海州区": "320706", "江苏省淮安市": "320800", "江苏省盐城市": "320900", "江苏省扬州市": "321000", "江苏省镇江市": "321100", "江苏省泰州市": "321200", "江苏省宿迁市": "321300", "浙江省杭州市": "330100", "浙江省杭州市萧山区": "330109", "浙江省宁波市": "330200", "浙江省温州市": "330300", "浙江省嘉兴市": "330400", "浙江省嘉兴市市辖区": "330401", "浙江省湖州市": "330500", "浙江省绍兴市": "330600", "浙江省金华市": "330700", "浙江省衢州市": "330800", "浙江省舟山市": "330900", "浙江省台州市": "331000", "浙江省丽水市": "331100", "浙江省丽水市市辖区": "331101", "安徽省合肥市": "340100", "安徽省芜湖市": "340200", "安徽省蚌埠市": "340300", "安徽省淮南市": "340400", "安徽省马鞍山市": "340500", "安徽省淮北市": "340600", "安徽省淮北市市辖区": "340601", "安徽省淮北市烈山区": "340604", "安徽省铜陵市": "340700", "安徽省安庆市": "340800", "安徽省黄山市": "341000", "安徽省滁州市": "341100", "安徽省阜阳市": "341200", "安徽省宿州市": "341300", "安徽省六安市": "341500", "安徽省亳州市": "341600", "安徽省池州市": "341700", "安徽省池州市贵池区": "341702", "安徽省宣城市": "341800", "福建省福州市": "350100", "福建省厦门市": "350200", "福建省莆田市": "350300", "福建省三明市": "350400", "福建省泉州市": "350500", "福建省漳州市": "350600", "福建省漳州市市辖区": "350601", "福建省南平市": "350700", "福建省龙岩市": "350800", "福建省宁德市": "350900", "江西省南昌市": "360100", "江西省景德镇市": "360200", "江西省萍乡市": "360300", "江西省上栗县": "360322", "江西省九江市": "360400", "江西省新余市": "360500", "江西省鹰潭市": "360600", "江西省赣州市": "360700", "江西省宜春市": "360900", "江西省抚州市": "361000", "江西省上饶市": "361100", "山东省济南市": "370100", "山东省济南市市辖区": "370101", "山东省青岛市": "370200", "山东省淄博市": "370300", "山东省枣庄市": "370400", "山东省东营市": "370500", "山东省烟台市": "370600", "山东省潍坊市": "370700", "山东省济宁市": "370800", "山东省泰安市": "370900", "山东省威海市": "371000", "山东省日照市": "371100", "山东省莱芜市": "371200", "山东省临沂市": "371300", "山东省德州市": "371400", "山东省聊城市": "371500", "山东省滨州市": "371600", "山东省菏泽市": "371700", "河南省郑州市": "410100", "河南省开封市": "410200", "河南省洛阳市": "410300", "河南省平顶山市": "410400", "河南省安阳市": "410500", "河南省鹤壁市": "410600", "河南省新乡市": "410700", "河南省焦作市": "410800", "河南省濮阳市": "410900", "河南省许昌市市辖区": "411001", "河南省漯河市": "411100", "河南省三门峡市": "411200", "河南省南阳市": "411300", "河南省商丘市": "411400", "河南省信阳市": "411500", "河南省周口市": "411600", "河南省驻马店市": "411700", "湖北省武汉市": "420100", "湖北省武汉市市辖区": "420101", "湖北省黄石市": "420200", "湖北省十堰市": "420300", "湖北省宜昌市": "420500", "湖北省襄阳": "420600", "湖北省鄂州市": "420700", "湖北省荆门市": "420800", "湖北省荆门市市辖区": "420801", "湖北省孝感市": "420900", "湖北省荆州市": "421000", "湖北省黄冈市": "421100", "湖北省咸宁市": "421200", "湖北省随州市": "421300", "湖南省长沙市": "430100", "湖南省株洲市": "430200", "湖南省湘潭市": "430300", "湖南省衡阳市": "430400", "湖南省邵阳市": "430500", "湖南省岳阳市": "430600", "湖南省常德市": "430700", "湖南省张家界市": "430800", "湖南省益阳市": "430900", "湖南省郴州市": "431000", "湖南省永州市": "431100", "湖南省娄底市": "431300", "湖南省吉首市": "433101", "广东省广州市": "440100", "广东省广州市市辖区": "440101", "广东省韶关市": "440200", "广东省深圳市": "440300", "广东省深圳市盐田区": "440308", "广东省珠海市": "440400", "广东省珠海市市辖区": "440401", "广东省汕头市": "440500", "广东省佛山市": "440600", "广东省佛山市市辖区": "440601", "广东省江门市": "440700", "广东省湛江市": "440800", "广东省茂名市": "440900", "广东省肇庆市": "441200", "广东省惠州市": "441300", "广东省梅州市": "441400", "广东省汕尾市": "441500", "广东省河源市": "441600", "广东省阳江市": "441700", "广东省清远市": "441800", "广东省东莞市": "441900", "广东省中山市": "442000", "广东省潮州市": "445100", "广东省揭阳市": "445200", "广东省云浮市": "445300", "广西南宁市": "450100", "广西柳州市": "450200", "广西桂林市": "450300", "广西梧州市": "450400", "广西北海市": "450500", "广西防城港市": "450600", "广西钦州市": "450700", "广西贵港市": "450800", "广西玉林市": "450900", "广西百色市": "451000", "广西贺州市": "451100", "广西河池市": "451200", "广西来宾市": "451300", "广西崇左市": "451400", "海南省海口市": "460100", "海南省三亚市": "460200", "重庆市县": "500200", "重庆市市": "500300", "四川省成都市": "510100", "四川省自贡市": "510300", "四川省攀枝花市": "510400", "四川省泸州市": "510500", "四川省德阳市": "510600", "四川省绵阳市": "510700", "四川省绵阳市市辖区": "510701", "四川省广元市": "510800", "四川省遂宁市": "510900", "四川省内江市": "511000", "四川省乐山市": "511100", "四川省南充市": "511300", "四川省眉山市": "511400", "四川省宜宾市": "511500", "四川省广安市": "511600", "四川省达州市": "511700", "四川省雅安市": "511800", "四川省巴中市": "511900", "四川省资阳市": "512000", "四川省甘孜藏族自治州": "513300", "四川省凉山彝族自治州": "513400", "四川省西昌市": "513401", "贵州省贵阳市": "520100", "贵州省六盘水市": "520200", "贵州省遵义市": "520300", "贵州省安顺市": "520400", "云南省昆明市": "530100", "云南省曲靖市": "530300", "云南省楚雄市": "532301", "西藏拉萨市": "540100", "陕西省西安市": "610100", "陕西省西安市市辖区": "610101", "陕西省铜川市": "610200", "陕西省宝鸡市": "610300", "陕西省咸阳市": "610400", "陕西省渭南市": "610500", "陕西省延安市": "610600", "陕西省汉中市": "610700", "陕西省榆林市": "610800", "陕西省安康市": "610900", "陕西省商洛市": "611000", "甘肃省兰州市": "620100", "青海省西宁市": "630100", "青海省格尔木市": "632801", "宁夏银川市": "640100", "宁夏石嘴山市": "640200", "宁夏固原市": "640400", "宁夏中卫市": "640500", "新疆乌鲁木齐市": "650100", "新疆克拉玛依市": "650200", "新疆哈密市": "652201", "新疆昌吉市": "652301", "新疆博尔塔拉蒙古自治州": "652700", "新疆和田市": "653201", "新疆伊犁哈萨克自治州": "654000", "新疆塔城市": "654201"}
var zhiMaCitiesArr = []string{
	"350000", "420000", "330000", "320000", "340000",
}

func ZhiMaGetRandomCity() string {
	return zhiMaCitiesArr[rand.Intn(len(zhiMaCitiesArr))]
}
