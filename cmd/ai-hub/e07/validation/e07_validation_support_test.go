package validation

import (
	e07support "aihub/cmd/ai-hub/e07/support"
	"aihub/internal/access"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestE07TemporaryGatewayKeyIsValidAndDistinct(t *testing.T) {
	providerKey := "provider-validation-key"
	key, err := e07support.E07TemporaryGatewayKey(providerKey)
	if err != nil {
		t.Fatal("temporary gateway key could not be created")
	}
	if err := access.ValidateAPIKey(key); err != nil {
		t.Fatal("temporary gateway key does not satisfy gateway grammar")
	}
	if key == providerKey {
		t.Fatal("temporary gateway key reused the provider key")
	}
}

func TestE07FingerprintDetectsTreeContentChanges(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "state", "config.toml")
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal("fingerprint fixture directory could not be created")
	}
	if err := os.WriteFile(path, []byte("model = \"test\"\n"), 0o600); err != nil {
		t.Fatal("fingerprint fixture could not be written")
	}
	before, err := e07support.E07FingerprintTree(root)
	if err != nil {
		t.Fatal("initial tree fingerprint failed")
	}
	after, err := e07support.E07FingerprintTree(root)
	if err != nil {
		t.Fatal("repeated tree fingerprint failed")
	}
	if before != after {
		t.Fatal("unchanged tree produced different fingerprints")
	}
	if err := os.WriteFile(path, []byte("model = \"changed\"\n"), 0o600); err != nil {
		t.Fatal("fingerprint fixture could not be changed")
	}
	changed, err := e07support.E07FingerprintTree(root)
	if err != nil {
		t.Fatal("changed tree fingerprint failed")
	}
	if before == changed {
		t.Fatal("tree fingerprint ignored a content change")
	}
}

func TestE07FingerprintFollowsRootSymbolicLink(t *testing.T) {
	root := t.TempDir()
	target := filepath.Join(root, "target")
	if err := os.Mkdir(target, 0o700); err != nil {
		t.Fatal("symbolic-link target could not be created")
	}
	if err := os.WriteFile(filepath.Join(target, "state"), []byte("before"), 0o600); err != nil {
		t.Fatal("symbolic-link target file could not be written")
	}
	link := filepath.Join(root, "codex-home")
	if err := os.Symlink(target, link); err != nil {
		t.Skip("symbolic links are unavailable")
	}
	before, err := e07support.E07FingerprintTree(link)
	if err != nil {
		t.Fatal("symbolic-link tree fingerprint failed")
	}
	if err := os.WriteFile(filepath.Join(target, "state"), []byte("after"), 0o600); err != nil {
		t.Fatal("symbolic-link target file could not be changed")
	}
	after, err := e07support.E07FingerprintTree(link)
	if err != nil {
		t.Fatal("changed symbolic-link tree fingerprint failed")
	}
	if before == after {
		t.Fatal("symbolic-link fingerprint ignored target content")
	}
}

func TestE07SSEObserverSeesArrivalsBeforeCompletion(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		flusher := w.(http.Flusher)
		_, _ = w.Write([]byte("data: first\n\n"))
		flusher.Flush()
		time.Sleep(25 * time.Millisecond)
		_, _ = w.Write([]byte("data: second\n\n"))
		flusher.Flush()
	}))
	defer server.Close()

	observation, err := e07support.E07MeasureSSE(server.URL, "test-key", nil)
	if err != nil {
		t.Fatal("SSE observation failed")
	}
	if observation.DataEvents < 2 || !observation.Progressive {
		t.Fatal("SSE observer did not establish progressive delivery")
	}
}

func TestE07ReadResponseRejectsOversizedBody(t *testing.T) {
	response := &http.Response{Body: io.NopCloser(strings.NewReader(strings.Repeat("x", (2<<20)+1)))}
	if _, err := e07support.E07ReadResponse(response); err == nil {
		t.Fatal("oversized HTTP response body was accepted")
	}
}
