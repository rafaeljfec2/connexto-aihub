export type GenerationPeriodPreset = '30d' | '7d' | 'all' | 'custom'

export function resolvePeriodPresetRange(
  preset: GenerationPeriodPreset,
  today = new Date(),
): { start_date: string; end_date: string } {
  if (preset === 'all') {
    return { start_date: '', end_date: '' }
  }

  const end = new Date(Date.UTC(today.getUTCFullYear(), today.getUTCMonth(), today.getUTCDate()))
  const start = new Date(end)

  if (preset === '7d') {
    start.setUTCDate(start.getUTCDate() - 6)
  } else if (preset === '30d') {
    start.setUTCDate(start.getUTCDate() - 29)
  }

  return {
    start_date: formatIsoDate(start),
    end_date: formatIsoDate(end),
  }
}

export function inferPeriodPreset(
  startDate: string,
  endDate: string,
  today = new Date(),
): GenerationPeriodPreset {
  if (!startDate && !endDate) {
    return 'all'
  }

  const ranges: GenerationPeriodPreset[] = ['30d', '7d']
  for (const preset of ranges) {
    const range = resolvePeriodPresetRange(preset, today)
    if (range.start_date === startDate && range.end_date === endDate) {
      return preset
    }
  }

  return 'custom'
}

function formatIsoDate(value: Date): string {
  return value.toISOString().slice(0, 10)
}
