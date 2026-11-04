import { Activity, CheckCircle2, Layers, Plug } from 'lucide-react'
import { Link } from 'react-router-dom'
import { StatCard } from '../../../shared/ui'
import type { ProviderDiagnostic } from '../api/queries'

type OverviewStatCardsProps = {
  isOperational: boolean
  readinessChecks?: Record<string, string>
  providers?: ProviderDiagnostic[]
  modelCount?: number
  executionCount?: number
  isLoading?: boolean
}

export function OverviewStatCards({
  isOperational,
  providers = [],
  modelCount,
  executionCount,
  isLoading,
}: OverviewStatCardsProps) {
  const degradedCount = providers.filter(
    (p) => p.state === 'degraded' || p.state === 'unreachable',
  ).length
  const totalProviders = providers.length

  return (
    <div className="grid grid-cols-1 gap-3 sm:grid-cols-2 lg:grid-cols-4">
      <StatCard
        label="Estado do gateway"
        icon={<CheckCircle2 className="size-3.5" />}
        value={
          isLoading ? (
            <span className="text-base text-muted animate-pulse">Carregando…</span>
          ) : isOperational ? (
            <span className="text-[22px] tracking-[-0.02em] text-[#88B995]">Operacional</span>
          ) : (
            <span className="text-[22px] tracking-[-0.02em] text-[#D88186]">Degradado</span>
          )
        }
        note={
          <div className="flex items-center gap-1.5">
            <span
              className={`size-1.5 rounded-full ${
                isOperational ? 'bg-healthy' : 'bg-failure'
              }`}
            />
            <span>
              {isOperational
                ? 'Pronto para receber requisições'
                : 'Verifique a conectividade dos serviços'}
            </span>
          </div>
        }
      />

      <StatCard
        label="Franquias em atenção"
        icon={<Plug className="size-3.5" />}
        value={
          isLoading ? (
            <span className="text-base text-muted animate-pulse">Carregando…</span>
          ) : (
            <div className="flex items-baseline gap-1">
              <span>{String(degradedCount).padStart(2, '0')}</span>
              <span className="text-sm font-normal text-muted">
                / {String(totalProviders).padStart(2, '0')}
              </span>
            </div>
          )
        }
        note={
          degradedCount > 0
            ? `${degradedCount} conexões requerem verificação`
            : 'Conexões monitoradas em estado normal'
        }
      />

      <StatCard
        label="Modelos disponíveis"
        icon={<Layers className="size-3.5" />}
        value={
          isLoading ? (
            <span className="text-base text-muted animate-pulse">Carregando…</span>
          ) : (
            <span>{modelCount !== undefined ? modelCount : '—'}</span>
          )
        }
        note={
          <Link
            to="/catalog"
            className="inline-flex items-center gap-1 text-[10px] text-accent-light hover:text-primary hover:underline"
          >
            Explorar catálogo ↗
          </Link>
        }
      />

      <StatCard
        label="Requisições registradas"
        icon={<Activity className="size-3.5" />}
        value={
          isLoading ? (
            <span className="text-base text-muted animate-pulse">Carregando…</span>
          ) : executionCount !== undefined ? (
            <span>{executionCount}</span>
          ) : (
            <span className="text-sm font-normal text-muted">Não instrumentado</span>
          )
        }
        note="Amostra de chamadas persistidas"
      />
    </div>
  )
}
