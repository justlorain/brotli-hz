package brotli_hz

import (
	"github.com/andybalholm/brotli"
	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/app/client"
)

const (
	BestSpeed          = brotli.BestSpeed
	BestCompression    = brotli.BestCompression
	DefaultCompression = brotli.DefaultCompression
)

func Brotli(level int, opts ...Option) app.HandlerFunc {
	return newBrotliSrvMiddleware(level, opts...).Handle
}

func BrotliStream(level int, opts ...Option) app.HandlerFunc {
	return newBrotliSrvMiddleware(level, opts...).StreamHandle
}

func BrotliClient(level int, opts ...ClientOption) client.Middleware {
	return newBrotliCliMiddleware(level, opts...).Handle
}
