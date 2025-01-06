package brotli_hz

import (
	"bytes"
	"context"
	"github.com/andybalholm/brotli"
	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/protocol/consts"
	"io"
)

// server middleware options
type (
	Option  func(*Options)
	Options struct {
		ExcludedExtensions  ExcludedExtensions
		ExcludedPaths       ExcludedPaths
		ExcludedPathRegexes ExcludedPathRegexes
		DecompressFn        app.HandlerFunc
	}
)

func newOptions(opts ...Option) *Options {
	options := &Options{
		ExcludedExtensions: NewExcludedExtensions([]string{".png", ".gif", ".jpeg", ".jpg"}),
	}
	for _, opt := range opts {
		opt(options)
	}
	return options
}

func WithExcludedExtensions(exts []string) Option {
	return func(o *Options) {
		o.ExcludedExtensions = NewExcludedExtensions(exts)
	}
}

func WithExcludedPaths(paths []string) Option {
	return func(o *Options) {
		o.ExcludedPaths = NewExcludedPaths(paths)
	}
}

func WithExcludedPathRegexes(regexes []string) Option {
	return func(o *Options) {
		o.ExcludedPathRegexes = NewExcludedPathRegexes(regexes)
	}
}

func WithDecompressFn(fn app.HandlerFunc) Option {
	return func(o *Options) {
		o.DecompressFn = fn
	}
}

func DefaultDecompressHandle(_ context.Context, c *app.RequestContext) {
	if len(c.Request.Body()) <= 0 {
		return
	}
	r := brotli.NewReader(bytes.NewReader(c.Request.Body()))
	data, err := io.ReadAll(r)
	if err != nil {
		_ = c.AbortWithError(consts.StatusBadRequest, err)
		return
	}
	c.Request.Header.DelBytes([]byte("Content-Encoding"))
	c.Request.Header.DelBytes([]byte("Content-Length"))
	c.Request.SetBody(data)
}
