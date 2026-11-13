package gateway

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"aihub/internal/access"
	"aihub/internal/execution"
	"aihub/internal/metering/capture"
	"aihub/internal/providers"
)

const (
	sessionHeader = "X-Opencode-Session"
	SessionPrefix = "aihub-ses-"
)

type responseSink struct {
	writer      http.ResponseWriter
	inactivity  time.Duration
	tap         *capture.StreamTap
	contentType string
}

func (s *responseSink) Commit(status int, contentType string) {
	s.contentType = contentType
	if contentType != "" {
		s.writer.Header().Set("Content-Type", contentType)
	}
	controller := http.NewResponseController(s.writer)
	_ = controller.SetWriteDeadline(time.Now().Add(s.inactivity))
	s.writer.WriteHeader(status)
}

func (s *responseSink) WriteChunk(chunk []byte) error {
	if s.tap != nil {
		s.tap.Observe(chunk)
	}
	if _, err := s.writer.Write(chunk); err != nil {
		return err
	}
	controller := http.NewResponseController(s.writer)
	if err := controller.Flush(); err != nil && !errors.Is(err, http.ErrNotSupported) {
		return err
	}
	_ = controller.SetWriteDeadline(time.Now().Add(s.inactivity))
	return nil
}

// KeepAlive refreshes the write deadline and, for SSE, emits a comment so
// clients do not treat model silence as a dead connection. It does not count
// as an upstream chunk, so the inactivity watchdog still applies.
func (s *responseSink) KeepAlive() error {
	controller := http.NewResponseController(s.writer)
	if strings.Contains(strings.ToLower(s.contentType), "text/event-stream") {
		if _, err := s.writer.Write([]byte(": ping\n\n")); err != nil {
			return err
		}
		if err := controller.Flush(); err != nil && !errors.Is(err, http.ErrNotSupported) {
			return err
		}
	}
	_ = controller.SetWriteDeadline(time.Now().Add(s.inactivity))
	return nil
}

func newExecutionManager(client *providers.Client, logger *slog.Logger, inactivity time.Duration) *execution.Manager {
	return execution.NewManager(logger, providers.OpenCodeGoID,
		func(ctx context.Context, target execution.Target, payload []byte, contentType, sessionID string) (*http.Response, error) {
			return client.OpenTargetResponses(ctx, providers.ResponsesTarget{
				ProviderID: target.ProviderID,
				ModelID:    target.ModelID,
			}, payload, contentType, sessionID)
		}, inactivity)
}

func providerSessionID(r *http.Request) (string, error) {
	if metadata, ok := access.MetadataFrom(r.Context()); ok && metadata.ProviderSessionID != "" {
		return metadata.ProviderSessionID, nil
	}
	return resolveSessionID(r)
}

func resolveSessionID(r *http.Request) (string, error) {
	if supplied := r.Header.Get(sessionHeader); supplied != "" {
		return supplied, nil
	}
	return newSessionID()
}

func newSessionID() (string, error) {
	return newPrefixedID(SessionPrefix)
}

func newPrefixedID(prefix string) (string, error) {
	var random [16]byte
	if _, err := rand.Read(random[:]); err != nil {
		return "", fmt.Errorf("generate provider session id: %w", err)
	}
	return prefix + hex.EncodeToString(random[:]), nil
}
