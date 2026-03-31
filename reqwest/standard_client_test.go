package reqwest

import (
	"fmt"
	_ "github.com/anxiwuyanzu/openscraper-framework/spider-common-go/v4/reqwest/proxz/providers"
	"testing"
	"time"
)

func TestStandardClient_DoRequestTimeout(t *testing.T) {
	client := NewStandardClient(nil, WithTimeout(3*time.Second))

	req := NewStandardRequest()
	req.SetRequestURI("https://www.google.com")

	start := time.Now()
	err := client.DoRequest(req)
	fmt.Println(err, time.Since(start))

	req = NewStandardRequest()
	req.SetRequestURI("https://www.google.com")
	start = time.Now()
	err = client.DoRequestTimeout(req, 5*time.Second)
	fmt.Println(err, time.Since(start))
}

func TestStandardClient_DoRequest(t *testing.T) {
	client := NewStandardClient(nil, WithTimeout(3*time.Second))

	req := NewStandardRequest()

	err := client.DoRequest(req)
	fmt.Println(err)
}

func TestStandardClient_DoRequestTimeoutAndRetry(t *testing.T) {
	client := NewStandardClient(nil, WithTimeout(3*time.Second))

	req := NewStandardRequest()
	req.SetRequestURI("https://www.google.com")

	err := client.DoRequestTimeoutAndRetry(req, 5*time.Second, 3)
	fmt.Println(err)
}
