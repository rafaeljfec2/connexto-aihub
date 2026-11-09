package providers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

type usageAnalyticsFixture struct {
	service  *CursorUsageAnalyticsService
	proxyHit *atomic.Int32
}

func newUsageAnalyticsFixture(
	t *testing.T,
	respond func(w http.ResponseWriter, r *http.Request) []byte,
	records []Record,
) usageAnalyticsFixture {
	t.Helper()

	cipher, err := NewCredentialCipher([]byte("01234567890123456789012345678901"))
	if err != nil {
		t.Fatalf("cipher init error: %v", err)
	}
	proxyHit := new(atomic.Int32)
	proxy := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		proxyHit.Add(1)
		if r.Method != http.MethodPost {
			t.Errorf("unexpected proxy method: %s", r.Method)
		}
		if got := r.Header.Get("X-Gateway-API-Key"); got != "gw-key" {
			t.Errorf("unexpected gateway key header: %q", got)
		}
		body := []byte{}
		if r.Body != nil {
			body, err = io.ReadAll(io.LimitReader(r.Body, 1<<16))
			if err != nil {
				t.Errorf("read collect body: %v", err)
			}
		}
		if string(body) != `{"accountId":"cursor-main"}` {
			t.Errorf("unexpected collect body: %s", body)
		}
		writeJSONBody(w, respond(w, r))
	}))
	t.Cleanup(proxy.Close)

	prepared := make([]Record, 0, len(records))
	managementCreds := map[string][]byte{}
	for _, record := range records {
		if record.Type == TypeCursor {
			record.Endpoint = proxy.URL
			sealed, sealErr := cipher.SealManagement(record.ID, "gw-key")
			if sealErr != nil {
				t.Fatalf("seal management credential: %v", sealErr)
			}
			managementCreds[record.ID] = sealed
		}
		prepared = append(prepared, record)
	}

	repo := &fakeQuotaRepo{records: prepared, managementCreds: managementCreds}
	admin := NewServiceWithProbes(repo, cipher, NewRegistry(), nil, []string{"127.0.0.1"})
	service := NewCursorUsageAnalyticsService(admin, proxy.Client(), nil)

	return usageAnalyticsFixture{service: service, proxyHit: proxyHit}
}

func cursorProviderRecord(id string, enabled bool) Record {
	return Record{
		ID:      id,
		Name:    "Cursor " + id,
		Type:    TypeCursor,
		Enabled: enabled,
	}
}

func availableCursorAnalyticsBody(accountID string) []byte {
	payload := map[string]interface{}{
		"status":    "available",
		"provider":  "cursor",
		"accountId": accountID,
		"usage": map[string]interface{}{
			"analytics_agent_requests_total": 4,
			"raw_analytics": map[string]interface{}{
				"dailyMetrics": []interface{}{
					map[string]interface{}{"agentRequests": 3},
					map[string]interface{}{"agentRequests": 1},
				},
				"period": map[string]interface{}{
					"startDate": "1762646400000",
					"endDate":   "1794182399999",
				},
			},
		},
	}
	body, err := json.Marshal(payload)
	if err != nil {
		panic(err)
	}
	return body
}

func proxyCursorStatusBody(status string) []byte {
	body, err := json.Marshal(map[string]interface{}{
		"status":    status,
		"provider":  "cursor",
		"accountId": "cursor-main",
		"usage":     map[string]interface{}{},
	})
	if err != nil {
		panic(err)
	}
	return body
}

func writeJSONBody(w http.ResponseWriter, body []byte) {
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write(body)
}

func waitForUsageAnalytics(
	t *testing.T,
	svc *CursorUsageAnalyticsService,
	providerID string,
	condition func(CursorUsageAnalyticsEnvelope) bool,
) CursorUsageAnalyticsEnvelope {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	var envelope CursorUsageAnalyticsEnvelope
	for {
		current, err := svc.Get(context.Background(), providerID, false)
		if err != nil {
			t.Fatalf("Get returned error: %v", err)
		}
		envelope = current
		if condition(envelope) {
			return envelope
		}
		if time.Now().After(deadline) {
			t.Fatalf("usage analytics condition not met in time: %v", envelope)
		}
		time.Sleep(20 * time.Millisecond)
	}
}

