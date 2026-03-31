package spider

// Spider 定义爬虫
type Spider interface {
	// Use 使用中间件
	Use(...Handler)
	UsePreCheck(Handler)
	// SetMiddleware 设置中间件
	SetMiddleware(handlers Handlers)
	// Middleware 返回中间件
	Middleware() Handlers
	PreCheck(ctx Context) bool
	// PreStart 预启动; 通常做些公共处理
	PreStart(ctx Context)
	// Start 启动爬虫; 需要配置 request, 否则任务会被取消
	Start(ctx Context)
	// Parse 默认的处理函数
	Parse(ctx Context)
	// OnRetry 重试的时候调用
	OnRetry(ctx Context)
	// OnFailed 在爬虫任务失败时调用
	OnFailed(ctx Context)
	// OnFinished 在爬虫任务结束时调用
	OnFinished(ctx Context)
	// StopSpider 主动结束 spider, 结束后本爬虫的worker退出, 不再接收新的任务
	StopSpider()
	IsStop() bool

	// ItemCh 返回一个 ItemCh, 这个ItemCh只供当前worker消费; 用于向特定worker放量
	ItemCh() ItemCh
}

type Handler func(ctx Context)

type Handlers []Handler

type Application struct {
	middleware  Handlers
	preCheckers Handlers
	stop        bool
	index       int
}

func (app *Application) Use(handlers ...Handler) {
	app.middleware = append(app.middleware, handlers...)
}

func (app *Application) UsePreCheck(handler Handler) {
	app.preCheckers = append(app.preCheckers, handler)
}

func (app *Application) SetMiddleware(handlers Handlers) {
	app.middleware = handlers
}

func (app *Application) Middleware() Handlers {
	return app.middleware
}

func (app *Application) PreCheck(ctx Context) bool {
	for _, checker := range app.preCheckers {
		checker(ctx)
		if ctx.IsStopped() {
			return false
		}
	}
	return true
}

func (app *Application) PreStart(ctx Context) {

}

func (app *Application) Parse(ctx Context) {

}

func (app *Application) OnFailed(ctx Context) {

}

func (app *Application) OnFinished(ctx Context) {

}

func (app *Application) OnRetry(ctx Context) {

}

func (app *Application) StopSpider() {
	app.stop = true
}

func (app *Application) IsStop() bool {
	return app.stop
}

func (app *Application) ItemCh() ItemCh {
	return nil
}
