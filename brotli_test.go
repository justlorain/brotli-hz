package brotli_hz

import (
	"bytes"
	"context"
	"fmt"
	"github.com/andybalholm/brotli"
	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/app/client"
	"github.com/cloudwego/hertz/pkg/app/server"
	"github.com/cloudwego/hertz/pkg/common/config"
	"github.com/cloudwego/hertz/pkg/common/ut"
	"github.com/cloudwego/hertz/pkg/protocol"
	"github.com/cloudwego/hertz/pkg/protocol/consts"
	"github.com/cloudwego/hertz/pkg/route"
	"github.com/stretchr/testify/assert"
	"io"
	"net/http"
	"strconv"
	"strings"
	"testing"
	"time"
)

const (
	testResponse = "Brotli Test Response"
)

func newServer() *route.Engine {
	router := route.NewEngine(config.NewOptions([]config.Option{}))
	router.Use(Brotli(DefaultCompression))
	router.GET("/", func(ctx context.Context, c *app.RequestContext) {
		c.Header("Content-Length", strconv.Itoa(len(testResponse)))
		c.String(200, testResponse)
	})
	return router
}

func TestBrotli(t *testing.T) {
	request := ut.PerformRequest(newServer(), consts.MethodGet, "/", nil, ut.Header{
		Key: "Accept-Encoding", Value: "br",
	})
	w := request.Result()
	assert.Equal(t, w.StatusCode(), 200)
	assert.Equal(t, w.Header.Get("Vary"), "Accept-Encoding")
	assert.Equal(t, w.Header.Get("Content-Encoding"), "br")
	assert.NotEqual(t, w.Header.Get("Content-Length"), "0")
	assert.NotEqual(t, len(w.Body()), len(testResponse))
	assert.Equal(t, fmt.Sprint(len(w.Body())), w.Header.Get("Content-Length"))
}

func TestWildcard(t *testing.T) {
	request := ut.PerformRequest(newServer(), consts.MethodGet, "/", nil, ut.Header{
		Key: "Accept-Encoding", Value: "*",
	})
	w := request.Result()
	assert.Equal(t, w.StatusCode(), 200)
	assert.Equal(t, w.Header.Get("Vary"), "Accept-Encoding")
	assert.Equal(t, w.Header.Get("Content-Encoding"), "br")
	assert.NotEqual(t, w.Header.Get("Content-Length"), "0")
	assert.NotEqual(t, len(w.Body()), len(testResponse))
	assert.Equal(t, fmt.Sprint(len(w.Body())), w.Header.Get("Content-Length"))
}

func TestBrotliPNG(t *testing.T) {
	router := route.NewEngine(config.NewOptions([]config.Option{}))
	router.Use(Brotli(DefaultCompression))
	router.GET("/image.png", func(ctx context.Context, c *app.RequestContext) {
		c.String(200, "this is a PNG!")
	})
	request := ut.PerformRequest(router, consts.MethodGet, "/image.png", nil, ut.Header{
		Key: "Accept-Encoding", Value: "br",
	})
	w := request.Result()
	assert.Equal(t, w.StatusCode(), 200)
	assert.Equal(t, w.Header.Get("Content-Encoding"), "")
	assert.Equal(t, w.Header.Get("Vary"), "")
	assert.Equal(t, string(w.Body()), "this is a PNG!")
}

func TestExcludedExtensions(t *testing.T) {
	content := "this is a HTML!"
	router := route.NewEngine(config.NewOptions([]config.Option{}))
	router.Use(Brotli(DefaultCompression, WithExcludedExtensions([]string{".html"})))
	router.GET("/index.html", func(ctx context.Context, c *app.RequestContext) {
		c.String(200, content)
	})
	request := ut.PerformRequest(router, consts.MethodGet, "/index.html", nil, ut.Header{
		Key: "Accept-Encoding", Value: "br",
	})
	w := request.Result()
	assert.Equal(t, http.StatusOK, w.StatusCode())
	assert.Equal(t, "", w.Header.Get("Content-Encoding"))
	assert.Equal(t, "", w.Header.Get("Vary"))
	assert.Equal(t, content, string(w.Body()))
	assert.Equal(t, fmt.Sprint(len(content)), w.Header.Get("Content-Length"))
}

