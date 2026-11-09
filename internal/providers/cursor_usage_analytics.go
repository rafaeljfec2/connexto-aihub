package providers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

const (
	defaultUsageAnalyticsStaleTTL = 2 * time.Minute
	usageAnalyticsCollectTimeout  = defaultCursorUsageTimeout
)

// Usage analytics envelope statuses surfaced to the frontend.
const (
	UsageAnalyticsStatusAvailable              = "available"
	UsageAnalyticsStatusCollecting             = "collecting"
	UsageAnalyticsStatusAuthenticationRequired = "authentication_required"
	UsageAnalyticsStatusAnalyticsUnavailable   = "analytics_unavailable"
	UsageAnalyticsStatusUpstreamError          = "upstream_error"
	UsageAnalyticsStatusProviderNotFound       = "provider_not_found"
	UsageAnalyticsStatusUnconfiguredGatewayKey = "unconfigured_gateway_key"
)

// CursorUsageAnalyticsEnvelope carries the raw get-user-analytics payload collected
// through the Connexto Proxy, plus the collection metadata the frontend needs to
// render live/refreshing/error states. Analytics is passed through untouched: the
// payload schema is owned by the grok.com console and mapped on the frontend.
type CursorUsageAnalyticsEnvelope struct {
	ProviderID        string              `json:"providerId"`
	ProviderName      string              `json:"providerName"`
	Analytics         json.RawMessage     `json:"analytics"`
	FetchedAt         string              `json:"fetchedAt"`
	SourceCollectedAt string              `json:"sourceCollectedAt,omitempty"`
	Status            string              `json:"status"`
	Error             *ProviderQuotaError `json:"error,omitempty"`
	Refreshing        bool                `json:"refreshing"`
}

type usageAnalyticsCacheEntry struct {
	envelope      CursorUsageAnalyticsEnvelope
	lastAttemptAt time.Time
	refreshing    bool
}

// CursorUsageAnalyticsService caches one analytics payload per cursor provider
// account and refreshes it in the background, mirroring the QuotaService contract:
// serve the cached payload immediately, collect asynchronously when stale, forced,
// or cold, and coalesce concurrent refreshes per account.
type CursorUsageAnalyticsService struct {
	adminService *Service
	httpClient   *http.Client
	logger       *slog.Logger
	ttl          time.Duration
	now          func() time.Time

	mu    sync.Mutex
	cache map[string]*usageAnalyticsCacheEntry
}

type UsageAnalyticsOption func(*CursorUsageAnalyticsService)

// WithUsageAnalyticsTTL sets the time-to-live before a cached analytics payload is
// considered stale.
func WithUsageAnalyticsTTL(ttl time.Duration) UsageAnalyticsOption {
	return func(s *CursorUsageAnalyticsService) {
		if ttl > 0 {
			s.ttl = ttl
		}
	}
}

// NewCursorUsageAnalyticsService builds the analytics collection service.
// The HTTP client is optional; the default keeps the 120s proxy collect budget.
func NewCursorUsageAnalyticsService(
	adminService *Service,
	httpClient *http.Client,
	logger *slog.Logger,
	opts ...UsageAnalyticsOption,
) *CursorUsageAnalyticsService {
	if httpClient == nil {
		httpClient = &http.Client{Timeout: usageAnalyticsCollectTimeout}
	}
	service := &CursorUsageAnalyticsService{
		adminService: adminService,
		httpClient:   httpClient,
		logger:       logger,
		ttl:          defaultUsageAnalyticsStaleTTL,
		now:          time.Now,
		cache:        make(map[string]*usageAnalyticsCacheEntry),
	}
	for _, option := range opts {
		option(service)
	}
	return service
}

