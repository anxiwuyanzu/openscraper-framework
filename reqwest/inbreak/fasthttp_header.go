package inbreak

import (
	_ "github.com/valyala/fasthttp"
	_ "unsafe"
)

//go:linkname strUserAgent github.com/valyala/fasthttp.strUserAgent
var strUserAgent []byte

//go:linkname strHost github.com/valyala/fasthttp.strHost
var strHost []byte

//go:linkname strContentType github.com/valyala/fasthttp.strContentType
var strContentType []byte

//go:linkname strContentLength github.com/valyala/fasthttp.strContentLength
var strContentLength []byte

//go:linkname strCookie github.com/valyala/fasthttp.strCookie
var strCookie []byte

//go:linkname strConnection github.com/valyala/fasthttp.strConnection
var strConnection []byte

// TurnFastHttpHeaderLower 将 fasthttp 关键 header 改成小写
func TurnFastHttpHeaderLower() {
	strUserAgent = []byte("user-agent")
	strHost = []byte("host")
	strContentType = []byte("content-type")
	strContentLength = []byte("content-length")
	strCookie = []byte("cookie")
	strConnection = []byte("connection")
}
