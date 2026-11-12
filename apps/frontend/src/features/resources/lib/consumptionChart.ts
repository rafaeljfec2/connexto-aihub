import type { AdmissionsDailyRow } from '../types'

const CHART_WIDTH = 560
const CHART_HEIGHT = 116
const CHART_PADDING_X = 0
const CHART_PADDING_Y = 8

const WEEKDAY_PT = ['DOM', 'SEG', 'TER', 'QUA', 'QUI', 'SEX', 'SÁB'] as const

function admissionsChartPoints(daily: AdmissionsDailyRow[]): Array<{ x: number; y: number }> | null {
  if (daily.length === 0) {
    return null
  }

  const counts = daily.map((row) => row.admitted_count)
  const max = Math.max(...counts, 1)
  const innerWidth = CHART_WIDTH - CHART_PADDING_X * 2
  const innerHeight = CHART_HEIGHT - CHART_PADDING_Y * 2

  return daily.map((row, index) => {
    const x =
      daily.length === 1
        ? CHART_WIDTH / 2
        : CHART_PADDING_X + (index / (daily.length - 1)) * innerWidth
    const y = CHART_PADDING_Y + innerHeight - (row.admitted_count / max) * innerHeight
    return { x, y }
  })
}

function formatCoord(value: number): string {
  return value.toFixed(1)
}

function clamp(value: number, min: number, max: number): number {
  return Math.min(max, Math.max(min, value))
}

/** Catmull-Rom spline drawn as cubic beziers, matching the smooth Figma stroke. */
function smoothLinePath(points: Array<{ x: number; y: number }>): string {
  const first = points[0]
  if (!first) {
    return ''
  }
  if (points.length === 1) {
    return `M 0 ${formatCoord(first.y)} L ${CHART_WIDTH} ${formatCoord(first.y)}`
  }

  const minY = CHART_PADDING_Y
  const maxY = CHART_HEIGHT - CHART_PADDING_Y
  let path = `M ${formatCoord(first.x)} ${formatCoord(first.y)}`

  for (let index = 0; index < points.length - 1; index += 1) {
    const previous = points[index - 1] ?? points[index]
    const current = points[index]
    const next = points[index + 1]
    const after = points[index + 2] ?? next
    if (!previous || !current || !next || !after) {
      continue
    }
    const controlStartX = current.x + (next.x - previous.x) / 6
    const controlStartY = clamp(current.y + (next.y - previous.y) / 6, minY, maxY)
    const controlEndX = next.x - (after.x - current.x) / 6
    const controlEndY = clamp(next.y - (after.y - current.y) / 6, minY, maxY)
    path += ` C ${formatCoord(controlStartX)} ${formatCoord(controlStartY)} ${formatCoord(controlEndX)} ${formatCoord(controlEndY)} ${formatCoord(next.x)} ${formatCoord(next.y)}`
  }

  return path
}

export function buildAdmissionsAreaPath(daily: AdmissionsDailyRow[]): string | null {
  const points = admissionsChartPoints(daily)
  if (!points) {
    return null
  }

  const last = points[points.length - 1]
  const first = points[0]
  if (!last || !first) {
    return null
  }

  const closeStartX = points.length === 1 ? 0 : first.x
  const closeEndX = points.length === 1 ? CHART_WIDTH : last.x
  return `${smoothLinePath(points)} L ${formatCoord(closeEndX)} ${CHART_HEIGHT} L ${formatCoord(closeStartX)} ${CHART_HEIGHT} Z`
}

export function buildAdmissionsLinePath(daily: AdmissionsDailyRow[]): string | null {
  const points = admissionsChartPoints(daily)
  if (!points) {
    return null
  }
  return smoothLinePath(points)
}

export const consumptionChartViewBox = `0 0 ${CHART_WIDTH} ${CHART_HEIGHT}`

export function consumptionChartDayLabels(daily: AdmissionsDailyRow[]): string[] {
  return daily.map((row) => {
    const date = new Date(row.period_start)
    if (Number.isNaN(date.getTime())) {
      return '—'
    }
    return WEEKDAY_PT[date.getUTCDay()] ?? '—'
  })
}

export function admissionsChartEndpoint(
  daily: AdmissionsDailyRow[],
): { x: number; y: number } | null {
  const points = admissionsChartPoints(daily)
  if (!points || points.length === 0) {
    return null
  }
  return points[points.length - 1] ?? null
}