// Get returns the cached analytics envelope for the provider immediately and
// triggers a background collection when the cache is cold, stale, or forced.
// An empty providerID selects the first enabled cursor provider; when none is
// configured the returned envelope carries the provider_not_found status.
func (s *CursorUsageAnalyticsService) Get(ctx context.Context, providerID string, force bool) (CursorUsageAnalyticsEnvelope, error) {
	record, resolved, err := s.resolveProviderRecord(ctx, providerID)
	if err != nil {
		return CursorUsageAnalyticsEnvelope{}, err
	}
	background := context.WithoutCancel(ctx)
	nowTime := s.now().UTC()

	s.mu.Lock()
	entry, hasCache := s.cache[record.ID]
	needsRefresh := !hasCache || force || nowTime.Sub(entry.lastAttemptAt) > s.ttl
	if needsRefresh && (!hasCache || !entry.refreshing) {
		next := entry
		if next == nil {
			next = &usageAnalyticsCacheEntry{}
		}
		next.lastAttemptAt = nowTime
		next.refreshing = true
		s.cache[record.ID] = next
		entry = next
		hasCache = true
		s.triggerCollection(background, next, record)
	}

	envelope := CursorUsageAnalyticsEnvelope{
		ProviderID:   record.ID,
		ProviderName: usageAnalyticsProviderName(record),
		Status:       UsageAnalyticsStatusCollecting,
	}
	if hasCache && usageAnalyticsHasData(entry.envelope) {
		envelope = entry.envelope
	}
	envelope.Refreshing = entry.refreshing
	envelope.FetchedAt = nowTime.Format(time.RFC3339)
	s.mu.Unlock()

	if !resolved {
		envelope.Status = UsageAnalyticsStatusProviderNotFound
		envelope.Analytics = nil
		envelope.SourceCollectedAt = ""
		return envelope, nil
	}
	return envelope, nil
}

func usageAnalyticsHasData(envelope CursorUsageAnalyticsEnvelope) bool {
	if envelope.Status == "" || envelope.Status == UsageAnalyticsStatusCollecting {
		return false
	}
	return len(envelope.Analytics) > 0 || envelope.Error != nil
}

func usageAnalyticsProviderName(record Record) string {
	if strings.TrimSpace(record.Name) != "" {
		return record.Name
	}
	return "Cursor"
}

func (s *CursorUsageAnalyticsService) resolveProviderRecord(ctx context.Context, providerID string) (Record, bool, error) {
	if strings.TrimSpace(providerID) != "" {
		record, err := s.adminService.repo.Get(ctx, providerID)
		if err != nil {
			return Record{}, false, err
		}
		if record.Type != TypeCursor {
			return Record{}, false, fmt.Errorf("%w: cursor usage analytics applies only to cursor providers", ErrInvalid)
		}
		return record, true, nil
	}

	records, err := s.adminService.repo.List(ctx)
	if err != nil {
		return Record{}, false, err
	}
	for _, record := range records {
		if record.Type == TypeCursor && record.Enabled {
			return record, true, nil
		}
	}
	return Record{}, false, nil
}

func (s *CursorUsageAnalyticsService) triggerCollection(ctx context.Context, entry *usageAnalyticsCacheEntry, record Record) {
	go func() {
		envelope := s.collect(ctx, record)
		if s.logger != nil && envelope.Status == UsageAnalyticsStatusUpstreamError {
			s.logger.Warn("cursor usage analytics collection failed", slog.String("provider_id", record.ID), slog.Any("error", envelope.Error))
		}
		s.mu.Lock()
		if envelope.Error != nil && len(entry.envelope.Analytics) > 0 {
			previous := entry.envelope
			previous.Error = envelope.Error
			previous.FetchedAt = envelope.FetchedAt
			entry.envelope = previous
			entry.lastAttemptAt = s.now().UTC()
			entry.refreshing = false
			s.mu.Unlock()
			return
		}
		entry.envelope = envelope
		entry.lastAttemptAt = s.now().UTC()
		entry.refreshing = false
		s.mu.Unlock()
	}()
}

