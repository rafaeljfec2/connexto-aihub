// Package server implements the local HTTP transport.
package server

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"

	"aihub/internal/config"
)

// maxHeaderBytes limits request headers accepted by the server.
const maxHeaderBytes = 64 << 10

// Server is the HTTP transport module. It owns the listener and the http.Server
// instance, while the routing tree is injected by the caller.
type Server struct {
	cfg              config.Server
	logger           *slog.Logger
	httpServer       *http.Server
	listener         net.Listener
	errCh            chan error
	allowNonLoopback bool
}

type options struct {
	allowNonLoopback bool
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

// New builds the HTTP server module.
func New(cfg config.Server, handler http.Handler, logger *slog.Logger, opts ...Option) *Server {
	settings := options{}
	for _, option := range opts {
		option(&settings)
	}

	server := &Server{
		cfg:              cfg,
		logger:           logger,
		errCh:            make(chan error, 1),
		allowNonLoopback: settings.allowNonLoopback,
	}

	server.httpServer = &http.Server{
		Addr:              cfg.Address(),
		Handler:           handler,
		ReadHeaderTimeout: cfg.ReadHeaderTimeout,
		ReadTimeout:       cfg.ReadTimeout,
		WriteTimeout:      cfg.WriteTimeout,
		IdleTimeout:       cfg.IdleTimeout,
		MaxHeaderBytes:    maxHeaderBytes,
		ErrorLog:          slog.NewLogLogger(logger.Handler(), slog.LevelError),
	}

	return server
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

	s.listener = listener
	s.logger.Info("http server listening", "address", listener.Addr().String())

	go func() {
		if err := s.httpServer.Serve(listener); err != nil && !errors.Is(err, http.ErrServerClosed) {
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
	if err := s.httpServer.Shutdown(ctx); err != nil {
		return fmt.Errorf("shutdown http server: %w", err)
	}

	s.logger.Info("http server stopped")
	return nil
}

// ForceClose closes active connections after the graceful shutdown period ends.
func (s *Server) ForceClose() error {
	if err := s.httpServer.Close(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return fmt.Errorf("force close http server: %w", err)
	}
	return nil
}