func TestCursorUsageAnalytics_ColdCacheCollectsInBackground(t *testing.T) {
	fixture := newUsageAnalyticsFixture(t, func(_ http.ResponseWriter, _ *http.Request) []byte {
		return availableCursorAnalyticsBody("cursor-main")
	}, []Record{cursorProviderRecord("cursor-main", true)})

	envelope, err := fixture.service.Get(context.Background(), "", false)
	if err != nil {
		t.Fatalf("Get returned error: %v", err)
	}
	if envelope.Status != UsageAnalyticsStatusCollecting {
		t.Errorf("expected collecting status on cold cache, got %q", envelope.Status)
	}
	if len(envelope.Analytics) != 0 {
		t.Errorf("expected no analytics on cold cache, got %s", envelope.Analytics)
	}
	if !envelope.Refreshing {
		t.Error("expected refreshing flag on cold cache")
	}

	final := waitForUsageAnalytics(t, fixture.service, "cursor-main", func(env CursorUsageAnalyticsEnvelope) bool {
		return env.Status == UsageAnalyticsStatusAvailable
	})
	if fixture.proxyHit.Load() != 1 {
		t.Errorf("expected a single proxy collect, got %d", fixture.proxyHit.Load())
	}
	if final.ProviderID != "cursor-main" {
		t.Errorf("unexpected provider id %q", final.ProviderID)
	}
	var analytics map[string]any
	if err := json.Unmarshal(final.Analytics, &analytics); err != nil {
		t.Fatalf("analytics payload is not a raw json object: %v", err)
	}
	if metrics, ok := analytics["dailyMetrics"].([]interface{}); !ok || len(metrics) != 2 {
		t.Errorf("expected dailyMetrics array passthrough, got: %s", final.Analytics)
	}
}

func TestCursorUsageAnalytics_TTLServesCacheWithoutProxyHit(t *testing.T) {
	fixture := newUsageAnalyticsFixture(t, func(_ http.ResponseWriter, _ *http.Request) []byte {
		return availableCursorAnalyticsBody("cursor-main")
	}, []Record{cursorProviderRecord("cursor-main", true)})

	if _, err := fixture.service.Get(context.Background(), "cursor-main", false); err != nil {
		t.Fatalf("initial Get returned error: %v", err)
	}
	waitForUsageAnalytics(t, fixture.service, "cursor-main", func(env CursorUsageAnalyticsEnvelope) bool {
		return env.Status == UsageAnalyticsStatusAvailable
	})
	hitsAfterCollection := fixture.proxyHit.Load()

	if _, err := fixture.service.Get(context.Background(), "cursor-main", false); err != nil {
		t.Fatalf("cached Get returned error: %v", err)
	}
	if fixture.proxyHit.Load() != hitsAfterCollection {
		t.Errorf("expected cache hit within TTL, proxy calls went from %d to %d", hitsAfterCollection, fixture.proxyHit.Load())
	}
}

func TestCursorUsageAnalytics_ForceAlwaysCollects(t *testing.T) {
	fixture := newUsageAnalyticsFixture(t, func(_ http.ResponseWriter, _ *http.Request) []byte {
		return availableCursorAnalyticsBody("cursor-main")
	}, []Record{cursorProviderRecord("cursor-main", true)})

	if _, err := fixture.service.Get(context.Background(), "cursor-main", false); err != nil {
		t.Fatalf("initial Get returned error: %v", err)
	}
	waitForUsageAnalytics(t, fixture.service, "cursor-main", func(env CursorUsageAnalyticsEnvelope) bool {
		return env.Status == UsageAnalyticsStatusAvailable
	})
	hitsAfterCollection := fixture.proxyHit.Load()

	waitForUsageAnalytics(t, fixture.service, "cursor-main", func(env CursorUsageAnalyticsEnvelope) bool {
		return !env.Refreshing
	})
	if _, err := fixture.service.Get(context.Background(), "cursor-main", true); err != nil {
		t.Fatalf("forced Get returned error: %v", err)
	}
	waitForUsageAnalytics(t, fixture.service, "cursor-main", func(env CursorUsageAnalyticsEnvelope) bool {
		return !env.Refreshing
	})
	if fixture.proxyHit.Load() == hitsAfterCollection {
		t.Errorf("expected forced refresh to collect again from the proxy")
	}
}

