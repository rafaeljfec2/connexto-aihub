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
	cfg        config.Server
	logger     *slog.Logger
	httpServer *http.Server
	listener   net.Listener
	errCh      chan error
}

// New builds the HTTP server module.
func New(cfg config.Server, handler http.Handler, logger *slog.Logger) *Server {
	server := &Server{
		cfg:    cfg,
		logger: logger,
		errCh:  make(chan error, 1),
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

	s.listener = listener
	s.logger.Info("http server listening", "address", listener.Addr().String())

	if !isLoopback(listener.Addr()) {
		s.logger.Warn("gateway is bound to a non-loopback address without authentication; do not expose it before E04",
			"address", listener.Addr().String())
	}

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
