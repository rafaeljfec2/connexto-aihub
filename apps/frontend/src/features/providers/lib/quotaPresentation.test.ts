import { describe, expect, it } from 'vitest'
import type { ProviderQuotaMetric, ProviderQuotaSnapshot, ProviderQuotaStatus } from '../../overview/types/quota'
import type { AdminProvider } from '../types'
import {
  availableBalanceDisplay,
  consumptionCell,
  countGovernanceMetrics,
  formatQuotaResetAt,
  formatQuotaSourceName,
  franchiseSummary,
  primaryQuotaSnapshot,
  providerQuotaSituation,
} from './quotaPresentation'

const provider: AdminProvider = {
  id: 'openrouter-conectado',
  name: 'OpenRouter',
  type: 'openrouter',
  endpoint: 'https://openrouter.ai',
  enabled: true,
  credential_configured: true,
  created_at: '',
  updated_at: '',
}

function snapshot(
  scope: 'api_key' | 'account' | 'subscription',
  quotas: ProviderQuotaMetric[],
  status: ProviderQuotaStatus = 'available',
  accountId = provider.id,
): ProviderQuotaSnapshot {
  const providerId = scope === 'subscription' ? 'opencode-go' : 'openrouter'
  const sourceName =
    scope === 'api_key'
      ? 'openrouter_key_api'
      : scope === 'account'
        ? 'openrouter_credits_api'
        : 'opencode_go_usage_api'
  return {
    account_id: accountId,
    account_label: scope === 'subscription' ? 'OpenCode Go' : 'OpenRouter',
    provider_id: providerId,
    provider_name: scope === 'subscription' ? 'OpenCode Go' : 'OpenRouter',
    status,
    scope,
    quotas,
    workspace_name: scope === 'subscription' ? 'go-plus' : undefined,
    source: {
      type: 'provider_api',
      name: sourceName,
      scope,
      fetched_at: '2026-11-04T17:34:36Z',
    },
  }
}

describe('primaryQuotaSnapshot', () => {
  it('prefers subscription then account then api_key', () => {
    const account = snapshot('account', [
      { metric: 'total_usage', label: 'Total account usage', nature: 'cost', used: 4.84, unit: 'USD' },
    ])
    const subscription = snapshot('subscription', [
      { metric: 'go_rolling', label: 'Go rolling', nature: 'quota', consumed_percent: 12 },
    ])
    expect(primaryQuotaSnapshot(provider.id, [account, subscription])?.scope).toBe('subscription')
    expect(primaryQuotaSnapshot(provider.id, [account])?.scope).toBe('account')
  })
})

describe('providerQuotaSituation', () => {
  it('keeps connection-level disabled/pending before quota', () => {
    expect(providerQuotaSituation({ ...provider, enabled: false }, undefined)).toEqual({
      state: 'off',
      label: 'Desabilitado',
    })
    expect(
      providerQuotaSituation({ ...provider, credential_configured: false }, undefined),
    ).toEqual({ state: 'warn', label: 'Pendente' })
  })

  it('maps available quota status to Disponível', () => {
    const go = snapshot('subscription', [
      { metric: 'go_rolling', label: 'Go rolling', nature: 'quota', consumed_percent: 12 },
    ])
    expect(providerQuotaSituation(provider, go)).toEqual({ state: 'ok', label: 'Disponível' })
  })

  it('maps missing snapshot to Sem leitura', () => {
    expect(providerQuotaSituation(provider, undefined)).toEqual({
      state: 'na',
      label: 'Sem leitura',
    })
  })

  it('treats Antigravity gateway-only as configured and maps quota unsupported to neutral label', () => {
    const agProvider: AdminProvider = {
      id: 'antigravity',
      name: 'Antigravity',
      type: 'antigravity',
      endpoint: 'http://127.0.0.1:9090/v1',
      enabled: true,
      credential_configured: false,
      management_credential_configured: true,
      created_at: '',
      updated_at: '',
    }
    const unsupportedAg: ProviderQuotaSnapshot = {
      account_id: 'antigravity',
      account_label: 'Antigravity',
      provider_id: 'antigravity',
      provider_name: 'Antigravity',
      status: 'unsupported',
      scope: 'unknown',
      quotas: [],
      source: {
        type: 'provider_api',
        name: 'antigravity',
        scope: 'unknown',
        fetched_at: '2026-11-06T12:00:00Z',
      },
      error: { code: 'quota_not_applicable', message: 'N/A' },
    }
    expect(providerQuotaSituation(agProvider, unsupportedAg)).toEqual({
      state: 'na',
      label: 'Sem leitura de franquia',
    })
  })
})

