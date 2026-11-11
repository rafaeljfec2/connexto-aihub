package providers_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"aihub/internal/config"
	"aihub/internal/providers"
)

func newCatalogClient(t *testing.T, serverURL string, httpClient *http.Client) *providers.Client {
	t.Helper()

	client, err := providers.NewClient(config.ProviderOpenCodeGo{
		BaseURL: serverURL,
		APIKey:  "test-key",
	}, httpClient)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	return client
}

func TestFetchModelDescriptorsParsesRealCatalogFixture(t *testing.T) {
	fixture, err := os.ReadFile("testdata/models.json")
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}

	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/v1/models" {
			t.Errorf("upstream saw %s %s, want GET /v1/models", r.Method, r.URL.Path)
		}

		if got := r.Header.Get("Authorization"); got != "Bearer test-key" {
			t.Errorf("Authorization = %q, want Bearer test-key", got)
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(fixture)
	}))
	defer upstream.Close()

	descriptors, err := newCatalogClient(t, upstream.URL, upstream.Client()).FetchModelDescriptors(context.Background())
	if err != nil {
		t.Fatalf("FetchModelDescriptors() error = %v", err)
	}

	if len(descriptors) != 28 {
		t.Fatalf("descriptor count = %d, want 28", len(descriptors))
	}

	missingCanonical := 0
	withInputLimit := 0

	for _, descriptor := range descriptors {
		if descriptor.CanonicalID == "" {
			missingCanonical++
		}

		if descriptor.LimitInput != nil {
			withInputLimit++
		}
	}

	if missingCanonical != 1 {
		t.Fatalf("models without canonical_id = %d, want 1", missingCanonical)
	}

	if withInputLimit != 4 {
		t.Fatalf("models with limit.input = %d, want 4", withInputLimit)
	}

	var deepseek *providers.ModelDescriptor

	for index := range descriptors {
		if descriptors[index].ID == "deepseek-v4.1-flash" {
			deepseek = &descriptors[index]

			break
		}
	}

	if deepseek == nil {
		t.Fatal("deepseek-v4.1-flash not found")
	}

	if deepseek.CanonicalID != "deepseek/deepseek-v4.1-flash" {
		t.Fatalf("canonical id = %q", deepseek.CanonicalID)
	}

	if deepseek.Name != "DeepSeek V4.1 Flash" || deepseek.Type != "chat" || deepseek.Family != "deepseek-flash" {
		t.Fatalf("descriptor metadata = %+v", deepseek)
	}

	if !deepseek.InputModalitiesPresent || !deepseek.OutputModalitiesPresent {
		t.Fatalf("modalities presence = %v/%v, want true/true", deepseek.InputModalitiesPresent, deepseek.OutputModalitiesPresent)
	}

	if !containsString(deepseek.InputModalities, "text") || !containsString(deepseek.InputModalities, "image/png") {
		t.Fatalf("input modalities = %v", deepseek.InputModalities)
	}

	if deepseek.LimitContext == nil || *deepseek.LimitContext != 1000000 {
		t.Fatalf("context limit = %v, want 1000000", deepseek.LimitContext)
	}

	if deepseek.LimitInput != nil {
		t.Fatalf("input limit = %v, want nil", deepseek.LimitInput)
	}

	if deepseek.LimitOutput == nil || *deepseek.LimitOutput != 384000 {
		t.Fatalf("output limit = %v, want 384000", deepseek.LimitOutput)
	}

	if deepseek.ToolsSupported == nil || !*deepseek.ToolsSupported {
		t.Fatalf("tools supported = %v, want true", deepseek.ToolsSupported)
	}

	if deepseek.ReasoningSupported == nil || !*deepseek.ReasoningSupported {
		t.Fatalf("reasoning supported = %v, want true", deepseek.ReasoningSupported)
	}

	if deepseek.StructuredOutput == nil || !*deepseek.StructuredOutput {
		t.Fatalf("structured output = %v, want true", deepseek.StructuredOutput)
	}

	if !json.Valid(deepseek.Raw) || !bytes.Contains(deepseek.Raw, []byte(`"id":"deepseek-v4.1-flash"`)) {
		t.Fatalf("raw metadata was not preserved: %s", deepseek.Raw)
	}
}

