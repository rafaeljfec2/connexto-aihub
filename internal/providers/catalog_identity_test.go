package providers

import "testing"

func TestExcludedFromCatalog(t *testing.T) {
	if !ExcludedFromCatalog(ConnextoProxyID) {
		t.Fatal("Connexto Proxy transport id must be excluded from catalog")
	}
	for _, id := range []string{CursorID, OpenCodeGoID, OpenRouterID, "other"} {
		if ExcludedFromCatalog(id) {
			t.Fatalf("ExcludedFromCatalog(%q) = true, want false", id)
		}
	}
}
