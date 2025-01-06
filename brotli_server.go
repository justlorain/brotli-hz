package brotli_hz

import (
	"bytes"
	"context"
	"github.com/andybalholm/brotli"
	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/protocol"
	"github.com/cloudwego/hertz/pkg/protocol/consts"
	"path/filepath"
	"strings"
)

type brotliSrvMiddleware struct {
	options *Options
	level   int
}

func newBrotliSrvMiddleware(level int, opts ...Option) *brotliSrvMiddleware {
	return &brotliSrvMiddleware{
		options: newOptions(opts...),
		level:   level,
	}
}

func (bs *brotliSrvMiddleware) Handle(ctx context.Context, c *app.RequestContext) {
	if fn := bs.options.DecompressFn; fn != nil && strings.EqualFold(c.Request.Header.Get("Content-Encoding"), "br") {
		fn(ctx, c)
	}

	if !bs.shouldCompress(&c.Request) {
		return
	}

	c.Next(ctx)

	c.Header("Content-Encoding", "br")
	c.Header("Vary", "Accept-Encoding")

	// use brotli in empty body
	if len(c.Response.Body()) <= 0 {
		return
	}

	var buf bytes.Buffer
	w := brotli.NewWriterLevel(&buf, bs.level)
	defer func() {
		w.Close() // nolint:errcheck
		c.Response.SetBodyStream(&buf, buf.Len())
	}()
	_, err := w.Write(c.Response.Body())
	if err != nil {
		_ = c.AbortWithError(consts.StatusBadRequest, err)
	}
}

func (bs *brotliSrvMiddleware) shouldCompress(req *protocol.Request) bool {
	if !(strings.Contains(req.Header.Get("Accept-Encoding"), "br") ||
		strings.TrimSpace(req.Header.Get("Accept-Encoding")) == "*") ||
		strings.Contains(req.Header.Get("Connection"), "Upgrade") ||
		strings.Contains(req.Header.Get("Content-Type"), "text/event-stream") {
		return false
	}

	path := string(req.URI().RequestURI())
	ext := filepath.Ext(path)

	if bs.options.ExcludedExtensions.Contains(ext) {
		return false
	}
	if bs.options.ExcludedPaths.Contains(path) {
		return false
	}
	if bs.options.ExcludedPathRegexes.Contains(path) {
		return false
	}

	return true
}
