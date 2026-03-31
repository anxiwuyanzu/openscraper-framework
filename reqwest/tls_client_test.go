package reqwest

import (
	"fmt"
	"github.com/anxiwuyanzu/openscraper-framework/v4/dot"
	"github.com/anxiwuyanzu/openscraper-framework/v4/reqwest/proxz"
	"testing"
	"time"
)

func TestNewTlsClient(t *testing.T) {
	dot.ConfigViper("", ".")
	pconfig, err := proxz.ParseConfigFromString("socks5://qingguo-relay?topic=jx-gmv&size=1")
	httpOption := &Option{
		Reqwest:     DefaultOption().Reqwest,
		ProxyConfig: pconfig,
	}
	client := NewClient(httpOption)
	for i := 0; i < 3; i++ {
		request := NewRequest()
		request.SetRequestURI("https://tls.browserleaks.com/json")
		request.SetMethod("GET")
		request.SetBodyBytes(nil)
		request.SetCookie("1", "13")
		request.SetCookie("2", "13")
		request.DelCookie("2")
		request.SetQueryString("queyr=1")
		request.SetQueryArg("123", "1")
		request.SetUserAgent("111")
		request.SetHeader("Accept-Encoding", "gzip, deflate, br")
		start := time.Now()
		err = client.DoRequest(request)
		fmt.Println(err, time.Since(start))
		//client.ReleaseProxy()
		if err != nil {
			t.Fatal(err)
		}

		body, err := request.ResponseBody()
		if err != nil {
			t.Fatal(err)
		}
		fmt.Println(string(body))
	}
}