func TestExcludedPaths(t *testing.T) {
	content := "this is books!"
	router := route.NewEngine(config.NewOptions([]config.Option{}))
	router.Use(Brotli(DefaultCompression, WithExcludedPaths([]string{"/api/"})))
	router.GET("/api/books", func(ctx context.Context, c *app.RequestContext) {
		c.String(200, content)
	})
	request := ut.PerformRequest(router, consts.MethodGet, "/api/books", nil, ut.Header{
		Key: "Accept-Encoding", Value: "br",
	})
	w := request.Result()
	assert.Equal(t, http.StatusOK, w.StatusCode())
	assert.Equal(t, "", w.Header.Get("Content-Encoding"))
	assert.Equal(t, "", w.Header.Get("Vary"))
	assert.Equal(t, content, string(w.Body()))
	assert.Equal(t, fmt.Sprint(len(content)), w.Header.Get("Content-Length"))
}

func TestExcludedPathRegexes(t *testing.T) {
	content := "this is a secret!"
	router := route.NewEngine(config.NewOptions([]config.Option{}))
	router.Use(Brotli(DefaultCompression, WithExcludedPathRegexes([]string{`^/secret/.*$`})))
	router.GET("/secret/data", func(ctx context.Context, c *app.RequestContext) {
		c.String(200, content)
	})
	request := ut.PerformRequest(router, consts.MethodGet, "/secret/data", nil, ut.Header{
		Key: "Accept-Encoding", Value: "br",
	})
	w := request.Result()
	assert.Equal(t, http.StatusOK, w.StatusCode())
	assert.Equal(t, "", w.Header.Get("Content-Encoding"))
	assert.Equal(t, "", w.Header.Get("Vary"))
	assert.Equal(t, content, string(w.Body()))
	assert.Equal(t, fmt.Sprint(len(content)), w.Header.Get("Content-Length"))
}

func TestNoBrotli(t *testing.T) {
	request := ut.PerformRequest(newServer(), consts.MethodGet, "/", nil)
	w := request.Result()
	assert.Equal(t, w.StatusCode(), 200)
	assert.Equal(t, w.Header.Get("Content-Encoding"), "")
	assert.Equal(t, w.Header.Get("Content-Length"), fmt.Sprint(len(testResponse)))
	assert.Equal(t, string(w.Body()), testResponse)
}

func TestDecompressBrotli(t *testing.T) {
	var buf bytes.Buffer
	bw := brotli.NewWriterLevel(&buf, DefaultCompression)
	if _, err := bw.Write([]byte(testResponse)); err != nil {
		bw.Close() // nolint:errcheck
		t.Fatal(err)
	}
	bw.Close() // nolint:errcheck

	router := route.NewEngine(config.NewOptions([]config.Option{}))
	router.Use(Brotli(DefaultCompression, WithDecompressFn(DefaultDecompressHandle)))

	router.POST("/", func(ctx context.Context, c *app.RequestContext) {
		if v := c.Request.Header.Get("Content-Encoding"); v != "" {
			t.Errorf("unexpected `Content-Encoding`: %s header", v)
		}
		if v := c.Request.Header.Get("Content-Length"); v != "" {
			t.Errorf("unexpected `Content-Length`: %s header", v)
		}
		data := c.GetRawData()
		c.Data(200, "text/plain", data)
	})

	request := ut.PerformRequest(router, consts.MethodPost, "/", &ut.Body{Body: &buf, Len: buf.Len()}, ut.Header{
		Key: "Content-Encoding", Value: "br",
	})

	w := request.Result()
	assert.Equal(t, http.StatusOK, w.StatusCode())
	assert.Equal(t, "", w.Header.Get("Content-Encoding"))
	assert.Equal(t, "", w.Header.Get("Vary"))
	assert.Equal(t, testResponse, string(w.Body()))
	assert.Equal(t, fmt.Sprint(len(testResponse)), w.Header.Get("Content-Length"))
}

