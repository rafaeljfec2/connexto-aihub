import { useTranslation } from 'react-i18next'
import { Link } from 'react-router-dom'
import { Panel, PanelHeader } from '../../../shared/ui'
import { formatDateTime } from '../../../shared/i18n/format'
import { formatDurationNs } from '../lib/formatExecution'
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
  const { t } = useTranslation('executions')
  const emDash = t('format.emDash')

  if (isError) {
    return (
      <Panel>
        <PanelHeader title={t('detailPanel.title')} subtitle={t('detailPanel.subtitleError')} />
        <p className="py-4 text-xs text-failure" role="alert">
          {errorMessage ?? t('detailPanel.loadErrorDefault')}
        </p>
      </Panel>
    )
  }

  if (isLoading) {
    return (
      <Panel>
        <PanelHeader title={t('detailPanel.title')} subtitle={t('detailPanel.subtitleLoading')} />
        <p className="py-8 text-center text-xs text-muted animate-pulse">
          {t('detailPanel.loadingDetail')}
        </p>
      </Panel>
    )
  }

  if (!record && !selectedId) {
    return (
      <Panel>
        <PanelHeader title={t('detailPanel.title')} subtitle={t('detailPanel.subtitleSelect')} />
        <p className="py-8 text-center text-xs text-muted">{t('detailPanel.selectPrompt')}</p>
      </Panel>
    )
  }

  const isCompleted = record?.outcome === 'completed'

  const durationText =
    record?.duration_ns !== undefined ? formatDurationNs(record.duration_ns) : emDash

  const formattedTime = record?.started_at
    ? formatDateTime(record.started_at, { hour: '2-digit', minute: '2-digit', second: '2-digit' })
    : emDash

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
      <PanelHeader title={t('detailPanel.title')} subtitle={t('detailPanel.subtitleSelected')} />

      <div className="rounded-lg border border-line bg-raised p-3">
        <small className="block font-mono text-[0.5625rem] uppercase tracking-[0.05em] text-muted">
          {t('detailPanel.requestId')}
        </small>
        <strong className="mt-1 block break-all font-mono text-xs text-primary">
          {record?.request_id ?? record?.execution_id ?? selectedId}
        </strong>
      </div>

      <div className="divide-y divide-line/30 text-[0.625rem]">
        <div className="flex justify-between py-2">
          <span className="text-muted">{t('detailPanel.fields.outcome')}</span>
          <strong
            className={`font-mono ${
              isCompleted ? 'text-healthy' : 'text-failure'
            }`}
          >
            {isCompleted ? t('table.legacy.completed') : t('table.legacy.failed')}
          </strong>
        </div>
        <div className="flex justify-between py-2">
          <span className="text-muted">{t('detailPanel.fields.model')}</span>
          <strong className="font-mono text-primary">
            {record?.model_id ?? emDash}
          </strong>
        </div>
        <div className="flex justify-between py-2">
          <span className="text-muted">{t('detailPanel.fields.provider')}</span>
          <strong className="font-mono text-secondary">
            {record?.provider_id ?? emDash}
          </strong>
        </div>
        <div className="flex justify-between py-2">
          <span className="text-muted">{t('detailPanel.fields.time')}</span>
          <strong className="font-mono text-secondary">{formattedTime}</strong>
        </div>
        <div className="flex justify-between py-2">
          <span className="text-muted">{t('detailPanel.fields.duration')}</span>
          <strong className="font-mono text-secondary">{durationText}</strong>
        </div>
        <div className="flex justify-between py-2">
          <span className="text-muted">{t('detailPanel.fields.operation')}</span>
          <strong className="font-mono text-secondary">responses.create</strong>
        </div>
      </div>

      <Link
        to="/catalog"
        className="block w-full rounded-md border border-cobalt/30 bg-accent-deep py-2 text-center text-xs font-semibold text-[#EAF0FC] transition-colors hover:bg-cobalt/30"
      >
        {t('detailPanel.exploreCatalog')}
      </Link>

      <div className="rounded-lg border border-line bg-[#141B25] p-3">
        <small className="mb-1.5 block font-mono text-[0.5625rem] uppercase tracking-[0.05em] text-muted">
          {t('detailPanel.summaryTitle')}
        </small>
        <pre className="max-h-48 overflow-auto font-mono text-[0.625rem] leading-relaxed text-[#B8C7DE]">
          {summaryJson}
        </pre>
      </div>
    </Panel>
  )
}
