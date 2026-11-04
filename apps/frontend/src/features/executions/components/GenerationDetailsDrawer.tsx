import * as DialogPrimitive from '@radix-ui/react-dialog'
import { X } from 'lucide-react'
import type { ReactNode } from 'react'
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
  const current = detail?.execution_id ? detail : generation
  const providerEvents = detail?.provider_responses ?? []
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
              <p className="generation-drawer-eyebrow">GENERATION DETAILS</p>
              <DialogPrimitive.Title className="generation-drawer-title">
                {current?.model_id ?? 'Model not reported'}
              </DialogPrimitive.Title>
              <DialogPrimitive.Description className="generation-drawer-description">
                {current ? `${current.provider_id} · ${current.outcome}` : 'Select a generation to inspect its metadata.'}
              </DialogPrimitive.Description>
            </div>
            <DialogPrimitive.Close className="generation-close" aria-label="Close generation details">
              <X aria-hidden="true" className="size-4" />
            </DialogPrimitive.Close>
          </header>

          <div className="generation-drawer-scroll">
            {isLoading ? <p className="py-8 text-center text-xs text-muted animate-pulse">Loading generation details…</p> : null}
            {isError ? <p className="banner banner-error" role="alert">{errorMessage ?? 'Unable to load generation details.'}</p> : null}
            {current ? (
              <>
                <section className="generation-detail-metrics" aria-label="Generation summary">
                  <Metric label="Provider latency" value={formatOptionalDuration(detail?.telemetry?.provider_latency_ns ?? null)} />
                  <Metric label="Throughput" value={formatGenerationThroughput(current)} />
                  <Metric label="Reported cost (usage.cost)" value={formatProviderReportedCost(current.usage.provider_reported_cost, current.usage.currency)} />
                  <Metric label="List cost (opencode.list_cost)" value={formatProviderReportedCost(current.usage.provider_list_cost, current.usage.currency)} />
                  <Metric label="Input → output tokens" value={`${formatGenerationTokens(current.usage.input_tokens)} → ${formatGenerationTokens(current.usage.output_tokens)}`} />
                </section>

                <DetailSection title="Generation summary">
                  <DetailRow label="Outcome" value={current.outcome} />
                  <DetailRow label="Started" value={formatGenerationTime(current.started_at)} />
                  <DetailRow label="Duration" value={formatOptionalDuration(current.duration_ns)} />
                  <DetailRow label="Usage status" value={formatGenerationStatus(current.usage.status)} />
                </DetailSection>

                <DetailSection title="Overview">
                  <DetailRow label="Provider" value={current.provider_id} />
                  <DetailRow label="Model ID" value={current.model_id ?? 'Not reported'} />
                  <DetailRow label="Canonical model ID" value="Not reported" />
                  <DetailRow label="Data policy" value="Not reported" />
                  <DetailRow label="Streaming" value={detail?.telemetry?.streaming === null || detail?.telemetry?.streaming === undefined ? 'Not reported' : String(detail.telemetry.streaming)} />
                  <DetailRow label="First token" value={formatOptionalDuration(detail?.telemetry?.time_to_first_token_ns ?? null)} />
                  <DetailRow label="Finish reason" value={detail?.telemetry?.finish_reason ?? 'Not reported'} />
                </DetailSection>

                <DetailSection title="Request">
                  <DetailRow label="Application" value="Not reported" />
                  <DetailRow label="Execution ID" value={current.execution_id} mono />
                  <DetailRow label="Request ID" value={current.request_id ?? 'Not reported'} mono />
                  <DetailRow label="Provider session" value={detail?.provider_session_id ?? 'Not reported'} mono />
                  <DetailRow label="Reported cost (usage.cost)" value={formatProviderReportedCost(current.usage.provider_reported_cost, current.usage.currency)} />
                  <DetailRow label="List cost (opencode.list_cost)" value={formatProviderReportedCost(current.usage.provider_list_cost, current.usage.currency)} />
                  <DetailRow label="Cost source" value={current.usage.source} />
                </DetailSection>

                <DetailSection title="Provider responses">
                  {providerEvents.length > 0 ? (
                    <ol className="generation-timeline">
                      {providerEvents.map((event) => (
                        <li key={`${event.sequence}-${event.type}`}>
                          <span className="generation-timeline-dot" aria-hidden="true" />
                          <div className="min-w-0">
                            <strong>{eventLabel(event.type)}</strong>
                            <small>{event.provider_id} · {formatGenerationTime(event.observed_at)}</small>
                          </div>
                          <span className="font-mono text-[9px] text-muted">
                            {event.duration_ns === null ? '—' : formatOptionalDuration(event.duration_ns)}
                          </span>
                        </li>
                      ))}
                    </ol>
                  ) : (
                    <p className="text-[10px] text-muted">No provider response events were recorded for this generation.</p>
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

function DetailRow({ label, value, mono = false }: { readonly label: string; readonly value: string; readonly mono?: boolean }) {
  return (
    <div className="generation-detail-row">
      <span>{label}</span>
      <strong className={mono ? 'font-mono' : undefined}>{value}</strong>
    </div>
  )
}

function eventLabel(type: string): string {
  switch (type) {
    case 'routing_selected': return 'Routing selected'
    case 'upstream_request_started': return 'Provider request started'
    case 'upstream_response_received': return 'Provider response received'
    case 'first_output': return 'First output observed'
    case 'generation_finished': return 'Generation finished'
    default: return 'Observed provider event'
  }
}
