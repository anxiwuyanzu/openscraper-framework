package reqwest

import (
	"bytes"
	"context"
	"github.com/anxiwuyanzu/openscraper-framework/v4/util/compress"
	http "github.com/bogdanfinn/fhttp"
	"github.com/valyala/fasthttp"
	"io"
	http2 "net/http"
	"net/url"
	"strings"
)

type ITlsRequest interface {
	SetResponse(resp *http.Response) error
	GetResponse() *http.Response
	GetRequest() *http.Request
}

type TlsRequest struct {
	*http.Request
	response  *http.Response
	respBody  []byte
	headers   map[string]string // 存储所有要设置的header
	cookieStr string            // 直接存储cookie字符串
}

var defaultHeaderOrder = []string{
	"sec-ch-ua-platform",
	"user-agent",
	"accept",
	"x-secsdk-csrf-token",
	"content-type",
	"sec-ch-ua",
	"sec-ch-ua-mobile",
	"origin",
	"sec-fetch-site",
	"sec-fetch-mode",
	"sec-fetch-dest",
	"sec-fetch-storage-access",
	"referer",
	"accept-encoding",
	"accept-language",
	"cookie",
	"priority",
}

func NewTlsRequest() *TlsRequest {
	req := &http.Request{
		Header: http.Header{},
		URL:    new(url.URL),
	}

	return &TlsRequest{
		Request: req,
		headers: make(map[string]string),
	}
}

func (r *TlsRequest) SetResponse(resp *http.Response) error {
	if r.response != nil && r.response.Body != nil {
		r.response.Body.Close()
	}
	r.response = resp
	// body 需要提前读出来
	_, err := r.ResponseBody()
	return err
}

func (r *TlsRequest) GetResponse() *http.Response {
	return r.response
}

func (r *TlsRequest) GetRequest() *http.Request {
	return r.Request
}

// Clone TODO https://stackoverflow.com/questions/62017146/http-request-clone-is-not-deep-clone
func (r *TlsRequest) Clone() Request {
	req := r.Request.Clone(context.Background())
	return &TlsRequest{
		Request: req,
	}
}

func (r *TlsRequest) Close() {
	if r.response != nil {
		r.response.Body.Close()
	}
}

func (r *TlsRequest) SetMethod(method string) {
	r.Request.Method = method
}

func (r *TlsRequest) SetRequestURI(uri string) {
	parsedURL, _ := url.Parse(uri)
	r.Request.URL = parsedURL
}

func (r *TlsRequest) SetQueryString(query string) {
	r.Request.URL.RawQuery = query
}

func (r *TlsRequest) SetBodyBytes(body []byte) {
	r.Request.GetBody = func() (io.ReadCloser, error) {
		return io.NopCloser(bytes.NewReader(body)), nil
	}
	reqBody, _ := r.Request.GetBody()
	r.Request.Body = reqBody
}

func (r *TlsRequest) SetUserAgent(ua string) {
	r.SetHeader(fasthttp.HeaderUserAgent, ua)
}

func (r *TlsRequest) GetRequestURI() string {
	return r.Request.URL.RequestURI()
}

func (r *TlsRequest) GetFullURI() string {
	return r.Request.URL.String()
}

func (r *TlsRequest) SetHeader(key, value string) {
	// 只存储header，不立即设置
	r.headers[key] = value

	// 收集所有header keys
	var orderedKeys []string
	existingKeys := make(map[string]bool)

	// 按预定义顺序添加存在的header
	for _, k := range defaultHeaderOrder {
		// 大小写不敏感比较
		for headerKey := range r.headers {
			if strings.EqualFold(headerKey, k) {
				orderedKeys = append(orderedKeys, headerKey)
				existingKeys[headerKey] = true
				break
			}
		}
	}

	// 将不在预定义顺序中的header添加到末尾
	for k := range r.headers {
		if !existingKeys[k] {
			orderedKeys = append(orderedKeys, k)
		}
	}

	// 创建新的header map
	newHeaders := make(http.Header)

	// 按顺序设置header
	for _, k := range orderedKeys {
		newHeaders[k] = []string{r.headers[k]}
	}

	// 设置header顺序
	newHeaders[http.HeaderOrderKey] = orderedKeys

	// 一次性替换所有header
	r.Request.Header = newHeaders
}

func (r *TlsRequest) SetCookie(key, value string) {
	r.cookieStr += key + "=" + value + "; "
	r.SetHeader("Cookie", r.cookieStr)
}

func (r *TlsRequest) DelHeader(key string) {
	r.Request.Header.Del(key)
	delete(r.Request.Header, key)
}

