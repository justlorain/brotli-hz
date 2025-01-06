package brotli_hz

import (
	"bytes"
	"context"
	"github.com/andybalholm/brotli"
	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/network"
	"github.com/cloudwego/hertz/pkg/protocol"
	"github.com/cloudwego/hertz/pkg/protocol/http1/ext"
	"github.com/cloudwego/hertz/pkg/protocol/http1/resp"
	"strings"
	"sync"
)

type brotliChunkedWriter struct {
	sync.Once
	finalizeErr error
	wroteHeader bool
	r           *protocol.Response
	w           network.Writer
	level       int
}

func NewBrotliChunkedWriter(r *protocol.Response, w network.Writer, level int) network.ExtWriter {
	return &brotliChunkedWriter{
		r:     r,
		w:     w,
		level: level,
	}
}

func (bc *brotliChunkedWriter) Write(p []byte) (n int, err error) {
	var buf bytes.Buffer
	w := brotli.NewWriterLevel(&buf, bc.level)
	if _, err = w.Write(p); err != nil {
		return
	}
	w.Close() // nolint:errcheck

	if !bc.wroteHeader {
		bc.r.Header.SetContentLength(-1)
		bc.r.Header.Set("Content-Encoding", "br")
		bc.r.Header.Set("Vary", "Accept-Encoding")
		if err = resp.WriteHeader(&bc.r.Header, bc.w); err != nil {
			return
		}
		bc.wroteHeader = true
	}

	if err = ext.WriteChunk(bc.w, buf.Bytes(), false); err != nil {
		return
	}

	n = buf.Len()
	return
}

func (bc *brotliChunkedWriter) Flush() error {
	return bc.w.Flush()
}

func (bc *brotliChunkedWriter) Finalize() error {
	bc.Do(func() {
		// in case no actual data from user
		if !bc.wroteHeader {
			// use Transfer-Encoding: chunked.
			bc.r.Header.SetContentLength(-1)
			bc.r.Header.Set("Content-Encoding", "br")
			bc.r.Header.Set("Vary", "Accept-Encoding")
			if bc.finalizeErr = resp.WriteHeader(&bc.r.Header, bc.w); bc.finalizeErr != nil {
				return
			}
			bc.wroteHeader = true
		}

		// write the ending chunk
		bc.finalizeErr = ext.WriteChunk(bc.w, nil, true)
		if bc.finalizeErr != nil {
			return
		}

		// write trailer
		bc.finalizeErr = ext.WriteTrailer(bc.r.Header.Trailer(), bc.w)
	})
	return bc.finalizeErr
}

func (bs *brotliSrvMiddleware) StreamHandle(ctx context.Context, c *app.RequestContext) {
	if fn := bs.options.DecompressFn; fn != nil && strings.EqualFold(c.Request.Header.Get("Content-Encoding"), "br") {
		fn(ctx, c)
	}

	if !bs.shouldCompress(&c.Request) {
		return
	}

	w := NewBrotliChunkedWriter(&c.Response, c.GetWriter(), bs.level)
	c.Response.HijackWriter(w)

	c.Next(ctx)
}
