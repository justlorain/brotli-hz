package main

import (
	"context"
	"fmt"
	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/app/client"
	"github.com/cloudwego/hertz/pkg/app/server"
	"github.com/cloudwego/hertz/pkg/protocol"
	brotli "github.com/justlorain/brotli-hz"
	"net/http"
	"time"
)

func main() {
	h := server.Default()
	h.Use(brotli.Brotli(brotli.DefaultCompression))
	h.GET("/ping", func(ctx context.Context, c *app.RequestContext) {
		c.String(http.StatusOK, "pong "+fmt.Sprint(time.Now().Unix()))
	})
	go h.Spin()

	cli, err := client.NewClient()
	if err != nil {
		panic(err)
	}
	cli.Use(brotli.BrotliClient(brotli.DefaultCompression))

	req := protocol.AcquireRequest()
	res := protocol.AcquireResponse()

	req.SetBodyString("bar")
	req.SetRequestURI("http://localhost:8888/ping")

	if err = cli.Do(context.Background(), req, res); err != nil {
		panic(err)
	}
	fmt.Println(string(res.Body()))
}
