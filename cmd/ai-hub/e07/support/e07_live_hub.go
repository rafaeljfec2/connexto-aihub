package support

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"aihub/cmd/ai-hub/lifecycle"

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

type E07Hub struct {
	Application *app.App
	HttpServer  *server.Server
	Provider    *providers.Client
	ProviderMod *providers.Manager
	Records     *records.Service
	Terminals   *E07TerminalRecorder
	RequestIDs  *E07RequestIDs
	BaseURL     string
}

func StartE07Hub(cfg config.Config, observer *E07ProviderObserver, logger *slog.Logger) (*E07Hub, error) {
	providerClient, err := providers.NewClient(cfg.Provider.OpenCodeGo, E07ProviderHTTPClient(observer, E07LiveRequestTimeout))
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
	terminals := NewE07TerminalRecorder(recordsModule)
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
			Handler:   gateway.NewResponsesHandlerWithManager(providerClient, executionManager, logger, gateway.StreamInactivityTimeout, nil),
			Admission: access.ResponsesAdmission(policy.MaxBodyBytes()),
		},
		server.Route{Method: http.MethodGet, Pattern: "/v1/records/executions", Handler: gateway.NewExecutionRecordsHandler(recordsModule)},
		server.Route{Method: http.MethodGet, Pattern: "/v1/records/executions/{execution_id}", Handler: gateway.NewExecutionRecordHandler(recordsModule)},
	)
	requestIDs := &E07RequestIDs{}
	serverConfig := cfg.Server
	serverConfig.Port = 0
	httpServer := server.New(serverConfig, &E07ResponseObserver{Next: router, RequestIDs: requestIDs}, logger)
	serviceLifecycle := lifecycle.NewExecutionLifecycle(httpServer, executionManager, cfg.Server.ShutdownTimeout)
	guardedProvider := lifecycle.NewGuardedProviderModule(providerModule, executionManager)
	application, err := app.New(logger, guardedProvider, recordsModule, serviceLifecycle)
	if err != nil {
		return nil, errors.New("application setup failed")
	}
	if err := application.Start(context.Background()); err != nil {
		return nil, errors.New("application startup failed")
	}
	if statuses := providerModule.Statuses(); len(statuses) != 1 || !statuses[0].Enabled || statuses[0].State != providers.StateUnknown {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		_ = application.Stop(ctx)
		return nil, errors.New("OpenCode Go provider did not start in the unknown health state")
	}
	return &E07Hub{
		Application: application,
		HttpServer:  httpServer,
		Provider:    providerClient,
		ProviderMod: providerModule,
		Records:     recordsModule,
		Terminals:   terminals,
		RequestIDs:  requestIDs,
		BaseURL:     "http://" + httpServer.Addr(),
	}, nil
}

func (h *E07Hub) Stop() error {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if err := h.Application.Stop(ctx); err != nil {
		return errors.New("AI Hub shutdown failed")
	}
	return nil
}

type E07RequestIDs struct {
	Mu     sync.Mutex
	Values []E07RequestCapture
}

type E07RequestCapture struct {
	ID     string
	Status int
}

func (c *E07RequestIDs) Append(value string, status int) {
	if value == "" {
		return
	}
	c.Mu.Lock()
	c.Values = append(c.Values, E07RequestCapture{ID: value, Status: status})
	c.Mu.Unlock()
}

func (c *E07RequestIDs) Snapshot() []E07RequestCapture {
	c.Mu.Lock()
	defer c.Mu.Unlock()
	return append([]E07RequestCapture(nil), c.Values...)
}

type E07ResponseObserver struct {
	Next       http.Handler
	RequestIDs *E07RequestIDs
}

func (o *E07ResponseObserver) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	writer := &E07CaptureWriter{ResponseWriter: w}
	o.Next.ServeHTTP(writer, r)
	if r.Method == http.MethodPost && r.URL.Path == "/v1/responses" {
		o.RequestIDs.Append(writer.Header().Get("X-Request-Id"), writer.StatusCode())
	}
}

type E07CaptureWriter struct {
	http.ResponseWriter
	Status int
}

func (w *E07CaptureWriter) WriteHeader(status int) {
	w.Status = status
	w.ResponseWriter.WriteHeader(status)
}

func (w *E07CaptureWriter) Write(content []byte) (int, error) {
	if w.Status == 0 {
		w.WriteHeader(http.StatusOK)
	}
	return w.ResponseWriter.Write(content)
}

func (w *E07CaptureWriter) StatusCode() int {
	if w.Status == 0 {
		return http.StatusOK
	}
	return w.Status
}

func (w *E07CaptureWriter) Flush() {
	if flusher, ok := w.ResponseWriter.(http.Flusher); ok {
		flusher.Flush()
	}
}

func (w *E07CaptureWriter) Unwrap() http.ResponseWriter {
	return w.ResponseWriter
}
