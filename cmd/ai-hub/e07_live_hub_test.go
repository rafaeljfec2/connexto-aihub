package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"aihub/internal/access"
	"aihub/internal/app"
	"aihub/internal/config"
	"aihub/internal/execution"
	"aihub/internal/gateway"
	"aihub/internal/health"
	"aihub/internal/providers"
	"aihub/internal/records"
	"aihub/internal/server"
)

type e07Hub struct {
	application *app.App
	httpServer  *server.Server
	provider    *providers.Client
	providerMod *providers.Manager
	records     *records.Service
	terminals   *e07TerminalRecorder
	requestIDs  *e07RequestIDs
	baseURL     string
}

func startE07Hub(cfg config.Config, observer *e07ProviderObserver, logger *slog.Logger) (*e07Hub, error) {
	providerClient, err := providers.NewClient(cfg.Provider.OpenCodeGo, e07ProviderHTTPClient(observer, e07LiveRequestTimeout))
	if err != nil {
		return nil, errors.New("provider client setup failed")
	}
	providerModule := providers.NewManager(logger, providerClient)
	recordsModule := records.New(records.Options{
		Dir:               cfg.Catalog.DataDir + "/records",
		Logger:            logger,
		MaxAge:            cfg.Records.MaxAge,
		MaxCount:          cfg.Records.MaxCount,
		MaxStorageBytes:   cfg.Records.MaxStorageBytes,
		SegmentBytes:      cfg.Records.SegmentBytes,
		RetentionInterval: cfg.Records.RetentionInterval,
	})
	terminals := newE07TerminalRecorder(recordsModule)
	executionManager := execution.NewManager(logger, providers.OpenCodeGoID,
		func(ctx context.Context, target execution.Target, payload []byte, contentType, sessionID string) (*http.Response, error) {
			return providerClient.OpenTargetResponses(ctx, providers.ResponsesTarget{
				ProviderID: target.ProviderID,
				ModelID:    target.ModelID,
			}, payload, contentType, sessionID)
		}, gateway.StreamInactivityTimeout, execution.WithObserver(terminals))
	policy, err := access.NewPolicy(access.PolicyConfig{
		APIKey:               cfg.Gateway.APIKey,
		AllowedHosts:         cfg.Gateway.AllowedHosts,
		AllowedHostsExplicit: cfg.Gateway.AllowedHostsSet,
		AllowedOrigins:       cfg.Gateway.AllowedOrigins,
		MaxBodyBytes:         cfg.Gateway.MaxBodyBytes,
	})
	if err != nil {
		return nil, errors.New("gateway policy setup failed")
	}

	readiness := health.NewRegistry()
	router := server.NewRouter(logger, policy, health.Live(), readiness.Ready(),
		server.Route{
			Method:    http.MethodPost,
			Pattern:   "/v1/responses",
			Handler:   gateway.NewResponsesHandlerWithManager(providerClient, executionManager, logger, gateway.StreamInactivityTimeout),
			Admission: access.ResponsesAdmission(policy.MaxBodyBytes()),
		},
		server.Route{Method: http.MethodGet, Pattern: "/v1/records/executions", Handler: gateway.NewExecutionRecordsHandler(recordsModule)},
		server.Route{Method: http.MethodGet, Pattern: "/v1/records/executions/{execution_id}", Handler: gateway.NewExecutionRecordHandler(recordsModule)},
	)
	requestIDs := &e07RequestIDs{}
	serverConfig := cfg.Server
	serverConfig.Port = 0
	httpServer := server.New(serverConfig, &e07ResponseObserver{next: router, requestIDs: requestIDs}, logger)
	lifecycle := newExecutionLifecycle(httpServer, executionManager, cfg.Server.ShutdownTimeout)
	guardedProvider := newGuardedProviderModule(providerModule, executionManager)
	application, err := app.New(logger, guardedProvider, recordsModule, lifecycle)
	if err != nil {
		return nil, errors.New("application setup failed")
	}
	if err := application.Start(context.Background()); err != nil {
		return nil, errors.New("application startup failed")
	}
	if statuses := providerModule.Statuses(); len(statuses) != 1 || !statuses[0].Enabled || statuses[0].State != providers.StateHealthy {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		_ = application.Stop(ctx)
		return nil, errors.New("OpenCode Go provider is unavailable")
	}
	return &e07Hub{
		application: application,
		httpServer:  httpServer,
		provider:    providerClient,
		providerMod: providerModule,
		records:     recordsModule,
		terminals:   terminals,
		requestIDs:  requestIDs,
		baseURL:     "http://" + httpServer.Addr(),
	}, nil
}

func (h *e07Hub) stop() error {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if err := h.application.Stop(ctx); err != nil {
		return errors.New("AI Hub shutdown failed")
	}
	return nil
}

type e07RequestIDs struct {
	mu     sync.Mutex
	values []e07RequestCapture
}

type e07RequestCapture struct {
	ID     string
	Status int
}

func (c *e07RequestIDs) append(value string, status int) {
	if value == "" {
		return
	}
	c.mu.Lock()
	c.values = append(c.values, e07RequestCapture{ID: value, Status: status})
	c.mu.Unlock()
}

func (c *e07RequestIDs) snapshot() []e07RequestCapture {
	c.mu.Lock()
	defer c.mu.Unlock()
	return append([]e07RequestCapture(nil), c.values...)
}

type e07ResponseObserver struct {
	next       http.Handler
	requestIDs *e07RequestIDs
}

func (o *e07ResponseObserver) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	writer := &e07CaptureWriter{ResponseWriter: w}
	o.next.ServeHTTP(writer, r)
	if r.Method == http.MethodPost && r.URL.Path == "/v1/responses" {
		o.requestIDs.append(writer.Header().Get("X-Request-Id"), writer.statusCode())
	}
}

type e07CaptureWriter struct {
	http.ResponseWriter
	status int
}

func (w *e07CaptureWriter) WriteHeader(status int) {
	w.status = status
	w.ResponseWriter.WriteHeader(status)
}

func (w *e07CaptureWriter) Write(content []byte) (int, error) {
	if w.status == 0 {
		w.WriteHeader(http.StatusOK)
	}
	return w.ResponseWriter.Write(content)
}

func (w *e07CaptureWriter) statusCode() int {
	if w.status == 0 {
		return http.StatusOK
	}
	return w.status
}

func (w *e07CaptureWriter) Flush() {
	if flusher, ok := w.ResponseWriter.(http.Flusher); ok {
		flusher.Flush()
	}
}

func (w *e07CaptureWriter) Unwrap() http.ResponseWriter {
	return w.ResponseWriter
}
