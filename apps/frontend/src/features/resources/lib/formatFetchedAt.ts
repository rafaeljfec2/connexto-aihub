export function formatFetchedAtUtc(updatedAtMs: number): string {
  return new Date(updatedAtMs).toISOString().replace('T', ' ').slice(0, 19)
}
