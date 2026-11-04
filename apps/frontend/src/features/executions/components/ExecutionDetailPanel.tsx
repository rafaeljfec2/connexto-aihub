import { Link } from 'react-router-dom'
import { Panel, PanelHeader } from '../../../shared/ui'
import type { ExecutionRecord } from '../types'

type ExecutionDetailPanelProps = {
  record?: ExecutionRecord
  isLoading: boolean
  isError: boolean
  errorMessage?: string
  selectedId: string | null
}

export function ExecutionDetailPanel({
  record,
  isLoading,
  isError,
  errorMessage,
  selectedId,
}: ExecutionDetailPanelProps) {
  if (isError) {
    return (
      <Panel>
        <PanelHeader title="Detalhes da execução" subtitle="ERRO DE CARREGAMENTO" />
        <p className="py-4 text-xs text-failure" role="alert">
          {errorMessage ?? 'Falha ao carregar os detalhes da execução.'}
        </p>
      </Panel>
    )
  }

  if (isLoading) {
    return (
      <Panel>
        <PanelHeader title="Detalhes da execução" subtitle="CARREGANDO…" />
        <p className="py-8 text-center text-xs text-muted animate-pulse">
          Carregando registro detalhado…
        </p>
      </Panel>
    )
  }

  if (!record && !selectedId) {
    return (
      <Panel>
        <PanelHeader title="Detalhes da execução" subtitle="SELECIONE UM REGISTRO" />
        <p className="py-8 text-center text-xs text-muted">
          Clique em uma execução da lista para inspecionar os metadados técnicos.
        </p>
      </Panel>
    )
  }

  const isCompleted = record?.outcome === 'completed'

  const durationMs =
    record?.duration_ns !== undefined
      ? Math.round(record.duration_ns / 1_000_000)
      : undefined

  const durationText = durationMs !== undefined ? `${durationMs} ms` : '—'

  const formattedTime = record?.started_at
    ? new Date(record.started_at).toLocaleTimeString('pt-BR')
    : '—'

  const summaryJson = JSON.stringify(
    {
      execution_id: record?.execution_id ?? selectedId,
      request_id: record?.request_id,
      provider: record?.provider_id,
      model: record?.model_id,
      outcome: record?.outcome,
      upstream_status: record?.upstream_status,
      duration: durationText,
    },
    null,
    2,
  )

  return (
    <Panel className="space-y-3.5">
      <PanelHeader title="Detalhes da execução" subtitle="SELECIONADA" />

      <div className="rounded-lg border border-line bg-raised p-3">
        <small className="block font-mono text-[9px] uppercase tracking-[0.05em] text-muted">
          REQUEST ID
        </small>
        <strong className="mt-1 block break-all font-mono text-xs text-primary">
          {record?.request_id ?? record?.execution_id ?? selectedId}
        </strong>
      </div>

      <div className="divide-y divide-line/30 text-[10px]">
        <div className="flex justify-between py-2">
          <span className="text-muted">Resultado</span>
          <strong
            className={`font-mono ${
              isCompleted ? 'text-healthy' : 'text-failure'
            }`}
          >
            {isCompleted ? 'Concluída' : 'Falha'}
          </strong>
        </div>
        <div className="flex justify-between py-2">
          <span className="text-muted">Modelo</span>
          <strong className="font-mono text-primary">
            {record?.model_id ?? '—'}
          </strong>
        </div>
        <div className="flex justify-between py-2">
          <span className="text-muted">Provedor</span>
          <strong className="font-mono text-secondary">
            {record?.provider_id ?? '—'}
          </strong>
        </div>
        <div className="flex justify-between py-2">
          <span className="text-muted">Horário</span>
          <strong className="font-mono text-secondary">{formattedTime}</strong>
        </div>
        <div className="flex justify-between py-2">
          <span className="text-muted">Duração</span>
          <strong className="font-mono text-secondary">{durationText}</strong>
        </div>
        <div className="flex justify-between py-2">
          <span className="text-muted">Operação</span>
          <strong className="font-mono text-secondary">responses.create</strong>
        </div>
      </div>

      <Link
        to="/catalog"
        className="block w-full rounded-md border border-cobalt/30 bg-accent-deep py-2 text-center text-xs font-semibold text-[#EAF0FC] transition-colors hover:bg-cobalt/30"
      >
        Explorar catálogo de modelos ↗
      </Link>

      <div className="rounded-lg border border-line bg-[#141B25] p-3">
        <small className="mb-1.5 block font-mono text-[9px] uppercase tracking-[0.05em] text-muted">
          Resumo do registro (sanitizado)
        </small>
        <pre className="max-h-48 overflow-auto font-mono text-[10px] leading-relaxed text-[#B8C7DE]">
          {summaryJson}
        </pre>
      </div>
    </Panel>
  )
}
