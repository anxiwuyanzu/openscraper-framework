package reqwest

import (
	"bytes"
	"github.com/valyala/fasthttp"
	"net/http"
	"net/url"
)

type IFastHttpRequest interface {
	GetResponse() *fasthttp.Response
	GetRequest() *fasthttp.Request
}

// FastHttpRequest wraps fasthttp.Request
type FastHttpRequest struct {
	*fasthttp.Request
	response *fasthttp.Response
	respBody []byte
}

func NewFastHttpRequest() *FastHttpRequest {
	request := fasthttp.AcquireRequest()
	request.Header.DisableNormalizing()
	response := fasthttp.AcquireResponse()

	return &FastHttpRequest{
		Request:  request,
		response: response,
	}
}

func (r *FastHttpRequest) GetResponse() *fasthttp.Response {
	return r.response
}

func (r *FastHttpRequest) GetRequest() *fasthttp.Request {
	return r.Request
}

func (r *FastHttpRequest) Clone() Request {
	req := fasthttp.AcquireRequest()
	r.Request.CopyTo(req)

	return &FastHttpRequest{
		Request:  req,
		response: r.response,
	}
}

func (r *FastHttpRequest) Close() {
	fasthttp.ReleaseRequest(r.Request)

	if r.response != nil {
		fasthttp.ReleaseResponse(r.response)
	}
}

func (r *FastHttpRequest) SetMethod(method string) {
	r.Request.Header.SetMethod(method)
}

func (r *FastHttpRequest) SetRequestURI(uri string) {
	r.Request.SetRequestURI(uri)
}

func (r *FastHttpRequest) SetQueryString(query string) {
	r.Request.URI().SetQueryString(query)
}

func (r *FastHttpRequest) SetBodyBytes(body []byte) {
	r.Request.SetBody(body)
}

func (r *FastHttpRequest) SetHeader(key, value string) {
	r.Request.Header.Set(key, value)
}

func (r *FastHttpRequest) SetCookie(key, value string) {
	r.Request.Header.SetCookie(key, value)
}

func (r *FastHttpRequest) DelHeader(key string) {
	r.Request.Header.Del(key)
}

func (r *FastHttpRequest) DelCookie(key string) {
	r.Request.Header.DelCookie(key)
}

// SetQueryArg 没有使用 r.Request.URI().QueryArgs().Set(key, value); 因为不想被标准化处理
func (r *FastHttpRequest) SetQueryArg(key, value string) {
	qs := setKeyValue(r.Request.URI().QueryString(), key, value)
	r.Request.URI().SetQueryStringBytes(qs)
}

func (r *FastHttpRequest) SetQueryArgEscape(key, value string) {
	r.SetQueryArg(key, url.QueryEscape(value))
}

func (r *FastHttpRequest) SetUserAgent(ua string) {
	r.Request.Header.SetUserAgent(ua)
}

func (r *FastHttpRequest) GetRequestURI() string {
	// return string(r.Request.RequestURI())
	return string(r.Request.URI().RequestURI())
}

func (r *FastHttpRequest) GetFullURI() string {
	return string(r.Request.URI().FullURI())
}

func (r *FastHttpRequest) GetQueryString() string {
	return string(r.Request.URI().QueryString())
}

func (r *FastHttpRequest) GetMethod() string {
	return string(r.Request.Header.Method())
}

func (r *FastHttpRequest) GetBody() []byte {
	return r.Request.Body()
}

func (r *FastHttpRequest) GetHeader(key string) string {
	return string(r.Request.Header.Peek(key))
}

func (r *FastHttpRequest) GetUserAgent() string {
	return string(r.Request.Header.UserAgent())
}

func (r *FastHttpRequest) GetHost() string {
	return string(r.Request.URI().Host())
}

func (r *FastHttpRequest) GetPath() string {
	return string(r.Request.URI().Path())
}

func (r *FastHttpRequest) GetQueryArg(key string) string {
	qs := r.Request.URI().QueryString()
	return string(getKeyValue(qs, key))
}

func (r *FastHttpRequest) VisitAllQueryArg(f func(key, value []byte)) {
	qs := r.Request.URI().QueryString()

	s := argsScanner{b: qs}
	kv := &argsKV{}
	for s.next(kv) {
		f(kv.key, kv.value)
	}
}

func (r *FastHttpRequest) VisitAllHeader(f func(key, value []byte)) {
	r.Request.Header.VisitAll(f)
}

func (r *FastHttpRequest) VisitAllCookie(f func(key, value []byte)) {
	r.Request.Header.VisitAllCookie(f)
}

func (r *FastHttpRequest) ResponseBody() ([]byte, error) {
	if r.respBody != nil {
		return r.respBody, nil
	}

	var err error
	r.respBody, err = ReadFastHttpBody(r.response)
	return r.respBody, err
}

func (r *FastHttpRequest) ResponseHeader(key string) string {
	return string(r.response.Header.Peek(key))
}

func (r *FastHttpRequest) ResponseCookie(key string) string {
	cookie := r.response.Header.PeekCookie(key)
	if len(cookie) == 0 {
		return ""
	}
	n := bytes.IndexByte(cookie, ';')
	return string(cookie[len(key)+1 : n])
}

func (r *FastHttpRequest) ResponseStatusCode() int {
	return r.response.StatusCode()
}

func (r *FastHttpRequest) VisitAllRespHeader(f func(key, value []byte)) {
	r.response.Header.VisitAll(f)
}

func (r *FastHttpRequest) VisitAllRespCookie(f func(key, value []byte)) {
	ff := func(key, value []byte) {
		// like: BUYIN_SASID=SID2_7370118439719551259; Path=/; Domain=jinritemai.com; Max-Age=259200
		if pos := bytes.Index(value, []byte{';'}); pos > 0 {
			f(key, value[len(key)+1:pos])
		} else {
			f(key, value)
		}
	}
	r.response.Header.VisitAllCookie(ff)
}

func (r *FastHttpRequest) GetRespCookies() []*http.Cookie {
	var cookies []*http.Cookie
	ff := func(key, value []byte) {
		h := http.Header{}
		h.Add("Set-Cookie", string(value))
		resp := http.Response{Header: h}
		ck := resp.Cookies()
		if len(ck) > 0 {
			cookies = append(cookies, ck[0])
		}
	}
	r.response.Header.VisitAllCookie(ff)
	return cookies
}

func ReadFastHttpBody(resp *fasthttp.Response) (body []byte, err error) {
	ce := string(resp.Header.Peek(fasthttp.HeaderContentEncoding))
	if ce == Gzip {
		body, err = resp.BodyGunzip()
	} else if ce == Br {
		body, err = resp.BodyUnbrotli()
	} else if ce == Inflate {
		body, err = resp.BodyInflate()
	} else {
		body = resp.Body()
	}
	return
}