func (s *CursorUsageAnalyticsService) collect(ctx context.Context, record Record) CursorUsageAnalyticsEnvelope {
	collectionTime := s.now().UTC().Format(time.RFC3339)
	envelope := CursorUsageAnalyticsEnvelope{
		ProviderID:        record.ID,
		ProviderName:      usageAnalyticsProviderName(record),
		FetchedAt:         collectionTime,
		SourceCollectedAt: collectionTime,
		Status:            UsageAnalyticsStatusCollecting,
	}

	gatewayKey, credErr := s.adminService.managementCredentialOf(ctx, record)
	if credErr != nil {
		return upstreamAnalyticsError(envelope, fmt.Sprintf("falha ao ler credencial do gateway: %v", credErr))
	}
	credential, ok := EffectiveGatewayKeyForAntigravityValidation(gatewayKey)
	if !ok {
		envelope.Status = UsageAnalyticsStatusUnconfiguredGatewayKey
		envelope.Error = &ProviderQuotaError{
			Code:    "unconfigured_gateway_key",
			Message: "Gateway API Key não configurada para comunicação com o proxy",
		}
		return envelope
	}
	target, err := usageAnalyticsTarget(record.Endpoint, s.adminService.allowedHosts)
	if err != nil {
		return upstreamAnalyticsError(envelope, err.Error())
	}

	payload, err := json.Marshal(CursorProxyUsageCollectBody{AccountID: record.ID})
	if err != nil {
		return upstreamAnalyticsError(envelope, fmt.Sprintf("marshal collect body: %v", err))
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodPost, target, bytes.NewReader(payload))
	if err != nil {
		return upstreamAnalyticsError(envelope, fmt.Sprintf("build collect request: %v", err))
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("X-Gateway-API-Key", credential)
	request.Header.Set("X-Hub-Account-Id", record.ID)

	response, err := s.httpClient.Do(request)
	if err != nil {
		return upstreamAnalyticsError(envelope, fmt.Sprintf("proxy collect request: %v", err))
	}
	defer func() { _ = response.Body.Close() }()

	body, err := io.ReadAll(io.LimitReader(response.Body, 1<<21))
	if err != nil {
		return upstreamAnalyticsError(envelope, fmt.Sprintf("read collect response: %v", err))
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return upstreamAnalyticsError(envelope, fmt.Sprintf("Proxy retornou HTTP %d", response.StatusCode))
	}

	var collected CursorProxyUsageCollectResponse
	if err := json.Unmarshal(body, &collected); err != nil {
		return upstreamAnalyticsError(envelope, fmt.Sprintf("decode collect response: %v", err))
	}

	switch collected.Status {
	case "available":
		return analyticsEnvelopeFromUsage(envelope, collected.Usage, collectionTime)
	case "authentication_required", "session_expired":
		envelope.Status = UsageAnalyticsStatusAuthenticationRequired
		envelope.Error = &ProviderQuotaError{
			Code:    "authentication_required",
			Message: "Autentique a conta Cursor no proxy para coletar o uso",
		}
		return envelope
	default:
		envelope.Status = UsageAnalyticsStatusAnalyticsUnavailable
		envelope.Error = &ProviderQuotaError{Code: collected.Status, Message: "Coleta de uso Cursor não disponível"}
		return envelope
	}
}

func analyticsEnvelopeFromUsage(envelope CursorUsageAnalyticsEnvelope, usage map[string]interface{}, collectionTime string) CursorUsageAnalyticsEnvelope {
	raw, present := usage["raw_analytics"]
	if !present || raw == nil {
		envelope.Status = UsageAnalyticsStatusAnalyticsUnavailable
		envelope.Error = &ProviderQuotaError{
			Code:    "analytics_unavailable",
			Message: "O proxy não retornou o analytics de uso do console",
		}
		return envelope
	}
	serialized, err := json.Marshal(raw)
	if err != nil {
		return upstreamAnalyticsError(envelope, fmt.Sprintf("encode analytics payload: %v", err))
	}
	envelope.Analytics = serialized
	envelope.Status = UsageAnalyticsStatusAvailable
	envelope.SourceCollectedAt = collectionTime
	return envelope
}

func upstreamAnalyticsError(envelope CursorUsageAnalyticsEnvelope, message string) CursorUsageAnalyticsEnvelope {
	envelope.Status = UsageAnalyticsStatusUpstreamError
	envelope.Error = &ProviderQuotaError{Code: "upstream_error", Message: message}
	return envelope
}

func usageAnalyticsTarget(endpoint string, allowedHosts map[string]struct{}) (string, error) {
	base := strings.TrimRight(strings.TrimSpace(endpoint), "/")
	if base == "" {
		base = DefaultConnextoProxyEndpoint
	}
	parsed, err := url.Parse(base + "/providers/cursor/usage/collect")
	if err != nil {
		return "", err
	}
	if len(allowedHosts) > 0 {
		host := strings.ToLower(parsed.Hostname())
		if _, allowed := allowedHosts[host]; !allowed {
			return "", fmt.Errorf("host %q not allowed for cursor usage analytics", host)
		}
	}
	return parsed.String(), nil
}
