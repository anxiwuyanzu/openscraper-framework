package util

import (
	"fmt"
	"math/rand"
	"testing"
	"time"
)

func ExampleRandTime() {
	rand.Seed(time.Now().Unix())
	fmt.Println(RandTime(1 * time.Second))
	fmt.Println(RandTime(1 * time.Second))
	fmt.Println(RandTime(1 * time.Second))

	fmt.Println(RandInt64(3, 9))
	fmt.Println(RandInt64(3, 9))
	fmt.Println(RandInt64(3, 9))
	// output: xx
}

func ExampleSendDingTalk() {
	err := SendDingTalk("cmm test", "809b3ce393cbfa4d07175e2da7d8bcb0cd1b15572b13a4264626927e577cc9f6")
	fmt.Println(err)
	// output: xx
}

func TestRand(t *testing.T) {
	fmt.Println(RandStr(6))
	fmt.Println(RandStr(6))
	fmt.Println(RandStr(6))

	fmt.Println(RandStrWithAlphabet(6, "01234567890"))
	fmt.Println(RandStrWithAlphabet(10, "123456789"))

	fmt.Println(RandStrUnsized(10, 15, NumericalAlphabet))
	fmt.Println(RandStrUnsized(10, 15, NumericalAlphabet))

	fmt.Println(RandStrUnsized(10, 15, NumericalNoZeroAlphabet))

}
