package compress

import (
	"github.com/andybalholm/brotli"
	"github.com/valyala/bytebufferpool"
	"io"
	"sync"
)

// UnBrotliReader 解压 Brotli
func UnBrotliReader(r io.Reader) ([]byte, error) {
	var bb bytebufferpool.ByteBuffer
	zr, err := acquireBrotliReader(r)
	if err != nil {
		return nil, err
	}
	_, err = copyZeroAlloc(&bb, zr)
	if err != nil {
		return nil, err
	}
	releaseBrotliReader(zr)
	return bb.B, nil
}

func acquireBrotliReader(r io.Reader) (*brotli.Reader, error) {
	v := brotliReaderPool.Get()
	if v == nil {
		return brotli.NewReader(r), nil
	}
	zr := v.(*brotli.Reader)
	if err := zr.Reset(r); err != nil {
		return nil, err
	}
	return zr, nil
}

func releaseBrotliReader(zr *brotli.Reader) {
	brotliReaderPool.Put(zr)
}

var brotliReaderPool sync.Pool
