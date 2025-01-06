package brotli_hz

import (
	"bytes"
	"context"
	"github.com/andybalholm/brotli"
	"github.com/cloudwego/hertz/pkg/app/client"
	"github.com/cloudwego/hertz/pkg/protocol"
	"path/filepath"
	"strings"
)

type brotliCliMiddleware struct {
	options *ClientOptions
	level   int
}

func newBrotliCliMiddleware(level int, opts ...ClientOption) *brotliCliMiddleware {
	return &brotliCliMiddleware{
		options: newClientOptions(opts...),
		level:   level,
	}
}

func (bc *brotliCliMiddleware) Handle(next client.Endpoint) client.Endpoint {
	return func(ctx context.Context, req *protocol.Request, resp *protocol.Response) (err error) {
		if !bc.shouldCompress(req) {
			return
		}

		req.SetHeader("Content-Encoding", "br")
		req.SetHeader("Vary", "Accept-Encoding")

		if len(req.Body()) > 0 {
			var buf bytes.Buffer
			w := brotli.NewWriterLevel(&buf, bc.level)
			_, err = w.Write(req.Body())
			if err != nil {
				return
			}
			w.Close() // nolint:errcheck
			req.SetBodyStream(&buf, buf.Len())
		}

		if err = next(ctx, req, resp); err != nil {
			return
		}

		if fn := bc.options.DecompressFn; fn != nil && strings.EqualFold(resp.Header.Get("Content-Encoding"), "br") {
			f := fn(next)
			if err = f(ctx, req, resp); err != nil {
				return
			}
		}

		return
	}
}

func (bc *brotliCliMiddleware) shouldCompress(req *protocol.Request) bool {
	if strings.Contains(req.Header.Get("Connection"), "Upgrade") ||
		strings.Contains(req.Header.Get("Accept"), "text/event-stream") {
		return false
	}

	path := string(req.URI().RequestURI())
	ext := filepath.Ext(path)

	if bc.options.ExcludedExtensions.Contains(ext) {
		return false
	}
	if bc.options.ExcludedPaths.Contains(path) {
		return false
	}
	if bc.options.ExcludedPathRegexes.Contains(path) {
		return false
	}

	return true
}
