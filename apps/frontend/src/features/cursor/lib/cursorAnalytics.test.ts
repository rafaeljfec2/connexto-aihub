import { describe, expect, it } from 'vitest'
import type { CursorDailyMetric } from '../types'
import { buildCursorDailySeries, rankCursorModelUsage, summarizeCursorDailyMetrics } from './cursorAnalytics'

const activeDay: CursorDailyMetric = {
  date: '1762819200000',
  linesAdded: 10,
  linesDeleted: 2,
  acceptedLinesAdded: 10,
  acceptedLinesDeleted: 2,
  totalApplies: 4,
  totalAccepts: 4,
  totalRejects: 1,
  agentRequests: 3,
  subscriptionIncludedReqs: 3,
  modelUsage: [
    { name: 'default', count: 3 },
  ],
}

const busyDay: CursorDailyMetric = {
  date: '1762905600000',
  linesAdded: 100,
  acceptedLinesAdded: 50,
  totalApplies: 2,
  totalAccepts: 2,
  agentRequests: 5,
  chatRequests: 2,
  subscriptionIncludedReqs: 6,
  usageBasedReqs: 1,
  modelUsage: [
    { name: 'default', count: 3 },
    { name: 'gemini-3-pro-preview', count: 2 },
    { name: 'agent_review' },
  ],
}

const idleDay: CursorDailyMetric = {
  date: '1762992000000',
}

describe('summarizeCursorDailyMetrics', () => {
  it('sums only fields the source actually reports', () => {
    const summary = summarizeCursorDailyMetrics([activeDay, busyDay, idleDay])

    expect(summary.agentRequests).toBe(8)
    expect(summary.chatRequests).toBe(2)
    expect(summary.subscriptionIncludedReqs).toBe(9)
    expect(summary.usageBasedReqs).toBe(1)
    expect(summary.linesAdded).toBe(110)
    expect(summary.linesDeleted).toBe(2)
    expect(summary.acceptedLinesAdded).toBe(60)
    expect(summary.acceptedLinesDeleted).toBe(2)
    expect(summary.totalApplies).toBe(6)
    expect(summary.totalAccepts).toBe(6)
    expect(summary.totalRejects).toBe(1)
    expect(summary.activeDays).toBe(2)
    expect(summary.firstActivityAt).toBe('1762819200000')
    expect(summary.lastActivityAt).toBe('1762905600000')
  })

  it('keeps fields absent when no day reports them', () => {
    const summary = summarizeCursorDailyMetrics([idleDay])

    expect(summary.agentRequests).toBeUndefined()
    expect(summary.totalRejects).toBeUndefined()
    expect(summary.modelUsages).toBeUndefined()
    expect(summary.activeDays).toBe(0)
  })
})

describe('rankCursorModelUsage', () => {
  it('aggregates usage across days and sorts by count', () => {
    const ranking = rankCursorModelUsage([activeDay, busyDay, idleDay], 5)

    expect(ranking).toEqual([
      { name: 'default', count: 6 },
      { name: 'gemini-3-pro-preview', count: 2 },
      { name: 'agent_review', count: 1 },
    ])
  })

  it('collapses entries beyond the limit into Others only when they exist', () => {
    const ranking = rankCursorModelUsage([busyDay], 2)

    expect(ranking).toEqual([
      { name: 'default', count: 3 },
      { name: 'gemini-3-pro-preview', count: 2 },
      { name: 'Others', count: 1 },
    ])
  })

  it('returns an empty list when nothing is reported', () => {
    expect(rankCursorModelUsage([idleDay], 5)).toEqual([])
  })
})

describe('buildCursorDailySeries', () => {
  it('marks days without the requested metric as gaps instead of zeros', () => {
    const series = buildCursorDailySeries([activeDay, busyDay, idleDay], 'agent')

    expect(series).toEqual([
      { date: '1762819200000', value: 3, hasData: true },
      { date: '1762905600000', value: 5, hasData: true },
      { date: '1762992000000', value: undefined, hasData: false },
    ])
  })

  it('supports chat and lines-added metrics', () => {
    expect(buildCursorDailySeries([activeDay, busyDay], 'chat')).toEqual([
      { date: '1762819200000', value: undefined, hasData: false },
      { date: '1762905600000', value: 2, hasData: true },
    ])
    expect(buildCursorDailySeries([activeDay, busyDay], 'lines')).toEqual([
      { date: '1762819200000', value: 10, hasData: true },
      { date: '1762905600000', value: 100, hasData: true },
    ])
  })
})
