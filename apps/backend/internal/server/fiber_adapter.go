package server

import (
	"bufio"
	"context"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/adaptor"
	"github.com/valyala/fasthttp"
)

var errResponseConnClosed = errors.New("response connection is closed")

// fiberHTTPHandler keeps the existing net/http handler contract at the
// application boundary while using Fiber as the only network server. The
// response is streamed through fasthttp's body stream writer instead of the
// generic HTTP adaptor, whose response writer buffers the whole body.
func fiberHTTPHandler(handler http.Handler) fiber.Handler {
	return func(c *fiber.Ctx) error {
		request, err := adaptor.ConvertRequest(c, true)
		if err != nil {
			return fiber.ErrBadRequest
		}

		requestContext := fiberRequestContext(c.Context())
		if c.Locals(accessLogManagedContextKey{}) == true {
			requestContext = context.WithValue(requestContext, accessLogManagedContextKey{}, true)
		}
		request = request.WithContext(requestContext)
		fasthttpContext := c.Context()
		writer := newFiberResponseWriter(fasthttpContext)
		fasthttpContext.SetBodyStreamWriter(func(stream *bufio.Writer) {
			// Drop the context before fasthttp resets it. A late deadline
			// update otherwise calls SetWriteDeadline on a nil connection.
			defer writer.releaseContext()
			for operation := range writer.operations {
				if operation.flush {
					if err := stream.Flush(); err != nil {
						writer.setWriteError(err)
						return
					}
					continue
				}
				if _, err := stream.Write(operation.body); err != nil {
					writer.setWriteError(err)
					return
				}
			}
			_ = stream.Flush()
		})
		go func() {
			defer writer.finish()
			handler.ServeHTTP(writer, request)
		}()
		<-writer.committed
		return nil
	}
}

// fiberRequestContext makes the lifecycle dependency explicit. fasthttp's
// request context is valid only during this request and is never retained by
// the adapter after the stream callback returns.
func fiberRequestContext(requestContext *fasthttp.RequestCtx) context.Context {
	// fasthttp cancels RequestCtx.Done when the server shuts down. That would
	// race E05's explicit shutdown arbitration and incorrectly record a client
	// cancellation. The stream writer still propagates a real disconnect as a
	// write error, so request work remains cancelable without inheriting the
	// transport's shutdown signal.
	return context.WithoutCancel(requestContext)
}

type fiberResponseWriter struct {
	context    *fasthttp.RequestCtx
	header     http.Header
	status     int
	mu         sync.Mutex
	operations chan responseOperation
	committed  chan struct{}
	finished   chan struct{}
	commitOnce sync.Once
	finishOnce sync.Once
	closeOnce  sync.Once
	errorMu    sync.RWMutex
	writeErr   error
}

type responseOperation struct {
	body  []byte
	flush bool
}

func newFiberResponseWriter(requestContext *fasthttp.RequestCtx) *fiberResponseWriter {
	return &fiberResponseWriter{
		context:    requestContext,
		header:     make(http.Header),
		operations: make(chan responseOperation),
		committed:  make(chan struct{}),
		finished:   make(chan struct{}),
	}
}

func (w *fiberResponseWriter) Header() http.Header {
	return w.header
}

func (w *fiberResponseWriter) WriteHeader(status int) {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.status != 0 {
		return
	}

	w.status = status
	for key, values := range w.header {
		if strings.EqualFold(key, "Content-Length") && len(values) > 0 {
			if n, err := strconv.Atoi(values[0]); err == nil {
				w.context.Response.Header.SetContentLength(n)
				continue
			}
		}
		for _, value := range values {
			w.context.Response.Header.Add(key, value)
		}
	}
	w.context.SetStatusCode(status)
	w.commitOnce.Do(func() { close(w.committed) })
}

func (w *fiberResponseWriter) Write(body []byte) (int, error) {
	if w.status == 0 {
		w.WriteHeader(http.StatusOK)
	}

	if err := w.send(responseOperation{body: append([]byte(nil), body...)}); err != nil {
		return 0, err
	}
	return len(body), nil
}

func (w *fiberResponseWriter) Flush() {
	_ = w.FlushError()
}

func (w *fiberResponseWriter) FlushError() error {
	if w.status == 0 {
		w.WriteHeader(http.StatusOK)
	}
	return w.send(responseOperation{flush: true})
}

func (w *fiberResponseWriter) SetWriteDeadline(deadline time.Time) error {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.context == nil {
		return errResponseConnClosed
	}
	conn := w.context.Conn()
	if conn == nil {
		return errResponseConnClosed
	}
	return conn.SetWriteDeadline(deadline)
}

func (w *fiberResponseWriter) releaseContext() {
	w.mu.Lock()
	w.context = nil
	w.mu.Unlock()
}

func (w *fiberResponseWriter) Unwrap() http.ResponseWriter {
	return w
}

func (w *fiberResponseWriter) send(operation responseOperation) error {
	select {
	case w.operations <- operation:
		return nil
	case <-w.finished:
		return w.error()
	}
}

func (w *fiberResponseWriter) setWriteError(err error) {
	w.errorMu.Lock()
	w.writeErr = err
	w.errorMu.Unlock()
	w.finishOnce.Do(func() { close(w.finished) })
}

func (w *fiberResponseWriter) error() error {
	w.errorMu.RLock()
	defer w.errorMu.RUnlock()
	if w.writeErr != nil {
		return w.writeErr
	}
	return errors.New("fiber response stream closed")
}

func (w *fiberResponseWriter) finish() {
	if w.status == 0 {
		w.WriteHeader(http.StatusOK)
	}
	w.finishOnce.Do(func() {
		close(w.finished)
	})
	w.closeOnce.Do(func() { close(w.operations) })
}
