import { useMemo } from 'react'
import { Link } from 'react-router-dom'
import { Edit2 } from 'lucide-react'
import { AiBrandIcon } from '../../../shared/branding'
import { Panel } from '../../../shared/ui'
import type { ProviderQuotaSnapshot } from '../../overview/types/quota'
import { categorizeAntigravityQuotas } from '../../overview/lib/antigravityCategories'
import { goPlusMetersFromSnapshot } from '../../overview/lib/goPlusMeters'
import {
  providerConnectionSubtitle,
  providerIconSlug,
} from '../lib/providerPresentation'
import {
  availableBalanceDisplay,
  consumptionCell,
  formatQuotaResetAt,
  formatQuotaSourceName,
  formatQuotaUpdatedAt,
  franchiseSummary,
  primaryGoPercent,
  primaryGoPeriodLabel,
  providerQuotaSituation,
} from '../lib/quotaPresentation'
import type { AdminProvider } from '../types'

type ProviderDetailPanelProps = Readonly<{
  provider: AdminProvider | null
  snapshot?: ProviderQuotaSnapshot
  hasProviders?: boolean
  onOpenManage: (provider: AdminProvider) => void
}>

export function ProviderDetailPanel({
  provider,
  snapshot,
  hasProviders = true,
  onOpenManage,
}: ProviderDetailPanelProps) {
  const isAntigravity = snapshot?.provider_id === 'antigravity'
  const antigravityBars = useMemo(
    () => buildAntigravityBars(isAntigravity, snapshot),
    [isAntigravity, snapshot],
  )

  if (!provider) {
    return (
      <Panel className="providers-panel min-w-0">
        <div className="providers-detail-head shrink-0">
          <strong>Detalhe da franquia</strong>
          <small>{hasProviders ? 'SELECIONE UM PROVEDOR' : 'SEM PROVEDORES'}</small>
        </div>
        <p className="py-6 text-center text-xs text-muted">
          {hasProviders
            ? 'Selecione um provedor na tabela para inspecionar a conexão.'
            : 'Nenhum provedor configurado para inspecionar.'}
        </p>
      </Panel>
    )
  }

  const situation = providerQuotaSituation(provider, snapshot)
  const updatedAt = formatQuotaUpdatedAt(snapshot, provider.updated_at)
  const goPercent = primaryGoPercent(snapshot)
  const goPeriod = primaryGoPeriodLabel(snapshot)
  const consumption = consumptionCell(snapshot)
  const franchise = franchiseSummary(snapshot)
  const balance = availableBalanceDisplay(snapshot)

  const percent = goPercent ?? (consumption.kind === 'percent' ? consumption.percent : undefined)
  const percentPeriod =
    goPercent != null
      ? (goPeriod ?? 'provedor')
      : consumption.kind === 'percent'
        ? consumption.label
        : undefined

  const mainQuotaValue =
    percent != null
      ? `${percent.toLocaleString('pt-BR')}%`
      : consumption.kind === 'usage'
        ? consumption.primary
        : 'Sem leitura'

  const mainQuotaSuffix =
    isAntigravity
      ? 'restante'
      : percent != null || consumption.kind === 'percent'
        ? 'utilizado'
        : 'utilizado'

  const primaryReading =
    percent != null
      ? `${percent.toLocaleString('pt-BR')}% (${percentPeriod})`
      : consumption.kind === 'usage'
        ? consumption.primary
        : '—'

  const antigravityFamilies =
    isAntigravity && snapshot?.status === 'available'
      ? categorizeAntigravityQuotas(snapshot.quotas).filter((f) => f.id !== 'autocomplete_background')
      : []

  const goMeters = snapshot?.status === 'available' ? goPlusMetersFromSnapshot(snapshot) : []

  return (
    <Panel className="providers-panel min-w-0">
      <div className="providers-detail-head shrink-0">
        <strong>Detalhe da franquia</strong>
        <small>INSPEÇÃO DE GOVERNANÇA</small>
      </div>

      <div className="providers-detail-body">
        <div className="providers-provider-name mt-2 border-b border-line/70 pb-2">
          <AiBrandIcon
            slug={providerIconSlug(provider)}
            label={provider.name}
            size={32}
            className="providers-provider-icon size-8"
          />
          <div>
            <strong className="block text-xs font-semibold text-[#E7ECF5]">{provider.name}</strong>
            <small className="mt-0.5 block text-[9px] text-muted">
              {providerConnectionSubtitle(provider)}
              {snapshot?.workspace_name ? ` · ${snapshot.workspace_name}` : ''}
            </small>
          </div>
        </div>

        <div className="providers-quota-box" data-testid={`providers-detail-quota-${provider.id}`}>
          <div className="providers-quota-box-label">CONSUMO DA FRANQUIA</div>
          <div className="providers-quota-box-main">
            <strong>{mainQuotaValue}</strong>
            <span>{mainQuotaSuffix}</span>
          </div>

          {isAntigravity && antigravityBars.length > 0 ? (
            <div className="space-y-2.5 my-2.5" data-testid="antigravity-quota-bars">
              {antigravityBars.map((bar) => (
                <div key={bar.id} className="space-y-1">
                  <div className="flex items-center justify-between text-[11px]">
                    <span className="text-[#AEBBD0] font-medium">{bar.label}</span>
                    <span className="font-semibold text-[#E7ECF5]">
                      {bar.percent.toLocaleString('pt-BR')}% restante
                    </span>
                  </div>
                  <div className="providers-large-track">
                    <div
                      className="providers-large-fill"
                      style={{ width: `${Math.min(Math.max(bar.percent, 0), 100)}%` }}
                    />
                  </div>
                </div>
              ))}
            </div>
          ) : percent != null ? (
            <div className="providers-large-track">
              <div
                className="providers-large-fill"
                style={{ width: `${Math.min(percent, 100)}%` }}
              />
            </div>
          ) : consumption.kind === 'usage' &&
            consumption.barValue != null &&
            consumption.barMax != null &&
            consumption.barMax > 0 ? (
            <div className="providers-large-track">
              <div
                className="providers-large-fill"
                style={{
                  width: `${Math.min(100, Math.max(0, (consumption.barValue / consumption.barMax) * 100))}%`,
                }}
              />
            </div>
          ) : (
            <div className="providers-large-track" aria-hidden />
          )}
          <div className="providers-quota-foot">
            <span title={franchise.title}>
              <span className="text-[#8A97A8]">Consumo: </span>
              {franchise.value !== '—'
                ? franchise.value
                : percent != null
                  ? `${percent.toLocaleString('pt-BR')}% (${percentPeriod})`
                  : 'não reportado'}
            </span>
            <span>
              <span className="text-[#8A97A8]">Limite: </span>
              {isAntigravity
                ? 'pools oficiais'
                : percent != null
                  ? 'percentual do provedor'
                  : balance !== '—'
                    ? `disponível ${balance}`
                    : 'leitura oficial'}
            </span>
          </div>
        </div>

        {antigravityFamilies.length > 0 ? (
          <div
            className="mt-2 rounded-lg border border-line/60 bg-[#171E29]/90 p-2 space-y-1.5"
            data-testid="antigravity-detail-families"
          >
            <div className="flex items-center justify-between text-[9px] uppercase tracking-wider font-semibold text-[#8290A5]">
              <span>Franquias por Família</span>
              <span className="font-mono text-[8px] text-muted">
                {snapshot?.quotas.length ?? 0} modelos
              </span>
            </div>
            {antigravityFamilies.map((family) => (
              <div
                key={family.id}
                className="space-y-1 rounded bg-[#0f1722]/50 p-1.5 border border-line/30"
              >
                <div className="flex items-center justify-between text-[10px] font-medium text-[#CBD5E2]">
                  <span>{family.title}</span>
                  <span className="text-[9px] text-[#8290A5]">
                    {family.totalModelsCount} modelo(s)
                  </span>
                </div>
                {family.limits.map((limit, idx) => (
                  <div
                    key={`${limit.windowType}-${idx}`}
                    className="flex items-center justify-between text-[9px]"
                  >
                    <span className="text-[#AEBBD0]">{limit.windowLabel}</span>
                    <span className="font-semibold text-[#CBD5E2]">
                      {limit.remainingPercent.toLocaleString('pt-BR')}% restante
                    </span>
                  </div>
                ))}
              </div>
            ))}
          </div>
        ) : null}

        {goMeters.length > 0 ? (
          <div
            className="mt-2 rounded-lg border border-line/60 bg-[#171E29]/90 p-2 space-y-1.5"
            data-testid="opencode-detail-meters"
          >
            <div className="flex items-center justify-between text-[9px] uppercase tracking-wider font-semibold text-[#8290A5]">
              <span>Janelas de franquia</span>
              <span className="font-mono text-[8px] text-muted">
                {goMeters.length} métrica(s)
              </span>
            </div>
            {goMeters.map((meter) => {
              const label =
                meter.id === 'go_monthly'
                  ? 'Mensal'
                  : meter.id === 'go_weekly'
                    ? 'Semanal'
                    : 'Janela 5h'
              return (
                <div
                  key={meter.id}
                  className="flex items-center justify-between text-[9px] rounded bg-[#0f1722]/50 p-1.5 border border-line/30"
                >
                  <span className="text-[#AEBBD0]">{label}</span>
                  <span className="font-semibold text-[#CBD5E2]">
                    {meter.percent.toLocaleString('pt-BR')}% utilizado
                  </span>
                </div>
              )
            })}
          </div>
        ) : null}

        <div className="mt-1">
          <div className="providers-detail-row">
            <span>Leitura principal</span>
            <strong>{primaryReading}</strong>
          </div>
          <div className="providers-detail-row">
            <span>Renovação / reset</span>
            <strong>{formatQuotaResetAt(snapshot)}</strong>
          </div>
          <div className="providers-detail-row">
            <span>Fonte da medição</span>
            <strong>{formatQuotaSourceName(snapshot)}</strong>
          </div>
          <div className="providers-detail-row">
            <span>Última atualização</span>
            <strong>{updatedAt}</strong>
          </div>
          <div className="providers-detail-row border-b-0 items-center">
            <span>Situação</span>
            <strong className="flex justify-start">
              <span className={`providers-state ${situation.state}`}>
                <i aria-hidden />
                {situation.label}
              </span>
            </strong>
          </div>
        </div>

        <p className="providers-source-note">
          Percentuais e saldos só aparecem com leitura confiável do gateway ou do provedor.
        </p>

        <div className="providers-detail-actions flex flex-col gap-2 pt-1">
          <button
            type="button"
            onClick={() => onOpenManage(provider)}
            className="flex items-center justify-center gap-2 w-full rounded-md bg-cobalt px-3.5 py-2 text-xs font-semibold text-white shadow-sm hover:bg-[#345ED0] transition-colors cursor-pointer"
            aria-label={`Editar credenciais e conexão de ${provider.name}`}
          >
            <Edit2 className="size-3.5" />
            Editar credenciais e conexão
          </button>

          <div className="flex items-center gap-2 w-full">
            <Link to="/resources" className="providers-detail-secondary flex-1 text-center">
              Ver políticas ↗
            </Link>
            <Link to="/catalog" className="providers-detail-ghost flex-1 text-center">
              Explorar modelos ↗
            </Link>
          </div>
        </div>
      </div>
    </Panel>
  )
}

