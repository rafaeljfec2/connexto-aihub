import { describe, expect, it } from 'vitest'
import { requiresOperatorLogin, type OperatorSession } from './operatorSession'

describe('requiresOperatorLogin', () => {
  it('requires login when unauthenticated with mode none', () => {
    const session: OperatorSession = { authenticated: false, mode: 'none', system: 'operator' }
    expect(requiresOperatorLogin(session)).toBe(true)
  })

  it('does not require login for loopback trust', () => {
    const session: OperatorSession = { authenticated: true, mode: 'loopback_trust', system: 'operator' }
    expect(requiresOperatorLogin(session)).toBe(false)
  })

  it('does not require login for passphrase session', () => {
    const session: OperatorSession = { authenticated: true, mode: 'passphrase', system: 'operator' }
    expect(requiresOperatorLogin(session)).toBe(false)
  })

  it('requires login for users system when unauthenticated', () => {
    const session: OperatorSession = { authenticated: false, mode: 'none', system: 'users' }
    expect(requiresOperatorLogin(session)).toBe(true)
  })
})
