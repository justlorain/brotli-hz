package brotli_hz

import (
	"bytes"
	"context"
	"github.com/andybalholm/brotli"
	"github.com/cloudwego/hertz/pkg/app/client"
	"github.com/cloudwego/hertz/pkg/protocol"
	"io"
)

// client middleware options
type (
	ClientOption  func(*ClientOptions)
	ClientOptions struct {
		ExcludedExtensions  ExcludedExtensions
		ExcludedPaths       ExcludedPaths
		ExcludedPathRegexes ExcludedPathRegexes
		DecompressFn        client.Middleware
	}
)

func newClientOptions(opts ...ClientOption) *ClientOptions {
	options := &ClientOptions{
		ExcludedExtensions: NewExcludedExtensions([]string{".png", ".gif", ".jpeg", ".jpg"}),
	}
	for _, opt := range opts {
		opt(options)
	}
	return options
}

func WithClientExcludedExtensions(exts []string) ClientOption {
	return func(o *ClientOptions) {
		o.ExcludedExtensions = NewExcludedExtensions(exts)
	}
}

func WithClientExcludedPaths(paths []string) ClientOption {
	return func(o *ClientOptions) {
		o.ExcludedPaths = NewExcludedPaths(paths)
	}
}

func WithClientExcludedPathRegexes(regexes []string) ClientOption {
	return func(o *ClientOptions) {
		o.ExcludedPathRegexes = NewExcludedPathRegexes(regexes)
	}
}

func WithClientDecompressFn(fn client.Middleware) ClientOption {
	return func(o *ClientOptions) {
		o.DecompressFn = fn
	}
}

func DefaultClientDecompressHandle(_ client.Endpoint) client.Endpoint {
	return func(ctx context.Context, req *protocol.Request, resp *protocol.Response) (err error) {
		if len(resp.Body()) <= 0 {
			return
		}
		r := brotli.NewReader(bytes.NewReader(resp.Body()))
		data, err := io.ReadAll(r)
		if err != nil {
			return
		}
		resp.Header.DelBytes([]byte("Content-Encoding"))
		resp.Header.DelBytes([]byte("Content-Length"))
		resp.Header.DelBytes([]byte("Vary"))
		resp.SetBodyStream(bytes.NewBuffer(data), len(data))
		return
	}
}
