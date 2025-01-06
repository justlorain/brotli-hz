package main

import (
	"context"
	"fmt"
	"github.com/andybalholm/brotli"
	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/app/client"
	"github.com/cloudwego/hertz/pkg/app/server"
	"github.com/cloudwego/hertz/pkg/protocol"
	"github.com/cloudwego/hertz/pkg/protocol/consts"
	"github.com/justlorain/brotli-hz"
	"io"
	"strings"
	"time"
)

func main() {
	firstData := `chunk 0:`
	secondData := `chunk 1: hi~`
	thirdData := `chunk 2: hi~hi~`
	h := server.Default()

	h.Use(brotli_hz.BrotliStream(brotli_hz.DefaultCompression))
	h.GET("/", func(ctx context.Context, c *app.RequestContext) {
		for i := range 3 {
			_, _ = c.Write([]byte(fmt.Sprintf("chunk %d: %s", i, strings.Repeat("hi~", i))))
			_ = c.Flush()
			time.Sleep(time.Second * 1)
		}
	})

	go h.Spin()

	time.Sleep(time.Second)

	c, err := client.NewClient(client.WithResponseBodyStream(true))
	if err != nil {
		panic(err)
	}

	req := &protocol.Request{}
	resp := &protocol.Response{}
	defer func() {
		protocol.ReleaseRequest(req)
		protocol.ReleaseResponse(resp)
	}()

	req.SetMethod(consts.MethodGet)
	req.SetRequestURI("http://127.0.0.1:8888/")
	req.Header.Set("Accept-Encoding", "br")

	err = c.Do(context.Background(), req, resp)
	if err != nil {
		panic(err)
	}

	bodyStream := resp.BodyStream()
	defer resp.CloseBodyStream() // nolint:errcheck

	r := brotli.NewReader(bodyStream)

	firstChunk := make([]byte, len(firstData))
	_, err = io.ReadFull(r, firstChunk)
	fmt.Println(string(firstChunk))
	if err != nil {
		panic(err)
	}

	err = r.Reset(bodyStream)
	if err != nil {
		panic(err)
	}

	secondChunk := make([]byte, len(secondData))
	_, err = io.ReadFull(r, secondChunk)
	fmt.Println(string(secondChunk))
	if err != nil {
		panic(err)
	}

	err = r.Reset(bodyStream)
	if err != nil {
		panic(err)
	}

	thirdChunk := make([]byte, len(thirdData))
	_, err = io.ReadFull(r, thirdChunk)
	fmt.Println(string(thirdChunk))
	if err != nil {
		panic(err)
	}
}
