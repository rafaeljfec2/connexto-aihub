package gateway

import (
	"time"

	"aihub/internal/providers"
)

type providerResponse struct {
	ID                             string    `json:"id"`
	Name                           string    `json:"name"`
	Type                           string    `json:"type"`
	Endpoint                       string    `json:"endpoint"`
	Enabled                        bool      `json:"enabled"`
	CredentialConfigured           bool      `json:"credential_configured"`
	ManagementCredentialConfigured bool      `json:"management_credential_configured"`
	CreatedAt                      time.Time `json:"created_at"`
	UpdatedAt                      time.Time `json:"updated_at"`
}

type createProviderRequest struct {
	ID                   string `json:"id"`
	ProviderID           string `json:"provider_id"`
	Name                 string `json:"name"`
	Type                 string `json:"type"`
	Endpoint             string `json:"endpoint"`
	Credential           string `json:"credential"`
	ManagementCredential string `json:"management_credential"`
	Enabled              *bool  `json:"enabled"`
}

type validateProviderRequest struct {
	Credential           string `json:"credential"`
	ManagementCredential string `json:"management_credential"`
}

type updateProviderRequest struct {
	Name                 *string `json:"name"`
	Endpoint             *string `json:"endpoint"`
	Credential           *string `json:"credential"`
	ManagementCredential *string `json:"management_credential"`
}

type antigravityProxyAuthStartRequest struct {
	RedirectURI *string `json:"redirect_uri"`
}

func toProviderResponse(record providers.Record) providerResponse {
	return providerResponse{
		ID:                             record.ID,
		Name:                           record.Name,
		Type:                           record.Type,
		Endpoint:                       record.Endpoint,
		Enabled:                        record.Enabled,
		CredentialConfigured:           record.HasCredential,
		ManagementCredentialConfigured: record.HasManagementCredential,
		CreatedAt:                      record.CreatedAt,
		UpdatedAt:                      record.UpdatedAt,
	}
}
