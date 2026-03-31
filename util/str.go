package util

import "strings"

// SplitAndFilter 将字符串按 sep 分割, 并且过滤空格
func SplitAndFilter(str, sep string) []string {
	str = strings.TrimSpace(str)
	spl := strings.Split(str, sep)
	var ret []string
	for _, s := range spl {
		if a := strings.TrimSpace(s); len(a) > 0 {
			ret = append(ret, a)
		}
	}
	return ret
}

// StringToCookie 将 passport_csrf_token=db1839bd38bc63fdb3f7abaceac5df54; Path=/; Domain=douyin.com; Max-Age=5184000; Secure; SameSite=None; 和
// passport_csrf_token=db1839bd38bc63fdb3f7abaceac5df54; 的字符串转成成 cookie 的 key-value map
func StringToCookie(ck string) map[string]string {
	cookies := make(map[string]string)
	cks := SplitAndFilter(ck, ";")
	for _, v := range cks {
		kv := SplitAndFilter(v, "=")
		if len(kv) == 2 {
			if kv[0] != "Path" && kv[0] != "Expires" && kv[0] != "Domain" && kv[0] != "Max-Age" && kv[0] != "Secure" && kv[0] != "SameSite" {
				cookies[kv[0]] = kv[1]
			}
		}
	}
	return cookies
}
