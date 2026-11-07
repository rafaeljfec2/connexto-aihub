import { describe, expect, it } from 'vitest'
import type { ProviderQuotaMetric } from '../types/quota'
import { categorizeAntigravityQuotas } from './antigravityCategories'

describe('categorizeAntigravityQuotas', () => {
  const baseTime = Date.parse('2026-11-06T19:00:00Z')

  it('handles empty or null quota lists', () => {
    expect(categorizeAntigravityQuotas([])).toEqual([])
  })

  it('groups Gemini, Claude/GPT, and internal models into distinct families', () => {
    const quotas: ProviderQuotaMetric[] = [
      {
        metric: 'claude-sonnet-5-5-medium',
        label: 'Claude Sonnet 5.5 (Medium)',
        nature: 'rate_limit',
        remaining_percent: 46.8,
        resets_at: '2026-11-11T21:00:00Z', // ~5 days ahead -> weekly
      },
      {
        metric: 'claude-opus-5-5-low',
        label: 'Claude Opus 5.5 (Low)',
        nature: 'rate_limit',
        remaining_percent: 46.8,
        resets_at: '2026-11-11T21:00:00Z',
      },
      {
        metric: 'gpt-oss-120b-medium',
        label: 'GPT-OSS 120B (Medium)',
        nature: 'rate_limit',
        remaining_percent: 46.8,
        resets_at: '2026-11-11T21:00:00Z',
      },
      {
        metric: 'gemini-2.5-pro',
        label: 'Gemini 2.5 Pro',
        nature: 'rate_limit',
        remaining_percent: 73.83,
        resets_at: '2026-11-06T21:20:00Z', // ~2h20m ahead -> 5h
      },
      {
        metric: 'gemini-3.5-flash-lite',
        label: 'Gemini 3.5 Flash Lite',
        nature: 'rate_limit',
        remaining_percent: 73.83,
        resets_at: '2026-11-06T21:20:00Z',
      },
      {
        metric: 'gemini-4-argon',
        label: 'Gemini 4 Argon',
        nature: 'rate_limit',
        remaining_percent: 100,
        period: 'weekly',
      },
      {
        metric: 'tab_jump_flash_lite_preview',
        label: 'tab_jump_flash_lite_preview',
        nature: 'rate_limit',
        remaining_percent: 100,
      },
      {
        metric: 'chat_20706',
        label: 'chat_20706',
        nature: 'rate_limit',
        remaining_percent: 100,
      },
    ]

    const groups = categorizeAntigravityQuotas(quotas, baseTime)

    // Order: Gemini Models -> Claude and GPT models -> Gemini Special -> Autocomplete & Background
    expect(groups.map((g) => g.id)).toEqual([
      'gemini_models',
      'claude_gpt_models',
      'gemini_special',
      'autocomplete_background',
    ])

    // Gemini Models verification
    const geminiGroup = groups.find((g) => g.id === 'gemini_models')!
    expect(geminiGroup.title).toBe('Gemini Models')
    expect(geminiGroup.totalModelsCount).toBe(2)
    expect(geminiGroup.limits).toHaveLength(1)
    expect(geminiGroup.limits[0].windowType).toBe('5h')
    expect(geminiGroup.limits[0].windowLabel).toBe('Limite 5h')
    expect(geminiGroup.limits[0].remainingPercent).toBe(73.83)
    expect(geminiGroup.limits[0].models).toHaveLength(2)

    // Claude and GPT models verification (deduplicated shared pool)
    const claudeGptGroup = groups.find((g) => g.id === 'claude_gpt_models')!
    expect(claudeGptGroup.title).toBe('Claude and GPT models')
    expect(claudeGptGroup.totalModelsCount).toBe(3)
    expect(claudeGptGroup.limits).toHaveLength(1)
    expect(claudeGptGroup.limits[0].windowType).toBe('weekly')
    expect(claudeGptGroup.limits[0].windowLabel).toBe('Limite semanal')
    expect(claudeGptGroup.limits[0].remainingPercent).toBe(46.8)
    expect(claudeGptGroup.limits[0].models.map((m) => m.id)).toEqual([
      'claude-sonnet-5-5-medium',
      'claude-opus-5-5-low',
      'gpt-oss-120b-medium',
    ])

    // Special model verification
    const argonGroup = groups.find((g) => g.id === 'gemini_special')!
    expect(argonGroup.title).toBe('Gemini Especial / Argon')
    expect(argonGroup.limits[0].windowLabel).toBe('Limite semanal')
    expect(argonGroup.limits[0].remainingPercent).toBe(100)

    // Internal completion models grouped
    const autoGroup = groups.find((g) => g.id === 'autocomplete_background')!
    expect(autoGroup.title).toBe('Autocomplete & Background')
    expect(autoGroup.totalModelsCount).toBe(2)
  })

  it('supports multiple window limits under the same family when reported', () => {
    const quotas: ProviderQuotaMetric[] = [
      {
        metric: 'claude-sonnet-5-5-medium',
        label: 'Claude Sonnet 5.5',
        nature: 'rate_limit',
        remaining_percent: 48,
        period: 'weekly',
        resets_at: '2026-11-12T00:00:00Z',
      },
      {
        metric: 'claude-sonnet-5-5-rolling',
        label: 'Claude Sonnet 5.5 5h',
        nature: 'rate_limit',
        remaining_percent: 98,
        period: '5h',
        resets_at: '2026-11-06T21:05:00Z',
      },
    ]

    const groups = categorizeAntigravityQuotas(quotas, baseTime)
    expect(groups).toHaveLength(1)
    const family = groups[0]
    expect(family.limits).toHaveLength(2)
    // Weekly comes before 5h
    expect(family.limits[0].windowType).toBe('weekly')
    expect(family.limits[0].remainingPercent).toBe(48)
    expect(family.limits[1].windowType).toBe('5h')
    expect(family.limits[1].remainingPercent).toBe(98)
  })
})
