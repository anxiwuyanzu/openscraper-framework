package reqwest

import (
	"context"
	"github.com/valyala/fasthttp"
	"github.com/valyala/fasthttp/fasthttpproxy"
	goproxy "golang.org/x/net/proxy"
	"net"
	"strings"
	"time"
)

type DialFunc func(ctx context.Context, network, addr string) (net.Conn, error)

func (f DialFunc) ToFastHttpDialFunc() fasthttp.DialFunc {
	return func(addr string) (net.Conn, error) {
		return f(context.Background(), "tcp", addr)
	}
}

func HttpProxyDialer(proxy string, timeout time.Duration) DialFunc {
	dialer := fasthttpproxy.FasthttpHTTPDialerTimeout(proxy, timeout)
	return func(ctx context.Context, network, addr string) (net.Conn, error) {
		return dialer(addr)
	}
}

// SocksDialer dial socks
func SocksDialer(proxy string, timeout time.Duration) (DialFunc, error) {
	var auth *goproxy.Auth
	if strings.Contains(proxy, "@") {
		split := strings.Split(proxy, "@")
		authSplit := strings.Split(split[0], ":")
		if len(authSplit) >= 2 {
			auth = &goproxy.Auth{User: authSplit[0], Password: authSplit[1]}
		} else {
			auth = &goproxy.Auth{User: split[0]}
		}

		proxy = split[1]
	}

	dialer, err := goproxy.SOCKS5("tcp", proxy, auth, goproxy.Direct)
	if err != nil {
		return nil, err
	}

	return func(ctx context.Context, network, addr string) (net.Conn, error) {
		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		defer cancel()
		return dialer.(goproxy.ContextDialer).DialContext(ctx, "tcp", addr)
	}, nil
}
