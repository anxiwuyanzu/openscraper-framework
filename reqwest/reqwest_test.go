package reqwest

import (
	"github.com/stretchr/testify/require"
	"github.com/valyala/fasthttp"
	"strings"
	"testing"
)

func TestNextValue(t *testing.T) {
	assert := require.New(t)

	assert.Equal(nextValue([]byte("query=111")), 9)
	assert.Equal(nextValue([]byte("query=111&t=1")), 9)
}

func TestSetQueryArg(t *testing.T) {
	//testSetQueryArg(t, func() Request {
	//	return NewFastHttpRequest()
	//})
	//
	//testSetQueryArg(t, func() Request {
	//	return NewStandardRequest()
	//})

	testSetRequestURI(t, func() Request {
		return NewFastHttpRequest()
	})

	testSetRequestURI(t, func() Request {
		return NewStandardRequest()
	})
}

func testSetQueryArg(t *testing.T, newFn func() Request) {
	assert := require.New(t)

	req := newFn()

	assert.Equal("", req.GetQueryString())
	assert.Equal("", req.GetQueryArg("foo"))

	req.SetQueryArg("foo", "bar")

	assert.Equal("foo=bar", req.GetQueryString())
	assert.Equal("bar", req.GetQueryArg("foo"))

	req.SetQueryArg("foo", "bar")
	assert.Equal("foo=bar", req.GetQueryString())
	assert.Equal("bar", req.GetQueryArg("foo"))

	req.SetQueryArg("foo1", "bar")
	assert.Equal("foo=bar&foo1=bar", req.GetQueryString())
	assert.Equal("bar", req.GetQueryArg("foo"))
	assert.Equal("bar", req.GetQueryArg("foo1"))

	req.SetQueryArg("foo1", "bar")
	assert.Equal("foo=bar&foo1=bar", req.GetQueryString())

	req.SetQueryArg("foo", "bar")
	assert.Equal("foo=bar&foo1=bar", req.GetQueryString())

	req.SetQueryArg("foo2", "bar2")
	assert.Equal("foo=bar&foo1=bar&foo2=bar2", req.GetQueryString())

	req.SetQueryArg("foo", "bar1")
	assert.Equal("foo=bar1&foo1=bar&foo2=bar2", req.GetQueryString())

	req.SetQueryArg("foo1", "bar1")
	assert.Equal("foo=bar1&foo1=bar1&foo2=bar2", req.GetQueryString())

	req.SetQueryArg("chan", "App%20Store")
	assert.Equal("foo=bar1&foo1=bar1&foo2=bar2&chan=App%20Store", req.GetQueryString())
	assert.Equal("App%20Store", req.GetQueryArg("chan"))
	assert.Equal("bar2", req.GetQueryArg("foo2"))
	assert.Equal("bar1", req.GetQueryArg("foo1"))
	assert.Equal("", req.GetQueryArg("foo3"))

	req.SetQueryArg("em", "")
	assert.Equal("foo=bar1&foo1=bar1&foo2=bar2&chan=App%20Store&em=", req.GetQueryString())
	assert.Equal("", req.GetQueryArg("em"))
	assert.Equal("foo=bar1&foo1=bar1&foo2=bar2&chan=App%20Store&em=", joinVisitAll(req))

	req.SetQueryArg("foo1", "")
	assert.Equal("foo=bar1&foo1=&foo2=bar2&chan=App%20Store&em=", req.GetQueryString())
	assert.Equal("", req.GetQueryArg("foo1"))
	assert.Equal("", req.GetQueryArg("em"))
	assert.Equal("foo=bar1&foo1=&foo2=bar2&chan=App%20Store&em=", joinVisitAll(req))

	req = newFn()
	req.SetRequestURI("http://www.test.com?q=1")
	req.SetQueryArg("foo", "bar")

	assert.Equal("q=1&foo=bar", req.GetQueryString())

	req = newFn()
	req.SetRequestURI("http://www.test.com?q=1&")
	req.SetQueryArg("foo", "bar")

	assert.Equal("q=1&foo=bar", req.GetQueryString())
	assert.Equal("q=1&foo=bar", joinVisitAll(req))

	req.SetQueryArg("foo1", "bar&")
	assert.Equal("q=1&foo=bar&foo1=bar&", req.GetQueryString())
	assert.Equal("q=1&foo=bar&foo1=bar", joinVisitAll(req))
	assert.Equal("bar", req.GetQueryArg("foo1"))

	req = newFn()
	req.SetRequestURI("http://www.test.com?source_params=abc")
	req.SetQueryArg("source", "user_view")
	assert.Equal("source_params=abc&source=user_view", req.GetQueryString())
	assert.Equal("source_params=abc&source=user_view", joinVisitAll(req))
	assert.Equal("user_view", req.GetQueryArg("source"))
	assert.Equal("abc", req.GetQueryArg("source_params"))
}

