package spider

import (
	"fmt"
	"github.com/anxiwuyanzu/openscraper-framework/spider-common-go/v4/internal/tests"
	"github.com/anxiwuyanzu/openscraper-framework/spider-common-go/v4/reqwest"
	"net/http"
	"testing"
	"time"

	"github.com/anxiwuyanzu/openscraper-framework/spider-common-go/v4/dot"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
	"github.com/valyala/fasthttp"
)

const (
	TestMainSpider Anchor = "test/main"
	TestSubSpider  Anchor = "test/sub"
)

// 主爬虫。
type testMainSpider struct {
	Application
	ch chan struct{} // 用于被子爬虫回调
}

func (s *testMainSpider) Start(ctx Context) {
	req := ctx.NewRequest()
	req.SetMethod(fasthttp.MethodGet)
	req.SetRequestURI("http://localhost:7502/main")
	ctx.OnResponse(s.parse)
}

func (s *testMainSpider) parse(ctx Context) {
	req := ctx.Request()
	if req.ResponseStatusCode() != fasthttp.StatusOK {
		log.Errorf("status code is not ok: %d.", req.ResponseStatusCode())
		ctx.Fail()
		return
	}
	body, err := req.ResponseBody()
	if err != nil {
		log.WithError(err).Panic("failed to read body.")
		ctx.Fail()
		return
	}

	// 放入子爬虫用item队列
	itemCtx, _ := TestSubSpider.AddTimeoutItem(dot.Item{"id": ""}, 3*time.Second)
	// 等待item被子爬虫消费
	<-itemCtx.Done()

	dot.WithValue("test_main", string(body))
	dot.WithValue("test_sub", itemCtx.Value("test_sub"))
	ctx.Ok()
}

// 子瓢虫。
type testSubSpider struct {
	Application
}

func (s *testSubSpider) Start(ctx Context) {
	req := ctx.NewRequest()
	req.SetMethod(fasthttp.MethodGet)
	req.SetRequestURI("http://localhost:7502/sub")
	ctx.OnResponse(s.parse)
}

func (s *testSubSpider) parse(ctx Context) {
	req := ctx.Request()
	if req.ResponseStatusCode() != fasthttp.StatusOK {
		log.Errorf("status code is not ok: %d.", req.ResponseStatusCode())
		ctx.Fail()
		return
	}
	body, err := req.ResponseBody()
	if err != nil {
		log.WithError(err).Panic("failed to read body.")
		ctx.Fail()
		return
	}
	ctx.WithCtxValue("test_sub", string(body))
	ctx.Ok()
}

// TestEngineWithSubspiders测试框架使用子爬虫。
func TestEngineWithSubspiders(t *testing.T) {
	server := &tests.Server{Mux: map[string]http.HandlerFunc{
		"/main": func(w http.ResponseWriter, req *http.Request) {
			fmt.Fprintf(w, "main")
		},
		"/sub": func(w http.ResponseWriter, req *http.Request) {
			fmt.Fprintf(w, "sub")
		},
	}, Port: 7502}
	server.Start()
	dot.EnsureClose(server.Close)

	TestSubSpider.Register(func() *Factory {
		m := testMiddleware{
			client: reqwest.NewClient(nil),
		}
		return &Factory{
			SpiderFactory: func(logger *log.Entry) Spider {
				s := &testSubSpider{}
				s.Use(m.doRequest)
				return s
			},
			WorkerNum:     1,
			MaxRetryTimes: 1,
		}
	})

	TestMainSpider.Register(func() *Factory {
		m := testMiddleware{
			client: reqwest.NewClient(nil),
		}
		return &Factory{
			SourceFactory: func(itemCh ItemCh, workerNum, mode int) {
				itemCh <- dot.Item{"id": ""}
				itemCh <- dot.Item{"id": ""}
			},
			SpiderFactory: func(logger *log.Entry) Spider {
				s := &testMainSpider{ch: make(chan struct{})}
				s.Use(m.doRequest)
				return s
			},
			WorkerNum:     1,
			MaxRetryTimes: 1,
			SubSpiders:    []Anchor{TestSubSpider},
		}
	})
	engine := NewEngine()
	engine.Start("test/main")

	assert := require.New(t)
	assert.Equal("main", dot.Value("test_main"))
	assert.Equal("sub", dot.Value("test_sub"))
}
