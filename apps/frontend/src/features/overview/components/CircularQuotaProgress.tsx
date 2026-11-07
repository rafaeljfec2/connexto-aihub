type CircularQuotaProgressProps = Readonly<{
  percent: number
  size?: number
  strokeWidth?: number
  className?: string
}>

export function CircularQuotaProgress({
  percent,
  size = 18,
  strokeWidth = 2.5,
  className = '',
}: CircularQuotaProgressProps) {
  const radius = (size - strokeWidth) / 2
  const circumference = 2 * Math.PI * radius
  const validPercent = Math.max(0, Math.min(100, Number.isFinite(percent) ? percent : 0))
  const strokeDashoffset = circumference - (validPercent / 100) * circumference

  let colorClass = 'text-emerald-400 stroke-emerald-400'
  if (validPercent < 15) {
    colorClass = 'text-rose-400 stroke-rose-400'
  } else if (validPercent < 35) {
    colorClass = 'text-amber-400 stroke-amber-400'
  }

  return (
    <svg
      width={size}
      height={size}
      viewBox={`0 0 ${size} ${size}`}
      className={`shrink-0 -rotate-90 transform ${className}`}
      aria-hidden="true"
    >
      <circle
        cx={size / 2}
        cy={size / 2}
        r={radius}
        fill="transparent"
        stroke="currentColor"
        strokeWidth={strokeWidth}
        className="text-white/10"
      />
      <circle
        cx={size / 2}
        cy={size / 2}
        r={radius}
        fill="transparent"
        stroke="currentColor"
        strokeWidth={strokeWidth}
        strokeDasharray={circumference}
        strokeDashoffset={strokeDashoffset}
        strokeLinecap="round"
        className={`transition-all duration-300 ${colorClass}`}
      />
    </svg>
  )
}

