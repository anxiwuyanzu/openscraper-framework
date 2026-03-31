package compress

import (
	"github.com/klauspost/compress/zstd"
	"os"
)

func NewZstdReaderFromFile(file string) (*zstd.Decoder, error) {
	dict, err := os.ReadFile(file)
	if err != nil {
		return nil, err
	}

	return zstd.NewReader(nil, zstd.WithDecoderDicts(dict), zstd.WithDecoderMaxMemory(2*1024*1024))
}
