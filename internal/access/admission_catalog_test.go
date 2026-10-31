package access_test

import (
	"net/http"
	"strings"
	"testing"

	"aihub/internal/access"
)

func TestCatalogAdmissionAllowsEmptyBodyWithoutMediaType(t *testing.T) {
	request := admissionRequest("/v1/catalog/sync", "", nil)

	recorder, probe := runAdmission(t, access.CatalogSyncAdmission(1<<20), request)
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", recorder.Code)
	}

	if !probe.called {
		t.Fatal("handler must receive an accepted empty catalog body")
	}
}

func TestCatalogAdmissionRequiresJSONForBody(t *testing.T) {
	request := admissionRequest("/v1/catalog/sync", `{"provider":"x"}`, nil)

	recorder, _ := runAdmission(t, access.CatalogSyncAdmission(1<<20), request)
	if recorder.Code != http.StatusUnsupportedMediaType {
		t.Fatalf("status = %d, want 415", recorder.Code)
	}
}

func TestCatalogAdmissionCapsAtFourKiB(t *testing.T) {
	request := admissionRequest("/v1/catalog/sync", strings.Repeat("a", 4097), map[string][]string{
		"Content-Type": {"application/json"},
	})

	recorder, probe := runAdmission(t, access.CatalogSyncAdmission(1<<20), request)
	if recorder.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("status = %d, want 413", recorder.Code)
	}

	if probe.called {
		t.Fatal("handler must not be called above the catalog cap")
	}
}

func TestCatalogAdmissionAcceptsFourKiB(t *testing.T) {
	request := admissionRequest("/v1/catalog/sync", strings.Repeat("a", 4096), map[string][]string{
		"Content-Type": {"application/json"},
	})

	recorder, _ := runAdmission(t, access.CatalogSyncAdmission(1<<20), request)
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", recorder.Code)
	}
}
