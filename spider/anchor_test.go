package spider

import "github.com/anxiwuyanzu/openscraper-framework/spider-common-go/v4/dot"

var (
	SpiderTest Anchor = "test"
)

func ExampleAnchor_Register() {
	SpiderTest.Register(func() *Factory {
		return &Factory{
			SourceFactory: nil,
			SpiderFactory: nil,
			WorkerNum:     0,
			BackLog:       0,
			MaxRetryTimes: 0,
			Delay:         0,
			RandomDelay:   0,
			SubSpiders:    nil,
			runForever:    false,
			parent:        nil,
			spiderCnt:     0,
		}
	})

	SpiderTest.AddItem(dot.Item{"id": "test-item"})
}
