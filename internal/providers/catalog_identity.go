package providers

// ExcludedFromCatalog reports provider IDs that must not appear as separate catalog
// providers. The Connexto Proxy CLI adapter is transport-only; models discovered
// through it belong to the logical Cursor provider (cursor).
func ExcludedFromCatalog(providerID string) bool {
	return providerID == ConnextoProxyID
}
