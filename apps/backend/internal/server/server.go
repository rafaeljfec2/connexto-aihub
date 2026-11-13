// Package server implements the local HTTP transport.
package server

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"sync"
	"sync/atomic"

	"aihub/internal/access"
	"aihub/internal/config"
	"github.com/gofiber/fiber/v2"
)

// maxHeaderBytes limits request headers accepted by the server.
const maxHeaderBytes = 64 << 10

// Server is the HTTP transport module. Fiber owns the network server and the
// routing tree remains injected as a net/http handler to preserve application
// and middleware contracts during the migration.
type Server struct {
	cfg              config.Server
	logger           *slog.Logger
	app              *fiber.App
	listener         *trackedListener
	errCh            chan error
	allowNonLoopback bool
	shuttingDown     atomic.Bool
}

type options struct {
	allowNonLoopback bool
	bodyLimit        int
}

// Option customizes server exposure policy.
type Option func(*options)

// WithNonLoopbackAllowed permits a listener that is not bound to loopback.
// Callers must only enable it after validating explicit gateway access settings.
func WithNonLoopbackAllowed(allowed bool) Option {
	return func(opts *options) {
		opts.allowNonLoopback = allowed
	}
}

// WithBodyLimit sets the transport request-body ceiling in bytes. The
// application admission policy remains the authority for each route. Zero
// selects the gateway default so Fiber's 4 MiB built-in limit cannot reject
// a body the gateway is configured to accept.
func WithBodyLimit(limit int) Option {
	return func(opts *options) {
		opts.bodyLimit = limit
	}
}

// New builds the HTTP server module.
func New(cfg config.Server, handler http.Handler, logger *slog.Logger, opts ...Option) *Server {
	settings := options{}
	for _, option := range opts {
		option(&settings)
	}

	bodyLimit := settings.bodyLimit
	if bodyLimit <= 0 {
		bodyLimit = int(access.DefaultMaxBodyBytes)
	}

	server := &Server{
		cfg:              cfg,
		logger:           logger,
		errCh:            make(chan error, 1),
		allowNonLoopback: settings.allowNonLoopback,
	}

	server.app = fiber.New(fiber.Config{
		ReadTimeout:           cfg.ReadTimeout,
		WriteTimeout:          cfg.WriteTimeout,
		IdleTimeout:           cfg.IdleTimeout,
		ReadBufferSize:        maxHeaderBytes,
		BodyLimit:             bodyLimit,
		DisableStartupMessage: true,
		ErrorHandler:          requestErrorHandler(logger, bodyLimit),
	})
	if accessLogEnabled(os.Getenv(config.EnvHTTPAccessLog)) {
		for _, middleware := range accessLogMiddlewares(os.Stdout) {
			server.app.Use(middleware)
		}
	}
	server.app.All("/*", fiberHTTPHandler(handler))

	return server
}

// requestErrorHandler answers transport failures itself. Returning the error
// makes Fiber log "failed to call ErrorHandler" and replace the status with 500.
func requestErrorHandler(logger *slog.Logger, bodyLimit int) fiber.ErrorHandler {
	return func(c *fiber.Ctx, err error) error {
		code := fiber.StatusInternalServerError
		var fiberErr *fiber.Error
		if errors.As(err, &fiberErr) {
			code = fiberErr.Code
		}

		if logger != nil && code == fiber.StatusRequestEntityTooLarge {
			logger.Warn("request body exceeds transport limit",
				"method", c.Method(),
				"path", c.Path(),
				"limit_bytes", bodyLimit,
			)
		}

		if writeErr := c.SendStatus(code); writeErr != nil && logger != nil {
			logger.Debug("transport error response was not written", "status", code, "error", writeErr)
		}

		return nil
	}
}

// Name implements app.Module.
func (s *Server) Name() string {
	return "http-server"
}

