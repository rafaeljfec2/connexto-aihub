import { render, screen } from '@testing-library/react'
import { describe, expect, it } from 'vitest'
import { PlaygroundMessageList } from './PlaygroundMessageList'

describe('PlaygroundMessageList', () => {
  it('renders empty state', () => {
    render(<PlaygroundMessageList messages={[]} />)
    expect(screen.getByText(/No messages yet/)).toBeTruthy()
  })
})
