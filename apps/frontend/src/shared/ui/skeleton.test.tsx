import { render } from '@testing-library/react'
import { describe, expect, it } from 'vitest'
import { Skeleton } from './skeleton'

describe('Skeleton', () => {
  it('uses a motion-safe pulse and stays decorative to assistive technology', () => {
    const { container } = render(<Skeleton motionSafe />)
    const skeleton = container.firstElementChild
    const classes = skeleton?.getAttribute('class')?.split(/\s+/) ?? []

    expect(classes).toContain('motion-safe:animate-pulse')
    expect(classes).not.toContain('animate-pulse')
    expect(skeleton?.getAttribute('aria-hidden')).toBe('true')
  })
})
