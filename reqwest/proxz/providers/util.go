package providers

import (
	"context"
	"errors"
	"fmt"
	"net"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/anxiwuyanzu/openscraper-framework/spider-common-go/v4/dot"
	log "github.com/sirupsen/logrus"
	"github.com/valyala/fasthttp"
)

var (
	myIp         string
	getMyIPLock  = sync.Mutex{}
	whiteIpState uint32
	//TimeZone, _  = time.LoadLocation("Asia/Shanghai")
)

const (
	ProviderZhiMaRelay   = "zhima-relay"
	ProviderQingGuoRelay = "qingguo-relay"
)

//func init() {
//	TimeZone, _ = time.LoadLocation("Asia/Shanghai")
//}

func getMyIP() (string, error) {
	getMyIPLock.Lock()
	defer getMyIPLock.Unlock()

	if len(myIp) > 0 {
		return myIp, nil
	}

	ipUrls := dot.Conf().Proxy.GetLocalIpUris
	if len(ipUrls) == 0 || ipUrls[0] == "" {
		ipUrls = []string{
			"http://ms.cds8.cn/ip",
			// "http://106.15.37.110:5001/",
		}
	}

	ch := make(chan string)
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	for _, ipUrl := range ipUrls {
		url := ipUrl
		go getIP(url, ch)
	}
	for {
		select {
		case <-ctx.Done():
			return "", errors.New(fmt.Sprintf("failed to get ip from all sources: %v", ipUrls))
		case ip := <-ch:
			myIp = ip
			return ip, nil
		}
	}
}

func getIP(url string, ch chan string) {
	req := fasthttp.AcquireRequest()
	defer fasthttp.ReleaseRequest(req)
	req.Header.SetMethod(fasthttp.MethodGet)
	req.SetRequestURI(url)

	res := fasthttp.AcquireResponse()
	defer fasthttp.ReleaseResponse(res)
	if err := fasthttp.DoTimeout(req, res, 15*time.Second); err != nil {
		log.WithError(err).Errorf("failed to get own IP from %s.", url)
		return
	}
	ownIp := strings.Replace(string(res.Body()), "\n", "", -1)
	address := net.ParseIP(ownIp)
	if address == nil {
		log.Errorf("wrong IP format %s from %s", ownIp, url)
		return
	}
	log.Infof("successfully get own IP %s from %s.", ownIp, url)
	ch <- ownIp
}

func TrySetWhiteIp(inner func() bool) {
	if !atomic.CompareAndSwapUint32(&whiteIpState, 0, 1) {
		return
	}

	n := 0
	for {
		if n > 10 {
			panic(fmt.Sprintf("failed to set white ip, retry %d times", n))
		}
		if n > 0 {
			time.Sleep(time.Duration(n) * time.Second)
		}
		n++

		if inner() {
			return
		}
	}
}
