package providers

// ExcludedFromCatalog reports provider IDs that must not appear as separate catalog
// providers. Legacy transport service ids and any future transport-only aliases
// belong here; models discovered through the Connexto Proxy belong to logical
// providers such as cursor.
func ExcludedFromCatalog(providerID string) bool {
	return IsLegacyTransportServiceID(providerID)
}
