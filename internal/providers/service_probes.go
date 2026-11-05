package providers

import "strings"

// NewServiceWithProbes binds a connectivity probe to each supported adapter type.
func NewServiceWithProbes(repo Repository, cipher *CredentialCipher, registry *Registry, probes map[string]ProbeFunc, allowedHosts []string) *Service {
	hosts := make(map[string]struct{}, len(allowedHosts))
	for _, host := range allowedHosts {
		hosts[strings.ToLower(host)] = struct{}{}
	}
	probeCopy := make(map[string]ProbeFunc, len(probes))
	for providerType, probe := range probes {
		probeCopy[providerType] = probe
	}
	return &Service{
		repo:             repo,
		cipher:           cipher,
		registry:         registry,
		probes:           probeCopy,
		allowedHosts:     hosts,
		defaultEndpoints: make(map[string]string),
	}
}
