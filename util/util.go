package util

import (
	"bytes"
	crand "crypto/rand"
	"encoding/hex"
	"fmt"
	uuid "github.com/satori/go.uuid"
	"golang.org/x/exp/constraints"
	"math/rand"
	"net/url"
	"os"
	"os/exec"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"time"
	"unsafe"
)

// https://github.com/tidwall/gjson/issues/102
func CopyString(s string) string {
	return string(append([]byte(nil), s...))
}

func CopyBuf(buf []byte) []byte {
	body := make([]byte, 0, len(buf))
	return append(body, buf...)
}

func Itoa[T constraints.Integer](v T) string {
	return fmt.Sprintf("%d", v)
}

func Atoi[T constraints.Integer](s string) T {
	v, _ := strconv.ParseInt(s, 10, 64)
	return T(v)
}

// If 条件选择
func If[T any](condition bool, yesValue T, noValue T) T {
	if condition {
		return yesValue
	}
	return noValue
}

func EncodeQuery(params map[string]string) string {
	if params == nil {
		return ""
	}
	var buf strings.Builder
	keys := make([]string, 0, len(params))
	for k := range params {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		keyEscaped := url.PathEscape(k)
		if buf.Len() > 0 {
			buf.WriteByte('&')
		}
		buf.WriteString(keyEscaped)
		buf.WriteByte('=')
		buf.WriteString(url.QueryEscape(params[k]))
	}
	return buf.String()
}

func NewUuid() string {
	return uuid.NewV4().String()
}

func RandInt64(min, max int64) int64 {
	if min >= max || min < 0 || max == 0 {
		return max
	}
	return rand.Int63n(max-min) + min
}

func RandInt(min, max int) int {
	if min >= max || min < 0 || max == 0 {
		return max
	}
	return rand.Intn(max-min) + min
}

func RandTime(dur time.Duration) time.Duration {
	start := int64(dur - 3*time.Second)
	if start < 0 {
		start = 0
	}
	return time.Duration(RandInt64(start, int64(dur+3*time.Second)))
}

func RandSec(min, max int) time.Duration {
	return time.Duration(RandInt(min, max)) * time.Second
}

func RandMilli(min, max int) time.Duration {
	return time.Duration(RandInt(min, max)) * time.Millisecond
}

// RandHex 生成16进制格式的随机字符串
func RandHex(n int) []byte {
	if n <= 0 {
		return []byte{}
	}
	var need int
	if n&1 == 0 { // even
		need = n
	} else { // odd
		need = n + 1
	}
	size := need / 2
	dst := make([]byte, need)
	src := dst[size:]
	if _, err := crand.Read(src[:]); err != nil {
		return []byte{}
	}
	hex.Encode(dst, src)
	return dst[:n]
}

func RandMac() string {
	buf := make([]byte, 6)
	crand.Read(buf)

	buf[0] |= 2
	mac := fmt.Sprintf("%02x:%02x:%02x:%02x:%02x:%02x", buf[0], buf[1], buf[2], buf[3], buf[4], buf[5])
	return mac
}

const (
	LettersAlphabet          = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
	LettersNumericalAlphabet = "0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
	LNFAlphabet              = "0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ-_"
	NumericalAlphabet        = "0123456789"
	NumericalNoZeroAlphabet  = "123456789"
	// 6 bits to represent a letter index
	letterIdBits = 6
	// All 1-bits as many as letterIdBits
	letterIdMask = 1<<letterIdBits - 1
	letterIdMax  = 63 / letterIdBits
)

func RandStr(n int) string {
	return RandStrWithAlphabet(n, LettersAlphabet)
}

func RandStrUnsized(min, max int, alphabet string) string {
	return RandStrWithAlphabet(RandInt(min, max), alphabet)
}

func RandStrWithAlphabet(n int, alphabet string) string {
	randSrc := rand.NewSource(time.Now().UnixNano())
	b := make([]byte, n)
	// A rand.Int63() generates 63 random bits, enough for letterIdMax letters!
	for i, cache, remain := n-1, randSrc.Int63(), letterIdMax; i >= 0; {
		if remain == 0 {
			cache, remain = randSrc.Int63(), letterIdMax
		}
		if idx := int(cache & letterIdMask); idx < len(alphabet) {
			b[i] = alphabet[idx]
			i--
		}
		cache >>= letterIdBits
		remain--
	}
	return *(*string)(unsafe.Pointer(&b))
}

func SafeClose(ch chan bool) (justClosed bool) {
	defer func() {
		if recover() != nil {
			// The return result can be altered
			// in a defer function call.
			justClosed = false
		}
	}()

	// assume ch != nil here.
	close(ch)   // panic if ch is closed
	return true // <=> justClosed = true; return
}

// string([]byte) 会发生copy; B2S不会; 但是当原byte发生改变, string也会变
func B2S(b []byte) string {
	return *(*string)(unsafe.Pointer(&b))
}

func S2B(s string) (b []byte) {
	strh := (*reflect.StringHeader)(unsafe.Pointer(&s))
	sh := (*reflect.SliceHeader)(unsafe.Pointer(&b))
	sh.Data = strh.Data
	sh.Len = strh.Len
	sh.Cap = strh.Len
	return b
}

// BytesToString converts byte slice to string.
func BytesToString(b []byte) string {
	return *(*string)(unsafe.Pointer(&b))
}

// StringToBytes converts string to byte slice.
func StringToBytes(s string) []byte {
	return *(*[]byte)(unsafe.Pointer(
		&struct {
			string
			Cap int
		}{s, len(s)},
	))
}

// Cmd 调用命令行
func Cmd(commandName string, params []string) (string, error) {
	cmd := exec.Command(commandName, params...)

	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = os.Stderr
	err := cmd.Start()
	if err != nil {
		return "", err
	}
	err = cmd.Wait()
	return out.String(), err
}
