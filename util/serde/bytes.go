package serde

import (
	"bufio"
	"encoding/base64"
	"encoding/hex"
	"os"
	"strings"
)

// StrBytes use for json dump preferred
type StrBytes []byte
type HexBytes []byte

func (f HexBytes) MarshalJSON() ([]byte, error) {
	return []byte(`"` + hex.EncodeToString(f) + `"`), nil
}

func (f StrBytes) MarshalJSON() ([]byte, error) {
	return []byte(`"` + string(f) + `"`), nil
}

func ReadHex(src string) []byte {
	src = strings.ReplaceAll(src, " ", "")
	src = strings.ReplaceAll(src, "\n", "")
	buf, err := hex.DecodeString(src)
	if err != nil {
		panic(err)
	}
	return buf
}

func ReadBase64(src string) []byte {
	buf, err := base64.StdEncoding.DecodeString(src)
	if err != nil {
		panic(err)
	}
	return buf
}

// ReadBytesFromHexBody 从Charles的raw数据读取bytes
func ReadBytesFromHexBody(path string) ([]byte, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	scanner := bufio.NewScanner(file)
	var body string
	for scanner.Scan() {
		l := scanner.Text()
		if len(l) >= 57 {
			body += l[10:57]
		} else if len(l) >= 10 {
			body += l[10:]
		} else {
			body += l
		}
	}

	body = strings.ReplaceAll(body, " ", "")
	return hex.DecodeString(body)
}
