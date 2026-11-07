import type { ProviderQuotaMetric } from '../types/quota'

export type AntigravityFamilyId =
  | 'gemini_models'
  | 'claude_gpt_models'
  | 'gemini_special'
  | 'autocomplete_background'
  | 'other'

export type AntigravityWindowType = 'weekly' | '5h' | 'other'

export type AntigravityMemberModel = {
  id: string
  name: string
  remainingPercent?: number
  resetsAt?: string
}

export type AntigravityWindowLimit = {
  windowType: AntigravityWindowType
  windowLabel: string
  remainingPercent: number
  consumedPercent?: number
  resetsAt?: string
  models: AntigravityMemberModel[]
}

export type AntigravityFamilyGroup = {
  id: AntigravityFamilyId
  title: string
  limits: AntigravityWindowLimit[]
  totalModelsCount: number
}

function detectFamily(id: string, label: string): AntigravityFamilyId {
  const lowerId = id.toLowerCase()
  const lowerLabel = label.toLowerCase()

  if (lowerId.includes('argon') || lowerLabel.includes('argon')) {
    return 'gemini_special'
  }

  if (lowerId.startsWith('tab_') || lowerId.startsWith('chat_')) {
    return 'autocomplete_background'
  }

  if (
    lowerId.startsWith('claude') ||
    lowerId.startsWith('gpt') ||
    lowerLabel.includes('claude') ||
    lowerLabel.includes('gpt')
  ) {
    return 'claude_gpt_models'
  }

  if (lowerId.startsWith('gemini') || lowerLabel.includes('gemini')) {
    return 'gemini_models'
  }

  return 'other'
}

function familyTitle(familyId: AntigravityFamilyId): string {
  switch (familyId) {
    case 'gemini_models':
      return 'Gemini Models'
    case 'claude_gpt_models':
      return 'Claude and GPT models'
    case 'gemini_special':
      return 'Gemini Especial / Argon'
    case 'autocomplete_background':
      return 'Autocomplete & Background'
    case 'other':
      return 'Outros Modelos'
  }
}

function detectWindowType(metric: ProviderQuotaMetric, nowMs = Date.now()): AntigravityWindowType {
  const period = metric.period?.toLowerCase()
  if (period === 'weekly' || period === 'week') return 'weekly'
  if (period === '5h' || period === 'rolling') return '5h'

  if (metric.resets_at) {
    const resetTime = Date.parse(metric.resets_at)
    if (!Number.isNaN(resetTime)) {
      const diffMs = resetTime - nowMs
      const hours = diffMs / (1000 * 60 * 60)
      if (hours > 24) {
        return 'weekly'
      }
      return '5h'
    }
  }

  return 'other'
}

function windowLabel(windowType: AntigravityWindowType): string {
  switch (windowType) {
    case 'weekly':
      return 'Limite semanal'
    case '5h':
      return 'Limite 5h'
    case 'other':
      return 'Limite'
  }
}

/**
 * Categorizes and aggregates flat Antigravity quota metrics into model families
 * and shared quota windows, matching the official Antigravity IDE UX.
 */
