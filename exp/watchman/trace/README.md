# OTel Go SDK Wrapper - Tracing

## 介绍
**Tracing(链路或事务追踪)主要用于记录某件事的开始到结束的整个过程(事件在OTel中用`Span`表示), 在其中你可以创建子事件(嵌套`Span`)**
**或者记录当前事件的状态(成功或出错)**

## 使用说明([样例文件](../../../examples/monitor/tracing_example.go))
1. 开启Tracing功能前创建`common.Config`, 并配置好(`Tracing`不需要配置`MetricSendInterval`)
2. 调用`trace.Setup()`启动Tracing功能
3. 在某个需要记录的事件开始前, 通过`trace.NewSpan`创建Span(如果是根Span(父Span), `rawCtx`可以设为nil)
   1. 返回的`context.Context`有两个用途: "通过ctx查找Span" 和 "创建子Span时将根Span(父Span)的ctx传入`trace.NewSpan`实现Span嵌套" 
   2. 如自行储存`Span`(如存在`spider.Context.Values`里)且不需要创建子Span, 可以忽略返回的`context.Context`
4. 调用`span.AddEvent()`记录某个关键点(如 获取到了代理/加密参数完成)
5. 调用`span.SetAttributes()`对该事件添加描述(如 获取到了X个代理/加密参数的原文字符串)
6. (可选)调用`span.SetError()`记录错误
7. 在最后, 调用`span.End()`结束该Span(任何Span都要结束, 否则不会发送数据出去, 可用defer)
8. 程序退出前, 调用`trace.Shutdown()`停止Tracing功能

## OTel Resource 说明[resource.go](../common/resource.go)
**Resource是OTel中对数据的概述的变量, 类似metadata的概念**
- `trace`包只提供了基础的 name/namespace设置, 如需要补充, 请通过`OTEL_RESOURCE_ATTRIBUTES`环境变量设置, 详情见`resource.go`