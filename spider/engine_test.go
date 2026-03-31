package spider

import (
	"fmt"
	"github.com/anxiwuyanzu/openscraper-framework/spider-common-go/v4/dot"
	"github.com/anxiwuyanzu/openscraper-framework/spider-common-go/v4/internal/tests"
	"github.com/anxiwuyanzu/openscraper-framework/spider-common-go/v4/reqwest"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
	"github.com/valyala/fasthttp"
	"net/http"
	"testing"
)

// 测试用中间件。
type testMiddleware struct {
	count  int
	client reqwest.IClient
}

func (m *testMiddleware) doRequest(ctx Context) {
	req := ctx.Request()
	err := m.client.DoRequest(req)
	if err != nil {
		ctx.Fail(err)
		return
	}
	m.count += 1
	ctx.ParseResponse()
	ctx.Next()
	dot.WithValue("count", m.count)
	ctx.Logger().Info("is max retry ", ctx.IsMaxRetry())
}

// 测试用爬虫。
type testSpider struct {
	Application
}

func (s *testSpider) Start(ctx Context) {
	id := ctx.Params().Id()
	req := ctx.NewRequest()
	req.SetMethod(fasthttp.MethodGet)
	req.SetRequestURI("http://localhost:7501/" + id)
	ctx.OnResponse(s.parse)
}

func (s *testSpider) parse(ctx Context) {
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
	dot.WithValue("test", string(body))
	ctx.Ok()
}

const (
	TestSpider Anchor = "test/test"
)

// TestEngineSimple测试框架正常爬取一次。
func TestEngineSimple(t *testing.T) {
	dot.InitFromViper()
	dot.Conf().Reqwest.Client = "fasthttp"

	server := &tests.Server{Mux: map[string]http.HandlerFunc{
		"/hello": func(w http.ResponseWriter, req *http.Request) {
			fmt.Fprintf(w, "hello")
		},
	}, Port: 7501}
	server.Start()
	dot.EnsureClose(server.Close)

	TestSpider.Register(func() *Factory {
		m := testMiddleware{
			client: reqwest.NewClient(nil),
		}
		return &Factory{
			SourceFactory: func(itemCh ItemCh, workerNum, mode int) {
				itemCh <- dot.Item{"id": "/hello"}
				itemCh <- dot.Item{"id": "/"}
			},
			SpiderFactory: func(logger *log.Entry) Spider {
				s := &testSpider{}
				s.Use(m.doRequest)
				return s
			},
			WorkerNum:     1,
			MaxRetryTimes: 1,
		}
	})
	engine := NewEngine()
	engine.Start("test/test")

	assert := require.New(t)
	assert.Equal("hello", dot.Value("test"))
	assert.Equal(3, dot.Value("count"))
}