describe('franchiseSummary and consumptionCell', () => {
  it('labels Antigravity quota-not-applicable without misleading credential copy', () => {
    const snap: ProviderQuotaSnapshot = {
      account_id: 'antigravity',
      account_label: 'Antigravity',
      provider_id: 'antigravity',
      provider_name: 'Antigravity',
      status: 'unsupported',
      scope: 'unknown',
      quotas: [],
      source: {
        type: 'provider_api',
        name: 'antigravity',
        scope: 'unknown',
        fetched_at: '2026-11-06T12:00:00Z',
      },
      error: {
        code: 'quota_not_applicable',
        message: 'Antigravity não expõe leitura oficial de franquia neste hub (apenas uso via proxy)',
      },
    }
    expect(franchiseSummary(snap).title).toBe('Franquia não integrada')
    expect(franchiseSummary(snap).value).toContain('proxy')
  })

  it('shortens Cursor unauthorized franchise copy for the detail panel', () => {
    const snap: ProviderQuotaSnapshot = {
      account_id: 'cursor',
      account_label: 'Cursor',
      provider_id: 'cursor',
      provider_name: 'Cursor',
      status: 'unauthorized',
      scope: 'account',
      quotas: [],
      source: {
        type: 'provider_api',
        name: 'cursor_proxy_usage_collect',
        scope: 'account',
        fetched_at: '2026-11-06T22:24:00Z',
      },
      error: {
        code: 'cursor_auth_required',
        message: 'Sessão Cursor no proxy requer autenticação assistida',
      },
    }
    expect(franchiseSummary(snap).value).toBe('Login web pendente no proxy')
    expect(formatQuotaSourceName(snap)).toBe('Cursor Proxy Usage Collector')
  })

  it('formats Antigravity available quotas properly', () => {
    const agProvider: AdminProvider = {
      id: 'antigravity',
      name: 'Antigravity',
      type: 'antigravity',
      endpoint: 'http://127.0.0.1:9090/v1',
      enabled: true,
      credential_configured: false,
      management_credential_configured: true,
      created_at: '',
      updated_at: '',
    }
    const availableAg: ProviderQuotaSnapshot = {
      account_id: 'antigravity',
      account_label: 'engineer@connexto.ai',
      provider_id: 'antigravity',
      provider_name: 'Antigravity',
      status: 'available',
      scope: 'account',
      workspace_name: 'Google One AI Premium',
      quotas: [
        {
          metric: 'gemini-2.5-pro',
          label: 'Gemini 2.5 Pro',
          nature: 'rate_limit',
          remaining_fraction: 0.85,
          remaining_percent: 85,
          consumed_percent: 15,
          resets_at: '2026-11-06T20:00:00Z',
        },
      ],
      source: {
        type: 'provider_api',
        name: 'antigravity_proxy_usage_api',
        scope: 'account',
        fetched_at: '2026-11-06T19:30:00Z',
      },
    }
    expect(providerQuotaSituation(agProvider, availableAg)).toEqual({
      state: 'ok',
      label: 'Disponível',
    })
    expect(franchiseSummary(availableAg)).toEqual({
      title: 'Antigravity (Google One AI Premium)',
      value: 'Gemini 85%',
    })
    expect(consumptionCell(availableAg)).toEqual({
      kind: 'percent',
      label: 'Gemini (Limite 5h)',
      percent: 85,
      barValue: 85,
      barMax: 100,
    })
  })

  it('prefers Go weekly percent to match Console chip (not rolling)', () => {
    const go = snapshot('subscription', [
      {
        metric: 'go_rolling',
        label: 'Go rolling',
        nature: 'quota',
        consumed_percent: 12,
        resets_at: '2026-11-04T18:31:46.000Z',
      },
      {
        metric: 'go_weekly',
        label: 'Go weekly',
        nature: 'quota',
        consumed_percent: 36,
        resets_at: '2026-11-09T00:00:00.000Z',
      },
      {
        metric: 'go_monthly',
        label: 'Go monthly',
        nature: 'quota',
        consumed_percent: 24,
        resets_at: '2026-12-01T19:00:43.000Z',
      },
    ])
    expect(franchiseSummary(go)).toEqual({
      title: 'Franquia Go Plus (semanal)',
      value: '36% reportado pelo provedor',
    })
    expect(consumptionCell(go)).toEqual({
      kind: 'percent',
      label: 'Percentual semanal',
      percent: 36,
      barValue: 36,
      barMax: 100,
    })
    expect(formatQuotaResetAt(go)).not.toContain('T')
  })

  it('falls back to rolling when weekly is absent', () => {
    const go = snapshot('subscription', [
      {
        metric: 'go_rolling',
        label: 'Go rolling',
        nature: 'quota',
        consumed_percent: 12,
        resets_at: '2026-11-04T18:31:46.000Z',
      },
    ])
    expect(franchiseSummary(go).value).toBe('12% reportado pelo provedor')
    expect(consumptionCell(go)).toMatchObject({ kind: 'percent', percent: 12 })
  })

  it('formats OpenRouter account usage without inventing a percent bar', () => {
    const account = snapshot('account', [
      { metric: 'total_usage', label: 'Total account usage', nature: 'cost', used: 4.84186443, unit: 'USD' },
      { metric: 'total_credits', label: 'Total account credits', nature: 'quota', limit: 10, unit: 'USD' },
    ])
    const franchise = franchiseSummary(account)
    expect(franchise.title).toBe('Leitura oficial')
    expect(franchise.value).toContain('US$ 4,84')
    expect(franchise.value).not.toContain('4.84186443')
    expect(franchise.value).toContain('US$ 10,00')

    const cell = consumptionCell(account)
    expect(cell.kind).toBe('usage')
    if (cell.kind === 'usage') {
      expect(cell.primary).toBe('US$ 4,84 / US$ 10,00')
      expect(cell.secondary).toBe('Uso / limite oficiais')
      expect(cell.barValue).toBe(4.84186443)
      expect(cell.barMax).toBe(10)
    }
    expect(availableBalanceDisplay(account)).toBe('US$ 5,16')
  })
})

