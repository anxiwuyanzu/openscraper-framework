package spider

import (
	"github.com/anxiwuyanzu/openscraper-framework/v4/reqwest"
	"github.com/anxiwuyanzu/openscraper-framework/v4/spider/memstore"
	"github.com/sirupsen/logrus"
)

func AcquireContext(spider Spider, name Anchor, item Item, maxRetryTimes, workerIndex int) *contextImpl {
	var params Item
	var ctxItem *ContextItem
	if ci, ok := item.(*ContextItem); ok {
		params = ci.Item
		ctxItem = ci
	} else {
		params = item
	}
	item.Elapsed() // Elapsed 里可以记录一些信息
	return &contextImpl{
		spider:            spider,
		name:              name,
		params:            params,
		ctxItem:           ctxItem,
		onResponseHandler: spider.Parse,
		statusCode:        StatusCodeUninit,
		maxRetryTimes:     maxRetryTimes,
		handlers:          spider.Middleware(),
		workerIndex:       workerIndex,
		logFields:         logrus.Fields{},
	}
}

// Context 维护爬虫运行一个任务时的状态
type Context interface {
	// Name 返回爬虫名字
	Name() Anchor
	Next()
	NextHandler() Handler
	OnResponse(Handler)
	// ParseResponse 调用最后的处理逻辑, 调用 OnResponse 设置的 Handler
	ParseResponse()
	// StopExecution 结束执行
	StopExecution()
	// IsStopped 判断是否结束执行
	IsStopped() bool
	// Params 返回输入参数
	Params() Item
	// Values 用作中间件之间数据传输
	Values() *memstore.Store

	NewRequest() reqwest.Request
	// Request 返回 Request
	Request() reqwest.Request

	// TryTimes 当前 request 重试次数
	TryTimes() int
	// Ok 设置爬虫成功
	Ok()
	// Fail 设置爬虫失败
	Fail(errs ...error)
	// Skip 设置跳过爬虫
	Skip(errs ...error)
	// StatusCode 返回状态码
	StatusCode() StatusCode
	IsMaxRetry() bool
	Logger() *logrus.Entry
	SetLogger(logger *logrus.Entry)
	// AddLogField 往爬虫日志添加字段
	AddLogField(string, interface{})
	// LogFields 返回爬虫日志字段
	LogFields() logrus.Fields

	Err() error
	// WithCtxValue 向 ContextItem 添加 value, 以便在爬虫外部获取结果
	WithCtxValue(key string, value any)
	// WithChildCtxValue 向 Packed-ContextItem 添加 value, 以便在爬虫外部获取结果;
	// Packed-ContextItem 指将多个爬虫任务打包成一个,比如批量请求接口
	// 注意 id 为 item.Id()
	WithChildCtxValue(key, id string, value any)
	// CtxValue 获取 ContextItem 中的 value
	CtxValue(key string) any
	CtxErr() error
	CtxDone() <-chan struct{}
	// WorkerIndex 返回当前worker index;
	WorkerIndex() int
	HasNewRequest() bool
}

type contextImpl struct {
	spider  Spider
	name    Anchor
	params  Item           // input.
	values  memstore.Store // generic storage, middleware communication.
	ctxItem *ContextItem

	// the route's handlers
	handlers Handlers
	// the current position of the handler's chain
	currentHandlerIndex int
	onResponseHandler   Handler // 请求完成后请求 onResponse, 默认为 spider.Parse
	tryTimes            int     // 当前request 重试次数, 初始值 0
	maxRetryTimes       int
	logger              *logrus.Entry
	request             reqwest.Request
	statusCode          StatusCode
	logFields           logrus.Fields
	err                 error
	hasNewRequest       bool
	workerIndex         int
}

func (ctx *contextImpl) Name() Anchor {
	return ctx.name
}

// Do calls the SetHandlers(handlers)
// and executes the first handler,
// handlers should not be empty.
//
// It's used by the router, developers may use that
// to replace and execute handlers immediately.
func (ctx *contextImpl) do() {
	ctx.init()
	if len(ctx.handlers) > 0 {
		ctx.handlerIndex(0)
		ctx.handlers[0](ctx)
	}
}

