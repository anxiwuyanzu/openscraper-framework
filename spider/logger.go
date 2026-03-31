package spider

import (
	"time"
)

func NewLoggerWithUri() Handler {
	return NewLogger(LoggerWithUri())
}

func NewLogger(opts ...LogOpOption) Handler {
	option := &LogOption{}
	for _, opFunc := range opts {
		opFunc(option)
	}

	return func(ctx Context) {
		ctx.AddLogField("id", ctx.Params().Id())

		start := time.Now()
		req := ctx.Request()
		tryTimes := ctx.TryTimes()

		ctx.Next()

		if option.withUri && req != nil {
			ctx.AddLogField("uri", req.GetFullURI())
		}

		ctx.AddLogField("took", time.Since(start).Milliseconds())
		if tryTimes > 0 {
			ctx.AddLogField("try", tryTimes)
		}

		if ctx.StatusCode() == StatusCodeSkip {
			ctx.Logger().WithFields(ctx.LogFields()).Info("skip result")
			return
		}

		if ctx.StatusCode() == StatusCodeFailed {
			ctx.AddLogField("state", 0)
			ctx.Logger().WithFields(ctx.LogFields()).Error("result")
		} else {
			ctx.AddLogField("state", 1)
			ctx.Logger().WithFields(ctx.LogFields()).Info("result")
		}
	}
}

type LogOpOption func(*LogOption)

type LogOption struct {
	withUri bool
}

func LoggerWithUri() LogOpOption {
	return func(op *LogOption) { op.withUri = true }
}
