### Dns Cache
golang 默认不会缓存dns, 每个 net.Dial 都会去请求dns解析; 这个库加上这个功能

代码来源: https://github.com/mercari/go-dnscache

做了些日志和默认参数的改动

简单例子:

```
client := http.Client{
    Transport: &http.Transport{
        DialContext: DialFuncWithDnsCache(nil, 0),
    },
}

_, err := client.Get("http://www.cip.cc")
if err != nil {
    panic(err)
}
```

BTW, `fasthttp` supports dns cache by default.