export function categorizeAntigravityQuotas(
  quotas: readonly ProviderQuotaMetric[],
  nowMs = Date.now(),
): AntigravityFamilyGroup[] {
  if (!quotas || quotas.length === 0) return []

  // Step 1: Bucket models by family and windowType
  type WindowBucket = {
    windowType: AntigravityWindowType
    remainingPercent: number
    consumedPercent?: number
    resetsAt?: string
    models: AntigravityMemberModel[]
  }

  const familyMap = new Map<AntigravityFamilyId, Map<string, WindowBucket>>()

  for (const metric of quotas) {
    const isGroupMetric = metric.scope === 'group' || metric.metric.startsWith('group:')
    const familyId = detectFamily(metric.metric, metric.label || metric.metric)
    const windowType = detectWindowType(metric, nowMs)
    const isGenerativePool = familyId === 'gemini_models' || familyId === 'claude_gpt_models'
    const bucketWindowType = isGenerativePool
      ? (windowType === 'weekly' ? 'weekly' : '5h')
      : windowType
    const key = isGenerativePool
      ? bucketWindowType
      : `${windowType}_${metric.resets_at ?? ''}`

    if (!familyMap.has(familyId)) {
      familyMap.set(familyId, new Map())
    }
    const windowMap = familyMap.get(familyId)!

    const member: AntigravityMemberModel = {
      id: metric.metric,
      name: metric.label || metric.metric,
      remainingPercent: metric.remaining_percent,
      resetsAt: metric.resets_at,
    }

    if (!windowMap.has(key)) {
      windowMap.set(key, {
        windowType: bucketWindowType,
        remainingPercent: metric.remaining_percent ?? 100,
        consumedPercent: metric.consumed_percent,
        resetsAt: metric.resets_at,
        models: isGroupMetric ? [] : [member],
      })
    } else {
      const bucket = windowMap.get(key)!
      if (!isGroupMetric) {
        bucket.models.push(member)
      }
      if (isGroupMetric) {
        if (metric.remaining_percent != null) {
          bucket.remainingPercent = metric.remaining_percent
        }
        if (metric.resets_at) {
          bucket.resetsAt = metric.resets_at
        }
      } else if (metric.remaining_percent != null) {
        // If no group metric has set an authoritative percentage yet, take lowest
        bucket.remainingPercent = Math.min(bucket.remainingPercent, metric.remaining_percent)
      }
    }
  }

  // Step 2: Assemble structured family groups in preferred order
  const familyOrder: AntigravityFamilyId[] = [
    'gemini_models',
    'claude_gpt_models',
    'gemini_special',
    'other',
    'autocomplete_background',
  ]

  const result: AntigravityFamilyGroup[] = []

  // Global weekly reset reference across account (e.g. from Claude weekly bucket if present)
  let sharedWeeklyReset: string | undefined
  for (const windowMap of familyMap.values()) {
    for (const bucket of windowMap.values()) {
      if (bucket.windowType === 'weekly' && bucket.resetsAt) {
        sharedWeeklyReset = bucket.resetsAt
        break
      }
    }
    if (sharedWeeklyReset) break
  }

  for (const fid of familyOrder) {
    const windowMap = familyMap.get(fid)
    if (!windowMap || windowMap.size === 0) continue

    // For active generative model families, ensure both 'weekly' and '5h' limits are present
    if (fid === 'gemini_models' || fid === 'claude_gpt_models') {
      const hasWeekly = Array.from(windowMap.values()).some((b) => b.windowType === 'weekly')
      const has5h = Array.from(windowMap.values()).some((b) => b.windowType === '5h')

      if (!hasWeekly) {
        windowMap.set('weekly_default', {
          windowType: 'weekly',
          remainingPercent: 100,
          resetsAt: sharedWeeklyReset,
          models: [],
        })
      }
      if (!has5h) {
        windowMap.set('5h_default', {
          windowType: '5h',
          remainingPercent: 100,
          models: [],
        })
      }
    }

    const limits: AntigravityWindowLimit[] = []
    let totalModels = 0

    // Order window limits: 'weekly' first, then '5h', then 'other'
    const sortedBuckets = Array.from(windowMap.values()).sort((a, b) => {
      const order: Record<AntigravityWindowType, number> = { weekly: 1, '5h': 2, other: 3 }
      return (order[a.windowType] ?? 99) - (order[b.windowType] ?? 99)
    })

    for (const bucket of sortedBuckets) {
      totalModels += bucket.models.length
      limits.push({
        windowType: bucket.windowType,
        windowLabel: windowLabel(bucket.windowType),
        remainingPercent: bucket.remainingPercent,
        consumedPercent: bucket.consumedPercent,
        resetsAt: bucket.resetsAt,
        models: bucket.models,
      })
    }

    result.push({
      id: fid,
      title: familyTitle(fid),
      limits,
      totalModelsCount: totalModels,
    })
  }

  return result
}

