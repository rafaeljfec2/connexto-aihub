/**
 * Eager Vite URL map for vendored AI brand SVGs.
 * Keys are basenames (e.g. "openrouter-color.svg").
 */
const modules = import.meta.glob('../../../vendor/ai-icons/svgs/*.svg', {
  query: '?url',
  import: 'default',
  eager: true,
}) as Record<string, string>

function basename(modulePath: string): string {
  const parts = modulePath.split('/')
  return parts[parts.length - 1] ?? modulePath
}

export const aiIconUrlByFilename: ReadonlyMap<string, string> = new Map(
  Object.entries(modules).map(([modulePath, url]) => [basename(modulePath), url]),
)

export function aiIconFilenameUrl(filename: string): string | undefined {
  return aiIconUrlByFilename.get(filename)
}
