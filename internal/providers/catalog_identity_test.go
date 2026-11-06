package providers

import "testing"

func TestExcludedFromCatalog(t *testing.T) {
	if !ExcludedFromCatalog(LegacyTransportServiceID) {
		t.Fatal("LegacyTransportServiceID must be excluded from catalog")
	}
	for _, id := range []string{CursorID, OpenCodeGoID, OpenRouterID, "other"} {
		if ExcludedFromCatalog(id) {
			t.Fatalf("ExcludedFromCatalog(%q) = true, want false", id)
		}
	}
}

func TestNormalizeLogicalProviderID(t *testing.T) {
	if got := NormalizeLogicalProviderID(LegacyTransportServiceID); got != CursorID {
		t.Fatalf("NormalizeLogicalProviderID(legacy) = %q, want %q", got, CursorID)
	}
	if got := NormalizeLogicalProviderID(CursorID); got != CursorID {
		t.Fatalf("NormalizeLogicalProviderID(cursor) = %q", got)
	}
}
