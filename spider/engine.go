package spider

import (
	"context"
	"github.com/anxiwuyanzu/openscraper-framework/spider-common-go/v4/dot"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"os"
	"os/signal"
	"sync"
	"syscall"
)

// Engine 负责启动爬虫
type Engine struct {
	runForever bool
	stopped    chan struct{}
}

func NewEngine() *Engine {
	engine := &Engine{
		stopped: make(chan struct{}),
	}
	engine.setup()
	return engine
}

// setup config engine
func (e *Engine) setup() {
	dot.InitFromViper()

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		for {
			select {
			case <-sigs:
				dot.Cancel()
			}
		}
	}()
}

// Close close engine
func (e *Engine) Close() {
	// wait for mongo & kafka close
	closeCallbacks := dot.EnsureCloseHandlers()
	for _, cb := range closeCallbacks {
		cb()
	}

	dot.Logger().Info("engine exited")
	close(e.stopped)
}

// WaitExit 等待 Engine 退出
func (e *Engine) WaitExit() {
	<-e.stopped
}

// StartForever 长久运行一个爬虫; 即 Factory.SourceFactory 已经退出, 也会持续运行
func (e *Engine) StartForever(spiderName string) {
	e.runForever = true
	e.Start(spiderName)
}

// Start 启动一个爬虫; 当 Factory.SourceFactory 退出, 并且任务执行完成, Engine 会退出
func (e *Engine) Start(spider string) {
	spiderName := Anchor(spider)
	builder, ok := factories[spiderName]

	if !ok {
		dot.Logger().WithField("spider", spiderName).Panic("spider not found")
		return
	}

	logSpiderName := spider
	if len(dot.Conf().Spider.LogTail) > 0 {
		logSpiderName += "-" + dot.Conf().Spider.LogTail
	}

	logger := logrus.WithFields(logrus.Fields{"spider": logSpiderName})
	dot.WithLogger(logger)

	factory := builder()
	factory.setupFromConfig()

	factory.runForever = e.runForever
	factory.Start(context.Background(), spiderName, logSpiderName, true)

	e.Close()
}

// GroupConfig 爬虫组配置
type GroupConfig struct {
	Spider        Anchor `mapstructure:"spider" json:"spider"`
	WorkerNum     int    `mapstructure:"worker_num" json:"worker_num"`
	MaxRetryTimes int    `mapstructure:"max_retry_times" json:"max_retry_times"`
}

// StartSpiderGroupFromConfig 通过配置启动爬虫组
func (e *Engine) StartSpiderGroupFromConfig(groupName string) {
	var configs []GroupConfig
	err := viper.UnmarshalKey("spider_groups."+groupName, &configs)
	if err != nil {
		dot.Logger().WithError(err).Error("failed to parse group")
		return
	}

	e.StartSpiderGroup(configs)
}

// StartSpiderGroup 启动爬虫组
func (e *Engine) StartSpiderGroup(configs []GroupConfig) {
	if len(configs) == 0 {
		dot.Logger().Error("group is empty")
		return
	}

	wg := sync.WaitGroup{}
	for _, config := range configs {
		wg.Add(1)
		e.RunSpider(config.Spider, func(factory *Factory) {
			if config.WorkerNum > 0 {
				factory.WorkerNum = config.WorkerNum
			}
			if config.MaxRetryTimes > 0 {
				factory.MaxRetryTimes = config.MaxRetryTimes
			}
		}, &wg)
	}

	wg.Wait()

	e.Close()
}

// RunSpider 启动指定爬虫在后台运行, configFn 用来重新配置 factory
// RunSpider 用在启动多个爬虫;
func (e *Engine) RunSpider(spiderName Anchor, configFn func(*Factory), wg *sync.WaitGroup) {
	builder, ok := factories[spiderName]
	if !ok {
		dot.Logger().WithField("spider", spiderName).Panic("spider not found")
		return
	}
	factory := builder()
	factory.setupFromConfig()
	configFn(factory)

	factory.runForever = true
	go func() {
		defer wg.Done()

		logSpiderName := string(spiderName)
		if len(dot.Conf().Spider.LogTail) > 0 {
			logSpiderName += "-" + dot.Conf().Spider.LogTail
		}

		factory.Start(context.Background(), spiderName, logSpiderName, false)
	}()
}