func TestDecompressBrotliWithEmptyBody(t *testing.T) {
	router := route.NewEngine(config.NewOptions([]config.Option{}))
	router.Use(Brotli(DefaultCompression, WithDecompressFn(DefaultDecompressHandle)))
	router.POST("/", func(ctx context.Context, c *app.RequestContext) {
		c.String(200, "ok")
	})

	request := ut.PerformRequest(router, consts.MethodPost, "/", nil, ut.Header{Key: "Content-Encoding", Value: "br"})
	w := request.Result()
	assert.Equal(t, http.StatusOK, w.StatusCode())
	assert.Equal(t, "", w.Header.Get("Content-Encoding"))
	assert.Equal(t, "", w.Header.Get("Vary"))
	assert.Equal(t, "ok", string(w.Body()))
	assert.Equal(t, "2", w.Header.Get("Content-Length"))
}

func TestDecompressBrotliWithSkipFunc(t *testing.T) {
	router := route.NewEngine(config.NewOptions([]config.Option{}))
	router.Use(Brotli(DefaultCompression, WithDecompressFn(DefaultDecompressHandle)))
	router.POST("/", func(ctx context.Context, c *app.RequestContext) {
		c.SetStatusCode(200)
	})

	request := ut.PerformRequest(router, consts.MethodPost, "/", nil, ut.Header{Key: "Accept-Encoding", Value: "br"})
	w := request.Result()
	assert.Equal(t, http.StatusOK, w.StatusCode())
	assert.Equal(t, "br", w.Header.Get("Content-Encoding"))
	assert.Equal(t, "Accept-Encoding", w.Header.Get("Vary"))
	assert.Equal(t, "", string(w.Body()))
	assert.Equal(t, "0", w.Header.Get("Content-Length"))
}

func TestDecompressBrotliWithIncorrectData(t *testing.T) {
	router := route.NewEngine(config.NewOptions([]config.Option{}))
	router.Use(Brotli(DefaultCompression, WithDecompressFn(DefaultDecompressHandle)))
	router.POST("/", func(ctx context.Context, c *app.RequestContext) {
		c.String(200, "ok")
	})
	reader := bytes.NewReader([]byte(testResponse))
	request := ut.PerformRequest(router, consts.MethodPost, "/", &ut.Body{Body: reader, Len: reader.Len()},
		ut.Header{Key: "Content-Encoding", Value: "br"})
	w := request.Result()
	assert.Equal(t, http.StatusBadRequest, w.StatusCode())
}

func TestBrotliClient(t *testing.T) {
	h := server.Default(server.WithHostPorts("127.0.0.1:2333"))

	h.GET("/ping", func(ctx context.Context, c *app.RequestContext) {
		c.Header("Content-Length", strconv.Itoa(len(testResponse)))
		c.String(200, testResponse)
	})
	go h.Spin()
	time.Sleep(time.Second)

	cli, err := client.NewClient()
	if err != nil {
		panic(err)
	}
	cli.Use(BrotliClient(DefaultCompression))

	req := protocol.AcquireRequest()
	res := protocol.AcquireResponse()

	req.SetBodyString("bar")
	req.SetRequestURI("http://127.0.0.1:2333/ping")

	err = cli.Do(context.Background(), req, res)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}

	assert.Equal(t, res.StatusCode(), 200)
	assert.Equal(t, req.Header.Get("Vary"), "Accept-Encoding")
	assert.Equal(t, req.Header.Get("Content-Encoding"), "br")
	assert.NotEqual(t, req.Header.Get("Content-Length"), "0")
	assert.NotEqual(t, fmt.Sprint(len(req.Body())), req.Header.Get("Content-Length"))
}

