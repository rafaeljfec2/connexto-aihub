import { afterEach, describe, expect, it, vi } from 'vitest'
import { copyTextToClipboard } from './copyText'

describe('copyTextToClipboard', () => {
  afterEach(() => {
    vi.unstubAllGlobals()
    vi.restoreAllMocks()
  })

  it('returns false for blank text', async () => {
    expect(await copyTextToClipboard('   ')).toBe(false)
  })

  it('uses navigator.clipboard when available', async () => {
    const writeText = vi.fn().mockResolvedValue(undefined)
    vi.stubGlobal('navigator', { clipboard: { writeText } })
    expect(await copyTextToClipboard('anthropic/claude-fable-5.1')).toBe(true)
    expect(writeText).toHaveBeenCalledWith('anthropic/claude-fable-5.1')
  })
})