type AntigravityQuotaBar = Readonly<{
  id: string
  label: string
  percent: number
}>

function buildAntigravityBars(
  isAntigravity: boolean,
  snapshot: ProviderQuotaSnapshot | undefined,
): AntigravityQuotaBar[] {
  if (!isAntigravity || snapshot?.status !== 'available') return []
  const families = categorizeAntigravityQuotas(snapshot.quotas).filter(
    (family) => family.id !== 'autocomplete_background',
  )
  return fillAntigravityBars(primaryAntigravityBars(families), families)
}

function primaryAntigravityBars(
  families: ReturnType<typeof categorizeAntigravityQuotas>,
): AntigravityQuotaBar[] {
  const bars: AntigravityQuotaBar[] = []
  const gemini = families.find((family) => family.id === 'gemini_models')
  const claudeGpt = families.find((family) => family.id === 'claude_gpt_models')
  const weeklyGemini = gemini?.limits.find((limit) => limit.windowType === 'weekly')
  const fiveHourGemini = gemini?.limits.find((limit) => limit.windowType === '5h')
  const weeklyClaudeGpt =
    claudeGpt?.limits.find((limit) => limit.windowType === 'weekly') ?? claudeGpt?.limits[0]

  if (weeklyGemini) {
    bars.push({ id: 'gemini-weekly', label: 'Gemini (Limite semanal)', percent: weeklyGemini.remainingPercent })
  }
  if (fiveHourGemini) {
    bars.push({ id: 'gemini-5h', label: 'Gemini (Limite 5h)', percent: fiveHourGemini.remainingPercent })
  }
  if (weeklyClaudeGpt) {
    bars.push({
      id: 'claude-gpt-weekly',
      label: `Claude/GPT (${weeklyClaudeGpt.windowLabel})`,
      percent: weeklyClaudeGpt.remainingPercent,
    })
  }
  return bars
}

function fillAntigravityBars(
  initialBars: AntigravityQuotaBar[],
  families: ReturnType<typeof categorizeAntigravityQuotas>,
): AntigravityQuotaBar[] {
  const bars = [...initialBars]
  for (const family of families) {
    for (const limit of family.limits) {
      const id = `${family.id}-${limit.windowType}`
      const alreadyIncluded = bars.some(
        (bar) => bar.id === id || bar.label.includes(limit.windowLabel),
      )
      if (alreadyIncluded) continue
      bars.push({
        id,
        label: `${antigravityFamilyShortTitle(family.id, family.title)} (${limit.windowLabel})`,
        percent: limit.remainingPercent,
      })
      if (bars.length >= 3) return bars
    }
  }
  return bars
}

function antigravityFamilyShortTitle(id: string, title: string): string {
  if (id === 'claude_gpt_models') return 'Claude/GPT'
  if (id === 'gemini_models') return 'Gemini'
  return title
}
