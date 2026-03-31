package spider

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	"github.com/anxiwuyanzu/openscraper-framework/v4/dot"
	"github.com/anxiwuyanzu/openscraper-framework/v4/util"
	"github.com/sirupsen/logrus"
)

// Factory 负责构造并配置爬虫
type Factory struct {
	// SourceFactory 定义爬虫任务的放量
	SourceFactory func(itemCh ItemCh, workerNum int, mode int)
	// SpiderFactory 定义怎么构建爬虫
	SpiderFactory func(logger *logrus.Entry) Spider
	// WorkerNum 运行爬虫的线程数
	WorkerNum int
	// BackLog 设置 ItemCh 最大长度
	BackLog int
	// MaxRetryTimes 最大重试次数
	MaxRetryTimes int
	// Delay 定义爬虫任务的延迟时间
	Delay time.Duration
	// Delay 定义爬虫任务的随机延迟时间
	RandomDelay time.Duration
	// SubSpiders 定义子爬虫
	SubSpiders []Anchor

	// 是否长期运行, 即 Factory.SourceFactory 已经退出, 也会持续运行
	runForever bool
	// parent 父级爬虫
	parent *Factory
	// spiderCnt 当前爬虫数量
	spiderCnt int32
}

// setupFromConfig 完成配置
func (f *Factory) setupFromConfig() {
	cfg := dot.Conf().Spider

	if cfg.WorkerNum > f.WorkerNum {
		f.WorkerNum = cfg.WorkerNum
	}
	if f.WorkerNum == 0 {
		f.WorkerNum = 1
	}

	if cfg.MaxRetryTimes > f.MaxRetryTimes {
		f.MaxRetryTimes = cfg.MaxRetryTimes
	}

	if cfg.Delay > f.Delay { // 取最大值
		f.Delay = cfg.Delay
	}
	if cfg.RandomDelay > f.RandomDelay {
		f.RandomDelay = cfg.RandomDelay
	}

	if cfg.Backlog > f.BackLog {
		f.BackLog = cfg.Backlog
	}
	if f.BackLog == 0 {
		f.BackLog = 3000
	}

	if dot.Debug() {
		f.MaxRetryTimes = cfg.MaxRetryTimes
		f.WorkerNum = cfg.WorkerNum
		f.Delay = cfg.Delay
	}
}

// Start 启动 Factory, 启动爬虫和子爬虫
func (f *Factory) Start(ctx context.Context, spiderName Anchor, logSpiderName string, main bool) {
	if !markRunning(spiderName, f) {
		dot.Logger().WithField("spider", spiderName).Warn("spider already started")
		return
	}
	dot.Logger().WithFields(logrus.Fields{
		"debug":      dot.Debug(),
		"spider":     logSpiderName,
		"client":     dot.Conf().Reqwest.Client,
		"http_v":     dot.Conf().Reqwest.HttpVersion,
		"proxy":      dot.Conf().Proxy.Proxy,
		"worker_num": f.WorkerNum,
	}).Info("start spider")

	itemCh := createItemCh(spiderName, f.BackLog)

	if main {
		var spiderCancel context.CancelFunc
		ctx, spiderCancel = context.WithCancel(dot.Context())

		go func() {
			var mode int
			if f.runForever {
				mode = ModeRunForever
			}
			f.SourceFactory(itemCh, f.WorkerNum, mode)
			if dot.Debug() {
				time.Sleep(time.Second)
			}
			if !f.runForever {
				spiderCancel() // 通知爬虫结束
			}
		}()
	} else {
		if f.SourceFactory != nil {
			go f.SourceFactory(itemCh, f.WorkerNum, ModeSubSpider) // 1 代表子爬虫
		}
	}

	wg := &sync.WaitGroup{}
	f.StartSubSpiders(ctx, wg)

	i := 0
	for i = 0; i < f.WorkerNum; i++ {
		wg.Add(1)
		go f.StartSpider(ctx, spiderName, logSpiderName, i, itemCh, wg)
	}
	wg.Wait()
}

// StartSubSpiders 启动子爬虫
func (f *Factory) StartSubSpiders(ctx context.Context, wg *sync.WaitGroup) {
	for _, spiderName := range f.SubSpiders {
		builder, ok := factories[spiderName]
		if !ok {
			dot.Logger().WithField("spider", spiderName).Panic("spider not found")
			continue
		}

		subFactory := builder()
		subFactory.setupFromConfig()
		subFactory.parent = f
		// 提前为子爬虫创建 itemCh; 避免 itemCh 不存在
		createItemCh(spiderName, subFactory.BackLog)

		wg.Add(1)
		go func(factory *Factory, spider Anchor) {
			defer wg.Done()

			logSpiderName := string(spider)
			if len(dot.Conf().Spider.LogTail) > 0 {
				logSpiderName += "-" + dot.Conf().Spider.LogTail
			}

			factory.Start(ctx, spider, logSpiderName, false)
		}(subFactory, spiderName)
	}
}

