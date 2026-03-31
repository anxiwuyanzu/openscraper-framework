package util

import (
	"errors"
	"fmt"
	"github.com/tidwall/gjson"
	"io/ioutil"
	"net/http"
	"strings"
)

// 发送 content 到钉钉
func SendDingTalk(content, accessToken string) error {
	body := fmt.Sprintf(`{"msgtype":"text","text":{"content":"%s"}}`, content)
	uri := fmt.Sprintf("https://oapi.dingtalk.com/robot/send?access_token=%s", accessToken)
	resp, err := http.Post(uri, "application/json", strings.NewReader(body))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	b, _ := ioutil.ReadAll(resp.Body)
	code := gjson.GetBytes(b, "errcode").Int()
	if code != 0 {
		return errors.New(gjson.GetBytes(b, "errmsg").String())
	}

	return nil
}