// Start binds the configured address and serves requests in the background.
// Binding happens synchronously so startup failures surface immediately.
func (s *Server) Start(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return fmt.Errorf("start http server: %w", err)
	}

	listener, err := net.Listen("tcp", s.cfg.Address())
	if err != nil {
		return fmt.Errorf("listen on %s: %w", s.cfg.Address(), err)
	}

	if !isListenerAllowed(listener.Addr(), s.allowNonLoopback) {
		_ = listener.Close()
		return fmt.Errorf("listen on %s: non-loopback exposure requires an explicit allowed-host policy", s.cfg.Address())
	}

	s.listener = newTrackedListener(listener)
	s.logger.Info("http server listening", "address", listener.Addr().String())

	go func() {
		if err := s.app.Listener(s.listener); err != nil && !s.shuttingDown.Load() {
			s.errCh <- fmt.Errorf("serve http: %w", err)
		}
	}()

	return nil
}

// isLoopback reports whether the bound address only accepts local connections.
func isLoopback(addr net.Addr) bool {
	if addr == nil {
		return false
	}

	tcpAddr, ok := addr.(*net.TCPAddr)
	if !ok {
		return false
	}

	return tcpAddr.IP.IsLoopback()
}

// isListenerAllowed accepts loopback listeners unconditionally and other TCP
// listeners only after an explicit gateway exposure policy is enabled.
func isListenerAllowed(addr net.Addr, allowNonLoopback bool) bool {
	tcpAddr, ok := addr.(*net.TCPAddr)
	if !ok || tcpAddr == nil {
		return false
	}

	if isLoopback(addr) {
		return true
	}

	return allowNonLoopback
}

// Errors reports fatal serving errors that happen after a successful start.
func (s *Server) Errors() <-chan error {
	return s.errCh
}

// Addr returns the bound address, or the configured address before Start.
func (s *Server) Addr() string {
	if s.listener != nil {
		return s.listener.Addr().String()
	}

	return s.cfg.Address()
}

// Stop gracefully shuts the server down, waiting for in-flight requests until
// the context deadline.
func (s *Server) Stop(ctx context.Context) error {
	s.shuttingDown.Store(true)
	if err := s.app.ShutdownWithContext(ctx); err != nil {
		return fmt.Errorf("shutdown http server: %w", err)
	}

	s.logger.Info("http server stopped")
	return nil
}

// ForceClose closes active connections after the graceful shutdown period ends.
func (s *Server) ForceClose() error {
	if s.listener != nil {
		if err := s.listener.forceClose(); err != nil {
			return fmt.Errorf("force close http server: %w", err)
		}
	}
	return nil
}

// trackedListener gives the lifecycle module a force-close operation because
// fasthttp exposes graceful shutdown but no public active-connection close API.
type trackedListener struct {
	net.Listener
	mu    sync.Mutex
	conns map[net.Conn]struct{}
}

func newTrackedListener(listener net.Listener) *trackedListener {
	return &trackedListener{Listener: listener, conns: make(map[net.Conn]struct{})}
}

func (l *trackedListener) Accept() (net.Conn, error) {
	conn, err := l.Listener.Accept()
	if err != nil {
		return nil, err
	}

	tracked := &trackedConn{Conn: conn, listener: l}
	l.mu.Lock()
	l.conns[tracked] = struct{}{}
	l.mu.Unlock()
	return tracked, nil
}

func (l *trackedListener) Close() error {
	return l.Listener.Close()
}

func (l *trackedListener) forceClose() error {
	err := l.Listener.Close()
	l.mu.Lock()
	conns := make([]net.Conn, 0, len(l.conns))
	for conn := range l.conns {
		conns = append(conns, conn)
	}
	l.mu.Unlock()

	for _, conn := range conns {
		_ = conn.Close()
	}
	return err
}

func (l *trackedListener) remove(conn net.Conn) {
	l.mu.Lock()
	delete(l.conns, conn)
	l.mu.Unlock()
}

type trackedConn struct {
	net.Conn
	listener *trackedListener
	once     sync.Once
}

func (c *trackedConn) Close() error {
	err := c.Conn.Close()
	c.once.Do(func() { c.listener.remove(c) })
	return err
}