func (r *TlsRequest) DelCookie(key string) {
	// 分割所有cookie
	cookies := strings.Split(r.cookieStr, "; ")
	var newCookies []string

	// 过滤掉要删除的cookie
	for _, cookie := range cookies {
		if !strings.HasPrefix(cookie, key+"=") {
			newCookies = append(newCookies, cookie)
		}
	}

	// 重新组合cookie字符串
	r.cookieStr = strings.Join(newCookies, "; ")
	if r.cookieStr != "" {
		r.SetHeader("Cookie", r.cookieStr)
	} else {
		delete(r.headers, "Cookie")
	}
}

func (r *TlsRequest) SetQueryArg(key, value string) {
	r.Request.URL.RawQuery = string(setKeyValue([]byte(r.Request.URL.RawQuery), key, value))
}

func (r *TlsRequest) SetQueryArgEscape(key, value string) {
	r.SetQueryArg(key, url.QueryEscape(value))
}

func (r *TlsRequest) GetQueryString() string {
	return r.Request.URL.RawQuery
}

func (r *TlsRequest) GetMethod() string {
	return r.Request.Method
}

func (r *TlsRequest) GetBody() []byte {
	if r.Request.GetBody == nil {
		return nil
	}
	reqBody, _ := r.Request.GetBody()
	if reqBody == nil {
		return nil
	}
	body, _ := io.ReadAll(reqBody)
	return body
}

func (r *TlsRequest) GetHeader(key string) string {
	if v, ok := r.Request.Header[key]; ok {
		if len(v) > 0 {
			return v[0]
		}
	}
	return ""
}

func (r *TlsRequest) GetUserAgent() string {
	return r.Request.Header.Get(fasthttp.HeaderUserAgent)
}

func (r *TlsRequest) GetHost() string {
	return r.Request.URL.Host
}

func (r *TlsRequest) GetPath() string {
	return r.Request.URL.Path
}

func (r *TlsRequest) GetQueryArg(key string) string {
	qs := []byte(r.Request.URL.RawQuery)
	return string(getKeyValue(qs, key))
}

func (r *TlsRequest) VisitAllQueryArg(f func(key, value []byte)) {
	qs := []byte(r.Request.URL.RawQuery)

	s := argsScanner{b: qs}
	kv := &argsKV{}
	for s.next(kv) {
		f(kv.key, kv.value)
	}
}

func (r *TlsRequest) VisitAllHeader(f func(key, value []byte)) {
	headers := r.Request.Header
	for k, v := range headers {
		f([]byte(k), []byte(v[0]))
	}
}

func (r *TlsRequest) VisitAllCookie(f func(key, value []byte)) {
	cookies := r.Request.Cookies()
	for _, cookie := range cookies {
		f([]byte(cookie.Name), []byte(cookie.Value))
	}
}

// ResponseBody 读取body, 优化读取方式
func (r *TlsRequest) ResponseBody() (body []byte, err error) {
	if r.respBody != nil {
		return r.respBody, nil
	}
	body, err = compress.ReadAll(r.response.Body)
	r.respBody = body

	return r.respBody, err
}

func (r *TlsRequest) ResponseHeader(key string) string {
	return r.response.Header.Get(key)
}

func (r *TlsRequest) ResponseCookie(key string) string {
	cookies := r.response.Cookies()
	for _, c := range cookies {
		if c.Name == key {
			return c.Value
		}
	}
	return ""
}

func (r *TlsRequest) ResponseStatusCode() int {
	return r.response.StatusCode
}

func (r *TlsRequest) VisitAllRespHeader(f func(key, value []byte)) {
	for k, v := range r.response.Header {
		f([]byte(k), []byte(v[0]))
	}
}

func (r *TlsRequest) VisitAllRespCookie(f func(key, value []byte)) {
	for _, cookie := range r.response.Cookies() {
		f([]byte(cookie.Name), []byte(cookie.Value))
	}
}

func (r *TlsRequest) GetRespCookies() []*http2.Cookie {
	cookies := r.response.Cookies()
	standardCookies := make([]*http2.Cookie, len(cookies))

	for i, cookie := range cookies {
		standardCookies[i] = &http2.Cookie{
			Name:       cookie.Name,
			Value:      cookie.Value,
			Path:       cookie.Path,
			Domain:     cookie.Domain,
			Expires:    cookie.Expires,
			RawExpires: cookie.RawExpires,
			MaxAge:     cookie.MaxAge,
			Secure:     cookie.Secure,
			HttpOnly:   cookie.HttpOnly,
			SameSite:   http2.SameSite(cookie.SameSite),
			Raw:        cookie.Raw,
			Unparsed:   cookie.Unparsed,
		}
	}
	return standardCookies
}