func TestBrotliClientPNG(t *testing.T) {
	h := server.Default(server.WithHostPorts("127.0.0.1:2334"))

	h.GET("/image.png", func(ctx context.Context, c *app.RequestContext) {
		c.Header("Content-Length", strconv.Itoa(len(testResponse)))
		c.String(200, testResponse)
	})
	go h.Spin()
	time.Sleep(time.Second)

	cli, err := client.NewClient()
	if err != nil {
		panic(err)
	}
	cli.Use(BrotliClient(DefaultCompression, WithClientExcludedExtensions([]string{".png"})))

	req := protocol.AcquireRequest()
	res := protocol.AcquireResponse()

	req.SetBodyString("bar")
	req.SetRequestURI("http://127.0.0.1:2334/image.png")

	err = cli.Do(context.Background(), req, res)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}

	assert.Equal(t, res.StatusCode(), 200)
	assert.Equal(t, req.Header.Get("Vary"), "")
	assert.Equal(t, req.Header.Get("Content-Encoding"), "")
}

func TestClientExcludedExtensions(t *testing.T) {
	h := server.Default(server.WithHostPorts("127.0.0.1:3333"))

	h.GET("/index.html", func(ctx context.Context, c *app.RequestContext) {
		c.Header("Content-Length", strconv.Itoa(len(testResponse)))
		c.String(200, testResponse)
	})
	go h.Spin()
	time.Sleep(time.Second)

	cli, err := client.NewClient()
	if err != nil {
		panic(err)
	}
	cli.Use(BrotliClient(DefaultCompression, WithClientExcludedExtensions([]string{".html"})))

	req := protocol.AcquireRequest()
	res := protocol.AcquireResponse()

	req.SetBodyString("bar")
	req.SetRequestURI("http://127.0.0.1:3333/index.html")

	err = cli.Do(context.Background(), req, res)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}

	assert.Equal(t, res.StatusCode(), 200)
	assert.Equal(t, req.Header.Get("Vary"), "")
	assert.Equal(t, req.Header.Get("Content-Encoding"), "")
}

func TestClientExcludedPaths(t *testing.T) {
	h := server.Default(server.WithHostPorts("127.0.0.1:2336"))

	h.GET("/api/books", func(ctx context.Context, c *app.RequestContext) {
		c.Header("Content-Length", strconv.Itoa(len(testResponse)))
		c.String(200, testResponse)
	})
	go h.Spin()
	time.Sleep(time.Second)

	cli, err := client.NewClient()
	if err != nil {
		panic(err)
	}
	cli.Use(BrotliClient(DefaultCompression, WithClientExcludedPaths([]string{"/api/"})))

	req := protocol.AcquireRequest()
	res := protocol.AcquireResponse()

	req.SetBodyString("bar")
	req.SetRequestURI("http://127.0.0.1:2336/api/books")

	err = cli.Do(context.Background(), req, res)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}

	assert.Equal(t, res.StatusCode(), 200)
	assert.Equal(t, req.Header.Get("Vary"), "")
	assert.Equal(t, req.Header.Get("Content-Encoding"), "")
}

func TestClientNoBrotli(t *testing.T) {
	h := server.Default(server.WithHostPorts("127.0.0.1:2337"))

	h.GET("/", func(ctx context.Context, c *app.RequestContext) {
		c.Header("Content-Length", strconv.Itoa(len(testResponse)))
		c.String(200, testResponse)
	})
	go h.Spin()

	time.Sleep(time.Second)

	cli, err := client.NewClient()
	if err != nil {
		panic(err)
	}
	req := protocol.AcquireRequest()
	res := protocol.AcquireResponse()

	req.SetBodyString("bar")
	req.SetRequestURI("http://127.0.0.1:2337/")

	err = cli.Do(context.Background(), req, res)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}

	assert.Equal(t, res.StatusCode(), 200)
	assert.Equal(t, req.Header.Get("Content-Encoding"), "")
	assert.Equal(t, req.Header.Get("Content-Length"), "3")
}

