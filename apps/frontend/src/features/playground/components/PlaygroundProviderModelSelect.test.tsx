import { render, screen } from '@testing-library/react'
import { describe, expect, it } from 'vitest'
import { PlaygroundProviderModelSelect } from './PlaygroundProviderModelSelect'

describe('PlaygroundProviderModelSelect', () => {
  it('requires provider before model selection', () => {
    render(
      <PlaygroundProviderModelSelect
        settings={{
          providerId: '',
          modelId: '',
          systemPrompt: '',
          temperature: 0.7,
        }}
        providers={['openrouter']}
        modelsForProvider={[]}
        disabled={false}
        onChange={() => undefined}
      />,
    )
    const modelSelect = screen.getByLabelText('Model') as HTMLSelectElement
    expect(modelSelect.disabled).toBe(true)
  })
})
