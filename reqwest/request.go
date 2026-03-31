package reqwest

import (
	"github.com/anxiwuyanzu/openscraper-framework/v4/dot"
	"net/http"
	"strings"
)

const (
	Gzip    = "gzip"
	Br      = "br"
	Inflate = "inflate"
)

type Request interface {
	Close()
	Clone() Request

	SetMethod(method string)
	SetRequestURI(uri string)
	SetQueryString(query string)
	SetBodyBytes(body []byte)
	SetHeader(key string, value string)
	DelHeader(key string)
	SetCookie(key string, value string)
	DelCookie(key string)
	SetQueryArg(key string, value string)
	SetQueryArgEscape(key string, value string)
	SetUserAgent(ua string)

	GetFullURI() string
	GetRequestURI() string
	GetQueryString() string
	GetMethod() string
	GetBody() []byte
	GetHeader(key string) string
	GetUserAgent() string
	GetHost() string
	GetPath() string
	GetQueryArg(key string) string
	VisitAllQueryArg(f func(key, value []byte))
	VisitAllHeader(f func(key, value []byte))
	VisitAllCookie(f func(key, value []byte))

	ResponseBody() ([]byte, error)
	ResponseHeader(key string) string
	ResponseStatusCode() int
	ResponseCookie(key string) string
	VisitAllRespHeader(f func(key, value []byte))
	VisitAllRespCookie(f func(key, value []byte))
	GetRespCookies() []*http.Cookie
}

func NewRequest() Request {
	if dot.Conf() == nil || dot.Conf().Reqwest.Client == "net/http" {
		return NewStandardRequest()
	}
	if dot.Conf().Reqwest.Client == "tlshttp" {
		return NewTlsRequest()
	}
	return NewFastHttpRequest()
}

func ParseCookieKeyValue(cookieStr string) map[string]string {
	splits := strings.Split(cookieStr, ";")
	keyValue := make(map[string]string, len(splits))
	for _, spl := range splits {
		cookie := strings.TrimSpace(spl)
		kvSpl := strings.Split(cookie, "=")
		if len(kvSpl) == 2 {
			keyValue[kvSpl[0]] = kvSpl[1]
		}
	}
	return keyValue
}
