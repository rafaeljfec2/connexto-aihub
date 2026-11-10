export type ContextCategory = {
  readonly id: string
  readonly name: string
  readonly description: string
  readonly glyph: string
  readonly evidenceStatus: string
  readonly adequacyLabel: string
}

export type DecisionStep = {
  readonly stepNumber: string
  readonly title: string
  readonly description: string
  readonly stateLabel: string
  readonly isFuture?: boolean
}

export type SignalItem = {
  readonly title: string
  readonly subtitle: string
  readonly status: 'exist' | 'pending'
  readonly statusLabel: string
}

export type EvolutionPhase = {
  readonly phaseNumber: string
  readonly phaseLabel: string
  readonly title: string
  readonly description: string
  readonly isActive?: boolean
}