func TestFetchModelDescriptorsDistinguishesAbsentAndExplicitValues(t *testing.T) {
	body := `{"data":[{"id":"minimal","name":"Minimal","modalities":{"input":[]},"capabilities":{"tools":{"supported":false}}}]}`

	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(body))
	}))
	defer upstream.Close()

	descriptors, err := newCatalogClient(t, upstream.URL, upstream.Client()).FetchModelDescriptors(context.Background())
	if err != nil {
		t.Fatalf("FetchModelDescriptors() error = %v", err)
	}

	if len(descriptors) != 1 {
		t.Fatalf("descriptor count = %d, want 1", len(descriptors))
	}

	descriptor := descriptors[0]

	if !descriptor.InputModalitiesPresent || descriptor.InputModalities == nil || len(descriptor.InputModalities) != 0 {
		t.Fatalf("input modalities = %v presence=%v, want present empty slice", descriptor.InputModalities, descriptor.InputModalitiesPresent)
	}

	if descriptor.OutputModalitiesPresent {
		t.Fatal("output modalities present = true, want false")
	}

	if descriptor.LimitContext != nil || descriptor.LimitInput != nil || descriptor.LimitOutput != nil {
		t.Fatalf("limits = %v/%v/%v, want nil", descriptor.LimitContext, descriptor.LimitInput, descriptor.LimitOutput)
	}

	if descriptor.ToolsSupported == nil || *descriptor.ToolsSupported {
		t.Fatalf("tools supported = %v, want explicit false", descriptor.ToolsSupported)
	}

	if descriptor.ReasoningSupported != nil {
		t.Fatalf("reasoning supported = %v, want nil", descriptor.ReasoningSupported)
	}

	if descriptor.StructuredOutput != nil {
		t.Fatalf("structured output = %v, want nil", descriptor.StructuredOutput)
	}
}

func TestFetchModelDescriptorsEmptyCatalog(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"object":"list","data":[]}`))
	}))
	defer upstream.Close()

	descriptors, err := newCatalogClient(t, upstream.URL, upstream.Client()).FetchModelDescriptors(context.Background())
	if err != nil {
		t.Fatalf("FetchModelDescriptors() error = %v", err)
	}

	if len(descriptors) != 0 {
		t.Fatalf("descriptor count = %d, want 0", len(descriptors))
	}
}

func TestFetchModelDescriptorsRejectsInvalidPayloads(t *testing.T) {
	cases := []struct {
		name string
		body string
	}{
		{name: "invalid json", body: `{"data":`},
		{name: "empty model id", body: `{"data":[{"id":""}]}`},
		{name: "duplicate model id", body: `{"data":[{"id":"a"},{"id":"a"}]}`},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				_, _ = w.Write([]byte(testCase.body))
			}))
			defer upstream.Close()

			_, err := newCatalogClient(t, upstream.URL, upstream.Client()).FetchModelDescriptors(context.Background())
			if err == nil {
				t.Fatal("FetchModelDescriptors() error = nil, want DecodeError")
			}

			var decodeErr *providers.DecodeError
			if !errors.As(err, &decodeErr) {
				t.Fatalf("error = %T, want *providers.DecodeError", err)
			}

			if decodeErr.StatusCode != http.StatusOK {
				t.Fatalf("decode status = %d, want 200", decodeErr.StatusCode)
			}
		})
	}
}

func TestFetchModelDescriptorsPreservesNon2xxStatus(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"error":"invalid credential"}`))
	}))
	defer upstream.Close()

	_, err := newCatalogClient(t, upstream.URL, upstream.Client()).FetchModelDescriptors(context.Background())
	if err == nil {
		t.Fatal("FetchModelDescriptors() error = nil, want UpstreamHTTPError")
	}

	var statusErr *providers.UpstreamHTTPError
	if !errors.As(err, &statusErr) {
		t.Fatalf("error = %T, want *providers.UpstreamHTTPError", err)
	}

	if statusErr.StatusCode != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", statusErr.StatusCode)
	}
}

func TestFetchModelDescriptorsRejectsOversizedPayload(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(strings.Repeat("a", (8<<20)+1)))
	}))
	defer upstream.Close()

	_, err := newCatalogClient(t, upstream.URL, upstream.Client()).FetchModelDescriptors(context.Background())
	if err == nil {
		t.Fatal("FetchModelDescriptors() error = nil, want DecodeError")
	}

	var decodeErr *providers.DecodeError
	if !errors.As(err, &decodeErr) {
		t.Fatalf("error = %T, want *providers.DecodeError", err)
	}
}

func TestFetchModelDescriptorsContextCancellation(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		<-r.Context().Done()
	}))
	defer upstream.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := newCatalogClient(t, upstream.URL, upstream.Client()).FetchModelDescriptors(ctx)
	if err == nil {
		t.Fatal("FetchModelDescriptors() error = nil, want cancellation error")
	}
}

func containsString(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}

	return false
}
