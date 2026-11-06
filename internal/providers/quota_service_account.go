package providers

import (
	"context"
	"log/slog"
	"time"
)

func (s *QuotaService) queryConfiguredAccount(ctx context.Context, record Record) []ProviderQuotaSnapshot {
	if !record.Enabled {
		return []ProviderQuotaSnapshot{unavailableSnapshot(record, ScopeUnknown, record.Type, QuotaStatusUnsupported,
			"account_disabled", "Conta desabilitada nas configurações")}
	}
	if record.Type == TypeAntigravity {
		adapter, supported := s.adapters[record.Type]
		if !supported {
			return []ProviderQuotaSnapshot{unavailableSnapshot(record, ScopeUnknown, record.Type, QuotaStatusUnsupported,
				"quota_not_applicable", "Antigravity não expõe leitura oficial de franquia neste hub (apenas uso via proxy)")}
		}
		return []ProviderQuotaSnapshot{s.queryAntigravityAccount(ctx, record, adapter)}
	}
	adapter, supported := s.adapters[record.Type]
	if !supported {
		if !record.HasCredential {
			return []ProviderQuotaSnapshot{unavailableSnapshot(record, ScopeUnknown, record.Type,
				QuotaStatusUnconfiguredCredential, "unconfigured_credential", "Conta sem credencial configurada")}
		}
		return []ProviderQuotaSnapshot{unavailableSnapshot(record, ScopeUnknown, record.Type, QuotaStatusUnsupported,
			"unsupported", "Provedor não disponibiliza leitura oficial para esta conta")}
	}

	if record.Type == TypeOpenCodeGo {
		return []ProviderQuotaSnapshot{s.queryOpenCodeSubscription(ctx, record, adapter)}
	}

	snapshots := []ProviderQuotaSnapshot{s.queryKeyScope(ctx, record, adapter)}
	if accountAdapter, ok := adapter.(AccountQuotaAdapter); ok {
		snapshots = append(snapshots, s.queryAccountScope(ctx, record, accountAdapter))
	}
	return snapshots
}

func (s *QuotaService) queryAntigravityAccount(ctx context.Context, record Record, adapter QuotaAdapter) ProviderQuotaSnapshot {
	managementKey, err := s.adminService.managementCredentialOf(ctx, record)
	if err != nil {
		s.logCredentialFailure(record, "management", err)
		return unavailableSnapshot(record, ScopeAccount, antigravityQuotaSourceName, QuotaStatusError,
			"credential_decrypt_failed", "Falha ao recuperar a Gateway API Key")
	}
	gatewayKey, ok := EffectiveGatewayKeyForAntigravityValidation(managementKey)
	if !ok {
		return unavailableSnapshot(record, ScopeAccount, antigravityQuotaSourceName, QuotaStatusUnconfiguredCredential,
			"unconfigured_credential", "Gateway API Key não configurada para comunicação com o proxy")
	}
	return s.queryWithCredential(ctx, record, adapter.FetchQuota, gatewayKey, ScopeAccount, antigravityQuotaSourceName)
}

func (s *QuotaService) queryOpenCodeSubscription(ctx context.Context, record Record, adapter QuotaAdapter) ProviderQuotaSnapshot {
	credential, err := s.adminService.credentialOf(ctx, record)
	if err != nil {
		s.logCredentialFailure(record, "inference", err)
		return unavailableSnapshot(record, ScopeSubscription, openCodeQuotaSourceName, QuotaStatusError,
			"credential_decrypt_failed", "Falha ao recuperar credencial da chave API")
	}
	return s.queryWithCredential(ctx, record, adapter.FetchQuota, credential, ScopeSubscription, openCodeQuotaSourceName)
}

func (s *QuotaService) queryKeyScope(ctx context.Context, record Record, adapter QuotaAdapter) ProviderQuotaSnapshot {
	credential, err := s.adminService.credentialOf(ctx, record)
	if err != nil {
		s.logCredentialFailure(record, "inference", err)
		return unavailableSnapshot(record, ScopeAPIKey, "openrouter_key_api", QuotaStatusError,
			"credential_decrypt_failed", "Falha ao recuperar credencial da chave API")
	}
	return s.queryWithCredential(ctx, record, adapter.FetchQuota, credential, ScopeAPIKey, "openrouter_key_api")
}

func (s *QuotaService) queryAccountScope(ctx context.Context, record Record, adapter AccountQuotaAdapter) ProviderQuotaSnapshot {
	credential, err := s.adminService.managementCredentialOf(ctx, record)
	if err != nil {
		s.logCredentialFailure(record, "management", err)
		return unavailableSnapshot(record, ScopeAccount, "openrouter_credits_api", QuotaStatusError,
			"management_credential_decrypt_failed", "Falha ao recuperar a Management API Key")
	}
	return s.queryWithCredential(ctx, record, adapter.FetchAccountQuota, credential, ScopeAccount, "openrouter_credits_api")
}

type quotaFetch func(context.Context, string, string) (ProviderQuotaSnapshot, error)

func (s *QuotaService) queryWithCredential(ctx context.Context, record Record, fetch quotaFetch,
	credential string, scope DataScope, source string) ProviderQuotaSnapshot {
	subCtx, cancel := context.WithTimeout(ctx, accountQueryTimeout)
	defer cancel()
	snapshot, err := fetch(subCtx, record.Endpoint, credential)
	if err != nil {
		s.logger.Error("provider quota query failed", slog.String("account_id", record.ID),
			slog.String("provider_type", record.Type), slog.String("scope", string(scope)))
		return unavailableSnapshot(record, scope, source, QuotaStatusError,
			"adapter_error", "Falha ao consultar a leitura oficial do provedor")
	}
	snapshot.AccountID = record.ID
	if snapshot.AccountLabel == "" {
		snapshot.AccountLabel = record.Name
	}
	snapshot.ProviderID = record.Type
	if snapshot.ProviderName == "" {
		snapshot.ProviderName = record.Name
	}
	snapshot.Scope = scope
	snapshot.Source.Scope = scope
	snapshot.Source.Name = source
	return snapshot
}

func (s *QuotaService) logCredentialFailure(record Record, kind string, err error) {
	s.logger.Error("failed to decrypt provider quota credential",
		slog.String("account_id", record.ID), slog.String("provider_type", record.Type),
		slog.String("credential_kind", kind), slog.String("error", err.Error()))
}

func unavailableSnapshot(record Record, scope DataScope, source string, status ProviderQuotaStatus,
	code, message string) ProviderQuotaSnapshot {
	return ProviderQuotaSnapshot{
		AccountID: record.ID, AccountLabel: record.Name, ProviderID: record.Type,
		ProviderName: record.Name, Status: status, Scope: scope,
		Quotas: []ProviderQuotaMetric{},
		Source: ProviderQuotaSource{Type: "provider_api", Name: source, Scope: scope,
			FetchedAt: time.Now().UTC().Format(time.RFC3339)},
		Error: &ProviderQuotaError{Code: code, Message: message},
	}
}