func TestCursorUsageAnalytics_AuthenticationRequired(t *testing.T) {
	fixture := newUsageAnalyticsFixture(t, func(_ http.ResponseWriter, _ *http.Request) []byte {
		return proxyCursorStatusBody("authentication_required")
	}, []Record{cursorProviderRecord("cursor-main", true)})

	envelope := waitForUsageAnalytics(t, fixture.service, "cursor-main", func(env CursorUsageAnalyticsEnvelope) bool {
		return env.Status == UsageAnalyticsStatusAuthenticationRequired
	})
	if len(envelope.Analytics) != 0 {
		t.Errorf("expected no analytics for authentication_required, got %s", envelope.Analytics)
	}
	if envelope.Error == nil || envelope.Error.Code != "authentication_required" {
		t.Errorf("expected authentication_required error, got %+v", envelope.Error)
	}
}

func TestCursorUsageAnalytics_MissingRawAnalytics(t *testing.T) {
	fixture := newUsageAnalyticsFixture(t, func(_ http.ResponseWriter, _ *http.Request) []byte {
		body, err := json.Marshal(map[string]interface{}{
			"status":    "available",
			"accountId": "cursor-main",
			"usage": map[string]interface{}{
				"analytics_agent_requests_total": 4,
			},
		})
		if err != nil {
			panic(err)
		}
		return body
	}, []Record{cursorProviderRecord("cursor-main", true)})

	envelope := waitForUsageAnalytics(t, fixture.service, "cursor-main", func(env CursorUsageAnalyticsEnvelope) bool {
		return env.Status == UsageAnalyticsStatusAnalyticsUnavailable
	})
	if envelope.Error == nil || envelope.Error.Code != "analytics_unavailable" {
		t.Errorf("expected analytics_unavailable error, got %+v", envelope.Error)
	}
}

func TestCursorUsageAnalytics_NoCursorProvider(t *testing.T) {
	fixture := newUsageAnalyticsFixture(t, func(_ http.ResponseWriter, _ *http.Request) []byte {
		return availableCursorAnalyticsBody("cursor-other")
	}, []Record{
		Record{ID: "openrouter-a", Name: "OpenRouter", Type: TypeOpenRouter, Enabled: true},
	})

	envelope, err := fixture.service.Get(context.Background(), "", false)
	if err != nil {
		t.Fatalf("Get returned error: %v", err)
	}
	if envelope.Status != UsageAnalyticsStatusProviderNotFound {
		t.Errorf("expected provider_not_found status, got %q", envelope.Status)
	}
}

func TestCursorUsageAnalytics_UnknownProviderErrors(t *testing.T) {
	fixture := newUsageAnalyticsFixture(t, func(_ http.ResponseWriter, _ *http.Request) []byte {
		return availableCursorAnalyticsBody("cursor-main")
	}, []Record{cursorProviderRecord("cursor-main", true)})

	if _, err := fixture.service.Get(context.Background(), "missing-id", false); !errors.Is(err, ErrNotFound) {
		t.Errorf("expected ErrNotFound for unknown provider, got %v", err)
	}
}

func TestCursorUsageAnalytics_ShortTTLTriggersBackgroundCollection(t *testing.T) {
	cipher, err := NewCredentialCipher([]byte("01234567890123456789012345678901"))
	if err != nil {
		t.Fatalf("cipher init error: %v", err)
	}
	proxyHit := new(atomic.Int32)
	proxy := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		proxyHit.Add(1)
		writeJSONBody(w, availableCursorAnalyticsBody("cursor-main"))
	}))
	t.Cleanup(proxy.Close)

	sealed, sealErr := cipher.SealManagement("cursor-main", "gw-key")
	if sealErr != nil {
		t.Fatalf("seal management credential: %v", sealErr)
	}
	record := cursorProviderRecord("cursor-main", true)
	record.Endpoint = proxy.URL
	repo := &fakeQuotaRepo{
		records:         []Record{record},
		managementCreds: map[string][]byte{"cursor-main": sealed},
	}
	admin := NewServiceWithProbes(repo, cipher, NewRegistry(), nil, []string{"127.0.0.1"})
	service := NewCursorUsageAnalyticsService(
		admin,
		proxy.Client(),
		nil,
		WithUsageAnalyticsTTL(50*time.Millisecond),
	)

	if _, err := service.Get(context.Background(), "cursor-main", false); err != nil {
		t.Fatalf("initial Get returned error: %v", err)
	}
	waitForUsageAnalytics(t, service, "cursor-main", func(env CursorUsageAnalyticsEnvelope) bool {
		return env.Status == UsageAnalyticsStatusAvailable && !env.Refreshing
	})
	hitsAfterCollection := proxyHit.Load()
	time.Sleep(80 * time.Millisecond)

	_ = waitForUsageAnalytics(t, service, "cursor-main", func(env CursorUsageAnalyticsEnvelope) bool {
		return env.Status == UsageAnalyticsStatusAvailable && !env.Refreshing
	})
	if proxyHit.Load() == hitsAfterCollection {
		t.Errorf("expected stale TTL to trigger a background collection (still %d proxy hits)", hitsAfterCollection)
	}
}