func testSetRequestURI(t *testing.T, newFn func() Request) {
	assert := require.New(t)

	req := newFn()
	req.SetQueryArg("foo", "bar")
	req.SetRequestURI("http://www.test.com?source_params=abc")
	assert.Equal("www.test.com", req.GetHost())
	assert.Equal("source_params=abc", req.GetQueryString())

	req.SetQueryArg("foo", "bar")
	assert.Equal("www.test.com", req.GetHost())
	assert.Equal("source_params=abc&foo=bar", req.GetQueryString())
	assert.Equal("/?source_params=abc&foo=bar", req.GetRequestURI())
	assert.Equal("http://www.test.com/?source_params=abc&foo=bar", req.GetFullURI())
}

func TestRequestURI(t *testing.T) {
	assert := require.New(t)

	req := &fasthttp.Request{}
	req.SetRequestURI("http://example.com/api/v1/")
	assert.Equal("http://example.com/api/v1/", string(req.URI().FullURI())) // ok
	assert.Equal("/api/v1/", string(req.URI().RequestURI()))                // ok
	assert.Equal("/api/v1/", string(req.RequestURI()))                      // ok
	assert.Equal("http://example.com/api/v1/", string(req.URI().FullURI())) // not ok, gets "http:///api/v1/"
	assert.Equal("example.com", string(req.Host()))                         // not ok, gets ""
}

func joinVisitAll(req Request) string {
	var qs string
	req.VisitAllQueryArg(func(key, value []byte) {
		qs += string(key) + "=" + string(value) + "&"
	})
	qs = strings.Trim(qs, "&")
	return qs
}

func TestSetKeyValue(t *testing.T) {
	// 初始化测试数据
	var qs []byte
	key := "test"
	value := "123"

	// 测试添加新参数
	expected := "test=123"
	actual := string(setKeyValue(qs, key, value))
	if actual != expected {
		t.Errorf("Expected %v, but got %v", expected, actual)
	}

	// 测试替换旧参数
	qs = []byte("test=abc")
	expected = "test=123"
	actual = string(setKeyValue(qs, key, value))
	if actual != expected {
		t.Errorf("Expected %v, but got %v", expected, actual)
	}

	// 测试替换多个同名参数
	qs = []byte("test=abc&test=def")
	expected = "test=123&test=def"
	actual = string(setKeyValue(qs, key, value))
	if actual != expected {
		t.Errorf("Expected %v, but got %v", expected, actual)
	}

	// 测试替换参数值为零长度字符串
	qs = []byte("test=abc")
	value = ""
	expected = "test="
	actual = string(setKeyValue(qs, key, value))
	if actual != expected {
		t.Errorf("Expected %v, but got %v", expected, actual)
	}

	// 测试一样的前缀
	qs = []byte("source_params=abc")
	value = ""
	expected = "source_params=abc&source=user_view"
	actual = string(setKeyValue(qs, "source", "user_view"))
	if actual != expected {
		t.Errorf("Expected %v, but got %v", expected, actual)
	}
}
