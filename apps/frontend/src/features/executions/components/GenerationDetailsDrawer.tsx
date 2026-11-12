import * as DialogPrimitive from '@radix-ui/react-dialog'
import { X } from 'lucide-react'
import type { ReactNode } from 'react'
import { useTranslation } from 'react-i18next'
import {
  formatGenerationStatus,
  formatGenerationThroughput,
  formatGenerationTime,
  formatGenerationTokens,
  formatOptionalDuration,
  formatProviderReportedCost,
} from '../lib/formatGeneration'
import type { Generation, GenerationDetail } from '../types'

type GenerationDetailsDrawerProps = {
  readonly open: boolean
  readonly onOpenChange: (open: boolean) => void
  readonly generation: Generation | null
  readonly detail: GenerationDetail | undefined
  readonly isLoading: boolean
  readonly isError: boolean
  readonly errorMessage?: string
  readonly onRestoreFocus: () => void
}

const PROVIDER_EVENT_TYPES = [
  'routing_selected',
  'upstream_request_started',
  'upstream_response_received',
  'first_output',
  'generation_finished',
] as const

export function GenerationDetailsDrawer({
  open,
  onOpenChange,
  generation,
  detail,
  isLoading,
  isError,
  errorMessage,
  onRestoreFocus,
}: GenerationDetailsDrawerProps) {
  const { t } = useTranslation('executions')
  const notReported = t('format.notReported')
  const current = detail?.execution_id ? detail : generation
  const providerEvents = detail?.provider_responses ?? []

  const eventLabel = (type: string): string => {
    if ((PROVIDER_EVENT_TYPES as readonly string[]).includes(type)) {
      return t(`drawer.events.${type}`)
    }
    return t('drawer.events.default')
  }

  return (
    <DialogPrimitive.Root modal={false} open={open} onOpenChange={onOpenChange}>
      <DialogPrimitive.Portal>
        <DialogPrimitive.Overlay className="generation-drawer-backdrop" />
        <DialogPrimitive.Content
          className="generation-drawer"
          onInteractOutside={(event) => event.preventDefault()}
          onCloseAutoFocus={(event) => {
            event.preventDefault()
            onRestoreFocus()
          }}
        >
          <header className="generation-drawer-head">
            <div className="min-w-0">
              <p className="generation-drawer-eyebrow">{t('drawer.eyebrow')}</p>
              <DialogPrimitive.Title className="generation-drawer-title">
                {current?.model_id ?? notReported}
              </DialogPrimitive.Title>
              <DialogPrimitive.Description className="generation-drawer-description">
                {current
                  ? `${current.provider_id} · ${current.outcome}`
                  : t('drawer.selectPrompt')}
              </DialogPrimitive.Description>
            </div>
            <DialogPrimitive.Close className="generation-close" aria-label={t('drawer.closeAriaLabel')}>
              <X aria-hidden="true" className="size-4" />
            </DialogPrimitive.Close>
          </header>

          <div className="generation-drawer-scroll">
            {isLoading ? (
              <p className="py-8 text-center text-xs text-muted animate-pulse">{t('drawer.loading')}</p>
            ) : null}
            {isError ? (
              <p className="banner banner-error" role="alert">
                {errorMessage ?? t('drawer.loadErrorDefault')}
              </p>
            ) : null}
            {current ? (
              <>
                <section className="generation-detail-metrics" aria-label={t('drawer.summaryAriaLabel')}>
                  <Metric label={t('drawer.metrics.providerLatency')} value={formatOptionalDuration(detail?.telemetry?.provider_latency_ns ?? null)} />
                  <Metric label={t('drawer.metrics.throughput')} value={formatGenerationThroughput(current)} />
                  <Metric
                    label={t('drawer.metrics.reportedCost')}
                    value={formatProviderReportedCost(current.usage.provider_reported_cost, current.usage.currency)}
                  />
                  <Metric
                    label={t('drawer.metrics.listCost')}
                    value={formatProviderReportedCost(current.usage.provider_list_cost, current.usage.currency)}
                  />
                  <Metric
                    label={t('drawer.metrics.tokenFlow')}
                    value={`${formatGenerationTokens(current.usage.input_tokens)} → ${formatGenerationTokens(current.usage.output_tokens)}`}
                  />
                </section>

                <DetailSection title={t('drawer.sections.summary')}>
                  <DetailRow label={t('drawer.fields.outcome')} value={current.outcome} />
                  <DetailRow label={t('drawer.fields.started')} value={formatGenerationTime(current.started_at)} />
                  <DetailRow label={t('drawer.fields.duration')} value={formatOptionalDuration(current.duration_ns)} />
                  <DetailRow label={t('drawer.fields.usageStatus')} value={formatGenerationStatus(current.usage.status)} />
                </DetailSection>

                <DetailSection title={t('drawer.sections.overview')}>
                  <DetailRow label={t('drawer.fields.provider')} value={current.provider_id} />
                  <DetailRow label={t('drawer.fields.modelId')} value={current.model_id ?? notReported} />
                  <DetailRow label={t('drawer.fields.canonicalModelId')} value={notReported} />
                  <DetailRow label={t('drawer.fields.dataPolicy')} value={notReported} />
                  <DetailRow
                    label={t('drawer.fields.streaming')}
                    value={
                      detail?.telemetry?.streaming === null || detail?.telemetry?.streaming === undefined
                        ? notReported
                        : String(detail.telemetry.streaming)
                    }
                  />
                  <DetailRow
                    label={t('drawer.fields.firstToken')}
                    value={formatOptionalDuration(detail?.telemetry?.time_to_first_token_ns ?? null)}
                  />
                  <DetailRow
                    label={t('drawer.fields.finishReason')}
                    value={detail?.telemetry?.finish_reason ?? notReported}
                  />
                </DetailSection>

                <DetailSection title={t('drawer.sections.request')}>
                  <DetailRow label={t('drawer.fields.application')} value={current.telemetry.client_app ?? notReported} />
                  <DetailRow label={t('drawer.fields.executionId')} value={current.execution_id} mono />
                  <DetailRow label={t('drawer.fields.requestId')} value={current.request_id ?? notReported} mono />
                  <DetailRow label={t('drawer.fields.providerSession')} value={detail?.provider_session_id ?? notReported} mono />
                  <DetailRow
                    label={t('drawer.fields.reportedCost')}
                    value={formatProviderReportedCost(current.usage.provider_reported_cost, current.usage.currency)}
                  />
                  <DetailRow
                    label={t('drawer.fields.listCost')}
                    value={formatProviderReportedCost(current.usage.provider_list_cost, current.usage.currency)}
                  />
                  <DetailRow label={t('drawer.fields.costSource')} value={current.usage.source} />
                </DetailSection>

                <DetailSection title={t('drawer.sections.providerResponses')}>
                  {providerEvents.length > 0 ? (
                    <ol className="generation-timeline">
                      {providerEvents.map((event) => (
                        <li key={`${event.sequence}-${event.type}`}>
                          <span className="generation-timeline-dot" aria-hidden="true" />
                          <div className="min-w-0">
                            <strong>{eventLabel(event.type)}</strong>
                            <small>
                              {event.provider_id} · {formatGenerationTime(event.observed_at)}
                            </small>
                          </div>
                          <span className="font-mono text-[0.5625rem] text-muted">
                            {event.duration_ns === null
                              ? t('format.emDash')
                              : formatOptionalDuration(event.duration_ns)}
                          </span>
                        </li>
                      ))}
                    </ol>
                  ) : (
                    <p className="text-[0.625rem] text-muted">{t('drawer.noProviderEvents')}</p>
                  )}
                </DetailSection>
              </>
            ) : null}
          </div>
        </DialogPrimitive.Content>
      </DialogPrimitive.Portal>
    </DialogPrimitive.Root>
  )
}

function Metric({ label, value }: { readonly label: string; readonly value: string }) {
  return (
    <div className="generation-detail-metric">
      <span>{label}</span>
      <strong>{value}</strong>
    </div>
  )
}

function DetailSection({ title, children }: { readonly title: string; readonly children: ReactNode }) {
  return (
    <section className="generation-detail-section">
      <h3>{title}</h3>
      <div className="generation-detail-rows">{children}</div>
    </section>
  )
}

function DetailRow({
  label,
  value,
  mono = false,
}: {
  readonly label: string
  readonly value: string
  readonly mono?: boolean
}) {
  return (
    <div className="generation-detail-row">
      <span>{label}</span>
      <strong className={mono ? 'font-mono' : undefined}>{value}</strong>
    </div>
  )
}