func TestClientDecompressBrotli(t *testing.T) {
	h := server.Default(server.WithHostPorts("127.0.0.1:2338"))

	h.Use(Brotli(DefaultCompression, WithDecompressFn(DefaultDecompressHandle)))
	h.GET("/", func(ctx context.Context, c *app.RequestContext) {
		c.Header("Content-Length", strconv.Itoa(len(testResponse)))
		c.String(200, testResponse)
	})

	go h.Spin()

	time.Sleep(time.Second)

	cli, err := client.NewClient()
	if err != nil {
		panic(err)
	}
	cli.Use(BrotliClient(DefaultCompression, WithClientDecompressFn(DefaultClientDecompressHandle)))

	req := protocol.AcquireRequest()
	res := protocol.AcquireResponse()

	req.SetBodyString("bar")
	req.SetRequestURI("http://127.0.0.1:2338/")
	req.SetHeader("Accept-Encoding", "br")

	err = cli.Do(context.Background(), req, res)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}

	assert.Equal(t, res.StatusCode(), 200)
	assert.Equal(t, res.Header.Get("Content-Encoding"), "")
	assert.Equal(t, res.Header.Get("Vary"), "")
	assert.Equal(t, testResponse, string(res.Body()))
	assert.Equal(t, fmt.Sprint(len(testResponse)), res.Header.Get("Content-Length"))
}

func TestStreamBrotli(t *testing.T) {
	firstData := `chunk 0:`
	secondData := `chunk 1: hi~`
	thirdData := `chunk 2: hi~hi~`
	h := server.Default(server.WithHostPorts("127.0.0.1:2339"))

	h.Use(BrotliStream(DefaultCompression))
	h.GET("/", func(ctx context.Context, c *app.RequestContext) {
		for i := range 3 {
			_, _ = c.Write([]byte(fmt.Sprintf("chunk %d: %s", i, strings.Repeat("hi~", i))))
			_ = c.Flush()
			time.Sleep(time.Second * 1)
		}
	})

	go h.Spin()

	time.Sleep(time.Second)

	c, _ := client.NewClient(client.WithResponseBodyStream(true))

	req := &protocol.Request{}
	resp := &protocol.Response{}
	defer func() {
		protocol.ReleaseRequest(req)
		protocol.ReleaseResponse(resp)
	}()

	req.SetMethod(consts.MethodGet)
	req.SetRequestURI("http://127.0.0.1:2339/")
	req.Header.Set("Accept-Encoding", "br")

	err := c.Do(context.Background(), req, resp)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}

	bodyStream := resp.BodyStream()
	defer resp.CloseBodyStream() // nolint:errcheck

	r := brotli.NewReader(bodyStream)

	firstChunk := make([]byte, len(firstData))
	_, err = io.ReadFull(r, firstChunk)
	if err != nil {
		t.Fatal(err)
	}

	err = r.Reset(bodyStream)
	if err != nil {
		t.Fatal(err)
	}

	secondChunk := make([]byte, len(secondData))
	_, err = io.ReadFull(r, secondChunk)
	if err != nil {
		t.Fatal(err)
	}

	err = r.Reset(bodyStream)
	if err != nil {
		t.Fatal(err)
	}

	thirdChunk := make([]byte, len(thirdData))
	_, err = io.ReadFull(r, thirdChunk)
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, "br", resp.Header.Get("Content-Encoding"))
	assert.Equal(t, "chunked", resp.Header.Get("Transfer-Encoding"))
	assert.Equal(t, "Accept-Encoding", resp.Header.Get("Vary"))
	assert.Equal(t, firstData, string(firstChunk))
	assert.Equal(t, secondData, string(secondChunk))
	assert.Equal(t, thirdData, string(thirdChunk))
}