// StartSpider 启动爬虫; 开始消费ItemCh, 并执行相关任务
func (f *Factory) StartSpider(ctx context.Context, spiderName Anchor, logSpiderName string, workerIndex int, itemCh ItemCh, wg *sync.WaitGroup) {
	atomic.AddInt32(&f.spiderCnt, 1)
	defer func() {
		wg.Done()
		atomic.AddInt32(&f.spiderCnt, -1)
	}()

	logger := logrus.WithFields(logrus.Fields{"spider": logSpiderName})
	spider := f.SpiderFactory(logger)
	if spider == nil {
		return
	}

	canStop := false
	workerItemCh := spider.ItemCh()

	for {
		if spider.IsStop() {
			return
		}
		// 所有 itemCh 消费完, 或者 上级爬虫结束
		if len(itemCh) == 0 && canStop && (f.parent == nil || f.parent.spiderCnt == 0) && (workerItemCh == nil || len(workerItemCh) == 0) {
			return
		}

		select {
		case item := <-itemCh:
			f.do(item, spider, logger, spiderName, workerIndex)
		case item := <-workerItemCh: // 假设往worker里添加任务
			f.do(item, spider, logger, spiderName, workerIndex)
		case <-dot.Context().Done():
			// stop immediately
			logger.Info("worker stopped")
			return
		case <-ctx.Done():
			// stop when itemCh is empty
			canStop = true
			time.Sleep(100 * time.Millisecond) // to avoid spamming
		}
	}
}

// do 执行爬虫任务
func (f *Factory) do(item Item, spider Spider, logger *logrus.Entry, spiderName Anchor, workerIndex int) {
	var start bool
	ctx := AcquireContext(spider, spiderName, item, f.MaxRetryTimes, workerIndex)
	ctx.SetLogger(logger)
	defer ctx.close()

	for {
		if spider.IsStop() {
			return
		}

		select {
		case <-dot.Context().Done():
			return
		case <-ctx.CtxDone(): // 任务被 cancel
			return
		default:
		}

		// 执行 spider.Start
		if !start {
			spider.PreCheck(ctx)
			spider.PreStart(ctx)
			if ctx.IsStopped() {
				spider.OnFinished(ctx)
				return
			}

			spider.Start(ctx)                          // Start 中需要创建 request.
			if !ctx.hasNewRequest || ctx.IsStopped() { // 没有请求, 直接返回
				spider.OnFinished(ctx)
				return
			}
			start = true
		}

		if f.MaxRetryTimes > 0 {
			doRetry(ctx, spider, f.MaxRetryTimes, f.Delay, f.RandomDelay)
		} else {
			ctx.do()
		}

		if f.Delay > 0 {
			util.Sleep(dot.Context(), f.Delay)
		} else if f.RandomDelay > 0 {
			util.Sleep(dot.Context(), util.RandTime(f.RandomDelay))
		}

		if ctx.StatusCode() == StatusCodeFailed {
			spider.OnFailed(ctx)
		}

		if !ctx.hasNewRequest || ctx.IsStopped() { // 没有新的请求
			spider.OnFinished(ctx)
			return
		}
		// 爬虫后续流程可以主动调用 ctx.NewRequest(), 这里会继续执行新的Request
		ctx.tryTimes = 0
	}
}

// doRetry 重试执行;
func doRetry(ctx *contextImpl, spider Spider, maxRetryTimes int, delay, randomDelay time.Duration) {
	for {
		ctx.do()

		if ctx.IsStopped() {
			return
		}

		if ctx.StatusCode() == StatusCodeFailed {
			if ctx.TryTimes()+1 > maxRetryTimes {
				return
			}

			// 开始重试
			if delay > 0 {
				util.Sleep(dot.Context(), delay)
			} else if randomDelay > 0 {
				util.Sleep(dot.Context(), util.RandTime(randomDelay))
			}

			// 判断spider是否该退出
			select {
			case <-dot.Context().Done():
				return
			case <-ctx.CtxDone(): // 任务被 cancel
				return
			default:
			}

			spider.OnRetry(ctx)
			spider.PreCheck(ctx)
			if ctx.IsStopped() {
				return
			}
			// 在多次请求的时候, 需要清除上次请求的状态, 避免比如url被重复append
			ctx.onRetry(spider.Start)

			continue
		}

		return
	}
}
