package reqwest

import (
	"bytes"
	"context"
	"fmt"
	"github.com/anxiwuyanzu/openscraper-framework/v4/util/compress"
	"github.com/valyala/fasthttp"
	"io"
	"net/http"
	"net/url"
)

type IStandardRequest interface {
	SetResponse(resp *http.Response) error
	GetResponse() *http.Response
	GetRequest() *http.Request
}

// StandardRequest wraps http.Request
type StandardRequest struct {
	*http.Request
	response *http.Response
	respBody []byte
}

func NewStandardRequest() *StandardRequest {
	req := &http.Request{
		Header: http.Header{},
		URL:    new(url.URL),
	}

	return &StandardRequest{
		Request: req,
	}
}

func (r *StandardRequest) SetResponse(resp *http.Response) error {
	if r.response != nil && r.response.Body != nil {
		r.response.Body.Close()
	}
	r.response = resp
	// body 需要提前读出来
	_, err := r.ResponseBody()
	return err
}

func (r *StandardRequest) GetResponse() *http.Response {
	return r.response
}

func (r *StandardRequest) GetRequest() *http.Request {
	return r.Request
}

// Clone TODO https://stackoverflow.com/questions/62017146/http-request-clone-is-not-deep-clone
func (r *StandardRequest) Clone() Request {
	req := r.Request.Clone(context.Background())
	return &StandardRequest{
		Request: req,
	}
}

func (r *StandardRequest) Close() {
	if r.response != nil {
		r.response.Body.Close()
	}
}

func (r *StandardRequest) SetMethod(method string) {
	r.Request.Method = method
}

func (r *StandardRequest) SetRequestURI(uri string) {
	r.Request.URL, _ = url.Parse(uri)
}

func (r *StandardRequest) SetQueryString(query string) {
	r.Request.URL.RawQuery = query
}

func (r *StandardRequest) SetBodyBytes(body []byte) {
	r.Request.GetBody = func() (io.ReadCloser, error) {
		return io.NopCloser(bytes.NewReader(body)), nil
	}
	reqBody, _ := r.Request.GetBody()
	r.Request.Body = reqBody
}

func (r *StandardRequest) SetUserAgent(ua string) {
	r.SetHeader(fasthttp.HeaderUserAgent, ua)
}

func (r *StandardRequest) GetRequestURI() string {
	return r.Request.URL.RequestURI()
}

func (r *StandardRequest) GetFullURI() string {
	return r.Request.URL.String()
}

func (r *StandardRequest) SetHeader(key, value string) {
	r.Request.Header[key] = []string{value}
	//r.Request.Header.Set(key, value)  // this will canonicalized key
}

func (r *StandardRequest) SetCookie(key, value string) {
	r.DelCookie(key)
	r.Request.AddCookie(&http.Cookie{Name: key, Value: value})
}

func (r *StandardRequest) DelHeader(key string) {
	r.Request.Header.Del(key)
	delete(r.Request.Header, key)
}

func (r *StandardRequest) DelCookie(key string) {
	cookies := r.Request.Cookies()
	var cookieJoined string
	var has bool
	for _, c := range cookies {
		if c.Name == key {
			has = true
			continue
		}
		if len(cookieJoined) == 0 {
			cookieJoined = fmt.Sprintf("%s=%s", c.Name, c.Value)
		} else {
			cookieJoined += fmt.Sprintf("; %s=%s", c.Name, c.Value)
		}
	}
	if !has {
		return
	}
	r.Request.Header.Set("Cookie", cookieJoined)
}

func (r *StandardRequest) SetQueryArg(key, value string) {
	r.Request.URL.RawQuery = string(setKeyValue([]byte(r.Request.URL.RawQuery), key, value))
}

func (r *StandardRequest) SetQueryArgEscape(key, value string) {
	r.SetQueryArg(key, url.QueryEscape(value))
}

func (r *StandardRequest) GetQueryString() string {
	return r.Request.URL.RawQuery
}

func (r *StandardRequest) GetMethod() string {
	return r.Request.Method
}

func (r *StandardRequest) GetBody() []byte {
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

func (r *StandardRequest) GetHeader(key string) string {
	if v, ok := r.Request.Header[key]; ok {
		if len(v) > 0 {
			return v[0]
		}
	}
	return ""
}

func (r *StandardRequest) GetUserAgent() string {
	return r.Request.Header.Get(fasthttp.HeaderUserAgent)
}

func (r *StandardRequest) GetHost() string {
	return r.Request.URL.Host
}

func (r *StandardRequest) GetPath() string {
	return r.Request.URL.Path
}

func (r *StandardRequest) GetQueryArg(key string) string {
	qs := []byte(r.Request.URL.RawQuery)
	return string(getKeyValue(qs, key))
}

func (r *StandardRequest) VisitAllQueryArg(f func(key, value []byte)) {
	qs := []byte(r.Request.URL.RawQuery)

	s := argsScanner{b: qs}
	kv := &argsKV{}
	for s.next(kv) {
		f(kv.key, kv.value)
	}
}

func (r *StandardRequest) VisitAllHeader(f func(key, value []byte)) {
	headers := r.Request.Header
	for k, v := range headers {
		f([]byte(k), []byte(v[0]))
	}
}

func (r *StandardRequest) VisitAllCookie(f func(key, value []byte)) {
	cookies := r.Request.Cookies()
	for _, cookie := range cookies {
		f([]byte(cookie.Name), []byte(cookie.Value))
	}
}

// ResponseBody 读取body, 优化读取方式
func (r *StandardRequest) ResponseBody() (body []byte, err error) {
	if r.respBody != nil {
		return r.respBody, nil
	}
	ce := r.response.Header.Get(fasthttp.HeaderContentEncoding)
	if ce == Gzip {
		body, err = compress.UnGzipReader(r.response.Body)
	} else if ce == Br {
		body, err = compress.UnBrotliReader(r.response.Body)
	} else if ce == Inflate {
		body, err = compress.UnFlateReader(r.response.Body)
	} else {
		// 对比 ioutil.ReadAll: https://mp.weixin.qq.com/s/e2A3ME4vhOK2S3hLEJtPsw
		body, err = compress.ReadAll(r.response.Body)
	}

	r.respBody = body

	return r.respBody, err
}

func (r *StandardRequest) ResponseHeader(key string) string {
	return r.response.Header.Get(key)
}

func (r *StandardRequest) ResponseCookie(key string) string {
	cookies := r.response.Cookies()
	for _, c := range cookies {
		if c.Name == key {
			return c.Value
		}
	}
	return ""
}

func (r *StandardRequest) ResponseStatusCode() int {
	return r.response.StatusCode
}

func (r *StandardRequest) VisitAllRespHeader(f func(key, value []byte)) {
	for k, v := range r.response.Header {
		f([]byte(k), []byte(v[0]))
	}
}

func (r *StandardRequest) VisitAllRespCookie(f func(key, value []byte)) {
	for _, cookie := range r.response.Cookies() {
		f([]byte(cookie.Name), []byte(cookie.Value))
	}
}

func (r *StandardRequest) GetRespCookies() []*http.Cookie {
	return r.response.Cookies()
}
