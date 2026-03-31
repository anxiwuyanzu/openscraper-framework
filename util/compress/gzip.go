package compress

import (
	"bytes"
	"compress/gzip"
	"github.com/anxiwuyanzu/openscraper-framework/spider-common-go/v4/dot"
	"github.com/valyala/bytebufferpool"
	"io"
	"sync"
)

var AGzipPool GzipPool

// GzipPool manages a pool of gzip.Writer.
// The pool uses sync.Pool internally.
type GzipPool struct {
	readers sync.Pool
	writers sync.Pool
}

// GetReader returns gzip.Reader from the pool, or creates a new one
// if the pool is empty.
func (pool *GzipPool) GetReader(src io.Reader) (reader *gzip.Reader) {
	if r := pool.readers.Get(); r != nil {
		reader = r.(*gzip.Reader)
		reader.Reset(src)
	} else {
		reader, _ = gzip.NewReader(src)
	}
	return reader
}

// PutReader closes and returns a gzip.Reader to the pool
// so that it can be reused via GetReader.
func (pool *GzipPool) PutReader(reader *gzip.Reader) {
	reader.Close()
	pool.readers.Put(reader)
}

// GetWriter returns gzip.Writer from the pool, or creates a new one
// with gzip.BestCompression if the pool is empty.
func (pool *GzipPool) GetWriter(dst io.Writer) (writer *gzip.Writer) {
	if w := pool.writers.Get(); w != nil {
		writer = w.(*gzip.Writer)
		writer.Reset(dst)
	} else {
		writer, _ = gzip.NewWriterLevel(dst, gzip.BestCompression)
	}
	return writer
}

// PutWriter closes and returns a gzip.Writer to the pool
// so that it can be reused via GetWriter.
func (pool *GzipPool) PutWriter(writer *gzip.Writer) {
	writer.Close()
	pool.writers.Put(writer)
}

func Gzip(src []byte) []byte {
	var buffer bytes.Buffer
	zw := AGzipPool.GetWriter(&buffer)
	_, err := zw.Write(src)
	if err != nil {
		dot.Logger().WithError(err).Error("gzip error")
		return nil
	}
	AGzipPool.PutWriter(zw)
	return buffer.Bytes()
}

var gzipHeader = []byte{31, 139, 8, 0}

// UnGzip 解压 gzip []byte
func UnGzip(src []byte) []byte {
	if len(src) < 5 || src[0] != gzipHeader[0] || src[1] != gzipHeader[1] ||
		src[2] != gzipHeader[2] || src[3] != gzipHeader[3] {
		return src
	}
	zr := AGzipPool.GetReader(bytes.NewReader(src))
	buf, err := ReadAll(zr)
	if err != nil {
		dot.Logger().WithError(err).Error("gzip error")
		return src
	}
	AGzipPool.PutReader(zr)
	return buf
}

// UnGzipReader 解压 gzip
func UnGzipReader(r io.Reader) ([]byte, error) {
	var bb bytebufferpool.ByteBuffer
	zr := AGzipPool.GetReader(r)
	_, err := copyZeroAlloc(&bb, zr)
	if err != nil {
		return nil, err
	}
	AGzipPool.PutReader(zr)
	return bb.B, nil
}

// ReadAll 替代 ioutil.ReadAll
func ReadAll(r io.Reader) ([]byte, error) {
	var bb bytebufferpool.ByteBuffer
	_, err := copyZeroAlloc(&bb, r)
	if err != nil {
		return nil, err
	}
	return bb.B, nil
}

func copyZeroAlloc(w io.Writer, r io.Reader) (int64, error) {
	vbuf := copyBufPool.Get()
	buf := vbuf.([]byte)
	n, err := io.CopyBuffer(w, r, buf)
	copyBufPool.Put(vbuf)
	return n, err
}

var copyBufPool = sync.Pool{
	New: func() interface{} {
		return make([]byte, 4096)
	},
}
