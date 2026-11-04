import { Panel, PanelHeader } from '../../../shared/ui/panel'
import type { AdmissionsDailyRow } from '../types'
import { SectionFetchedAt } from './SectionFetchedAt'

type AdmissionsDailyChartProps = {
  daily: AdmissionsDailyRow[]
  totalAdmitted: number
  fetchedAtMs?: number
}

export function AdmissionsDailyChart({ daily, totalAdmitted, fetchedAtMs }: AdmissionsDailyChartProps) {
  const max = daily.reduce((acc, row) => Math.max(acc, row.admitted_count), 0)

  return (
    <Panel>
      <PanelHeader
        title="Resource admissions"
        subtitle="Allowed admissions per UTC day (GET /api/resource-analytics/admissions)"
      />
      <div className="mb-1 flex items-baseline gap-2">
        <strong className="text-[22px] font-semibold tracking-[-0.04em]">
          {totalAdmitted.toLocaleString('en-US')}
        </strong>
        <span className="font-mono text-[9px] text-[#8FA0B6]">TOTAL ADMITTED</span>
      </div>
      {daily.length === 0 ? (
        <p className="text-[10px] text-muted">No admissions in the selected period.</p>
      ) : (
        <div className="grid grid-cols-[repeat(auto-fill,minmax(36px,1fr))] items-end gap-1.5 pt-2">
          {daily.map((row) => {
            const height = max > 0 ? Math.max(4, Math.round((row.admitted_count / max) * 72)) : 4
            const label = row.period_start.slice(0, 10)
            return (
              <div key={row.period_start} className="flex min-w-0 flex-col items-center gap-1">
                <div
                  className="w-full rounded-t bg-cobalt/80"
                  style={{ height }}
                  title={`${label}: ${row.admitted_count}`}
                  role="img"
                  aria-label={`${label} admitted ${row.admitted_count}`}
                />
                <span className="font-mono text-[8px] text-[#758196]">{label.slice(5)}</span>
              </div>
            )
          })}
        </div>
      )}
      <SectionFetchedAt updatedAtMs={fetchedAtMs} />
    </Panel>
  )
}
