import { buildProviderUsageDistribution } from '../lib/usageDistribution'
import type { ProviderUsageDistributionResponse, ProviderUsageResponse } from '../types'
import { SectionFetchedAt } from './SectionFetchedAt'

type UsageDistributionProps = {
  distribution: ProviderUsageDistributionResponse | undefined
  usage: ProviderUsageResponse | undefined
  isLoading?: boolean
  isError?: boolean
  fetchedAtMs?: number
}

export function UsageDistribution({
  distribution,
  usage,
  isLoading,
  isError,
  fetchedAtMs,
}: UsageDistributionProps) {
  const slices = buildProviderUsageDistribution(distribution?.by_provider ?? [])

  return (
    <section className="resources-panel" aria-label="Distribuição de uso">
      <div className="resources-panel-head">
        <div>
          <h3 className="resources-panel-title">Distribuição de uso</h3>
          <p className="resources-panel-subtitle">Composição do consumo informado pelos provedores</p>
        </div>
      </div>

      {isLoading ? (
        <p className="resources-empty">Carregando distribuição…</p>
      ) : isError ? (
        <p className="resources-empty">Distribuição de uso indisponível para o período selecionado.</p>
      ) : slices.length === 0 ? (
        <p className="resources-empty">Nenhum uso por provedor no período selecionado.</p>
      ) : (
        <div className="resources-distribution-bars">
          {slices.map((slice, index) => (
            <div key={slice.providerId} className="resources-distribution-row">
              <label>{slice.providerId}</label>
              <div className="resources-distribution-track">
                <div
                  className="resources-distribution-fill"
                  style={{
                    width: `${Math.min(100, slice.percent)}%`,
                    opacity: Math.max(0.45, 1 - index * 0.15),
                  }}
                />
              </div>
              <strong>{Math.round(slice.percent)}%</strong>
            </div>
          ))}
        </div>
      )}

      <p className="resources-measurement-note">
        A origem e as unidades vêm de /api/resource-analytics/provider-usage/distribution.
        {usage
          ? ` Medição: ${usage.reported_executions} reportadas · ${usage.partial_executions} parciais · ${usage.unavailable_executions} indisponíveis · ${usage.invalid_executions} inválidas.`
          : null}
      </p>

      <SectionFetchedAt updatedAtMs={fetchedAtMs} />
    </section>
  )
}
