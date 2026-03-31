package reqwest

import (
	"bytes"
)

// 从参数值之后开始查找下一个 "&" 的索引
func nextValue(qs []byte) int {
	for i := 0; i < len(qs); i++ {
		if qs[i] == '&' {
			return i
		}
	}
	return len(qs)
}

func setKeyValue(qs []byte, key, value string) []byte {
	qs = bytes.Trim(qs, "&")

	// 检查是否存在同名参数
	if index := findKey(qs, key); index >= 0 {
		// 替换旧的参数值
		vIndex := nextValue(qs[index:])
		return replaceValue(qs, index+len(key)+1, index+vIndex, value)
	}

	// 添加新的参数
	if len(qs) > 0 {
		qs = append(qs, '&')
	}
	qs = append(qs, key...)
	qs = append(qs, '=')
	qs = append(qs, value...)
	return qs
}

func getKeyValue(qs []byte, key string) []byte {
	if index := findKey(qs, key); index >= 0 {
		vIndex := nextValue(qs[index:])
		return qs[index+len(key)+1 : index+vIndex]
	}
	return nil
}

// 查找给定名称的参数的索引，如果不存在，则返回-1。
func findKey(qs []byte, key string) int {
	buf := []byte("&" + key + "=")
	if bytes.HasPrefix(qs, buf[1:]) {
		return 0
	}

	if index := bytes.Index(qs, buf); index >= 0 {
		return index + 1
	}
	return -1
}

// 用新的参数值替换参数中的旧值
func replaceValue(qs []byte, start, end int, value string) []byte {
	pre := make([]byte, 0, start)
	pre = append(pre, qs[:start]...)
	pre = append(pre, value...)
	return append(pre, qs[end:]...)
}

type argsKV struct {
	key     []byte
	value   []byte
	noValue bool
}

type argsScanner struct {
	b []byte
}

func (s *argsScanner) next(kv *argsKV) bool {
	if len(s.b) == 0 {
		return false
	}
	kv.noValue = false

	isKey := true
	k := 0
	for i, c := range s.b {
		switch c {
		case '=':
			if isKey {
				isKey = false
				kv.key = append(kv.key[:0], s.b[:i]...)
				k = i + 1
			}
		case '&':
			if isKey {
				kv.key = append(kv.key[:0], s.b[:i]...)
				kv.value = kv.value[:0]
				kv.noValue = true
			} else {
				kv.value = append(kv.value[:0], s.b[k:i]...)
			}
			s.b = s.b[i+1:]
			return true
		}
	}

	if isKey {
		kv.key = append(kv.key[:0], s.b...)
		kv.value = kv.value[:0]
		kv.noValue = true
	} else {
		kv.value = append(kv.value[:0], s.b[k:]...)
	}
	s.b = s.b[len(s.b):]
	return true
}

//func setKeyValue1(qs []byte, key, value string) []byte {
//	buf := append([]byte{'&'}, key...)
//	buf = append(buf, '=')
//
//	qs = bytes.Trim(qs, "&")
//	if len(qs) == 0 {
//		buf = append(buf[1:], value...)
//		return buf
//	}
//
//	if bytes.HasPrefix(qs, buf[1:]) {
//		index := nextValue(qs)
//		buf = append(qs[index:], buf...)
//		buf = append(buf, value...)
//
//		if buf[0] == '&' { // 只有一个参数
//			buf = buf[1:]
//		}
//
//		return buf
//	}
//
//	if index := bytes.Index(qs, buf); index > 0 {
//		vIndex := nextValue(qs[index+1:])
//		old := append(qs[:index], qs[vIndex+index+1:]...)
//		buf = append(old, buf...)
//		buf = append(buf, value...)
//
//		return buf
//	}
//	buf = append(qs, buf...)
//	buf = append(buf, value...)
//	return buf
//}
