export type UiAuthSystem = 'operator' | 'users' | 'unknown'

let cachedUiAuthSystem: UiAuthSystem = 'unknown'

export function getUiAuthSystem(): UiAuthSystem {
  return cachedUiAuthSystem
}

export function setUiAuthSystem(system: UiAuthSystem): void {
  cachedUiAuthSystem = system
}