describe('countGovernanceMetrics', () => {
  it('counts providers without a displayable reading', () => {
    const goProvider: AdminProvider = {
      ...provider,
      id: 'opencode-go-conectado',
      name: 'OpenCode Go Primary',
      type: 'opencode-go',
    }
    const go = snapshot(
      'subscription',
      [{ metric: 'go_rolling', label: 'Go rolling', nature: 'quota', consumed_percent: 12 }],
      'available',
      goProvider.id,
    )
    const openrouterUnavailable = snapshot('account', [], 'unsupported')

    const metrics = countGovernanceMetrics([goProvider, provider], [go, openrouterUnavailable])
    expect(metrics.monitored).toBe(2)
    expect(metrics.withoutReading).toBe(1)
    expect(metrics.nearLimit).toBe(0)
    expect(metrics.atLimit).toBe(0)
  })

  it('counts provider-reported alert scopes without Hub-side 80% thresholds', () => {
    const goProvider: AdminProvider = {
      ...provider,
      id: 'opencode-go-conectado',
      name: 'OpenCode Go Primary',
      type: 'opencode-go',
    }
    const near = snapshot(
      'subscription',
      [
        {
          metric: 'go_rolling',
          label: 'Go rolling',
          nature: 'quota',
          consumed_percent: 90,
          scope: 'status:warning',
        },
      ],
      'available',
      goProvider.id,
    )
    expect(countGovernanceMetrics([goProvider], [near]).nearLimit).toBe(1)

    const limited = snapshot(
      'subscription',
      [
        {
          metric: 'go_rolling',
          label: 'Go rolling',
          nature: 'quota',
          consumed_percent: 100,
          scope: 'status:limited',
        },
      ],
      'available',
      goProvider.id,
    )
    expect(countGovernanceMetrics([goProvider], [limited]).atLimit).toBe(1)
  })
})

describe('franchiseSummary — Cursor', () => {
  it('shows weekly USD franchise when metrics exist', () => {
    const cursorSnapshot: ProviderQuotaSnapshot = {
      account_id: 'cursor-1',
      account_label: 'Cursor',
      provider_id: 'cursor',
      provider_name: 'Cursor',
      status: 'available',
      scope: 'account',
      quotas: [
        { metric: 'total_usage', label: 'Total account usage', nature: 'cost', used: 41.68, unit: 'USD' },
        { metric: 'total_credits', label: 'Weekly included budget', nature: 'quota', limit: 46.15, unit: 'USD' },
      ],
      source: {
        type: 'provider_api',
        name: 'cursor_proxy_usage_collect',
        scope: 'account',
        fetched_at: '2026-11-06T20:00:00Z',
      },
    }
    expect(franchiseSummary(cursorSnapshot)).toEqual({
      title: 'Franquia semanal Cursor',
      value: 'US$ 41,68 consumido · Limite US$ 46,15',
    })
  })
})