func (ctx *contextImpl) doHandlers(handlers ...Handler) {
	ctx.handlers = handlers
	if len(ctx.handlers) > 0 {
		ctx.handlerIndex(0)
		ctx.handlers[0](ctx)
	}
}

// AddHandler can add handler(s)
// to the current request in serve-time,
// these handlers are not persistenced to the router.
//
// Router is calling this function to add the route's handler.
// If AddHandler called then the handlers will be inserted
// to the end of the already-defined route's handler.
func (ctx *contextImpl) addHandler(handlers ...Handler) {
	ctx.handlers = append(ctx.handlers, handlers...)
}

// SetHandlers replaces all handlers with the new.
func (ctx *contextImpl) setHandlers(handlers Handlers) {
	ctx.handlers = handlers
}

// HandlerIndex sets the current index of the
// current contextImpl's handlers chain.
// If -1 passed then it just returns the
// current handler index without change the current index.rns that index, useless return value.
//
// Look Handlers(), Next() and StopExecution() too.
func (ctx *contextImpl) handlerIndex(n int) (currentIndex int) {
	if n < 0 || n > len(ctx.handlers)-1 {
		return ctx.currentHandlerIndex
	}

	ctx.currentHandlerIndex = n
	return n
}

// Next calls all the next handler from the handlers chain,
// it should be used inside a middleware.
//
// Note: Custom contextImpl should override this method in order to be able to pass its own contextImpl.GContext implementation.
func (ctx *contextImpl) Next() {
	if ctx.IsStopped() {
		return
	}
	if n := ctx.handlerIndex(-1) + 1; n < len(ctx.handlers) {
		ctx.handlerIndex(n)
		ctx.handlers[n](ctx)
	}
}

// NextHandler returns (it doesn't execute) the next handler from the handlers chain.
//
// Use .Skip() to skip this handler if needed to execute the next of this returning handler.
func (ctx *contextImpl) NextHandler() Handler {
	if ctx.IsStopped() {
		return nil
	}
	nextIndex := ctx.currentHandlerIndex + 1
	// check if it has a next middleware
	if nextIndex < len(ctx.handlers) {
		return ctx.handlers[nextIndex]
	}
	return nil
}

const stopExecutionIndex = -1 // I don't set to a max value because we want to be able to reuse the handlers even if stopped with .Skip

// StopExecution if called then the following .Next calls are ignored,
// as a result the next handlers in the chain will not be fire.
func (ctx *contextImpl) StopExecution() {
	ctx.currentHandlerIndex = stopExecutionIndex
}

// IsStopped checks and returns true if the current position of the contextImpl is -1,
// means that the StopExecution() was called.
func (ctx *contextImpl) IsStopped() bool {
	return ctx.currentHandlerIndex == stopExecutionIndex
}

//  +------------------------------------------------------------+
//  | Current "user/request" storage                             |
//  | and share information between the handlers - Values().     |
//  | Save and get named path parameters - Params()              |
//  +------------------------------------------------------------+

// Params returns the current url's named parameters key-value storage.
// Named path parameters are being saved here.
// This storage, as the whole contextImpl, is per-request lifetime.
func (ctx *contextImpl) Params() Item {
	return ctx.params
}

// Values returns the current "user" storage.
// Named path parameters and any optional data can be saved here.
// This storage, as the whole contextImpl, is per-request lifetime.
//
// You can use this function to Set and Get local values
// that can be used to share information between handlers and middleware.
func (ctx *contextImpl) Values() *memstore.Store {
	return &ctx.values
}

// OnResponse 设置请求完成后请求 onResponse, 默认为 spider.Parse
func (ctx *contextImpl) OnResponse(h Handler) {
	ctx.onResponseHandler = h
}

func (ctx *contextImpl) ParseResponse() {
	ctx.onResponseHandler(ctx)
}

