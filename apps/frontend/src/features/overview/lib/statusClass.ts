export function statusClass(state: string): string {
  if (state === 'healthy' || state === 'up' || state === 'ok') {
    return 'status-healthy'
  }
  if (state === 'disabled' || state === 'unknown') {
    return 'status-muted'
  }
  return 'status-unhealthy'
}
