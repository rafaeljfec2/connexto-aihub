const illustrativeBars = [
  { label: '05 OUT', height: 42 },
  { label: '10 OUT', height: 58 },
  { label: '15 OUT', height: 36 },
  { label: '20 OUT', height: 72 },
  { label: '25 OUT', height: 48 },
  { label: '30 OUT', height: 64 },
  { label: '05 NOV', height: 52 },
] as const

/** Decorative chart only — not backed by a generation aggregate API (see FE-EXECUTIONS-GENERATIONS-REFINEMENT). */
export function GenerationsActivityPanel() {
  return (
    <section
      className="generations-activity-panel"
      aria-label="Atividade de gerações (visualização ilustrativa)"
    >
      <div className="generations-activity-head">
        <h2 className="generations-activity-title">Atividade de gerações</h2>
        <span className="generations-activity-note">VISUALIZAÇÃO ILUSTRATIVA · 30 DIAS</span>
      </div>
      <div className="generations-activity-chart" role="img" aria-hidden="true">
        {illustrativeBars.map((bar) => (
          <div key={bar.label} className="generations-activity-bar-wrap">
            <div className="generations-activity-bar" style={{ height: `${bar.height}px` }} />
            <span>{bar.label}</span>
          </div>
        ))}
      </div>
    </section>
  )
}