func (ctx *contextImpl) NewRequest() reqwest.Request {
	if ctx.request != nil {
		ctx.request.Close()
	}

	req := reqwest.NewRequest()
	ctx.request = req
	ctx.hasNewRequest = true
	return req
}

func (ctx *contextImpl) Request() reqwest.Request {
	return ctx.request
}

func (ctx *contextImpl) onRetry(h Handler) {
	ctx.tryTimes++
	ctx.statusCode = StatusCodeOnRetry // 为了不重置 tryTimes
	h(ctx)
	ctx.statusCode = 0
}

// init 在每次请求,或者重试请求的时候执行初始化
func (ctx *contextImpl) init() {
	ctx.hasNewRequest = false
	ctx.statusCode = StatusCodeFailed
	ctx.err = nil

	if _, ok := ctx.logFields["err"]; ok {
		delete(ctx.logFields, "err")
	}
	//ctx.logFields = logrus.Fields{}
}

func (ctx *contextImpl) HasNewRequest() bool {
	return ctx.hasNewRequest
}

func (ctx *contextImpl) TryTimes() int {
	return ctx.tryTimes
}

func (ctx *contextImpl) IsMaxRetry() bool {
	return ctx.tryTimes == ctx.maxRetryTimes
}

func (ctx *contextImpl) Ok() {
	ctx.statusCode = StatusCodeOk
}

func (ctx *contextImpl) Fail(errs ...error) {
	ctx.statusCode = StatusCodeFailed
	if len(errs) > 0 {
		ctx.err = errs[0]
		ctx.AddLogField("err", errs[0].Error())
	}
}

func (ctx *contextImpl) Skip(errs ...error) {
	ctx.statusCode = StatusCodeSkip
	if len(errs) > 0 && ctx.request != nil {
		ctx.err = errs[0]
		ctx.AddLogField("err", errs[0].Error())
	}
}

func (ctx *contextImpl) Err() error {
	return ctx.err
}

func (ctx *contextImpl) StatusCode() StatusCode {
	return ctx.statusCode
}

func (ctx *contextImpl) Logger() *logrus.Entry {
	return ctx.logger
}

func (ctx *contextImpl) SetLogger(logger *logrus.Entry) {
	ctx.logger = logger
}

func (ctx *contextImpl) close() {
	if ctx.request != nil {
		ctx.request.Close()
	}

	if ctx.ctxItem != nil {
		ctx.ctxItem.cancel()
	}
}

func (ctx *contextImpl) LogFields() logrus.Fields {
	return ctx.logFields
}

func (ctx *contextImpl) AddLogField(key string, value interface{}) {
	if v, ok := value.(string); ok && v == DeleteField {
		delete(ctx.logFields, key)
		return
	}
	ctx.logFields[key] = value
}

// WithCtxValue set value to ctx; only works when ctx.ctxItem is not nil
func (ctx *contextImpl) WithCtxValue(key string, value any) {
	if ctx.ctxItem != nil {
		ctx.ctxItem.withValue(key, value)
	}
}

// WithChildCtxValue set packed value to ctx; only works when ctx.ctxItem is not nil
func (ctx *contextImpl) WithChildCtxValue(key, id string, value any) {
	if ctx.ctxItem != nil {
		ctx.ctxItem.withChildCtxValue(key, id, value)
	}
}

// CtxValue get value from ctx; only works when ctx.ctxItem is not nil
func (ctx *contextImpl) CtxValue(key string) any {
	if ctx.ctxItem != nil {
		return ctx.ctxItem.Context.Value(key)
	}
	return nil
}

// CtxErr get err from ctx; only works when ctx.ctxItem is not nil
// 可以用来检测 ctxItem 是否 Done
func (ctx *contextImpl) CtxErr() error {
	if ctx.ctxItem != nil {
		return ctx.ctxItem.Context.Err()
	}
	return nil
}

// CtxDone return ctx.Done
func (ctx *contextImpl) CtxDone() <-chan struct{} {
	if ctx.ctxItem != nil {
		return ctx.ctxItem.Context.Done()
	}
	return nil
}

func (ctx *contextImpl) WorkerIndex() int {
	return ctx.workerIndex
}