// guards against silent regression of error wrapping on upstream HTTP failure.
func TestCursorUsageAnalytics_UpstreamHTTPError(t *testing.T) {
	fixture := newUsageAnalyticsFixture(t, func(w http.ResponseWriter, _ *http.Request) []byte {
		w.WriteHeader(http.StatusBadGateway)
		return []byte(fmt.Sprintf(`{"error":{"type":"upstream","message":"proxy down"}}`))
	}, []Record{cursorProviderRecord("cursor-main", true)})

	envelope := waitForUsageAnalytics(t, fixture.service, "cursor-main", func(env CursorUsageAnalyticsEnvelope) bool {
		return env.Status == UsageAnalyticsStatusUpstreamError
	})
	if envelope.Error == nil || envelope.Error.Code != "upstream_error" {
		t.Errorf("expected upstream_error, got %+v", envelope.Error)
	}
}

// concurrent forced refreshes while a collection is in flight must coalesce into
// a single proxy collect per account.
func TestCursorUsageAnalytics_ConcurrentForcedRefreshCoalesces(t *testing.T) {
	release := make(chan struct{})
	fixture := newUsageAnalyticsFixture(t, func(_ http.ResponseWriter, _ *http.Request) []byte {
		<-release
		return availableCursorAnalyticsBody("cursor-main")
	}, []Record{cursorProviderRecord("cursor-main", true)})

	if _, err := fixture.service.Get(context.Background(), "cursor-main", true); err != nil {
		t.Fatalf("first forced Get returned error: %v", err)
	}
	if _, err := fixture.service.Get(context.Background(), "cursor-main", true); err != nil {
		t.Fatalf("second forced Get returned error: %v", err)
	}

	close(release)
	waitForUsageAnalytics(t, fixture.service, "cursor-main", func(env CursorUsageAnalyticsEnvelope) bool {
		return env.Status == UsageAnalyticsStatusAvailable && !env.Refreshing
	})

	if fixture.proxyHit.Load() != 1 {
		t.Errorf("expected a single coalesced proxy collect, got %d", fixture.proxyHit.Load())
	}
}

func TestCursorUsageAnalytics_RefreshFailureKeepsLastSuccessfulAnalytics(t *testing.T) {
	var calls atomic.Int32
	fixture := newUsageAnalyticsFixture(t, func(w http.ResponseWriter, _ *http.Request) []byte {
		if calls.Add(1) == 1 {
			return availableCursorAnalyticsBody("cursor-main")
		}
		w.WriteHeader(http.StatusBadGateway)
		return []byte(`{"error":"proxy unavailable"}`)
	}, []Record{cursorProviderRecord("cursor-main", true)})

	waitForUsageAnalytics(t, fixture.service, "cursor-main", func(envelope CursorUsageAnalyticsEnvelope) bool {
		return envelope.Status == UsageAnalyticsStatusAvailable && len(envelope.Analytics) > 0
	})
	if _, err := fixture.service.Get(context.Background(), "cursor-main", true); err != nil {
		t.Fatalf("forced Get returned error: %v", err)
	}

	envelope := waitForUsageAnalytics(t, fixture.service, "cursor-main", func(envelope CursorUsageAnalyticsEnvelope) bool {
		return !envelope.Refreshing && envelope.Error != nil
	})
	if envelope.Status != UsageAnalyticsStatusAvailable || len(envelope.Analytics) == 0 {
		t.Fatalf("expected last successful analytics to remain available, got %+v", envelope)
	}
}
