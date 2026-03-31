package compress

import (
	"github.com/klauspost/compress/zlib"
	"github.com/valyala/bytebufferpool"
	"io"
	"sync"
)

var flateReaderPool sync.Pool

// UnFlateReader 解压 Flate
func UnFlateReader(r io.Reader) ([]byte, error) {
	var bb bytebufferpool.ByteBuffer
	zr, err := acquireFlateReader(r)
	if err != nil {
		return nil, err
	}
	_, err = copyZeroAlloc(&bb, zr)
	if err != nil {
		return nil, err
	}
	releaseFlateReader(zr)
	return bb.B, nil
}

func acquireFlateReader(r io.Reader) (io.ReadCloser, error) {
	v := flateReaderPool.Get()
	if v == nil {
		zr, err := zlib.NewReader(r)
		if err != nil {
			return nil, err
		}
		return zr, nil
	}
	zr := v.(io.ReadCloser)
	if err := resetFlateReader(zr, r); err != nil {
		return nil, err
	}
	return zr, nil
}

func releaseFlateReader(zr io.ReadCloser) {
	zr.Close()
	flateReaderPool.Put(zr)
}

func resetFlateReader(zr io.ReadCloser, r io.Reader) error {
	zrr, ok := zr.(zlib.Resetter)
	if !ok {
		panic("BUG: zlib.Reader doesn't implement zlib.Resetter???")
	}
	return zrr.Reset(r, nil)
}
